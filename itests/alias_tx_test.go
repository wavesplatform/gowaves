package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxSuite struct {
	f.BaseSuite
}

func (suite *AliasTxSuite) Test_AliasPositive() {
	versions := alias_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, _, actualDiffBalanceInWaves := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td,
					v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
				errMsg := caseName + "Alias Tx: " + tx.TxID.String()

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala,
					errMsg)
				utl.WavesDiffBalanceCheck(
					suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
			})
		}
	}
}

func (suite *AliasTxSuite) Test_AliasMaxValuesPositive() {
	versions := alias_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	for _, v := range versions {
		n := transfer_utilities.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, n)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, _, actualDiffBalanceInWaves := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td,
					v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
				errMsg := caseName + "Alias Tx: " + tx.TxID.String()

				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala,
					errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
			})
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := alias_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := false
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, _, actualDiffBalanceInWaves := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td,
					v, waitForTx)
				txIds[name] = &tx.TxID
				errMsg := caseName + "Alias Tx: " + tx.TxID.String()

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *AliasTxSuite) Test_SameAliasNegative() {
	versions := alias_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	name := "Values for same alias"
	//Count of tx id in blockchain after tx, for v1 and v2 it should be 2: 1 for each node
	var idsCount = 2
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdslice := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		for _, td := range tdslice {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//first alias tx should be successful
				tx1, _, actualDiffBalanceInWaves1 := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite,
					td, v, waitForTx)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
				errMsg := caseName + "Alias Tx1: " + tx1.TxID.String()

				utl.TxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala, errMsg)
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddressAfterFirstTx.Bytes(),
					addrByAliasGo, addrByAliasScala, errMsg)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceAfterFirstTx,
					actualDiffBalanceInWaves1.BalanceInWavesGo, actualDiffBalanceInWaves1.BalanceInWavesScala,
					errMsg)

				//second alias tx with same alias had same ID for v1 and v2
				tx2, _, actualDiffBalanceInWaves2 := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite,
					td, v, !waitForTx)
				//already there for v1 and v2, and should be new for v3
				txIds[name] = &tx2.TxID
				errMsg = caseName + "Alias Tx2: " + tx2.TxID.String()

				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves2.BalanceInWavesGo,
					actualDiffBalanceInWaves2.BalanceInWavesScala, errMsg)
			})
		}
	}
	//should have same tx ID for Go and Scala v1 and v2
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds)
}

func (suite *AliasTxSuite) Test_SameAliasDiffAddressesNegative() {
	versions := alias_utilities.GetVersions(&suite.BaseSuite)
	waitForTx := true
	name := "Same alias for different accounts"
	var idsCount = 2
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		tdSlice := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		caseName := utl.GetTestcaseNameWithVersion(name, v)
		suite.Run(caseName, func() {
			//send alias tx from account that is in first element of testdata slice
			tx, _, actualDiffBalanceInWaves := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdSlice[0],
				v, waitForTx)
			errMsg := caseName + "Alias Tx: " + tx.TxID.String()

			utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
			utl.WavesDiffBalanceCheck(suite.T(), tdSlice[0].Expected.WavesDiffBalanceAfterFirstTx,
				actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala, errMsg)
			//send alias tx from account that is in each next slice element
			for j := 1; j < len(tdSlice); j++ {
				tx, _, actualDiffBalanceInWaves := alias_utilities.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, tdSlice[j],
					v, !waitForTx)
				txIds[name] = &tx.TxID
				errMsg = caseName + "Alias Tx: " + tx.TxID.String()

				utl.WavesDiffBalanceCheck(suite.T(), tdSlice[j].Expected.WavesDiffBalance,
					actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesGo, errMsg)
				//because of new IDs for v3
				if v == 3 {
					idsCount = 0
					utl.ErrorMessageCheck(suite.T(), tdSlice[j].Expected.ErrGoMsg, tdSlice[j].Expected.ErrScalaMsg,
						tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				}
			}
		})
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds)
}

func TestAliasTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxSuite))
}
