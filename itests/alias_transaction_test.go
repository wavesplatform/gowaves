package itests

import (
	"fmt"
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
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasPositiveDataMatrix(&suite.BaseSuite)
		for name, td := range tdmatrix {
			fmt.Println("Name: ", name, "Version: ", i, "Alias: ", td.Alias)
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txId, errGo, errScala := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			utl.ExistenceTxInfoCheck(suite.T(), errGo, errScala, name, "version:", i, txId.String())

			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasMaxValuesPositive() {
	versions := testdata.GetVersions()
	timeout := 30 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasMaxPositiveDataMatrix(&suite.BaseSuite, int(i))
		for name, td := range tdmatrix {
			fmt.Println("Name: ", name, "Version: ", i, "Alias: ", td.Alias)
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			txId, errGo, errScala := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)

			currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
			actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala

			utl.ExistenceTxInfoCheck(suite.T(), errGo, errScala, name, "version:", i, txId.String())
			addrByAliasGo, addrByAliasScala := utl.GetAddressesByAlias(&suite.BaseSuite, td.Alias)
			utl.AddressByAliasCheck(suite.T(), td.Expected.ExpectedAddress.Bytes(), addrByAliasGo, addrByAliasScala)
			utl.WavesDiffBalanceCheck(
				suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala,
				name, "version:", i)
		}
	}
}

func (suite *AliasTxSuite) Test_AliasNegative() {
	versions := testdata.GetVersions()
	timeout := 5 * time.Second
	for _, i := range versions {
		tdmatrix := testdata.GetAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			fmt.Println("Name: ", name, "Version: ", i, "Alias: ", td.Alias)
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

func (suite *AliasTxSuite) Test_SameAliasNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	var n = 2
	for _, i := range versions {
		tdmatrix := testdata.GetSameAliasNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)
		for name, td := range tdmatrix {
			initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			//first alias tx should be successful
			txId1, errGo1, errScala1 := alias_utl.Alias(&suite.BaseSuite, td, i, timeout)
			txIds[name] = &txId1

			currentBalanceInWavesGo1, currentBalanceInWavesScala1 := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)
			actualDiffBalanceInWavesGo1 := initBalanceInWavesGo - currentBalanceInWavesGo1
			actualDiffBalanceInWavesScala1 := initBalanceInWavesScala - currentBalanceInWavesScala1
			utl.ExistenceTxInfoCheck(suite.T(), errGo1, errScala1, name, "version:", i, txId1.String())
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalanceAfterFirstTx,
				actualDiffBalanceInWavesGo1, actualDiffBalanceInWavesScala1)

			//second alias tx with same alias had same ID for v1 and v2
			txId2, errGo2, errScala2 := alias_utl.Alias(&suite.BaseSuite, testdata.AliasDataChangedTimestamp(&td), i, timeout)
			//already there for v1 and v2, and new for v3
			txIds[name] = &txId2

			currentBalanceInWavesGo2, currentBalanceInWavesScala2 := utl.GetAvailableBalanceInWaves(
				&suite.BaseSuite, td.Account.Address)

			actualDiffBalanceInWavesGo2 := currentBalanceInWavesGo1 - currentBalanceInWavesGo2
			actualDiffBalanceInWavesScala2 := currentBalanceInWavesScala1 - currentBalanceInWavesScala2
			utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo2, actualDiffBalanceInWavesScala2)
			if i == 3 {
				utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, errGo2, errScala2)
			}
		}
		if i == 3 {
			n = 0
		}
		//should have same tx ID for Go and Scala v1 and v2
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)
		suite.Lenf(actualTxIds, n, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func (suite *AliasTxSuite) Test_SameAliasDiffAddressesNegative() {
	versions := testdata.GetVersions()
	timeout := 15 * time.Second
	n := 4
	for _, i := range versions {
		tdmatrix := testdata.GetSameAliasDiffAddressNegativeDataMatrix(&suite.BaseSuite)
		txIds := make(map[string]*crypto.Digest)

		initBalanceInWavesGo1, initBalanceInWavesScala1 := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, tdmatrix["Account2"].Account.Address)
		initBalanceInWavesGo2, initBalanceInWavesScala2 := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, tdmatrix["Account3"].Account.Address)

		txId1, errGo1, errScala1 := alias_utl.Alias(&suite.BaseSuite, tdmatrix["Account2"], i, timeout)
		txIds["Account2"] = &txId1

		txId2, errGo2, errScala2 := alias_utl.Alias(&suite.BaseSuite, tdmatrix["Account3"], i, timeout)
		txIds["Account3"] = &txId2

		currentBalanceInWavesGo1, currentBalanceInWavesScala1 := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, tdmatrix["Account2"].Account.Address)
		currentBalanceInWavesGo2, currentBalanceInWavesScala2 := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, tdmatrix["Account3"].Account.Address)

		actualDiffBalanceInWavesGo1 := initBalanceInWavesGo1 - currentBalanceInWavesGo1
		actualDiffBalanceInWavesScala1 := initBalanceInWavesScala1 - currentBalanceInWavesScala1

		actualDiffBalanceInWavesGo2 := initBalanceInWavesGo2 - currentBalanceInWavesGo2
		actualDiffBalanceInWavesScala2 := initBalanceInWavesScala2 - currentBalanceInWavesScala2

		utl.ExistenceTxInfoCheck(suite.T(), errGo1, errScala1, "Account2", "version:", i, txId1.String())
		utl.WavesDiffBalanceCheck(suite.T(), tdmatrix["Account2"].Expected.WavesDiffBalanceAfterFirstTx, actualDiffBalanceInWavesGo1, actualDiffBalanceInWavesScala1)
		utl.WavesDiffBalanceCheck(suite.T(), tdmatrix["Account3"].Expected.WavesDiffBalance, actualDiffBalanceInWavesGo2, actualDiffBalanceInWavesScala2)

		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 2*timeout, timeout)

		if i == 3 {
			n = 2
			utl.ErrorMessageCheck(suite.T(), tdmatrix["Account3"].Expected.ErrGoMsg, tdmatrix["Account3"].Expected.ErrScalaMsg, errGo2, errScala2)
		}
		suite.Lenf(actualTxIds, n, "IDs: %#v", actualTxIds, "Version: %#v", i)
	}
}

func TestAliasTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AliasTxSuite))
}
