package internal

import (
	"bytes"
	"compress/flate"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/xenolf/lego/log"
	"go.uber.org/zap"
)

const (
	amountAssetPlaceholder = "AmountAsset"
	priceAssetPlaceholder  = "PriceAsset"
	limitPlaceholder       = "Limit"
	addressPlaceHolder     = "Address"
	timeFramePlaceholder   = "TimeFrame"
	fromPlaceholder        = "From"
	toPlaceholder          = "To"

	defaultTimeout = 30 * time.Second
)

var (
	//go:embed swagger
	res embed.FS
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

type status struct {
	CurrentHeight int           `json:"current_height"`
	LastBlockID   proto.BlockID `json:"last_block_id"`
}

type DataFeedAPI struct {
	interrupt <-chan struct{}
	done      chan struct{}
	Storage   *state.Storage
	Symbols   *data.Symbols
}

func NewDataFeedAPI(interrupt <-chan struct{}, logger *zap.Logger, storage *state.Storage, address string, symbols *data.Symbols) *DataFeedAPI {
	a := DataFeedAPI{interrupt: interrupt, done: make(chan struct{}), Storage: storage, Symbols: symbols}
	swaggerFS, err := fs.Sub(res, "swagger")
	if err != nil {
		log.Fatalf("Failed to initialise Swagger: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(flate.DefaultCompression))
	r.Mount("/", a.swagger(swaggerFS))
	r.Mount("/api", a.routes())
	apiServer := &http.Server{Addr: address, Handler: r, ReadHeaderTimeout: defaultTimeout, ReadTimeout: defaultTimeout}
	go func() {
		err := apiServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.S().Fatalf("Failed to start API: %v", err)
			return
		}
	}()
	go func() {
		<-a.interrupt
		zap.S().Info("Shutting down API...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := apiServer.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("Failed to shutdown API server: %v", err)
		}
		cancel()
		close(a.done)
	}()
	return &a
}

func (a *DataFeedAPI) Done() <-chan struct{} {
	return a.done
}

func (a *DataFeedAPI) swagger(fs fs.FS) chi.Router {
	r := chi.NewRouter()
	h := http.FileServer(http.FS(fs))
	r.Mount("/", h)
	return r
}

func (a *DataFeedAPI) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.SetHeader("Content-Type", "application/json; charset=UTF-8"))
	r.Get("/status", a.status)
	r.Get("/symbols", a.getSymbols)
	r.Get("/markets", a.markets)
	r.Get("/tickers", a.tickers)
	r.Get(fmt.Sprintf("/ticker/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder), a.ticker)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder, limitPlaceholder), a.trades)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, fromPlaceholder, toPlaceholder), a.tradesRange)
	r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:[1-9A-Za-z]+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, addressPlaceHolder, limitPlaceholder), a.tradesByAddress)
	r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, limitPlaceholder), a.candles)
	r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, fromPlaceholder, toPlaceholder), a.candlesRange)
	return r
}

func (a *DataFeedAPI) status(w http.ResponseWriter, _ *http.Request) {
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
}

func (a *DataFeedAPI) getSymbols(w http.ResponseWriter, _ *http.Request) {
	s := a.Symbols.All()
	err := json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Symbols to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (a *DataFeedAPI) markets(w http.ResponseWriter, _ *http.Request) {
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
			zap.S().Warnf("Failed to load AssetInfo: %s", err.Error())
			continue // Skip assets with unavailable info, probably issued by InvokeScript transaction
		}
		pai, err := a.Storage.AssetInfo(m.PriceAsset)
		if err != nil {
			zap.S().Warnf("Failed to load AssetInfo: %s", err.Error())
			continue // Skip assets with unavailable info, probably issued by InvokeScript transaction
		}
		aab, err := a.getIssuerBalance(aai.IssuerAddress, m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pab, err := a.getIssuerBalance(pai.IssuerAddress, m.PriceAsset)
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
}

func (a *DataFeedAPI) tickers(w http.ResponseWriter, _ *http.Request) {
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
			zap.S().Warnf("Failed to load AssetInfo: %s", err.Error())
			continue // Skip assets with unavailable info, probably issued by InvokeScript transaction
		}
		pai, err := a.Storage.AssetInfo(m.PriceAsset)
		if err != nil {
			zap.S().Warnf("Failed to load AssetInfo: %s", err.Error())
			continue // Skip assets with unavailable info, probably issued by InvokeScript transaction
		}
		aab, err := a.getIssuerBalance(aai.IssuerAddress, m.AmountAsset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		pab, err := a.getIssuerBalance(pai.IssuerAddress, m.PriceAsset)
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
}

func (a *DataFeedAPI) ticker(w http.ResponseWriter, r *http.Request) {
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
	aab, err := a.getIssuerBalance(aai.IssuerAddress, amountAsset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get issuer's balance: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	pab, err := a.getIssuerBalance(pai.IssuerAddress, priceAsset)
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
}

func (a *DataFeedAPI) trades(w http.ResponseWriter, r *http.Request) {
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
	ts, err := a.Storage.Trades(amountAsset, priceAsset, 2*limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tis, err := a.convertToTradesInfos(ts, aai.Decimals, pai.Decimals)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert Trades: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	sort.Sort(data.TradesByTimestampBackward(tis))
	if len(tis) < limit {
		limit = len(tis)
	}
	err = json.NewEncoder(w).Encode(tis[:limit])
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (a *DataFeedAPI) tradesRange(w http.ResponseWriter, r *http.Request) {
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
	f, err := strconv.ParseUint(chi.URLParam(r, fromPlaceholder), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}
	t, err := strconv.ParseUint(chi.URLParam(r, toPlaceholder), 10, 64)
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
	trades, err := a.Storage.TradesRange(amountAsset, priceAsset, f, t)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	tis, err := a.convertToTradesInfos(trades, aai.Decimals, pai.Decimals)
	sort.Sort(data.TradesByTimestampBackward(tis))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert trades: %s", err.Error()), http.StatusInternalServerError)
	}
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (a *DataFeedAPI) tradesByAddress(w http.ResponseWriter, r *http.Request) {
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
	sort.Sort(data.TradesByTimestampBackward(tis))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert to TradeInfos: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(tis)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal Trades to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (a *DataFeedAPI) candles(w http.ResponseWriter, r *http.Request) {
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
	res := make(data.ByCandlesTimestampBackward, len(cis))
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

}

func (a *DataFeedAPI) candlesRange(w http.ResponseWriter, r *http.Request) {
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
	res := make(data.ByCandlesTimestampBackward, len(cis))
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
	aat, ok := a.Symbols.Token(aa.ID)
	if ok {
		sb.WriteString(aat)
		pat, ok := a.Symbols.Token(pa.ID)
		if ok {
			sb.WriteRune('/')
			sb.WriteString(pat)
		} else {
			sb.Reset()
		}
	}
	return data.NewTickerInfo(sb.String(), *aa, *pa, aaBalance, paBalance, c)
}

func (a *DataFeedAPI) getIssuerBalance(issuer proto.WavesAddress, asset crypto.Digest) (uint64, error) {
	if bytes.Equal(issuer[:], data.WavesIssuerAddress[:]) {
		return 0, nil
	}
	balance, err := a.Storage.IssuerBalance(issuer, asset)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
