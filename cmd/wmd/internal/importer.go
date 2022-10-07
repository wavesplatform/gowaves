package internal

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type Importer struct {
	interruptChannel <-chan struct{}
	storage          *state.Storage
	scheme           byte
	matchers         []crypto.PublicKey
}

func NewImporter(interrupt <-chan struct{}, scheme byte, storage *state.Storage, matchers []crypto.PublicKey) *Importer {
	return &Importer{interruptChannel: interrupt, scheme: scheme, storage: storage, matchers: matchers}
}

func (im *Importer) Import(n string) error {
	start := time.Now()

	defer func() {
		elapsed := time.Since(start)
		zap.S().Infof("Import took %s", elapsed)
	}()

	f, err := os.Open(n) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
	if err != nil {
		return errors.Wrapf(err, "failed to open blockchain file '%s'", n)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			zap.S().Errorf("Failed to close blockchain file: %s", err.Error())
		}
	}()

	st, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "failed to get file info")
	}
	zap.S().Infof("Importing blockchain file '%s' of size %d bytes", n, st.Size())

	err = im.readBlocks(f)
	if err != nil {
		return errors.Wrap(err, "import failure")
	}
	return nil
}

func (im *Importer) readBlocks(f io.Reader) error {
	r := bufio.NewReader(f)
	h := 1
	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	total := 0
	thousands := 0
	for {
		select {
		case <-im.interruptChannel:
			zap.S().Warnf("Block reading aborted")
			return nil
		default:
			h++
			_, err := io.ReadFull(r, sb)
			if err != nil {
				if err != io.EOF {
					zap.S().Errorf("Unable to read data size: %s", err.Error())
					return err
				}
				zap.S().Debug("EOF received while reading size")
				return err
			}

			s := binary.BigEndian.Uint32(sb)
			bb := buf[:s]
			_, err = io.ReadFull(r, bb)
			if err != nil {
				if err != io.EOF {
					zap.S().Errorf("Unable to read block: %s", err.Error())
					return err
				}
				zap.S().Debug("EOF received while reading block")
				return err
			}
			var b proto.Block
			err = b.UnmarshalBinary(bb, im.scheme)
			if err != nil {
				err = b.UnmarshalFromProtobuf(bb)
				if err != nil {
					zap.S().Error("Failed to unmarshal block")
					return err
				}
			}
			id := b.BlockID()
			blockExists, err := im.storage.HasBlock(h, id)
			if err != nil {
				zap.S().Errorf("Failed to check block existence: %s", err.Error())
				return err
			}
			if !blockExists {
				validSig, err := b.VerifySignature(im.scheme)
				if err != nil {
					zap.S().Errorf("Failed to verify block signature: %s", err.Error())
				}
				if !validSig {
					zap.S().Errorf("Block %s has invalid signature. Aborting.", b.BlockID().String())
					return err
				}
				trades, issues, assets, accounts, aliases, err := im.extractTransactions(b.Transactions, b.GeneratorPublicKey)
				if err != nil {
					return err
				}
				err = im.storage.PutBalances(h, id, issues, assets, accounts, aliases)
				if err != nil {
					zap.S().Errorf("Failed to update state: %s", err.Error())
					return err
				}
				err = im.storage.PutTrades(h, id, trades)
				if err != nil {
					zap.S().Errorf("Failed to update state: %s", err.Error())
					return err
				}
				c := len(trades)
				total += c
				th := total / 10000
				if th > thousands {
					zap.S().Infof("Imported %4d transactions at height %8d so far", total, h)
					thousands = th
				}
				zap.S().Debugf("Collected %4d transaction at height %8d, total transactions so far %8d", c, h, total)
			}
			if h%10000 == 0 {
				zap.S().Infof("Scanned %8d blocks", h)
			}
		}
	}
}

func (im *Importer) extractTransactions(transactions []proto.Transaction, miner crypto.PublicKey) ([]data.Trade, []data.IssueChange, []data.AssetChange, []data.AccountChange, []data.AliasBind, error) {
	trades := make([]data.Trade, 0)
	accountChanges := make([]data.AccountChange, 0)
	assetChanges := make([]data.AssetChange, 0)
	issueChanges := make([]data.IssueChange, 0)
	binds := make([]data.AliasBind, 0)
	for _, tx := range transactions {
		switch t := tx.(type) {
		case *proto.IssueWithProofs:
			ic, ac, err := data.FromIssueWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferWithProofs:
			if t.AmountAsset.Present || t.FeeAsset.Present {
				u, err := data.FromTransferWithProofs(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				accountChanges = append(accountChanges, u...)
			}
		case *proto.ReissueWithProofs:
			as, ac, err := data.FromReissueWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnWithProofs:
			as, ac, err := data.FromBurnWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeWithProofs:
			if im.checkMatchers(t.SenderPK) {
				tr, err := data.NewTradeFromExchangeWithProofs(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.SponsorshipWithProofs:
			assetChanges = append(assetChanges, data.FromSponsorshipWithProofs(t))
		case *proto.CreateAliasWithProofs:
			b, err := data.FromCreateAliasWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			binds = append(binds, b)
		case *proto.IssueWithSig:
			ic, ac, err := data.FromIssueWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferWithSig:
			if t.AmountAsset.Present || t.FeeAsset.Present {
				ac, err := data.FromTransferWithSig(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.ReissueWithSig:
			as, ac, err := data.FromReissueWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnWithSig:
			as, ac, err := data.FromBurnWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeWithSig:
			if im.checkMatchers(t.SenderPK) {
				tr, err := data.NewTradeFromExchangeWithSig(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.MassTransferWithProofs:
			if t.Asset.Present {
				ac, err := data.FromMassTransferWithProofs(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.CreateAliasWithSig:
			b, err := data.FromCreateAliasWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			binds = append(binds, b)
		}
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}

func (im *Importer) checkMatchers(pk crypto.PublicKey) bool {
	for _, m := range im.matchers {
		if m == pk {
			return true
		}
	}
	return false
}
