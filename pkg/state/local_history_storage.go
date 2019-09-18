package state

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
)

var errNotFound = errors.New("not found")

type history struct {
	key   []byte
	value *historyRecord
}

type byKey []history

func (k byKey) Len() int {
	return len(k)
}

func (k byKey) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func (k byKey) Less(i, j int) bool {
	return bytes.Compare(k[i].key, k[j].key) == -1
}

func sortEntries(entries []history) {
	sort.Sort(byKey(entries))
}

type localHistoryStorage struct {
	entries []history
	index   map[string]int
}

func newLocalHistoryStorage() (*localHistoryStorage, error) {
	return &localHistoryStorage{index: make(map[string]int)}, nil
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

func (l *localHistoryStorage) reset() {
	l.entries = nil
	l.index = make(map[string]int)
}
