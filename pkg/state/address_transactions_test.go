package state

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

func testIterImpl(t *testing.T, params StateParams) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	st, err := NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)

	defer func() {
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	blockHeight := proto.Height(107)
	blocks, err := ReadMainnetBlocksToHeight(blockHeight)
	assert.NoError(t, err)
	err = st.AddOldDeserializedBlocks(blocks)
	assert.NoError(t, err)
	err = st.StartProvidingExtendedApi()
	assert.NoError(t, err)

	addr, err := proto.NewAddressFromString("3P2CVwf4MxPBkYZKTgaNMfcTt5SwbNXQWz6")
	assert.NoError(t, err)

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
	assert.NoError(t, err)
	err = json.Unmarshal([]byte(txJs1), tx1)
	assert.NoError(t, err)
	validTxs := []proto.Transaction{tx1, tx0}

	iter, err := st.NewAddrTransactionsIterator(addr)
	assert.NoError(t, err)
	i := 0
	for iter.Next() {
		tx, err := iter.Transaction()
		assert.NoError(t, err)
		assert.Equal(t, validTxs[i], tx)
		i++
	}
	assert.Equal(t, 2, i)
	iter.Release()
	assert.NoError(t, iter.Error())
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
