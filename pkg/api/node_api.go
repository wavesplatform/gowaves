package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/state"
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

type NodeApi struct {
	state state.State
}

func NewNodeApi(state state.State) *NodeApi {
	return &NodeApi{
		state: state,
	}
}

func (a *NodeApi) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/blocks/last", a.BlocksLast)
	r.Get("/blocks/first", a.BlocksFirst)
	//r.Get("/symbols", a.getSymbols)
	//r.Get("/markets", a.markets)
	//r.Get("/tickers", a.tickers)
	//r.Get(fmt.Sprintf("/ticker/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder), a.ticker)
	//r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s}", amountAssetPlaceholder, priceAssetPlaceholder, limitPlaceholder), a.trades)
	//r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, fromPlaceholder, toPlaceholder), a.tradesRange)
	//r.Get(fmt.Sprintf("/trades/{%s}/{%s}/{%s:[1-9A-Za-z]+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, addressPlaceHolder, limitPlaceholder), a.tradesByAddress)
	//r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, limitPlaceholder), a.candles)
	//r.Get(fmt.Sprintf("/candles/{%s}/{%s}/{%s:\\d+}/{%s:\\d+}/{%s:\\d+}", amountAssetPlaceholder, priceAssetPlaceholder, timeFramePlaceholder, fromPlaceholder, toPlaceholder), a.candlesRange)
	return r
}

func (a *NodeApi) BlocksLast(w http.ResponseWriter, r *http.Request) {
	h, err := a.state.Height()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	block, err := a.state.BlockByHeight(h)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	height, err := a.state.BlockIDToHeight(block.BlockSignature)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	block.Height = height
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (a *NodeApi) BlocksFirst(w http.ResponseWriter, r *http.Request) {
	block, err := a.state.BlockByHeight(1)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	block.Height = 1
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func Run(ctx context.Context, address string, n *NodeApi) error {
	apiServer := &http.Server{Addr: address, Handler: n.routes()}
	go func() {
		select {
		case <-ctx.Done():
			zap.S().Info("Shutting down API...")
			err := apiServer.Shutdown(ctx)
			if err != nil {
				zap.S().Errorf("Failed to shutdown API server: %v", err)
			}
			return
		}
	}()
	err := apiServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
