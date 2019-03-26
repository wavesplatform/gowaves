package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
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
	storage   *storage
	registry  *PeerRegistry
	srv       *http.Server
}

type PublicAddressInfo struct {
	Address         string    `json:"address"`
	Version         string    `json:"version"`
	Status          string    `json:"status"`
	Attempts        int       `json:"attempts"`
	NextAttemptTime time.Time `json:"next_attempt_time"`
}

func newInfoFromPublicAddress(pa PublicAddress) PublicAddressInfo {
	status := "UNKNOWN"
	switch pa.state {
	case NewPublicAddress:
		status = "NEW"
	case HostilePublicAddress:
		status = "HOSTILE"
	case GreetedPublicAddress:
		status = "GREETED"
	case RespondingPublicAddress:
		status = "RESPONDING"
	case DiscardedPublicAddress:
		status = "DISCARDED"
	}
	return PublicAddressInfo{
		Address:         pa.address.String(),
		Version:         pa.version.String(),
		Status:          status,
		Attempts:        pa.attempts,
		NextAttemptTime: pa.nextAttempt,
	}
}

func NewAPI(interrupt <-chan struct{}, storage *storage, registry *PeerRegistry, bind string) (*api, error) {
	if bind == "" {
		return nil, errors.New("empty address to bin")
	}
	a := api{interrupt: interrupt, storage: storage, registry: registry}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger(zap.L()))
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.DefaultCompress)
	r.Mount("/api", a.routes())
	a.srv = &http.Server{Addr: bind, Handler: r}
	return &a, nil
}

func (a *api) Start() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		err := a.srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.S().Fatalf("Failed to start API: %v", err)
			close(done)
			return
		}
	}()
	select {
	case <-done:
		return done
	default:
	}
	go func() {
		for {
			select {
			case <-a.interrupt:
				zap.S().Debug("Shutting down API...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := a.srv.Shutdown(ctx)
				if err != nil {
					zap.S().Errorf("Failed to shutdown API server: %v", err)
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
	r.Get("/addresses", a.addresses)
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
	pas, err := a.storage.publicAddresses()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	s := status{ShortForksCount: short, LongForksCount: long, ConnectedNodesCount: 0, KnowNodesCount: len(pas)}
	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) addresses(w http.ResponseWriter, r *http.Request) {
	pas, err := a.storage.publicAddresses()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete request: %v", err), http.StatusInternalServerError)
		return
	}
	infos := make([]PublicAddressInfo, len(pas))
	for i, pa := range pas {
		info := newInfoFromPublicAddress(pa)
		infos[i] = info
	}
	err = json.NewEncoder(w).Encode(infos)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal addresses to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *api) peers(w http.ResponseWriter, r *http.Request) {
	peers := a.registry.Peers()
	err := json.NewEncoder(w).Encode(peers)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal peers to JSON: %v", err), http.StatusInternalServerError)
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
