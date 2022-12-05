package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
)

type AliasTxPositiveSuite struct {
	f.BaseSuite
}

func (suite *AliasTxPositiveSuite) Test_AliasPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", v, tx.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
				utl.WavesDiffBalanceCheck(
					suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo, actualDiffBalanceInWaves.BalanceInWavesScala,
					name, "version:", v)
			})
		}
	}
}

func (suite *AliasTxPositiveSuite) Test_AliasMaxValuesPositive() {
	versions := testdata.GetVersions()
	positive := true
	timeout := 30 * time.Second
	for _, v := range versions {
		n, _ := utl.AddNewAccount(&suite.BaseSuite, testdata.TestChainID)
		utl.TransferFunds(&suite.BaseSuite, testdata.TestChainID, 5, n, 1000_00000000)
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, n)
		for name, td := range tdmatrix {
			suite.T().Run(name, func(t *testing.T) {
				tx, _, actualDiffBalanceInWaves := alias_utl.SendAliasTxAndGetWavesBalances(&suite.BaseSuite, td, v, timeout, positive)
				addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)

				utl.ExistenceTxInfoCheck(suite.T(), tx.WtErr.ErrWtGo, tx.WtErr.ErrWtScala, name, "version:", v, tx.TxID.String())
				utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
				utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWaves.BalanceInWavesGo,
					actualDiffBalanceInWaves.BalanceInWavesScala, name, "version:", v)
			})
		}
	}
}

func TestAliasTxPositiveSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxPositiveSuite))
}
