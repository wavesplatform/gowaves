package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

type Importer struct {
	rootContext context.Context
	log         *zap.SugaredLogger
	storage     *state.Storage
	scheme      byte
	matcher     crypto.PublicKey
}

func NewImporter(ctx context.Context, log *zap.SugaredLogger, scheme byte, storage *state.Storage, matcher crypto.PublicKey) *Importer {
	return &Importer{rootContext: ctx, log: log, scheme: scheme, storage: storage, matcher: matcher}
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
		im.log.Infof("Import took %s", elapsed)
	}()

	f, err := os.Open(n)
	if err != nil {
		return errors.Wrapf(err, "failed to open blockchain file '%s'", n)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			im.log.Errorf("Failed to close blockchain file: %s", err.Error())
		}
	}()

	st, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "failed to get file info")
	}
	im.log.Infof("Importing blockchain file '%s' of size %d bytes", n, st.Size())

	tasks := im.readBlocks(f)

	numWorkers := runtime.NumCPU()
	im.log.Debugf("Number of workers: %d", numWorkers)
	workers := make([]<-chan result, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = im.worker(tasks)
	}

	total := 0
	thousands := 0
	for r := range im.collect(workers...) {
		select {
		case <-im.rootContext.Done():
			im.log.Errorf("Aborted")
			break
		default:
			if r.error != nil {
				im.log.Errorf("Failed to collect transactions for block at height %d: %s", r.height, r.error.Error())
				break
			}
			err := im.storage.PutBlock(r.height, r.id, r.trades, r.issues, r.assets, r.accounts, r.aliases)
			if err != nil {
				im.log.Errorf("Failed to update storage: %s", err.Error())
			}
			c := len(r.trades)
			total += c
			th := total / 10000
			if th > thousands {
				im.log.Infof("Imported %d transactions so far", total)
				thousands = th
			}
			im.log.Debugf("Collected %d transaction at height %d, total transactions so far %d", c, r.height, total)
		}
	}
	im.log.Infof("Total exchange transactions count: %d", total)
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
			case <-im.rootContext.Done():
				im.log.Warnf("Block reading aborted")
				return
			default:
				h++
				t := task{height: h}
				_, err := io.ReadFull(r, sb)
				if err != nil {
					if err != io.EOF {
						im.log.Errorf("Unable to read data size: %s", err.Error())
						return
					}
					im.log.Debug("EOF received while reading size")
					return
				}

				s := binary.BigEndian.Uint32(sb)
				im.log.Debugf("Size: %d", s)
				bb := buf[:s]
				_, err = io.ReadFull(r, bb)
				if err != nil {
					if err != io.EOF {
						im.log.Errorf("Unable to read block: %s", err.Error())
						return
					}
					im.log.Debug("EOF received while reading block")
					return
				}
				err = t.block.UnmarshalBinary(bb)
				if err != nil {
					im.log.Errorf("Failed to unmarshal block: %s", err.Error())
					return
				}
				if !crypto.Verify(t.block.GenPublicKey, t.block.BlockSignature, bb[:len(bb)-crypto.SignatureSize]) {
					im.log.Errorf("Block %s has invalid signature. Aborting.", t.block.BlockSignature.String())
					return
				}
				blockExists, err := im.storage.HasBlock(h, t.block.BlockSignature)
				if err != nil {
					im.log.Errorf("Failed to check block existence: %s", err.Error())
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
	ctx := im.rootContext
	results := make(chan result)

	processTask := func(t task) result {
		r := result{height: t.height, id: t.block.BlockSignature}
		r.trades, r.issues, r.assets, r.accounts, r.aliases, r.error = im.extractTransactions(t.block.Transactions, t.block.TransactionCount, t.block.GenPublicKey)
		return r
	}

	go func() {
		defer close(results)
		for t := range tasks {
			select {
			case <-ctx.Done():
				return
			case results <- processTask(t):
			}
		}
	}()

	return results
}

func (im *Importer) collect(channels ...<-chan result) <-chan result {
	ctx := im.rootContext
	var wg sync.WaitGroup
	multiplexedStream := make(chan result)

	multiplex := func(c <-chan result) {
		defer wg.Done()
		for i := range c {
			select {
			case <-ctx.Done():
				return
			case multiplexedStream <- i:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

func (im *Importer) extractTransactions(d []byte, n int, miner crypto.PublicKey) ([]data.Trade, []data.IssueChange, []data.AssetChange, []data.AccountChange, []data.AliasBind, error) {
	wrapErr := func(err error, transaction string) error {
		return errors.Wrapf(err, "failed to extract %s transaction", transaction)
	}

	trades := make([]data.Trade, 0)
	accountChanges := make([]data.AccountChange, 0)
	assetChanges := make([]data.AssetChange, 0)
	issueChanges := make([]data.IssueChange, 0)
	binds := make([]data.AliasBind, 0)
	for i := 0; i < n; i++ {
		s := int(binary.BigEndian.Uint32(d[0:4]))
		txb := d[4 : s+4]
		switch txb[0] {
		case 0:
			switch txb[1] {
			case byte(proto.IssueTransaction):
				var tx proto.IssueV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "IssueV2")
				}
				ic, ac, err := data.FromIssueV2(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "IssueV2")
				}
				issueChanges = append(issueChanges, ic)
				accountChanges = append(accountChanges, ac)
			case byte(proto.TransferTransaction):
				var tx proto.TransferV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV2")
				}
				if tx.AmountAsset.Present || tx.FeeAsset.Present {
					u, err := data.FromTransferV2(im.scheme, tx, miner)
					if err != nil {
						return nil, nil, nil, nil, nil, wrapErr(err, "TransferV2")
					}
					accountChanges = append(accountChanges, u...)
				}
			case byte(proto.ReissueTransaction):
				var tx proto.ReissueV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV2")
				}
				as, ac, err := data.FromReissueV2(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV2")
				}
				assetChanges = append(assetChanges, as)
				accountChanges = append(accountChanges, ac)
			case byte(proto.BurnTransaction):
				var tx proto.BurnV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "BurnV2")
				}
				as, ac, err := data.FromBurnV2(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "BurnV2")
				}
				assetChanges = append(assetChanges, as)
				accountChanges = append(accountChanges, ac)
			case byte(proto.ExchangeTransaction):
				var tx proto.ExchangeV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
				}
				if bytes.Equal(im.matcher[:], tx.SenderPK[:]) {
					t, err := data.NewTradeFromExchangeV2(im.scheme, tx)
					if err != nil {
						return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
					}
					trades = append(trades, t)
				}
				ac, err := data.FromExchangeV2(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV2")
				}
				accountChanges = append(accountChanges, ac...)
			case byte(proto.SponsorshipTransaction):
				var tx proto.SponsorshipV1
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "SponsorshipV1")
				}
				assetChanges = append(assetChanges, data.FromSponsorshipV1(tx))
			case byte(proto.CreateAliasTransaction):
				var tx proto.CreateAliasV2
				err := tx.UnmarshalBinary(txb)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV2")
				}
				binds = append(binds, data.FromCreateAliasV2(tx))
			}
		case byte(proto.IssueTransaction):
			var tx proto.IssueV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV1")
			}
			if ok, err := tx.Verify(tx.SenderPK); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "IssueV1")
				}
				return nil, nil, nil, nil, nil, wrapErr(errors.Errorf("Transaction %s has invalid signature", tx.ID.String()), "IssueV1")
			}
			ic, ac, err := data.FromIssueV1(im.scheme, tx)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "IssueV1")
			}
			issueChanges = append(issueChanges, ic)
			accountChanges = append(accountChanges, ac)
		case byte(proto.TransferTransaction):
			var tx proto.TransferV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "TransferV1")
			}
			if tx.AmountAsset.Present || tx.FeeAsset.Present {
				ac, err := data.FromTransferV1(im.scheme, tx, miner)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "TransferV1")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case byte(proto.ReissueTransaction):
			var tx proto.ReissueV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV1")
			}
			if ok, err := tx.Verify(tx.SenderPK); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV1")
				}
				return nil, nil, nil, nil, nil, wrapErr(errors.Errorf("Transaction %s has invalid signature", tx.ID.String()), "ReissueV1")
			}
			as, ac, err := data.FromReissueV1(im.scheme, tx)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ReissueV1")
			}
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case byte(proto.BurnTransaction):
			var tx proto.BurnV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "BurnV1")
			}
			if ok, err := tx.Verify(tx.SenderPK); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "BurnV1")
				}
				return nil, nil, nil, nil, nil, wrapErr(errors.Errorf("Transaction %s has invalid signature", tx.ID.String()), "BurnV1")
			}
			as, ac, err := data.FromBurnV1(im.scheme, tx)
			assetChanges = append(assetChanges, as)
			accountChanges = append(accountChanges, ac)
		case byte(proto.ExchangeTransaction):
			var tx proto.ExchangeV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
			}
			if ok, err := tx.Verify(tx.SenderPK); !ok {
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
				}
				return nil, nil, nil, nil, nil, wrapErr(errors.Errorf("Transaction %s has invalid signature", tx.ID.String()), "ExchangeV1")
			}
			if bytes.Equal(im.matcher[:], tx.SenderPK[:]) {
				t, err := data.NewTradeFromExchangeV1(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
				}
				trades = append(trades, t)
			}
			ac, err := data.FromExchangeV1(im.scheme, tx)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "ExchangeV1")
			}
			accountChanges = append(accountChanges, ac...)
		case byte(proto.MassTransferTransaction):
			var tx proto.MassTransferV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferV1")
			}
			if tx.Asset.Present {
				ac, err := data.FromMassTransferV1(im.scheme, tx)
				if err != nil {
					return nil, nil, nil, nil, nil, wrapErr(err, "MassTransferV1")
				}
				accountChanges = append(accountChanges, ac...)
			}
		case byte(proto.CreateAliasTransaction):
			var tx proto.CreateAliasV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, nil, nil, nil, nil, wrapErr(err, "CreateAliasV1")
			}
			binds = append(binds, data.FromCreateAliasV1(tx))
		}
		d = d[4+s:]
	}
	return trades, issueChanges, assetChanges, accountChanges, binds, nil
}
