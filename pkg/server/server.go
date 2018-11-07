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
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/p2p"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	db            *leveldb.DB
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

func (s *Server) doHaveAllBlocks(n *NodeState) bool {
	for _, v := range n.pendingBlocksHave {
		if !v {
			return v
		}
	}
	return true
}

func (s *Server) processSignatures(c *p2p.Conn, m proto.SignaturesMessage, from string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.nodeStates[from]
	state.pendingSignatures = make([]proto.BlockSignature, len(m.Signatures))
	copy(state.pendingSignatures, m.Signatures)

	for _, sig := range m.Signatures {
		has, err := s.db.Has(sig[:], nil)
		state.pendingBlocksHave[sig] = false

		if err != nil {
			zap.S().Error("failed to query leveldb: ", err)
			continue
		}

		if !has {
			zap.S().Debug("asking for block", base58.Encode(sig[:]))
			var blockID proto.BlockID
			copy(blockID[:], sig[:])
			gbm := proto.GetBlockMessage{BlockID: blockID}
			c.Send() <- gbm
			continue
		}
		state.pendingBlocksHave[sig] = true
	}

	if s.doHaveAllBlocks(state) {
		zap.S().Info("have all blocks")
	}
}

func (s *Server) processBlock(c *p2p.Conn, m proto.BlockMessage, from string) {
	msg := m
	var b proto.Block
	b.UnmarshalBinary(msg.BlockBytes)
	str := base58.Encode(b.BlockSignature[:])

	//zap.S().Infow("got block", "block", str)

	has, err := s.db.Has(b.BlockSignature[:], nil)

	if err != nil {
		zap.S().Error("failed to query leveldb", err)
		return
	}

	if !has {
		zap.S().Infof("block %v not found in db", str)
		if err = s.db.Put(b.BlockSignature[:], msg.BlockBytes, nil); err != nil {
			zap.S().Error("failed to query leveldb", err)
		}
	}

	return
}

func (s *Server) loadState(peer string) {
	stateBytes, err := s.db.Get([]byte(peer), nil)
	var state NodeState
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		state.State = stateSyncing

		state.LastKnownBlock = s.genesis
		state.pendingBlocksHave = make(map[proto.BlockSignature]bool, 0)
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
	if err := s.db.Put([]byte(peer), bytes, nil); err != nil {
		zap.S().Error("failed to store peer state in db: ", err)
	}
}

func (s *Server) handleClient(ctx context.Context, peer string) {
	ingress := make(chan p2p.ConnMessage, 1024)

	zap.S().Info("handling client")

	s.loadState(peer)

	customTransport := p2p.Transport{DialContext: dialContext}
	conn, err := p2p.NewConn(
		p2p.WithVersion(proto.Version{Major: 0, Minor: 5, Patch: 14}),
		p2p.WithTransport(&customTransport),
		p2p.WithRemote("tcp", peer),
		p2p.WithIngress(ingress),
	)
	s.mu.Lock()
	s.conns[conn] = true
	s.mu.Unlock()

	if err != nil {
		zap.S().Error("failed to create a new connection: ", err)
		return
	}
	err = conn.DialContext(ctx, "tcp", peer)
	if err != nil {
		zap.S().Error("error while dialing: ", err)
		return
	}

	conn.Run()

	var gp proto.GetSignaturesMessage
	gp.Blocks = append(gp.Blocks, s.genesis)

	conn.Send() <- gp
	conn.Send() <- gp
	defer conn.Close()

LOOP:
	for {
		select {
		case <-ctx.Done():
			zap.S().Info("stopping connection loop")
			break LOOP
		case m := <-ingress:
			switch v := m.Message.(type) {
			case proto.SignaturesMessage:
				s.processSignatures(conn, v, m.From.String())
			case proto.BlockMessage:
				s.processBlock(conn, v, m.From.String())
			default:
				zap.S().Infof("Got type %T", v)
			}
		}

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
		db, err := leveldb.OpenFile(s.dbpath, nil)
		if err != nil {
			return nil, err
		}
		s.db = db
	}

	zap.S().Info("staring server with genesis block", base58.Encode(s.genesis[:]))
	return s, nil
}
