package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/mux"
	"github.com/mr-tron/base58/base58"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/db"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Server struct {
	BootPeerAddrs []string
	Listen        string
	dbpath        string
	db            *db.WavesDB
	genesis       crypto.Signature
	peers         map[string]*Peer
	newPeers      chan proto.PeerInfo

	apiAddr string
	router  *mux.Router
	server  *http.Server
	ctx     context.Context
}

func (s *Server) Run(ctx context.Context) {
	if s.Listen == "" {
		return
	}

	l, err := net.Listen("tcp", s.Listen)

	if err != nil {
		return
	}
	defer l.Close()

	for {
		_, err := l.Accept()
		if err != nil {
			zap.S().Error("error while accepting connections: ", err)
			break
		}
	}
}

func (s *Server) printPeers() {
	for k, v := range s.peers {
		b, err := json.Marshal(v.State())
		if err != nil {
			zap.S().Error("failed to marshal peer state: ", k)
			continue
		}

		zap.S().Info("node: ", k, "state: ", string(b))
	}
}

func (s *Server) printStats(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			s.printPeers()
		}
	}
}

func (s *Server) updatePeers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case peer, ok := <-s.newPeers:
			if !ok {
				return
			}
			if _, ok := s.peers[peer.String()]; !ok {
				zap.S().Info("received new peer: ", peer.String())
				p, err := NewPeer(s.genesis, s.db,
					WithAddr(peer.String()),
					WithPeersChan(s.newPeers))
				if err != nil {
					continue
				}
				s.peers[peer.String()] = p
			}
		}
	}
}

func (s *Server) RunClients(ctx context.Context) {
	go s.printStats(ctx)

	s.ctx = ctx
	for _, addr := range s.BootPeerAddrs {
		fmt.Println(addr)
		peer, err := NewPeer(s.genesis, s.db,
			WithAddr(addr),
			WithPeersChan(s.newPeers))
		if err != nil {
			continue
		}
		s.peers[addr] = peer
	}
	go s.updatePeers(ctx)
}

func (s *Server) Stop() {
	for _, peer := range s.peers {
		peer.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	s.server.Shutdown(ctx)
	zap.S().Info("stopped server")
}

type Option func(*Server) error

func WithBindAddr(addr string) Option {
	return func(s *Server) error {
		s.apiAddr = addr
		return nil
	}
}

func WithLevelDBPath(dbpath string) Option {
	return func(s *Server) error {
		s.dbpath = dbpath
		return nil
	}
}

func WithPeers(peers []string) Option {
	return func(s *Server) error {
		s.BootPeerAddrs = peers
		return nil
	}
}

func WithGenesis(gen string) Option {
	return func(s *Server) error {
		if gen == "" {
			return nil
		}
		decoded, err := base58.Decode(gen)
		if err != nil {
			return err
		}
		copy(s.genesis[:], decoded[:len(s.genesis)])
		return nil
	}
}

func decodeBlockID(b string) (*crypto.Signature, error) {
	var res crypto.Signature

	decoded, err := base58.Decode(b)
	if err != nil {
		return nil, err
	}
	if len(decoded) != len(res) {
		return nil, fmt.Errorf("unexpected blockID length: want %v have %v", len(res), len(decoded))
	}
	copy(res[:], decoded)
	return &res, nil
}

func (s *Server) getBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["sig"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "no block signature specified")
		return
	}

	decoded, err := base58.Decode(id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid block signature")
		return
	}
	var blockID crypto.Signature
	copy(blockID[:], decoded)
	block, err := s.db.Get(blockID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "block not found")
		return
	}
	respondWithJSON(w, http.StatusOK, block)
}

func (s *Server) getNodes(w http.ResponseWriter, r *http.Request) {
	addrs := make([]string, 0, len(s.peers))
	for addr := range s.peers {
		addrs = append(addrs, addr)
	}
	respondWithJSON(w, http.StatusOK, addrs)
}

func (s *Server) getNodesVerbose(w http.ResponseWriter, r *http.Request) {
	states := make([]NodeState, 0, len(s.peers))
	for _, peer := range s.peers {
		states = append(states, peer.State())
	}
	respondWithJSON(w, http.StatusOK, states)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addr, ok := vars["addr"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "no node addr specified")
		return
	}

	peer, ok := s.peers[addr]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "no such node")
		return
	}

	respondWithJSON(w, http.StatusOK, peer.State())
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (s *Server) initRoutes() {
	s.router.HandleFunc("/blocks/signature/{sig:[a-zA-Z0-9]{88}}", s.getBlock).Methods("GET")
	s.router.HandleFunc("/nodes", s.getNodes).Methods("GET")
	s.router.HandleFunc("/node/{addr}", s.getNode).Methods("GET")
	s.router.HandleFunc("/nodes/verbose", s.getNodesVerbose).Methods("GET")
}

func (s *Server) startREST() {
	srv := &http.Server{
		Addr:         s.apiAddr,
		WriteTimeout: time.Second * 5,
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 60,
		Handler:      s.router,
	}
	s.server = srv

	go func() {
		zap.S().Info("starting REST API on ", s.apiAddr)
		if err := srv.ListenAndServe(); err != nil {
			zap.S().Error(err)
		}
	}()
}

func NewServer(opts ...Option) (*Server, error) {
	s := &Server{
		peers: make(map[string]*Peer),
	}

	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	if s.dbpath != "" {
		db, err := db.NewDB(s.dbpath, s.genesis)
		if err != nil {
			return nil, err
		}
		s.db = db
	}

	s.newPeers = make(chan proto.PeerInfo, 1024)
	s.router = mux.NewRouter()
	s.initRoutes()
	s.startREST()

	zap.S().Info("staring server with genesis block", base58.Encode(s.genesis[:]))
	return s, nil
}
