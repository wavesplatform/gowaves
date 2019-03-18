package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

type peerStorage struct {
	db keyvalue.IterableKeyVal
}

func newPeerStorage(db keyvalue.IterableKeyVal) *peerStorage {
	return &peerStorage{
		db: db,
	}
}

func (a *peerStorage) savePeers(peers []KnownPeer) error {
	for _, p := range peers {
		k := p.key()
		err := a.db.PutDirectly(k[:], nil)
		if err != nil {
			return StateError{errorType: ModificationError, originalError: err}
		}
	}
	return nil
}

func (a *peerStorage) peers() ([]KnownPeer, error) {
	iter, err := a.db.NewKeyIterator([]byte{knownPeersPrefix})
	if err != nil {
		return nil, err
	}
	defer iter.Release()

	var peers []KnownPeer
	for iter.Next() {
		p := KnownPeer{}
		err = p.UnmarshalBinary(iter.Key())
		if err != nil {
			return nil, err
		}

		peers = append(peers, p)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return peers, nil
}
