package itests

import (
	"fmt"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/itests/testdata"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func newSignIssueTx(suite *IssueTxSuite, testdata testdata.IssueTestData) *proto.IssueWithSig {
	tx := proto.NewUnsignedIssueWithSig(testdata.Account.PublicKey, testdata.AssetName,
		testdata.AssetDesc, testdata.Quantity, testdata.Decimals, testdata.Reissuable, testdata.Timestamp, testdata.Fee)
	err := tx.Sign(testdata.ChainID, testdata.Account.SecretKey)
	require.NoError(suite.T(), err, "failed to create proofs from signature")
	return tx
}

func issueBroadcastToGoNode(suite *IssueTxSuite, testdata testdata.IssueTestData) (*client.Response, error) {
	tx := newSignIssueTx(suite, testdata)
	return suite.Clients.GoClients.HttpClient.TransactionBroadcast(tx)
}

func issueBroadcastToScalaNode(suite *IssueTxSuite, testdata testdata.IssueTestData) (*client.Response, error) {
	tx := newSignIssueTx(suite, testdata)
	return suite.Clients.ScalaClients.HttpClient.TransactionBroadcast(tx)
}

func (suite *IssueTxSuite) Test_Issue() {
	tdmatrix := testdata.GetPositiveDataMatrix(&suite.BaseSuite)
	//timeout := 1 * time.Minute
	for name, td := range tdmatrix {
		respGo, errGo := issueBroadcastToGoNode(suite, td)
		fmt.Println(name, "Go Resp", respGo, "Go Error", errGo)
		respScala, errScala := issueBroadcastToScalaNode(suite, td)
		fmt.Println(name, "Scala Resp", respScala, "Scala Error", errScala)
	}
}
