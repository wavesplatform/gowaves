package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/mr-tron/base58/base58"
	"github.com/wavesplatform/gowaves/pkg/p2p"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/db"
)

const (
	testnetGenesis = "5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"
)

type Server struct {
	BootPeerAddrs []string
	Listen        string
	wg            sync.WaitGroup
	mu            sync.Mutex
	conns         map[*p2p.Conn]bool
	dbpath        string
	db            *db.WavesDB
	genesis       proto.BlockID
	nodeStates    map[string]*NodeState
}

func handleRequest(ctx context.Context, conn net.Conn) {
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var major, minor, patch uint32

	dialer := net.Dialer{}

	for i := 0xe; i < 20; i++ {
		if i > 0xe {
			ticker := time.NewTimer(16 * time.Minute)

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		minor = uint32(i)
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			continue
		}

		zap.S().Infof("Trying to connect with version %v.%v.%v", major, minor, patch)
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

func (s *Server) doHaveAllBlocks(conn *p2p.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]

	for _, v := range state.pendingBlocksHave {
		if !v {
			return v
		}
	}
	return true
}

func (s *Server) addPendingBlock(conn *p2p.Conn, block proto.BlockID, have bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]

	state.pendingBlocksHave[block] = have
}

func (s *Server) clearWaitingState(conn *p2p.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]

	state.pendingSignatures = make([]proto.BlockID, 0, 128)
	state.pendingBlocksHave = make(map[proto.BlockID]bool)
}

func (s *Server) receivedBlock(conn *p2p.Conn, block proto.Block) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[conn.RemoteAddr().String()]

	if _, ok := state.pendingBlocksHave[block.BlockSignature]; ok {
		state.pendingBlocksHave[block.BlockSignature] = true
	}
}

func (s *Server) processSignatures(conn *p2p.Conn, m proto.SignaturesMessage) {
	s.clearWaitingState(conn)

	zap.S().Info("signatures len ", len(m.Signatures))
	zap.S().Info("signatures from ", base58.Encode(m.Signatures[0][:]), " ", base58.Encode(m.Signatures[len(m.Signatures)-1][:]))
	for i, sig := range m.Signatures {
		has, err := s.db.Has(sig)
		//state.pendingBlocksHave[sig] = false

		s.addPendingBlock(conn, sig, has)
		if err != nil {
			zap.S().Error("failed to query leveldb: ", err)
			continue
		}

		if !has {
			zap.S().Debug("asking for block ", i, " ", base58.Encode(sig[:]))
			var blockID proto.BlockID
			copy(blockID[:], sig[:])
			gbm := proto.GetBlockMessage{BlockID: blockID}
			if err = conn.SendMessage(gbm); err != nil {
				zap.S().Error("failed to send get block message ", err)
				break
			}

		}
	}

}

func (s *Server) waitForBlocks(conn *p2p.Conn) {
	for !s.doHaveAllBlocks(conn) {
		msg, err := conn.ReadMessage()
		if err != nil {
			zap.S().Error("got error ", err)
			break
		}

		switch v := msg.(type) {
		case proto.BlockMessage:
			s.processBlock(conn, v)
		default:
			zap.S().Infof("got message of type %T", v)
		}
	}

	zap.S().Info("received all blocks")
}

func (s *Server) processBlock(c *p2p.Conn, m proto.BlockMessage) {
	msg := m
	var b proto.Block
	b.UnmarshalBinary(msg.BlockBytes)
	str := base58.Encode(b.BlockSignature[:])

	zap.S().Info("got block ", str, " from ", c.RemoteAddr().String())

	has, err := s.db.Has(b.BlockSignature)

	if err != nil {
		zap.S().Error("failed to query leveldb", err)
		return
	}

	if !has {
		zap.S().Infof("block %v not found in db", str)
		if err = s.db.Put(&b); err != nil {
			switch err {
			case db.ErrBlockOrphaned:
				zap.S().Error("the block is orphaned, cannot write to db")
			default:
				zap.S().Error("failed to query leveldb", err)
			}
		}
	}

	s.receivedBlock(c, b)

	return
}

func (s *Server) loadState(peer string) {
	stateBytes, err := s.db.GetRaw([]byte(peer))
	var state NodeState
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		state.State = stateSyncing

		state.LastKnownBlock = s.genesis
		state.pendingBlocksHave = make(map[proto.BlockID]bool, 0)
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

func (s *Server) syncState(conn *p2p.Conn) error {
LOOP:
	for {
		var gs proto.GetSignaturesMessage
		s.mu.Lock()
		state, ok := s.nodeStates[conn.RemoteAddr().String()]
		s.mu.Unlock()
		if !ok {
			break
		}
		gs.Blocks = append(gs.Blocks, state.LastKnownBlock)

		zap.S().Info("Asking for signatures")
		conn.SendMessage(gs)
		conn.SendMessage(gs)
		for {
			msg, err := conn.ReadMessage()
			if err != nil {
				break LOOP
			}

			switch v := msg.(type) {
			case proto.SignaturesMessage:
				zap.S().Info("got signatures message from ", conn.RemoteAddr().String())
				s.processSignatures(conn, v)
				if !s.doHaveAllBlocks(conn) {
					s.waitForBlocks(conn)
				} else {
					zap.S().Info("have all blocks")
				}
				//break LOOP
			default:
				zap.S().Infof("got message of type %T", v)
			}
		}
	}

	return nil
}

func (s *Server) serveConn(conn *p2p.Conn) {
	err := s.syncState(conn)
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
	}
}

func (s *Server) handleClient(ctx context.Context, peer string) {
	zap.S().Info("handling client")

	s.loadState(peer)

	customTransport := p2p.Transport{DialContext: dialContext}
	conn, err := p2p.NewConn(
		p2p.WithVersion(proto.Version{Major: 0, Minor: 5, Patch: 14}),
		p2p.WithTransport(&customTransport),
		p2p.WithRemote("tcp", peer),
	)
	if err != nil {
		zap.S().Error("failed to create a new connection: ", err)
		return
	}
	if err = conn.DialContext(ctx, "tcp", peer); err != nil {
		zap.S().Error("error while dialing: ", err)
		return
	}
	defer conn.Close()
	s.mu.Lock()
	s.conns[conn] = true
	s.mu.Unlock()

	go s.serveConn(conn)

	select {
	case <-ctx.Done():
		zap.S().Info("cancelled")
	}

	s.storeState(peer)
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

	m.wg.Wait()
	zap.S().Info("stopped server")
}

type Option func(*Server) error

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

func decodeBlockID(b string) (*proto.BlockID, error) {
	var res proto.BlockID

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

func NewServer(opts ...Option) (*Server, error) {
	s := &Server{
		conns:      make(map[*p2p.Conn]bool),
		nodeStates: make(map[string]*NodeState),
	}
	genesis, err := decodeBlockID(testnetGenesis)
	if err != nil {
		return nil, err
	}
	s.genesis = *genesis
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

	zap.S().Info("staring server with genesis block", base58.Encode(s.genesis[:]))
	return s, nil
}
