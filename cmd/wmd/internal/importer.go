package internal

import (
	"bufio"
	"context"
	"encoding/binary"
	"github.com/pkg/errors"
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
	storage     *Storage
}

func NewImporter(ctx context.Context, log *zap.SugaredLogger, storage *Storage) *Importer {
	return &Importer{rootContext: ctx, log: log, storage: storage}
}

type task struct {
	height int
	block  Block
}

type result struct {
	height int
	id     crypto.Signature
	txs    []proto.ExchangeV1
	error  error
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
			trades := make([]Trade, 0)
			for _, tx := range r.txs {
				t := NewTradeFromExchangeV1(tx)
				trades = append(trades, t)
			}
			err := im.storage.PutBlock(r.height, r.id, trades)
			if err != nil {
				im.log.Errorf("Failed to update storage: %s", err.Error())
			}
			c := len(r.txs)
			total += c
			th := total / 1000
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
				ok, err := im.storage.ShouldImportBlock(h, t.block.BlockSignature)
				if err != nil {
					im.log.Errorf("Failed to check block in DB: %s", err.Error())
					return
				}
				if ok {
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
		r.txs, r.error = im.parseTransactions(t.block)
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

func (im *Importer) parseTransactions(b Block) ([]proto.ExchangeV1, error) {
	switch b.Version {
	case 3:
		n := binary.BigEndian.Uint32(b.Transactions[:4])
		return im.extractTransactions(b.Transactions[4:], int(n))
	default:
		n := b.Transactions[0]
		return im.extractTransactions(b.Transactions[1:], int(n))
	}
}

func (im *Importer) extractTransactions(d []byte, n int) ([]proto.ExchangeV1, error) {
	r := make([]proto.ExchangeV1, 0)
	for i := 0; i < n; i++ {
		s := int(binary.BigEndian.Uint32(d[0:4]))
		txb := d[4:s]
		if txb[0] == byte(proto.ExchangeTransaction) {
			var tx proto.ExchangeV1
			err := tx.UnmarshalBinary(txb)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract ExchangeV1 transactions")
			}
			if ok, err := tx.Verify(tx.SenderPK); !ok {
				if err != nil {
					return nil, errors.Wrap(err, "failed to verify ExchangeV1 transaction signature")
				}
				return nil, errors.Errorf("Transaction %s has invalid signature", tx.ID.String())
			}
			r = append(r, tx)
		}
		d = d[4+s:]
	}
	return r, nil
}
