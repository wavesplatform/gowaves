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
	id       proto.BlockID
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
				err = t.block.UnmarshalBinary(bb, im.scheme)
				if err != nil {
					zap.S().Errorf("Failed to unmarshal block: %s", err.Error())
					return
				}
				validSig, err := t.block.VerifySignature(im.scheme)
				if err != nil {
					zap.S().Errorf("Failed to verify block signature: %s", err.Error())
				}
				if !validSig {
					zap.S().Errorf("Block %s has invalid signature. Aborting.", t.block.BlockID().String())
					return
				}
				blockExists, err := im.storage.HasBlock(h, t.block.BlockID())
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
		r := result{height: t.height, id: t.block.BlockID()}
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
		case *proto.IssueWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			ic, ac, err := data.FromIssueWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueWithProofs")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.AmountAsset.Present || t.FeeAsset.Present {
				u, err := data.FromTransferWithProofs(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferWithProofs")
				}
				accountChanges = append(accountChanges, u...)
			}
		case *proto.ReissueWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromReissueWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueWithProofs")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromBurnWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnWithProofs")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if bytes.Equal(im.matcher[:], t.SenderPK[:]) {
				tr, err := data.NewTradeFromExchangeWithProofs(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithProofs")
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithProofs")
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.SponsorshipWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			assetChanges = append(assetChanges, data.FromSponsorshipWithProofs(t))
		case *proto.CreateAliasWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			b, err := data.FromCreateAliasWithProofs(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasWithProofs")
			}
			binds = append(binds, b)
		case *proto.IssueWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			ic, ac, err := data.FromIssueWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueWithSig")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case *proto.TransferWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.AmountAsset.Present || t.FeeAsset.Present {
				ac, err := data.FromTransferWithSig(im.scheme, t, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferWithSig")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.ReissueWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromReissueWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueWithSig")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.BurnWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			as, ac, err := data.FromBurnWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnWithSig")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case *proto.ExchangeWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if bytes.Equal(im.matcher[:], t.SenderPK[:]) {
				tr, err := data.NewTradeFromExchangeWithSig(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithSig")
				}
				trades = append(trades, tr)
			}
			ac, err := data.FromExchangeWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeWithSig")
			}
			accountChanges = append(accountChanges, ac...)
		case *proto.MassTransferWithProofs:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			if t.Asset.Present {
				ac, err := data.FromMassTransferWithProofs(im.scheme, t)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferWithProofs")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case *proto.CreateAliasWithSig:
			if ok, err := t.Verify(im.scheme, t.GetSenderPK()); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "failed to verify tx signature")
				}
				id, _ := tx.GetID(im.scheme)
				return nil, nil, nil, nil, nil, errors.Errorf("Transaction %s has invalid signature", base58.Encode(id))
			}
			b, err := data.FromCreateAliasWithSig(im.scheme, t)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasWithSig")
			}
			binds = append(binds, b)
		}
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}
