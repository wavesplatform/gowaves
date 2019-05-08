package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"net/http"
	"strconv"
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
	node  *node.Node
	app   *App
}

func NewNodeApi(app *App, state state.State, node *node.Node) *NodeApi {
	return &NodeApi{
		state: state,
		node:  node,
		app:   app,
	}
}

func (a *NodeApi) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/blocks/last", a.BlocksLast)
	r.Get("/blocks/height", a.BlockHeight)
	r.Get("/blocks/first", a.BlocksFirst)
	r.Get("/blocks/at/{id:\\d+}", a.BlockAt)

	r.Route("/peers", func(r chi.Router) {
		r.Get("/all", a.PeersAll)
		r.Get("/connected", a.PeersConnected)
		r.Post("/connect", a.PeersConnect)
	})

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

func (a *NodeApi) BlockAt(w http.ResponseWriter, r *http.Request) {

	s := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	block, err := a.state.BlockByHeight(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	block.Height = id
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

type BlockHeightResponse struct {
	Height uint64 `json:"height"`
}

func (a *NodeApi) BlockHeight(w http.ResponseWriter, r *http.Request) {
	height, err := a.state.Height()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(&BlockHeightResponse{Height: height})
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

func (a *NodeApi) PeersAll(w http.ResponseWriter, r *http.Request) {
	peers, err := a.app.PeersAll()
	if err != nil {
		handleError(err, w)
		return
	}
	sendJson(peers, w)
}

type PeersConnectRequest struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (a *NodeApi) PeersConnect(w http.ResponseWriter, r *http.Request) {

	zap.S().Info("PeersConnect ", r.Header.Get("X-API-Key"))

	req := new(PeersConnectRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusBadRequest)
		return
	}

	zap.S().Info("PeersConnect ", r.Header.Get("X-API-Key"), req)

	apiKey := r.Header.Get("X-API-Key")
	rs, err := a.app.PeersConnect(context.Background(), apiKey, fmt.Sprintf("%s:%d", req.Host, req.Port))
	if err != nil {
		handleError(err, w)
		return
	}
	sendJson(rs, w)
}

func (a *NodeApi) PeersConnected(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.PeersConnected()
	if err != nil {
		handleError(err, w)
		return
	}
	sendJson(rs, w)
}

func handleError(err error, w http.ResponseWriter) {
	switch err.(type) {
	case *AuthError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusForbidden)
	case *BadRequestError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusBadRequest)
	default:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
	}
}

func sendJson(v interface{}, w http.ResponseWriter) {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
	}
}
