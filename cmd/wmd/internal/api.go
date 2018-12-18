package internal

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
	Storage *Storage
	Symbols *Symbols
	Scheme  byte
}

func NewDataFeedAPI(log *zap.SugaredLogger, storage *Storage, symbols *Symbols) *DataFeedAPI {
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
	mis := make([]MarketInfo, 0, len(markets))
	for m, md := range markets {
		c, err := a.Storage.DayCandle(m.AmountAsset, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load DayCandle: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aai, err := a.Storage.readAssetInfo(m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pai, err := a.Storage.readAssetInfo(m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ti := a.convertToTickerInfo(aai, pai, c)

		mi := NewMarketInfo(ti, md)
		mis = append(mis, mi)
	}
	sort.Sort(ByMarkets(mis))
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
	tis := make([]TickerInfo, 0, len(markets))
	for m := range markets {
		c, err := a.Storage.DayCandle(m.AmountAsset, m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load DayCandle: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		aai, err := a.Storage.readAssetInfo(m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pai, err := a.Storage.readAssetInfo(m.PriceAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ti := a.convertToTickerInfo(aai, pai, c)
		tis = append(tis, ti)
	}
	sort.Sort(ByTickers(tis))
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
	aai, err := a.Storage.readAssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.readAssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ti := a.convertToTickerInfo(aai, pai, c)
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
	aai, err := a.Storage.readAssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.readAssetInfo(priceAsset)
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
	aai, err := a.Storage.readAssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.readAssetInfo(priceAsset)
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
	pk, err := a.Storage.PublicKey(address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to find PublicKey for Address: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ts, err := a.Storage.TradesByPublicKey(amountAsset, priceAsset, pk, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	aai, err := a.Storage.readAssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.readAssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
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
	aai, err := a.Storage.readAssetInfo(amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pai, err := a.Storage.readAssetInfo(priceAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AssetInfo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	ftf := timeFrameFromTimestampMS(f)
	ttf := timeFrameFromTimestampMS(t)
	tfs := tf / DefaultTimeFrame
	cis := make(map[uint32]CandleInfo)
	for x := scaleTimeFrame(ftf, tfs); x <= scaleTimeFrame(ttf, tfs); x += uint32(tfs) {
		cis[x] = EmptyCandleInfo(uint(aai.Decimals), uint(pai.Decimals), timestampMSFromTimeFrame(x))
	}
	candles, err := a.Storage.CandlesRange(amountAsset, priceAsset, ftf, ttf, tfs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect Candles: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	csm := make(map[uint32]Candle)
	for _, c := range candles {
		ctf := scaleTimeFrame(timeFrameFromTimestampMS(c.minTimestamp), tfs)
		if cc, ok := csm[ctf]; !ok {
			csm[ctf] = c
		} else {
			cc.Combine(c)
			csm[ctf] = cc
		}
	}
	res := make(ByTimestampBackward, len(cis))
	i := 0
	for k, v := range cis {
		if c, ok := csm[k]; ok {
			res[i] = CandleInfoFromCandle(c, uint(aai.Decimals), uint(pai.Decimals), tfs)
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

func (a *DataFeedAPI) convertToTradesInfos(trades []Trade, amountAssetDecimals, priceAssetDecimals byte) ([]TradeInfo, error) {
	var r []TradeInfo
	for i := 0; i < len(trades); i++ {
		ti, err := NewTradeInfo(trades[i], a.Scheme, uint(amountAssetDecimals), uint(priceAssetDecimals))
		if err != nil {
			return nil, err
		}
		r = append(r, *ti)
	}
	return r, nil
}

func (a *DataFeedAPI) convertToTickerInfo(aa, pa *AssetInfo, c Candle) TickerInfo {
	var sb strings.Builder
	aat, ok := a.Symbols.tokens[aa.ID]
	if ok {
		sb.WriteString(aat)
		pat, ok := a.Symbols.tokens[pa.ID]
		if ok {
			sb.WriteRune('/')
			sb.WriteString(pat)
		} else {
			sb.Reset()
		}
	}
	return NewTickerInfo(sb.String(), *aa, *pa, c)
}
