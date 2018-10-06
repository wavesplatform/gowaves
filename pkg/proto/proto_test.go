package proto

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"testing"
)

type marshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type comparable interface {
	Equal(comparable) bool
}

type testable interface {
	marshallable
	comparable
}

type protocolMarshallingTest struct {
	testMessage testable
	testEncoded string
}

func (h *header) Equal(d comparable) bool {
	p, ok := d.(*header)
	if !ok {
		return false
	}
	return *h == *p
}

func (h *Handshake) Equal(d comparable) bool {
	p, ok := d.(*Handshake)
	if !ok {
		return false
	}
	return h.NameLength == p.NameLength &&
		h.Name == p.Name && h.VersionMajor == p.VersionMajor &&
		h.VersionMinor == p.VersionMinor && h.VersionPatch == p.VersionPatch &&
		h.NodeNameLength == p.NodeNameLength && h.NodeName == p.NodeName &&
		h.NodeNonce == p.NodeNonce && h.DeclaredAddrLength == p.DeclaredAddrLength &&
		bytes.Compare(p.DeclaredAddrBytes, h.DeclaredAddrBytes) == 0 &&
		h.Timestamp == p.Timestamp
}

func (m *GetPeersMessage) Equal(d comparable) bool {
	p, ok := d.(*GetPeersMessage)
	if !ok {
		return false
	}

	return *m == *p
}

func (m *PeersMessage) Equal(d comparable) bool {
	p, ok := d.(*PeersMessage)
	if !ok {
		return false
	}

	if len(m.Peers) != len(p.Peers) {
		return false
	}

	for i := 0; i < len(m.Peers); i++ {
		if !m.Peers[i].addr.Equal(p.Peers[i].addr) {
			return false
		}
		if m.Peers[i].port != p.Peers[i].port {
			return false
		}
	}

	return true
}

func (m *GetSignaturesMessage) Equal(d comparable) bool {
	p, ok := d.(*GetSignaturesMessage)
	if !ok {
		return false
	}
	if len(m.Blocks) != len(p.Blocks) {
		return false
	}
	for i := 0; i < len(m.Blocks); i++ {
		if m.Blocks[i] != p.Blocks[i] {
			return false
		}
	}

	return true
}

func (m *SignaturesMessage) Equal(d comparable) bool {
	p, ok := d.(*SignaturesMessage)
	if !ok {
		return false
	}
	if len(m.Signatures) != len(p.Signatures) {
		return false
	}
	for i := 0; i < len(m.Signatures); i++ {
		if m.Signatures[i] != p.Signatures[i] {
			return false
		}
	}

	return true
}

func (m *GetBlockMessage) Equal(d comparable) bool {
	p, ok := d.(*GetBlockMessage)
	if !ok {
		return false
	}

	return *m == *p
}

func (m *BlockMessage) Equal(d comparable) bool {
	p, ok := d.(*BlockMessage)
	if !ok {
		return false
	}

	return bytes.Equal(m.BlockBytes, p.BlockBytes)
}

func (m *ScoreMessage) Equal(d comparable) bool {
	p, ok := d.(*ScoreMessage)
	if !ok {
		return false
	}
	return bytes.Equal(m.Score, p.Score)
}

func (m *CheckPointMessage) Equal(d comparable) bool {
	p, ok := d.(*CheckPointMessage)
	if !ok {
		return false
	}
	if len(m.Checkpoints) != len(p.Checkpoints) {
		return false
	}
	for i := 0; i < len(m.Checkpoints); i++ {
		if m.Checkpoints[i] != p.Checkpoints[i] {
			return false
		}
	}
	return true
}

func (m *TransactionMessage) Equal(d comparable) bool {
	p, ok := d.(*TransactionMessage)
	if !ok {
		return false
	}
	return bytes.Equal(m.Transaction, p.Transaction)
}

var tests = []protocolMarshallingTest{
	{
		&header{0x42, 0x42000000, 8, 0x666, 0x999},
		"0000004212345678080000066600000999",
	},
	{
		&header{0x4200, 0x420000, 255, 0xaabbddee, 0xdeadbeef},
		"0000420012345678ffaabbddeedeadbeef",
	},
	{
		&Handshake{0x2, "ab", 0x10, 0x3, 0x8, 0x2, "dc", 0x701, 0x2, []byte{10, 20}, 0x8000},
		"0261620000001000000003000000080264630000000000000701000000020a140000000000008000",
	},
	{
		&Handshake{0x6, "wavesT", 0x0, 0xe, 0x5, 0xf, "My TESTNET node", 0x1c61, 0x08, []byte{0xb9, 0x29, 0x70, 0x1e, 0x00, 0x00, 0x1a, 0xcf}, 0x5bb482c9},
		"06776176657354000000000000000e000000050f4d7920544553544e4554206e6f64650000000000001c6100000008b929701e00001acf000000005bb482c9",
	},
	{
		&GetPeersMessage{},
		"0000001112345678010000000000000000",
	},
	{
		&PeersMessage{1, []PeerInfo{{net.IPv4(1, 2, 3, 4), 0x8888}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000001b  12345678          02         0000000a      00000000   00000001 01020304 8888",
	},
	{
		&GetSignaturesMessage{[]BlockID{BlockID{0x01}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000055  12345678          14         00000044      00000000   00000001 01000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&SignaturesMessage{[]BlockSignature{BlockSignature{0x13}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000055  12345678          15         00000044      00000000   00000001 13000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&GetBlockMessage{BlockID{0x15, 0x12}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000051  12345678          16         00000040      00000000   15120000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&BlockMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          17         00000002      00000000   6642",
	},
	{
		&ScoreMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          18         00000002      00000000   6642",
	},
	{
		&TransactionMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000013  12345678          19         00000002      00000000   6642",
	},
	{
		&CheckPointMessage{[]CheckpointItem{{0xdeadbeef, BlockSignature{0x10, 0x11}}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000005d  12345678          64         0000004c      00000000   00000001 00000000 deadbeef 10110000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestProtocolMarshalling(t *testing.T) {
	for _, v := range tests {
		t.Run(fmt.Sprintf("%T", v.testMessage), func(t *testing.T) {
			trimmed := strings.Replace(v.testEncoded, " ", "", -1)
			decoded, err := hex.DecodeString(trimmed)
			if err != nil {
				t.Error(err)
			}

			data, err := v.testMessage.MarshalBinary()
			if err != nil {
				t.Error(err)
			}
			if res := bytes.Compare(data, decoded); res != 0 {
				strEncoded := hex.EncodeToString(data)
				t.Errorf("want: %s, have %s", v.testEncoded, strEncoded)
			}
		})
	}
}
