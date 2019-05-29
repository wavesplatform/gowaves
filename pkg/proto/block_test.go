package proto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

type test struct {
	hexEncoded  string
	jsonEncoded string
	signature   string
}

var headerTests = []test{
	{
		hexEncoded:  "03000001605ea7b885a7e632b29f7b0ca842676bde33f83545f0530e0f228d38ce764a5bbabc5aed5dee2dc569e1cddd94741dd22e65e2ffb126bdbff1e010b839b5543d0511ca6f8100000028000000000000004dbda1dafbfe0e3d00f0ccc829a28fbd257db8dad50e9dda45b958551e09223408000001250000000100000002000100026a2a33a9933f467c7bb9d642fb7c981fd1044991342c7151f930b943a9e7621f83d4ecd5f1469f2143fb84b216d3553a31f766fc00cf71258a9afdc370722cc19b36553f94597b9d290acfba00a4ba4469d23edd0c06407c4d5ee88be3991587",
		jsonEncoded: "{\"version\":3,\"timestamp\":1513416538245,\"reference\":\"4MhRMRYAteqrTDiBpkj7kqwmrMAQjwJc1vkPPacwgvaLQfsyyBg2AoJRrqV3cfxVd9iKofBY4S8jMV1NxAEzfgxp\",\"features\":[1,2],\"nxt-consensus\":{\"base-target\":77,\"generation-signature\":\"DmFCdtLsrkMx6yrFohxD3wSqJbJcURszuQQ3V51B5dy9\"},\"transactionBlockLength\":293,\"transactionCount\":1,\"signature\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\"}",
	},
	{
		hexEncoded:  "0200000159e07071aaf5a2d8bdd2e65a6e29e9c06f9d8ba2b4c55dfa47d692f0893efd822ff820b30d008702da37fa99e9650c8d7bdff20c9293aeb846bf2dbace98e3f390787bca8d000000280000000009299ff3a11fafdcf909d719cc5d739d5910307308eb26de54d0ee4bcabe3ac3dc450dc50000000100d528aabec35ca100d87c7b7a128632faf19cd44531819457445113a32a21ef22331a903084e7288f2c61ae6548b54683632bfcfe4a8d63b39e4901b8699e1a7b1c180288b30439c8d58354e3d054312be866a89986ee23b7e23fd224777ac282",
		jsonEncoded: "{\"version\":2,\"timestamp\":1485529182634,\"reference\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"nxt-consensus\":{\"base-target\":153722867,\"generation-signature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\"},\"transactionBlockLength\":1,\"transactionCount\":0,\"signature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\"}",
	},
}

var blockTests = []test{
	{
		hexEncoded:  "03000001605ea7b885a7e632b29f7b0ca842676bde33f83545f0530e0f228d38ce764a5bbabc5aed5dee2dc569e1cddd94741dd22e65e2ffb126bdbff1e010b839b5543d0511ca6f8100000028000000000000004dbda1dafbfe0e3d00f0ccc829a28fbd257db8dad50e9dda45b958551e0922340800000125000000010000011d0473ebd754b8c89cd85e171c735b3b6d988f4c7e2a83d1f373fe3cd5a0b434ebf68b65fcb225453bcd50e6a1e04e9bbdf0f99bf9101e4aadba4aa486ad614bdb0e0488c2b0dd21c17e271f122ac2f1d2b341f59206f1e5ad0bfe7977f83fe76c804501fc80b686cd167170def6c3e81bbf91e645a23770d287877ea54564b8c0914b7a01fc80b686cd167170def6c3e81bbf91e645a23770d287877ea54564b8c0914b7a000001605ea7b44f00000000000f424000000000009896800157f338fcd6eaf626424f01ef207a300ef9573e3f1dae8dcd6d004534343566333832656164393437393261393637373864313431623835363639393062376134636266396633326530353334393964323937366138663534383661204968a33f00000002000100026a2a33a9933f467c7bb9d642fb7c981fd1044991342c7151f930b943a9e7621f83d4ecd5f1469f2143fb84b216d3553a31f766fc00cf71258a9afdc370722cc19b36553f94597b9d290acfba00a4ba4469d23edd0c06407c4d5ee88be3991587",
		jsonEncoded: "{\"version\":3,\"timestamp\":1513416538245,\"reference\":\"4MhRMRYAteqrTDiBpkj7kqwmrMAQjwJc1vkPPacwgvaLQfsyyBg2AoJRrqV3cfxVd9iKofBY4S8jMV1NxAEzfgxp\",\"features\":[1,2],\"nxt-consensus\":{\"base-target\":77,\"generation-signature\":\"DmFCdtLsrkMx6yrFohxD3wSqJbJcURszuQQ3V51B5dy9\"},\"transactionBlockLength\":293,\"transactionCount\":1,\"signature\":\"3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2\",\"transactions\":[{\"type\":4,\"version\":1,\"id\":\"HFjhY9wh9DRrTUaUZoXreLNbN8TXSSBuDkRqeoHZ3c8i\",\"signature\":\"3KRXpjNqp21TAxeJc6u5ffn8JCdZTMqeyEse9wVmdd9my5EPyaHSoRWdK7Xhzg8D7oXEZVKigT6FihkNdxA1GU3P\",\"senderPublicKey\":\"ACrdghi6PDpLn158GQ7SNieaHeJEDiDCZmCPshTstUzx\",\"assetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"feeAssetId\":\"HzfaJp8YQWLvQG4FkUxq2Q7iYWMYQ2k8UF89vVJAjWPj\",\"timestamp\":1513416537167,\"amount\":1000000,\"fee\":10000000,\"recipient\":\"3PQ6wCS3zAkDEJtvGntQZbjuLw24kxTqndr\",\"attachment\":\"X9RJU4oxDGVzoc6bBDBZr6z1NT9UtZcGhKmTLZDp8QL55B4NkMzK6YKJwtZAP3H5ofj6bTvwm8fVKsouy7pkXXu6xuHr5L\"}]}",
		signature:   "3dsdFaMqVKpJhBUYYYYwP8DkpHVivhn8AqG22kRSryiAmXFcDB31SEMyH4t38ihxk79QcFiPXUy3w1aWbddcW5k2",
	},
	{
		hexEncoded:  "0200000159e07071aaf5a2d8bdd2e65a6e29e9c06f9d8ba2b4c55dfa47d692f0893efd822ff820b30d008702da37fa99e9650c8d7bdff20c9293aeb846bf2dbace98e3f390787bca8d000000280000000009299ff3a11fafdcf909d719cc5d739d5910307308eb26de54d0ee4bcabe3ac3dc450dc50000000100d528aabec35ca100d87c7b7a128632faf19cd44531819457445113a32a21ef22331a903084e7288f2c61ae6548b54683632bfcfe4a8d63b39e4901b8699e1a7b1c180288b30439c8d58354e3d054312be866a89986ee23b7e23fd224777ac282",
		jsonEncoded: "{\"version\":2,\"timestamp\":1485529182634,\"reference\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"nxt-consensus\":{\"base-target\":153722867,\"generation-signature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\"},\"transactionBlockLength\":1,\"transactionCount\":0,\"signature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\"}",
		signature:   "22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT",
	},
}

func makeBlock(t *testing.T) *Block {
	decoded, err := hex.DecodeString(blockTests[0].hexEncoded)
	assert.NoError(t, err, "hex.DecodeString failed")
	var block Block
	err = block.UnmarshalBinary(decoded)
	assert.NoError(t, err, "block.UnmarshalBinary failed")
	return &block
}

func blockFromBinaryToBinary(t *testing.T, hexStr, jsonStr string) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var b Block
	err = b.UnmarshalBinary(decoded)
	assert.NoError(t, err, "UnmarshalBinary() for block failed")
	bytes, err := json.Marshal(b)
	assert.NoError(t, err, "json.Marshal() for block failed")
	str := string(bytes)
	assert.Equalf(t, jsonStr, str, "block marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
	bin, err := b.MarshalBinary()
	assert.NoError(t, err, "MarshalBinary() for block failed")
	assert.Equal(t, decoded, bin, "bin for block differs")
}

func blockFromJSONToJSON(t *testing.T, jsonStr string) {
	var b Block
	err := json.Unmarshal([]byte(jsonStr), &b)
	assert.NoError(t, err, "json.Unmarshal() for block failed")
	bytes, err := json.Marshal(b)
	assert.NoError(t, err, "json.Marshal() for block failed")
	str := string(bytes)
	assert.Equalf(t, jsonStr, str, "block marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
}

func headerFromBinaryToBinary(t *testing.T, hexStr, jsonStr string) {
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	var header BlockHeader
	err = header.UnmarshalHeaderFromBinary(decoded)
	assert.NoError(t, err, "UnmarshalHeaderFromBinary() failed")
	bytes, err := json.Marshal(header)
	assert.NoError(t, err, "json.Marshal() for header failed")
	str := string(bytes)
	assert.Equalf(t, jsonStr, str, "header marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
	bin, err := header.MarshalHeaderToBinary()
	assert.NoError(t, err, "MarshalHeaderToBinary() failed")
	assert.Equal(t, hexStr, hex.EncodeToString(bin), "hex for header differs")
}

func headerFromJSONToJSON(t *testing.T, jsonStr string) {
	var header BlockHeader
	err := json.Unmarshal([]byte(jsonStr), &header)
	assert.NoError(t, err, "json.Unmarshal() for header failed")
	bytes, err := json.Marshal(header)
	assert.NoError(t, err, "json.Marshal() for header failed")
	str := string(bytes)
	assert.Equalf(t, jsonStr, str, "header marshaled to wrong json:\nhave: %s\nwant: %s", str, jsonStr)
}

func TestHeaderSerialization(t *testing.T) {
	for i, v := range headerTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			headerFromBinaryToBinary(t, v.hexEncoded, v.jsonEncoded)
			headerFromJSONToJSON(t, v.jsonEncoded)
		})
	}
}

func TestAppendHeaderBytesToTransactions(t *testing.T) {
	block := makeBlock(t)
	headerBytes, err := block.MarshalHeaderToBinary()
	assert.NoError(t, err, "MarshalHeaderToBinary() failed")
	transactions := block.Transactions
	blockBytes, err := block.MarshalBinary()
	assert.NoError(t, err, "block.MarshalBinary() failed")
	blockBytes1, err := AppendHeaderBytesToTransactions(headerBytes, transactions)
	assert.NoError(t, err, "AppendHeaderBytesToTransactions() failed")
	assert.Equal(t, blockBytes, blockBytes1)
}

func TestBlockSerialization(t *testing.T) {
	for i, v := range blockTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
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
