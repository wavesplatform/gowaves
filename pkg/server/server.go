package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gorilla/mux"
	"github.com/mr-tron/base58/base58"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/db"
	"github.com/wavesplatform/gowaves/pkg/p2p"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Server struct {
	BootPeerAddrs []string
	Listen        string
	wg            sync.WaitGroup
	mu            sync.Mutex
	conns         map[*p2p.Conn]bool
	dbpath        string
	db            *db.WavesDB
	genesis       crypto.Signature
	nodeStates    map[string]*NodeState

	apiAddr string
	router  *mux.Router
	server  *http.Server
}

func handleRequest(ctx context.Context, conn net.Conn) {
}

func (s *Server) dialContext(v proto.Version) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		var major, minor, patch uint32

		major = v.Major
		minor = v.Minor
		patch = v.Patch
		dialer := net.Dialer{}

		for i := minor; i > 0; i-- {
			if i < v.Minor {
				ticker := time.NewTimer(16 * time.Minute)

				select {
				case <-ticker.C:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			minor = uint32(i)
			zap.S().Infof("Trying to connect with version %v.%v.%v", major, minor, patch)
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				continue
			}

			handshake := proto.Handshake{Name: "wavesT",
				Version:           proto.Version{Major: major, Minor: minor, Patch: patch},
				NodeName:          "gowaves",
				NodeNonce:         0x0,
				DeclaredAddrBytes: []byte{},
				Timestamp:         uint64(time.Now().Unix())}

			_, err = handshake.WriteTo(conn)
			if err != nil {
				zap.S().Error("failed to send handshake: ", err)
				continue
			}
			_, err = handshake.ReadFrom(conn)
			if err != nil {
				zap.S().Error("failed to read handshake: ", err)
				continue
			}

			s.setKnownVersion(addr, handshake.Version)

			var b []byte
			b, e := json.Marshal(handshake)
			if e != nil {
				return nil, err
			}
			js := string(b)
			zap.S().Info("received handshake: ", js)

			return conn, nil
		}
		return nil, errors.New("TODO")
	}
}

func (s *Server) processSignatures(conn *p2p.Conn, m proto.SignaturesMessage) []crypto.Signature {
	unknownBlocks := make([]crypto.Signature, 0, len(m.Signatures))

	zap.S().Info("signatures len ", len(m.Signatures))
	zap.S().Info("signatures from ", base58.Encode(m.Signatures[0][:]), " ", base58.Encode(m.Signatures[len(m.Signatures)-1][:]))
	for _, sig := range m.Signatures {
		has, err := s.db.Has(sig)
		if err != nil {
			zap.S().Error("failed to query leveldb: ", err)
			continue
		}

		if !has {
			unknownBlocks = append(unknownBlocks, sig)
			//zap.S().Debug("asking for block ", i, " ", base58.Encode(sig[:]))
			var blockID crypto.Signature
			copy(blockID[:], sig[:])
			gbm := proto.GetBlockMessage{BlockID: blockID}
			if err = conn.SendMessage(gbm); err != nil {
				zap.S().Error("failed to send get block message ", err)
				break
			}
		}
	}

	return unknownBlocks
}

func (s *Server) waitForBlocks(conn *p2p.Conn, blocks []crypto.Signature) (*blockBatch, error) {
	batch, err := NewBatch(blocks)
	if err != nil {
		return nil, err
	}

	for !batch.haveAll() {
		msg, err := conn.ReadMessage()
		if err != nil && err != p2p.ErrUnknownMessage {
			zap.S().Error("got error ", err)
			return nil, err
		}

		switch v := msg.(type) {
		case proto.BlockMessage:
			var b proto.Block
			if err = b.UnmarshalBinary(v.BlockBytes); err != nil {
				zap.S().Info("failed to unmarshal block ", err)
				continue
			}
			batch.addBlock(&b)
		default:
			zap.S().Infof("got message of type %T", v)
		}
	}

	zap.S().Info("received all blocks")

	return batch, nil
}

func (s *Server) loadState(peer string) {
	stateBytes, err := s.db.GetRaw([]byte(peer))
	var state NodeState
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		state.State = stateSyncing

		state.LastKnownBlock = s.genesis
		state.Addr = peer

		s.nodeStates[peer] = &state
		zap.S().Info("storage has no info about node ", peer)
		return
	}

	zap.S().Info("state is ", string(stateBytes))
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		zap.S().Info("failed to parse node ", peer, " state: ", err)
		return
	}
	str, err := json.Marshal(state)
	if err != nil {
		zap.S().Error("failed to marshal binary: ", err)
		return
	}
	zap.S().Info("loaded node ", peer, " state: ", string(str))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeStates[peer] = &state
}

func (s *Server) storeState(peer string) {
	var state *NodeState

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.nodeStates[peer]
	if !ok {
		return
	}

	bytes, err := json.Marshal(state)
	if err != nil {
		zap.S().Error("failed to marshal peer state: ", err)
		return
	}
	if err := s.db.PutRaw([]byte(peer), bytes); err != nil {
		zap.S().Error("failed to store peer state in db: ", err)
	}
	zap.S().Info("stored state ", peer, " ", string(bytes))
}

func (s *Server) lastKnownBlock(conn *p2p.Conn) crypto.Signature {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]
	return state.LastKnownBlock
}

func (s *Server) setLastKnownBlock(conn *p2p.Conn, block crypto.Signature) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]
	state.LastKnownBlock = block
}

func (s *Server) setKnownVersion(addr string, v proto.Version) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[addr]
	state.KnownVersion = v
}

func (s *Server) processBatch(batch []*proto.Block) error {
	for _, block := range batch {
		if err := s.db.Put(block); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) jumpBack(conn *p2p.Conn, n int) {
	last := s.lastKnownBlock(conn)

	for i := 0; i < n; i++ {
		lastB, err := s.db.Get(last)
		if err != nil {
			last = s.genesis
			break
		}
		last = lastB.Parent
	}

	zap.S().Info("unwinded back to block ", base58.Encode(last[:]))
	s.setLastKnownBlock(conn, last)
}

func (s *Server) syncState(conn *p2p.Conn) error {
LOOP:
	for {
		var gs proto.GetSignaturesMessage
		s.mu.Lock()
		state, ok := s.nodeStates[conn.RemoteAddr().String()]
		if !ok {
			break
		}
		gs.Blocks = make([]crypto.Signature, 1)
		gs.Blocks[0] = state.LastKnownBlock
		s.mu.Unlock()

		zap.S().Info("Asking for signatures")
		conn.SendMessage(gs)
		sigDeadLine := time.Now().Add(time.Second * 10)
	LOOP2:
		for {
			msg, err := conn.ReadWithDeadline(sigDeadLine)
			if netE, ok := err.(net.Error); ok {
				if netE.Timeout() {
					zap.S().Info("signatures request timed out")
					s.jumpBack(conn, 10)
					break
				}
			}
			if err != nil && err != p2p.ErrUnknownMessage {
				break LOOP
			}

			switch v := msg.(type) {
			case proto.SignaturesMessage:
				zap.S().Info("got signatures message from ", conn.RemoteAddr().String())
				unknown := s.processSignatures(conn, v)
				if len(v.Signatures) == 1 {
					break LOOP
				}
				if len(unknown) == 0 {
					zap.S().Info("have all blocks")
					s.setLastKnownBlock(conn, v.Signatures[len(v.Signatures)-1])
					break LOOP2
				}

				batch, err := s.waitForBlocks(conn, unknown)
				if err != nil {
					break LOOP
				}

				orBatch, err := batch.orderedBatch()
				if err != nil {
					zap.S().Error(err)
				}
				zap.S().Info("batch of length ", len(orBatch), " first block ",
					base58.Encode(orBatch[0].BlockSignature[:]), " last block ",
					base58.Encode(orBatch[len(orBatch)-1].BlockSignature[:]))

				err = s.processBatch(orBatch)
				if err != nil {
					zap.S().Info("failed to process batch: ", err)
				}
				s.setLastKnownBlock(conn, orBatch[len(orBatch)-1].BlockSignature)
				break LOOP2
			default:
				zap.S().Infof("got message of type %T", v)
			}
		}
	}

	return nil
}

func (s *Server) updateState(conn *p2p.Conn) error {
	for {
		msg, err := conn.ReadMessage()

		if err != nil {
			if err == p2p.ErrUnknownMessage {
				continue
			}
			zap.S().Info("failed to receive message ", err)
			break
		}

		switch v := msg.(type) {
		case proto.BlockMessage:
			var b proto.Block
			if err = b.UnmarshalBinary(v.BlockBytes); err != nil {
				zap.S().Info("failed to unmarshal block ", err)
				continue
			}
			lastKnown := s.lastKnownBlock(conn)
			last, _ := s.db.Get(lastKnown)
			if b.Parent == last.BlockSignature {
				s.db.Put(&b)
				s.setLastKnownBlock(conn, b.BlockSignature)
				continue
			}

			s.jumpBack(conn, 10)

			s.syncState(conn)
		default:
			zap.S().Infof("got message %T", msg)
		}
	}

	return nil
}

func (s *Server) serveConn(conn *p2p.Conn) error {
	err := s.syncState(conn)
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
		return err
	}

	err = s.updateState(conn)
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
		return err
	}

	return nil
}

func (s *Server) addConnection(conn *p2p.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conns[conn] = true
}

func (s *Server) delConnection(conn *p2p.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conns, conn)
}

func (s *Server) newConnection(ctx context.Context, peer string) error {
	zap.S().Info("handling client", peer)

	s.loadState(peer)
	defer s.storeState(peer)

	customTransport := p2p.Transport{DialContext: s.dialContext(proto.Version{Major: 0, Minor: 15, Patch: 1})}
	conn, err := p2p.NewConn(
		p2p.WithTransport(&customTransport),
		p2p.WithRemote("tcp", peer),
	)
	if err != nil {
		zap.S().Error("failed to create a new connection: ", err)
		return err
	}
	if err = conn.DialContext(ctx, "tcp", peer); err != nil {
		zap.S().Error("error while dialing: ", err)
		return err
	}
	s.addConnection(conn)
	defer conn.Close()
	defer s.delConnection(conn)

	if err = s.serveConn(conn); err != nil {
		return err
	}

	return nil
}

func (s *Server) handleClient(ctx context.Context, peer string) {
	for {
		res := make(chan error, 1)
		go func() {
			err := s.newConnection(ctx, peer)
			res <- err
		}()

		select {
		case <-ctx.Done():
			return
		case err := <-res:
			if err != nil {
				zap.S().Error("connection to peer ", peer, " failed: ", err)
			}
		}
	}
}

func (m *Server) Run(ctx context.Context) {
	if m.Listen == "" {
		return
	}

	l, err := net.Listen("tcp", m.Listen)

	if err != nil {
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			zap.S().Error("error while accepting connections: ", err)
			break
		}

		m.wg.Add(1)
		go func(conn net.Conn) {
			handleRequest(ctx, conn)
			m.wg.Done()
		}(conn)
	}
}

func (s *Server) printPeers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.nodeStates {
		b, err := json.Marshal(v)
		if err != nil {
			zap.S().Error("failed to marshal peer state: ", k)
			continue
		}

		zap.S().Info("node: ", k, "state: ", string(b))
	}
}

func (s *Server) printStats(ctx context.Context) {
	defer s.wg.Done()
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-time.After(5 * time.Second):
			s.printPeers()
		}
	}
}

func (m *Server) RunClients(ctx context.Context) {
	m.wg.Add(1)
	go m.printStats(ctx)

	for _, peer := range m.BootPeerAddrs {
		m.wg.Add(1)
		go func(peer string) {
			m.handleClient(ctx, peer)
			m.wg.Done()
		}(peer)
	}
}

func (m *Server) Stop() {
	m.mu.Lock()
	for k := range m.conns {
		k.Close()
	}
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	m.server.Shutdown(ctx)
	m.wg.Wait()
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
	s.mu.Lock()
	addrs := make([]string, 0, len(s.nodeStates))
	for _, state := range s.nodeStates {
		addrs = append(addrs, state.Addr)
	}
	s.mu.Unlock()
	respondWithJSON(w, http.StatusOK, addrs)
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addr, ok := vars["addr"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "no node addr specified")
		return
	}
	s.mu.Lock()

	state, ok := s.nodeStates[addr]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "no such node")
		return
	}
	stateCopy := *state
	s.mu.Unlock()

	respondWithJSON(w, http.StatusOK, stateCopy)
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
		conns:      make(map[*p2p.Conn]bool),
		nodeStates: make(map[string]*NodeState),
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

	s.router = mux.NewRouter()
	s.initRoutes()
	s.startREST()

	zap.S().Info("staring server with genesis block", base58.Encode(s.genesis[:]))
	return s, nil
}
