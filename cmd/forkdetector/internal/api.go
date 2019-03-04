package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"go.uber.org/zap"
	"net"
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

type status struct {
	ShortForksCount     int `json:"short_forks_count"`
	LongForksCount      int `json:"long_forks_count"`
	KnowNodesCount      int `json:"know_nodes_count"`
	ConnectedNodesCount int `json:"connected_nodes_count"`
}

type api struct {
	interrupt <-chan struct{}
	log       *zap.SugaredLogger
	storage   *storage
}

func StartForkDetectorAPI(interrupt <-chan struct{}, logger *zap.Logger, storage *storage, bind string) <-chan struct{} {
	done := make(chan struct{})
	if bind == "" {
		close(done)
		return done
	}
	a := api{interrupt: interrupt, log: logger.Sugar(), storage: storage}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.DefaultCompress)
	r.Mount("/api", a.routes())
	apiServer := &http.Server{Addr: bind, Handler: r}
	go func() {
		err := apiServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			a.log.Fatalf("Failed to start API: %v", err)
			return
		}
	}()
	go func() {
		for {
			select {
			case <-a.interrupt:
				a.log.Info("Shutting down API...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := apiServer.Shutdown(ctx)
				if err != nil {
					a.log.Errorf("Failed to shutdown API server: %v", err)
				}
				cancel()
				close(done)
				return
			}
		}
	}()
	return done
}

func (a *api) routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/status", a.status)
	r.Get("/peers", a.peers)
	r.Get("/parentedForks", a.forks)
	r.Get("/node/{address}", a.node)
	r.Get("/height/{height:\\d+}", a.blocksAtHeight)
	r.Get("/block/{id:[a-km-zA-HJ-NP-Z1-9]+}", a.block)
	return r
}

func (a *api) status(w http.ResponseWriter, r *http.Request) {
	forks, err := a.storage.parentedForks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	short, long := countForksByLength(forks)
	peers, err := a.storage.peers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	s := status{ShortForksCount: short, LongForksCount: long, ConnectedNodesCount: connectedPeersCount(peers), KnowNodesCount: len(peers)}
	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) peers(w http.ResponseWriter, r *http.Request) {
	peers, err := a.storage.peers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(peers)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) forks(w http.ResponseWriter, r *http.Request) {
	forks, err := a.storage.parentedForks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(forks)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) node(w http.ResponseWriter, r *http.Request) {
	addr := chi.URLParam(r, "address")
	ip := net.ParseIP(addr)
	if ip == nil {
		http.Error(w, fmt.Sprintf("Invalid IP address '%s'", addr), http.StatusBadRequest)
		return
	}
	fork, err := a.storage.fork(ip)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(fork)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) blocksAtHeight(w http.ResponseWriter, r *http.Request) {
	p := chi.URLParam(r, "height")
	h, err := strconv.Atoi(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid height: %v", err), http.StatusBadRequest)
		return
	}
	blocks, err := a.storage.blocks(uint32(h))
	err = json.NewEncoder(w).Encode(blocks)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) block(w http.ResponseWriter, r *http.Request) {
	//TODO: This method doesn't return the whole block with transactions.
	//		It should be reimplemented with the complete JSON representation of block or using upcoming protobufs.
	p := chi.URLParam(r, "id")
	sig, err := crypto.NewSignatureFromBase58(p)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid block signature: %v", err), http.StatusBadRequest)
		return
	}
	block, ok, err := a.storage.block(sig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Block not found", http.StatusNotFound)
	}
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func countForksByLength(forks []Fork) (int, int) {
	r := 0
	for _, f := range forks {
		if f.Length < 10 {
			r++
		}
	}
	return r, len(forks) - r
}

func connectedPeersCount(peers []PeerDescription) int {
	r := 0
	for _, p := range peers {
		if p.Connected {
			r++
		}
	}
	return r
}
