package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/state/history"
)

// fullHistory returns combination of history from DB and the local storage (if any).
func fullHistory(
	key []byte,
	db keyvalue.KeyValue,
	localStor map[string][]byte,
	fmt *history.HistoryFormatter,
) ([]byte, error) {
	newHist, _ := localStor[string(key)]
	has, err := db.Has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		// New history.
		return newHist, nil
	}
	prevHist, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	prevHist, err = fmt.Normalize(prevHist)
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
	fmt *history.HistoryFormatter,
) error {
	for keyStr := range localStor {
		key := []byte(keyStr)
		newRecord, err := fullHistory(key, db, localStor, fmt)
		if err != nil {
			return err
		}
		dbBatch.Put(key, newRecord)
	}
	return nil
}
