//go:build !smoke

package itests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wavesplatform/gowaves/itests/config"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/alias"
	"github.com/wavesplatform/gowaves/itests/utilities/invoke"
	"github.com/wavesplatform/gowaves/itests/utilities/setscript"
	"github.com/wavesplatform/gowaves/itests/utilities/transfer"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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
		utl.MinTxFeeWavesInvokeDApp)
	// create new account with funds
	s.NewAccount = utl.GetAccount(&s.BaseSuite, transfer.GetNewAccountWithFunds(&s.BaseSuite,
		testdata.TransferMaxVersion, utl.TestChainID, utl.DefaultAccountForLoanFunds, 1000000000))
	// set alias for new account
	alias.SetAliasToAccount(&s.BaseSuite, testdata.AliasMaxVersion, utl.TestChainID, newAccAlias,
		&s.NewAccount, utl.MinTxFeeWaves)
}

// Positive test for dApp where is checked that account storage is untouched.
func (s *InvokeScriptTxSuite) Test_CheckThatAccountStorageIsUntouched() {
	for _, version := range s.Versions {
		s.Run("storage is untouched", func() {
			testData := testdata.GetInvokeScriptAccountStorageUntouchedTestData(&s.BaseSuite,
				s.DApp, s.NewAccount)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGoByKey(&s.BaseSuite, s.DApp.Address,
					td.Expected.AccountStorage.DataEntries[0].Key)
				dataScala := utl.GetAccountDataScalaByKey(&s.BaseSuite, s.DApp.Address,
					td.Expected.AccountStorage.DataEntries[0].Key)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())

				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntryAndKeyCheck(s.T(), td.Expected.AccountStorage.DataEntries[0], dataGo, dataScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

// Positive tests for dApp where is checked that data is written correct in dApp Account Storage.
func (s *InvokeScriptTxSuite) Test_CheckWrittenDataInAccountStorage() {
	for _, version := range s.Versions {
		s.Run("written data in account storage is correct", func() {
			testData := testdata.GetInvokeScriptWriteToStorageTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())

				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorage.DataEntries, dataGo, dataScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

// Positive tests for dApp where is checked that max value of string data and function name is written correct
// in dApp Account Storage.
func (s *InvokeScriptTxSuite) Test_CheckWrittenStringDataInAccountStorage() {
	for _, version := range s.Versions {
		s.Run("written data in account storage is correct", func() {
			testData := testdata.GetInvokeScriptWriteToStorageStringTestData(&s.BaseSuite, version, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				dataGo := utl.GetAccountDataGoByKey(&s.BaseSuite, s.DApp.Address,
					td.Expected.AccountStorage.DataEntries[0].Key)
				dataScala := utl.GetAccountDataScalaByKey(&s.BaseSuite, s.DApp.Address,
					td.Expected.AccountStorage.DataEntries[0].Key)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())

				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntryAndKeyCheck(s.T(), td.Expected.AccountStorage.DataEntries[0], dataGo, dataScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
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
	// create new proxy dApp account with deployed script
	s.DAppProxy1 = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "proxy_dapp1.ride")
	// create new proxy dApp account with deployed script
	s.DAppProxy2 = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "proxy_dapp2.ride")
	// create new target dApp account with funds
	s.DAppTarget = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "target_dapp.ride")
}

// Positive tests for dApp where is checked that dApp is invoked from another dApp correctly.
func (s *InvokeScriptFromScriptSuite) Test_InvokeDAppFromDApp() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp from dApp", func() {
			testData := testdata.GetInvokeScriptDAppFromDAppTestData(&s.BaseSuite, version, s.DAppProxy1,
				s.DAppProxy2, s.DAppTarget)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppProxy1Go := utl.GetAccountDataGo(&s.BaseSuite, s.DAppProxy1.Address)
				dataDAppProxy1Scala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppProxy1.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorages[0].DataEntries, dataDAppProxy1Go,
					dataDAppProxy1Scala)

				dataDAppProxy2Go := utl.GetAccountDataGo(&s.BaseSuite, s.DAppProxy2.Address)
				dataDAppProxy2Scala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppProxy2.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorages[1].DataEntries, dataDAppProxy2Go,
					dataDAppProxy2Scala)

				dataDAppTargetGo := utl.GetAccountDataGo(&s.BaseSuite, s.DAppTarget.Address)
				dataDAppTargetScala := utl.GetAccountDataScala(&s.BaseSuite, s.DAppTarget.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorages[2].DataEntries, dataDAppTargetGo,
					dataDAppTargetScala)

				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

func TestInvokeScriptFromScriptSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptFromScriptSuite))
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
	// create new proxy dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "dapp_data_entries_recursive.ride")
}

// Positive tests for dApp where is checked that dApp is invoked recursively.
func (s *InvokeRecursiveScriptSuite) Test_InvokeDAppRecursive() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp from dApp", func() {
			testData := testdata.GetInvokeScriptDAppRecursiveTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataDAppScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorages[0].DataEntries,
					dataDAppGo, dataDAppScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

func TestInvokeDAppRecursiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeRecursiveScriptSuite))
}

type InvokeScriptMaxComplexitySuite struct {
	f.BaseSettingSuite
	Versions []byte
	DApp     config.AccountInfo
}

func (s *InvokeScriptMaxComplexitySuite) SetupSuite() {
	s.BaseSettingSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptMaxComplexitySuite) SetupSubTest() {
	// create new proxy dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "dapp_max_complexity.ride")
}

// Positive test for dApp where is checked that dApp with max complexity is invoked correctly.
func (s *InvokeScriptMaxComplexitySuite) Test_InvokeDAppComplexity() {
	for _, version := range s.Versions {
		s.Run("check dApp with max complexity", func() {
			testData := testdata.GetInvokeScriptMaxComplexityTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())
				utl.TxInfoCheck(s.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)

				utl.WaitForNewHeight(&s.BaseSuite)

				dataDAppGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataDAppScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorage.DataEntries, dataDAppGo, dataDAppScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

func TestInvokeScriptComplexitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptMaxComplexitySuite))
}

type InvokeScriptExecutionFailedSuite struct {
	f.BaseSettingScriptExecutionFailedSuite
	Versions []byte
	DApp     config.AccountInfo
}

func (s *InvokeScriptExecutionFailedSuite) SetupSuite() {
	s.BaseSettingScriptExecutionFailedSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptExecutionFailedSuite) SetupSubTest() {
	// create new proxy dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "dapp_negative.ride")
}

// Tests for Dapp where is checked that Dapp has script execution failed status when saving failed transactions.
func (s *InvokeScriptExecutionFailedSuite) Test_InvokeScriptExecutionFailed() {
	for _, version := range s.Versions {
		s.Run("check invoke dApp with script execution failed status", func() {
			testData := testdata.GetInvokeScriptExecutionFailedTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, true)
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())

				dataDAppGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataDAppScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)

				statusScala := utl.GetApplicationStatusScala(&s.BaseSuite, tx.TxID)
				statusGo := utl.GetApplicationStatusGo(&s.BaseSuite, tx.TxID)

				utl.ApplicationStatusCheck(s.T(), td.Expected.ApplicationStatus,
					statusGo.String(), statusScala.String())
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorage.DataEntries, dataDAppGo, dataDAppScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
}

func TestInvokeScriptExecutionFailedSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptExecutionFailedSuite))
}

type InvokeScriptNegativeSuite struct {
	f.BaseNegativeSuite
	Versions []byte
	DApp     config.AccountInfo
}

func (s *InvokeScriptNegativeSuite) SetupSuite() {
	s.BaseNegativeSuite.SetupSuite()
	s.Versions = invoke.GetVersionsInvokeScript(&s.BaseSuite)
}

func (s *InvokeScriptNegativeSuite) SetupSubTest() {
	// create new proxy dApp account with deployed script
	s.DApp = setscript.CreateDAppAccount(&s.BaseSuite, utl.DefaultAccountForLoanFunds,
		1000000000, "dapp_negative.ride")
}

func (s *InvokeScriptNegativeSuite) Test_InvokeScriptNegative() {
	txIDs := make(map[string]*crypto.Digest)
	for _, version := range s.Versions {
		s.Run("check invoke dApp with invalid data", func() {
			testData := testdata.GetInvokeScriptNegativeTestData(&s.BaseSuite, s.DApp)
			for name, td := range testData {
				caseName := utl.GetTestcaseNameWithVersion(name, version)
				s.T().Logf("Test case: %s\n", caseName)
				tx, diffBalances := invoke.SendWithTestDataAndGetDiffBalances(&s.BaseSuite, td, version, false)
				txIDs[name] = &tx.TxID
				errMsg := fmt.Sprintf("Case: %s; Invoke script tx: %s", caseName, tx.TxID.String())

				dataDAppGo := utl.GetAccountDataGo(&s.BaseSuite, s.DApp.Address)
				dataDAppScala := utl.GetAccountDataScala(&s.BaseSuite, s.DApp.Address)

				utl.ErrorMessageCheck(s.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg,
					tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, errMsg)
				utl.DataEntriesAndKeysCheck(s.T(), td.Expected.AccountStorage.DataEntries, dataDAppGo, dataDAppScala)
				utl.WavesDiffBalanceCheck(s.T(), td.Expected.WavesDiffBalance, diffBalances.BalanceInWavesGo,
					diffBalances.BalanceInWavesScala, errMsg)
			}
		})
	}
	actualTxIDs := utl.GetTxIdsInBlockchain(&s.BaseSuite, txIDs)
	s.Lenf(actualTxIDs, 0, "IDs: %#v", actualTxIDs)
}

func TestInvokeScriptNegativeSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InvokeScriptNegativeSuite))
}
