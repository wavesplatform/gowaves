package db

import (
	"encoding/binary"
	"errors"

	_ "github.com/mr-tron/base58/base58"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	heightSuffix = "h"
)

/*
	switch {
	case b.BlockSignature == s.genesis:
		height = 1
	default:
		heightBytes, err := s.db.Get(heightKey)
		if err != nil {
			zap.S().Info("height not found for ", str)
			return
		}
		if len(heightBytes) != 8 {
			zap.S().Info("len is ", len(heightBytes))
			return
		}
		height = binary.BigEndian.Uint64(heightBytes)
		height++
	}
	var heightBytes [8]byte
	binary.BigEndian.PutUint64(heightBytes[:], height)
	heightKey = append(b.BlockSignature[:], []byte("h")...)
*/

var ErrBlockOrphaned = errors.New("block orphaned")

type WavesDB struct {
	genesis proto.BlockID
	db      *leveldb.DB
}

func (w *WavesDB) GetRaw(key []byte) ([]byte, error) {
	return w.db.Get(key, nil)
}

func (w *WavesDB) PutRaw(key, value []byte) error {
	return w.db.Put(key, value, nil)
}

func (w *WavesDB) Has(block proto.BlockID) (bool, error) {
	return w.db.Has(block[:], nil)
}

func (w *WavesDB) Put(block *proto.Block) error {
	var height uint64

	switch {
	case block.BlockSignature == w.genesis:
		height = 1
	default:
		parentHeight := append(block.Parent[:], []byte(heightSuffix)...)
		has, err := w.db.Has(parentHeight, nil)
		if err != nil {
			return err
		}
		if !has {
			return ErrBlockOrphaned
		}
		heightBytes, err := w.db.Get(parentHeight, nil)
		if err != nil {
			return err
		}
		height = binary.BigEndian.Uint64(heightBytes)
		height++
	}
	bytes, err := block.MarshalBinary()
	if err != nil {
		return err
	}
	if err = w.db.Put(block.BlockSignature[:], bytes, nil); err != nil {
		return err
	}

	heightKey := append(block.BlockSignature[:], []byte(heightSuffix)...)
	heightValue := make([]byte, 8)
	binary.BigEndian.PutUint64(heightValue, height)
	if err = w.db.Put(heightKey, heightValue, nil); err != nil {
		return err
	}

	return nil
}

func (w *WavesDB) Get(block proto.BlockID) (*proto.Block, error) {
	bytes, err := w.db.Get(block[:], nil)
	if err != nil {
		return nil, err
	}
	var res proto.Block
	if err = res.UnmarshalBinary(bytes); err != nil {
		return nil, err
	}

	return &res, nil
}

func NewDB(path string, genesis proto.BlockID) (*WavesDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &WavesDB{db: db, genesis: genesis}, nil
}
