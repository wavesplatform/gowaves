package httpserver

import (
	"context"
	"encoding/json"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"net/http"
	"net/http/pprof"
	"sort"

	"github.com/gorilla/mux"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type HttpServer struct {
	retransmitter Retransmitter
	srv           http.Server
}

type Retransmitter interface {
	Counter() *utils.Counter
	KnownPeers() *utils.KnownPeers
	SpawnedPeers() *utils.SpawnedPeers
	ActiveConnections() *utils.Addr2Peers
}

func NewHttpServer(r Retransmitter) *HttpServer {
	return &HttpServer{
		retransmitter: r,
	}
}

type ActiveConnection struct {
	Addr          string        `json:"addr"`
	DeclAddr      string        `json:"decl_addr"`
	Direction     string        `json:"direction"`
	RemoteAddr    string        `json:"remote_addr"`
	LocalAddr     string        `json:"local_addr"`
	Version       proto.Version `json:"version"`
	AppName       string        `json:"app_name"`
	NodeName      string        `json:"node_name"`
	SendClosed    bool          `json:"send_closed"`
	ReceiveClosed bool          `json:"receive_closed"`
}

type ActiveConnections []ActiveConnection

func (a ActiveConnections) Len() int           { return len(a) }
func (a ActiveConnections) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ActiveConnections) Less(i, j int) bool { return a[i].Addr < a[j].Addr }

type FullState struct {
	Active  []ActiveConnection
	Spawned []string
	Known   []string
}

func (a *HttpServer) ActiveConnections(rw http.ResponseWriter, r *http.Request) {
	var out ActiveConnections
	addr2peer := a.retransmitter.ActiveConnections()
	addr2peer.Each(func(id string, p peer.Peer) {
		c := p.Connection()
		out = append(out, ActiveConnection{
			Addr:          id,
			Direction:     p.Direction().String(),
			DeclAddr:      p.Handshake().DeclaredAddr.String(),
			RemoteAddr:    p.RemoteAddr().String(),
			LocalAddr:     p.Connection().Conn().LocalAddr().String(),
			Version:       p.Handshake().Version,
			AppName:       p.Handshake().AppName,
			NodeName:      p.Handshake().NodeName,
			SendClosed:    c.SendClosed(),
			ReceiveClosed: c.ReceiveClosed(),
		})
	})

	sort.Sort(out)

	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}

func (a *HttpServer) KnownPeers(rw http.ResponseWriter, r *http.Request) {
	out := a.retransmitter.KnownPeers().GetAll()
	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}

func (a *HttpServer) Spawned(rw http.ResponseWriter, r *http.Request) {
	out := a.retransmitter.SpawnedPeers().GetAll()
	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}

func (a *HttpServer) counter(rw http.ResponseWriter, r *http.Request) {
	c := a.retransmitter.Counter()
	out := c.Get()
	bts, err := json.Marshal(out)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(bts)
}

func (a *HttpServer) ListenAndServe() error {
	router := mux.NewRouter()
	router.HandleFunc("/active", a.ActiveConnections)
	router.HandleFunc("/known", a.KnownPeers)
	router.HandleFunc("/spawned", a.Spawned)
	router.HandleFunc("/counter", a.counter)

	// Register pprof handlers
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))

	http.Handle("/", router)

	a.srv = http.Server{
		Handler: router,
		Addr:    "0.0.0.0:8000",
	}
	return a.srv.ListenAndServe()
}

func (a *HttpServer) Shutdown(ctx context.Context) error {
	return a.srv.Shutdown(ctx)
}
