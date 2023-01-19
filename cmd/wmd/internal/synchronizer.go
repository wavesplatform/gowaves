package internal

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Synchronizer struct {
	interrupt <-chan struct{}
	done      chan struct{}
	conn      *grpc.ClientConn
	storage   *state.Storage
	scheme    byte
	matchers  []crypto.PublicKey
	interval  time.Duration
	lag       int
	symbols   *data.Symbols
}

func NewSynchronizer(interrupt <-chan struct{}, storage *state.Storage, scheme byte, matchers []crypto.PublicKey, node string, interval time.Duration, lag int, symbols *data.Symbols) (*Synchronizer, error) {
	conn, err := grpc.Dial(node, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new synchronizer")
	}
	zap.S().Infof("Synchronization interval set to %v", interval)
	done := make(chan struct{})
	s := Synchronizer{interrupt: interrupt, done: done, conn: conn, storage: storage, scheme: scheme, matchers: matchers, interval: interval, lag: lag, symbols: symbols}
	go s.run()
	return &s, nil
}

func (s *Synchronizer) Done() <-chan struct{} {
	return s.done
}

func (s *Synchronizer) run() {
	ticker := time.NewTicker(s.interval)
	defer func() {
		ticker.Stop()
		close(s.done)
	}()
	for {
		select {
		case <-s.interrupt:
			zap.S().Info("Shutting down synchronizer...")
			return
		case <-ticker.C:
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
		const delta = 10
		err = s.applyBlocksRange(ch+1, rh, delta)
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

func (s *Synchronizer) applyBlocksRange(start, end, delta int) error {
	zap.S().Infof("Synchronizing %d blocks in range starting from height %d with delta %d", end-start+1, start, delta)
	cnv := proto.ProtobufConverter{FallbackChainID: s.scheme}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for h := start; h <= end; h += delta {
		if s.interrupted() {
			return errors.New("synchronization was interrupted")
		}
		if h+delta > end {
			delta = end - h + 1
		}
		stream, err := s.blockRange(h, h+delta-1, ctx, true)
		if err != nil {
			return errors.Wrapf(err, "failed to get %d blocks from node from height %d to height %d ", delta, start, end)
		}

		ids, miners, txss, err := s.recvBlockRange(h, delta, stream, cnv)

		if err != nil {
			return errors.Wrapf(err, "failed to receive %d blocks from node from height %d to height %d ", delta, start, end)
		}
		for i := 0; i < delta; i++ {
			err = s.applyBlock(h+i, ids[i], txss[i], miners[i])
			if err != nil {
				return errors.Wrapf(err, "failed apply block at height %d", h)
			}
		}

	}
	return nil
}

func (s *Synchronizer) recvBlockRange(h int, delta int, stream g.BlocksApi_GetBlockRangeClient, cnv proto.ProtobufConverter) ([]proto.BlockID, []crypto.PublicKey, [][]proto.Transaction, error) {
	var txss [][]proto.Transaction
	var headersIDs []proto.BlockID
	var headersGenPublicKeys []crypto.PublicKey
	for i := h; i < h+delta; i++ {
		block, err := stream.Recv()
		if err != nil {
			return []proto.BlockID{}, []crypto.PublicKey{}, nil, err
		}
		header, err := cnv.BlockHeader(block.GetBlock())
		if err != nil {
			return []proto.BlockID{}, []crypto.PublicKey{}, nil, err
		}
		headersIDs = append(headersIDs, header.ID)
		headersGenPublicKeys = append(headersGenPublicKeys, header.GeneratorPublicKey)

		txs, err := cnv.SignedTransactions(block.GetBlock().GetTransactions())
		if err != nil {
			return []proto.BlockID{}, []crypto.PublicKey{}, nil, err
		}
		txss = append(txss, txs)
	}

	return headersIDs, headersGenPublicKeys, txss, nil
}

func (s *Synchronizer) blockRange(start int, end int, ctx context.Context, full bool) (g.BlocksApi_GetBlockRangeClient, error) {
	return g.NewBlocksApiClient(s.conn).GetBlockRange(ctx, &g.BlockRangeRequest{
		FromHeight:          uint32(start),
		ToHeight:            uint32(end),
		Filter:              nil,
		IncludeTransactions: full,
	}, grpc.EmptyCallOption{})
}

var emptyID = proto.BlockID{}

func (s *Synchronizer) applyBlock(height int, id proto.BlockID, txs []proto.Transaction, miner crypto.PublicKey) error {
	if id == emptyID {
		return errors.Errorf("Empty block id at height: %d", height)
	}
	zap.S().Infof("Applying block '%s' at %d containing %d transactions", id.String(), height, len(txs))
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := g.NewBlocksApiClient(s.conn)
	h, err := c.GetCurrentHeight(ctx, &emptypb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return 0, err
	}
	return int(h.Value), nil
}

func (s *Synchronizer) block(height int, full bool) (*g.BlockWithHeight, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return g.NewBlocksApiClient(s.conn).GetBlock(ctx, &g.BlockRequest{IncludeTransactions: full, Request: &g.BlockRequest_Height{Height: int32(height)}}, grpc.EmptyCallOption{})
}

func (s *Synchronizer) nodeBlockID(height int) (proto.BlockID, error) {
	cnv := proto.ProtobufConverter{FallbackChainID: s.scheme}
	res, err := s.block(height, false)
	if err != nil {
		return proto.BlockID{}, err
	}
	header, err := cnv.BlockHeader(res.Block)
	if err != nil {
		return proto.BlockID{}, err
	}
	return header.BlockID(), nil
}

func (s *Synchronizer) findLastCommonHeight(start, stop int) (int, error) {
	var r int
	for start <= stop {
		if s.interrupted() {
			return 0, errors.New("binary search was interrupted")
		}
		middle := (start + stop) / 2
		ok, err := s.equalIDs(middle)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to compare blocks ids at height %d", middle)
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

func (s *Synchronizer) equalIDs(height int) (bool, error) {
	rbs, err := s.nodeBlockID(height)
	if err != nil {
		return false, err
	}
	lbs, err := s.storage.BlockID(height)
	if err != nil {
		return false, err
	}
	return bytes.Equal(rbs.Bytes(), lbs.Bytes()), nil
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
		case *proto.IssueWithSig:
			zap.S().Debugf("#%d: IssueWithSig: %v", i, t)
			ic, ac, err := data.FromIssueWithSig(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueWithSig")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)

		case *proto.IssueWithProofs:
			zap.S().Debugf("%d: IssueWithProofs: %v", i, t)
			ic, ac, err := data.FromIssueWithProofs(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueWithProofs")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)

		case *proto.TransferWithSig:
			zap.S().Debugf("%d: TransferWithSig: %v", i, t)
			if t.AmountAsset.Present || t.FeeAsset.Present {
				u, err := data.FromTransferWithSig(s.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferWithSig")
				}
				accountChanges = append(accountChanges, u...)
			}

		case *proto.TransferWithProofs:
			zap.S().Debugf("%d: TransferWithProofs: %v", i, t)
			if t.AmountAsset.Present || t.FeeAsset.Present {
				u, err := data.FromTransferWithProofs(s.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferWithProofs")
				}
				accountChanges = append(accountChanges, u...)
			}

		case *proto.ReissueWithSig:
			zap.S().Debugf("%d: ReissueWithSig: %v", i, t)
			as, ac, err := data.FromReissueWithSig(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueWithSig")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.ReissueWithProofs:
			zap.S().Debugf("%d: ReissueWithProofs: %v", i, t)
			as, ac, err := data.FromReissueWithProofs(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueWithProofs")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.BurnWithSig:
			zap.S().Debugf("%d: BurnWithSig: %v", i, t)
			as, ac, err := data.FromBurnWithSig(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnWithSig")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.BurnWithProofs:
			zap.S().Debugf("%d: BurnWithProofs: %v", i, t)
			as, ac, err := data.FromBurnWithProofs(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnWithProofs")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)

		case *proto.ExchangeWithSig:
			zap.S().Debugf("%d: ExchangeWithSig: %v", i, t)
			if s.checkMatcher(t.SenderPK) {
				t, err := data.NewTradeFromExchangeWithSig(s.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithSig")
				}
				trades = append(trades, t)
			}
			ac, err := data.FromExchangeWithSig(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithSig")
			}
			accountChanges = append(accountChanges, ac...)

		case *proto.ExchangeWithProofs:
			zap.S().Debugf("%d: ExchangeWithProofs: %v", i, t)
			if s.checkMatcher(t.SenderPK) {
				t, err := data.NewTradeFromExchangeWithProofs(s.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithProofs")
				}
				trades = append(trades, t)
			}
			ac, err := data.FromExchangeWithProofs(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithProofs")
			}
			accountChanges = append(accountChanges, ac...)

		case *proto.SponsorshipWithProofs:
			zap.S().Debugf("%d: SponsorshipWithProofs: %v", i, t)
			assetChanges = append(assetChanges, data.FromSponsorshipWithProofs(t))

		case *proto.CreateAliasWithSig:
			zap.S().Debugf("%d: CreateAliasWithSig: %v", i, t)
			b, err := data.FromCreateAliasWithSig(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasWithSig")
			}
			binds = append(binds, b)

		case *proto.CreateAliasWithProofs:
			zap.S().Debugf("%d: CreateAliasWithProofs: %v", i, t)
			b, err := data.FromCreateAliasWithProofs(s.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasWithProofs")
			}
			binds = append(binds, b)

		case *proto.MassTransferWithProofs:
			zap.S().Debugf("%d: MassTransferWithProofs: %v", i, t)
			if t.Asset.Present {
				ac, err := data.FromMassTransferWithProofs(s.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferWithProofs")
				}
				accountChanges = append(accountChanges, ac...)
			}

		case *proto.Genesis:
		case *proto.Payment:
		case *proto.LeaseWithSig:
		case *proto.LeaseWithProofs:
		case *proto.LeaseCancelWithSig:
		case *proto.LeaseCancelWithProofs:
		case *proto.DataWithProofs:
		case *proto.SetScriptWithProofs:
		case *proto.SetAssetScriptWithProofs:
		case *proto.InvokeScriptWithProofs:
		default:
			zap.S().Warnf("%d: Unknown transaction type", i)
		}
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}

func (s *Synchronizer) checkMatcher(pk crypto.PublicKey) bool {
	for _, m := range s.matchers {
		if m == pk {
			return true
		}
	}
	return false
}
