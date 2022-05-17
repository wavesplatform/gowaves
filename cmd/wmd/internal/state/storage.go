package state

import (
	"bytes"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type Storage struct {
	Path   string
	Scheme byte
	db     *leveldb.DB
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

const (
	maxLimit = 1000
)

func (s *Storage) PutTrades(height int, block proto.BlockID, trades []data.Trade) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store trades of block '%s' at height %d", block.String(), height)
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	batch := new(leveldb.Batch)
	err = putTrades(bs, batch, uint32(height), trades)
	if err != nil {
		return wrapError(err)
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (s *Storage) PutBalances(height int, block proto.BlockID, issues []data.IssueChange, assets []data.AssetChange, accounts []data.AccountChange, aliases []data.AliasBind) error {
	h := uint32(height)
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to store block '%s' at height %d", block.String(), height)
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	batch := new(leveldb.Batch)
	err = putAliases(bs, batch, h, aliases)
	if err != nil {
		return wrapError(err)
	}
	err = putIssues(bs, batch, s.Scheme, h, issues)
	if err != nil {
		return wrapError(err)
	}
	err = putAssets(bs, batch, h, assets)
	if err != nil {
		return wrapError(err)
	}
	err = putAccounts(bs, batch, h, accounts)
	if err != nil {
		return wrapError(err)
	}
	err = putBlock(batch, h, block)
	if err != nil {
		return wrapError(err)
	}
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (s *Storage) SafeRollbackHeight(height int) (int, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	tf, ok := earliestTimeFrame(snapshot, uint32(height))
	if !ok {
		return height, nil
	}
	eh, err := earliestAffectedHeight(snapshot, tf)
	if err != nil {
		return 0, err
	}
	if int(eh) >= height {
		return height, nil
	} else {
		return s.SafeRollbackHeight(int(eh))
	}
}

func (s *Storage) Rollback(removeHeight int) error {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return err
	}
	defer snapshot.Release()
	max, err := height(snapshot)
	if err != nil {
		return err
	}
	if removeHeight > max {
		return errors.Errorf("nothing to rollback, current height is %d", max)
	} else {
		zap.S().Infof("Rolling back from height %d to height %d, removing %d blocks", max, removeHeight-1, max-removeHeight+1)
	}
	batch := new(leveldb.Batch)
	rh := uint32(removeHeight)

	if err := rollbackTrades(snapshot, batch, rh); err != nil {
		return err
	}
	if err := rollbackAccounts(snapshot, batch, rh); err != nil {
		return err
	}
	if err := rollbackAssets(snapshot, batch, rh); err != nil {
		return err
	}
	if err := rollbackAliases(snapshot, batch, rh); err != nil {
		return err
	}
	if err := rollbackBlocks(snapshot, batch, rh); err != nil {
		return err
	}
	if err := s.db.Write(batch, nil); err != nil {
		return err
	}
	return nil
}

func (s *Storage) Height() (int, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	return height(snapshot)
}

func (s *Storage) AssetInfo(asset crypto.Digest) (*data.AssetInfo, error) {
	if bytes.Equal(asset[:], data.WavesID[:]) {
		return &data.WavesAssetInfo, nil
	}
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	a, ok, err := bs.assetInfo(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read AssetInfo")
	}
	if !ok {
		return nil, errors.Errorf("failed to locate asset '%s'", asset.String())
	}
	return &data.AssetInfo{
		ID:            asset,
		Name:          a.name,
		IssuerAddress: a.issuer,
		Decimals:      a.decimals,
		Reissuable:    a.reissuable,
		Supply:        a.supply,
	}, nil
}

func (s *Storage) Trades(amountAsset, priceAsset crypto.Digest, limit int) ([]data.Trade, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return trades(snapshot, amountAsset, priceAsset, 0, math.MaxInt64, limit)
}

func (s *Storage) TradesRange(amountAsset, priceAsset crypto.Digest, from, to uint64) ([]data.Trade, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return trades(snapshot, amountAsset, priceAsset, from, to, maxLimit)
}

func (s *Storage) TradesByAddress(amountAsset, priceAsset crypto.Digest, address proto.WavesAddress, limit int) ([]data.Trade, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return addressTrades(snapshot, amountAsset, priceAsset, address, limit)
}

func (s *Storage) CandlesRange(amountAsset, priceAsset crypto.Digest, from, to uint32, timeFrameScale int) ([]data.Candle, error) {
	limit := timeFrameScale * maxLimit
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, err
	}
	defer snapshot.Release()
	return candles(snapshot, amountAsset, priceAsset, from, to+uint32(timeFrameScale), limit)
}

func (s *Storage) DayCandle(amountAsset, priceAsset crypto.Digest) (data.Candle, error) {
	now := uint64(time.Now().Unix() * 1000)
	ttf := data.TimeFrameFromTimestampMS(now)
	ftf := ttf - 289
	zap.S().Debugf("DayCandle: %s - %s", time.Unix(int64(data.TimestampMSFromTimeFrame(ftf)/1000), 0), time.Unix(int64(data.TimestampMSFromTimeFrame(ttf)/1000), 0))
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return data.Candle{}, err
	}
	defer snapshot.Release()
	cs, err := candles(snapshot, amountAsset, priceAsset, ftf, ttf, math.MaxInt32)
	zap.S().Debugf("Combining %d candles", len(cs))
	if err != nil {
		return data.Candle{}, err
	}
	r := data.Candle{}
	zap.S().Debugf("Empty candle: %v", r)
	for _, c := range cs {
		r.Combine(c)
		zap.S().Debugf("Appended candle: %v, with result: %v", c, r)
	}
	return r, nil
}

func (s *Storage) HasBlock(height int, block proto.BlockID) (bool, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return false, err
	}
	defer snapshot.Release()
	return hasBlock(snapshot, uint32(height), block)
}

func (s *Storage) Markets() (map[data.MarketID]data.Market, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect markets")
	}
	defer snapshot.Release()
	return marketsMap(snapshot)
}

func (s *Storage) BlockID(height int) (proto.BlockID, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return proto.BlockID{}, err
	}
	defer snapshot.Release()
	b, ok, err := block(snapshot, uint32(height))
	if err != nil {
		return proto.BlockID{}, err
	}
	if !ok {
		return proto.BlockID{}, errors.Errorf("no block at height %d", height)
	}
	return b, nil
}

func (s *Storage) IssuerBalance(issuer proto.WavesAddress, asset crypto.Digest) (uint64, error) {
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}
	defer snapshot.Release()
	bs := newBlockState(snapshot)
	b, _, err := bs.balance(issuer, asset)
	if err != nil {
		return 0, err
	}
	return b, nil
}
