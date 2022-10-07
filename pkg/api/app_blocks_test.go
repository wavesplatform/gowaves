package api

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
)

func TestApp_BlocksFirst(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	g := &proto.Block{
		BlockHeader: proto.BlockHeader{
			BlockSignature: crypto.MustSignatureFromBase58("5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa"),
		},
	}

	s := mock.NewMockState(ctrl)
	s.EXPECT().BlockByHeight(proto.Height(1)).Return(g, nil)

	app, err := NewApp("api-key", nil, services.Services{State: s})
	require.NoError(t, err)
	first, err := app.BlocksFirst()
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Height)
}
func TestApp_BlocksLast(t *testing.T) {
	g := &proto.Block{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockState(ctrl)
	s.EXPECT().Height().Return(proto.Height(1), nil)
	s.EXPECT().BlockByHeight(proto.Height(1)).Return(g, nil)

	app, err := NewApp("api-key", nil, services.Services{State: s})
	require.NoError(t, err)
	first, err := app.BlocksLast()
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Height)
}

func TestAPIBlockMarshalUnmarshalJSON(t *testing.T) {
	blockJSONs := []string{
		"{\"version\":3,\"generator\":\"11111111111111111111111111\",\"timestamp\":1513416538245,\"reference\":\"4MhRMRYAteqrTDiBpkj7kqwmrMAQjwJc1vkPPacwgvaLQfsyyBg2AoJRrqV3cfxVd9iKofBY4S8jMV1NxAEzfgxp\",\"features\":[1,2],\"desiredReward\":-1,\"nxt-consensus\":{\"base-target\":77,\"generation-signature\":\"DmFCdtLsrkMx6yrFohxD3wSqJbJcURszuQQ3V51B5dy9\"},\"transactionBlockLength\":293,\"transactionCount\":1,\"generatorPublicKey\":\"89RYHiy2HD9GLfznD9NpXwuY28PDGXVhmpTJ6J7BhneA\",\"signature\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\",\"id\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\",\"height\":101,\"transactions\":[{\"type\":4,\"version\":1,\"id\":\"HFjhY9wh9DRrTUaUZoXreLNbN8TXSSBuDkRqeoHZ3c8i\",\"signature\":\"3KRXpjNqp21TAxeJc6u5ffn8JCdZTMqeyEse9wVmdd9my5EPyaHSoRWdK7Xhzg8D7oXEZVKigT6FihkNdxA1GU3P\",\"senderPublicKey\":\"ACrdghi6PDpLn158GQ7SNieaHeJEDiDCZmCPshTstUzx\",\"assetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"feeAssetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"timestamp\":1513416537167,\"amount\":1000000,\"fee\":10000000,\"recipient\":\"3PQ6wCS3zAkDEJtvGntQZbjuLw24kxTqndr\",\"attachment\":\"X9RJU4oxDGVzoc6bBDBZr6z1NT9UtZcGhKmTLZDp8QL55B4NkMzK6YKJwtZAP3H5ofj6bTvwm8fVKsouy7pkXXu6xuHr5L\"}]}",
		"{\"version\":2,\"generator\":\"11111111111111111111111111\",\"timestamp\":1485529182634,\"reference\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"desiredReward\":-1,\"nxt-consensus\":{\"base-target\":153722867,\"generation-signature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\"},\"transactionBlockLength\":1,\"transactionCount\":0,\"generatorPublicKey\":\"FM5ojNqW7e9cZ9zhPYGkpSP1Pcd8Z3e3MNKYVS5pGJ8Z\",\"signature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\",\"id\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\",\"height\":202}",
	}
	for i, blockJSON := range blockJSONs {
		var (
			tcNum = i + 1
			block = new(Block)
		)
		err := json.Unmarshal([]byte(blockJSON), block)
		require.NoError(t, err, "test case#%d", tcNum)
		actualJSON, err := json.Marshal(block)
		require.NoError(t, err, "test case#%d", tcNum)
		require.JSONEq(t, blockJSON, string(actualJSON), "test case#%d", tcNum)
	}
}
