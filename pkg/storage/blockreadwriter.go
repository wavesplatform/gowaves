package storage

import (
	"bufio"
	"encoding/binary"
	"os"
	"path"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

type BlockReadWriter struct {
	// keyvalue.KeyValue to store ID --> offset in blockchain for blocks and transactions.
	idKeyVal keyvalue.KeyValue

	// Series of transactions.
	blockchain *os.File
	// Series of BlockHeader.
	headers *os.File
	// Height is used as index for block IDs.
	blockHeight2ID *os.File
	// IDs of transactions.
	txIDs *os.File

	blockchainBuf *bufio.Writer

	blockBounds  []byte
	txBounds     []byte
	headerBounds []byte
	heightBuf    []byte
	txNumberBuf  []byte

	// offsetEnd is common for headers and the blockchain, since the limit for any offset length is 8 bytes.
	offsetEnd                 uint64
	blockchainLen, headersLen uint64
	height                    uint64
	// Total number of transactions.
	txNumber                   uint64
	offsetLen, headerOffsetLen int

	mtx sync.RWMutex
}

func openOrCreate(path string) (*os.File, uint64, error) {
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return nil, 0, err
		}
		stat, err := os.Stat(path)
		if err != nil {
			return nil, 0, err
		}
		return file, uint64(stat.Size()), nil
	} else if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return nil, 0, err
		}
		return file, 0, nil
	} else {
		return nil, 0, err
	}
}

func NewBlockReadWriter(dir string, offsetLen, headerOffsetLen int, keyVal keyvalue.KeyValue) (*BlockReadWriter, error) {
	blockchain, blockchainSize, err := openOrCreate(path.Join(dir, "blockchain"))
	if err != nil {
		return nil, err
	}
	headers, headersSize, err := openOrCreate(path.Join(dir, "headers"))
	if err != nil {
		return nil, err
	}
	blockHeight2ID, blockHeight2IDSize, err := openOrCreate(path.Join(dir, "block_height_to_id"))
	if err != nil {
		return nil, err
	}
	txIDs, txIDsSize, err := openOrCreate(path.Join(dir, "tx_ids"))
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
		txIDs:           txIDs,
		blockchainBuf:   bufio.NewWriter(blockchain),
		txBounds:        make([]byte, offsetLen*2),
		headerBounds:    make([]byte, headerOffsetLen*2),
		blockBounds:     make([]byte, offsetLen*2),
		heightBuf:       make([]byte, 8),
		txNumberBuf:     make([]byte, 8),
		offsetEnd:       uint64(1<<uint(8*offsetLen) - 1),
		blockchainLen:   blockchainSize,
		headersLen:      headersSize,
		height:          blockHeight2IDSize / crypto.SignatureSize,
		txNumber:        txIDsSize / crypto.DigestSize,
		offsetLen:       offsetLen,
		headerOffsetLen: headerOffsetLen,
	}, nil
}

func (rw *BlockReadWriter) BlockIdsFilePath() (string, error) {
	if rw.blockHeight2ID != nil {
		return rw.blockHeight2ID.Name(), nil
	}
	return "", errors.New("block IDs file is not set")
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
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	binary.LittleEndian.PutUint64(rw.blockBounds[rw.offsetLen:], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[rw.headerOffsetLen:], rw.headersLen)
	binary.LittleEndian.PutUint64(rw.heightBuf, rw.height)
	rw.height++
	val := append(rw.blockBounds, rw.headerBounds...)
	val = append(val, rw.heightBuf...)
	val = append(val, rw.txNumberBuf...)
	if err := rw.idKeyVal.Put(blockID[:], val); err != nil {
		return err
	}
	if err := rw.idKeyVal.Flush(); err != nil {
		return err
	}
	if err := rw.blockchainBuf.Flush(); err != nil {
		return err
	}
	return nil
}

func (rw *BlockReadWriter) WriteTransaction(txID []byte, tx []byte) error {
	if _, err := rw.blockchainBuf.Write(tx); err != nil {
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
	if _, err := rw.txIDs.Write(txID); err != nil {
		return err
	}
	rw.txNumber++
	binary.LittleEndian.PutUint64(rw.txNumberBuf, rw.txNumber)
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

func (rw *BlockReadWriter) BlockIDByHeight(height uint64) (crypto.Signature, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	idBytes := make([]byte, crypto.SignatureSize)
	readPos := int64(height * crypto.SignatureSize)
	var res crypto.Signature
	if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
		return res, err
	} else if n != crypto.SignatureSize {
		return res, errors.New("BlockIDByHeight(): invalid id size")
	}
	copy(res[:], idBytes)
	return res, nil
}

func (rw *BlockReadWriter) HeightByBlockID(blockID crypto.Signature) (uint64, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	blockInfo, err := rw.idKeyVal.Get(blockID[:])
	if err != nil {
		return 0, err
	}
	height := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-16 : len(blockInfo)-8])
	return height, nil
}

func (rw *BlockReadWriter) CurrentHeight() uint64 {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.height
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
	headerBounds := blockInfo[rw.offsetLen*2 : len(blockInfo)-16]
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

func (rw *BlockReadWriter) cleanIDs(newHeight, newTxNumber uint64) error {
	// Clean block IDs.
	offset := rw.height
	blocksIdsToRemove := int(rw.height - newHeight)
	for i := 0; i < blocksIdsToRemove; i++ {
		readPos := int64((offset - 1) * crypto.SignatureSize)
		idBytes := make([]byte, crypto.SignatureSize)
		if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
			return err
		} else if n != crypto.SignatureSize {
			return errors.New("cleanIDs(): invalid id size")
		}
		if err := rw.idKeyVal.Delete(idBytes); err != nil {
			return err
		}
		offset--
	}
	// Clean transaction IDs.
	offset = rw.txNumber
	txIdsToRemove := int(rw.txNumber - newTxNumber)
	for i := 0; i < txIdsToRemove; i++ {
		readPos := int64((offset - 1) * crypto.DigestSize)
		idBytes := make([]byte, crypto.DigestSize)
		if n, err := rw.txIDs.ReadAt(idBytes, readPos); err != nil {
			return err
		} else if n != crypto.DigestSize {
			return errors.New("cleanIDs(): invalid id size")
		}
		if err := rw.idKeyVal.Delete(idBytes); err != nil {
			return err
		}
		offset--
	}
	// Remove blockIDs from blockHeight2ID file.
	newOffset := int64(newHeight * crypto.SignatureSize)
	if err := rw.blockHeight2ID.Truncate(newOffset); err != nil {
		return err
	}
	if _, err := rw.blockHeight2ID.Seek(newOffset, 0); err != nil {
		return err
	}
	// Remove txIDs from txIDs file.
	newOffset = int64(newTxNumber * crypto.DigestSize)
	if err := rw.txIDs.Truncate(newOffset); err != nil {
		return err
	}
	if _, err := rw.txIDs.Seek(newOffset, 0); err != nil {
		return err
	}
	return nil
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
	headerBounds := blockInfo[rw.offsetLen*2 : len(blockInfo)-16]
	headerEnd := int64(binary.LittleEndian.Uint64(headerBounds[rw.headerOffsetLen:]))
	if err := rw.headers.Truncate(headerEnd); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(headerEnd, 0); err != nil {
		return err
	}
	newHeight := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-16:len(blockInfo)-8]) + 1
	newTxNumber := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:]) + 1
	// Clean IDs of blocks and transactions.
	if err := rw.cleanIDs(newHeight, newTxNumber); err != nil {
		return nil
	}
	// Decrease counters.
	rw.blockchainLen = uint64(blockEnd)
	rw.headersLen = uint64(headerEnd)
	rw.height = newHeight
	rw.txNumber = newTxNumber
	// Reset buffer.
	rw.blockchainBuf.Reset(rw.blockchain)
	return nil
}

func (rw *BlockReadWriter) Close() error {
	if err := rw.idKeyVal.Flush(); err != nil {
		return err
	}
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
	if err := rw.txIDs.Close(); err != nil {
		return err
	}
	return nil
}
