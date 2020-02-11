package internal

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"io"
	"os"
	"time"
)

type Importer struct {
	interruptChannel <-chan struct{}
	storage          *state.Storage
	scheme           byte
	matcher          crypto.PublicKey
}

func NewImporter(interrupt <-chan struct{}, scheme byte, storage *state.Storage, matcher crypto.PublicKey) *Importer {
	return &Importer{interruptChannel: interrupt, scheme: scheme, storage: storage, matcher: matcher}
}

type task struct {
	height int
	block  proto.Block
}

type result struct {
	height   int
	id       crypto.Signature
	trades   []data.Trade
	issues   []data.IssueChange
	assets   []data.AssetChange
	accounts []data.AccountChange
	aliases  []data.AliasBind
	error    error
}

func (im *Importer) Import(n string) error {
	start := time.Now()

	defer func() {
		elapsed := time.Since(start)
		zap.S().Infof("Import took %s", elapsed)
	}()

	f, err := os.Open(n)
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

	tasks := im.readBlocks(f)

	total := 0
	thousands := 0
	for r := range im.worker(tasks) {
		select {
		case <-im.interruptChannel:
			zap.S().Errorf("Aborted")
			break
		default:
			if r.error != nil {
				zap.S().Errorf("Failed to collect transactions for block at height %d: %s", r.height, r.error.Error())
				break
			}
			err := im.storage.PutBalances(r.height, r.id, r.issues, r.assets, r.accounts, r.aliases)
			if err != nil {
				zap.S().Errorf("Failed to update state: %s", err.Error())
			}
			err = im.storage.PutTrades(r.height, r.id, r.trades)
			if err != nil {
				zap.S().Errorf("Failed to update state: %s", err.Error())
			}
			c := len(r.trades)
			total += c
			th := total / 10000
			if th > thousands {
				zap.S().Infof("Imported %d transactions at height %d so far", total, r.height)
				thousands = th
			}
			zap.S().Debugf("Collected %d transaction at height %d, total transactions so far %d", c, r.height, total)
		}
	}
	zap.S().Infof("Total exchange transactions count: %d", total)
	return nil
}

func (im *Importer) readBlocks(f io.Reader) <-chan task {
	out := make(chan task)
	r := bufio.NewReader(f)
	go func() {
		defer close(out)
		h := 1
		sb := make([]byte, 4)
		buf := make([]byte, 2*1024*1024)
		for {
			select {
			case <-im.interruptChannel:
				zap.S().Warnf("Block reading aborted")
				return
			default:
				h++
				t := task{height: h}
				_, err := io.ReadFull(r, sb)
				if err != nil {
					if err != io.EOF {
						zap.S().Errorf("Unable to read data size: %s", err.Error())
						return
					}
					zap.S().Debug("EOF received while reading size")
					return
				}

				s := binary.BigEndian.Uint32(sb)
				zap.S().Debugf("Size: %d", s)
				bb := buf[:s]
				_, err = io.ReadFull(r, bb)
				if err != nil {
					if err != io.EOF {
						zap.S().Errorf("Unable to read block: %s", err.Error())
						return
					}
					zap.S().Debug("EOF received while reading block")
					return
				}
				err = t.block.UnmarshalBinary(bb)
				if err != nil {
					zap.S().Errorf("Failed to unmarshal block: %s", err.Error())
					return
				}
				if !crypto.Verify(t.block.GenPublicKey, t.block.BlockSignature, bb[:len(bb)-crypto.SignatureSize]) {
					zap.S().Errorf("Block %s has invalid signature. Aborting.", t.block.BlockSignature.String())
					return
				}
				blockExists, err := im.storage.HasBlock(h, t.block.BlockSignature)
				if err != nil {
					zap.S().Errorf("Failed to check block existence: %s", err.Error())
					return
				}
				if !blockExists {
					out <- t
				}
			}
		}
	}()
	return out
}

func (im *Importer) worker(tasks <-chan task) <-chan result {
	results := make(chan result)

	processTask := func(t task) result {
		r := result{height: t.height, id: t.block.BlockSignature}
		r.trades, r.issues, r.assets, r.accounts, r.aliases, r.error = im.extractTransactions(t.block.Transactions, t.block.GenPublicKey)
		return r
	}

	go func() {
		defer close(results)
		for t := range tasks {
			select {
			case <-im.interruptChannel:
				return
			case results <- processTask(t):
			}
		}
	}()

	return results
}

func (im *Importer) extractTransactions(transactions []proto.Transaction, miner crypto.PublicKey) ([]data.Trade, []data.IssueChange, []data.AssetChange, []data.AccountChange, []data.AliasBind, error) {
	wrapErr := func(err error, transaction string) error {
		return errors.Wrapf(err, "failed to extract %s transaction", transaction)
	}

	trades := make([]data.Trade, 0)
	accountChanges := make([]data.AccountChange, 0)
	assetChanges := make([]data.AssetChange, 0)
	issueChanges := make([]data.IssueChange, 0)
	binds := make([]data.AliasBind, 0)
	for _, tx := range transactions {
		switch t := tx.(type) {
		case *proto.IssueV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			ic, ac, err := data.FromIssueV2(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV2")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.AmountAsset.Present || t.FeeAsset.Present {
				u, err := data.FromTransferV2(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV2")
				}
				accountChanges = append(accountChanges, u...)
			}
		case *proto.ReissueV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromReissueV2(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV2")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromBurnV2(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnV2")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if bytes.Equal(im.matcher[:], t.SenderPK[:]) {
				tr, err := data.NewTradeFromExchangeV2(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeV2(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.SponsorshipV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			assetChanges = append(assetChanges, data.FromSponsorshipV1(t))
		case *proto.CreateAliasV2:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			b, err := data.FromCreateAliasV2(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV2")
			}
			binds = append(binds, b)
		case *proto.IssueV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			ic, ac, err := data.FromIssueV1(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV1")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.AmountAsset.Present || t.FeeAsset.Present {
				ac, err := data.FromTransferV1(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV1")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.ReissueV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromReissueV1(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV1")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromBurnV1(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnV1")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if bytes.Equal(im.matcher[:], t.SenderPK[:]) {
				tr, err := data.NewTradeFromExchangeV1(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeV1(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.MassTransferV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.Asset.Present {
				ac, err := data.FromMassTransferV1(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferV1")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.CreateAliasV1:
			if ok, err := t.Verify(t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID()
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			b, err := data.FromCreateAliasV1(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV1")
			}
			binds = append(binds, b)
		}
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}
