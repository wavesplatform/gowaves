package state

import (
	"encoding/binary"
	"sync"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	stateInfoSize = 3
)

var (
	void = []byte{}

	stateInfoKeyBytes         = []byte{stateInfoKeyPrefix}
	rollbackMinHeightKeyBytes = []byte{rollbackMinHeightKeyPrefix}
	lastBlockNumKeyBytes      = []byte{lastBlockNumKeyPrefix}
	dbHeightKeyBytes          = []byte{dbHeightKeyPrefix}
	rwHeightKeyBytes          = []byte{rwHeightKeyPrefix}
)

type stateInfo struct {
	version            uint16
	hasExtendedApiData bool
}

func (inf *stateInfo) marshalBinary() []byte {
	buf := make([]byte, 2+1)
	binary.BigEndian.PutUint16(buf[:2], inf.version)
	proto.PutBool(buf[2:], inf.hasExtendedApiData)
	return buf
}

func (inf *stateInfo) unmarshalBinary(data []byte) error {
	if len(data) != stateInfoSize {
		return errInvalidDataSize
	}
	inf.version = binary.BigEndian.Uint16(data[:2])
	var err error
	inf.hasExtendedApiData, err = proto.Bool(data[2:])
	if err != nil {
		return err
	}
	return nil
}

func saveStateInfo(db keyvalue.KeyValue, storeApiData bool) error {
	has, err := db.Has(stateInfoKeyBytes)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	info := &stateInfo{version: StateVersion, hasExtendedApiData: storeApiData}
	infoBytes := info.marshalBinary()
	if err := db.Put(stateInfoKeyBytes, infoBytes); err != nil {
		return err
	}
	return nil
}

// stateDB is responsible for all the actions which operate on the whole DB.
// For instance, list of valid blocks and height are DB-wide entities.
type stateDB struct {
	db          keyvalue.KeyValue
	dbBatch     keyvalue.Batch
	dbWriteLock *sync.Mutex // `dbWriteLock` is lock for writing to database.
	rw          *blockReadWriter

	newestBlockIdToNum map[proto.BlockID]uint32
	newestBlockNumToId map[uint32]proto.BlockID

	blocksNum int
}

func newStateDB(db keyvalue.KeyValue, dbBatch keyvalue.Batch, rw *blockReadWriter, storeApiData bool) (*stateDB, error) {
	heightBuf := make([]byte, 8)
	has, err := db.Has(dbHeightKeyBytes)
	if err != nil {
		return nil, err
	}
	if !has {
		binary.LittleEndian.PutUint64(heightBuf, 0)
		if err := db.Put(dbHeightKeyBytes, heightBuf); err != nil {
			return nil, err
		}
	}
	has, err = db.Has(rollbackMinHeightKeyBytes)
	if err != nil {
		return nil, err
	}
	if !has {
		binary.LittleEndian.PutUint64(heightBuf, 1)
		if err := db.Put(rollbackMinHeightKeyBytes, heightBuf); err != nil {
			return nil, err
		}
	}
	dbWriteLock := &sync.Mutex{}
	if err := saveStateInfo(db, storeApiData); err != nil {
		return nil, err
	}
	return &stateDB{
		db:                 db,
		dbBatch:            dbBatch,
		dbWriteLock:        dbWriteLock,
		rw:                 rw,
		newestBlockIdToNum: make(map[proto.BlockID]uint32),
		newestBlockNumToId: make(map[uint32]proto.BlockID),
	}, nil
}

// Returns database write lock.
func (s *stateDB) retrieveWriteLock() *sync.Mutex {
	return s.dbWriteLock
}

// Sync blockReadWriter's storage (files) with the database.
func (s *stateDB) syncRw() error {
	dbHeightBytes, err := s.db.Get(dbHeightKeyBytes)
	if err != nil {
		return err
	}
	dbHeight := binary.LittleEndian.Uint64(dbHeightBytes)
	rwHeightBytes, err := s.db.Get(rwHeightKeyBytes)
	if err != nil {
		return err
	}
	rwHeight := binary.LittleEndian.Uint64(rwHeightBytes)
	zap.S().Infof("Synced to state height %d", dbHeight)
	if rwHeight < dbHeight {
		// This should never happen, because we update block storage before writing changes into DB.
		zap.S().Fatal("Impossible to sync: DB is ahead of block storage; remove data dir and restart the node")
	}
	if dbHeight == 0 {
		if err := s.rw.removeEverything(false); err != nil {
			return err
		}
	} else {
		last, err := s.rw.blockIDByHeight(dbHeight)
		if err != nil {
			return err
		}
		if err := s.rw.rollback(last, false); err != nil {
			return errors.Errorf("failed to remove blocks from block storage: %v", err)
		}
	}
	return nil
}

// addBlock() makes block officially valid (but only after batch is flushed).
func (s *stateDB) addBlock(blockID proto.BlockID) error {
	lastBlockNum, err := s.getLastBlockNum()
	if err != nil {
		return err
	}
	if _, err := s.blockIdToNum(blockID); err == nil {
		// Block is already in there.
		return nil
	}
	// Unique number of new block.
	newBlockNum := lastBlockNum + uint32(s.blocksNum)
	if _, ok := s.newestBlockNumToId[newBlockNum]; ok {
		return errors.Errorf("block number %d is already taken by some block", newBlockNum)
	}
	// Add unique block number to the list of valid nums.
	validBlocKey := validBlockNumKey{newBlockNum}
	s.dbBatch.Put(validBlocKey.bytes(), void)
	// Save block number for this ID.
	s.newestBlockIdToNum[blockID] = newBlockNum
	idToNumKey := blockIdToNumKey{blockID}
	newBlockNumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(newBlockNumBytes, newBlockNum)
	s.dbBatch.Put(idToNumKey.bytes(), newBlockNumBytes)
	// Save ID for this block number.
	s.newestBlockNumToId[newBlockNum] = blockID
	numToIdKey := blockNumToIdKey{newBlockNum}
	idBytes := blockID.Bytes()
	s.dbBatch.Put(numToIdKey.bytes(), idBytes)
	// Increase blocks counter.
	s.blocksNum++
	return nil
}

func (s *stateDB) isValidBlock(blockNum uint32) (bool, error) {
	key := validBlockNumKey{blockNum}
	return s.db.Has(key.bytes())
}

func (s *stateDB) blockIdToNum(blockID proto.BlockID) (uint32, error) {
	blockNum, ok := s.newestBlockIdToNum[blockID]
	if ok {
		return blockNum, nil
	}
	idToNumKey := blockIdToNumKey{blockID}
	blockNumBytes, err := s.db.Get(idToNumKey.bytes())
	if err != nil {
		return 0, err
	}
	blockNum = binary.LittleEndian.Uint32(blockNumBytes)
	return blockNum, nil
}

func (s *stateDB) blockNumToId(blockNum uint32) (proto.BlockID, error) {
	blockId, ok := s.newestBlockNumToId[blockNum]
	if ok {
		return blockId, nil
	}
	numToIdKey := blockNumToIdKey{blockNum}
	blockIdBytes, err := s.db.Get(numToIdKey.bytes())
	if err != nil {
		return proto.BlockID{}, err
	}
	blockId, err = proto.NewBlockIDFromBytes(blockIdBytes)
	if err != nil {
		return proto.BlockID{}, err
	}
	return blockId, nil
}

func (s *stateDB) newestBlockNumByHeight(height uint64) (uint32, error) {
	blockID, err := s.rw.newestBlockIDByHeight(height)
	if err != nil {
		return 0, err
	}
	return s.blockIdToNum(blockID)
}

func (s *stateDB) blockNumByHeight(height uint64) (uint32, error) {
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return 0, err
	}
	return s.blockIdToNum(blockID)
}

func (s *stateDB) rollbackBlock(blockID proto.BlockID) error {
	// Decrease DB's height (for sync/recovery).
	height, err := s.getHeight()
	if err != nil {
		return err
	}
	if err := s.setHeight(height-1, true); err != nil {
		return err
	}
	blockNum, err := s.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	key := validBlockNumKey{blockNum}
	if err := s.db.Delete(key.bytes()); err != nil {
		return err
	}
	numKey := blockIdToNumKey{blockID}
	if err := s.db.Delete(numKey.bytes()); err != nil {
		return err
	}
	idKey := blockNumToIdKey{blockNum}
	if err := s.db.Delete(idKey.bytes()); err != nil {
		return err
	}
	return nil
}

func (s *stateDB) setLastBlockNum(lastBlockNum uint32) error {
	lastBlockNumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lastBlockNumBytes, lastBlockNum)
	s.dbBatch.Put(lastBlockNumKeyBytes, lastBlockNumBytes)
	return nil
}

func (s *stateDB) getLastBlockNum() (uint32, error) {
	lastBlockNumBytes, err := s.db.Get(lastBlockNumKeyBytes)
	if err == keyvalue.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(lastBlockNumBytes), nil
}

func (s *stateDB) setRollbackMinHeight(height uint64) error {
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, height)
	s.dbBatch.Put(rollbackMinHeightKeyBytes, heightBytes)
	return nil
}

func (s *stateDB) getRollbackMinHeight() (uint64, error) {
	heightBytes, err := s.db.Get(rollbackMinHeightKeyBytes)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(heightBytes), nil
}

func (s *stateDB) setHeight(height uint64, directly bool) error {
	dbHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(dbHeightBytes, height)
	if directly {
		if err := s.db.Put(dbHeightKeyBytes, dbHeightBytes); err != nil {
			return err
		}
	} else {
		s.dbBatch.Put(dbHeightKeyBytes, dbHeightBytes)
	}
	return nil
}

func (s *stateDB) getHeight() (uint64, error) {
	dbHeightBytes, err := s.db.Get(dbHeightKeyBytes)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(dbHeightBytes), nil
}

func (s *stateDB) stateVersion() (int, error) {
	stateInfoBytes, err := s.db.Get(stateInfoKeyBytes)
	if err != nil {
		return 0, err
	}
	var info stateInfo
	if err := info.unmarshalBinary(stateInfoBytes); err != nil {
		return 0, err
	}
	return int(info.version), nil
}

// stateStoresApiData indicates if additional data for gRPC API must be stored.
func (s *stateDB) stateStoresApiData() (bool, error) {
	stateInfoBytes, err := s.db.Get(stateInfoKeyBytes)
	if err != nil {
		return false, err
	}
	var info stateInfo
	if err := info.unmarshalBinary(stateInfoBytes); err != nil {
		return false, err
	}
	return info.hasExtendedApiData, nil
}

func (s *stateDB) calculateNewRollbackMinHeight(newHeight uint64) (uint64, error) {
	prevRollbackMinHeight, err := s.getRollbackMinHeight()
	if err != nil {
		return 0, err
	}
	if newHeight < prevRollbackMinHeight {
		return prevRollbackMinHeight, nil
	}
	if newHeight-prevRollbackMinHeight < rollbackMaxBlocks {
		return prevRollbackMinHeight, nil
	}
	return newHeight - rollbackMaxBlocks, nil
}

func (s *stateDB) flush() error {
	// Update last block number.
	prevLastBlockNum, err := s.getLastBlockNum()
	if err != nil {
		return err
	}
	newLastBlockNum := prevLastBlockNum + uint32(s.blocksNum)
	if err := s.setLastBlockNum(newLastBlockNum); err != nil {
		return err
	}
	// Update height.
	prevHeight, err := s.getHeight()
	if err != nil {
		return err
	}
	newHeight := prevHeight + uint64(s.blocksNum)
	if err := s.setHeight(newHeight, false); err != nil {
		return err
	}
	// Update rollback minimum height.
	newRollbackMinHeight, err := s.calculateNewRollbackMinHeight(newHeight)
	if err != nil {
		return err
	}
	if err := s.setRollbackMinHeight(newRollbackMinHeight); err != nil {
		return err
	}
	s.dbWriteLock.Lock()
	// Write the whole batch to DB.
	if err := s.db.Flush(s.dbBatch); err != nil {
		return err
	}
	s.dbWriteLock.Unlock()
	return nil
}

func (s *stateDB) reset() {
	s.newestBlockIdToNum = make(map[proto.BlockID]uint32)
	s.newestBlockNumToId = make(map[uint32]proto.BlockID)
	s.blocksNum = 0
	s.dbBatch.Reset()
}

func (s *stateDB) close() error {
	return s.db.Close()
}
