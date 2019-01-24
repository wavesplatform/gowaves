package internal

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	amountAssetPlaceholder = "AmountAsset"
	priceAssetPlaceholder  = "PriceAsset"
	limitPlaceholder       = "Limit"
	addressPlaceHolder     = "Address"
	timeFramePlaceholder   = "TimeFrame"
	fromPlaceholder        = "From"
	toPlaceholder          = "To"
)

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return.
func Logger(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				l.Info("Served",
					zap.String("proto", r.Proto),
					zap.String("path", r.URL.Path),
					zap.Duration("lat", time.Since(t1)),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.String("reqId", middleware.GetReqID(r.Context())))
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

type DataFeedAPI struct {
	log     *zap.SugaredLogger
	Storage *state.Storage
	Symbols *data.Symbols
	Scheme  byte
}

type status struct {
	CurrentHeight int              `json:"current_height"`
	LastBlockID   crypto.Signature `json:"last_block_id"`
}

func NewDataFeedAPI(log *zap.SugaredLogger, storage *state.Storage, symbols *data.Symbols) *DataFeedAPI {
	return &DataFeedAPI{log: log, Storage: storage, Symbols: symbols}
}

func (a *DataFeedAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/status", a.Status)
	r.Get("/symbols", a.GetSymbols)
	r.Get("/markets", a.Markets)
	r.Get("/tickers", a.Tickers)
	r.Get(fmt.Sprintf("/ticker/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder), a.Ticker)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder, limitPlaceholder), a.Trades)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, fromPlaceholder, toPlaceholder), a.TradesRange)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:[1-9A-Za-z]+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, addressPlaceHolder, limitPlaceholder), a.TradesByAddress)
	r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, limitPlaceholder), a.Candles)
	r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, fromPlaceholder, toPlaceholder), a.CandlesRange)
	return r
}

func (a *DataFeedAPI) Status(w http.ResponseWriter, r *http.Request) {
	h, err := a.Storage.Height()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	blockID, err := a.Storage.BlockID(h)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	s := status{CurrentHeight: h, LastBlockID: blockID}
	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) GetSymbols(w http.ResponseWriter, r *http.Request) {
	s := a.Symbols.All()
	err := json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Symbols to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) Markets(w http.ResponseWriter, r *http.Request) {
	markets, err := a.Storage.Markets()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect Markets: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	mis := make([]data.MarketInfo, 0, len(markets))
	for m, md := range markets {
		c, err := a.Storage.DayCandle(m.AmountAsset, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load DayCandle: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aai, err := a.Storage.AssetInfo(m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pai, err := a.Storage.AssetInfo(m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aab, err := a.getIssuerBalance(aai.Issuer, m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pab, err := a.getIssuerBalance(pai.Issuer, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ti := a.convertToTickerInfo(aai, pai, aab, pab, c)
		mi := data.NewMarketInfo(ti, md)
		mis = append(mis, mi)
	}
	sort.Sort(data.ByMarkets(mis))
	err = json.NewEncoder(w).Encode(mis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Markets to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) Tickers(w http.ResponseWriter, r *http.Request) {
	markets, err := a.Storage.Markets()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect Tickers: %s", err), http.StatusInternalServerError)
		return
	}
	tis := make([]data.TickerInfo, 0, len(markets))
	for m := range markets {
		c, err := a.Storage.DayCandle(m.AmountAsset, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load DayCandle: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aai, err := a.Storage.AssetInfo(m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pai, err := a.Storage.AssetInfo(m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aab, err := a.getIssuerBalance(aai.Issuer, m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pab, err := a.getIssuerBalance(pai.Issuer, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ti := a.convertToTickerInfo(aai, pai, aab, pab, c)
		tis = append(tis, ti)
	}
	sort.Sort(data.ByTickers(tis))
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Tickers to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) Ticker(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	c, err := a.Storage.DayCandle(amountAsset, priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load DayCandle: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	aab, err := a.getIssuerBalance(aai.Issuer, amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pab, err := a.getIssuerBalance(pai.Issuer, priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ti := a.convertToTickerInfo(aai, pai, aab, pab, c)
	err = json.NewEncoder(w).Encode(ti)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Ticker to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) Trades(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(chi.URLParam(r, limitPlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if limit < 1 || limit > 1000 {
		http.Error(w, fmt.Sprintf("Bad request: %d is invalid limit value, allowed between 1 and 1000", limit), http.StatusBadRequest)
		return

	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ts, err := a.Storage.Trades(amountAsset, priceAsset, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tis, err := a.convertToTradesInfos(ts, aai.Decimals, pai.Decimals)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) TradesRange(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	f, err := strconv.Atoi(chi.URLParam(r, fromPlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	t, err := strconv.Atoi(chi.URLParam(r, toPlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	trades, err := a.Storage.TradesRange(amountAsset, priceAsset, uint64(f), uint64(t))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tis, err := a.convertToTradesInfos(trades, aai.Decimals, pai.Decimals)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert trades: %s", err.Error()), http.StatusInternalServerError)
	}
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) TradesByAddress(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	limit, err := strconv.Atoi(chi.URLParam(r, limitPlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if limit < 1 || limit > 1000 {
		http.Error(w, fmt.Sprintf("Bad request: %d is invalid limit value, allowed between 1 and 1000", limit), http.StatusBadRequest)
		return

	}
	ad := chi.URLParam(r, addressPlaceHolder)
	address, err := proto.NewAddressFromString(ad)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ts, err := a.Storage.TradesByAddress(amountAsset, priceAsset, address, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tis, err := a.convertToTradesInfos(ts, aai.Decimals, pai.Decimals)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert to TradeInfos: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) Candles(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	tf, err := strconv.Atoi(chi.URLParam(r, timeFramePlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	if tf != 5 && tf != 15 && tf != 30 && tf != 60 && tf != 240 && tf != 1440 {
		http.Error(w, fmt.Sprintf("Bad request: incorrect time frame %d, allowed values: 5, 15, 30, 60, 240 and 1440 minutes", tf), http.StatusBadRequest)
	}
	limit, err := strconv.Atoi(chi.URLParam(r, limitPlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if limit < 1 || limit > 1000 {
		http.Error(w, fmt.Sprintf("Bad request: %d is invalid limit value, allowed between 1 and 1000", limit), http.StatusBadRequest)
		return

	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tfs := tf / data.DefaultTimeFrame
	ttf := data.TimeFrameFromTimestampMS(uint64(time.Now().Unix() * 1000))
	ftf := ttf - uint32((limit-1)*tfs)
	cis := make(map[uint32]data.CandleInfo)
	for x := data.ScaleTimeFrame(ftf, tfs); x <= data.ScaleTimeFrame(ttf, tfs); x += uint32(tfs) {
		cis[x] = data.EmptyCandleInfo(uint(aai.Decimals), uint(pai.Decimals), data.TimestampMSFromTimeFrame(x))
	}
	candles, err := a.Storage.CandlesRange(amountAsset, priceAsset, ftf, ttf, tfs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect Candles: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	csm := make(map[uint32]data.Candle)
	for _, c := range candles {
		ctf := data.ScaleTimeFrame(data.TimeFrameFromTimestampMS(c.MinTimestamp), tfs)
		if cc, ok := csm[ctf]; !ok {
			csm[ctf] = c
		} else {
			cc.Combine(c)
			csm[ctf] = cc
		}
	}
	res := make(data.ByTimestampBackward, len(cis))
	i := 0
	for k, v := range cis {
		if c, ok := csm[k]; ok {
			res[i] = data.CandleInfoFromCandle(c, uint(aai.Decimals), uint(pai.Decimals), tfs)
		} else {
			res[i] = v
		}
		i++
	}
	sort.Sort(res)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal CandleInfos to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

}

func (a *DataFeedAPI) CandlesRange(w http.ResponseWriter, r *http.Request) {
	aa := chi.URLParam(r, amountAssetPlaceholder)
	amountAsset, err := a.Symbols.ParseTicker(aa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	pa := chi.URLParam(r, priceAssetPlaceholder)
	priceAsset, err := a.Symbols.ParseTicker(pa)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	tf, err := strconv.Atoi(chi.URLParam(r, timeFramePlaceholder))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	if tf != 5 && tf != 15 && tf != 30 && tf != 60 && tf != 240 && tf != 1440 {
		http.Error(w, fmt.Sprintf("Bad request: incorrect time frame %d, allowed values: 5, 15, 30, 60, 240 and 1440 minutes", tf), http.StatusBadRequest)
	}
	f, err := strconv.ParseUint(chi.URLParam(r, fromPlaceholder), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	t, err := strconv.ParseUint(chi.URLParam(r, toPlaceholder), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	if f > t {
		http.Error(w, fmt.Sprintf("Bad request: start of the range %d should be less than %d", f, t), http.StatusBadRequest)
	}
	aai, err := a.Storage.AssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.AssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ftf := data.TimeFrameFromTimestampMS(f)
	ttf := data.TimeFrameFromTimestampMS(t)
	tfs := tf / data.DefaultTimeFrame
	cis := make(map[uint32]data.CandleInfo)
	for x := data.ScaleTimeFrame(ftf, tfs); x <= data.ScaleTimeFrame(ttf, tfs); x += uint32(tfs) {
		cis[x] = data.EmptyCandleInfo(uint(aai.Decimals), uint(pai.Decimals), data.TimestampMSFromTimeFrame(x))
	}
	candles, err := a.Storage.CandlesRange(amountAsset, priceAsset, ftf, ttf, tfs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect Candles: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	csm := make(map[uint32]data.Candle)
	for _, c := range candles {
		ctf := data.ScaleTimeFrame(data.TimeFrameFromTimestampMS(c.MinTimestamp), tfs)
		if cc, ok := csm[ctf]; !ok {
			csm[ctf] = c
		} else {
			cc.Combine(c)
			csm[ctf] = cc
		}
	}
	res := make(data.ByTimestampBackward, len(cis))
	i := 0
	for k, v := range cis {
		if c, ok := csm[k]; ok {
			res[i] = data.CandleInfoFromCandle(c, uint(aai.Decimals), uint(pai.Decimals), tfs)
		} else {
			res[i] = v
		}
		i++
	}
	sort.Sort(res)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal CandleInfos to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func (a *DataFeedAPI) convertToTradesInfos(trades []data.Trade, amountAssetDecimals, priceAssetDecimals byte) ([]data.TradeInfo, error) {
	var r []data.TradeInfo
	for i := 0; i < len(trades); i++ {
		ti := data.NewTradeInfo(trades[i], uint(amountAssetDecimals), uint(priceAssetDecimals))
		r = append(r, ti)
	}
	return r, nil
}

func (a *DataFeedAPI) convertToTickerInfo(aa, pa *data.AssetInfo, aaBalance, paBalance uint64, c data.Candle) data.TickerInfo {
	var sb strings.Builder
	aat, ok := a.Symbols.Tokens()[aa.ID]
	if ok {
		sb.WriteString(aat)
		pat, ok := a.Symbols.Tokens()[pa.ID]
		if ok {
			sb.WriteRune('/')
			sb.WriteString(pat)
		} else {
			sb.Reset()
		}
	}
	return data.NewTickerInfo(sb.String(), *aa, *pa, aaBalance, paBalance, c)
}

func (a *DataFeedAPI) getIssuerBalance(issuer crypto.PublicKey, asset crypto.Digest) (uint64, error) {
	address, err := proto.NewAddressFromPublicKey(a.Scheme, issuer)
	if err != nil {
		return 0, err
	}
	balance, err := a.Storage.IssuerBalance(address, asset)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
