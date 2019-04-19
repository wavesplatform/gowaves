package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"math/big"
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
	peers node.PeerManager
}

func NewNodeApi(state state.State, node *node.Node, peers node.PeerManager) *NodeApi {
	return &NodeApi{
		state: state,
		node:  node,
		peers: peers,
	}
}

func (a *NodeApi) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/blocks/last", a.BlocksLast)
	r.Get("/blocks/first", a.BlocksFirst)
	r.Get("/blocks/at/{id:\\d+}", a.BlockAt)

	// peers
	r.Get("/peers/all", a.PeersAll)
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

type PeersAll struct {
	Peers []Peer `json:"peers"`
}

type Peer struct {
	Address  string `json:"address"`
	LastSeen uint64 `json:"lastSeen"`
}

func (a *NodeApi) PeersAll(w http.ResponseWriter, r *http.Request) {
	peers, err := a.state.Peers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var out []Peer
	for _, row := range peers {
		out = append(out, Peer{Address: row.String()})
	}

	err = json.NewEncoder(w).Encode(PeersAll{Peers: out})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

type PeersConnected struct {
	Peers []PeersConnectedRow `json:"peers"`
}

type PeersConnectedRow struct {
	Address            string
	DeclaredAddress    string
	peerName           string
	peerNonce          uint64
	applicationName    string
	applicationVersion proto.Version
}

func (a *NodeApi) PeersConnected(w http.ResponseWriter, r *http.Request) {
	var out []PeersConnectedRow
	a.peers.EachConnected(func(peer peer.Peer, i *big.Int) {

		v := PeersConnectedRow{
			Address:            peer.RemoteAddr().String(),
			DeclaredAddress:    peer.Handshake().DeclaredAddr.String(),
			peerName:           peer.Handshake().NodeName,
			peerNonce:          peer.Handshake().NodeNonce,
			applicationName:    peer.Handshake().AppName,
			applicationVersion: peer.Handshake().Version,
		}

		out = append(out, v)

	})

	err := json.NewEncoder(w).Encode(PeersConnected{Peers: out})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
