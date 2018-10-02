package proto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"testing"
)

type headerMarshallingTestData struct {
	header        Header
	encodedHeader string
}

var headerMarshallingTests = []headerMarshallingTestData{
	{
		Header{0x42, 0x42000000, 8, 0x666, 0x999},
		"0000004242000000080000066600000999",
	},
	{
		Header{0x4200, 0x420000, 255, 0xaabbddee, 0xdeadbeef},
		"0000420000420000ffaabbddeedeadbeef",
	},
}

func TestHeaderMarshalling(t *testing.T) {
	for _, v := range headerMarshallingTests {
		decoded, err := hex.DecodeString(v.encodedHeader)
		if err != nil {
			log.Fatal(err)
		}

		data, err := v.header.MarshalBinary()
		if err != nil {
			log.Fatal(err)
		}

		res := bytes.Compare(data, decoded)
		if res != 0 {
			strEncoded := hex.EncodeToString(data)
			log.Fatal(fmt.Errorf("want: %s, have: %s", v.encodedHeader, strEncoded))
		}

	}
}
