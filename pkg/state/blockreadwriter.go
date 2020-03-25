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
	offset uint64
}

type recentDataStorage struct {
	id2Pos map[string]bytesInfo
	stor   [][]byte
}

func newRecentDataStorage() *recentDataStorage {
	return &recentDataStorage{id2Pos: make(map[string]bytesInfo)}
}

func (r *recentDataStorage) appendBytes(id, data []byte) error {
	r.id2Pos[string(id)] = bytesInfo{pos: len(r.stor)}
	r.stor = append(r.stor, data)
	return nil
}

func (r *recentDataStorage) appendBytesWithInfo(id, data []byte, height, offset uint64) error {
	r.id2Pos[string(id)] = bytesInfo{pos: len(r.stor), height: height, offset: offset}
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

func (r *recentDataStorage) offsetById(id []byte) (uint64, error) {
	info, ok := r.id2Pos[string(id)]
	if !ok {
		return 0, errNotFound
	}
	return info.offset, nil
}

func (r *recentDataStorage) writeDataToBuffer(buf *bufio.Writer) error {
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

type protobufInfo struct {
	protobufTxStart      uint64
	protobufHeadersStart uint64
	protobufAfterHeight  uint64
}

func (info *protobufInfo) marshalBinary() []byte {
	res := make([]byte, 24)
	binary.BigEndian.PutUint64(res[:8], info.protobufTxStart)
	binary.BigEndian.PutUint64(res[8:16], info.protobufHeadersStart)
	binary.BigEndian.PutUint64(res[16:24], info.protobufAfterHeight)
	return res
}

func (info *protobufInfo) unmarshalBinary(data []byte) error {
	if len(data) != 24 {
		return errInvalidDataSize
	}
	info.protobufTxStart = binary.BigEndian.Uint64(data[:8])
	info.protobufHeadersStart = binary.BigEndian.Uint64(data[8:16])
	info.protobufAfterHeight = binary.BigEndian.Uint64(data[16:24])
	return nil
}

type blockReadWriter struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch

	scheme proto.Scheme

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
	height2IDCache map[uint64]proto.BlockID
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

	// Protobuf-related stuff.
	protobufActivated                     bool
	protobufTxStart, protobufHeadersStart uint64
	protobufAfterHeight                   uint64

	mtx sync.RWMutex
}

func openOrCreateForAppending(path string) (*os.File, uint64, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, 0, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	size := stat.Size()
	if _, err := file.Seek(size, 0); err != nil {
		return nil, 0, err
	}
	return file, uint64(size), nil
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
	scheme proto.Scheme,
) (*blockReadWriter, error) {
	blockchain, blockchainSize, err := openOrCreateForAppending(path.Join(dir, "blockchain"))
	if err != nil {
		return nil, err
	}
	headers, headersSize, err := openOrCreateForAppending(path.Join(dir, "headers"))
	if err != nil {
		return nil, err
	}
	blockHeight2ID, _, err := openOrCreateForAppending(path.Join(dir, "block_height_to_id"))
	if err != nil {
		return nil, err
	}
	if offsetLen != 8 {
		// TODO: support different offset lengths.
		return nil, errors.New("only offsetLen 8 is currently supported")
	}
	if headerOffsetLen != 8 {
		// TODO: support different offset lengths.
		return nil, errors.New("only headerOffsetLen 8 is currently supported")
	}
	height, err := initHeight(db)
	if err != nil {
		return nil, err
	}
	rw := &blockReadWriter{
		db:                db,
		dbBatch:           dbBatch,
		scheme:            scheme,
		blockchain:        blockchain,
		headers:           headers,
		blockHeight2ID:    blockHeight2ID,
		blockchainBuf:     bufio.NewWriter(blockchain),
		headersBuf:        bufio.NewWriter(headers),
		blockHeight2IDBuf: bufio.NewWriter(blockHeight2ID),
		rtx:               newRecentDataStorage(),
		rheaders:          newRecentDataStorage(),
		height2IDCache:    make(map[uint64]proto.BlockID),
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
	}
	if err := rw.loadProtobufInfo(); err != nil {
		return nil, err
	}
	return rw, nil
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

func (rw *blockReadWriter) startBlock(blockID proto.BlockID) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	rw.addingBlock = true
	if _, err := rw.blockHeight2IDBuf.Write(blockID.Bytes()); err != nil {
		return err
	}
	rw.height2IDCache[rw.height+1] = blockID
	binary.LittleEndian.PutUint64(rw.blockBounds[:rw.offsetLen], rw.blockchainLen)
	binary.LittleEndian.PutUint64(rw.headerBounds[:rw.headerOffsetLen], rw.headersLen)
	return nil
}

func (rw *blockReadWriter) finishBlock(blockID proto.BlockID) error {
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

func (rw *blockReadWriter) writeTransaction(tx proto.Transaction) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	txID, err := tx.GetID(rw.scheme)
	if err != nil {
		return err
	}
	var txBytes []byte
	if rw.protobufActivated {
		txBytes, err = tx.MarshalSignedToProtobuf(rw.scheme)
		if err != nil {
			return err
		}
	} else {
		txBytes, err = tx.MarshalBinary()
		if err != nil {
			return err
		}
	}
	// Append tx size at the beginning.
	txSize := uint32(len(txBytes))
	txSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(txSizeBytes, txSize)
	txBytesTotal := make([]byte, 4+txSize)
	copy(txBytesTotal[:4], txSizeBytes)
	copy(txBytesTotal[4:], txBytes)
	// Save tx to local storage.
	if err := rw.rtx.appendBytesWithInfo(txID, txBytesTotal, rw.height+1, rw.blockchainLen); err != nil {
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

func (rw *blockReadWriter) writeBlockHeader(header *proto.BlockHeader) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	blockID := header.BlockID()
	var err error
	var headerBytes []byte
	if rw.protobufActivated {
		protoBytes, err := header.MarshalHeaderToProtobuf(rw.scheme)
		if err != nil {
			return err
		}
		// Put addl info that is missing in Protobuf.
		headerBytes = make([]byte, 8+len(protoBytes))
		binary.BigEndian.PutUint32(headerBytes[:4], uint32(header.TransactionCount))
		binary.BigEndian.PutUint32(headerBytes[4:8], uint32(header.TransactionBlockLength))
		copy(headerBytes[8:], protoBytes)
	} else {
		headerBytes, err = header.MarshalHeaderToBinary()
		if err != nil {
			return err
		}
	}
	if err := rw.rheaders.appendBytes(blockID.Bytes(), headerBytes); err != nil {
		return err
	}
	rw.headersLen += uint64(len(headerBytes))
	if rw.headersLen > rw.offsetEnd {
		return errors.Errorf("offsetLen is not enough for this offset: %d > %d", rw.headersLen, rw.offsetEnd)
	}
	return nil
}

func (rw *blockReadWriter) newestBlockIDByHeight(height uint64) (proto.BlockID, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	// For blockReadWriter, heights start from 0.
	if id, ok := rw.height2IDCache[height]; ok {
		return id, nil
	}
	return rw.blockIDByHeightImpl(height)
}

func (rw *blockReadWriter) blockIDByHeight(height uint64) (proto.BlockID, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.blockIDByHeightImpl(height)
}

func (rw *blockReadWriter) heightToIDSize(height uint64) int {
	if !rw.protobufActivated {
		return crypto.SignatureSize
	}
	if height >= rw.protobufAfterHeight {
		return crypto.DigestSize
	}
	return crypto.SignatureSize
}

func (rw *blockReadWriter) heightToIDOffset(height uint64) uint64 {
	if !rw.protobufActivated {
		return height * crypto.SignatureSize
	}
	if height > rw.protobufAfterHeight {
		offsetBeforeProtobuf := rw.protobufAfterHeight * crypto.SignatureSize
		blocksAfterProtobuf := height - rw.protobufAfterHeight
		offsetAfterProtobuf := blocksAfterProtobuf * crypto.DigestSize
		return offsetBeforeProtobuf + offsetAfterProtobuf
	}
	return height * crypto.SignatureSize
}

func (rw *blockReadWriter) blockIDByHeightImpl(height uint64) (proto.BlockID, error) {
	// For blockReadWriter, heights start from 0.
	height -= 1
	idBytes := make([]byte, rw.heightToIDSize(height))
	readPos := int64(rw.heightToIDOffset(height))
	if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
		return proto.BlockID{}, err
	} else if n != len(idBytes) {
		return proto.BlockID{}, errors.New("blockIDByHeight(): invalid id size")
	}
	return proto.NewBlockIDFromBytes(idBytes)
}

func (rw *blockReadWriter) heightFromBlockInfo(blockInfo []byte) (uint64, error) {
	if len(blockInfo) < 8 {
		return 0, errInvalidDataSize
	}
	height := binary.LittleEndian.Uint64(blockInfo[len(blockInfo)-8:])
	return height, nil
}

func (rw *blockReadWriter) newestHeightByBlockID(blockID proto.BlockID) (uint64, error) {
	key := blockOffsetKey{blockID: blockID}
	blockInfo, ok := rw.blockInfo[key]
	if ok {
		return rw.heightFromBlockInfo(blockInfo)
	}
	return rw.heightByBlockID(blockID)
}

func (rw *blockReadWriter) heightByBlockID(blockID proto.BlockID) (uint64, error) {
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
	key := txHeightKey{txID: txID}
	heightBytes, err := rw.db.Get(key.bytes())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(heightBytes), nil
}

func (rw *blockReadWriter) transactionOffsetByID(txID []byte) (uint64, error) {
	key := txOffsetKey{txID: txID}
	offsetBytes, err := rw.db.Get(key.bytes())
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(offsetBytes), nil
}

func (rw *blockReadWriter) newestTransactionOffsetByID(txID []byte) (uint64, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	offset, err := rw.rtx.offsetById(txID)
	if err == nil {
		return offset, nil
	}
	return rw.transactionOffsetByID(txID)
}

func (rw *blockReadWriter) readTransactionSize(offset uint64) (uint32, error) {
	sizeBytes := make([]byte, 4)
	n, err := rw.blockchain.ReadAt(sizeBytes, int64(offset))
	if err != nil {
		return 0, err
	} else if n != 4 {
		return 0, errors.New("ReadAt did not read 4 bytes")
	}
	return binary.BigEndian.Uint32(sizeBytes), nil
}

func (rw *blockReadWriter) readNewestTransaction(txID []byte) (proto.Transaction, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	txBytes, err := rw.rtx.bytesById(txID)
	if err != nil {
		return rw.readTransactionImpl(txID)
	}
	if len(txBytes) < 4 {
		return nil, errors.New("invalid tx size")
	}
	return rw.txFromBytes(txBytes[4:], rw.protobufActivated)
}

func (rw *blockReadWriter) readTransaction(txID []byte) (proto.Transaction, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readTransactionImpl(txID)
}

func (rw *blockReadWriter) readTransactionImpl(txID []byte) (proto.Transaction, error) {
	offset, err := rw.transactionOffsetByID(txID)
	if err != nil {
		return nil, err
	}
	return rw.readTransactionByOffsetImpl(offset)
}

func (rw *blockReadWriter) readTransactionByOffset(offset uint64) (proto.Transaction, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readTransactionByOffsetImpl(offset)
}

func (rw *blockReadWriter) readTransactionByOffsetImpl(offset uint64) (proto.Transaction, error) {
	txSize, err := rw.readTransactionSize(offset)
	if err != nil {
		return nil, err
	}
	// First 4 bytes are tx size, actual tx starts at `offset + 4`.
	txStart := offset + 4
	txEnd := txStart + uint64(txSize)
	return rw.txByBounds(txStart, txEnd)
}

func (rw *blockReadWriter) readNewestBlockHeader(blockID proto.BlockID) (*proto.BlockHeader, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	headerBytes, err := rw.rheaders.bytesById(blockID.Bytes())
	if err != nil {
		return rw.readBlockHeaderImpl(blockID)
	}
	return rw.headerFromBytes(headerBytes, rw.protobufActivated)
}

func (rw *blockReadWriter) readBlockHeader(blockID proto.BlockID) (*proto.BlockHeader, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readBlockHeaderImpl(blockID)
}

func (rw *blockReadWriter) readBlockHeaderImpl(blockID proto.BlockID) (*proto.BlockHeader, error) {
	key := blockOffsetKey{blockID: blockID}
	blockInfo, err := rw.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	headerBounds := blockInfo[rw.offsetLen*2 : len(blockInfo)-8]
	headerStart := binary.LittleEndian.Uint64(headerBounds[:rw.headerOffsetLen])
	headerEnd := binary.LittleEndian.Uint64(headerBounds[rw.headerOffsetLen:])
	return rw.headerByBounds(headerStart, headerEnd)
}

func (rw *blockReadWriter) readBlock(blockID proto.BlockID) (*proto.Block, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	header, err := rw.readBlockHeaderImpl(blockID)
	if err != nil {
		return nil, err
	}
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
	var res proto.Transactions
	if rw.isProtobufTxOffset(blockStart) {
		if err := res.UnmarshalFromProtobuf(blockBytes); err != nil {
			return nil, err
		}
	} else {
		res, err = proto.NewTransactionsFromBytes(blockBytes, header.TransactionCount, rw.scheme)
		if err != nil {
			return nil, err
		}
	}
	return &proto.Block{
		BlockHeader:  *header,
		Transactions: res,
	}, nil
}

func (rw *blockReadWriter) cleanIDs(oldHeight, newBlockchainLen uint64) error {
	newHeight, err := rw.getHeight()
	if err != nil {
		return err
	}
	// Clean block IDs.
	for h := oldHeight - 1; h >= newHeight; h-- {
		readPos := int64(rw.heightToIDOffset(h))
		idBytes := make([]byte, rw.heightToIDSize(h))
		if n, err := rw.blockHeight2ID.ReadAt(idBytes, readPos); err != nil {
			return err
		} else if n != len(idBytes) {
			return errors.New("cleanIDs(): invalid id size")
		}
		blockID, err := proto.NewBlockIDFromBytes(idBytes)
		if err != nil {
			return err
		}
		key := blockOffsetKey{blockID: blockID}
		if err := rw.db.Delete(key.bytes()); err != nil {
			return err
		}
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
		tx, err := rw.txByBounds(readPos, readPos+uint64(txSize))
		if err != nil {
			return err
		}
		readPos += uint64(txSize)
		txID, err := tx.GetID(rw.scheme)
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
	// Remove protobuf info (protobuf encoding never starts from 0).
	if err := rw.db.Delete([]byte{rwProtobufInfoKeyPrefix}); err != nil {
		return err
	}
	// Decrease counters.
	rw.height = 0
	rw.blockchainLen = 0
	rw.headersLen = 0
	// Protobuf.
	rw.protobufActivated = false
	rw.protobufTxStart = 0
	rw.protobufHeadersStart = 0
	rw.protobufAfterHeight = 0
	// Reset buffers.
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.headersBuf.Reset(rw.headers)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	return nil
}

func (rw *blockReadWriter) rollback(removalEdge proto.BlockID, cleanIDs bool) error {
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
	newOffset := int64(rw.heightToIDOffset(newHeight))
	if err := rw.blockHeight2ID.Truncate(newOffset); err != nil {
		return err
	}
	if _, err := rw.blockHeight2ID.Seek(newOffset, 0); err != nil {
		return err
	}
	if blockEnd < rw.protobufTxStart {
		// Protobuf.
		if err := rw.db.Delete([]byte{rwProtobufInfoKeyPrefix}); err != nil {
			return err
		}
		rw.protobufActivated = false
		rw.protobufTxStart = 0
		rw.protobufHeadersStart = 0
		rw.protobufAfterHeight = 0
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
	rw.height2IDCache = make(map[uint64]proto.BlockID)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	rw.blockInfo = make(map[blockOffsetKey][]byte)
}

func (rw *blockReadWriter) flush() error {
	if err := rw.rtx.writeDataToBuffer(rw.blockchainBuf); err != nil {
		return err
	}
	if err := rw.blockchainBuf.Flush(); err != nil {
		return err
	}
	if err := rw.rheaders.writeDataToBuffer(rw.headersBuf); err != nil {
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

func (rw *blockReadWriter) loadProtobufInfo() error {
	key := []byte{rwProtobufInfoKeyPrefix}
	has, err := rw.db.Has(key)
	if err != nil {
		return err
	}
	if !has {
		// Nothing found, means protobuf is not yet activated.
		rw.protobufActivated = false
		return nil
	}
	infoBytes, err := rw.db.Get(key)
	if err != nil {
		return err
	}
	var info protobufInfo
	if err := info.unmarshalBinary(infoBytes); err != nil {
		return err
	}
	if info.protobufTxStart > rw.blockchainLen || info.protobufHeadersStart > rw.headersLen || info.protobufAfterHeight > rw.height {
		// Might happen if rollback does not complete correctly.
		if err := rw.db.Delete([]byte{rwProtobufInfoKeyPrefix}); err != nil {
			return err
		}
		rw.protobufActivated = false
		return nil
	}
	rw.protobufActivated = true
	rw.protobufTxStart = info.protobufTxStart
	rw.protobufHeadersStart = info.protobufHeadersStart
	rw.protobufAfterHeight = info.protobufAfterHeight
	return nil
}

func (rw *blockReadWriter) storeProtobufInfo(info *protobufInfo) {
	infoBytes := info.marshalBinary()
	key := []byte{rwProtobufInfoKeyPrefix}
	rw.dbBatch.Put(key, infoBytes)
}

func (rw *blockReadWriter) setProtobufActivated() {
	if rw.protobufActivated {
		// Already activated.
		return
	}
	rw.protobufActivated = true
	rw.protobufHeadersStart = rw.headersLen
	rw.protobufTxStart = rw.blockchainLen
	rw.protobufAfterHeight = rw.height
	info := &protobufInfo{
		protobufHeadersStart: rw.protobufHeadersStart,
		protobufTxStart:      rw.protobufTxStart,
		protobufAfterHeight:  rw.protobufAfterHeight,
	}
	rw.storeProtobufInfo(info)
}

func (rw *blockReadWriter) isProtobufTxOffset(offset uint64) bool {
	if !rw.protobufActivated {
		return false
	}
	return offset >= rw.protobufTxStart
}

func (rw *blockReadWriter) isProtobufHeaderOffset(offset uint64) bool {
	if !rw.protobufActivated {
		return false
	}
	return offset >= rw.protobufHeadersStart
}

func (rw *blockReadWriter) headerFromBytes(headerBytes []byte, protobuf bool) (*proto.BlockHeader, error) {
	if protobuf {
		if len(headerBytes) < 8 {
			return nil, errInvalidDataSize
		}
		txCount := binary.BigEndian.Uint32(headerBytes[:4])
		txLen := binary.BigEndian.Uint32(headerBytes[4:8])
		var b proto.Block
		if err := b.UnmarshalFromProtobuf(headerBytes[8:]); err != nil {
			return nil, err
		}
		b.TransactionCount = int(txCount)
		b.TransactionBlockLength = txLen
		return &b.BlockHeader, nil
	}
	var header proto.BlockHeader
	if err := header.UnmarshalHeaderFromBinary(headerBytes, rw.scheme); err != nil {
		return nil, err
	}
	return &header, nil
}

func (rw *blockReadWriter) headerByBounds(start, end uint64) (*proto.BlockHeader, error) {
	if end <= start {
		return nil, errors.New("invalid bounds")
	}
	headerBytes := make([]byte, end-start)
	n, err := rw.headers.ReadAt(headerBytes, int64(start))
	if err != nil {
		return nil, err
	} else if n != len(headerBytes) {
		return nil, errors.New("did not read the whole header")
	}
	protobuf := rw.isProtobufHeaderOffset(start)
	return rw.headerFromBytes(headerBytes, protobuf)
}

func (rw *blockReadWriter) txFromBytes(txBytes []byte, protobuf bool) (proto.Transaction, error) {
	if protobuf {
		return proto.SignedTxFromProtobuf(txBytes)
	}
	return proto.BytesToTransaction(txBytes, rw.scheme)
}

func (rw *blockReadWriter) txByBounds(start, end uint64) (proto.Transaction, error) {
	if end <= start {
		return nil, errors.New("invalid bounds")
	}
	txBytes := make([]byte, end-start)
	n, err := rw.blockchain.ReadAt(txBytes, int64(start))
	if err != nil {
		return nil, err
	} else if n != len(txBytes) {
		return nil, errors.New("did not read the whole tx")
	}
	protobuf := rw.isProtobufTxOffset(start)
	return rw.txFromBytes(txBytes, protobuf)
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
