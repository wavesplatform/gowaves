package itests

import (
	"github.com/wavesplatform/gowaves/itests/utilities/reissue_utilities"
)

type ReissueTxSuite struct {
	reissue_utilities.CommonReissueTxSuite
	//issue_utilities.CommonIssueTxSuite
}

/*func (suite *ReissueTxSuite) Test_ReissuePositive() {
	//создаем тестовые данные для транзакции выпуска
	issuedata := testdata.GetCommonIssueData(&suite.CommonIssueTxSuite.BaseSuite)
	//определяем промежуток времени, в течение которого будем ожидать появление информации о транзакции
	timeout := 1 * time.Minute
	//создаем транзакцию выпуска
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.CommonIssueTxSuite, issuedata["reissuable"], timeout)
	//проверяем, что транзакция выпуска попала в блокчейн
	utl.ExistenceTxInfoCheck(suite.CommonIssueTxSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())
	//создаем тестовые данные для транзакции довыпуска
	tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.CommonReissueTxSuite.BaseSuite, *itx.ID)
	for name, td := range tdmatrix {
		//запоминаем баланс waves аккаунта перед транзакцией довыпуска
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.CommonReissueTxSuite.BaseSuite, td.Account.Address)
		//запоминаем баланс ассетов перед транзакцией довыпуска
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.CommonReissueTxSuite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		//создаем транзакцию довыпуска
		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.CommonReissueTxSuite, td, timeout)
		//запоминаем баланс waves аккаунта после транзакции довыпуска
		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.CommonReissueTxSuite.BaseSuite, td.Account.Address)
		//определяем разницу между балансом waves до и после транзакции довыпуска
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		//запоминаем баланс ассетов после транзакции довыпуска
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.CommonReissueTxSuite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceInWavesGo := initAssetBalanceGo - currentAssetBalanceGo
		actualDiffAssetBalanceInWavesScala := initAssetBalanceScala - currentAssetBalanceScala
		//проверяем то, что транзакции попали в блокчейн
		utl.ExistenceTxInfoCheck(suite.CommonReissueTxSuite.T(), rErrGo, rErrScala, name, rtx.ID.String())
		//проверяем то, что разница баланса вавесов соответствует ожидаемому результату
		utl.WavesDiffBalanceCheck(suite.CommonReissueTxSuite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		//проверяем то, что разница баланса ассетов соответствует ожидаемому результату
		utl.AssetDiffBalanceCheck(suite.CommonReissueTxSuite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceInWavesGo, actualDiffAssetBalanceInWavesScala, name)
	}
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}*/
