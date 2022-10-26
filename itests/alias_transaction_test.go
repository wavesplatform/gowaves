package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	alias_utl "github.com/wavesplatform/gowaves/itests/utilities/alias_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type AliasTxSuite struct {
	f.BaseSuite
}

func (suite *AliasTxSuite) Test_AliasPositive() {
	versions := testdata.GetVersions(&suite.BaseSuite)
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txId, errGo, errScala := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
			utl.ExistenceTxInfoCheck(suite.T(), errGo, errScala, name, "version:", i, txId.String())
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := testdata.GetVersions(&suite.BaseSuite)
	timeout := 5 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txId, errGo, errScala := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &txId

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errGo, errScala)
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala)
		}
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 20*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func TestAliasTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxSuite))
}
