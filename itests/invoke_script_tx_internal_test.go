package itests

import (
	"fmt"
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/setscript"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"testing"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/invoke"
)

type InvokeScriptTxSuite struct {
	f.BaseSettingSuite
	Versions   []byte
	DApp       config.AccountInfo
	NewAccount config.AccountInfo
	TestData   map[string]testdata.InvokeScriptTestData[testdata.ExpectedInvokeScriptDataPositive]
}

func (s *InvokeScriptTxSuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptTxSuite) SetupSubTest() {
	fmt.Println("*******SUBTEST: Setup accounts in subtests********")
	dAppAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	newAccAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	// create new dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "account_data_storage.ride")
	// set alias for dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppAlias, &s.DApp, utl.MinTxFeeWavesDApp)
	fmt.Println("DApp was setup:", s.DApp)
	// create new account with funds
	s.NewAccount = utl.GetAccount(&s.BaseSuite, transfer.GetNewAccountWithFunds(&s.BaseSuite,
		testdata.TransferMaxVersion, utl.TestChainID, utl.DefaultAccountForLoanFunds, 1000000000))
	// set alias for new account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, newAccAlias,
		&s.NewAccount, utl.MinTxFeeWaves)
	fmt.Println("NewAccount was setup:", s.NewAccount)
	s.TestData = testdata.GetInvokeScriptAccountStorageUntouchedTestData(&s.BaseSuite,
		s.DApp, s.NewAccount)
	fmt.Println("TESTDATA WAS SETTING UP:", s.TestData)
	fmt.Println("*******SUBTEST: End setup accounts in subtests********")
}

// Positive test for dApp where is checked that account storage is untouched
func (s *InvokeScriptTxSuite) Test_CheckThatAccountStorageIsUntouched() {
	fmt.Println("-----------START TEST------------")
	for _, version := range s.Versions {
		fmt.Println("Version:", version)
		s.Run("start run tests", func() {
			fmt.Println("--------TESTDATA IN TEST:", s.TestData)
			for name, td := range s.TestData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.Run(caseName, func() {
					fmt.Println("Case: ", caseName)
					fmt.Println("DApp:", s.DApp)
					fmt.Println("NewAccount:", s.NewAccount)
					tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
					dataGo := utl.GetAccountDataGoByKey(&s.BaseSuite, s.DApp.Address, "test")
					dataScala := utl.GetAccountDataScalaByKey(&s.BaseSuite, s.DApp.Address, "test")
					errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
					utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
					utl.DataEntryAndKeyCheck(s.T(), td.Expected.DataEntry, dataGo, dataScala)
				})
			}
		})
	}
	fmt.Println("-----------END TEST------------")
}

// Positive test for dApp where is checked that data is written correct in dApp Account Storage
func (s *InvokeScriptTxSuite) Test_CheckWrittenDataInAccountStorage() {
	versions := invoke.GetVersionsInvokeScript(&s.BaseSuite)
	// create new dApp account with deployed script
	dApp := setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "account_data_storage.ride")
	// get test data for invoke script transaction
	invokeScriptTestData := testdata.GetInvokeScriptWriteToStorageTestData(&s.BaseSuite, s.DApp)
	for _, version := range versions {
		for name, td := range invokeScriptTestData {
			caseName := utl.GetTestcaseNameWithVersion(name, version)
			s.Run(caseName, func() {
				fmt.Println("Case: ", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGo(&s.BaseSuite, dApp.Address)
				dataScala := utl.GetAccountDataScala(&s.BaseSuite, dApp.Address)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.DataEntries, dataGo, dataScala)
			})
		}
	}
}

func TestInvokeScriptTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptTxSuite))
}
