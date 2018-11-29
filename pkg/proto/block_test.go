package proto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
)

type blockTest struct {
	hexEncoded  string
	jsonEncoded string
}

var blockTests = []blockTest{
	{
		hexEncoded:  "0200000159e07071aaf5a2d8bdd2e65a6e29e9c06f9d8ba2b4c55dfa47d692f0893efd822ff820b30d008702da37fa99e9650c8d7bdff20c9293aeb846bf2dbace98e3f390787bca8d000000280000000009299ff3a11fafdcf909d719cc5d739d5910307308eb26de54d0ee4bcabe3ac3dc450dc50000000100d528aabec35ca100d87c7b7a128632faf19cd44531819457445113a32a21ef22331a903084e7288f2c61ae6548b54683632bfcfe4a8d63b39e4901b8699e1a7b1c180288b30439c8d58354e3d054312be866a89986ee23b7e23fd224777ac282",
		jsonEncoded: "{\"Version\":2,\"Timestamp\":1485529182634,\"Parent\":\"5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa\",\"ConsensusBlockLength\":40,\"BaseTarget\":153722867,\"GenSignature\":\"BqxfUrYe27eJf96JnSEu7zG76V54gh3gydy5ZxnVaaV2\",\"TransactionBlockLength\":1,\"TransactionCount\":0,\"GenPublicKey\":\"FM5ojNqW7e9cZ9zhPYGkpSP1Pcd8Z3e3MNKYVS5pGJ8Z\",\"BlockSignature\":\"22G6NgN3PgcjYsgWmkpkNHQV6eZiYecRtSt6kNXuFwxDDC3CSLkP11WY3HzkdgeVxW9dfyF2FUypfBXTFLxrTxoT\",\"Height\":0}",
	},
}

func TestBlockMarshaling(t *testing.T) {
	for i, v := range blockTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			decoded, err := hex.DecodeString(v.hexEncoded)
			if err != nil {
				t.Fatal(err)
			}
			var b Block
			if err = b.UnmarshalBinary(decoded); err != nil {
				t.Fatal(err)
			}

			bytes, err := json.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}
			str := string(bytes)
			if str != v.jsonEncoded {
				t.Error("unmarshaled to wrong json document:\nhave: ", str, "\nwant: ", v.jsonEncoded)
			}
		})
	}
}
