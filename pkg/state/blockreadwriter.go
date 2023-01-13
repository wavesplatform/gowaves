package state

import (
	"bufio"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	txInfoSize    = 8 + 8 + 1
	blockMetaSize = 8 * 5
)

type txInfo struct {
	height uint64
	offset uint64
	failed bool
}

func (i *txInfo) bytes() []byte {
	buf := make([]byte, txInfoSize)
	binary.BigEndian.PutUint64(buf[:8], i.height)
	binary.BigEndian.PutUint64(buf[8:16], i.offset)
	if i.failed {
		buf[16] = 1
	}
	return buf
}

func (i *txInfo) unmarshal(data []byte) error {
	if len(data) < txInfoSize {
		return errInvalidDataSize
	}
	i.height = binary.BigEndian.Uint64(data[:8])
	i.offset = binary.BigEndian.Uint64(data[8:16])
	if data[16] == 1 {
		i.failed = true
	}
	return nil
}

type txInfoWithTx struct {
	tx proto.Transaction
	txInfo
}

type recentTransactions struct {
	infos map[string]txInfoWithTx
}

func newRecentTransactions() *recentTransactions {
	return &recentTransactions{infos: make(map[string]txInfoWithTx)}
}

func (r *recentTransactions) appendTx(id []byte, inf *txInfoWithTx) {
	r.infos[string(id)] = *inf
}

func (r *recentTransactions) txInfoById(id []byte) (txInfoWithTx, error) {
	info, ok := r.infos[string(id)]
	if !ok {
		return txInfoWithTx{}, errNotFound
	}
	return info, nil
}

func (r *recentTransactions) reset() {
	r.infos = make(map[string]txInfoWithTx)
}

type blockMeta struct {
	txStartOffset     uint64
	txEndOffset       uint64
	headerStartOffset uint64
	headerEndOffset   uint64
	height            uint64
}

func (m *blockMeta) unmarshal(data []byte) error {
	if len(data) != blockMetaSize {
		return errInvalidDataSize
	}
	m.txStartOffset = binary.BigEndian.Uint64(data[:8])
	m.txEndOffset = binary.BigEndian.Uint64(data[8:16])
	m.headerStartOffset = binary.BigEndian.Uint64(data[16:24])
	m.headerEndOffset = binary.BigEndian.Uint64(data[24:32])
	m.height = binary.BigEndian.Uint64(data[32:40])
	return nil
}

func (m *blockMeta) bytes() []byte {
	res := make([]byte, blockMetaSize)
	binary.BigEndian.PutUint64(res[:8], m.txStartOffset)
	binary.BigEndian.PutUint64(res[8:16], m.txEndOffset)
	binary.BigEndian.PutUint64(res[16:24], m.headerStartOffset)
	binary.BigEndian.PutUint64(res[24:32], m.headerEndOffset)
	binary.BigEndian.PutUint64(res[32:40], m.height)
	return res
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

	stateDB *stateDB

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
	rtx            *recentTransactions
	rheaders       map[proto.BlockID]proto.BlockHeader
	blockInfo      map[proto.BlockID]blockMeta
	height2IDCache map[uint64]proto.BlockID

	curBlockMeta blockMeta

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

// openOrCreateForAppending function opens file if it exists or creates new in other case.
func openOrCreateForAppending(path string) (*os.File, uint64, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
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

func newBlockReadWriter(
	dir string,
	offsetLen int,
	headerOffsetLen int,
	stateDB *stateDB,
	scheme proto.Scheme,
) (*blockReadWriter, error) {
	blockchain, blockchainSize, err := openOrCreateForAppending(filepath.Join(dir, "blockchain"))
	if err != nil {
		return nil, err
	}
	headers, headersSize, err := openOrCreateForAppending(filepath.Join(dir, "headers"))
	if err != nil {
		return nil, err
	}
	blockHeight2ID, _, err := openOrCreateForAppending(filepath.Join(dir, "block_height_to_id"))
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
	height, err := stateDB.getHeight()
	if err != nil {
		return nil, errors.Errorf("failed to retrieve height: %v", err)
	}
	rw := &blockReadWriter{
		db:                stateDB.db,
		dbBatch:           stateDB.dbBatch,
		stateDB:           stateDB,
		scheme:            scheme,
		blockchain:        blockchain,
		headers:           headers,
		blockHeight2ID:    blockHeight2ID,
		blockchainBuf:     bufio.NewWriter(blockchain),
		headersBuf:        bufio.NewWriter(headers),
		blockHeight2IDBuf: bufio.NewWriter(blockHeight2ID),
		rtx:               newRecentTransactions(),
		rheaders:          make(map[proto.BlockID]proto.BlockHeader),
		blockInfo:         make(map[proto.BlockID]blockMeta),
		height2IDCache:    make(map[uint64]proto.BlockID),
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
	if err := rw.syncWithDb(); err != nil {
		return nil, err
	}
	return rw, nil
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
	rw.curBlockMeta.txStartOffset = rw.blockchainLen
	rw.curBlockMeta.headerStartOffset = rw.headersLen
	return nil
}

func (rw *blockReadWriter) finishBlock(blockID proto.BlockID) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	rw.addingBlock = false
	rw.curBlockMeta.txEndOffset = rw.blockchainLen
	rw.curBlockMeta.headerEndOffset = rw.headersLen
	rw.curBlockMeta.height = rw.height + 1
	rw.blockInfo[blockID] = rw.curBlockMeta
	rw.curBlockMeta = blockMeta{}
	rw.height++
	return nil
}

func (rw *blockReadWriter) marshalTransaction(tx proto.Transaction) ([]byte, error) {
	var txBytes []byte
	var err error
	if rw.protobufActivated {
		txBytes, err = tx.MarshalSignedToProtobuf(rw.scheme)
		if err != nil {
			return nil, err
		}
	} else {
		txBytes, err = tx.MarshalBinary(rw.scheme)
		if err != nil {
			return nil, err
		}
	}
	// Append tx size at the beginning.
	txSize := uint32(len(txBytes))
	txSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(txSizeBytes, txSize)
	txBytesTotal := make([]byte, 4+txSize)
	copy(txBytesTotal[:4], txSizeBytes)
	copy(txBytesTotal[4:], txBytes)
	return txBytesTotal, nil
}

func (rw *blockReadWriter) writeTranasctionToMemImpl(tx proto.Transaction, txID []byte, failed bool) {
	info := &txInfoWithTx{
		tx: tx,
		txInfo: txInfo{
			height: rw.height + 1,
			offset: rw.blockchainLen,
			failed: failed,
		},
	}
	rw.rtx.appendTx(txID, info)
}

func (rw *blockReadWriter) writeTransactionToMem(tx proto.Transaction, failed bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	txID, err := tx.GetID(rw.scheme)
	if err != nil {
		return err
	}
	rw.writeTranasctionToMemImpl(tx, txID, failed)
	return nil
}

func (rw *blockReadWriter) writeTransaction(tx proto.Transaction, failed bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	txID, err := tx.GetID(rw.scheme)
	if err != nil {
		return err
	}
	// Save transaction to local storage.
	rw.writeTranasctionToMemImpl(tx, txID, failed)
	// Write transaction information to DB batch.
	key := txInfoKey{txID: txID}
	val := txInfo{offset: rw.blockchainLen, failed: failed, height: rw.height + 1}
	rw.dbBatch.Put(key.bytes(), val.bytes())
	// Update length of blockchain.
	txBytes, err := rw.marshalTransaction(tx)
	if err != nil {
		return err
	}
	rw.blockchainLen += uint64(len(txBytes))
	if rw.blockchainLen > rw.offsetEnd {
		return errors.Errorf("offset overflow: %d > %d", rw.blockchainLen, rw.offsetEnd)
	}
	// Write transaction itself.
	if _, err := rw.blockchainBuf.Write(txBytes); err != nil {
		return err
	}
	return nil
}

func (rw *blockReadWriter) marshalHeader(h *proto.BlockHeader) ([]byte, error) {
	if !rw.protobufActivated {
		return h.MarshalHeaderToBinary()
	}
	protoBytes, err := h.MarshalHeaderToProtobuf(rw.scheme)
	if err != nil {
		return nil, err
	}
	// Put addl info that is missing in Protobuf.
	headerBytes := make([]byte, 8+len(protoBytes))
	binary.BigEndian.PutUint32(headerBytes[:4], uint32(h.TransactionCount))
	binary.BigEndian.PutUint32(headerBytes[4:8], uint32(h.TransactionBlockLength))
	copy(headerBytes[8:], protoBytes)
	return headerBytes, nil
}

func (rw *blockReadWriter) writeBlockHeader(header *proto.BlockHeader) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()
	blockID := header.BlockID()
	headerBytes, err := rw.marshalHeader(header)
	if err != nil {
		return err
	}
	if _, err := rw.headersBuf.Write(headerBytes); err != nil {
		return err
	}
	rw.rheaders[blockID] = *header
	rw.headersLen += uint64(len(headerBytes))
	if rw.headersLen > rw.offsetEnd {
		return errors.Errorf("headersLen is not enough for this offset: %d > %d", rw.headersLen, rw.offsetEnd)
	}
	return nil
}

func (rw *blockReadWriter) newestBlockIDByHeight(height uint64) (proto.BlockID, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.newestBlockIDByHeightImpl(height)
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

func (rw *blockReadWriter) newestBlockIDByHeightImpl(height uint64) (proto.BlockID, error) {
	// For blockReadWriter, heights start from 0.
	if id, ok := rw.height2IDCache[height]; ok {
		return id, nil
	}
	return rw.blockIDByHeightImpl(height)
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

func (rw *blockReadWriter) blockMeta(blockID proto.BlockID) (*blockMeta, error) {
	key := blockOffsetKey{blockID: blockID}
	metaBytes, err := rw.db.Get(key.bytes())
	if err != nil {
		return nil, err
	}
	var bm blockMeta
	if err := bm.unmarshal(metaBytes); err != nil {
		return nil, err
	}
	return &bm, nil
}

func (rw *blockReadWriter) newestBlockMeta(blockID proto.BlockID) (*blockMeta, error) {
	bm, ok := rw.blockInfo[blockID]
	if ok {
		return &bm, nil
	}
	return rw.blockMeta(blockID)
}

func (rw *blockReadWriter) blockMetaByHeight(height uint64) (*blockMeta, error) {
	blockID, err := rw.blockIDByHeight(height)
	if err != nil {
		return nil, err
	}
	return rw.blockMeta(blockID)
}

func (rw *blockReadWriter) newestHeightByBlockID(blockID proto.BlockID) (uint64, error) {
	bm, err := rw.newestBlockMeta(blockID)
	if err != nil {
		return 0, err
	}
	return bm.height, nil
}

func (rw *blockReadWriter) heightByBlockID(blockID proto.BlockID) (uint64, error) {
	bm, err := rw.blockMeta(blockID)
	if err != nil {
		return 0, err
	}
	return bm.height, nil
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

func (rw *blockReadWriter) newestTransactionHeightByID(txID []byte) (uint64, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	info, err := rw.rtx.txInfoById(txID)
	if err == nil {
		return info.height, nil
	}
	return rw.transactionHeightByID(txID)
}

func (rw *blockReadWriter) transactionHeightByID(txID []byte) (uint64, error) {
	info, err := rw.transactionInfoByID(txID)
	if err != nil {
		return 0, err
	}
	return info.height, nil
}

func (rw *blockReadWriter) transactionInfoByID(txID []byte) (txInfo, error) {
	key := txInfoKey{txID: txID}
	infoBytes, err := rw.db.Get(key.bytes())
	if err != nil {
		return txInfo{}, err
	}
	var info txInfo
	err = info.unmarshal(infoBytes)
	if err != nil {
		return txInfo{}, err
	}
	return info, nil
}

func (rw *blockReadWriter) newestTransactionInfoByID(txID []byte) (txInfo, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	info, err := rw.rtx.txInfoById(txID)
	if err == nil {
		return info.txInfo, nil
	}
	return rw.transactionInfoByID(txID)
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

func (rw *blockReadWriter) readNewestTransaction(txID []byte) (proto.Transaction, bool, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	info, err := rw.rtx.txInfoById(txID)
	if err != nil {
		return rw.readTransactionImpl(txID)
	}
	return info.tx, info.failed, nil
}

func (rw *blockReadWriter) readTransaction(txID []byte) (proto.Transaction, bool, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readTransactionImpl(txID)
}

func (rw *blockReadWriter) readTransactionImpl(txID []byte) (proto.Transaction, bool, error) {
	info, err := rw.transactionInfoByID(txID)
	if err != nil {
		return nil, false, err
	}
	tx, err := rw.readTransactionByOffsetImpl(info.offset)
	return tx, info.failed, err
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

func (rw *blockReadWriter) readNewestBlockHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	blockID, err := rw.newestBlockIDByHeightImpl(height)
	if err != nil {
		return nil, err
	}
	header, err := rw.readNewestBlockHeaderImpl(blockID)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (rw *blockReadWriter) readNewestBlockHeader(blockID proto.BlockID) (*proto.BlockHeader, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readNewestBlockHeaderImpl(blockID)
}

func (rw *blockReadWriter) readBlockHeader(blockID proto.BlockID) (*proto.BlockHeader, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	return rw.readBlockHeaderImpl(blockID)
}

func (rw *blockReadWriter) readNewestBlockHeaderImpl(blockID proto.BlockID) (*proto.BlockHeader, error) {
	header, ok := rw.rheaders[blockID]
	if !ok {
		return rw.readBlockHeaderImpl(blockID)
	}
	cp := header
	return &cp, nil
}

func (rw *blockReadWriter) readBlockHeaderImpl(blockID proto.BlockID) (*proto.BlockHeader, error) {
	blockMeta, err := rw.blockMeta(blockID)
	if err != nil {
		return nil, err
	}
	headerStart := blockMeta.headerStartOffset
	headerEnd := blockMeta.headerEndOffset
	return rw.headerByBounds(headerStart, headerEnd)
}

func (rw *blockReadWriter) readBlock(blockID proto.BlockID) (*proto.Block, error) {
	rw.mtx.RLock()
	defer rw.mtx.RUnlock()
	header, err := rw.readBlockHeaderImpl(blockID)
	if err != nil {
		return nil, err
	}
	blockMeta, err := rw.blockMeta(blockID)
	if err != nil {
		return nil, err
	}
	blockStart := blockMeta.txStartOffset
	blockEnd := blockMeta.txEndOffset
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

func (rw *blockReadWriter) cleanIDs(removalEdge proto.BlockID) error {
	blockMeta, err := rw.blockMeta(removalEdge)
	if err != nil {
		return err
	}
	newHeight := blockMeta.height
	oldHeight, err := rw.stateDB.getHeight()
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
		rw.dbBatch.Delete(key.bytes())
	}
	// Clean transaction IDs.
	readPos := blockMeta.txEndOffset
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
		key := txInfoKey{txID: txID}
		rw.dbBatch.Delete(key.bytes())
	}
	return nil
}

func (rw *blockReadWriter) truncate(newHeight, newBlockchainLen, newHeadersLen uint64, removeProtobufInfo bool) error {
	rw.mtx.Lock()
	defer rw.mtx.Unlock()

	// Remove transactions.
	if err := rw.blockchain.Truncate(int64(newBlockchainLen)); err != nil {
		return err
	}
	if _, err := rw.blockchain.Seek(int64(newBlockchainLen), 0); err != nil {
		return err
	}
	// Remove headers.
	if err := rw.headers.Truncate(int64(newHeadersLen)); err != nil {
		return err
	}
	if _, err := rw.headers.Seek(int64(newHeadersLen), 0); err != nil {
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
	if removeProtobufInfo {
		// Protobuf.
		rw.dbBatch.Delete([]byte{rwProtobufInfoKeyPrefix})
		rw.protobufActivated = false
		rw.protobufTxStart = 0
		rw.protobufHeadersStart = 0
		rw.protobufAfterHeight = 0
	}
	// Decrease counters.
	rw.height = newHeight
	rw.blockchainLen = newBlockchainLen
	rw.headersLen = newHeadersLen
	// Reset buffers.
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.headersBuf.Reset(rw.headers)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	return nil
}

func (rw *blockReadWriter) rollback(newHeight uint64) error {
	if newHeight == 0 {
		// Remove everything.
		return rw.truncate(0, 0, 0, true)
	}
	blockMeta, err := rw.blockMetaByHeight(newHeight)
	if err != nil {
		return err
	}
	blockEnd := blockMeta.txEndOffset
	headerEnd := blockMeta.headerEndOffset
	removeProtobufInfo := false
	if blockEnd < rw.protobufTxStart {
		removeProtobufInfo = true
	}
	return rw.truncate(newHeight, blockEnd, headerEnd, removeProtobufInfo)
}

func (rw *blockReadWriter) reset() {
	rw.rtx.reset()
	rw.blockchainBuf.Reset(rw.blockchain)
	rw.rheaders = make(map[proto.BlockID]proto.BlockHeader)
	rw.headersBuf.Reset(rw.headers)
	rw.height2IDCache = make(map[uint64]proto.BlockID)
	rw.blockHeight2IDBuf.Reset(rw.blockHeight2ID)
	rw.blockInfo = make(map[proto.BlockID]blockMeta)
}

func (rw *blockReadWriter) flush() error {
	if err := rw.blockchainBuf.Flush(); err != nil {
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
	for blockID, info := range rw.blockInfo {
		key := blockOffsetKey{blockID}
		rw.dbBatch.Put(key.bytes(), info.bytes())
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

func (rw *blockReadWriter) syncWithDb() error {
	dbHeight, err := rw.stateDB.getHeight()
	if err != nil {
		return err
	}
	if err := rw.rollback(dbHeight); err != nil {
		return errors.Errorf("failed to remove blocks from block storage: %v", err)
	}
	zap.S().Infof("Synced to state height %d", dbHeight)
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
