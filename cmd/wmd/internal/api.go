package internal

import (
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"
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
	Storage *Storage
	Symbols *Symbols
}

func (a *DataFeedAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/status", a.Status)
	r.Get("/symbols", a.GetSymbols)
	r.Get("/markets", a.Markets)
	r.Get("/tickers", a.Tickers)
	r.Get("/ticker/{AmountAsset}/{PriceAsset}", a.Ticker)
	r.Get("/trades/{AmountAsset}/{PriceAsset}/{limit}", a.LastTrades)
	r.Get("/trades/{AmountAsset}/{PriceAsset}/{p1}/{p2}", a.Trades)
	r.Get("/candles/{AmountAsset}/{PriceAsset}/{TimeFrame}/{limit}", a.LastCandles)
	r.Get("/candles/{AmountAsset}/{PriceAsset}/{TimeFrame}/{from}/{to}", a.Candles)
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
	w.Write(js)
}

func (a *DataFeedAPI) Markets(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Tickers(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Ticker(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) LastTrades(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Trades(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) LastCandles(w http.ResponseWriter, r *http.Request) {

}

func (a *DataFeedAPI) Candles(w http.ResponseWriter, r *http.Request) {

}
