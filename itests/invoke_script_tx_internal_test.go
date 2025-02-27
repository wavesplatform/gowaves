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
}

func (s *InvokeScriptTxSuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptTxSuite) SetupSubTest() {
	dAppAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	newAccAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	// create new dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "account_data_storage.ride")
	// set alias for dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppAlias, &s.DApp, utl.MinTxFeeWavesDApp)
	// create new account with funds
	s.NewAccount = utl.GetAccount(&s.BaseSuite, transfer.GetNewAccountWithFunds(&s.BaseSuite,
		testdata.TransferMaxVersion, utl.TestChainID, utl.DefaultAccountForLoanFunds, 1000000000))
	// set alias for new account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, newAccAlias,
		&s.NewAccount, utl.MinTxFeeWaves)
}

// Positive test for dApp where is checked that account storage is untouched
func (s *InvokeScriptTxSuite) Test_CheckThatAccountStorageIsUntouched() {
	for _, version := range s.Versions {
		s.Run("storage is untouched", func() {
			testdata := testdata.GetInvokeScriptAccountStorageUntouchedTestData(&s.BaseSuite,
				s.DApp, s.NewAccount)
			for name, td := range testdata {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGoByKey(&s.BaseSuite, s.DApp.Address, td.Expected.DataEntry.Key)
				dataScala := utl.GetAccountDataScalaByKey(&s.BaseSuite, s.DApp.Address, td.Expected.DataEntry.Key)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntryAndKeyCheck(s.T(), td.Expected.DataEntry, dataGo, dataScala)
			}
		})
	}
}

// Positive test for dApp where is checked that data is written correct in dApp Account Storage
func (s *InvokeScriptTxSuite) Test_CheckWrittenDataInAccountStorage() {
	for _, version := range s.Versions {
		s.Run("written data in account storage is correct", func() {
			testdata := testdata.GetInvokeScriptWriteToStorageTestData(&s.BaseSuite, s.DApp)
			for name, td := range testdata {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.DataEntries, dataGo, dataScala)
			}
		})
	}
}

// Positive test for dApp where is checked that string data is written correct in dApp Account Storage
func (s *InvokeScriptTxSuite) Test_CheckWrittenStringDataInAccountStorage() {
	for _, version := range s.Versions {
		s.Run("written data in account storage is correct", func() {
			testdata := testdata.GetInvokeScriptWriteToStorageStringTestData(&s.BaseSuite, version, s.DApp)
			for name, td := range testdata {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGoByKey(&s.BaseSuite, s.DApp.Address, td.Expected.DataEntry.Key)
				dataScala := utl.GetAccountDataScalaByKey(&s.BaseSuite, s.DApp.Address, td.Expected.DataEntry.Key)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntryAndKeyCheck(s.T(), td.Expected.DataEntry, dataGo, dataScala)
			}
		})
	}
}

func TestInvokeScriptTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptTxSuite))
}
