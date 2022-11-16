package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxSuite struct {
	f.BaseSuite
}

func (suite *AliasTxSuite) Test_AliasPositive() {
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			tx := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)

			actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasMaxValuesPositive() {
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, int(i+1))
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			tx := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)

			actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := testdata.GetVersions()
	timeout := 5 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			tx := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &tx.TxID

			actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
				tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func (suite *AliasTxSuite) Test_SameAliasNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	name := "Values for same alias"
	//Count of tx id in blockchain after tx, for v1 and v2 it should be 2: 1 for each node
	var idsCount = 2
	for _, i := range versions {
		tdmatrix := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for _, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			//first alias tx should be successful
			tx1 := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)
			actualDiffBalanceInWavesGo1, actualDiffBalanceInWavesScala1 := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, td.Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
			utl.ExistenceTxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala, name, "version:", i, tx1.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceAfterFirstTx,
				actualDiffBalanceInWavesGo1, actualDiffBalanceInWavesScala1)

			//get current balance and use it as init balance for second tx
			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			//second alias tx with same alias had same ID for v1 and v2
			tx2 := alias_utl.Alias(&suite.BaseSuite, testdata.AliasDataChangedTimestamp(&td), i, timeout)
			//already there for v1 and v2, and new for v3
			txIds[name] = &tx2.TxID

			actualDiffBalanceInWavesGo2, actualDiffBalanceInWavesScala2 := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, td.Account.Address, currentBalanceInWavesGo, currentBalanceInWavesScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo2, actualDiffBalanceInWavesScala2)
			if i == 3 {
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx2.WtErr.ErrWtGo, tx2.WtErr.ErrWtScala, name, "version:", i, tx2.TxID.String())
			}
		}
		if i == 3 {
			idsCount = 0
		}
		//should have same tx ID for Go and Scala v1 and v2
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func (suite *AliasTxSuite) Test_SameAliasDiffAddressesNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	name := "Same alias for different accounts "
	var idsCount = 2
	for _, i := range versions {
		tdslice := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)

		//get init balance in waves for account[0] before alias tx
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, tdslice[0].Account.Address)
		//alias tx for first account address in slice should be success
		tx := alias_utl.Alias(&suite.BaseSuite, tdslice[0], i, timeout)
		utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
		actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
			&suite.BaseSuite, tdslice[0].Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
		utl.WavesDiffBalanceCheck(
			suite.T(), tdslice[0].Expected.WavesDiffBalanceAfterFirstTx, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)

		for j := 1; j < len(tdslice); j++ {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, tdslice[j].Account.Address)
			tx := alias_utl.Alias(&suite.BaseSuite, tdslice[j], i, timeout)
			txIds[name] = &tx.TxID
			actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala := utl.GetActualDiffBalanceInWaves(
				&suite.BaseSuite, tdslice[j].Account.Address, initBalanceInWavesGo, initBalanceInWavesScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), tdslice[j].Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
			if i == 3 {
				idsCount = 0
				utl.ErrorMessageCheck(suite.T(), tdslice[j].Expected.ErrGoMsg, tdslice[j].Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			}
		}

		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func TestAliasTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxSuite))
}
