package server

import (
	"bufio"
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

func (s *Server) processSignatures(w bufio.Writer, r bufio.Reader, m proto.SignaturesMessage) {
}

func (s *Server) handleClient(ctx context.Context, peer string) {
	ingress := make(chan interface{}, 1024)

	zap.S().Info("handling client")

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
	defer conn.Close()

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case m := <-ingress:
			switch v := m.(type) {
			case proto.SignaturesMessage:
				var b []byte
				b, e := json.Marshal(v)
				if e != nil {
					return
				}
				js := string(b)
				zap.S().Info("Got signatures", js)
			default:
				zap.S().Infof("Got type %T", v)
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

func (m *Server) RunClients(ctx context.Context) {
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
		conns: make(map[*p2p.Conn]bool),
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
