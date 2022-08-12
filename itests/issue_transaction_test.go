package integration_test

import (
	"fmt"
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

type TestData struct {
	AssetName  string
	AssetDesc  string
	Quantity   uint64
	Decimals   byte
	Reissuable bool
	Fee        uint64
	Timestamp  uint64
	Expected   string //ожидаемый результат
}

func NewTestData(assetName string, assetDesc string, quantity uint64, decimals byte, reissuable bool, fee uint64,
	timestamp uint64, expected string) *TestData {
	return &TestData{
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

func getDataMatrix() []TestData {
	var t = []TestData{
		*NewTestData("test", "t", uint64(1), byte(0), true, uint64(100000000), getCurrentTimestampInMs(), ""),
	}
	return t
}

func (suite *ItestSuite) Test_IssueTxWithMinValues() {
	//test data (need to use something for parametrization)
	//assetName := "test"
	//assetDesc := "t"
	//quantity := uint64(1)
	//decimals := byte(0)
	//reissuable := true
	//fee := uint64(100000000)
	//ts := getCurrentTimestampInMs()
	//
	testdata := getDataMatrix()
	//steps
	tx := proto.NewUnsignedIssueWithSig(suite.cfg.Accounts[0].PublicKey, testdata[0].AssetName, testdata[0].AssetDesc,
		testdata[0].Quantity, testdata[0].Decimals, testdata[0].Reissuable, testdata[0].Fee, testdata[0].Timestamp)
	err := tx.Sign('L', suite.cfg.Accounts[0].SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	utils.WaitForTransaction(suite.T(), suite.ctx, tx.ID, 1*time.Minute)
	fmt.Println(*tx)
	//asserts
	//проверка того, что ID транзакции соответствует ID выпущенного токена (возможно имеет смысл перенести на уровень unit)
	//assertBalance (не хватает метода, аналогичного miner.assertBalances(firstAddress, balance1 - issueFee, eff1 - issueFee))
	//assertAssetBalance (не хватает метода, аналогичного miner.assertAssetBalance(firstAddress, issueTx.id, someAssetAmount))
}
