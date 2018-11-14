package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	HeightKey byte = 0
)

var (
	defaultReadOptions  = &opt.ReadOptions{}
	defaultWriteOptions = &opt.WriteOptions{}
	keyHeight           = []byte{HeightKey}
)

type Storage struct {
	DBPath string
	db     *leveldb.DB
}

func (s *Storage) Open(path string) error {
	o := &opt.Options{}
	o.ReadOnly = false
	o.ErrorIfMissing = true
	db, err := leveldb.OpenFile(s.DBPath, o)
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
