package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

const API_KEY = "X-API-Key"

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

func (a *NodeApi) TransactionsBroadcast(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		handleError(w, &BadRequestError{err})
		return
	}

	err = a.app.TransactionsBroadcast(b)
	if err != nil {
		handleError(w, err)
		return
	}
}

func (a *NodeApi) BlocksLast(w http.ResponseWriter, _ *http.Request) {
	block, err := a.app.BlocksLast()
	if err != nil {
		handleError(w, err)
		return
	}

	bts, err := proto.BlockEncodeJson(block)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(bts)
}

func (a *NodeApi) BlocksFirst(w http.ResponseWriter, r *http.Request) {
	block, err := a.state.BlockByHeight(1)
	if err != nil {
		handleError(w, err)
		return
	}
	block.Height = 1
	bts, err := proto.BlockEncodeJson(block)
	if err != nil {
		handleError(w, err)
		return
	}
	_, _ = w.Write(bts)
}

func (a *NodeApi) BlockAt(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "height")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	block, err := a.state.BlockByHeight(id)
	if err != nil {
		handleError(w, err)
		return
	}
	block.Height = id
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		handleError(w, err)
		return
	}
}

func (a *NodeApi) DebugSyncEnabled(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "enabled")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.app.DebugSyncEnabled(id == 1)
}

func (a *NodeApi) BlockIDAt(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	block, err := a.state.Block(id)
	if err != nil {
		handleError(w, err)
		return
	}
	height, err := a.state.BlockIDToHeight(id)
	if err != nil {
		handleError(w, err)
		return
	}
	block.Height = height
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		handleError(w, err)
		return
	}
}

type BlockHeightResponse struct {
	Height uint64 `json:"height"`
}

func (a *NodeApi) BlockHeight(w http.ResponseWriter, r *http.Request) {
	lock := a.state.Mutex().RLock()
	height, err := a.state.Height()
	lock.Unlock()
	if err != nil {
		handleError(w, err)
		return
	}
	err = json.NewEncoder(w).Encode(&BlockHeightResponse{Height: height})
	if err != nil {
		handleError(w, err)
		return
	}
}

func (a *NodeApi) BlockScoreAt(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		handleError(w, &BadRequestError{err})
		return
	}
	rs, err := a.app.BlocksScoreAt(id)
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func Run(ctx context.Context, address string, n *NodeApi) error {
	apiServer := &http.Server{Addr: address, Handler: n.routes()}
	go func() {
		<-ctx.Done()
		zap.S().Info("Shutting down API...")
		err := apiServer.Shutdown(ctx)
		if err != nil {
			zap.S().Errorf("Failed to shutdown API server: %v", err)
		}
	}()
	err := apiServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *NodeApi) PeersAll(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.PeersAll()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersSpawned(w http.ResponseWriter, r *http.Request) {
	rs := a.app.PeersSpawned()
	sendJson(w, rs)
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
		handleError(w, err)
		return
	}

	zap.S().Info("PeersConnect ", r.Header.Get("X-API-Key"), req)

	apiKey := r.Header.Get("X-API-Key")
	rs, err := a.app.PeersConnect(r.Context(), apiKey, fmt.Sprintf("%s:%d", req.Host, req.Port))
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersConnected(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.PeersConnected()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersSuspended(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.PeersSuspended()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) BlocksGenerators(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.BlocksGenerators()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) poolTransactions(w http.ResponseWriter, r *http.Request) {
	rs := a.app.PoolTransactions()
	sendJson(w, rs)
}

type rollbackRequest struct {
	Height uint64 `json:"height"`
}

type rollbackToHeight interface {
	RollbackToHeight(string, proto.Height) error
}

func RollbackToHeight(app rollbackToHeight) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		js := &rollbackRequest{}
		err := json.NewDecoder(r.Body).Decode(js)
		if err != nil {
			handleError(w, err)
			return
		}
		apiKey := r.Header.Get("X-API-Key")
		err = app.RollbackToHeight(apiKey, js.Height)
		if err != nil {
			handleError(w, err)
			return
		}
		sendJson(w, nil)
	}
}

type walletLoadKeysRequest struct {
	Password string `json:"password"`
}
type walletLoadKeys interface {
	LoadKeys(apiKey string, password []byte) error
}

func WalletLoadKeys(app walletLoadKeys) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		js := &walletLoadKeysRequest{}
		err := json.NewDecoder(r.Body).Decode(js)
		if err != nil {
			handleError(w, err)
			return
		}
		apiKey := r.Header.Get("X-API-Key")
		err = app.LoadKeys(apiKey, []byte(js.Password))
		if err != nil {
			handleError(w, err)
			return
		}
		sendJson(w, nil)
	}
}

func (a *NodeApi) Minerinfo(w http.ResponseWriter, r *http.Request) {
	rs, err := a.app.Miner()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) nodeProcesses(w http.ResponseWriter, r *http.Request) {
	rs := a.app.NodeProcesses()
	sendJson(w, rs)
}

func handleError(w http.ResponseWriter, err error) {
	switch err.(type) {
	case *AuthError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusForbidden)
	case *BadRequestError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusBadRequest)
	default:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
	}
}

func sendJson(w http.ResponseWriter, v interface{}) {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
	}
}
