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
	header        header
	encodedHeader string
}

var headerMarshallingTests = []headerMarshallingTestData{
	{
		header{0x42, 0x42000000, 8, 0x666, 0x999},
		"0000004212345678080000066600000999",
	},
	{
		header{0x4200, 0x420000, 255, 0xaabbddee, 0xdeadbeef},
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
		GetPeersMessage{},
		"0000001112345678010000000000000000",
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
		PeersMessage{1, []PeerInfo{{net.IPv4(1, 2, 3, 4), 0x8888}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000001b  12345678          02         0000000a      00000000   00000001 01020304 8888",
	},
}

func TestPeersMessageMarshalling(t *testing.T) {
	for _, v := range peersMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
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
			t.Errorf("failed to marshal PeersMessage; want %s, have %s", rawString, strEncoded)
		}
	}
}

type getSignaturesMessageMarshallingTestData struct {
	message        GetSignaturesMessage
	encodedMessage string
}

var getSignaturesMessageTests = []getSignaturesMessageMarshallingTestData{
	{
		GetSignaturesMessage{[]BlockID{BlockID{0x01}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000055  12345678          14         00000044      00000000   00000001 01000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestGetSignaturesMessageMarshalling(t *testing.T) {
	for _, v := range getSignaturesMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		res := bytes.Compare(data, decoded)
		if res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal GetSignatures message; want %s, have %s", rawString, strEncoded)
		}

		var message GetSignaturesMessage

		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to unmarshal GetPeersMessage; %s", err)
		}

		if len(message.Blocks) != len(v.message.Blocks) {
			t.Error("failed to correctly unmarshal GetPeersMessage")
		}

		for i := 0; i < len(message.Blocks); i++ {
			if message.Blocks[i] != v.message.Blocks[i] {
				t.Error("failed to correctly unmarshal GetPeersMessage")
			}
		}
	}
}

type signaturesMarshallingTestData struct {
	message        SignaturesMessage
	encodedMessage string
}

var signaturesMessageTests = []signaturesMarshallingTestData{
	{
		SignaturesMessage{[]BlockSignature{BlockSignature{0x13}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000055  12345678          15         00000044      00000000   00000001 13000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestSignaturesMessageMarshalling(t *testing.T) {
	for _, v := range signaturesMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal Signatures message; want %s, have %s", rawString, strEncoded)
		}

		var message SignaturesMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal Signatures message; %s", err)
		}
		if len(message.Signatures) != len(v.message.Signatures) {
			t.Error("failed to correctly unmarshal Signatures message")
		}
		for i := 0; i < len(message.Signatures); i++ {
			if message.Signatures[i] != v.message.Signatures[i] {
				t.Error("failed to correctly unmarshal Signatures message")
			}
		}
	}
}

type getBlockMarshallingData struct {
	message        GetBlockMessage
	encodedMessage string
}

var getBlockMessageTests = []getBlockMarshallingData{
	{
		GetBlockMessage{BlockID{0x15, 0x12}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000051  12345678          16         00000040      00000000   15120000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestGetBlockMessageMarshalling(t *testing.T) {
	for _, v := range getBlockMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal GetBlock message; want %s, have %s", rawString, strEncoded)
		}

		var message GetBlockMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal GetBlock message; %s", err)
		}
		if message != v.message {
			t.Errorf("failed to correctly unmarshal GetBlock message")
		}
	}
}

type blockMarshallingData struct {
	message        BlockMessage
	encodedMessage string
}

var blockMessageTests = []blockMarshallingData{
	{
		BlockMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          17         00000002      00000000   6642",
	},
}

func TestBlockMessageMarshalling(t *testing.T) {
	for _, v := range blockMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal Block message; want %s, have %s", rawString, strEncoded)
		}

		var message BlockMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal Block message; %s", err)
		}
		if bytes.Compare(message.BlockBytes, v.message.BlockBytes) != 0 {
			t.Errorf("failed to correctly unmarshal Block message")
		}
	}
}

type scoreMarshallingData struct {
	message        ScoreMessage
	encodedMessage string
}

var scoreMessageTests = []scoreMarshallingData{
	{
		ScoreMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          18         00000002      00000000   6642",
	},
}

func TestScoreMessageMarshalling(t *testing.T) {
	for _, v := range scoreMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal Score message; want %s, have %s", rawString, strEncoded)
		}

		var message ScoreMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal Score message; %s", err)
		}
		if bytes.Compare(message.Score, v.message.Score) != 0 {
			t.Errorf("failed to correctly unmarshal Score message")
		}
	}
}

type transactionMarshallingData struct {
	message        TransactionMessage
	encodedMessage string
}

var transactionMessageTests = []transactionMarshallingData{
	{
		TransactionMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          19         00000002      00000000   6642",
	},
}

func TestTransactionMessageMarshalling(t *testing.T) {
	for _, v := range transactionMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal Score message; want %s, have %s", rawString, strEncoded)
		}

		var message TransactionMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal Transaction message; %s", err)
		}
		if bytes.Compare(message.Transaction, v.message.Transaction) != 0 {
			t.Errorf("failed to correctly unmarshal Transaction message")
		}
	}
}

type checkPointMarshallingTestData struct {
	message        CheckPointMessage
	encodedMessage string
}

var checkPointMessageTests = []checkPointMarshallingTestData{
	{
		CheckPointMessage{[]CheckpointItem{{0xdeadbeef, BlockSignature{0x10, 0x11}}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000005d  12345678          64         0000004c      00000000   00000001 00000000 deadbeef 10110000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestCheckPointMessageMarshalling(t *testing.T) {
	for _, v := range checkPointMessageTests {
		rawString := strings.Replace(v.encodedMessage, " ", "", -1)
		decoded, err := hex.DecodeString(rawString)
		if err != nil {
			t.Error(err)
		}
		data, err := v.message.MarshalBinary()
		if err != nil {
			t.Error(err)
		}
		if res := bytes.Compare(data, decoded); res != 0 {
			strEncoded := hex.EncodeToString(data)
			t.Errorf("failed to marshal CheckPoint message; want %s, have %s", rawString, strEncoded)
		}

		var message CheckPointMessage
		if err = message.UnmarshalBinary(decoded); err != nil {
			t.Errorf("failed to correctly unmarshal CheckPoint message; %s", err)
		}
		if len(message.Checkpoints) != len(v.message.Checkpoints) {
			t.Errorf("failed to correctly unmarshal CheckPoint message: %d %d", len(message.Checkpoints), len(v.message.Checkpoints))
		}

		for i := 0; i < len(message.Checkpoints); i++ {
			if message.Checkpoints[i].Height != v.message.Checkpoints[i].Height {
				t.Errorf("wrong height")
			}
			if bytes.Compare(message.Checkpoints[i].Signature[:], v.message.Checkpoints[i].Signature[:]) != 0 {
				t.Errorf("signatures don't match")
			}
		}
	}
}
