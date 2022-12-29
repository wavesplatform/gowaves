package itests

import (
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
)

type TransferTxSuite struct {
	f.BaseSuite
}

func (suite *TransferTxSuite) Test_TransferPositive() {
	versions := testdata.GetVersions()
	waitForTx := true
	for _, v := range versions {
		reissuable := testdata.GetCommonIssueData(&suite.BaseSuite).Reissuable
		itx := issue_utilities.IssueSend(&suite.BaseSuite, reissuable, v, waitForTx)
		tdmatrix := testdata.GetCommonTransferData(&suite.BaseSuite, itx.TxID)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {

			})
		}
	}
}

func TestTransferTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TransferTxSuite))
}
