package state

import (
	"bytes"
	"slices"

	"github.com/pkg/errors"
)

var errNotFound = errors.New("not found")

type history struct {
	key   []byte
	value *historyRecord
}

func compareHistoryKeys(a, b history) int {
	return bytes.Compare(a.key, b.key)
}

// sortEntries sorts entries by key in ascending byte order.
func sortEntries(entries []history) {
	slices.SortFunc(entries, compareHistoryKeys)
}

type localHistoryStorage struct {
	entries []history
	index   map[string]int
}

func newLocalHistoryStorage() *localHistoryStorage {
	return &localHistoryStorage{index: make(map[string]int)}
}

func (l *localHistoryStorage) set(key []byte, value *historyRecord) error {
	index, ok := l.index[string(key)]
	entry := history{key, value}
	if !ok {
		l.index[string(key)] = len(l.entries)
		l.entries = append(l.entries, entry)
	} else {
		l.entries[index] = entry
	}
	return nil
}

func (l *localHistoryStorage) get(key []byte) (*historyRecord, error) {
	index, ok := l.index[string(key)]
	if !ok {
		return nil, errNotFound
	}
	if index < 0 || index >= len(l.entries) {
		return nil, errors.New("invalid index")
	}
	return l.entries[index].value, nil
}

func (l *localHistoryStorage) getEntries() []history {
	return l.entries
}

// reset empties the storage, keeping the allocated index map and entries slice for reuse.
// Slices previously returned by getEntries must not be used after reset.
func (l *localHistoryStorage) reset() {
	clear(l.entries) // Drop references to keys and records so they can be collected.
	l.entries = l.entries[:0]
	clear(l.index)
}
