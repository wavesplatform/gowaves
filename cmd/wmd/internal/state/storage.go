package state

import (
	"bytes"
	"log/slog"
	"math"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	mx, err := height(snapshot)
	if err != nil {
		return err
	}
	if removeHeight > mx {
		return errors.Errorf("nothing to rollback, current height is %d", mx)
	} else {
		slog.Info("Rolling back and removing blocks",
			"fromHeight", mx, "toHeight", removeHeight-1, "blocksCount", mx-removeHeight+1)
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
	const millis = 1000
	now, err := safecast.ToUint64(time.Now().Unix() * millis)
	if err != nil {
		return data.Candle{}, errors.Wrap(err, "failed to convert current time to int64")
	}
	ttf := data.TimeFrameFromTimestampMS(now)
	const candleWidth = 289
	ftf := ttf - candleWidth
	sts, err := safecast.ToInt64(data.TimestampMSFromTimeFrame(ftf) / millis)
	if err != nil {
		return data.Candle{}, errors.Wrap(err, "failed to convert timestamp to int64")
	}
	ets, err := safecast.ToInt64(data.TimestampMSFromTimeFrame(ttf) / millis)
	if err != nil {
		return data.Candle{}, errors.Wrap(err, "failed to convert timestamp to int64")
	}
	slog.Debug("DayCandle", "start", time.Unix(sts, 0), "end", time.Unix(ets, 0))
	snapshot, err := s.db.GetSnapshot()
	if err != nil {
		return data.Candle{}, err
	}
	defer snapshot.Release()
	cs, err := candles(snapshot, amountAsset, priceAsset, ftf, ttf, math.MaxInt32)
	slog.Debug("Combining candles", "count", len(cs))
	if err != nil {
		return data.Candle{}, err
	}
	r := data.Candle{}
	slog.Debug("Empty candle", "candle", r)
	for _, c := range cs {
		r.Combine(c)
		slog.Debug("Appended to candle", "candle", c, "result", r)
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
