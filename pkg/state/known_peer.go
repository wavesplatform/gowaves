package state

import (
	"encoding/binary"
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const KnownPeerKeyLength = 1 + 16 + 2

type KnownPeerKey [KnownPeerKeyLength]byte

type KnownPeer proto.NodeAddr

func (a *KnownPeer) key() KnownPeerKey {
	key := KnownPeerKey{}
	key[0] = knownPeersPrefix
	copy(key[1:17], a.IP[:])
	binary.BigEndian.PutUint16(key[17:], a.Port)
	return key
}

func (a *KnownPeer) FromKey(k KnownPeerKey) error {
	copy(a.IP[:], k[1:17])
	a.Port = binary.BigEndian.Uint16(k[17:])
	return nil
}

func (a *KnownPeer) UnmarshalBinary(b []byte) error {
	if len(b) < KnownPeerKeyLength {
		return errors.Errorf("too low bytes to unmarshal KnownPeer, expected at least %d, got %d", knownPeersPrefix, len(b))
	}

	k := KnownPeerKey{}
	copy(k[:], b)
	return a.FromKey(k)
}
