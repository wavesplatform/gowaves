package itests

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/invoke"
	"github.com/wavesplatform/gowaves/itests/utilities/setscript"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"testing"
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
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppAlias, &s.DApp,
		utl.MinTxFeeWavesDApp)
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
			testData := testdata.GetInvokeScriptAccountStorageUntouchedTestData(&s.BaseSuite,
				s.DApp, s.NewAccount)
			for name, td := range testData {
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
			testData := testdata.GetInvokeScriptWriteToStorageTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
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
			testData := testdata.GetInvokeScriptWriteToStorageStringTestData(&s.BaseSuite, version, s.DApp)
			for name, td := range testData {
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

type InvokeScriptFromScriptSuite struct {
	f.BaseSettingSuite
	Versions   []byte
	DAppProxy1 config.AccountInfo
	DAppProxy2 config.AccountInfo
	DAppTarget config.AccountInfo
}

func (s *InvokeScriptFromScriptSuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptFromScriptSuite) SetupSubTest() {
	dAppProxy1Alias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	dAppProxy2Alias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	dAppTargetAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	// create new proxy dApp account with deployed script
	s.DAppProxy1 = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "proxy_dapp1.ride")
	// set alias for proxy dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppProxy1Alias,
		&s.DAppProxy1, utl.MinTxFeeWavesDApp)
	// create new proxy dApp account with deployed script
	s.DAppProxy2 = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "proxy_dapp2.ride")
	// set alias for proxy dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppProxy2Alias,
		&s.DAppProxy2, utl.MinTxFeeWavesDApp)
	// create new target dApp account with funds
	s.DAppTarget = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "target_dapp.ride")
	// set alias for target dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppTargetAlias,
		&s.DAppTarget, utl.MinTxFeeWavesDApp)
}

func (s *InvokeScriptFromScriptSuite) Test_InvokeDAppFromDApp() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp from dApp", func() {
			testData := testdata.GetInvokeScriptDAppFromDAppTestData(&s.BaseSuite, version, s.DAppProxy1,
				s.DAppProxy2, s.DAppTarget)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppProxy1Go := utl.GetAccountDataGo(&s.BaseSuite, s.DAppProxy1.Address)
				dataDAppProxy1Scala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppProxy1.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[0].DataEntries, dataDAppProxy1Go,
					dataDAppProxy1Scala)

				dataDAppProxy2Go := utl.GetAccountDataGo(&s.BaseSuite, s.DAppProxy2.Address)
				dataDAppProxy2Scala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppProxy2.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[1].DataEntries, dataDAppProxy2Go,
					dataDAppProxy2Scala)

				dataDAppTargetGo := utl.GetAccountDataGo(&s.BaseSuite, s.DAppTarget.Address)
				dataDAppTargetScala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppTarget.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[2].DataEntries, dataDAppTargetGo,
					dataDAppTargetScala)
			}
		})
	}
}

func TestInvokeScriptFromScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptFromScriptSuite))
}

type InvokeDAppTargetFromDAppProxySuite struct {
	f.BaseSettingSuite
	Versions   []byte
	DAppProxy  config.AccountInfo
	DAppTarget config.AccountInfo
}

func (s *InvokeDAppTargetFromDAppProxySuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeDAppTargetFromDAppProxySuite) SetupSubTest() {
	dAppProxyAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	dAppTargetAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	// create new proxy dApp account with deployed script
	s.DAppProxy = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "temp_proxy.ride")
	// set alias for proxy dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppProxyAlias,
		&s.DAppProxy, utl.MinTxFeeWavesDApp)

	// create new target dApp account with funds
	s.DAppTarget = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "temp_target.ride")
	// set alias for target dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppTargetAlias,
		&s.DAppTarget, utl.MinTxFeeWavesDApp)
}

func (s *InvokeDAppTargetFromDAppProxySuite) Test_InvokeDAppTargetFromDAppProxy() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp from dApp", func() {
			testData := testdata.GetInvokeDAppTargetFromDAppProxy(&s.BaseSuite, version, s.DAppProxy, s.DAppTarget)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppProxyGo := utl.GetAccountDataGo(&s.BaseSuite, s.DAppProxy.Address)
				dataDAppProxyScala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppProxy.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[0].DataEntries, dataDAppProxyGo,
					dataDAppProxyScala)

				dataDAppTargetGo := utl.GetAccountDataGo(&s.BaseSuite, s.DAppTarget.Address)
				dataDAppTargetScala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppTarget.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[1].DataEntries, dataDAppTargetGo,
					dataDAppTargetScala)
			}
		})
	}
}

func TestInvokeDAppTargetFromDAppProxySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeDAppTargetFromDAppProxySuite))
}

type InvokeRecursiveScriptSuite struct {
	f.BaseSettingSuite
	Versions []byte
	DApp     config.AccountInfo
}

func (s *InvokeRecursiveScriptSuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeRecursiveScriptSuite) SetupSubTest() {
	dAppAlias := utl.RandStringBytes(5, testdata.AliasSymbolSet)
	// create new proxy dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "dapp_all_data_entries.ride")
	// set alias for proxy dApp account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, dAppAlias,
		&s.DApp, utl.MinTxFeeWavesDApp)
}

func (s *InvokeRecursiveScriptSuite) Test_InvokeDAppRecursive() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp from dApp", func() {
			testData := testdata.GetInvokeScriptDAppRecursiveTestData(&s.BaseSuite, version, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx := invoke.SendWithTestData(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataDAppScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.Entries[0].DataEntries, dataDAppGo, dataDAppScala)
			}
		})
	}
}

func TestInvokeDAppRecursive(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeRecursiveScriptSuite))
}
