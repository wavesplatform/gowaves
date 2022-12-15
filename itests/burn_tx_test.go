package itests

import (
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

type BurnTxSuite struct {
	f.BaseSuite
}

func (suite *BurnTxSuite) Test_BurnTxPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)
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

func (suite *BurnTxSuite) Test_BurnNFTFromAnotherAccountPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		//get NFT
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["NFT"], v, timeout, positive)
		utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), itx.WtErr.ErrWtGo, itx.WtErr.ErrWtScala,
			"Issue: "+itx.TxID.String(), "Version: ", v)
		//data for transfer
		transferdata := testdata.GetCommonTransferData(&suite.BaseSuite, itx.TxID)
		tdmatrix := testdata.GetBurnNFTFromAnotherAccount(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				//transfer NFT from Account 2 to Account 3
				ttx := transfer_utilities.TransferSend(&suite.BaseSuite, transferdata["NFT"], v, timeout, positive)
				utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), ttx.WtErr.ErrWtGo, ttx.WtErr.ErrWtScala,
					"Transfer: "+ttx.TxID.String(), "version:")
				//burn NFT from Account 3
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, positive)
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

func (suite *BurnTxSuite) Test_BurnTxNegative() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 1 * time.Second
	for _, v := range versions {
		issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
		itx := issue_utilities.IssueSend(&suite.BaseSuite, issuedata["reissuable"], v, 15*timeout, positive)
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
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn_utilities.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, timeout, !positive)
				txIds[name] = &tx.TxID

				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, tx.WtErr.ErrWtGo,
					tx.WtErr.ErrWtScala, "Case: ", name, "Version: ", v)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, "Case: ", name, "Version: ", v)
				utl.AssetBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffBalanceInAsset.BalanceInAssetGo,
					actualDiffBalanceInAsset.BalanceInAssetScala, "Case: ", name, "Version: ", v)
			})
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 30*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestBurnTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BurnTxSuite))
}
