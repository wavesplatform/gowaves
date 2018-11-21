package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	HeightKey  byte = 0
	BlocksKey  byte = 1
	TradesKey  byte = 2
	CandlesKey byte = 3
)

var (
	defaultReadOptions  = &opt.ReadOptions{}
	defaultWriteOptions = &opt.WriteOptions{}
	keyHeight           = []byte{HeightKey}
)

type Storage struct {
	Path string
	db   *leveldb.DB
}

func (s *Storage) Open() error {
	o := &opt.Options{}
	db, err := leveldb.OpenFile(s.Path, o)
	if err != nil {
		return errors.Wrap(err, "failed to open Storage")
	}
	s.db = db
	return nil
}

func (s *Storage) Close() {
	s.db.Close()
}

func (s *Storage) GetHeight() (int, error) {
	v, err := s.db.Get(keyHeight, defaultReadOptions)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get Height")
	}
	if l := len(v); l != 4 {
		return 0, errors.Errorf("%d is incorrect length for Height value, expected 4", l)
	}
	return int(binary.BigEndian.Uint32(v)), nil
}

func (s *Storage) PutHeight(h int) error {
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(h))
	err := s.db.Put(keyHeight, v, defaultWriteOptions)
	if err != nil {
		errors.Wrapf(err, "failed to store Height value %d", h)
	}
	return nil
}

func (s *Storage) PutBlock(blockID crypto.Signature, height int) error {
	key := make([]byte, 1+crypto.SignatureSize)
	key[0] = BlocksKey
	copy(key[1:], blockID[:])
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(height))
	err := s.db.Put(key, v, defaultWriteOptions)
	if err != nil {
		errors.Wrapf(err, "failed to store BlockID %s", blockID.String())
	}
	return nil
}

func (s *Storage) PutTrades(height int, trades []Trade) error {
	for _, t := range trades {
		key := make([]byte, 1+crypto.DigestSize)
		key[0] = TradesKey
		copy(key[1:], t.TransactionID[:])
		v := make([]byte, TradeSize)
		v, err := t.MarshalBinary()
		if err != nil {
			return errors.Wrap(err, "failed to store Trade")
		}
		err = s.db.Put(key, v, defaultWriteOptions)
		if err != nil {
			return errors.Wrap(err, "failed to store Trade")
		}
		err = s.updateCandle(t)
		if err != nil {
			return errors.Wrap(err, "failed to update Candle")
		}
	}
	return nil
}

func (s *Storage) updateCandle(t Trade) error {
	k := make([]byte, 1+8)
	k[0] = CandlesKey
	b := StartOfTheFrame(t.Timestamp)
	binary.BigEndian.PutUint64(k[1:], b)
	v, err := s.db.Get(k, defaultReadOptions)
	var c Candle
	if err != nil {
		if err != leveldbErrors.ErrNotFound {
			return errors.Wrap(err, "failed to update Candle")
		}
		c = NewCandle(t.Timestamp)
	} else {
		err = c.UnmarshalBinary(v)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal Candle")
		}
	}
	c.UpdateFromTrade(t)
	v, err = c.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to marshal Candle")
	}
	err = s.db.Put(k, v, defaultWriteOptions)
	if err != nil {
		return errors.Wrap(err, "failed to update Candle")
	}
	return nil
}
