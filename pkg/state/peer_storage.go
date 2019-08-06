package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type peerStorage struct {
	db keyvalue.IterableKeyVal
}

func newPeerStorage(db keyvalue.IterableKeyVal) *peerStorage {
	return &peerStorage{
		db: db,
	}
}

func (a *peerStorage) savePeers(peers []proto.TCPAddr) error {
	if len(peers) == 0 {
		return nil
	}

	batch, err := a.db.NewBatch()
	if err != nil {
		return StateError{errorType: ModificationError, originalError: err}
	}

	for _, p := range peers {
		k := intoBytes(p)
		batch.Put(k[:], nil)
	}

	err = a.db.Flush(batch)
	if err != nil {
		return err
	}
	return nil
}

func (a *peerStorage) peers() ([]proto.TCPAddr, error) {
	iter, err := a.db.NewKeyIterator([]byte{knownPeersPrefix})
	if err != nil {
		return nil, err
	}
	defer iter.Release()

	var peers []proto.TCPAddr
	for iter.Next() {
		p, err := fromBytes(iter.Key())
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
