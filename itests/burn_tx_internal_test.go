//go:build !smoke

package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/burn"
	"github.com/wavesplatform/gowaves/itests/utilities/issue"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"golang.org/x/exp/maps"
)

type BurnTxSuite struct {
	f.BaseSuite
}

func (suite *BurnTxSuite) Test_BurnTxPositive() {
	versions := burn.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetBurnPositiveDataMatrix(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Burn tx: %s", caseName, tx.TxID.String())
				burn.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *BurnTxSuite) Test_BurnTxAssetWithMaxAvailableFeePositive() {
	versions := burn.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		n := transfer.GetNewAccountWithFunds(&suite.BaseSuite, v, utl.TestChainID,
			utl.DefaultAccountForLoanFunds, 10000000000)
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetBurnAllAssetWithMaxAvailableFee(&suite.BaseSuite, itx.TxID, n)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg := fmt.Sprintf("Case: %s; Burn tx: %s", caseName, tx.TxID.String())
				burn.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *BurnTxSuite) Test_BurnNFTFromOwnerAccountPositive() {
	versions := burn.GetVersions(&suite.BaseSuite)
	for _, v := range versions {
		nft := testdata.GetCommonIssueData(&suite.BaseSuite).NFT
		//get NFT
		itx := issue.SendWithTestData(&suite.BaseSuite, nft, v, true)
		//data for transfer
		transferNFT := testdata.GetCommonTransferData(&suite.BaseSuite, &itx.TxID).NFT
		tdmatrix := testdata.GetBurnNFTFromOwnerAccount(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				//transfer NFT from Account 2 to Account 3
				ttx := transfer.SendWithTestData(&suite.BaseSuite, transferNFT, v, true)
				errMsg := fmt.Sprintf("Case: %s; Transfer tx: %s", caseName, ttx.TxID.String())

				utl.TxInfoCheck(suite.BaseSuite.T(), ttx.WtErr.ErrWtGo, ttx.WtErr.ErrWtScala, errMsg)

				//burn NFT from Account 3
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, true)
				errMsg = fmt.Sprintf("Case: %s; Burn tx: %s", caseName, tx.TxID.String())
				burn.PositiveChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
}

func (suite *BurnTxSuite) Test_BurnTxNegative() {
	versions := burn.GetVersions(&suite.BaseSuite)
	txIds := make(map[string]*crypto.Digest)
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue.SendWithTestData(&suite.BaseSuite, reissuable, v, true)
		tdmatrix := testdata.GetBurnNegativeDataMatrix(&suite.BaseSuite, itx.TxID)
		//TODO (ipereiaslavskaia) For v1 of burn tx negative cases for chainID will be ignored
		if v >= 2 {
			maps.Copy(tdmatrix, testdata.GetBurnChainIDNegativeDataMatrix(&suite.BaseSuite, itx.TxID))
		}
		for name, td := range tdmatrix {
			caseName := utl.GetTestcaseNameWithVersion(name, v)
			suite.Run(caseName, func() {
				tx, actualDiffBalanceInWaves, actualDiffBalanceInAsset := burn.SendBurnTxAndGetBalances(
					&suite.BaseSuite, td, v, false)
				txIds[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Burn tx: %s", caseName, tx.TxID.String())
				burn.NegativeChecks(suite.T(), tx, td, actualDiffBalanceInWaves, actualDiffBalanceInAsset, errMsg)
			})
		}
	}
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func TestBurnTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BurnTxSuite))
}
