package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type test struct {
	hexEncoded  string
	jsonEncoded string
	signature   string
}

var headerTests = []test{
	{
		hexEncoded:  "03000001605ea7b885a7e632b29f7b0ca842676bde33f83545f0530e0f228d38ce764a5bbabc5aed5dee2dc569e1cddd94741dd22e65e2ffb126bdbff1e010b839b5543d0511ca6f8100000028000000000000004dbda1dafbfe0e3d00f0ccc829a28fbd257db8dad50e9dda45b958551e09223408000001250000000100000002000100026a2a33a9933f467c7bb9d642fb7c981fd1044991342c7151f930b943a9e7621f83d4ecd5f1469f2143fb84b216d3553a31f766fc00cf71258a9afdc370722cc19b36553f94597b9d290acfba00a4ba4469d23edd0c06407c4d5ee88be3991587",
		jsonEncoded: `{"version":3,"timestamp":1513416538245,"reference":"4MhRMRYAteqrTDiBpkj7kqwmrMAQjwJc1vkPPacwgvaLQfsyyBg2AoJRrqV3cfxVd9iKofBY4S8jMV1NxAEzfgxp","features":[1,2],"desiredReward":-1,"nxt-consensus":{"base-target":77,"generation-signature":"DmFCdtLsrkMx6yrFohxD3wSqJbJcURszuQQ3V51B5dy9"},"transactionBlockLength":293,"transactionCount":1,"generatorPublicKey":"89RYHiy2HD9GLfznD9NpXwuY28PDGXVhmpTJ6J7BhneA","signature":"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2","id":"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2"}`,
	},
	{
		hexEncoded:  "0200000159e07071aaf5a2d8bdd2e65a6e29e9c06f9d8ba2b4c55dfa47d692f0893efd822ff820b30d008702da37fa99e9650c8d7bdff20c9293aeb846bf2dbace98e3f390787bca8d000000280000000009299ff3a11fafdcf909d719cc5d739d5910307308eb26de54d0ee4bcabe3ac3dc450dc50000000100d528aabec35ca100d87c7b7a128632faf19cd44531819457445113a32a21ef22331a903084e7288f2c61ae6548b54683632bfcfe4a8d63b39e4901b8699e1a7b1c180288b30439c8d58354e3d054312be866a89986ee23b7e23fd224777ac282",
		jsonEncoded: "{\"version\":2,\"timestamp\":1485529182634,\"reference\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"desiredReward\":-1,\"nxt-consensus\":{\"base-target\":153722867,\"generation-signature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\"},\"transactionBlockLength\":1,\"transactionCount\":0,\"generatorPublicKey\":\"FM5ojNqW7e9cZ9zhPYGkpSP1Pcd8Z3e3MNKYVS5pGJ8Z\",\"signature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\",\"id\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\"}",
	},
}

var blockTests = []test{
	{
		hexEncoded:  "03000001605ea7b885a7e632b29f7b0ca842676bde33f83545f0530e0f228d38ce764a5bbabc5aed5dee2dc569e1cddd94741dd22e65e2ffb126bdbff1e010b839b5543d0511ca6f8100000028000000000000004dbda1dafbfe0e3d00f0ccc829a28fbd257db8dad50e9dda45b958551e0922340800000125000000010000011d0473ebd754b8c89cd85e171c735b3b6d988f4c7e2a83d1f373fe3cd5a0b434ebf68b65fcb225453bcd50e6a1e04e9bbdf0f99bf9101e4aadba4aa486ad614bdb0e0488c2b0dd21c17e271f122ac2f1d2b341f59206f1e5ad0bfe7977f83fe76c804501fc80b686cd167170def6c3e81bbf91e645a23770d287877ea54564b8c0914b7a01fc80b686cd167170def6c3e81bbf91e645a23770d287877ea54564b8c0914b7a000001605ea7b44f00000000000f424000000000009896800157f338fcd6eaf626424f01ef207a300ef9573e3f1dae8dcd6d004534343566333832656164393437393261393637373864313431623835363639393062376134636266396633326530353334393964323937366138663534383661204968a33f00000002000100026a2a33a9933f467c7bb9d642fb7c981fd1044991342c7151f930b943a9e7621f83d4ecd5f1469f2143fb84b216d3553a31f766fc00cf71258a9afdc370722cc19b36553f94597b9d290acfba00a4ba4469d23edd0c06407c4d5ee88be3991587",
		jsonEncoded: "{\"version\":3,\"timestamp\":1513416538245,\"reference\":\"4MhRMRYAteqrTDiBpkj7kqwmrMAQjwJc1vkPPacwgvaLQfsyyBg2AoJRrqV3cfxVd9iKofBY4S8jMV1NxAEzfgxp\",\"features\":[1,2],\"desiredReward\":-1,\"nxt-consensus\":{\"base-target\":77,\"generation-signature\":\"DmFCdtLsrkMx6yrFohxD3wSqJbJcURszuQQ3V51B5dy9\"},\"transactionBlockLength\":293,\"transactionCount\":1,\"generatorPublicKey\":\"89RYHiy2HD9GLfznD9NpXwuY28PDGXVhmpTJ6J7BhneA\",\"signature\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\",\"id\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\",\"transactions\":[{\"type\":4,\"version\":1,\"id\":\"HFjhY9wh9DRrTUaUZoXreLNbN8TXSSBuDkRqeoHZ3c8i\",\"signature\":\"3KRXpjNqp21TAxeJc6u5ffn8JCdZTMqeyEse9wVmdd9my5EPyaHSoRWdK7Xhzg8D7oXEZVKigT6FihkNdxA1GU3P\",\"senderPublicKey\":\"ACrdghi6PDpLn158GQ7SNieaHeJEDiDCZmCPshTstUzx\",\"assetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"feeAssetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"timestamp\":1513416537167,\"amount\":1000000,\"fee\":10000000,\"recipient\":\"3PQ6wCS3zAkDEJtvGntQZbjuLw24kxTqndr\",\"attachment\":\"X9RJU4oxDGVzoc6bBDBZr6z1NT9UtZcGhKmTLZDp8QL55B4NkMzK6YKJwtZAP3H5ofj6bTvwm8fVKsouy7pkXXu6xuHr5L\"}]}",
		signature:   "3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2",
	},
	{
		hexEncoded:  "0200000159e07071aaf5a2d8bdd2e65a6e29e9c06f9d8ba2b4c55dfa47d692f0893efd822ff820b30d008702da37fa99e9650c8d7bdff20c9293aeb846bf2dbace98e3f390787bca8d000000280000000009299ff3a11fafdcf909d719cc5d739d5910307308eb26de54d0ee4bcabe3ac3dc450dc50000000100d528aabec35ca100d87c7b7a128632faf19cd44531819457445113a32a21ef22331a903084e7288f2c61ae6548b54683632bfcfe4a8d63b39e4901b8699e1a7b1c180288b30439c8d58354e3d054312be866a89986ee23b7e23fd224777ac282",
		jsonEncoded: "{\"version\":2,\"timestamp\":1485529182634,\"reference\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"desiredReward\":-1,\"nxt-consensus\":{\"base-target\":153722867,\"generation-signature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\"},\"transactionBlockLength\":1,\"transactionCount\":0,\"generatorPublicKey\":\"FM5ojNqW7e9cZ9zhPYGkpSP1Pcd8Z3e3MNKYVS5pGJ8Z\",\"signature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\",\"id\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\"}",
		signature:   "22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT",
	},
	{
		hexEncoded:  "040000016d81652266095c1e11ec1eaec2a9dccb0853b446db84a68af68bb764cc4d26355fa0cf87bdbb01203963d71ba9ea65535bc44b3deb2c9092b918edcd46bf69456d789a8802000000280000000000000f529b5c33b3d589e42366024456e870180fe1097edb7619217235c26f515cfa88ab0000000400000000000000000000000029b927004d4b45109482931d0774585e610e9bce168f70070339d673ecd6a3047275415496924d7f841d3ccb45023c5fc6e6af9dafbbdd5c2a60ed48873e08a288b012f0853c117503eeec3f4bdac248027918c475ad59b05caeddd38dcfafccc8cf4f04",
		jsonEncoded: "{\"version\":4,\"timestamp\":1569833951846,\"reference\":\"BrWuVmpSvbLBSAb6juXXcy9w81dCrU4ykvKTpu3T6KJe8VbRvKFnphMECYDVQvBbViLjeVSEmWoFYp6DS9hy6ND\",\"desiredReward\":700000000,\"nxt-consensus\":{\"base-target\":3922,\"generation-signature\":\"BTTjkPdMoUexBcwgLGwyHT1YSctWA8TiW2MSxnUjKMWz\"},\"transactionBlockLength\":4,\"transactionCount\":0,\"generatorPublicKey\":\"6CixnBTJeWC85SvqrwXUpquYW57PRPGyumcPYtMcqgZh\",\"signature\":\"41c1RfETCxmLkJJuUQQE5kaGoEKRHztG6vjUtSk17AUNdJvDH6tHRpCxZAZG1b77QFsSx4zRk5aJUre2jFsa4Vfq\"}",
		signature:   "41c1RfETCxmLkJJuUQQE5kaGoEKRHztG6vjUtSk17AUNdJvDH6tHRpCxZAZG1b77QFsSx4zRk5aJUre2jFsa4Vfq",
	},
}

func makeBlock(t *testing.T) *Block {
	decoded, err := hex.DecodeString(blockTests[0].hexEncoded)
	assert.NoError(t, err, "hex.DecodeString failed")
	var block Block
	err = block.UnmarshalBinary(decoded, MainNetScheme)
	assert.NoError(t, err, "block.UnmarshalBinary failed")
	return &block
}

func blockFromProtobufToProtobuf(t *testing.T, hexStr string) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var block Block
	err = block.UnmarshalBinary(decoded, MainNetScheme)
	assert.NoError(t, err, "UnmarshalBinary() failed")
	protobuf, err := block.MarshalToProtobuf(MainNetScheme)
	assert.NoError(t, err, "MarshalToProtobuf() failed")
	var res Block
	err = res.UnmarshalFromProtobuf(protobuf)
	assert.NoError(t, err)
	assert.Equal(t, block, res)
}

func blockFromBinaryToBinary(t *testing.T, hexStr, jsonStr string) {
	const scheme = MainNetScheme

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var b Block
	err = b.UnmarshalBinary(decoded, scheme)
	assert.NoError(t, err, "UnmarshalBinary() for block failed")
	bts, err := json.Marshal(&b)
	assert.NoError(t, err, "json.Marshal() for block failed")
	str := string(bts)
	assert.Equalf(t, jsonStr, str, "block marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
	bin, err := b.MarshalBinary(scheme)
	assert.NoError(t, err, "MarshalBinary() for block failed")
	assert.Equal(t, decoded, bin, "bin for block differs")
}

func blockFromJSONToJSON(t *testing.T, jsonStr string) {
	var b Block
	err := json.Unmarshal([]byte(jsonStr), &b)
	assert.NoError(t, err, "json.Unmarshal() for block failed")
	bts, err := json.Marshal(&b)
	assert.NoError(t, err, "json.Marshal() for block failed")
	str := string(bts)
	assert.JSONEqf(t, jsonStr, str, "block marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
}

func headerFromProtobufToProtobuf(t *testing.T, hexStr string) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var header BlockHeader
	err = header.UnmarshalHeaderFromBinary(decoded, MainNetScheme)
	assert.NoError(t, err, "UnmarshalHeaderFromBinary() failed")
	if header.TransactionCount != 0 {
		// Protobuf headers do not store transaction count.
		return
	}
	protobuf, err := header.MarshalHeaderToProtobuf(MainNetScheme)
	assert.NoError(t, err, "MarshalHeaderToProtobuf() failed")
	var res Block
	err = res.UnmarshalFromProtobuf(protobuf)
	assert.NoError(t, err)
	assert.Equal(t, header, res.BlockHeader)
}

func headerFromBinaryToBinary(t *testing.T, hexStr, jsonStr string) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var header BlockHeader
	err = header.UnmarshalHeaderFromBinary(decoded, MainNetScheme)
	assert.NoError(t, err, "UnmarshalHeaderFromBinary() failed")
	bts, err := json.Marshal(header)
	assert.NoError(t, err, "json.Marshal() for header failed")
	str := string(bts)
	assert.Equalf(t, jsonStr, str, "header marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
	bin, err := header.MarshalHeaderToBinary()
	assert.NoError(t, err, "MarshalHeaderToBinary() failed")
	assert.Equal(t, hexStr, hex.EncodeToString(bin), "hex for header differs")
}

func headerFromJSONToJSON(t *testing.T, jsonStr string) {
	var header BlockHeader
	err := json.Unmarshal([]byte(jsonStr), &header)
	assert.NoError(t, err, "json.Unmarshal() for header failed")
	bts, err := json.Marshal(header)
	assert.NoError(t, err, "json.Marshal() for header failed")
	str := string(bts)
	assert.JSONEqf(t, jsonStr, str, "header marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
}

func TestHeaderSerialization(t *testing.T) {
	for i, v := range headerTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			headerFromProtobufToProtobuf(t, v.hexEncoded)
			headerFromBinaryToBinary(t, v.hexEncoded, v.jsonEncoded)
			headerFromJSONToJSON(t, v.jsonEncoded)
		})
	}
}

func TestAppendHeaderBytesToTransactions(t *testing.T) {
	const scheme = TestNetScheme

	block := makeBlock(t)
	headerBytes, err := block.MarshalHeaderToBinary()
	assert.NoError(t, err, "MarshalHeaderToBinary() failed")
	transactions := block.Transactions
	blockBytes, err := block.MarshalBinary(scheme)
	assert.NoError(t, err, "block.MarshalBinary() failed")
	transactionsBts, err := transactions.MarshalBinary(scheme)
	assert.NoError(t, err)
	blockBytes1, err := AppendHeaderBytesToTransactions(headerBytes, transactionsBts)
	assert.NoError(t, err, "AppendHeaderBytesToTransactions() failed")
	assert.Equal(t, blockBytes, blockBytes1)
}

func TestBlockSerialization(t *testing.T) {
	for i, v := range blockTests[:2] {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			blockFromProtobufToProtobuf(t, v.hexEncoded)
			blockFromBinaryToBinary(t, v.hexEncoded, v.jsonEncoded)
			blockFromJSONToJSON(t, v.jsonEncoded)
		})
	}
}

func TestBlockGetSignature(t *testing.T) {
	for i, v := range blockTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			decoded, err := hex.DecodeString(v.hexEncoded)
			if err != nil {
				t.Fatal(err)
			}
			rs, err := BlockGetSignature(decoded)
			if err != nil {
				t.Fatal(err)
			}
			require.Equal(t, v.signature, rs.String())
		})
	}
}

func TestTransactions_WriteToBinary(t *testing.T) {
	const scheme = TestNetScheme

	secret, public, err := crypto.GenerateKeyPair([]byte("test"))
	assert.NoError(t, err)
	alias, err := NewAliasFromString("alias:T:aaaa")
	require.NoError(t, err)
	createAlias := NewUnsignedCreateAliasWithSig(public, *alias, 10000, NewTimestampFromTime(time.Now()))
	require.NoError(t, createAlias.Sign(scheme, secret))
	bts, _ := createAlias.MarshalBinary(scheme)

	buf := new(bytes.Buffer)
	ts := Transactions{createAlias}

	_, err = ts.WriteToBinary(buf, scheme)
	require.NoError(t, err)

	length := binary.BigEndian.Uint32(buf.Bytes()[:4])
	require.EqualValues(t, length, len(bts))
	require.Equal(t, buf.Bytes()[4:], bts)
}

func TestBlock_WriteTo(t *testing.T) {
	const scheme = TestNetScheme

	sig, err := crypto.NewSignatureFromBase58("2kcBqiM5y3DAtg8UrDp5X5dqhKUQ2cNSndZ98c7QMDWgXaz7g1gPGKyND16vSGYvoVN2UqxNk9dSonJUqWmjE5Ee")
	require.NoError(t, err)
	parentSig, err := crypto.NewSignatureFromBase58("3ov5nyERRYrNd8Uun7nuUWYwztXL8jjt3Cbr5HMfsGhoXAKkctAYVVmUFChz95fPHKyrWopuaygdirQ4kMa3fkwJ")
	require.NoError(t, err)
	parent := NewBlockIDFromSignature(parentSig)
	gensig, err := base58.Decode("5fkwJc2yZVT2WLDxXs8qFJHdzb2FXji5MC3PDdAFC145")
	require.NoError(t, err)

	// transaction
	secret, public, err := crypto.GenerateKeyPair([]byte("test"))
	require.NoError(t, err)
	alias, err := NewAliasFromString("alias:T:aaaa")
	require.NoError(t, err)
	createAlias := NewUnsignedCreateAliasWithSig(public, *alias, 10000, NewTimestampFromTime(time.Now()))
	require.NoError(t, createAlias.Sign(scheme, secret))

	transactions := Transactions{createAlias}

	block := Block{
		BlockHeader: BlockHeader{
			Version:                3,
			Timestamp:              1558019400034,
			Parent:                 parent,
			FeaturesCount:          0,   // ??
			Features:               nil, // ??
			RewardVote:             -1,
			ConsensusBlockLength:   40, //  ??
			TransactionBlockLength: uint32(transactions.BinarySize() + 4),
			TransactionCount:       len(transactions),
			GeneratorPublicKey:     public,
			BlockSignature:         sig, //

			NxtConsensus: NxtConsensus{
				BaseTarget:   1010,   // 8
				GenSignature: gensig, //
			},
		},
		Transactions: transactions,
	}

	buf := new(bytes.Buffer)
	_, err = block.WriteToWithoutSignature(buf, scheme)
	require.NoError(t, err)
	marshaledBytes, _ := block.MarshalBinary(scheme)

	// writeTo doesn't write signature
	require.Equal(t, marshaledBytes[:len(marshaledBytes)-crypto.SignatureSize], buf.Bytes())
}

func TestBlock_Clone(t *testing.T) {
	var js = `{
  "reference": "37R8rEa1FKwebXyrdg2o8RL1wqeLwRgy78JQSJFWCzgurU8tEMomZGqLyCKQJRXsnDgE78N8V2Nk94yesr33dejA",
  "blocksize": 822,
  "signature": "ScjRq6fo6Dnegg1cShBZq5zD2ydvxJW5H6pfBFvcTLqDAFTMweu5VD8Y74DHkL1vWgYaS2zhQJQXTMrXgqHGHvt",
  "totalFee": 4,
  "nxt-consensus": {
    "base-target": 692299067,
    "generation-signature": "6LTMWYS5gr95gMTeeE7onQwfT6yHvNhZNwR2K8zkQtWe"
  },
  "fee": 4,
  "generator": "3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy",
  "transactionCount": 4,
  "transactions": [
    {
      "senderPublicKey": "8xmjhwv1BRuqtdomKWzgZ2J74SwN3nNSYYUp1PhCaDrj",
      "amount": 500000000,
      "sender": "3PHrvC7W13eZDCkE5u1DV4CbEoeHvPbt387",
      "feeAssetId": null,
      "signature": "ScPLwz1T5VRYfvUc4AxoHzab6HrqH73FjF2DceRrZ5SGWFAkTbDKPuef4WPyXtKftYAGkKVJmGJwNyA67mNPdfM",
      "proofs": [
        "ScPLwz1T5VRYfvUc4AxoHzab6HrqH73FjF2DceRrZ5SGWFAkTbDKPuef4WPyXtKftYAGkKVJmGJwNyA67mNPdfM"
      ],
      "fee": 1,
      "recipient": "3PQuyEy3LWjRCKq9JcDHvXfNapQnWHdXPZ3",
      "id": "ScPLwz1T5VRYfvUc4AxoHzab6HrqH73FjF2DceRrZ5SGWFAkTbDKPuef4WPyXtKftYAGkKVJmGJwNyA67mNPdfM",
      "type": 2,
      "timestamp": 1466335015875
    },
    {
      "senderPublicKey": "35CKZtLH9vrN9jFiPoZhKvMP8sdk2dm6ZukWUh3MJbgP",
      "amount": 110000000,
      "sender": "3PNUydgTUKBrJKyJbteuVJU5CrLeMMM8pbS",
      "feeAssetId": null,
      "signature": "3QyARB92kv1cRxfRyGrJKV7bTz6Dze6uyjQwagdzm8jvhfumbbyZb8oxM98EtCgUYr1kNYptYV3HvaDkseoev1Zn",
      "proofs": [
        "3QyARB92kv1cRxfRyGrJKV7bTz6Dze6uyjQwagdzm8jvhfumbbyZb8oxM98EtCgUYr1kNYptYV3HvaDkseoev1Zn"
      ],
      "fee": 1,
      "recipient": "3PDw3VxMiTKKykDaTyXeZ6xuprSUKs9pyk9",
      "id": "3QyARB92kv1cRxfRyGrJKV7bTz6Dze6uyjQwagdzm8jvhfumbbyZb8oxM98EtCgUYr1kNYptYV3HvaDkseoev1Zn",
      "type": 2,
      "timestamp": 1466335007548
    },
    {
      "senderPublicKey": "8ebcrtnt2a2Lyw6LK21XAHyy1thQKubunwT255RGVz5E",
      "amount": 400000000,
      "sender": "3PBj1yoVAKhcvGqvZHCBtmUrW4G6iuXgdr5",
      "feeAssetId": null,
      "signature": "5R8sj4P2tr5mNBaNDt1eK4utWxCviQ6z9XCR4PbdpFekrdkVUYZkLCGCECJz1rxvAACaQr4Bw6VYNmghG4xhg98R",
      "proofs": [
        "5R8sj4P2tr5mNBaNDt1eK4utWxCviQ6z9XCR4PbdpFekrdkVUYZkLCGCECJz1rxvAACaQr4Bw6VYNmghG4xhg98R"
      ],
      "fee": 1,
      "recipient": "3PD18NJNjUYHLRSeewWKZF8z4rnosTtun2K",
      "id": "5R8sj4P2tr5mNBaNDt1eK4utWxCviQ6z9XCR4PbdpFekrdkVUYZkLCGCECJz1rxvAACaQr4Bw6VYNmghG4xhg98R",
      "type": 2,
      "timestamp": 1466334987703
    },
    {
      "senderPublicKey": "46t5F1bUxG4mAQUiDyMKDBpWhHChLQSyhnVJ8R5jaLqH",
      "amount": 499999999,
      "sender": "3P31zvGdh6ai6JK6zZ18TjYzJsa1B83YPoj",
      "feeAssetId": null,
      "signature": "4sBHrSMdamRsxz6LAt3puk2mwBDwfEqBLmhEwTjb2nCGuFMrmJnGJQoqjV4KMnj821d6bZBSMrFHsNhyCuJmuRfE",
      "proofs": [
        "4sBHrSMdamRsxz6LAt3puk2mwBDwfEqBLmhEwTjb2nCGuFMrmJnGJQoqjV4KMnj821d6bZBSMrFHsNhyCuJmuRfE"
      ],
      "fee": 1,
      "recipient": "3P6DFn9briPL2mufCKoBTDDGCsus94gSQ56",
      "id": "4sBHrSMdamRsxz6LAt3puk2mwBDwfEqBLmhEwTjb2nCGuFMrmJnGJQoqjV4KMnj821d6bZBSMrFHsNhyCuJmuRfE",
      "type": 2,
      "timestamp": 1466335012046
    }
  ],
  "version": 2,
  "timestamp": 1466335031786
}`

	b1 := &Block{}
	err := json.Unmarshal([]byte(js), b1)
	require.NoError(t, err)

	b2 := b1.Clone()
	require.Equal(t, b1, b2)
}

func TestEmptyBlockMarshall(t *testing.T) {
	const scheme = TestNetScheme

	b1 := Block{}
	bts, err := b1.MarshalBinary(scheme)
	require.Error(t, err)

	b2 := Block{}
	err = b2.UnmarshalBinary(bts, scheme)
	require.Error(t, err)
}

func TestBlockVerifyRootHash(t *testing.T) {
	// Waves
	waves := NewOptionalAssetWaves()

	// Key pair
	secret, public, err := crypto.GenerateKeyPair([]byte("test"))
	require.NoError(t, err)

	// Addresses
	addr, err := NewAddressFromPublicKey(TestNetScheme, public)
	require.NoError(t, err)
	recipient := NewRecipientFromAddress(addr)

	// Transactions
	tx1 := NewUnsignedTransferWithProofs(MaxTransferTransactionVersion, public, waves, waves, 1, 2, 3, recipient, []byte("aaa"))
	err = tx1.Sign(TestNetScheme, secret)
	assert.NoError(t, err)
	tx2 := NewUnsignedTransferWithProofs(MaxTransferTransactionVersion, public, waves, waves, 4, 5, 6, recipient, []byte("bbb"))
	err = tx2.Sign(TestNetScheme, secret)
	assert.NoError(t, err)

	txs1 := Transactions{tx1}
	txs2 := Transactions{tx1, tx2}

	// Signatures
	parentSig, err := crypto.NewSignatureFromBase58("3ov5nyERRYrNd8Uun7nuUWYwztXL8jjt3Cbr5HMfsGhoXAKkctAYVVmUFChz95fPHKyrWopuaygdirQ4kMa3fkwJ")
	require.NoError(t, err)
	parent := NewBlockIDFromSignature(parentSig)
	gs, err := base58.Decode("5fkwJc2yZVT2WLDxXs8qFJHdzb2FXji5MC3PDdAFC145")
	require.NoError(t, err)

	// Block
	block := Block{
		BlockHeader: BlockHeader{
			Version:              ProtobufBlockVersion,
			Timestamp:            1,
			Parent:               parent,
			FeaturesCount:        0,
			Features:             nil,
			RewardVote:           -1,
			ConsensusBlockLength: 40,
			NxtConsensus: NxtConsensus{
				BaseTarget:   1010,
				GenSignature: gs,
			},
			TransactionBlockLength: uint32(txs2.BinarySize() + 4),
			TransactionCount:       len(txs2),
			GeneratorPublicKey:     public,
		},
		Transactions: txs2,
	}
	err = block.SetTransactionsRoot(TestNetScheme)
	require.NoError(t, err)

	err = block.Sign(TestNetScheme, secret)
	require.NoError(t, err)

	ok, err := block.VerifySignature(TestNetScheme)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = block.VerifyTransactionsRoot(TestNetScheme)
	require.NoError(t, err)
	assert.True(t, ok)

	block.Transactions = txs1
	ok, err = block.VerifySignature(TestNetScheme)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = block.VerifyTransactionsRoot(TestNetScheme)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestBlockAfterLightNodeFeature(t *testing.T) {
	const (
		blockchainScheme           = 'D'
		blockBytesWithChallengeHex = "0af90308441220b9a269272e037e7f90c1af5076c46a46ec11bfc1ff42913393d28e94aea79c23188003226063bd0a3508afd1c5fb0699be3e775ed7d17e724a4941321c6ec5c2bc09b00e1978faaffae91070ffa226e5d01b2af4bea2dddbbf7f6355fda601c42fe1f5980d5301f48d57e39f79d5d54c81e84098b8102b1ba94e556af64fd62b310da7ba0a3086fec3b6d1313805422097a186a5a964dddb69438f7039775d516ac303ab4606c6d2b155c7f39f5136764880cab5ee0152205f5b2e36e4986503fb1aaf8ca33fb7f80a866885f23f34ccdeccd05878458c9a5a20271c305e4823797bb532788d8b0f8273b7d91c4df3632337c8edf75b1dbe559262f8010880031260805bddec5cd6dc210dca38828069e55eaf0fd40408352e6a12cdf0aabb1966725754ee2cc7b47058d650f9e90d656a89e19144aa7a1f934239705120fca6f405f9fe782016695711e3eaefa4aa7eda137a240414e6379abf849aad7bbb4ba60a2085cdc4b6d1312a2099d7b4bd83fdc2fe32f8ccefef9437a0d5fa89093eb0b557ded6554167eb6e363080cab5ee013a20010101010101010101010101010101010101010101010101010101010101010142404a2ac66cfb02c74a117266135bc6295215026c323d646349fad638b38786092d48057978eaabf3c949a37175965df5dc18ac36d3104ad6ba583bc59ffb836b0c1240839fbb955249fa15997c87397abf7a5fccdb705e46e5307e183c14df78becec4e1bdc25a1e8641954d364f9352281ee76b2104e6d734df53564c3e64642ef90a1aa3010a5f0844122070a0b097af1813f2571c759c4c72ccc92bc6c8c1b847f437c106d68a930b517c1a0410a0c21e20b7d3c4b6d1312802a207290a160a14f9b561bba9e294b1f79f3e29fcaa066bd05c4da0120f0109010000000463616c6c0000000012403eab127938c969d4c928f602c2a0fecd5eb0705fd2e7feff66a01a2f641879a1b2b3cf58fa0f08993a5439331d11fec68d7890792c81742dcea2f30d4ebfc4061a96010a520844122070a0b097af1813f2571c759c4c72ccc92bc6c8c1b847f437c106d68a930b517c1a0410a08d0620a6d5c4b6d1312803c2061c0a160a14f9b561bba9e294b1f79f3e29fcaa066bd05c4da0120210011240a84c703028e852a1faf59c8defd8c560b2f84bafa0f7db305def6ec6b17db08f1f31f29f2b3a0da1d2ac6b424f5c58de2dcb0f5a3f9d3bd963246cb223705502" //nolint:lll
	)
	blockBytes, err := hex.DecodeString(blockBytesWithChallengeHex)
	require.NoError(t, err)

	block := new(Block)
	err = block.UnmarshalFromProtobuf(blockBytes)
	require.NoError(t, err, "failed to unmarshal block from protobuf")

	t.Run("MarshalToProtobuf", func(t *testing.T) {
		marshaledBlockBytes, mErr := block.MarshalToProtobuf(blockchainScheme)
		require.NoError(t, mErr)
		require.Equal(t, blockBytes, marshaledBlockBytes)
	})

	t.Run("VerifySignature", func(t *testing.T) {
		ok, vErr := block.VerifySignature(blockchainScheme)
		require.NoError(t, vErr)
		require.True(t, ok)
	})
}

func TestBlockID_ReadFrom(t *testing.T) {
	for i, tc := range []struct {
		data string
		exp  string
		n    int64
		err  error
	}{
		{"", "", 0, ErrInvalidBlockIDDataSize},
		{"000102", "", 3, ErrInvalidBlockIDDataSize},
		{"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 32, nil},
		{"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 48, nil},
		{"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f" +
			"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1GMkH3brNXiNNs1tiFZHtUoRG8yzjuLTbgUVK7TPtAgWVQRKi4nR5nhQURVuXYLRpG22QW5uWmkK6SHHKZ7WSa",
			64, nil},
		{"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f" +
			"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1GMkH3brNXiNNs1tiFZHtUoRG8yzjuLTbgUVK7TPtAgWVQRKi4nR5nhQURVuXYLRpG22QW5uWmkK6SHHKZ7WSa",
			64, nil}, // Excess data left unread in reader.
	} {
		t.Run(fmt.Sprintf("Case_%d", i+1), func(t *testing.T) {
			data := hex.NewDecoder(newSlowReader(strings.NewReader(tc.data)))

			var id BlockID
			n, err := id.ReadFrom(data)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.n, n)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, id.String())
			assert.Equal(t, tc.n, n)
		})
	}
}

func TestBlockID_ReadFrom_Initialized(t *testing.T) {
	for i, tc := range []struct {
		idType BlockIDType
		data   string
		exp    string
		n      int64
		err    error
	}{
		{SignatureID, "", "", 0, ErrInvalidBlockIDDataSize},
		{DigestID, "", "", 0, ErrInvalidBlockIDDataSize},
		{SignatureID, "000102", "", 3, ErrInvalidBlockIDDataSize},
		{DigestID, "000102", "", 3, ErrInvalidBlockIDDataSize},
		{SignatureID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"", 32, ErrInvalidBlockIDDataSize},
		{DigestID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 32, nil},
		{SignatureID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f" +
			"000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 48, ErrInvalidBlockIDDataSize},
		{DigestID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f" +
			"000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 32, nil},
		{SignatureID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f" +
			"000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1GMkH3brNXiNNs1tiFZHtUoRG8yzjuLTbgUVK7TPtAgWVQRKi4nR5nhQURVuXYLRpG22QW5uWmkK6SHHKZ7WSa",
			64, nil},
		{DigestID, "000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0" +
			"00102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f",
			"1thX6LZfHDZZKUs92febWaf4WJZnsKRiVwJusXxB7L", 32, nil},
	} {
		t.Run(fmt.Sprintf("Case_%d", i+1), func(t *testing.T) {
			data := newSlowReader(hex.NewDecoder(strings.NewReader(tc.data)))

			id := BlockID{idType: tc.idType}
			n, err := id.ReadFrom(data)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.n, n)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, id.String())
			assert.Equal(t, tc.n, n)
		})
	}
}

// slowReader is a reader that reads only 2 bytes at a time.
type slowReader struct {
	r io.Reader
}

func newSlowReader(r io.Reader) *slowReader {
	return &slowReader{r: r}
}
func (r *slowReader) Read(p []byte) (int, error) {
	return io.ReadFull(r.r, p[:2])
}
