package integration_test

import (
	"github.com/wavesplatform/gowaves/itests/utils"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

//creating token with min valid values
//Precondition:
//node with account with balance that allows to pay fee
//Steps:
//Save value of account balance in variable
//Create and send tx
//Assert that:
//Транзакция успешна, создается новый токен:
//Устанавливается ID
//assetId = ID
//applicationStatus : succeeded
//Устанавливается высота блока.
//Баланс WAVES отправителя уменьшается на комиссию.
//Баланс ассетов/токенов отправителя увеличивается на quantity.
//Postconditions:
//Необходимо восстанавливать первоначальный баланс токенов и WAVES у аккаунта. (Посткондишены при тестировании блокчейна
//не очень удобно использовать, так как вернуть систему в первоначальное состояние достаточно "затратный" процесс, поэтому
//следует использовать достаточно "богатый" аккаунт)

type IssueTestData struct {
	AssetName  string
	AssetDesc  string
	Quantity   uint64
	Decimals   byte
	Reissuable bool
	Fee        uint64
	Timestamp  uint64
	Expected   string //ожидаемый результат
}

func NewIssueTestData(assetName string, assetDesc string, quantity uint64, decimals byte, reissuable bool, fee uint64,
	timestamp uint64, expected string) *IssueTestData {
	return &IssueTestData{
		AssetName:  assetName,
		AssetDesc:  assetDesc,
		Quantity:   quantity,
		Decimals:   decimals,
		Reissuable: reissuable,
		Fee:        fee,
		Timestamp:  timestamp,
		Expected:   expected,
	}
}

//Timestamp in milisec
func getCurrentTimestampInMs() uint64 {
	return uint64(time.Now().UnixNano() / 1000000)
}

func getDataMatrix() []IssueTestData {
	var t = []IssueTestData{
		*NewIssueTestData(
			"test",
			"t",
			1,
			0,
			true,
			100000000,
			getCurrentTimestampInMs(),
			""),
		*NewIssueTestData(
			"testtest",
			"testtesttestest",
			100000000000,
			4,
			true,
			100000000,
			getCurrentTimestampInMs(),
			""),
		*NewIssueTestData(
			"testtesttestest",
			"testtesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"stesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttest"+
				"testtestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestteste"+
				"sttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttes"+
				"ttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestestt"+
				"esttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttesttestesttesttestte"+
				"sttesttesttestt",
			9223372036854775807,
			8,
			true,
			100000000,
			getCurrentTimestampInMs(),
			""),
	}
	return t
}

func createAndSignTransaction(suite *ItestSuite, name string, description string, quantity uint64,
	decimals byte, reissuable bool, timestamp uint64, fee uint64) (tx *proto.IssueWithSig, err error) {
	tx = proto.NewUnsignedIssueWithSig(suite.cfg.Accounts[0].PublicKey, name, description,
		quantity, decimals, reissuable, timestamp, fee)
	err = tx.Sign('L', suite.cfg.Accounts[0].SecretKey)
	return tx, err
}

func createAndSendTransaction(suite *ItestSuite, name string, description string, quantity uint64,
	decimals byte, reissuable bool, timestamp uint64, fee uint64) {
	tx, err := createAndSignTransaction(suite, name, description, quantity, decimals, reissuable, timestamp, fee)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	utils.WaitForTransaction(suite.T(), suite.ctx, tx.ID, 1*time.Minute)
}

func (suite *ItestSuite) Test_IssueTx() {
	testdata := getDataMatrix()
	for _, td := range testdata {
		createAndSendTransaction(suite, td.AssetName, td.AssetDesc,
			td.Quantity, td.Decimals, td.Reissuable, td.Timestamp, td.Fee)
	}

	//steps

	//asserts
	//проверка того, что ID транзакции соответствует ID выпущенного токена (возможно имеет смысл перенести на уровень unit)
	//assertBalance (не хватает метода, аналогичного miner.assertBalances(firstAddress, balance1 - issueFee, eff1 - issueFee))
	//assertAssetBalance (не хватает метода, аналогичного miner.assertAssetBalance(firstAddress, issueTx.id, someAssetAmount))
}
