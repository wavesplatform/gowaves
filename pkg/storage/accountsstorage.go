package storage

import (
	"bufio"
	"encoding/binary"
	"io"
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
		return res, errors.New("failed to convert bytes to block ID: invalid length of bytes")
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

func NewAccountsStorage(globalStor, addr2Index, asset2Index keyvalue.IterableKeyVal, blockIdsFile string) (*AccountsStorage, error) {
	validIDs := make(map[crypto.Signature]struct{})
	if blockIdsFile != "" {
		blockIDs, err := os.Open(blockIdsFile)
		if err != nil {
			return nil, errors.Errorf("failed to open block IDs file: %v\n", err)
		}
		idBuf := make([]byte, crypto.SignatureSize)
		r := bufio.NewReader(blockIDs)
		// Copy block IDs to in-memory map.
		for {
			if n, err := io.ReadFull(r, idBuf); err != nil {
				if err != io.EOF {
					return nil, errors.Errorf("can not read block IDs from file: %v\n", err)
				}
				break
			} else if n != crypto.SignatureSize {
				return nil, errors.New("can not read ID of proper size from file")
			}
			blockID, err := toBlockID(idBuf)
			if err != nil {
				return nil, err
			}
			validIDs[blockID] = Empty
		}
		if err := blockIDs.Close(); err != nil {
			return nil, errors.Errorf("failed to close block IDs file: %v\n", err)
		}
	}
	if err := initIndexStores(addr2Index, asset2Index); err != nil {
		return nil, errors.Errorf("failed to initialise index store: %v\n", err)
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
		return 0, errors.Errorf("failed to check if address exists: %v\n", err)
	}
	if !has {
		// TODO: think about this scenario.
		return 0, nil
	}
	if asset != nil {
		has, err = s.asset2Index.Has(asset)
		if err != nil {
			return 0, errors.Errorf("failed to check if asset exists: %v\n", err)
		}
		if !has {
			// TODO: think about this scenario.
			return 0, nil
		}
	}
	key, err := s.getKey(addr, asset)
	if err != nil {
		return 0, errors.Errorf("failed to get key from address and asset: %v\n", err)
	}
	state, err := s.globalStor.Get(key)
	if err != nil {
		return 0, errors.Errorf("failed to get state for given key: %v\n", err)
	}
	// Delete invalid records.
	state, err = s.filterState(key, state)
	if err != nil {
		return 0, errors.Errorf("failed to filter state: %v\n", err)
	}
	if len(state) == 0 {
		// There were no valid records, so the state is empty after filtering.
		return 0, nil
	}
	balanceEnd := len(state) - crypto.SignatureSize
	balance := binary.LittleEndian.Uint64(state[balanceEnd-8 : balanceEnd])
	return balance, nil
}

func (s *AccountsStorage) newState(newRecord []byte, key []byte, blockID crypto.Signature) ([]byte, error) {
	has, err := s.globalStor.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		// New state.
		return newRecord, nil
	}
	// Get current state.
	state, err := s.globalStor.Get(key)
	if err != nil {
		return nil, err
	}
	// Delete invalid records.
	state, err = s.filterState(key, state)
	if err != nil {
		return nil, err
	}
	if len(state) < RECORD_SIZE {
		// State is empty after filtering, new record is the first one.
		return newRecord, nil
	}
	lastRecord := state[len(state)-RECORD_SIZE:]
	idBytes := lastRecord[len(lastRecord)-crypto.SignatureSize:]
	lastBlockID, err := toBlockID(idBytes)
	if err != nil {
		return nil, err
	}
	if lastBlockID == blockID {
		// If the last record is the same block, rewrite it.
		copy(state[len(state)-RECORD_SIZE:], newRecord)
	} else {
		// Append new record to the end.
		state = append(state, newRecord...)
	}
	return state, nil
}

func (s *AccountsStorage) SetAccountBalance(addr proto.Address, asset []byte, balance uint64, blockID crypto.Signature) error {
	key, err := s.getKey(addr, asset)
	if err != nil {
		return errors.Errorf("failed to get key from address and asset: %v", err)
	}
	if _, ok := s.validIDs[blockID]; !ok {
		s.validIDs[blockID] = Empty
	}
	// Prepare new record.
	balanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(balanceBuf, balance)
	newRecord := append(balanceBuf, blockID[:]...)
	state, err := s.newState(newRecord, key, blockID)
	if err != nil {
		return err
	}
	if err := s.globalStor.Put(key, state); err != nil {
		return err
	}
	return nil
}

func (s *AccountsStorage) RollbackBlock(blockID crypto.Signature) error {
	delete(s.validIDs, blockID)
	return nil
}
