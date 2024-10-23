package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/invoke"
	"github.com/wavesplatform/gowaves/itests/utilities/setscript"
)

type InvokeScriptSuite struct {
	f.BaseSettingSuite
}

func (suite *InvokeScriptSuite) Test_InvokeScriptPositive() {
	//versions := invoke.GetVersionsInvokeScript(&suite.BaseSuite)
	//get testdata for set script tx
	setScriptTestData := testdata.GetSetScriptDataMatrix(&suite.BaseSuite)
	//create set script tx and wait response from nodes
	for name, td := range setScriptTestData {
		caseName := utl.GetTestcaseNameWithVersion(name, 2)
		suite.Run(caseName, func() {
			tx := setscript.SendWithTestData(&suite.BaseSuite, 2, td, true)
			fmt.Sprintf("Set script tx: %s", tx.TxID.String())
		})
	}

	//get testdata for invoke script tx
	invokeScriptTestData := testdata.GetInvokeScriptTestData(&suite.BaseSuite)
	//create invoke script tx and wait response from nodes
	for name, td := range invokeScriptTestData {
		caseName := utl.GetTestcaseNameWithVersion(name, 2)
		suite.Run(caseName, func() {
			tx := invoke.SendWithTestData(&suite.BaseSuite, td, 2, true)
			fmt.Sprintf("Invoke script tx: %s", tx.TxID.String())
		})
	}
}

func (suite *InvokeScriptSuite) Test_AttachedPaymentsValidation() {
	//create dApp1 account
	dApp1 := setscript.CreateDAppAccount(&suite.BaseSuite, 2, utl.DefaultAccountForLoanFunds,
		10000000000, "src1.ride")
	//create dApp2 account
	dApp2 := setscript.CreateDAppAccount(&suite.BaseSuite, 2, utl.DefaultAccountForLoanFunds,
		10000000000, "src2.ride")
	invokeScriptTestData := testdata.CheckInvokeDAppFromDApp(&suite.BaseSuite, dApp2, dApp1)
	for name, td := range invokeScriptTestData {
		caseName := utl.GetTestcaseNameWithVersion(name, 2)
		suite.Run(caseName, func() {
			tx := invoke.SendWithTestData(&suite.BaseSuite, td, 2, true)
			fmt.Sprintf("Invoke script tx: %s", tx.TxID.String())
		})
	}
}

func TestInvokeScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptSuite))
}
