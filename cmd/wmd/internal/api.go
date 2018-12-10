package internal

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net/http"
	"strconv"
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
	js, err := json.Marshal(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		a.log.Errorf("Failed to send reply: %s", err.Error())
	}
}

func (a *DataFeedAPI) Markets(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Tickers(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Ticker(w http.ResponseWriter, r *http.Request) {

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
	trades, err := a.Storage.TradeInfos(amountAsset, priceAsset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(trades)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		a.log.Errorf("Failed to send reply: %s", err.Error())
	}
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
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	t, err := strconv.Atoi(chi.URLParam(r, toPlaceholder))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	trades, err := a.Storage.TradeInfosRange(amountAsset, priceAsset, uint64(f), uint64(t))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(trades)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		a.log.Errorf("Failed to send reply: %s", err.Error())
	}
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
	trades, err := a.Storage.TradeInfosByAddress(amountAsset, priceAsset, address, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(trades)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		a.log.Errorf("Failed to send reply: %s", err.Error())
	}
}

func (a *DataFeedAPI) Candles(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) CandlesRange(w http.ResponseWriter, r *http.Request) {

}
