package proto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"testing"
)

type headerMarshallingTestData struct {
	header        Header
	encodedHeader string
}

var headerMarshallingTests = []headerMarshallingTestData{
	{
		Header{0x42, 0x42000000, 8, 0x666, 0x999},
		"0000004212345678080000066600000999",
	},
	{
		Header{0x4200, 0x420000, 255, 0xaabbddee, 0xdeadbeef},
		"0000420012345678ffaabbddeedeadbeef",
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

type handshakeMarshallingTestData struct {
	handshake        Handshake
	encodedHandshake string
}

var handshakeMarshallingTests = []handshakeMarshallingTestData{
	{
		Handshake{0x2, "ab", 0x10, 0x3, 0x8, 0x2, "dc", 0x701, 0x2, []byte{10, 20}, 0x8000},
		"0261620000001000000003000000080264630000000000000701000000020a140000000000008000",
	},
	{
		Handshake{0x6, "wavesT", 0x0, 0xe, 0x5, 0xf, "My TESTNET node", 0x1c61, 0x08, []byte{0xb9, 0x29, 0x70, 0x1e, 0x00, 0x00, 0x1a, 0xcf}, 0x5bb482c9},
		"06776176657354000000000000000e000000050f4d7920544553544e4554206e6f64650000000000001c6100000008b929701e00001acf000000005bb482c9",
	},
}

func TestHandshakeMarshalling(t *testing.T) {
	for _, v := range handshakeMarshallingTests {
		decoded, err := hex.DecodeString(v.encodedHandshake)
		if err != nil {
			log.Fatal(err)
		}

		data, err := v.handshake.MarshalBinary()
		if err != nil {
			log.Fatal(err)
		}

		res := bytes.Compare(data, decoded)
		if res != 0 {
			strEncoded := hex.EncodeToString(data)
			log.Fatal(fmt.Errorf("want: %s, have: %s", v.encodedHandshake, strEncoded))
		}
	}
}

type getPeersMessageMarshallingTestData struct {
	message        GetPeersMessage
	encodedMessage string
}

var getPeersMessageTests = []getPeersMessageMarshallingTestData{
	{
		GetPeersMessage{Header{0x02, headerMagic, ContentIDGetPeers, 0, 0}},
		"0000000212345678010000000000000000",
	},
}

func TestGetPeersMessageMarshalling(t *testing.T) {
	for _, v := range getPeersMessageTests {
		decoded, err := hex.DecodeString(v.encodedMessage)
		if err != nil {
			log.Fatal(err)
		}

		data, err := v.message.MarshalBinary()
		if err != nil {
			log.Fatal(err)
		}

		res := bytes.Compare(data, decoded)
		if res != 0 {
			strEncoded := hex.EncodeToString(data)
			log.Fatal(fmt.Errorf("failed to marshal GetPeersMessage; want: %s, have: %s", v.encodedMessage, strEncoded))
		}

		var message GetPeersMessage

		if err = message.UnmarshalBinary(decoded); err != nil {
			log.Fatal(fmt.Errorf("failed to unmarshal GetPeersMessage; %s", err))
		}

		if message != v.message {
			log.Fatal(errors.New("failed to correctly unmarshal GetPeersMessage"))
		}
	}
}

type peersMessageMarshallingTestData struct {
	message        PeersMessage
	encodedMessage string
}

var peersMessageTests = []peersMessageMarshallingTestData{
	{
		PeersMessage{Header{0x02, headerMagic, ContentIDPeers, 0, 0}, 1, []PeerInfo{{net.IPv4(1, 2, 3, 4), 0x1488}}},
		"00000002 12345678 02 00000000 00000000 00000001 01020304 1488",
	},
}

func TestPeersMessageMarshalling(t *testing.T) {
	for _, v := range peersMessageTests {
		decoded, err := hex.DecodeString(strings.Replace(v.encodedMessage, " ", "", -1))
		if err != nil {
			log.Fatal(err)
		}

		data, err := v.message.MarshalBinary()
		if err != nil {
			log.Fatal(err)
		}

		res := bytes.Compare(data, decoded)
		if res != 0 {
			strEncoded := hex.EncodeToString(data)
			log.Fatal(fmt.Errorf("failed to marshal PeersMessage; want %s, have %s", v.encodedMessage, strEncoded))
		}
	}
}
