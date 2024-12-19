package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/invoke"
	"github.com/wavesplatform/gowaves/itests/utilities/setscript"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
)

type InvokeScriptSuite struct {
	f.BaseSettingSuite
}

// Positive test for dApp where is checked that account storage is untouched
func (suite *InvokeScriptSuite) Test_CheckThatAccountStorageIsUntouched() {
	versions := invoke.GetVersionsInvokeScript(&suite.BaseSuite)
	// create new dApp account with deployed script
	dApp := setscript.CreateDAppAccount(&suite.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "account_data_storage.ride")
	// set alias for dApp account
	alias.SetAliasToRecipient(&suite.BaseSuite, "testdapp", dApp)
	// create new account with funds
	newAccount := transfer.GetNewAccountWithFunds(&suite.BaseSuite, testdata.TransferMaxVersion,
		utl.TestChainID, utl.DefaultAccountForLoanFunds, 1000000000)
	// set alias for new account
	alias.SetAliasToRecipient(&suite.BaseSuite, "testAcc", utl.GetAccount(&suite.BaseSuite, newAccount))
	// get test data for invoke script transaction
	invokeScriptTestData := testdata.GetInvokeScriptAccountStorageUntouchedTestData(&suite.BaseSuite,
		"testdapp", "testAcc")
	for _, version := range versions {
		for name, td := range invokeScriptTestData {
			caseName := utl.GetTestcaseNameWithVersion(name, version)
			suite.Run(caseName, func() {
				tx := invoke.SendWithTestData(&suite.BaseSuite, td, version, true)
				fmt.Sprintf("Invoke script tx: %s", tx.TxID.String())
				dataGo := utl.GetAccountDataGoByKey(&suite.BaseSuite, dApp.Address, "test")
				dataScala := utl.GetAccountDataScalaByKey(&suite.BaseSuite, dApp.Address, "test")
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntryAndKeyCheck(suite.T(), &td.Expected.DataEntry, dataGo, dataScala)
			})
		}
	}
}

func TestInvokeScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptSuite))
}
