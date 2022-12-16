package itests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/burn_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"golang.org/x/exp/maps"
)

type BurnTxApiSuite struct {
	f.BaseSuite
}

func (suite *BurnTxApiSuite) Test_BurnTxApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 15 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.BroadcastBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, "Case: ", name, "Version: ", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
					"Burn: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, "Case: ", name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, "Case: ", name, "Version: ", v)
			})
		}
	}
}

func (suite *BurnTxSuite) Test_BurnTxApiAssetWithMaxAvailableFee() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 15 * time.Second
	for _, v := range versions {
		n, _ := utl.AddNewAccount(&suite.BaseSuite, testdata.TestChainID)
		utl.TransferFunds(&suite.BaseSuite, testdata.TestChainID, 5, n, 1000_00000000)
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetBurnAllAssetWithMaxAvailableFee(&suite.BaseSuite, itx.TxID, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.BroadcastBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)

				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, "Case: ", name, "Version: ", v)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
					"Burn: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, "Case: ", name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, "Case: ", name, "Version: ", v)
			})
		}
	}
}

func (suite *BurnTxApiSuite) Test_BurnNFTFromOwnerAccountApiPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 15 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		//get NFT
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["NFT"], v, timeout, positive)
		utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, itx, "NFT", "version", v)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		//data for transfer
		transferdata := testdata.GetCommonTransferData(&suite.BaseSuite, itx.TxID)
		tdmatrix := testdata.GetBurnNFTFromOwnerAccount(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				//transfer NFT from Account 2 to Account 3
				ttx := transfer_utilities.TransferBroadcast(&suite.BaseSuite, transferdata["NFT"], v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, ttx, name, "version", v)
				utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), ttx.WtErr.ErrWtGo, ttx.WtErr.ErrWtScala,
					"Transfer: "+ttx.TxID.String(), "version:")
				//burn NFT from Account 3
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.BroadcastBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)
				utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, tx, "Case: ", name, "Version: ", v)
				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name,
					"Burn: "+tx.TxID.String(), "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, "Case: ", name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, "Case: ", name, "Version: ", v)
			})
		}
	}
}

func (suite *BurnTxApiSuite) Test_BurnTxApiNegative() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 1 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueBroadcast(&suite.BaseSuite, issuedata["reissuable"], v, 15*timeout, positive)
		utl.StatusCodesCheck(suite.T(), http.StatusOK, http.StatusOK, itx, "reissuable", "version", v)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetBurnNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of burn tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetBurnChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.BroadcastBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, !positive)

				utl.StatusCodesCheck(suite.T(), http.StatusInternalServerError, http.StatusBadRequest, tx,
					"Case: ", name, "Version: ", v)
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrBrdCstGoMsg, td.Expected.ErrBrdCstScalaMsg,
					tx.BrdCstErr.ErrorBrdCstGo, tx.BrdCstErr.ErrorBrdCstScala, "Case: ", name, "Version: ", v)

				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, "Case: ", name, "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, "Case: ", name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, "Case: ", name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestBurnTxApiSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BurnTxApiSuite))
}
