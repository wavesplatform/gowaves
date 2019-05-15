package internal

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"hash/fnv"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	hourDelay = time.Hour
	dayDelay  = 24 * time.Hour
)

var (
	emptyVersion = proto.Version{}
)

type Registry struct {
	scheme      byte
	self        net.IP
	versions    versions
	storage     *storage
	mu          sync.Mutex
	connections map[uint64]PeerNode
	pending     map[uint64]struct{}
}

func NewRegistry(scheme byte, self net.Addr, versions []proto.Version, storage *storage) *Registry {
	ip, _, err := splitAddr(self)
	if err != nil {
		ip = net.IPv4zero.To16()
	}
	return &Registry{
		scheme:      scheme,
		self:        ip,
		versions:    newVersions(versions),
		storage:     storage,
		connections: make(map[uint64]PeerNode, 0),
		pending:     make(map[uint64]struct{}, 0),
	}
}

// Check verifies that given address with the parameters could be accepted or connected.
func (r *Registry) Check(addr net.Addr, application string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, _, err := splitAddr(addr)
	if err != nil {
		return err
	}
	// Check that blockchain scheme is acceptable
	if s := application[len(application)-1]; s != r.scheme {
		return errors.Errorf("incompatible blockchain scheme %d", s)
	}
	// Check that this is not a connection to itself
	if bytes.Equal(ip, r.self) {
		return errors.New("connection to itself")
	}
	if ip.IsLoopback() {
		return errors.New("connection to itself")
	}
	// Check that there is no second connection from the same address
	_, ok := r.connections[hash(ip)]
	if ok {
		return errors.Errorf("duplicate connection from %s", addr)
	}
	// Check what we already know about the address
	peer, err := r.storage.Peer(ip)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil
		}
		return err
	}
	if peer.State == NodeHostile {
		return errors.Errorf("peer %s registered as hostile", addr)
	}
	return nil
}

func (r *Registry) SuggestVersion(addr net.Addr) (proto.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, _, err := splitAddr(addr)
	if err != nil {
		return proto.Version{}, err
	}
	peer, err := r.storage.Peer(ip)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return proto.Version{}, err
		}
	}
	switch peer.State {
	case NodeGreeted:
		return peer.Version, nil
	default:
		if peer.Version == emptyVersion {
			peer.Version = r.versions.bestVersion()
		} else {
			peer.Version = r.versions.nextVersion(peer.Version)
		}
		peer.Attempts++
		if peer.Attempts > len(r.versions) {
			peer.NextAttempt = time.Now().Add(dayDelay)
		} else {
			peer.NextAttempt = time.Now().Add(hourDelay)
		}
		err = r.storage.PutPeer(ip, peer)
		if err != nil {
			return proto.Version{}, err
		}
		return peer.Version, nil
	}
}

func (r *Registry) PeerConnected(addr net.Addr) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port, err := splitAddr(addr)
	if err != nil {
		return err
	}
	peer, err := r.storage.Peer(ip)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
		peer.Address = ip
		peer.Port = port
		peer.Nonce = 0
		peer.Name = "N/A"
		peer.Version = proto.Version{}
		peer.Attempts = 0
		peer.NextAttempt = time.Time{}
	}
	switch peer.State {
	case NodeUnknown, NodeDiscarded, NodeResponding:
		peer.State = NodeResponding
	case NodeHostile:
		return errors.Errorf("Peer %s already registered as hostile", ip.String())
	case NodeGreeted:
		return nil
	}
	err = r.storage.PutPeer(ip, peer)
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) PeerGreeted(addr net.Addr, nonce uint64, name string, v proto.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port, err := splitAddr(addr)
	if err != nil {
		return err
	}
	peer, err := r.storage.Peer(ip)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	}

	peer.Address = ip
	peer.Port = port
	peer.Nonce = nonce
	peer.Name = name
	peer.Version = v
	peer.Attempts = 0
	peer.NextAttempt = time.Time{}
	peer.State = NodeGreeted
	err = r.storage.PutPeer(ip, peer)
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) PeerHostile(addr net.Addr, nonce uint64, name string, v proto.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port, err := splitAddr(addr)
	if err != nil {
		return err
	}
	peer, err := r.storage.Peer(ip)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	}

	peer.Address = ip
	peer.Port = port
	peer.Nonce = nonce
	peer.Name = name
	peer.Version = v
	peer.State = NodeHostile
	err = r.storage.PutPeer(ip, peer)
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) PeerDiscarded(addr net.Addr) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, _, err := splitAddr(addr)
	if err != nil {
		return err
	}
	peer, err := r.storage.Peer(ip)
	if err != nil && err != leveldb.ErrNotFound {
		return err
	}
	peer.State = NodeDiscarded
	if peer.Attempts > len(r.versions) {
		peer.NextAttempt = time.Now().Add(dayDelay)
	} else {
		peer.NextAttempt = time.Now().Add(hourDelay)
	}
	err = r.storage.PutPeer(ip, peer)
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) Activate(addr net.Addr, h proto.Handshake) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, port, err := splitAddr(addr)
	if err != nil {
		return errors.Wrap(err, "failed to activate address")
	}
	_, ok := r.connections[hash(ip)]
	if ok {
		return errors.Errorf("attempt to activate already active address %s", addr.String())
	}
	p := PeerNode{
		Address: ip,
		Port:    port,
		Nonce:   h.NodeNonce,
		Name:    h.NodeName,
		Version: h.Version,
		State:   NodeGreeted,
	}
	r.connections[hash(ip)] = p
	return nil
}

func (r *Registry) Deactivate(addr net.Addr) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ip, _, err := splitAddr(addr)
	if err != nil {
		return errors.Wrap(err, "failed to deactivate an address")
	}
	_, ok := r.connections[hash(ip)]
	if !ok {
		return errors.Errorf("no active address %s", addr.String())
	}
	delete(r.connections, hash(ip))
	return nil
}

func (r *Registry) Connections() ([]PeerNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	connections := make([]PeerNode, len(r.connections))
	i := 0
	for _, p := range r.connections {
		sp, err := r.storage.Peer(p.Address)
		if err == nil {
			connections[i] = sp
		} else {
			connections[i] = p
		}
		i++
	}
	sort.Sort(PeerNodesByName(connections))
	return connections, nil
}

func (r *Registry) AppendAddresses(addresses []net.TCPAddr) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, addr := range addresses {
		ip, port, err := splitAddr(&addr)
		if err != nil {
			zap.S().Warnf("Error adding address: %v", err)
			continue
		}
		yes, err := r.storage.HasPeer(ip)
		if err != nil {
			zap.S().Warnf("Failed to append addresses: %v", err)
			return count
		}
		if !yes {
			peer := PeerNode{Address: ip, Port: port, State: NodeUnknown}
			err := r.storage.PutPeer(ip, peer)
			if err != nil {
				zap.S().Warnf("Failed to append addresses: %v", err)
				return count
			}
			count++
		}
	}
	return count
}

func (r *Registry) Peers() ([]PeerNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers, err := r.storage.Peers()
	if err != nil {
		return nil, err
	}
	return peers, nil
}

func (r *Registry) Addresses() ([]net.Addr, error) {
	addresses := make([]net.Addr, 0)
	peers, err := r.storage.Peers()
	if err != nil {
		return addresses, errors.Wrap(err, "failed to get public addresses from storage")
	}
	for _, peer := range peers {
		if peer.State != NodeGreeted {
			continue
		}
		addr := &net.TCPAddr{IP: peer.Address, Port: int(peer.Port)}
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (r Registry) TakeAvailableAddresses() ([]net.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	addresses := make([]net.Addr, 0)
	peers, err := r.storage.Peers()
	if err != nil {
		return addresses, errors.Wrap(err, "failed to get available addresses from storage")
	}
	for _, peer := range peers {
		if peer.State == NodeHostile {
			continue
		}
		if peer.NextAttempt.After(time.Now()) {
			continue
		}
		_, ok := r.connections[hash(peer.Address)]
		if ok {
			continue
		}
		_, ok = r.pending[hash(peer.Address)]
		if ok {
			continue
		}
		addr := &net.TCPAddr{IP: peer.Address, Port: int(peer.Port)}
		addresses = append(addresses, addr)
		r.pending[hash(peer.Address)] = struct{}{}
	}
	return addresses, nil
}

type versions []proto.Version

func newVersions(vs []proto.Version) versions {
	sorted := proto.ByVersion(vs)
	sort.Sort(sort.Reverse(sorted))
	for i := 0; i < len(vs); i++ {
		sorted[i].Patch = 0
	}
	return versions(sorted)
}

func (vs versions) bestVersion() proto.Version {
	return vs[0]
}

func (vs versions) nextVersion(v proto.Version) proto.Version {
	i := 0
	for ; i < len(vs); i++ {
		x := vs[i]
		if v.Major == x.Major && v.Minor == x.Minor {
			break
		}
	}
	if i == len(vs)-1 {
		return vs[0]
	}
	return vs[i+1]
}

func splitAddr(addr net.Addr) (net.IP, uint16, error) {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return net.IP{}, 0, errors.Errorf("not a TCP address '%s'", addr.String())
	}
	return tcpAddr.IP.To16(), uint16(tcpAddr.Port), nil
}

func hash(ip net.IP) uint64 {
	h := fnv.New64()
	h.Reset()
	_, err := h.Write(ip)
	if err != nil {
		panic("err should be always nil")
	}
	return h.Sum64()
}
