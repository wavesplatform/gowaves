package state

import (
	"bufio"
	"encoding/binary"
	"os"
	"path"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlockReadWriter struct {
	Db keyvalue.KeyValue

	// Series of transactions.
	blockchain *os.File
	// Series of BlockHeader.
	headers *os.File
	// Height is used as index for block IDs.
	blockHeight2ID *os.File

	blockchainBuf *bufio.Writer

	blockBounds  []byte
	txBounds     []byte
	headerBounds []byte
	heightBuf    []byte

	// offsetEnd is common for headers and the blockchain, since the limit for any offset length is 8 bytes.
	offsetEnd                 uint64
	blockchainLen, headersLen uint64
	// Total number of transactions.
	offsetLen, headerOffsetLen int

	mtx sync.RWMutex
}

func openOrCreate(path string) (*os.File, uint64, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, 0, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	return file, uint64(stat.Size()), nil
}

func initHeight(db keyvalue.KeyValue) (uint64, error) {
	has, err := db.Has([]byte{RwHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	if !has {
		heightBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.PutDirectly([]byte{RwHeightKeyPrefix}, heightBuf); err != nil {
			return 0, err
		}
		return 0, nil
	} else {
		heightBytes, err := db.Get([]byte{RwHeightKeyPrefix})
		if err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint64(heightBytes), nil
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
	blockHeight2ID, _, err := openOrCreate(path.Join(dir, "block_height_to_id"))
	if err != nil {
		return nil, err
	}
	if offsetLen > 8 {
		return nil, errors.New("offsetLen is too large")
	}
	if headerOffsetLen > 8 {
		return nil, errors.New("headerOffsetLen is too large")
	}
	if _, err := initHeight(keyVal); err != nil {
		return nil, err
	}
	return &BlockReadWriter{
		Db:              keyVal,
		blockchain:      blockchain,
		headers:         headers,
		blockHeight2ID:  blockHeight2ID,
		blockchainBuf:   bufio.NewWriter(blockchain),
		txBounds:        make([]byte, offsetLen*2),
		headerBounds:    make([]byte, headerOffsetLen*2),
		blockBounds:     make([]byte, offsetLen*2),
		heightBuf:       make([]byte, 8),
		offsetEnd:       uint64(1<<uint(8*offsetLen) - 1),
		blockchainLen:   blockchainSize,
		headersLen:      headersSize,
		offsetLen:       offsetLen,
		headerOffsetLen: headerOffsetLen,
	}, nil
}

func (rw *BlockReadWriter) SetHeight(height uint64, directly bool) error {
	rwHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(rwHeightBytes, height)
	if directly {
		if err := rw.Db.PutDirectly([]byte{RwHeightKeyPrefix}, rwHeightBytes); err != nil {
			return err
		}
	} else {
		if err := rw.Db.Put([]byte{RwHeightKeyPrefix}, rwHeightBytes); err != nil {
			return err
		}
	}
	return nil
}

func (rw *BlockReadWriter) GetHeight() (uint64, error) {
	rwHeightBytes, err := rw.Db.Get([]byte{RwHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(rwHeightBytes), nil
}

func (rw *BlockReadWriter) SyncFiles() error {
	if err := rw.blockchain.Sync(); err != nil {
		return err
	}
	if err := rw.headers.Sync(); err != nil {
		return err
	}
	if err := rw.blockHeight2ID.Sync(); err != nil {
		return err
	}
	return nil
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
	height, err := rw.GetHeight()
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(rw.blockBounds[rw.offsetLen:], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[rw.headerOffsetLen:], rw.headersLen)
	binary.LittleEndian.PutUint64(rw.heightBuf, height)
	val := append(rw.blockBounds, rw.headerBounds...)
	val = append(val, rw.heightBuf...)
	key := BlockOffsetKey{BlockID: blockID}
	if err := rw.Db.Put(key.Bytes(), val); err != nil {
		return err
	}
	if err := rw.blockchainBuf.Flush(); err != nil {
		return err
	}
	if err := rw.SyncFiles(); err != nil {
		return err
	}
	if err := rw.SetHeight(height+1, false); err != nil {
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
	key := TxOffsetKey{TxID: txID}
	if err := rw.Db.Put(key.Bytes(), rw.txBounds); err != nil {
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
	key := BlockOffsetKey{BlockID: blockID}
	blockInfo, err := rw.Db.Get(key.Bytes())
	if err != nil {
		return 0, err
	}
	height := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:])
	return height, nil
}

func (rw *BlockReadWriter) CurrentHeight() (uint64, error) {
	height, err := rw.GetHeight()
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (rw *BlockReadWriter) ReadTransaction(txID []byte) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	key := TxOffsetKey{TxID: txID}
	txBounds, err := rw.Db.Get(key.Bytes())
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
	key := BlockOffsetKey{BlockID: blockID}
	blockInfo, err := rw.Db.Get(key.Bytes())
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
	key := BlockOffsetKey{BlockID: blockID}
	blockInfo, err := rw.Db.Get(key.Bytes())
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

func (rw *BlockReadWriter) cleanIDs(oldHeight, newBlockchainLen uint64) error {
	newHeight, err := rw.GetHeight()
	if err != nil {
		return err
	}
	// Clean block IDs.
	offset := oldHeight
	blocksIdsToRemove := int(oldHeight - newHeight)
	for i := 0; i < blocksIdsToRemove; i++ {
		readPos := int64((offset - 1) * crypto.SignatureSize)
		idBytes := make([]byte, crypto.SignatureSize)
		if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
			return err
		} else if n != crypto.SignatureSize {
			return errors.New("cleanIDs(): invalid id size")
		}
		blockID, err := toBlockID(idBytes)
		if err != nil {
			return err
		}
		key := BlockOffsetKey{BlockID: blockID}
		if err := rw.Db.Delete(key.Bytes()); err != nil {
			return err
		}
		offset--
	}
	// Clean transaction IDs.
	readPos := newBlockchainLen
	for readPos < rw.blockchainLen {
		txSizeBytes := make([]byte, 4)
		if _, err := rw.blockchain.ReadAt(txSizeBytes, int64(readPos)); err != nil {
			return err
		}
		txSize := binary.BigEndian.Uint32(txSizeBytes)
		readPos += 4
		txBytes := make([]byte, txSize)
		if _, err := rw.blockchain.ReadAt(txBytes, int64(readPos)); err != nil {
			return err
		}
		readPos += uint64(txSize)
		tx, err := proto.BytesToTransaction(txBytes)
		if err != nil {
			return err
		}
		key := TxOffsetKey{TxID: tx.GetID()}
		if err := rw.Db.Delete(key.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (rw *BlockReadWriter) Rollback(removalEdge crypto.Signature, cleanIDs bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	key := BlockOffsetKey{BlockID: removalEdge}
	blockInfo, err := rw.Db.Get(key.Bytes())
	if err != nil {
		return err
	}
	newHeight := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:]) + 1
	// Set new height first of all.
	oldHeight, err := rw.GetHeight()
	if err != nil {
		return err
	}
	if oldHeight < newHeight {
		return errors.New("new height is greater than current height")
	}
	if err := rw.SetHeight(newHeight, true); err != nil {
		return err
	}
	blockBounds := blockInfo[:rw.offsetLen*2]
	blockEnd := binary.LittleEndian.Uint64(blockBounds[rw.offsetLen:])
	if cleanIDs {
		// Clean IDs of blocks and transactions.
		if err := rw.cleanIDs(oldHeight, blockEnd); err != nil {
			return err
		}
	}
	// Remove transactions.
	if err := rw.blockchain.Truncate(int64(blockEnd)); err != nil {
		return err
	}
	if _, err := rw.blockchain.Seek(int64(blockEnd), 0); err != nil {
		return err
	}
	// Remove headers.
	headerBounds := blockInfo[rw.offsetLen*2 : len(blockInfo)-8]
	headerEnd := binary.LittleEndian.Uint64(headerBounds[rw.headerOffsetLen:])
	if err := rw.headers.Truncate(int64(headerEnd)); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(int64(headerEnd), 0); err != nil {
		return err
	}
	// Remove blockIDs from blockHeight2ID file.
	newOffset := int64(newHeight * crypto.SignatureSize)
	if err := rw.blockHeight2ID.Truncate(newOffset); err != nil {
		return err
	}
	if _, err := rw.blockHeight2ID.Seek(newOffset, 0); err != nil {
		return err
	}
	// Decrease counters.
	rw.blockchainLen = blockEnd
	rw.headersLen = headerEnd
	// Reset buffers.
	rw.blockchainBuf.Reset(rw.blockchain)
	return nil
}

func (rw *BlockReadWriter) Close() error {
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
