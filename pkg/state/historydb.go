package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

// fullHistory returns combination of history from DB and the local storage (if any).
func fullHistory(
	key []byte,
	db keyvalue.KeyValue,
	localStor map[string][]byte,
	fmt *historyFormatter,
	filter bool,
) ([]byte, error) {
	newHist, _ := localStor[string(key)]
	prevHist, err := db.Get(key)
	if err == keyvalue.ErrNotFound {
		// New history.
		return newHist, nil
	}
	if err != nil {
		return nil, err
	}
	prevHist, err = fmt.normalize(prevHist, filter)
	if err != nil {
		return nil, err
	}
	return append(prevHist, newHist...), nil
}

// addHistoryToBatch moves history from local storage into DB batch.
func addHistoryToBatch(
	db keyvalue.KeyValue,
	dbBatch keyvalue.Batch,
	localStor map[string][]byte,
	fmt *historyFormatter,
	filter bool,
) error {
	for keyStr := range localStor {
		key := []byte(keyStr)
		newRecord, err := fullHistory(key, db, localStor, fmt, filter)
		if err != nil {
			return err
		}
		dbBatch.Put(key, newRecord)
	}
	return nil
}
