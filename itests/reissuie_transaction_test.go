package itests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	f "github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/testdata"
	utl "github.com/wavesplatform/gowaves/itests/utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/issue_utilities"
	"github.com/wavesplatform/gowaves/itests/utilities/reissue_utilities"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type ReissueTxSuite struct {
	f.BaseSuite
}

func (suite *ReissueTxSuite) Test_ReissuePositive() {
	//создаем тестовые данные для транзакции выпуска
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	//определяем промежуток времени, в течение которого будем ожидать появление информации о транзакции
	timeout := 15 * time.Second
	//создаем транзакцию выпуска
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], timeout)
	//проверяем, что транзакция выпуска попала в блокчейн
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())
	//создаем тестовые данные для транзакции довыпуска
	tdmatrix := testdata.GetReissuePositiveDataMatrix(&suite.BaseSuite, *itx.ID)
	for name, td := range tdmatrix {
		//запоминаем баланс waves аккаунта перед транзакцией довыпуска
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//запоминаем баланс ассетов перед транзакцией довыпуска
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		//создаем транзакцию довыпуска
		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)
		//запоминаем баланс waves аккаунта после транзакции довыпуска
		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//определяем разницу между балансом waves до и после транзакции довыпуска
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		//запоминаем баланс ассетов после транзакции довыпуска
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceInWavesGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceInWavesScala := currentAssetBalanceScala - initAssetBalanceScala
		//проверяем то, что транзакции попали в блокчейн
		utl.ExistenceTxInfoCheck(suite.T(), rErrGo, rErrScala, name, rtx.ID.String())
		//проверяем то, что разница баланса вавесов соответствует ожидаемому результату
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		//проверяем то, что разница баланса ассетов соответствует ожидаемому результату
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceInWavesGo, actualDiffAssetBalanceInWavesScala, name)
	}
}

func (suite *ReissueTxSuite) Test_ReissueNFTNegative() {
	//создаем тестовые данные для транзакции выпуска
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	//определяем промежуток времени, в течение которого будем ожидать появление информации о транзакции
	timeout := 15 * time.Second
	//создаем транзакцию выпуска
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["NFT"], timeout)
	//проверяем, что транзакция выпуска попала в блокчейн
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())
	//создаем тестовые данные для транзакции довыпуска
	tdmatrix := testdata.GetReissueNFTData(&suite.BaseSuite, *itx.ID)
	txIds := make(map[string]*crypto.Digest)
	for name, td := range tdmatrix {
		//запоминаем баланс waves аккаунта перед транзакцией довыпуска
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//запоминаем баланс ассетов перед транзакцией довыпуска
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		//создаем транзакцию довыпуска
		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)
		//запоминаем id reissue tx
		txIds[name] = rtx.ID
		//запоминаем баланс waves аккаунта после транзакции довыпуска
		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//определяем разницу между балансом waves до и после транзакции довыпуска
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		//запоминаем баланс ассетов после транзакции довыпуска
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceInWavesGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceInWavesScala := currentAssetBalanceScala - initAssetBalanceScala
		//проверяем, что получили ожидаемые сообщения об ошибках
		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala)
		//проверяем то, что разница баланса вавесов соответствует ожидаемому результату
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		//проверяем то, что разница баланса ассетов соответствует ожидаемому результату
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceInWavesGo, actualDiffAssetBalanceInWavesScala, name)
	}
	//проверяем, что транзакция не попала в блокчейн
	actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, timeout, timeout)
	suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
}

func (suite *ReissueTxSuite) Test_ReissueNegative() {
	//создаем тестовые данные для транзакции выпуска
	issuedata := testdata.GetCommonIssueData(&suite.BaseSuite)
	//определяем промежуток времени, в течение которого будем ожидать появление информации о транзакции
	timeout := 3 * time.Second
	//создаем транзакцию выпуска
	itx, iErrGo, iErrScala := issue_utilities.Issue(&suite.BaseSuite, issuedata["reissuable"], 5*timeout)
	//проверяем, что транзакция выпуска попала в блокчейн
	utl.ExistenceTxInfoCheck(suite.BaseSuite.T(), iErrGo, iErrScala, "Issue: "+itx.ID.String())
	//создаем тестовые данные для транзакции довыпуска
	tdmatrix := testdata.GetReissueNegativeDataMatrix(&suite.BaseSuite, *itx.ID)
	txIds := make(map[string]*crypto.Digest)
	for name, td := range tdmatrix {
		//запоминаем баланс waves аккаунта перед транзакцией довыпуска
		initBalanceInWavesGo, initBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//запоминаем баланс ассетов перед транзакцией довыпуска
		initAssetBalanceGo, initAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		//создаем транзакцию довыпуска
		rtx, rErrGo, rErrScala := reissue_utilities.Reissue(&suite.BaseSuite, td, timeout)
		txIds[name] = rtx.ID
		//запоминаем баланс waves аккаунта после транзакции довыпуска
		currentBalanceInWavesGo, currentBalanceInWavesScala := utl.GetAvailableBalanceInWaves(
			&suite.BaseSuite, td.Account.Address)
		//определяем разницу между балансом waves до и после транзакции довыпуска
		actualDiffBalanceInWavesGo := initBalanceInWavesGo - currentBalanceInWavesGo
		actualDiffBalanceInWavesScala := initBalanceInWavesScala - currentBalanceInWavesScala
		//запоминаем баланс ассетов после транзакции довыпуска
		currentAssetBalanceGo, currentAssetBalanceScala := utl.GetAssetBalance(
			&suite.BaseSuite, td.Account.Address, itx.ID.Bytes())
		actualDiffAssetBalanceInWavesGo := currentAssetBalanceGo - initAssetBalanceGo
		actualDiffAssetBalanceInWavesScala := currentAssetBalanceScala - initAssetBalanceScala
		//проверяем то, что транзакции не попали в блокчейн
		utl.ErrorMessageCheck(suite.T(), td.Expected.ErrGoMsg, td.Expected.ErrScalaMsg, rErrGo, rErrScala)
		//проверяем то, что разница баланса вавесов соответствует ожидаемому результату
		utl.WavesDiffBalanceCheck(suite.T(), td.Expected.WavesDiffBalance, actualDiffBalanceInWavesGo, actualDiffBalanceInWavesScala, name)
		//проверяем то, что разница баланса ассетов соответствует ожидаемому результату
		utl.AssetDiffBalanceCheck(suite.T(), td.Expected.AssetDiffBalance, actualDiffAssetBalanceInWavesGo, actualDiffAssetBalanceInWavesScala, name)
		//проверяем, что транзакция не попала в блокчейн
		actualTxIds := utl.GetTxIdsInBlockchain(&suite.BaseSuite, txIds, 5*timeout, timeout)
		suite.Lenf(actualTxIds, 0, "IDs: %#v", actualTxIds)
	}
}

func TestReissueTxSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ReissueTxSuite))
}
