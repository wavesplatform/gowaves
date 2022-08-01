package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func testIterImpl(t *testing.T, params StateParams) {
	dataDir := t.TempDir()
	st, err := NewState(dataDir, true, params, settings.MainNetSettings)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = st.Close()
		require.NoError(t, err)
	})

	blockHeight := proto.Height(9900)
	blocks, err := ReadMainnetBlocksToHeight(blockHeight)
	require.NoError(t, err)
	// Add extra blocks and rollback to check that rollback scenario is handled correctly.
	_, err = st.AddDeserializedBlocks(blocks)
	require.NoError(t, err)
	err = st.RollbackToHeight(8000)
	require.NoError(t, err)
	err = st.StartProvidingExtendedApi()
	require.NoError(t, err)

	addr, err := proto.NewAddressFromString("3P2CVwf4MxPBkYZKTgaNMfcTt5SwbNXQWz6")
	require.NoError(t, err)

	var txJs0 = `
	{
	"senderPublicKey": "7LBopaBdBzQbgqrnwgmgCDhcSTb32MYhE96SnSHcqZC2",
	"amount": 569672223116,
	"sender": "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ",
	"feeAssetId": null,
	"signature": "54PsXsEBv62sB7TVREEWz8FJe59LYJFKCcXpCjQ7Dzr4HYUVKtUNibE34N6qnoYep17srBgZwGVD3FB7ChBtTMn8",
	"proofs": [
		"54PsXsEBv62sB7TVREEWz8FJe59LYJFKCcXpCjQ7Dzr4HYUVKtUNibE34N6qnoYep17srBgZwGVD3FB7ChBtTMn8"
	],
	"fee": 1,
	"recipient": "3P2CVwf4MxPBkYZKTgaNMfcTt5SwbNXQWz6",
	"id": "54PsXsEBv62sB7TVREEWz8FJe59LYJFKCcXpCjQ7Dzr4HYUVKtUNibE34N6qnoYep17srBgZwGVD3FB7ChBtTMn8",
	"type": 2,
	"timestamp": 1465747778493,
	"height": 28
	}
	`
	var txJs1 = `
	{
	"senderPublicKey": "i8qS8qkbbUcuKkvztSn4Gn9AVpYJGiKq8GaKBeWuvma",
	"amount": 100000000,
	"sender": "3PDKuBuTSag8QGMwwx8XmHJNr8vdDaH7UgB",
	"feeAssetId": null,
	"signature": "42qzKopS4Wc5BYR5bXD8fEJ65cQUo51cSFSWQKhjS97Srvxzwb5FcHwTASGoeQGToHsLGST4bBceP6pWkh1MhyCf",
	"proofs": [
	  "42qzKopS4Wc5BYR5bXD8fEJ65cQUo51cSFSWQKhjS97Srvxzwb5FcHwTASGoeQGToHsLGST4bBceP6pWkh1MhyCf"
	],
	"fee": 1,
	"recipient": "3P2CVwf4MxPBkYZKTgaNMfcTt5SwbNXQWz6",
	"id": "42qzKopS4Wc5BYR5bXD8fEJ65cQUo51cSFSWQKhjS97Srvxzwb5FcHwTASGoeQGToHsLGST4bBceP6pWkh1MhyCf",
	"type": 2,
	"timestamp": 1465753398476,
	"height": 107
	}
	`

	tx0 := &proto.Payment{Version: 1}
	tx1 := &proto.Payment{Version: 1}
	err = json.Unmarshal([]byte(txJs0), tx0)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(txJs1), tx1)
	require.NoError(t, err)
	validTxs := []proto.Transaction{tx1, tx0}

	iter, err := st.NewAddrTransactionsIterator(addr)
	require.NoError(t, err)
	i := 0
	for iter.Next() {
		tx, fs, err := iter.Transaction()
		require.NoError(t, err)
		assert.False(t, fs)
		assert.Equal(t, validTxs[i], tx)
		i++
	}
	assert.Equal(t, 2, i)
	iter.Release()
	require.NoError(t, iter.Error())
}

func TestTransactionsByAddrIterator(t *testing.T) {
	params := DefaultTestingStateParams()
	params.StoreExtendedApiData = true
	params.ProvideExtendedApi = true
	testIterImpl(t, params)
}

func TestTransactionsByAddrIteratorOptimized(t *testing.T) {
	params := DefaultTestingStateParams()
	params.StoreExtendedApiData = true
	params.ProvideExtendedApi = false
	testIterImpl(t, params)
}

func TestAddrTransactionsIdempotent(t *testing.T) {
	stor := createStorageObjects(t, true)

	params := &addressTransactionsParams{
		dir:                 t.TempDir(),
		batchedStorMemLimit: AddressTransactionsMemLimit,
		maxFileSize:         MaxAddressTransactionsFileSize,
		providesData:        false,
	}
	atx, err := newAddressTransactions(stor.db, stor.stateDB, stor.rw, params, stor.hs.amend)
	require.NoError(t, err)
	addr, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)
	tx := createPayment(t)
	txID, err := tx.GetID(proto.MainNetScheme)
	require.NoError(t, err)
	// Save the same transaction ID twice.
	// Then make sure it was added to batchedStor only once.
	err = stor.rw.writeTransaction(tx, false)
	require.NoError(t, err)
	stor.addBlock(t, blockID0)
	err = atx.saveTxIdByAddress(addr, txID, blockID0)
	require.NoError(t, err)
	err = atx.saveTxIdByAddress(addr, txID, blockID0)
	require.NoError(t, err)
	stor.flush(t)
	err = atx.flush()
	require.NoError(t, err)
	atx.reset()
	err = atx.startProvidingData()
	require.NoError(t, err)

	iter, err := atx.newTransactionsByAddrIterator(addr)
	require.NoError(t, err)
	i := 0
	for iter.Next() {
		transaction, fs, err := iter.Transaction()
		require.NoError(t, err)
		assert.False(t, fs)
		assert.Equal(t, tx, transaction)
		i++
	}
	assert.Equal(t, 1, i)
	iter.Release()
	require.NoError(t, iter.Error())
}

func TestFailedTransaction(t *testing.T) {
	stor := createStorageObjects(t, true)

	params := &addressTransactionsParams{
		dir:                 t.TempDir(),
		batchedStorMemLimit: AddressTransactionsMemLimit,
		maxFileSize:         MaxAddressTransactionsFileSize,
		providesData:        false,
	}
	atx, err := newAddressTransactions(stor.db, stor.stateDB, stor.rw, params, stor.hs.amend)
	require.NoError(t, err)
	addr, err := proto.NewAddressFromString(testAddr)
	require.NoError(t, err)

	tx := createPayment(t)
	txID, err := tx.GetID(proto.MainNetScheme)
	require.NoError(t, err)

	err = stor.rw.writeTransaction(tx, true)
	require.NoError(t, err)
	stor.addBlock(t, blockID0)
	err = atx.saveTxIdByAddress(addr, txID, blockID0)
	require.NoError(t, err)
	stor.flush(t)
	err = atx.flush()
	require.NoError(t, err)

	atx.reset()
	err = atx.startProvidingData()
	require.NoError(t, err)

	// Read transaction status from main transaction storage
	_, fs1, err := stor.rw.readTransaction(tx.ID.Bytes())
	require.NoError(t, err)
	assert.True(t, fs1)

	// Read transaction failure status from account's transactions storage
	iter, err := atx.newTransactionsByAddrIterator(addr)
	require.NoError(t, err)
	i := 0
	for iter.Next() {
		transaction, fs, err := iter.Transaction()
		require.NoError(t, err)
		assert.True(t, fs)
		assert.Equal(t, tx, transaction)
		i++
	}
	assert.Equal(t, 1, i)
	iter.Release()
	require.NoError(t, iter.Error())
}
