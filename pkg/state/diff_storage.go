package state

import (
	"github.com/pkg/errors"
)

// balanceChanges is a full collection of changes for given key.
// balanceDiffs is slice of per-block cumulative diffs.
type balanceChanges struct {
	// Key in main DB.
	key []byte
	// Cumulative diffs of blocks transactions.
	balanceDiffs []balanceDiff
}

// newBalanceChanges() constructs new balanceChanges from the first balance diff.
func newBalanceChanges(key []byte, diff balanceDiff) *balanceChanges {
	return &balanceChanges{key, []balanceDiff{diff}}
}

func (ch *balanceChanges) safeCopy() *balanceChanges {
	newChanges := &balanceChanges{}
	newChanges.key = make([]byte, len(ch.key))
	copy(newChanges.key[:], ch.key[:])
	newChanges.balanceDiffs = make([]balanceDiff, len(ch.balanceDiffs))
	copy(newChanges.balanceDiffs[:], ch.balanceDiffs[:])
	return newChanges
}

func (ch *balanceChanges) addDiff(newDiff balanceDiff) error {
	if len(ch.balanceDiffs) < 1 {
		return errors.New("trying to addDiff() to empty balanceChanges")
	}
	last := len(ch.balanceDiffs) - 1
	lastDiff := ch.balanceDiffs[last]
	if err := newDiff.addInsideBlock(&lastDiff); err != nil {
		return errors.Errorf("failed to add diffs: %v\n", err)
	}
	if newDiff.blockID != lastDiff.blockID {
		ch.balanceDiffs = append(ch.balanceDiffs, newDiff)
	} else {
		ch.balanceDiffs[last] = newDiff
	}
	return nil
}

func (ch *balanceChanges) latestDiff() (balanceDiff, error) {
	if len(ch.balanceDiffs) == 0 {
		return balanceDiff{}, errNotFound
	}
	return ch.balanceDiffs[len(ch.balanceDiffs)-1], nil
}

// Diff storage stores balances diffs, grouping them by keys.
// For each key, a complete history for all the blocks is stored.
// These changes can be retrieved either altogether or by the keys list.
type diffStorage struct {
	changes []balanceChanges
	keys    map[string]int // key --> index in changes.
}

func newDiffStorage() (*diffStorage, error) {
	return &diffStorage{keys: make(map[string]int)}, nil
}

func (s *diffStorage) latestDiffByKey(key string) (balanceDiff, error) {
	index, ok := s.keys[key]
	if !ok {
		return balanceDiff{}, errNotFound
	}
	return s.changes[index].latestDiff()
}

func (s *diffStorage) setBalanceChanges(changes *balanceChanges) {
	key := string(changes.key)
	if index, ok := s.keys[key]; ok {
		s.changes[index] = *changes
	} else {
		s.keys[key] = len(s.changes)
		s.changes = append(s.changes, *changes)
	}
}

func (s *diffStorage) balanceChanges(key string) (*balanceChanges, error) {
	index, ok := s.keys[key]
	if !ok {
		return nil, errNotFound
	}
	return s.changes[index].safeCopy(), nil
}

func (s *diffStorage) balanceChangesWithNewDiff(key string, newDiff balanceDiff) (*balanceChanges, error) {
	// Changes for this key are already in the stor, retrieve them.
	changes, err := s.balanceChanges(key)
	if err == errNotFound {
		// Fresh changes with the first diff set.
		return newBalanceChanges([]byte(key), newDiff), nil
	} else if err != nil {
		return nil, errors.Wrap(err, "can not retrieve balance changes")
	}
	// Add new diff.
	if err := changes.addDiff(newDiff); err != nil {
		return nil, errors.Wrap(err, "can not update balance changes")
	}
	return changes, nil
}

// addBalanceDiff() adds new balance diff to storage.
func (s *diffStorage) addBalanceDiff(key string, diff balanceDiff) error {
	index, ok := s.keys[key]
	if !ok {
		changes := newBalanceChanges([]byte(key), diff)
		s.setBalanceChanges(changes)
		return nil
	}
	changes := &s.changes[index]
	// Add new diff to existing changes.
	if err := changes.addDiff(diff); err != nil {
		return errors.Wrap(err, "can not update balance changes")
	}
	return nil
}

func (s *diffStorage) changesByTxDiff(diff txDiff) ([]balanceChanges, error) {
	var changes []balanceChanges
	for key, balanceDiff := range diff {
		change, err := s.balanceChangesWithNewDiff(key, balanceDiff)
		if err != nil {
			return nil, err
		}
		changes = append(changes, *change)
	}
	return changes, nil
}

func (s *diffStorage) saveTxDiff(diff txDiff) error {
	for key, balanceDiff := range diff {
		if err := s.addBalanceDiff(key, balanceDiff); err != nil {
			return err
		}
	}
	return nil
}

func (s *diffStorage) saveTransactionsDiffs(diffs []txDiff) error {
	for _, diff := range diffs {
		if err := s.saveTxDiff(diff); err != nil {
			return err
		}
	}
	return nil
}

func (s *diffStorage) saveBlockDiff(diff blockDiff) error {
	if err := s.saveTxDiff(diff.minerDiff); err != nil {
		return err
	}
	if err := s.saveTransactionsDiffs(diff.txDiffs); err != nil {
		return err
	}
	return nil
}

func (s *diffStorage) changesByKeys(keys []string) ([]balanceChanges, error) {
	changes := make([]balanceChanges, len(keys))
	for i, key := range keys {
		change, err := s.balanceChanges(key)
		if err != nil {
			return nil, err
		}
		changes[i] = *change
	}
	return changes, nil
}

func (s *diffStorage) allChanges() []balanceChanges {
	return s.changes
}

func (s *diffStorage) reset() {
	s.changes = nil
	s.keys = make(map[string]int)
}

// diffStorageWrapped consists of two regular diffStorages.
// invokeDiffsStor is used for invoke partial diffs to provide intermediate balances
// to RIDE when validating InvokeScript transactions.
type diffStorageWrapped struct {
	diffStorage     *diffStorage
	invokeDiffsStor *diffStorage
}

func newDiffStorageWrapped(mainStor *diffStorage) (*diffStorageWrapped, error) {
	invokeStor, err := newDiffStorage()
	if err != nil {
		return nil, err
	}
	return &diffStorageWrapped{diffStorage: mainStor, invokeDiffsStor: invokeStor}, nil
}

func (s *diffStorageWrapped) saveTxDiff(diff txDiff) error {
	for key, balanceDiff := range diff {
		if _, ok := s.invokeDiffsStor.keys[key]; ok {
			// If invoke stor already has changes for this key,
			// they are newer than ones from main stor, so we just need to add new diff
			// to these changes.
			if err := s.invokeDiffsStor.addBalanceDiff(key, balanceDiff); err != nil {
				return err
			}
			continue
		}
		// We don't have any changes for this key yet.
		// Changes are retrieved from main stor and new diffs are applied to them.
		change, err := s.diffStorage.balanceChangesWithNewDiff(key, balanceDiff)
		if err != nil {
			return err
		}
		// The result is saved to invoke stor.
		s.invokeDiffsStor.setBalanceChanges(change)
	}
	return nil
}

func (s *diffStorageWrapped) latestDiffByKey(key string) (balanceDiff, error) {
	if diff, err := s.invokeDiffsStor.latestDiffByKey(key); err == nil {
		// Found diff in invoke storage, return from there.
		// `minBalance` field should be ignored, since it isn't correct in invoke storage.
		diff.minBalance = 0
		return diff, nil
	}
	// Not found, return diff from main diff stor.
	return s.diffStorage.latestDiffByKey(key)
}
