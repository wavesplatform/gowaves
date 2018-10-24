package proto

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

type marshallable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	io.WriterTo
	io.ReaderFrom
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
	return h.Name == p.Name && h.Version.Major == p.Version.Major &&
		h.Version.Minor == p.Version.Minor && h.Version.Patch == p.Version.Patch &&
		h.NodeName == p.NodeName &&
		h.NodeNonce == p.NodeNonce &&
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
		if !m.Peers[i].Addr.Equal(p.Peers[i].Addr) {
			return false
		}
		if m.Peers[i].Port != p.Peers[i].Port {
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
		&Handshake{"ab", Version{0x10, 0x3, 0x8}, "dc", 0x701, []byte{10, 20}, 0x8000},
		"0261620000001000000003000000080264630000000000000701000000020a140000000000008000",
	},
	{
		&Handshake{"wavesT", Version{0x0, 0xe, 0x5}, "My TESTNET node", 0x1c61, []byte{0xb9, 0x29, 0x70, 0x1e, 0x00, 0x00, 0x1a, 0xcf}, 0x5bb482c9},
		"06776176657354000000000000000e000000050f4d7920544553544e4554206e6f64650000000000001c6100000008b929701e00001acf000000005bb482c9",
	},
	{
		&GetPeersMessage{},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000009  12345678          01         00000000",
	},
	{
		&PeersMessage{[]PeerInfo{
			{net.IPv4(0x8e, 0x5d, 0x25, 0x79), 0x1ad4},
			{net.IPv4(0x34, 0x4d, 0x6f, 0xdb), 0x1acf},
			{net.IPv4(0x34, 0x1c, 0x42, 0xd9), 0x1acf},
			{net.IPv4(0x34, 0x1e, 0x2f, 0x43), 0x1acf},
			{net.IPv4(0x34, 0x33, 0x5c, 0xb6), 0x1acf},
		},
		},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000039  12345678          02          0000002c     0b9ebfaf   00000005 8e5d2579 00001ad4 344d6fdb 00001acf 341c42d9 00001acf 341e2f43 00001acf 34335cb6 00001acf",
	},
	{
		&PeersMessage{[]PeerInfo{{net.IPv4(1, 2, 3, 4), 0x8888}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000019  12345678          02         0000000c      648fa8c8   00000001 01020304 00008888",
	},
	{
		&GetSignaturesMessage{[]BlockID{{0x01}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000051  12345678          14         00000044      5474fb17   00000001 01000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&SignaturesMessage{[]BlockSignature{{0x13}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000051  12345678          15         00000044      5e0c8bee   00000001 13000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&GetBlockMessage{BlockID{0x15, 0x12}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000004d  12345678          16         00000040      01d5a895   15120000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&BlockMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000000f  12345678          17         00000002      0e5751c0   6642",
	},
	{
		&ScoreMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000000f  12345678          18         00000002      c2426c62   6642",
	},
	{
		&ScoreMessage{[]byte{0x01, 0x47, 0x02, 0x0e, 0x5b, 0x00, 0x75, 0x7a, 0xbe}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000016  12345678          18         00000009      74580717   01 47 02 0e 5b 00 75 7a be",
	},
	{
		&TransactionMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"0000000f  12345678          19         00000002      0e5751c0   6642",
	},
	{
		&CheckPointMessage{[]CheckpointItem{{0xdeadbeef, BlockSignature{0x10, 0x11}}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadCsum | Payload
		"00000059  12345678          64         0000004c      fcb6b02a   00000001 00000000 deadbeef 10110000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
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

			var writerBuffer bytes.Buffer
			writer := io.Writer(&writerBuffer)

			v.testMessage.WriteTo(writer)

			if !bytes.Equal(writerBuffer.Bytes(), data) {
				t.Errorf("failed to write message to writer")
			}

			v.testMessage.WriteTo(writer)

			reader := io.Reader(&writerBuffer)

			m := v.testMessage
			if err = m.UnmarshalBinary(decoded); err != nil {
				t.Errorf("failed to unmarshal: %s", err)
			}
			if !v.testMessage.Equal(m) {
				t.Errorf("failed to correclty unmarshal message")
			}

			m.ReadFrom(reader)
			if !v.testMessage.Equal(m) {
				t.Errorf("failed to correctly read message from reader")
			}
		})
	}
}
