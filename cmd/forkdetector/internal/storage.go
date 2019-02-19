package internal

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
)

const (
	peerKeyPrefix = iota
	blockKeyPrefix
	heightKeyPrefix
)

type peerKey struct {
	ip net.IP
}

type heightBlockKey struct {
	height uint32
	block  crypto.Signature
}

type Storage struct {
	db      *leveldb.DB
	log     *zap.SugaredLogger
	genesis crypto.Signature
}

func NewStorage(path string, log *zap.SugaredLogger) (*Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open storage")
	}
	return &Storage{db: db, log: log}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) PutPeer(peer Peer) error {
	return nil
}

func (s *Storage) PutPeers(peers []Peer) error {
	return nil
}

func (s *Storage) GetPeers() ([]Peer, error) {
	return nil, nil
}

func (s *Storage) PutBlock(block proto.Block) error {
	var height uint32
	switch {
	case block.BlockSignature == s.genesis:
		height = 1
	default:
		height, err := s.getHeight(block.Parent)
		if err != nil {
			return errors.Wrap(err, "failed to find parent's height")
		}
		height++
	}
	batch := new(leveldb.Batch)
	k := heightBlockKey{height: height, block: block.BlockSignature}
	v := block
	batch.Put(k.bytes(), )
	return nil
}

func (s *Storage) GetBlock(id crypto.Signature) (*proto.Block, error) {
	return nil, nil
}

func (s *Storage) GetBlocks() ([]proto.Block, error) {
	return nil, nil
}

func (s *Storage) getHeight(id crypto.Signature) (uint32, error) {
	return 0, nil
}
