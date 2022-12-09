package itests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxApiSuite struct {
	f.BaseSuite
}

func (suite *AliasTxApiSuite) Test_AliasTxApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", v, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", v)
			})
		}
	}
}

func (suite *AliasTxApiSuite) Test_AliasTxApiMaxValuesPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		n, _ := utl.AddNewAccount(&suite.BaseSuite, testdata.TestChainID)
		utl.TransferFunds(&suite.BaseSuite, testdata.TestChainID, 5, n, 1000_00000000)
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version", v, tx.TxID.String())
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", v)
			})
		}
	}
}

func (suite *AliasTxApiSuite) Test_AliasTxApiNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 5 * time.Second
	for _, v := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version: ", v)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, name, "version", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", v)
	}
}

func (suite *AliasTxApiSuite) Test_SameAliasApiNegative() {
	versions := testdata.GetVersions()
	positive := false
	timeout := 15 * time.Second
	name := "Values for same alias"
	//Count of tx id in blockchain after tx, for v1 and v2 it should be 2: 1 for each node
	var idsCount = 2
	for _, v := range versions {
		txIds := make(map[string]*crypto.Digest)
		tdslice := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		for _, td := range tdslice {
			suite.T().Run(name, func(t *testing.T) {
				//first alias tx should be successful
				tx1, _, actualDiffBalanceInWaves1 := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx1, name, "version: ", v)

				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.ExistenceTxInfoCheck(suite.T(), tx1.WtErr.ErrWtGo, tx1.WtErr.ErrWtScala, name, "version:", v, tx1.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddressAfterFirstTx.Bytes(), addrByAliasGo, addrByAliasScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceAfterFirstTx, actualDiffBalanceInWaves1.BalanceInWavesGo,
					actualDiffBalanceInWaves1.BalanceInWavesScala, name, "version:", v)

				//second alias tx with same alias had same ID for v1 and v2
				tx2, _, actualDiffBalanceInWaves2 := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx2, name, "version: ", v)
				//already there for v1 and v2, and should be new for v3
				txIds[name] = &tx2.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx2.BrdCstErr.ErrorBrdCstGo, tx2.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves2.BalanceInWavesGo,
					actualDiffBalanceInWaves2.BalanceInWavesScala)
			})
		}
		//should have same tx ID for Go and Scala v1 and v2
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version:", v)
	}
}

func (suite *AliasTxApiSuite) Test_SameAliasDiffAddressesApiNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	positive := false
	name := "Same alias for different accounts "
	var idsCount = 2
	for _, v := range versions {
		tdSlice := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		suite.T().Run(name, func(t *testing.T) {
			//send alias tx from account that is in first element of testdata slice
			tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite,
				tdSlice[0], v, timeout, positive)

			utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, name, "version: ", v)
			utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", v, tx.TxID.String())
			utl.WavesDiffBalanceCheck(suite.T(), tdSlice[0].Expected.WavesDiffBalanceAfterFirstTx,
				actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala)
			//send alias tx from account that is in each next slice element
			for j := 1; j < len(tdSlice); j++ {
				tx, _, actualDiffBalanceInWaves := alias_utl.BroadcastAliasTxAndGetWavesBalances(&suite.BaseSuite, tdSlice[j], v, timeout, positive)
				txIds[name] = &tx.TxID

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx, name, "version: ", v)

				utl.WavesDiffBalanceCheck(suite.T(), tdSlice[j].Expected.WavesDiffBalance,
					actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesGo)
				//because of new IDs for v3
				if v == 3 {
					idsCount = 0
					utl.ErrorMessageCheck(suite.T(), tdSlice[j].Expected.ErrBrdCstGoMsg, tdSlice[j].Expected.ErrBrdCstScalaMsg,
						tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, name, "version", v)
				}
			}
		})
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, idsCount, "IDs: %#v", actualTxIds, "Version: %#v", v)
	}
}

func TestAliasTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxApiSuite))
}
