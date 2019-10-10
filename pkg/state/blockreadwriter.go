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

type bytesInfo struct {
	pos    int
	height uint64
}

type recentDataStorage struct {
	id2Pos map[string]bytesInfo
	stor   [][]byte
}

func newRecentDataStorage() *recentDataStorage {
	return &recentDataStorage{id2Pos: make(map[string]bytesInfo)}
}

func (r *recentDataStorage) appendBytes(id, data []byte, height uint64) error {
	r.id2Pos[string(id)] = bytesInfo{len(r.stor), height}
	r.stor = append(r.stor, data)
	return nil
}

func (r *recentDataStorage) bytesById(id []byte) ([]byte, error) {
	info, ok := r.id2Pos[string(id)]
	if !ok {
		return nil, errNotFound
	}
	if info.pos < 0 || info.pos >= len(r.stor) {
		return nil, errors.New("invalid pos")
	}
	return r.stor[info.pos], nil
}

func (r *recentDataStorage) heightById(id []byte) (uint64, error) {
	info, ok := r.id2Pos[string(id)]
	if !ok {
		return 0, errNotFound
	}
	return info.height, nil
}

func (r *recentDataStorage) writeToBuffer(buf *bufio.Writer) error {
	for _, data := range r.stor {
		if _, err := buf.Write(data); err != nil {
			return err
		}
	}
	r.reset()
	return nil
}

func (r *recentDataStorage) reset() {
	r.id2Pos = make(map[string]bytesInfo)
	r.stor = nil
}

type blockReadWriter struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch

	// Series of transactions.
	blockchain *os.File
	// Series of BlockHeader.
	headers *os.File
	// Height is used as index for block IDs.
	blockHeight2ID *os.File

	blockchainBuf     *bufio.Writer
	headersBuf        *bufio.Writer
	blockHeight2IDBuf *bufio.Writer

	// Storages for recent data that is not in persistent storage yet.
	rtx            *recentDataStorage
	rheaders       *recentDataStorage
	height2IDCache map[uint64]crypto.Signature
	blockInfo      map[blockOffsetKey][]byte

	blockBounds  []byte
	txStarts     []byte
	headerBounds []byte
	heightBuf    []byte

	// offsetEnd is common for headers and the blockchain, since the limit for any offset length is 8 bytes.
	offsetEnd                 uint64
	blockchainLen, headersLen uint64

	offsetLen, headerOffsetLen int
	height                     uint64

	addingBlock bool

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
	has, err := db.Has([]byte{rwHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	if !has {
		heightBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.Put([]byte{rwHeightKeyPrefix}, heightBuf); err != nil {
			return 0, err
		}
		return 0, nil
	} else {
		heightBytes, err := db.Get([]byte{rwHeightKeyPrefix})
		if err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint64(heightBytes), nil
	}
}

func newBlockReadWriter(
	dir string,
	offsetLen int,
	headerOffsetLen int,
	db keyvalue.KeyValue,
	dbBatch keyvalue.Batch,
) (*blockReadWriter, error) {
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
	height, err := initHeight(db)
	if err != nil {
		return nil, err
	}
	return &blockReadWriter{
		db:                db,
		dbBatch:           dbBatch,
		blockchain:        blockchain,
		headers:           headers,
		blockHeight2ID:    blockHeight2ID,
		blockchainBuf:     bufio.NewWriter(blockchain),
		headersBuf:        bufio.NewWriter(headers),
		blockHeight2IDBuf: bufio.NewWriter(blockHeight2ID),
		rtx:               newRecentDataStorage(),
		rheaders:          newRecentDataStorage(),
		height2IDCache:    make(map[uint64]crypto.Signature),
		blockInfo:         make(map[blockOffsetKey][]byte),
		txStarts:          make([]byte, offsetLen),
		headerBounds:      make([]byte, headerOffsetLen*2),
		blockBounds:       make([]byte, offsetLen*2),
		heightBuf:         make([]byte, 8),
		offsetEnd:         uint64(1<<uint(8*offsetLen) - 1),
		blockchainLen:     blockchainSize,
		headersLen:        headersSize,
		offsetLen:         offsetLen,
		headerOffsetLen:   headerOffsetLen,
		height:            height,
	}, nil
}

func (rw *blockReadWriter) setHeight(height uint64, directly bool) error {
	rwHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(rwHeightBytes, height)
	if directly {
		if err := rw.db.Put([]byte{rwHeightKeyPrefix}, rwHeightBytes); err != nil {
			return err
		}
	} else {
		rw.dbBatch.Put([]byte{rwHeightKeyPrefix}, rwHeightBytes)
	}
	return nil
}

func (rw *blockReadWriter) getHeight() (uint64, error) {
	rwHeightBytes, err := rw.db.Get([]byte{rwHeightKeyPrefix})
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(rwHeightBytes), nil
}

func (rw *blockReadWriter) syncFiles() error {
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

func (rw *blockReadWriter) startBlock(blockID crypto.Signature) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	rw.addingBlock = true
	if _, err := rw.blockHeight2IDBuf.Write(blockID[:]); err != nil {
		return err
	}
	rw.height2IDCache[rw.height+1] = blockID
	binary.LittleEndian.PutUint64(rw.blockBounds[:rw.offsetLen], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[:rw.headerOffsetLen], rw.headersLen)
	return nil
}

func (rw *blockReadWriter) finishBlock(blockID crypto.Signature) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	rw.addingBlock = false
	binary.LittleEndian.PutUint64(rw.blockBounds[rw.offsetLen:], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[rw.headerOffsetLen:], rw.headersLen)
	binary.LittleEndian.PutUint64(rw.heightBuf, rw.height+1)
	val := append(rw.blockBounds, rw.headerBounds...)
	val = append(val, rw.heightBuf...)
	key := blockOffsetKey{blockID: blockID}
	rw.blockInfo[key] = val
	rw.height++
	return nil
}

func (rw *blockReadWriter) writeTransaction(txID []byte, tx []byte) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	// Append tx size at the beginning.
	txSize := uint32(len(tx))
	txSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(txSizeBytes, txSize)
	txBytesTotal := make([]byte, 4+txSize)
	copy(txBytesTotal[:4], txSizeBytes)
	copy(txBytesTotal[4:], tx)
	// Save tx to local storage.
	if err := rw.rtx.appendBytes(txID, txBytesTotal, rw.height+1); err != nil {
		return err
	}
	// Write tx start pos to DB batch.
	binary.LittleEndian.PutUint64(rw.txStarts[:rw.offsetLen], rw.blockchainLen)
	rw.blockchainLen += uint64(len(txBytesTotal))
	if rw.blockchainLen > rw.offsetEnd {
		return errors.Errorf("offsetLen is not enough for this offset: %d > %d", rw.blockchainLen, rw.offsetEnd)
	}
	key := txOffsetKey{txID: txID}
	rw.dbBatch.Put(key.bytes(), rw.txStarts)
	// Write tx height by ID.
	heightKey := txHeightKey{txID: txID}
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, rw.height+1)
	rw.dbBatch.Put(heightKey.bytes(), heightBytes)
	return nil
}

func (rw *blockReadWriter) writeBlockHeader(blockID crypto.Signature, header []byte) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	if err := rw.rheaders.appendBytes(blockID[:], header, rw.height+1); err != nil {
		return err
	}
	rw.headersLen += uint64(len(header))
	if rw.headersLen > rw.offsetEnd {
		return errors.Errorf("offsetLen is not enough for this offset: %d > %d", rw.headersLen, rw.offsetEnd)
	}
	return nil
}

func (rw *blockReadWriter) newestBlockIDByHeight(height uint64) (crypto.Signature, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	// For blockReadWriter, heights start from 0.
	if id, ok := rw.height2IDCache[height]; ok {
		return id, nil
	}
	return rw.blockIDByHeight(height)
}

func (rw *blockReadWriter) blockIDByHeight(height uint64) (crypto.Signature, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	// For blockReadWriter, heights start from 0.
	height -= 1
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

func (rw *blockReadWriter) heightFromBlockInfo(blockInfo []byte) (uint64, error) {
	if len(blockInfo) < 8 {
		return 0, errors.New("invalid data size")
	}
	height := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:])
	return height, nil
}

func (rw *blockReadWriter) newestHeightByBlockID(blockID crypto.Signature) (uint64, error) {
	key := blockOffsetKey{blockID: blockID}
	blockInfo, ok := rw.blockInfo[key]
	if ok {
		return rw.heightFromBlockInfo(blockInfo)
	}
	return rw.heightByBlockID(blockID)
}

func (rw *blockReadWriter) heightByBlockID(blockID crypto.Signature) (uint64, error) {
	key := blockOffsetKey{blockID: blockID}
	blockInfo, err := rw.db.Get(key.bytes())
	if err != nil {
		return 0, err
	}
	return rw.heightFromBlockInfo(blockInfo)
}

func (rw *blockReadWriter) addingBlockHeight() uint64 {
	if rw.addingBlock {
		return rw.height + 1
	}
	return rw.height
}

func (rw *blockReadWriter) recentHeight() uint64 {
	return rw.height
}

func (rw *blockReadWriter) currentHeight() (uint64, error) {
	height, err := rw.getHeight()
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (rw *blockReadWriter) newestTransactionHeightByID(txID []byte) (uint64, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	height, err := rw.rtx.heightById(txID)
	if err == nil {
		return height, nil
	}
	return rw.transactionHeightByID(txID)
}

func (rw *blockReadWriter) transactionHeightByID(txID []byte) (uint64, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	key := txHeightKey{txID: txID}
	heightBytes, err := rw.db.Get(key.bytes())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(heightBytes), nil
}

func (rw *blockReadWriter) readTransactionSize(txID []byte) (uint32, error) {
	key := txOffsetKey{txID: txID}
	txStarts, err := rw.db.Get(key.bytes())
	if err != nil {
		return 0, err
	}
	txStart := binary.LittleEndian.Uint64(txStarts[:rw.offsetLen])
	sizeBytes := make([]byte, 4)
	n, err := rw.blockchain.ReadAt(sizeBytes, int64(txStart))
	if err != nil {
		return 0, err
	} else if n != 4 {
		return 0, errors.New("ReadAt did not read 4 bytes")
	}
	return binary.BigEndian.Uint32(sizeBytes), nil
}

func (rw *blockReadWriter) readNewestTransaction(txID []byte) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	tx, err := rw.rtx.bytesById(txID)
	if err != nil {
		return rw.readTransaction(txID)
	}
	if len(tx) < 4 {
		return nil, errors.New("invalid tx size")
	}
	return tx[4:], nil
}

func (rw *blockReadWriter) readTransaction(txID []byte) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	txSize, err := rw.readTransactionSize(txID)
	if err != nil {
		return nil, err
	}
	key := txOffsetKey{txID: txID}
	txStarts, err := rw.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	txStart := binary.LittleEndian.Uint64(txStarts[:rw.offsetLen])
	// First 4 bytes are tx size, actual tx starts at `txStart + 4`.
	txRealStart := txStart + 4
	txBytes := make([]byte, txSize)
	n, err := rw.blockchain.ReadAt(txBytes, int64(txRealStart))
	if err != nil {
		return nil, err
	} else if n != len(txBytes) {
		return nil, errors.New("ReadAt did not read the whole tx")
	}
	if len(txBytes) < 4 {
		return nil, errors.New("Tx size is less than 4")
	}
	return txBytes, nil
}

func (rw *blockReadWriter) readNewestBlockHeader(blockID crypto.Signature) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	header, err := rw.rheaders.bytesById(blockID[:])
	if err != nil {
		return rw.readBlockHeader(blockID)
	}
	return header, nil
}

func (rw *blockReadWriter) readBlockHeader(blockID crypto.Signature) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	key := blockOffsetKey{blockID: blockID}
	blockInfo, err := rw.db.Get(key.bytes())
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

func (rw *blockReadWriter) readTransactionsBlock(blockID crypto.Signature) ([]byte, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	key := blockOffsetKey{blockID: blockID}
	blockInfo, err := rw.db.Get(key.bytes())
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

func (rw *blockReadWriter) cleanIDs(oldHeight, newBlockchainLen uint64) error {
	newHeight, err := rw.getHeight()
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
		blockID, err := crypto.NewSignatureFromBytes(idBytes)
		if err != nil {
			return err
		}
		key := blockOffsetKey{blockID: blockID}
		if err := rw.db.Delete(key.bytes()); err != nil {
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
		txID, err := tx.GetID()
		if err != nil {
			return err
		}
		key := txOffsetKey{txID: txID}
		if err := rw.db.Delete(key.bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (rw *blockReadWriter) removeEverything(cleanIDs bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	// Set new height first of all.
	if err := rw.setHeight(0, true); err != nil {
		return err
	}
	oldHeight, err := rw.getHeight()
	if err != nil {
		return err
	}
	if cleanIDs {
		// Clean IDs of blocks and transactions.
		if err := rw.cleanIDs(oldHeight, 0); err != nil {
			return err
		}
	}
	// Remove transactions.
	if err := rw.blockchain.Truncate(0); err != nil {
		return err
	}
	if _, err := rw.blockchain.Seek(0, 0); err != nil {
		return err
	}
	// Remove headers.
	if err := rw.headers.Truncate(0); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(0, 0); err != nil {
		return err
	}
	// Remove blockIDs from blockHeight2ID file.
	if err := rw.blockHeight2ID.Truncate(0); err != nil {
		return err
	}
	if _, err := rw.blockHeight2ID.Seek(0, 0); err != nil {
		return err
	}
	// Decrease counters.
	rw.height = 0
	rw.blockchainLen = 0
	rw.headersLen = 0
	// Reset buffers.
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.headersBuf.Reset(rw.headers)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	return nil
}

func (rw *blockReadWriter) rollback(removalEdge crypto.Signature, cleanIDs bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	key := blockOffsetKey{blockID: removalEdge}
	blockInfo, err := rw.db.Get(key.bytes())
	if err != nil {
		return err
	}
	newHeight := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:])
	// Set new height first of all.
	oldHeight, err := rw.getHeight()
	if err != nil {
		return err
	}
	if oldHeight < newHeight {
		return errors.New("new height is greater than current height")
	}
	if err := rw.setHeight(newHeight, true); err != nil {
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
	rw.height = newHeight
	rw.blockchainLen = blockEnd
	rw.headersLen = headerEnd
	// Reset buffers.
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.headersBuf.Reset(rw.headers)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	return nil
}

func (rw *blockReadWriter) reset() {
	rw.rtx.reset()
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.rheaders.reset()
	rw.headersBuf.Reset(rw.headers)
	rw.height2IDCache = make(map[uint64]crypto.Signature)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	rw.blockInfo = make(map[blockOffsetKey][]byte)
}

func (rw *blockReadWriter) flush() error {
	if err := rw.rtx.writeToBuffer(rw.blockchainBuf); err != nil {
		return err
	}
	if err := rw.blockchainBuf.Flush(); err != nil {
		return err
	}
	if err := rw.rheaders.writeToBuffer(rw.headersBuf); err != nil {
		return err
	}
	if err := rw.headersBuf.Flush(); err != nil {
		return err
	}
	if err := rw.blockHeight2IDBuf.Flush(); err != nil {
		return err
	}
	if err := rw.syncFiles(); err != nil {
		return err
	}
	for key, info := range rw.blockInfo {
		rw.dbBatch.Put(key.bytes(), info)
	}
	if err := rw.setHeight(rw.height, false); err != nil {
		return err
	}
	return nil
}

func (rw *blockReadWriter) close() error {
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
