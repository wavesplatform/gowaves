package storage

import (
	"bufio"
	"encoding/binary"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type KeyValue interface {
	Put(key []byte, val []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
}

type BlockReadWriter struct {
	// KeyValue to store ID --> offset in blockchain for blocks and transactions.
	idKeyVal KeyValue

	// Series of transactions.
	blockchain *os.File
	// Series of BlockHeader.
	headers *os.File
	// Height is used as index for block IDs.
	blockHeight2ID *os.File

	blockchainBuf *bufio.Writer
	compressedBuf []byte

	blockBounds  []byte
	txBounds     []byte
	headerBounds []byte
	heightBuf    []byte

	// offsetEnd is common for headers and the blockchain, since the limit for any offset length is 8 bytes.
	offsetEnd                  uint64
	blockchainLen, headersLen  uint64
	height                     uint64
	offsetLen, headerOffsetLen int

	mtx sync.RWMutex
}

func NewBlockReadWriter(dir string, offsetLen, headerOffsetLen int, keyVal KeyValue) (*BlockReadWriter, error) {
	if contentList, err := ioutil.ReadDir(dir); err != nil {
		return nil, errors.Wrap(err, "Error when reading output dir")
	} else if len(contentList) != 0 {
		return nil, errors.Errorf("Output dir is not empty")
	}
	blockchain, err := os.Create(path.Join(dir, "blockchain"))
	if err != nil {
		return nil, err
	}
	headers, err := os.Create(path.Join(dir, "headers"))
	if err != nil {
		return nil, err
	}
	blockHeight2ID, err := os.Create(path.Join(dir, "block_height_to_id"))
	if err != nil {
		return nil, err
	}
	if offsetLen > 8 {
		return nil, errors.New("offsetLen is too large")
	}
	if headerOffsetLen > 8 {
		return nil, errors.New("headerOffsetLen is too large")
	}
	return &BlockReadWriter{
		idKeyVal:        keyVal,
		blockchain:      blockchain,
		headers:         headers,
		blockHeight2ID:  blockHeight2ID,
		blockchainBuf:   bufio.NewWriter(blockchain),
		txBounds:        make([]byte, offsetLen*2),
		headerBounds:    make([]byte, headerOffsetLen*2),
		blockBounds:     make([]byte, offsetLen*2),
		heightBuf:       make([]byte, 8),
		offsetEnd:       uint64(1<<uint(8*offsetLen) - 1),
		offsetLen:       offsetLen,
		headerOffsetLen: headerOffsetLen,
	}, nil
}

func (rw *BlockReadWriter) blockIDByHeight(height uint64) (crypto.Signature, error) {
	idBytes := make([]byte, crypto.SignatureSize)
	readPos := int64(height * crypto.SignatureSize)
	var res crypto.Signature
	if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
		return res, err
	} else if n != crypto.SignatureSize {
		return res, errors.New("blockIDByHeight(): invalid id size")
	}
	copy(res[:], idBytes)
	return res, nil
}

func (rw *BlockReadWriter) StartBlock(blockID crypto.Signature) error {
	if _, err := rw.blockHeight2ID.Write(blockID[:]); err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(rw.blockBounds[:rw.offsetLen], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[:rw.headerOffsetLen], rw.headersLen)
	return nil
}

func (rw *BlockReadWriter) FinishBlock(blockID crypto.Signature) error {
	binary.LittleEndian.PutUint64(rw.blockBounds[rw.offsetLen:], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[rw.headerOffsetLen:], rw.headersLen)
	binary.LittleEndian.PutUint64(rw.heightBuf, rw.height)
	rw.height++
	val := append(rw.blockBounds, rw.headerBounds...)
	val = append(val, rw.heightBuf...)
	if err := rw.idKeyVal.Put(blockID[:], val); err != nil {
		return err
	}
	return nil
}

func (rw *BlockReadWriter) WriteTransaction(txID []byte, tx []byte) error {
	rw.compressedBuf = snappy.Encode(rw.compressedBuf, tx)
	if _, err := rw.blockchainBuf.Write(rw.compressedBuf); err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(rw.txBounds[:rw.offsetLen], rw.blockchainLen)
	rw.blockchainLen += uint64(len(tx))
	if rw.blockchainLen > rw.offsetEnd {
		return errors.Errorf("offsetLen is not enough for this offset: %d > %d", rw.blockchainLen, rw.offsetEnd)
	}
	binary.LittleEndian.PutUint64(rw.txBounds[rw.offsetLen:], rw.blockchainLen)
	if err := rw.idKeyVal.Put(txID, rw.txBounds); err != nil {
		return err
	}
	return nil
}

func (rw *BlockReadWriter) WriteBlockHeader(blockID crypto.Signature, header []byte) error {
	if _, err := rw.headers.Write(header); err != nil {
		return err
	}
	rw.headersLen += uint64(len(header))
	if rw.headersLen > rw.offsetEnd {
		return errors.Errorf("offsetLen is not enough for this offset: %d > %d", rw.headersLen, rw.offsetEnd)
	}
	return nil
}

func (rw *BlockReadWriter) ReadTransaction(txID []byte) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	txBounds, err := rw.idKeyVal.Get(txID)
	if err != nil {
		return nil, err
	}
	txStart := binary.LittleEndian.Uint64(txBounds[:rw.offsetLen])
	txEnd := binary.LittleEndian.Uint64(txBounds[rw.offsetLen:])
	txBytes := make([]byte, txEnd-txStart)
	n, err := rw.blockchain.ReadAt(txBytes, int64(txStart))
	if err != nil {
		return nil, err
	} else if n != len(txBytes) {
		return nil, errors.New("ReadAt did not read the whole tx")
	}
	return txBytes, nil
}

func (rw *BlockReadWriter) ReadBlockHeader(blockID crypto.Signature) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	blockInfo, err := rw.idKeyVal.Get(blockID[:])
	if err != nil {
		return nil, err
	}
	headerBounds := blockInfo[rw.offsetLen*2 : len(blockInfo)-8]
	headerStart := binary.LittleEndian.Uint64(headerBounds[:rw.headerOffsetLen])
	headerEnd := binary.LittleEndian.Uint64(headerBounds[rw.headerOffsetLen:])
	headerBytes := make([]byte, headerEnd-headerStart)
	n, err := rw.headers.ReadAt(headerBytes, int64(headerStart))
	if err != nil {
		return nil, err
	} else if n != len(headerBytes) {
		return nil, errors.New("ReadAt did not read the whole header")
	}
	return headerBytes, nil
}

func (rw *BlockReadWriter) ReadTransactionsBlock(blockID crypto.Signature) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	blockInfo, err := rw.idKeyVal.Get(blockID[:])
	if err != nil {
		return nil, err
	}
	blockBounds := blockInfo[:rw.offsetLen*2]
	blockStart := binary.LittleEndian.Uint64(blockBounds[:rw.offsetLen])
	blockEnd := binary.LittleEndian.Uint64(blockBounds[rw.offsetLen:])
	blockBytes := make([]byte, blockEnd-blockStart)
	n, err := rw.blockchain.ReadAt(blockBytes, int64(blockStart))
	if err != nil {
		return nil, err
	} else if n != len(blockBytes) {
		return nil, errors.New("ReadAt did not read the whole block")
	}
	return blockBytes, nil
}

func (rw *BlockReadWriter) RemoveBlocks(removalEdge crypto.Signature) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	blockInfo, err := rw.idKeyVal.Get(removalEdge[:])
	if err != nil {
		return err
	}
	// Remove transactions.
	blockBounds := blockInfo[:rw.offsetLen*2]
	blockEnd := int64(binary.LittleEndian.Uint64(blockBounds[rw.offsetLen:]))
	if err := rw.blockchain.Truncate(blockEnd); err != nil {
		return err
	}
	if _, err := rw.blockchain.Seek(blockEnd, 0); err != nil {
		return err
	}
	// Remove headers.
	headerBounds := blockInfo[rw.offsetLen*2:]
	headerEnd := int64(binary.LittleEndian.Uint64(headerBounds[rw.headerOffsetLen:]))
	if err := rw.headers.Truncate(headerEnd); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(headerEnd, 0); err != nil {
		return err
	}
	// Remove blockIDs from blocHeight2ID.
	newHeight := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:])
	newOffset := int64(newHeight * crypto.SignatureSize)
	if err := rw.blockHeight2ID.Truncate(newOffset); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(newOffset, 0); err != nil {
		return err
	}
	// Decrease counters.
	rw.blockchainLen = uint64(blockEnd)
	rw.headersLen = uint64(headerEnd)
	rw.height = newHeight
	// Reset buffer.
	rw.blockchainBuf.Reset(rw.blockchain)
	// TODO remove outdated IDs from idKeyVal.
	// Iterating all the transactions IDs slows down block rollback too much,
	// this needs smarter approach.
	return nil
}

func (rw *BlockReadWriter) Close() error {
	if err := rw.blockchainBuf.Flush(); err != nil {
		return err
	}
	if err := rw.blockchain.Close(); err != nil {
		return err
	}
	if err := rw.headers.Close(); err != nil {
		return err
	}
	if err := rw.blockHeight2ID.Close(); err != nil {
		return err
	}
	return nil
}
