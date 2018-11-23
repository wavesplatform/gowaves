package internal

import (
	"github.com/go-chi/chi"
	"net/http"
)

type DataFeedAPI struct {
	Storage *Storage
}

func (a *DataFeedAPI) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/status", a.Status)
	r.Get("/symbols", a.Symbols)
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

func (a *DataFeedAPI) Symbols(w http.ResponseWriter, r *http.Request) {

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
