package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/mr-tron/base58/base58"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/db"
	"github.com/wavesplatform/gowaves/pkg/p2p"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Peer manages all the connection logic with a single peer
//
// It may be used in a two ways: as a client connection and as a server connection
//
// When using as a client connection it should be created with a remote
// peer addr specified. In that case it will attempt to dial this addr
// and then continue processing with a dialed connection
//
// When using as a server connection, it is created with an existing *p2p.Conn
// that was previously accepted on a listening server
type Peer struct {
	addr     string
	conn     *p2p.Conn
	db       *db.WavesDB
	state    NodeState
	genesis  crypto.Signature
	declAddr proto.PeerInfo

	peers chan proto.PeerInfo

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (p *Peer) dialContext(v proto.Version) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		var major, minor, patch uint32

		major = v.Major
		minor = v.Minor
		patch = v.Patch
		dialer := net.Dialer{}

		for i := minor; i > 0; i-- {
			if i < v.Minor {
				ticker := time.NewTimer(31 * time.Minute)

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

			bytes, err := p.declAddr.MarshalBinary()
			if err != nil {
				return nil, err
			}

			handshake := proto.Handshake{Name: "wavesT",
				Version:           proto.Version{Major: major, Minor: minor, Patch: patch},
				NodeName:          "gowaves",
				NodeNonce:         0x0,
				DeclaredAddrBytes: bytes,
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

			p.state.KnownVersion = handshake.Version
			return conn, nil
		}

		return nil, errors.New("failed to dial peer")
	}

}

func (p *Peer) handshake(conn net.Conn) error {
	var handshake proto.Handshake
	_, err := handshake.ReadFrom(conn)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
		return err
	}

	bytes, err := p.declAddr.MarshalBinary()
	if err != nil {
		return err
	}

	handshake.NodeName = "gowaves"
	handshake.NodeNonce = 0
	handshake.DeclaredAddrBytes = bytes
	handshake.Timestamp = uint64(time.Now().Unix())

	_, err = handshake.WriteTo(conn)
	if err != nil {
		zap.S().Error("failed to send handshake: ", err)
		return err
	}

	p.state.KnownVersion = handshake.Version
	return nil
}

func (p *Peer) connect() error {
	customTransport := p2p.Transport{DialContext: p.dialContext(
		proto.Version{
			Major: 0,
			Minor: 15,
			Patch: 1,
		})}

	conn, err := p2p.NewConn(
		p2p.WithTransport(&customTransport),
		p2p.WithRemote("tcp", p.addr),
	)

	if err != nil {
		return err
	}
	if err = conn.DialContext(p.ctx, "tcp", p.addr); err != nil {
		return err
	}
	p.state.State = stateConnected
	p.conn = conn
	return nil
}

func (p *Peer) State() NodeState {
	return p.state
}

func (p *Peer) jumpBack(n int) {
	last := p.state.LastKnownBlock

	for i := 0; i < n; i++ {
		lastB, err := p.db.Get(last)
		if err != nil {
			last = p.genesis
			break
		}
		last = lastB.Parent
	}

	zap.S().Info("unwinded back to block ", base58.Encode(last[:]))
	p.state.LastKnownBlock = last
}

func (p *Peer) processSignatures(m proto.SignaturesMessage) []crypto.Signature {
	unknownBlocks := make([]crypto.Signature, 0, len(m.Signatures))

	zap.S().Info("signatures len ", len(m.Signatures))
	zap.S().Info("signatures from ", base58.Encode(m.Signatures[0][:]), " ", base58.Encode(m.Signatures[len(m.Signatures)-1][:]))
	for _, sig := range m.Signatures {
		has, err := p.db.Has(sig)
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
			if err = p.conn.SendMessage(gbm); err != nil {
				zap.S().Error("failed to send get block message ", err)
				break
			}
		}
	}

	return unknownBlocks
}

func (p *Peer) waitForBlocks(blocks []crypto.Signature) (*blockBatch, error) {
	batch, err := NewBatch(blocks)
	if err != nil {
		return nil, err
	}

	for !batch.haveAll() {
		msg, err := p.conn.ReadMessage()
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

func (p *Peer) processBatch(batch []*proto.Block) error {
	for _, block := range batch {
		if err := p.db.Put(block); err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) syncState() error {
LOOP:
	for {
		var gs proto.GetSignaturesMessage
		gs.Blocks = make([]crypto.Signature, 1)
		known := p.state.LastKnownBlock
		gs.Blocks[0] = known

		zap.S().Info("Asking for signatures")
		p.conn.SendMessage(gs)
		sigDeadLine := time.Now().Add(time.Second * 10)
	LOOP2:
		for {
			msg, err := p.conn.ReadWithDeadline(sigDeadLine)
			if netE, ok := err.(net.Error); ok {
				if netE.Timeout() {
					zap.S().Info("signatures request timed out")
					p.jumpBack(10)
					break
				}
			}
			if err != nil && err != p2p.ErrUnknownMessage {
				break LOOP
			}

			switch v := msg.(type) {
			case proto.SignaturesMessage:
				zap.S().Info("got signatures message from ", p.conn.RemoteAddr().String())
				unknown := p.processSignatures(v)
				if len(v.Signatures) == 1 {
					break LOOP
				}
				if len(unknown) == 0 {
					zap.S().Info("have all blocks")
					p.state.LastKnownBlock = v.Signatures[len(v.Signatures)-1]
					break LOOP2
				}

				batch, err := p.waitForBlocks(unknown)
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

				err = p.processBatch(orBatch)
				if err != nil {
					zap.S().Info("failed to process batch: ", err)
				}
				p.state.LastKnownBlock = orBatch[len(orBatch)-1].BlockSignature
				break LOOP2
			case proto.GetPeersMessage:
				var b proto.PeersMessage
				p.conn.SendMessage(b)
			default:
				zap.S().Infof("got message of type %T", v)
			}
		}
	}

	return nil
}

func (p *Peer) updateState() error {
	for {
		msg, err := p.conn.ReadMessage()

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
			lastKnown := p.state.LastKnownBlock
			last, _ := p.db.Get(lastKnown)
			if b.Parent == last.BlockSignature {
				p.db.Put(&b)
				p.state.LastKnownBlock = b.BlockSignature
				continue
			}

			p.jumpBack(10)

			p.syncState()
		case proto.GetPeersMessage:
			var b proto.PeersMessage
			p.conn.SendMessage(b)
		default:
			zap.S().Infof("got message %T", msg)
		}
	}

	return nil
}

func (p *Peer) getPeers() error {
	var gp proto.GetPeersMessage
	p.conn.SendMessage(gp)

	for {
		msg, err := p.conn.ReadMessage()

		if err != nil {
			return err
		}
		if v, ok := msg.(proto.PeersMessage); ok {
			for _, peer := range v.Peers {
				if p.peers != nil {
					p.peers <- peer
				}
				addr := peer.String()
				zap.S().Info(addr)
			}

			return nil
		}
	}
}

func (p *Peer) loadState() {
	stateBytes, err := p.db.GetRaw([]byte(p.addr))
	var state NodeState
	if err != nil {
		state.State = stateConnecting

		state.LastKnownBlock = p.genesis
		state.Addr = p.addr

		p.state = state
		zap.S().Info("storage has no info about node ", p.addr)
		return
	}

	zap.S().Info("state is ", string(stateBytes))
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		zap.S().Info("failed to parse node ", p.addr, " state: ", err)
		return
	}
	p.state = state
	str, err := json.Marshal(state)
	if err != nil {
		zap.S().Error("failed to marshal binary: ", err)
		return
	}
	zap.S().Info("loaded node ", p.addr, " state: ", string(str))
}

func (p *Peer) storeState() {
	bytes, err := json.Marshal(p.state)
	if err != nil {
		zap.S().Error("failed to marshal peer state: ", err)
		return
	}
	if err := p.db.PutRaw([]byte(p.addr), bytes); err != nil {
		zap.S().Error("failed to store peer state in db: ", err)
	}
	zap.S().Info("stored state ", p.addr, " ", string(bytes))
}

func (p *Peer) serveConn() {
	defer p.wg.Done()
	defer p.storeState()
	defer func() { p.state.State = "disconnected" }()
	if p.conn == nil {
		if err := p.connect(); err != nil {
			zap.S().Error("failed to connect to peer: ", p.addr, " ", err)
			return
		}
	}

	err := p.syncState()
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
		return
	}

	err = p.getPeers()
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
		return
	}
	err = p.updateState()
	if err != nil {
		zap.S().Error("stopped serving conn: ", err)
		return
	}
}

// Stop stops the processing of connection with peer
func (p *Peer) Stop() {
	// TODO: race
	if p.conn != nil {
		p.conn.Close()
	}
	p.cancel()
	p.wg.Wait()
}

// PeerOption is a creating option for creating Peer
type PeerOption func(*Peer) error

// WithAddr configures peer to have an addr
func WithAddr(addr string) PeerOption {
	return func(p *Peer) error {
		p.addr = addr
		return nil
	}
}

// WithConn configures peer with an existing connection
func WithConn(conn net.Conn) PeerOption {
	return func(p *Peer) error {
		p.handshake(conn)
		c, err := p2p.NewConn(p2p.WithNetConn(conn))
		if err != nil {
			return err
		}

		p.conn = c
		return nil
	}
}

// WithPeersChan configures peer with a channel to send peer infos to
func WithPeersChan(c chan proto.PeerInfo) PeerOption {
	return func(p *Peer) error {
		p.peers = c
		return nil
	}
}

func WithDeclAddr(addr string) PeerOption {
	return func(p *Peer) error {
		var declAddr proto.PeerInfo
		split := strings.Split(addr, ":")
		if len(split) != 2 {
			zap.S().Error("addr ", addr)
			return errors.New("addr in wrong format: " + addr)
		}
		declAddr.Addr = net.ParseIP(split[0])
		port, err := strconv.ParseInt(split[1], 10, 16)
		if err != nil {
			return err
		}
		declAddr.Port = uint16(port)
		p.declAddr = declAddr
		return nil
	}
}

// NewPeer creates a new peer
func NewPeer(gen crypto.Signature, db *db.WavesDB, opts ...PeerOption) (*Peer, error) {
	p := &Peer{}

	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}

	if p.conn == nil && p.addr == "" {
		return nil, errors.New("remote addr or existing connection have to be specified")
	}

	if p.ctx == nil {
		p.ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(p.ctx)
	p.ctx = ctx
	p.db = db
	p.cancel = cancel
	p.genesis = gen

	p.loadState()
	p.state.State = stateConnecting
	p.wg.Add(1)
	go p.serveConn()

	return p, nil
}
