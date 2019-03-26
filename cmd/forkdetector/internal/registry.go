package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PublicAddressState byte

const (
	NewPublicAddress        PublicAddressState = iota // Totally unknown public address
	DiscardedPublicAddress                            // Network connection to the address failed
	RespondingPublicAddress                           // Network connection to the address was successful
	GreetedPublicAddress                              // Handshake with the node on the address was successful
	HostilePublicAddress                              // The node has different blockchain scheme
)

type PublicAddress struct {
	address     PeerAddr
	state       PublicAddressState
	attempts    int
	nextAttempt time.Time
	version     proto.Version
}

const (
	timeBinarySize = 1 + 8 + 4 + 2
)

func (a PublicAddress) String() string {
	sb := strings.Builder{}
	sb.WriteString(a.address.String())
	sb.WriteRune(' ')
	switch a.state {
	case NewPublicAddress:
		sb.WriteString("NEW")
	case DiscardedPublicAddress:
		sb.WriteString("DISCARDED")
	case RespondingPublicAddress:
		sb.WriteString("RESPONDING")
	case GreetedPublicAddress:
		sb.WriteString("GREETED")
	case HostilePublicAddress:
		sb.WriteString("HOSTILE")
	}
	sb.WriteRune(' ')
	sb.WriteString(a.version.String())
	sb.WriteRune(' ')
	sb.WriteRune('[')
	sb.WriteString(strconv.Itoa(a.attempts))
	sb.WriteRune('|')
	sb.WriteString(a.nextAttempt.Format(time.RFC3339))
	sb.WriteRune(']')
	return sb.String()
}

func (a PublicAddress) MarshalBinary() ([]byte, error) {
	buf := make([]byte, PeerAddrLen+1+4+timeBinarySize+3*4)
	ab, err := a.address.MarshalBinary()
	copy(buf, ab)
	buf[PeerAddrLen] = byte(a.state)
	binary.BigEndian.PutUint32(buf[PeerAddrLen+1:], uint32(a.attempts))
	tb, err := a.nextAttempt.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal PublicAddress to bytes")
	}
	copy(buf[PeerAddrLen+1+4:], tb)
	binary.BigEndian.PutUint32(buf[PeerAddrLen+1+4+timeBinarySize:], a.version.Major)
	binary.BigEndian.PutUint32(buf[PeerAddrLen+1+4+timeBinarySize+4:], a.version.Minor)
	binary.BigEndian.PutUint32(buf[PeerAddrLen+1+4+timeBinarySize+4+4:], a.version.Patch)
	return buf, nil
}

func (a *PublicAddress) UnmarshalBinary(data []byte) error {
	if l := len(data); l < PeerAddrLen+1+4+timeBinarySize+3*4 {
		return errors.Errorf("%d is not enough bytes for PublicAddress", l)
	}
	var addr PeerAddr
	err := addr.UnmarshalBinary(data[:PeerAddrLen])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal PublicAddress")
	}
	a.address = addr
	a.state = PublicAddressState(data[PeerAddrLen])
	a.attempts = int(binary.BigEndian.Uint32(data[PeerAddrLen+1:]))
	t := time.Time{}
	err = t.UnmarshalBinary(data[PeerAddrLen+1+4 : PeerAddrLen+1+4+timeBinarySize])
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal announcement")
	}
	a.nextAttempt = t
	var v proto.Version
	v.Major = binary.BigEndian.Uint32(data[PeerAddrLen+1+4+timeBinarySize:])
	v.Minor = binary.BigEndian.Uint32(data[PeerAddrLen+1+4+timeBinarySize+4:])
	v.Patch = binary.BigEndian.Uint32(data[PeerAddrLen+1+4+timeBinarySize+4+4:])
	a.version = v
	return nil
}

type PublicAddressRegistry struct {
	coolDownDuration time.Duration
	banDuration      time.Duration
	versions         []proto.Version
	storage          *storage
	operating        map[uint64]struct{}
	mu               sync.Mutex
}

func NewPublicAddressRegistry(storage *storage, short, long time.Duration, versions []proto.Version) *PublicAddressRegistry {
	sorted := proto.ByVersion(versions)
	sort.Sort(sort.Reverse(sorted))
	for i := 0; i < len(versions); i++ {
		sorted[i].Patch = 0
	}
	return &PublicAddressRegistry{
		coolDownDuration: short,
		banDuration:      long,
		versions:         sorted,
		storage:          storage,
		operating:        make(map[uint64]struct{}),
		mu:               sync.Mutex{},
	}
}

func (r *PublicAddressRegistry) RegisterNewAddresses(addresses []net.TCPAddr) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	added := 0
	for _, a := range addresses {
		ok, err := r.storage.hasPublicAddress(PeerAddr(a))
		if err != nil {
			return added, errors.Wrap(err, "failed to register new public addresses")
		}
		if !ok {
			pa := PublicAddress{
				address:     PeerAddr(a),
				state:       NewPublicAddress,
				attempts:    0,
				nextAttempt: time.Time{},
				version:     r.bestVersion(),
			}
			err := r.storage.putPublicAddress(pa)
			if err != nil {
				return added, errors.Wrap(err, "failed to register new public address")
			}
			added++
		}
	}
	return added, nil
}

func (r *PublicAddressRegistry) RegisterNewAddress(a PeerAddr, v proto.Version) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ok, err := r.storage.hasPublicAddress(a)
	if err != nil {
		return false, errors.Wrap(err, "failed to register new public addresses")
	}
	if !ok {
		pa := PublicAddress{
			address:     a,
			state:       NewPublicAddress,
			attempts:    0,
			nextAttempt: time.Time{},
			version:     v,
		}
		err := r.storage.putPublicAddress(pa)
		if err != nil {
			return false, errors.Wrap(err, "failed to register new public address")
		}
		return true, nil
	}
	return false, nil
}

func (r *PublicAddressRegistry) FeasibleAddresses() ([]PublicAddress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pas, err := r.storage.publicAddresses()
	if err != nil {
		return nil, errors.Wrap(err, "failed to public addresses")
	}
	filtered := pas[:0]
	for _, pa := range pas {
		if _, ok := r.operating[pa.address.Hash()]; ok {
			continue
		}
		if pa.state == HostilePublicAddress {
			continue
		}
		if pa.state == DiscardedPublicAddress && time.Now().Before(pa.nextAttempt) {
			continue
		}
		if pa.state == RespondingPublicAddress && time.Now().Before(pa.nextAttempt) {
			continue
		}
		filtered = append(filtered, pa)
		r.operating[pa.address.Hash()] = struct{}{}
	}
	return filtered, nil
}

func (r *PublicAddressRegistry) Discard(pa *PublicAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.operating, pa.address.Hash())
	pa.state = DiscardedPublicAddress
	pa.attempts = pa.attempts + 1
	pa.nextAttempt = time.Now().Add(r.banDuration)
	err := r.storage.putPublicAddress(*pa)
	if err != nil {
		return errors.Wrap(err, "failed to store discarded public address")
	}
	return nil
}

func (r *PublicAddressRegistry) Hostile(pa *PublicAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.operating, pa.address.Hash())
	pa.state = HostilePublicAddress
	pa.nextAttempt = time.Time{}
	err := r.storage.putPublicAddress(*pa)
	if err != nil {
		return errors.Wrap(err, "failed to store hostile public address")
	}
	return nil
}

func (r *PublicAddressRegistry) Connected(pa *PublicAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.operating, pa.address.Hash())
	pa.state = RespondingPublicAddress
	pa.version = r.nextVersion(pa.version)
	pa.nextAttempt = time.Now().Add(r.coolDownDuration)
	if pa.version == r.bestVersion() {
		pa.attempts = pa.attempts + 1
	}
	if pa.attempts > 0 {
		pa.nextAttempt = time.Now().Add(r.banDuration)
	}
	err := r.storage.putPublicAddress(*pa)
	if err != nil {
		return errors.Wrap(err, "failed to store discarded public address")
	}
	return nil
}

func (r *PublicAddressRegistry) Greeted(pa *PublicAddress, v proto.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	pa.state = GreetedPublicAddress
	pa.version = v
	pa.attempts = 0
	pa.nextAttempt = time.Time{}
	err := r.storage.putPublicAddress(*pa)
	if err != nil {
		return errors.Wrap(err, "failed to store discarded public address")
	}
	return nil
}

func (r *PublicAddressRegistry) bestVersion() proto.Version {
	return r.versions[0]
}

func (r *PublicAddressRegistry) nextVersion(v proto.Version) proto.Version {
	i := 0
	for ; i < len(r.versions); i++ {
		x := r.versions[i]
		if v.Major == x.Major && v.Minor == x.Minor {
			break
		}
	}
	if i == len(r.versions)-1 {
		return r.versions[0]
	}
	return r.versions[i+1]
}

type peer struct {
	description PeerDescription
	handler     *handler
}

type PeerRegistry struct {
	peers     map[uint64]peer
	mu        sync.Mutex
}

func NewPeerRegistry(self *PeerDesignation) *PeerRegistry {
	pm := make(map[uint64]peer)
	if self != nil {
		zap.S().Debug("Self peer is not nil")
		pm[self.Hash()] = peer{description: PeerDescription{Name: "Self"}}
	}
	return &PeerRegistry{peers: pm, mu: sync.Mutex{}}
}

func (r *PeerRegistry) HasPeer(id PeerDesignation) bool {
	_, ok := r.peers[id.Hash()]
	return  ok
}

func (r *PeerRegistry) Peers() []PeerDescription {
	r.mu.Lock()
	defer r.mu.Unlock()

	ps := make([]PeerDescription, 0, len(r.peers))
	for _, v := range r.peers {
		ps = append(ps, v.description)
	}
	return ps
}

func (r *PeerRegistry) Register(id PeerDesignation, desc PeerDescription, h *handler) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.peers[id.Hash()]; !ok {
		r.peers[id.Hash()] = peer{description: desc, handler: h}
		return true
	}
	return false
}

func (r *PeerRegistry) Unregister(pd PeerDesignation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.peers[pd.Hash()]; ok {
		delete(r.peers, pd.Hash())
	}
}
