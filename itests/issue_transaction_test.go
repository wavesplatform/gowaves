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
//Необходимо восстанавливать первоначальный баланс токенов и WAVES у аккаунта

func (suite *ItestSuite) Test_IssueTxWithMinValues() {
	//test data (need to use something for parametrization)
	assetName := "t"
	assetDesc := "t"
	quantity := uint64(1)
	decimals := byte(0)
	reissuable := true
	fee := uint64(100000000)
	ts := uint64(time.Now().UnixNano() / 1000000)
	//steps
	tx := proto.NewUnsignedIssueWithSig(suite.cfg.Accounts[0].PublicKey, assetName, assetDesc, quantity, decimals,
		reissuable, ts, fee)
	err := tx.Sign('L', suite.cfg.Accounts[0].SecretKey)
	suite.NoError(err, "failed to create proofs from signature")

	bts, err := tx.MarshalBinary()
	suite.NoError(err, "failed to marshal tx")
	txMsg := proto.TransactionMessage{Transaction: bts}

	suite.conns.SendToEachNode(suite.T(), &txMsg)

	utils.WaitForTransaction(suite.T(), suite.ctx, tx.ID, 1*time.Minute)
	//asserts (need to add it)
}
