package internal

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	client "github.com/wavesplatform/gowaves/pkg/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Synchronizer struct {
	interrupt <-chan struct{}
	done      chan struct{}
	conn      *grpc.ClientConn
	storage   *state.Storage
	scheme    byte
	matcher   crypto.PublicKey
	ticker    *time.Ticker
	lag       int
	symbols   *data.Symbols
}

func NewSynchronizer(interrupt <-chan struct{}, storage *state.Storage, scheme byte, matcher crypto.PublicKey, node string, interval int, lag int, symbols *data.Symbols) (*Synchronizer, error) {
	conn, err := grpc.Dial(node, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new synchronizer")
	}
	d := time.Duration(interval) * time.Second
	t := time.NewTicker(d)
	zap.S().Infof("Synchronization interval set to %v", d)
	done := make(chan struct{})
	s := Synchronizer{interrupt: interrupt, done: done, conn: conn, storage: storage, scheme: scheme, matcher: matcher, ticker: t, lag: lag, symbols: symbols}
	go s.run()
	return &s, nil
}

func (s *Synchronizer) Done() <-chan struct{} {
	return s.done
}

func (s *Synchronizer) run() {
	defer close(s.done)
	for {
		select {
		case <-s.interrupt:
			s.ticker.Stop()
			zap.S().Info("Shutting down synchronizer...")
			return
		case <-s.ticker.C:
			s.synchronize()
		}
	}
}

func (s *Synchronizer) interrupted() bool {
	select {
	case <-s.interrupt:
		return true
	default:
	}
	return false
}

func (s *Synchronizer) synchronize() {
	rh, err := s.nodeHeight()
	rh = rh - s.lag
	if err != nil {
		zap.S().Errorf("Failed to synchronize with node: %v", err)
		return
	}
	lh, err := s.storage.Height()
	if err != nil {
		zap.S().Errorf("Failed to synchronize with node: %v", err)
		return
	}
	if s.interrupted() {
		return
	}
	if rh > lh {
		zap.S().Infof("Local height %d, node height %d", lh, rh)
		ch, err := s.findLastCommonHeight(1, lh)
		if err != nil {
			zap.S().Errorf("Failed to find last common height: %v", err)
			return
		}
		if ch < lh {
			rollbackHeight, err := s.storage.SafeRollbackHeight(ch)
			if err != nil {
				zap.S().Errorf("Failed to get rollback height: %v", err)
				return
			}
			zap.S().Warnf("Rolling back to safe height %d", rollbackHeight)
			err = s.storage.Rollback(rollbackHeight)
			if err != nil {
				zap.S().Errorf("Failed to rollback to height %d: %v", rollbackHeight, err)
				return
			}
			ch = rollbackHeight - 1
		}
		err = s.applyBlocks(ch+1, rh)
		if err != nil && !strings.Contains(err.Error(), "Invalid status code") {
			zap.S().Errorf("Failed to apply blocks: %v", err)
			return
		}
		if s.symbols != nil {
			err = s.symbols.UpdateFromOracle(s.conn)
			if err != nil {
				zap.S().Warnf("Failed to update tickers from oracle: %v", err)
			}
		}
	}
}

func (s *Synchronizer) applyBlocks(start, end int) error {
	zap.S().Infof("Synchronizing %d blocks starting from height %d", end-start+1, start)
	for h := start; h <= end; h++ {
		if s.interrupted() {
			return errors.New("synchronization was interrupted")
		}
		header, txs, err := s.nodeBlock(h)
		if err != nil {
			return err
		}
		err = s.applyBlock(h, header.BlockSignature, txs, len(txs), header.GenPublicKey)
		if err != nil {
			return err
		}
	}
	return nil
}

var emptySignature = crypto.Signature{}

func (s *Synchronizer) applyBlock(height int, id crypto.Signature, txs []proto.Transaction, count int, miner crypto.PublicKey) error {
	if bytes.Equal(id[:], emptySignature[:]) {
		return errors.Errorf("Empty block signature at height: %d", height)
	}
	zap.S().Infof("Applying block '%s' at %d containing %d transactions", id.String(), height, count)
	trades, issues, assets, accounts, aliases, err := s.extractTransactions(txs, miner)
	if err != nil {
		return err
	}
	err = s.storage.PutBalances(height, id, issues, assets, accounts, aliases)
	if err != nil {
		zap.S().Errorf("Failed to update state: %s", err.Error())
	}
	err = s.storage.PutTrades(height, id, trades)
	if err != nil {
		zap.S().Errorf("Failed to update state: %s", err.Error())
	}
	return nil
}

func (s *Synchronizer) nodeHeight() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c := client.NewBlocksApiClient(s.conn)
	h, err := c.GetCurrentHeight(ctx, &empty.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return 0, err
	}
	return int(h.Value), nil
}

func (s *Synchronizer) block(height int, full bool) (*client.BlockWithHeight, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.NewBlocksApiClient(s.conn).GetBlock(ctx, &client.BlockRequest{IncludeTransactions: full, Request: &client.BlockRequest_Height{Height: int32(height)}}, grpc.EmptyCallOption{})
}

func (s *Synchronizer) nodeBlockSignature(height int) (crypto.Signature, error) {
	cnv := client.SafeConverter{}
	res, err := s.block(height, false)
	if err != nil {
		return crypto.Signature{}, err
	}
	header, err := cnv.BlockHeader(res)
	if err != nil {
		return crypto.Signature{}, err
	}
	return header.BlockSignature, nil
}

func (s *Synchronizer) nodeBlock(height int) (proto.BlockHeader, []proto.Transaction, error) {
	cnv := client.SafeConverter{}
	res, err := s.block(height, true)
	if err != nil {
		return proto.BlockHeader{}, nil, err
	}
	header, err := cnv.BlockHeader(res)
	if err != nil {
		return proto.BlockHeader{}, nil, err
	}
	cnv.Reset()
	txs, err := cnv.BlockTransactions(res)
	if err != nil {
		return proto.BlockHeader{}, nil, err
	}
	return header, txs, nil
}

func (s *Synchronizer) findLastCommonHeight(start, stop int) (int, error) {
	var r int
	for start <= stop {
		if s.interrupted() {
			return 0, errors.New("binary search was interrupted")
		}
		middle := (start + stop) / 2
		ok, err := s.equalSignatures(middle)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to compare blocks signatures at height %d", middle)
		}
		if !ok {
			stop = middle - 1
			r = stop
		} else {
			start = middle + 1
			r = middle
		}
	}
	if r == 0 {
		r = 1 //Pretend that we know the genesis block
	}
	return r, nil
}

func (s *Synchronizer) equalSignatures(height int) (bool, error) {
	rbs, err := s.nodeBlockSignature(height)
	if err != nil {
		return false, err
	}
	lbs, err := s.storage.BlockID(height)
	if err != nil {
		return false, err
	}
	return bytes.Equal(rbs[:], lbs[:]), nil
}

func (s *Synchronizer) extractTransactions(txs []proto.Transaction, miner crypto.PublicKey) ([]data.Trade, []data.IssueChange, []data.AssetChange, []data.AccountChange, []data.AliasBind, error) {
	wrapErr := func(err error, transaction string) error {
		return errors.Wrapf(err, "failed to extract %s transaction", transaction)
	}

	trades := make([]data.Trade, 0)
	accountChanges := make([]data.AccountChange, 0)
	assetChanges := make([]data.AssetChange, 0)
	issueChanges := make([]data.IssueChange, 0)
	binds := make([]data.AliasBind, 0)

	for i, tx := range txs {
		switch t := tx.(type) {
		case *proto.IssueV1:
			zap.S().Debugf("#%d: IssueV1: %v", i, t)
			ic, ac, err := data.FromIssueV1(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV1")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)

		case *proto.IssueV2:
			zap.S().Debugf("%d: IssueV2: %v", i, t)
			ic, ac, err := data.FromIssueV2(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV2")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)

		case *proto.TransferV1:
			zap.S().Debugf("%d: TransferV1: %v", i, t)
			tt := *t
			if tt.AmountAsset.Present || tt.FeeAsset.Present {
				u, err := data.FromTransferV1(s.scheme, tt, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV1")
				}
				accountChanges = append(accountChanges, u...)
			}

		case *proto.TransferV2:
			zap.S().Debugf("%d: TransferV2: %v", i, t)
			tt := *t
			if tt.AmountAsset.Present || tt.FeeAsset.Present {
				u, err := data.FromTransferV2(s.scheme, tt, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV2")
				}
				accountChanges = append(accountChanges, u...)
			}

		case *proto.ReissueV1:
			zap.S().Debugf("%d: ReissueV1: %v", i, t)
			as, ac, err := data.FromReissueV1(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV1")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.ReissueV2:
			zap.S().Debugf("%d: ReissueV2: %v", i, t)
			as, ac, err := data.FromReissueV2(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV2")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.BurnV1:
			zap.S().Debugf("%d: BurnV1: %v", i, t)
			as, ac, err := data.FromBurnV1(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnV1")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.BurnV2:
			zap.S().Debugf("%d: BurnV2: %v", i, t)
			as, ac, err := data.FromBurnV2(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnV2")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.ExchangeV1:
			zap.S().Debugf("%d: ExchangeV1: %v", i, t)
			tt := *t
			if bytes.Equal(s.matcher[:], tt.SenderPK[:]) {
				t, err := data.NewTradeFromExchangeV1(s.scheme, tt)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
				}
				trades = append(trades, t)
			}
			ac, err := data.FromExchangeV1(s.scheme, tt)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
			}
			accountChanges = append(accountChanges, ac...)

		case *proto.ExchangeV2:
			zap.S().Debugf("%d: ExchangeV2: %v", i, t)
			tt := *t
			if bytes.Equal(s.matcher[:], tt.SenderPK[:]) {
				t, err := data.NewTradeFromExchangeV2(s.scheme, tt)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
				}
				trades = append(trades, t)
			}
			ac, err := data.FromExchangeV2(s.scheme, tt)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
			}
			accountChanges = append(accountChanges, ac...)

		case *proto.SponsorshipV1:
			zap.S().Debugf("%d: SponsorshipV1: %v", i, t)
			assetChanges = append(assetChanges, data.FromSponsorshipV1(*t))

		case *proto.CreateAliasV1:
			zap.S().Debugf("%d: CreateAliasV1: %v", i, t)
			b, err := data.FromCreateAliasV1(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV1")
			}
			binds = append(binds, b)

		case *proto.CreateAliasV2:
			zap.S().Debugf("%d: CreateAliasV2: %v", i, t)
			b, err := data.FromCreateAliasV2(s.scheme, *t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV2")
			}
			binds = append(binds, b)

		case *proto.MassTransferV1:
			zap.S().Debugf("%d: MassTransferV1: %v", i, t)
			tt := *t
			if tt.Asset.Present {
				ac, err := data.FromMassTransferV1(s.scheme, tt)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferV1")
				}
				accountChanges = append(accountChanges, ac...)
			}

		case *proto.Genesis:
		case *proto.Payment:
		case *proto.LeaseV1:
		case *proto.LeaseV2:
		case *proto.LeaseCancelV1:
		case *proto.LeaseCancelV2:
		case *proto.DataV1:
		case *proto.SetScriptV1:
		case *proto.SetAssetScriptV1:
		case *proto.InvokeScriptV1:
		default:
			zap.S().Warnf("%d: Unknown transaction type", i)
		}
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}
