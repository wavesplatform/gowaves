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
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasMaxValuesPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, int(i+1))
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala, name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 5 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			txIds[name] = &tx.TxID

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
				tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
				actualDiffBalanceInWaves.BalanceInWavesScala)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func (suite *AliasTxSuite) Test_SameAliasNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 15 * time.Second
	name := "Values for same alias"
	//Count of tx id in blockchain after tx, for v1 and v2 it should be 2: 1 for each node
	var idsCount = 2
	for _, i := range versions {
		tdmatrix := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for _, td := range tdmatrix {
			//first alias tx should be successful
			tx1, _, actualDiffBalanceInWaves1 := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

			utl.ExistenceTxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala, name, "version:", i, tx1.TxID.String())
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddressAfterFirstTx.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves1.BalanceInWavesGo,
				actualDiffBalanceInWaves1.BalanceInWavesScala, name, "version:", i)

			//second alias tx with same alias had same ID for v1 and v2
			tx2, _, actualDiffBalanceInWaves2 := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, i, timeout, positive)
			//already there for v1 and v2, and new for v3
			txIds[name] = &tx2.TxID

			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves2.BalanceInWavesGo,
				actualDiffBalanceInWaves2.BalanceInWavesScala)
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
	positive := false
	timeout := 15 * time.Second
	name := "Same alias for different accounts "
	var idsCount = 2
	for _, i := range versions {
		tdslice := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)

		tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdslice[0], i, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", i, tx.TxID.String())
		utl.WavesDiffBalanceCheck(suite.T(), tdslice[0].Expected.WavesDiffBalanceAfterFirstTx,
			actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala)

		for j := 1; j < len(tdslice); j++ {
			tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdslice[0], i, timeout, positive)
			txIds[name] = &tx.TxID
			utl.WavesDiffBalanceCheck(suite.T(), tdslice[j].Expected.WavesDiffBalance,
				actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesGo)
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
