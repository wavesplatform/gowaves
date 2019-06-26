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

func (s *diffStorage) setBalanceChanges(changes *balanceChanges) error {
	key := string(changes.key)
	if index, ok := s.keys[key]; ok {
		s.changes[index] = *changes
	} else {
		s.keys[key] = len(s.changes)
		s.changes = append(s.changes, *changes)
	}
	return nil
}

func (s *diffStorage) balanceChanges(key string) (*balanceChanges, error) {
	index, ok := s.keys[key]
	if !ok {
		return nil, errNotFound
	}
	return s.changes[index].safeCopy(), nil
}

// constructBalanceChanges() checks whether changes for given change key already exist, and adds new diff to them in such case.
// Otherwise, it creates fresh changes with the first diff equal to the argument.
func (s *diffStorage) constructBalanceChanges(key string, diff balanceDiff) (*balanceChanges, error) {
	// Changes for this key are already in the stor, retrieve them.
	changes, err := s.balanceChanges(key)
	if err == errNotFound {
		// Fresh changes with the first diff set.
		return newBalanceChanges([]byte(key), diff), nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "can not retrieve balance changes")
	}
	// Add new diff to existing changes.
	if err := changes.addDiff(diff); err != nil {
		return nil, errors.Wrap(err, "can not update balance changes")
	}
	return changes, nil
}

// addBalanceDiff() adds new balance diff to storage.
func (s *diffStorage) addBalanceDiff(key string, diff balanceDiff) error {
	changes, err := s.constructBalanceChanges(key, diff)
	if err != nil {
		return errors.Wrap(err, "failed to construct balance changes for given key and diff")
	}
	if err := s.setBalanceChanges(changes); err != nil {
		return errors.Wrap(err, "failed to save changes to changes storage")
	}
	return nil
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
