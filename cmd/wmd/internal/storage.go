package internal

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	ldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
	ldbUtil "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"math"
)

const (
	LastHeightKeyPrefix byte = iota
	BlockAtHeightKeyPrefix
	TradesKeyPrefix
	CandlesKeyPrefix
	BlockTradesKeyPrefix
	EarliestHeightKeyPrefix
)

var (
	defaultReadOptions  = &opt.ReadOptions{}
	defaultWriteOptions = &opt.WriteOptions{}
	lastHeightKey       = []byte{LastHeightKeyPrefix}
)

type blockInfo struct {
	Empty             bool
	ID                crypto.Signature
	EarliestTimeFrame uint64
}

func (b *blockInfo) marshalBinary() []byte {
	buf := make([]byte, crypto.SignatureSize+1+8)
	if b.Empty {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	copy(buf[1:], b.ID[:])
	binary.BigEndian.PutUint64(buf[1+crypto.SignatureSize:], b.EarliestTimeFrame)
	return buf
}

func (b *blockInfo) unmarshalBinary(data []byte) error {
	if len(data) != crypto.SignatureSize+1+8 {
		return errors.New("incorrect data for blockInfo")
	}
	b.Empty = data[0] == 1
	copy(b.ID[:], data[1:])
	b.EarliestTimeFrame = binary.BigEndian.Uint64(data[1+crypto.SignatureSize:])
	return nil
}

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

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) GetLastHeight() (int, error) {
	v, err := s.db.Get(lastHeightKey, defaultReadOptions)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return 0, errors.Wrap(err, "failed to get Height")
		}
		return 0, nil
	}
	if l := len(v); l != 4 {
		return 0, errors.Errorf("%d is incorrect length for Height value, expected 4", l)
	}
	return int(binary.BigEndian.Uint32(v)), nil
}

func (s *Storage) ShouldImportBlock(height int, block crypto.Signature) (bool, error) {
	v, err := s.db.Get(uint32Key(BlockAtHeightKeyPrefix, uint32(height)), defaultReadOptions)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return false, errors.Wrapf(err, "failed to check existence of block '%s'", block.String())
		}
		return true, nil
	}
	var bi blockInfo
	err = bi.unmarshalBinary(v)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check existence of block '%s'", block.String())
	}
	if bytes.Equal(bi.ID[:], block[:]) {
		return false, nil
	}
	rollbackHeight, err := s.FindCorrectRollbackHeight(height - 1)
	if err != nil {
		return false, errors.Wrapf(err, "failed to find correct Rollback height while applying block '%s'", block.String())
	}
	err = s.Rollback(rollbackHeight)
	if err != nil {
		return false, errors.Wrapf(err, "failed to Rollback to height %d", height-1)
	}
	return true, nil
}

func (s *Storage) PutBlock(height int, block crypto.Signature, trades []Trade) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}

	candles := map[CandleKey]Candle{}

	affectedTimeFrames := make([]uint64, 0)
	for _, t := range trades {
		tf := timeFrame(t.Timestamp)
		if !contains(affectedTimeFrames, tf) {
			affectedTimeFrames = append(affectedTimeFrames, tf)
		}
		ck := CandleKey{t.AmountAsset, t.PriceAsset, tf}
		var c Candle
		c, ok := candles[ck]
		if !ok {
			ckb, err := ck.MarshalBinary()
			if err != nil {
				return wrapError(err)
			}
			cb, err := s.db.Get(ckb, defaultReadOptions)
			if err != nil {
				if err != ldbErrors.ErrNotFound {
					return wrapError(err)
				}
				c = NewCandle(t.Timestamp)
			} else {
				err = c.UnmarshalBinary(cb)
				if err != nil {
					return wrapError(err)
				}
			}
		}
		c.UpdateFromTrade(t)
		candles[ck] = c
	}

	batch := new(leveldb.Batch)
	n := len(trades)
	updateLastHeight(batch, height)
	mtf := min(affectedTimeFrames)
	putBlockInfo(batch, height, block, n == 0, mtf)
	putTradesIDs(batch, block, trades)
	err := s.updateAffectedTimeFrames(batch, affectedTimeFrames, height)
	if err != nil {
		return wrapError(err)
	}
	for _, t := range trades {
		k1 := digestKey(TradesKeyPrefix, t.TransactionID)
		v1 := make([]byte, TradeSize)
		v1, err := t.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(k1, v1)
	}
	for ck, cv := range candles {
		k, err := ck.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		v, err := cv.MarshalBinary()
		if err != nil {
			return wrapError(err)
		}
		batch.Put(k, v)
	}
	err = s.db.Write(batch, defaultWriteOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}
	return nil
}

func (s *Storage) FindCorrectRollbackHeight(height int) (int, error) {
	ok, bi, err := s.findFirstNonEmptyBlock(height)
	if err != nil {
		return 0, errors.Wrap(err, "failed to find first non empty block")
	}
	if !ok {
		return height, nil
	}
	etf := bi.EarliestTimeFrame
	k := uint64Key(EarliestHeightKeyPrefix, etf)
	b, err := s.db.Get(k, defaultReadOptions)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get earliest height for time frame %d", etf)
	}
	eh := int(binary.BigEndian.Uint32(b))
	if eh >= height {
		return height, nil
	} else {
		return s.FindCorrectRollbackHeight(eh)
	}
}

func (s *Storage) Rollback(height int) error {
	max, err := s.GetLastHeight()
	if err != nil {
		return errors.Wrap(err, "failed to Rollback")
	}
	if height >= max {
		return errors.Errorf("Rollback is impossible, current height %d is not lower then given %d", max, height)
	}
	for h := max; h > height; h-- {
		bi, err := s.blockInfo(h)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
		trades, err := s.getTrades(bi.ID)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
		batch := new(leveldb.Batch)
		err = s.removeCandlesAfter(batch, youngestTimeFrame(trades))
		if err != nil {
			batch.Reset()
			return errors.Wrapf(err, "rollback failure at height %d", h)
		}
		removeTrades(batch, trades)
		err = s.removeEarliestTimeFrames(batch, bi.EarliestTimeFrame)
		if err != nil {
			batch.Reset()
			return errors.Wrapf(err, "rollback failure at height %d", h)
		}
		batch.Delete(signatureKey(BlockTradesKeyPrefix, bi.ID))
		hk := uint32Key(BlockAtHeightKeyPrefix, uint32(h))
		batch.Delete(hk)
		updateLastHeight(batch, h-1)
		err = s.db.Write(batch, defaultWriteOptions)
		if err != nil {
			return errors.Wrapf(err, "Rollback failure at height %d", h)
		}
	}
	return nil
}

func (s *Storage) findFirstNonEmptyBlock(height int) (bool, blockInfo, error) {
	bi, err := s.blockInfo(height)
	if err != nil {
		if err != ldbErrors.ErrNotFound {
			return false, blockInfo{}, err
		}
		return false, blockInfo{}, nil
	}
	if !bi.Empty {
		return true, bi, nil
	}
	return s.findFirstNonEmptyBlock(height - 1)
}

func (s *Storage) blockInfo(height int) (blockInfo, error) {
	k := uint32Key(BlockAtHeightKeyPrefix, uint32(height))
	v, err := s.db.Get(k, defaultReadOptions)
	if err != nil {
		return blockInfo{}, errors.Wrapf(err, "failed to get info about block at height %d", height)
	}
	var bi blockInfo
	err = bi.unmarshalBinary(v)
	if err != nil {
		return blockInfo{}, errors.Wrapf(err, "Rollback failure at height %d", height)
	}
	return bi, nil
}

func (s *Storage) getTrades(block crypto.Signature) ([]Trade, error) {
	k := signatureKey(BlockTradesKeyPrefix, block)
	v, err := s.db.Get(k, defaultReadOptions)
	if err != nil && err != ldbErrors.ErrNotFound {
		return []Trade{}, errors.Wrap(err, "failed to collect Trades")
	}
	n := len(v) / crypto.DigestSize
	r := make([]Trade, n)
	tid := make([]byte, crypto.DigestSize)
	for i := 0; i < n; i++ {
		copy(tid, v[i*crypto.DigestSize:])
		tk := prepend(TradesKeyPrefix, tid)
		tv, err := s.db.Get(tk, defaultReadOptions)
		if err != nil {
			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
		}
		err = r[i].UnmarshalBinary(tv)
		if err != nil {
			return []Trade{}, errors.Wrap(err, "failed to collect Trades")
		}
	}
	return r, nil
}

func (s *Storage) removeCandlesAfter(batch *leveldb.Batch, tf uint64) error {
	start := uint64Key(CandlesKeyPrefix, tf)
	end := []byte{CandlesKeyPrefix + 1}
	it := s.db.NewIterator(&ldbUtil.Range{Start: start, Limit: end}, defaultReadOptions)
	for it.Next() {
		batch.Delete(it.Key())
	}
	it.Release()
	return it.Error()
}

func (s *Storage) removeEarliestTimeFrames(batch *leveldb.Batch, etf uint64) error {
	start := uint64Key(EarliestHeightKeyPrefix, etf)
	end := uint64Key(EarliestHeightKeyPrefix, math.MaxUint64)
	it := s.db.NewIterator(&ldbUtil.Range{Start: start, Limit: end}, defaultReadOptions)
	for it.Next() {
		batch.Delete(it.Key())
	}
	it.Release()
	return it.Error()
}

func (s *Storage) updateAffectedTimeFrames(batch *leveldb.Batch, timeFrames []uint64, height int) error {
	for _, tf := range timeFrames {
		k := uint64Key(EarliestHeightKeyPrefix, tf)
		b, err := s.db.Get(k, defaultReadOptions)
		h := math.MaxInt32
		if err != nil && err != ldbErrors.ErrNotFound {
			return err
		}
		if len(b) > 0 {
			h = int(binary.BigEndian.Uint32(b))
		}
		if height < h {
			v := make([]byte, 4)
			binary.BigEndian.PutUint32(v, uint32(height))
			batch.Put(k, v)
		}
	}
	return nil
}

func putBlockInfo(batch *leveldb.Batch, height int, block crypto.Signature, empty bool, earliestTimeFrame uint64) {
	k := uint32Key(BlockAtHeightKeyPrefix, uint32(height))
	bi := blockInfo{empty, block, earliestTimeFrame}
	v := bi.marshalBinary()
	batch.Put(k, v)
}

func putTradesIDs(batch *leveldb.Batch, block crypto.Signature, trades []Trade) {
	n := len(trades)
	k := signatureKey(BlockTradesKeyPrefix, block)
	v := make([]byte, crypto.DigestSize*n)
	for i, t := range trades {
		copy(v[i*crypto.DigestSize:], t.TransactionID[:])
	}
	batch.Put(k, v)
}

func youngestTimeFrame(trades []Trade) uint64 {
	ts := uint64(math.MaxUint64)
	for _, t := range trades {
		if t.Timestamp < ts {
			ts = t.Timestamp
		}
	}
	return timeFrame(ts)
}

func uint32Key(prefix byte, key uint32) []byte {
	k := make([]byte, 5)
	k[0] = prefix
	binary.BigEndian.PutUint32(k[1:], key)
	return k
}

func uint64Key(prefix byte, key uint64) []byte {
	k := make([]byte, 9)
	k[0] = prefix
	binary.BigEndian.PutUint64(k[1:], key)
	return k
}

func digestKey(prefix byte, key crypto.Digest) []byte {
	k := make([]byte, 1+crypto.DigestSize)
	k[0] = prefix
	copy(k[1:], key[:])
	return k
}

func signatureKey(prefix byte, sig crypto.Signature) []byte {
	k := make([]byte, 1+crypto.SignatureSize)
	k[0] = prefix
	copy(k[1:], sig[:])
	return k
}

func prepend(prefix byte, key []byte) []byte {
	k := make([]byte, 1+len(key))
	k[0] = prefix
	copy(k[1:], key)
	return k
}

func removeTrades(batch *leveldb.Batch, trades []Trade) {
	for _, t := range trades {
		batch.Delete(digestKey(TradesKeyPrefix, t.TransactionID))
	}
}

func updateLastHeight(batch *leveldb.Batch, height int) {
	v := make([]byte, 4)
	binary.BigEndian.PutUint32(v, uint32(height))
	batch.Put(lastHeightKey, v)
}

func contains(a []uint64, v uint64) bool {
	for _, x := range a {
		if v == x {
			return true
		}
	}
	return false
}

func min(a []uint64) uint64 {
	r := uint64(math.MaxUint64)
	for _, x := range a {
		if x < r {
			r = x
		}
	}
	return r
}
