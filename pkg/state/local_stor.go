package state

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
)

var errNotFound = errors.New("not found")

type keyValueEntry struct {
	key   []byte
	value []byte
	// Specifies what type of blockchain entity is stored in this entry.
	entityType blockchainEntity
}

type byKey []keyValueEntry

func (k byKey) Len() int {
	return len(k)
}

func (k byKey) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func (k byKey) Less(i, j int) bool {
	return bytes.Compare(k[i].key, k[j].key) == -1
}

func sortEntries(entries []keyValueEntry) {
	sort.Sort(byKey(entries))
}

type localStorage struct {
	entries []keyValueEntry
	index   map[string]int
}

func newLocalStorage() (*localStorage, error) {
	return &localStorage{index: make(map[string]int)}, nil
}

func (l *localStorage) set(entityType blockchainEntity, key, value []byte) error {
	index, ok := l.index[string(key)]
	entry := keyValueEntry{key, value, entityType}
	if !ok {
		l.index[string(key)] = len(l.entries)
		l.entries = append(l.entries, entry)
	} else {
		l.entries[index] = entry
	}
	return nil
}

func (l *localStorage) get(key []byte) ([]byte, error) {
	index, ok := l.index[string(key)]
	if !ok {
		return nil, errNotFound
	}
	if index < 0 || index >= len(l.entries) {
		return nil, errors.New("invalid index")
	}
	return l.entries[index].value, nil
}

func (l *localStorage) getEntries() []keyValueEntry {
	return l.entries
}

func (l *localStorage) reset() {
	l.entries = nil
	l.index = make(map[string]int)
}
