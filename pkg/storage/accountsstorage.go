package storage

import (
	"bufio"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	ROLLBACK_MAX_BLOCKS = 2000
	RECORD_SIZE         = crypto.SignatureSize + 8
)

var (
	lastKey = []byte("last") // For addr2Index, asset2Index.
)

type AccountsStorage struct {
	globalStor  keyvalue.IterableKeyVal // AddrIndex+AssetIndex -> [(blockID, balance), (blockID, balance), ...]
	addr2Index  keyvalue.IterableKeyVal
	asset2Index keyvalue.IterableKeyVal
	validIDs    map[crypto.Signature]struct{}
}

var Empty struct{}

func toBlockID(bytes []byte) (crypto.Signature, error) {
	var res crypto.Signature
	if len(bytes) != crypto.SignatureSize {
		return res, errors.New("Failed to convert bytes to block ID: invalid length of bytes.")
	}
	copy(res[:], bytes)
	return res, nil
}

func initIndexStores(addr2Index, asset2Index keyvalue.KeyValue) error {
	has, err := addr2Index.Has(lastKey)
	if err != nil {
		return err
	}
	if !has {
		lastBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(lastBuf, 0)
		if err := addr2Index.Put(lastKey, lastBuf); err != nil {
			return err
		}
	}
	has, err = asset2Index.Has(lastKey)
	if err != nil {
		return err
	}
	if !has {
		lastBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lastBuf, 0)
		if err := asset2Index.Put(lastKey, lastBuf); err != nil {
			return err
		}
	}
	return nil
}

// The data is stored in temporary dirs.
// It is caller's responsibility to remove them, the path list is returned.
func CreateTestAccountsStorage(blockIdsFile string) (*AccountsStorage, []string, error) {
	res := make([]string, 3)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, res, err
	}
	globalStor, err := keyvalue.NewKeyVal(dbDir0, 0)
	if err != nil {
		return nil, res, err
	}
	dbDir1, err := ioutil.TempDir(os.TempDir(), "dbDir1")
	if err != nil {
		return nil, res, err
	}
	addr2Index, err := keyvalue.NewKeyVal(dbDir1, 1)
	if err != nil {
		return nil, res, err
	}
	dbDir2, err := ioutil.TempDir(os.TempDir(), "dbDir2")
	if err != nil {
		return nil, res, err
	}
	asset2Index, err := keyvalue.NewKeyVal(dbDir2, 0)
	if err != nil {
		return nil, res, err
	}
	stor, err := NewAccountsStorage(globalStor, addr2Index, asset2Index, blockIdsFile)
	if err != nil {
		return nil, res, err
	}
	res = []string{dbDir0, dbDir1, dbDir2}
	return stor, res, nil
}

func NewAccountsStorage(globalStor, addr2Index, asset2Index keyvalue.IterableKeyVal, blockIdsFile string) (*AccountsStorage, error) {
	validIDs := make(map[crypto.Signature]struct{})
	if blockIdsFile != "" {
		blockIDs, err := os.Open(blockIdsFile)
		if err != nil {
			return nil, errors.Errorf("Failed to open block IDs file: %v\n", err)
		}
		idBuf := make([]byte, crypto.SignatureSize)
		r := bufio.NewReader(blockIDs)
		// Copy block IDs to in-memory map.
		for {
			if n, err := io.ReadFull(r, idBuf); err != nil {
				if err != io.EOF {
					return nil, errors.Errorf("Can not read block IDs from file: %v\n", err)
				}
				break
			} else if n != crypto.SignatureSize {
				return nil, errors.New("Can not read ID of proper size from file")
			}
			blockID, err := toBlockID(idBuf)
			if err != nil {
				return nil, err
			}
			validIDs[blockID] = Empty
		}
		if err := blockIDs.Close(); err != nil {
			return nil, errors.Errorf("Failed to close block IDs file: %v\n", err)
		}
	}
	if err := initIndexStores(addr2Index, asset2Index); err != nil {
		return nil, errors.Errorf("Failed to initialise index store: %v\n", err)
	}
	return &AccountsStorage{
		globalStor:  globalStor,
		addr2Index:  addr2Index,
		asset2Index: asset2Index,
		validIDs:    validIDs,
	}, nil
}

func (s *AccountsStorage) getKey(addr proto.Address, asset []byte) ([]byte, error) {
	has, err := s.addr2Index.Has(addr[:])
	if err != nil {
		return nil, err
	}
	addrIndex := make([]byte, 8)
	if has {
		addrIndex, err = s.addr2Index.Get(addr[:])
		if err != nil {
			return nil, err
		}
	} else {
		last, err := s.addr2Index.Get(lastKey)
		if err != nil {
			return nil, err
		}
		lastVal := binary.LittleEndian.Uint64(last)
		binary.LittleEndian.PutUint64(addrIndex, lastVal+1)
		if err := s.addr2Index.Put(lastKey, addrIndex); err != nil {
			return nil, err
		}
		if err := s.addr2Index.Put(addr[:], addrIndex); err != nil {
			return nil, err
		}
	}
	if asset == nil {
		// Waves.
		return addrIndex, nil
	}
	has, err = s.asset2Index.Has(asset)
	if err != nil {
		return nil, err
	}
	assetIndex := make([]byte, 4)
	if has {
		assetIndex, err = s.asset2Index.Get(asset)
		if err != nil {
			return nil, err
		}
	} else {
		last, err := s.asset2Index.Get(lastKey)
		if err != nil {
			return nil, err
		}
		lastVal := binary.LittleEndian.Uint32(last)
		binary.LittleEndian.PutUint32(assetIndex, lastVal+1)
		if err := s.asset2Index.Put(lastKey, assetIndex); err != nil {
			return nil, err
		}
		if err := s.asset2Index.Put(asset, assetIndex); err != nil {
			return nil, err
		}
	}
	return append(addrIndex, assetIndex...), nil
}

func (s *AccountsStorage) filterState(stateKey []byte, state []byte) ([]byte, error) {
	for i := len(state); i >= RECORD_SIZE; i -= RECORD_SIZE {
		record := state[i-RECORD_SIZE : i]
		idBytes := record[len(record)-crypto.SignatureSize:]
		blockID, err := toBlockID(idBytes)
		if err != nil {
			return nil, err
		}
		if _, ok := s.validIDs[blockID]; ok {
			return state, nil
		} else {
			// Erase invalid (outdated due to rollbacks) record.
			state = state[:i-RECORD_SIZE]
			if err := s.globalStor.Put(stateKey, state); err != nil {
				return nil, err
			}
		}
	}
	// If we are here, there were no valid records in state, so it should be removed.
	if err := s.globalStor.Delete(stateKey); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *AccountsStorage) WavesAddressesNumber() (uint64, error) {
	iter, err := s.addr2Index.NewKeyIterator(nil)
	if err != nil {
		return 0, err
	}
	addressesNumber := uint64(0)
	for iter.Next() {
		if string(iter.Key()) != string(lastKey) {
			addr, err := proto.NewAddressFromBytes(iter.Key())
			if err != nil {
				return 0, err
			}
			balance, err := s.AccountBalance(addr, nil)
			if err != nil {
				return 0, err
			}
			if balance > 0 {
				addressesNumber++
			}
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return 0, err
	}
	return addressesNumber, nil
}

func (s *AccountsStorage) AccountBalance(addr proto.Address, asset []byte) (uint64, error) {
	has, err := s.addr2Index.Has(addr[:])
	if err != nil {
		return 0, errors.Errorf("Failed to check if address exists: %v\n", err)
	}
	if !has {
		// TODO: think about this scenario.
		return 0, nil
	}
	if asset != nil {
		has, err = s.asset2Index.Has(asset)
		if err != nil {
			return 0, errors.Errorf("Failed to check if asset exists: %v\n", err)
		}
		if !has {
			// TODO: think about this scenario.
			return 0, nil
		}
	}
	key, err := s.getKey(addr, asset)
	if err != nil {
		return 0, errors.Errorf("Failed to get key from address and asset: %v\n", err)
	}
	state, err := s.globalStor.Get(key)
	if err != nil {
		return 0, errors.Errorf("Failed to get state for given key: %v\n", err)
	}
	// Delete invalid records.
	state, err = s.filterState(key, state)
	if err != nil {
		return 0, errors.Errorf("Failed to filter state: %v\n", err)
	}
	if len(state) == 0 {
		// There were no valid records, so the state is empty after filtering.
		return 0, nil
	}
	balanceEnd := len(state) - crypto.SignatureSize
	balance := binary.LittleEndian.Uint64(state[balanceEnd-8 : balanceEnd])
	return balance, nil
}

func (s *AccountsStorage) SetAccountBalance(addr proto.Address, asset []byte, balance uint64, blockID crypto.Signature) error {
	key, err := s.getKey(addr, asset)
	if err != nil {
		return errors.Errorf("Failed to get key from address and asset: %v", err)
	}
	if _, ok := s.validIDs[blockID]; !ok {
		s.validIDs[blockID] = Empty
	}
	// Prepare new record.
	balanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(balanceBuf, balance)
	newRecord := append(balanceBuf, blockID[:]...)
	has, err := s.globalStor.Has(key)
	if err != nil {
		return err
	}
	var state []byte
	if !has {
		// New state.
		state = newRecord
	} else {
		// Get current state.
		state, err = s.globalStor.Get(key)
		if err != nil {
			return err
		}
		// Delete invalid records.
		state, err = s.filterState(key, state)
		if err != nil {
			return err
		}
		if len(state) >= RECORD_SIZE {
			lastRecord := state[len(state)-RECORD_SIZE:]
			idBytes := lastRecord[len(lastRecord)-crypto.SignatureSize:]
			lastBlockID, err := toBlockID(idBytes)
			if err != nil {
				return err
			}
			if lastBlockID == blockID {
				// If the last record is the same block, rewrite it.
				copy(state[len(state)-RECORD_SIZE:], newRecord)
			} else {
				// Append new record to the end.
				state = append(state, newRecord...)
			}
		} else {
			// State is empty after filtering, new record is the first one.
			state = newRecord
		}
	}
	if err := s.globalStor.Put(key, state); err != nil {
		return err
	}
	return nil
}

func (s *AccountsStorage) RollbackBlock(blockID crypto.Signature) error {
	if _, ok := s.validIDs[blockID]; !ok {
		return errors.New("BlockID is not present")
	}
	delete(s.validIDs, blockID)
	return nil
}
