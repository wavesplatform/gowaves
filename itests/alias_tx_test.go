package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxSuite struct {
	f.BaseSuite
}

func (suite *AliasTxSuite) Test_AliasPositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
					utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
				utl.WavesDiffBalanceCheck(
					suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *AliasTxSuite) Test_AliasMaxValuesPositive() {
	versions := utl.GetVersions()
	waitForTx := true
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, n)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
					utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := utl.GetVersions()
	waitForTx := false
	for _, v := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, waitForTx)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", v)
	}
}

func (suite *AliasTxSuite) Test_SameAliasNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	name := "Values for same alias"
	//Count of tx id in blockchain after tx, for v1 and v2 it should be 2: 1 for each node
	var idsCount = 2
	for _, v := range versions {
		txIds := make(map[string]*crypto.Digest)
		tdslice := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		for _, td := range tdslice {
			suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
				//first alias tx should be successful
				tx1, _, actualDiffBalanceInWaves1 := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite,
					td, v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.TxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala,
					utl.GetTestcaseNameWithVersion(name, v), tx1.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddressAfterFirstTx.Bytes(),
					addrByAliasGo, addrByAliasScala, utl.GetTestcaseNameWithVersion(name, v))
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceAfterFirstTx,
					actualDiffBalanceInWaves1.BalanceInWavesGo, actualDiffBalanceInWaves1.BalanceInWavesScala,
					utl.GetTestcaseNameWithVersion(name, v))

				//second alias tx with same alias had same ID for v1 and v2
				tx2, _, actualDiffBalanceInWaves2 := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite,
					td, v, !waitForTx)
				//already there for v1 and v2, and should be new for v3
				txIds[name] = &tx2.TxID

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves2.BalanceInWavesGo,
					actualDiffBalanceInWaves2.BalanceInWavesScala, utl.GetTestcaseNameWithVersion(name, v))
			})
		}
		//should have same tx ID for Go and Scala v1 and v2
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version:", v)
	}
}

func (suite *AliasTxSuite) Test_SameAliasDiffAddressesNegative() {
	versions := utl.GetVersions()
	waitForTx := true
	name := "Same alias for different accounts "
	var idsCount = 2
	for _, v := range versions {
		tdSlice := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		suite.Run(utl.GetTestcaseNameWithVersion(name, v), func() {
			//send alias tx from account that is in first element of testdata slice
			tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdSlice[0],
				v, waitForTx)

			utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala,
				utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), tdSlice[0].Expected.WavesDiffBalanceAfterFirstTx,
				actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala)
			//send alias tx from account that is in each next slice element
			for j := 1; j < len(tdSlice); j++ {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdSlice[j],
					v, !waitForTx)
				txIds[name] = &tx.TxID

				utl.WavesDiffBalanceCheck(suite.T(), tdSlice[j].Expected.WavesDiffBalance,
					actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesGo)
				//because of new IDs for v3
				if v == 3 {
					idsCount = 0
					utl.ErrorMessageCheck(suite.T(), tdSlice[j].Expected.ErrGoMsg, tdSlice[j].Expected.ErrScalaMsg,
						tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, utl.GetTestcaseNameWithVersion(name, v), tx.TxID.String())
				}
			}
		})
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version: %#v", v)
	}
}

func TestAliasTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxSuite))
}
