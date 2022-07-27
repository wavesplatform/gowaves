package proto

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type marshallable interface {
	io.WriterTo
	io.ReaderFrom
}

type comparable interface {
	Equal(comparable) bool
}

type testable interface {
	marshallable
	//comparable
}

type protocolMarshallingTest struct {
	testMessage testable
	testEncoded string
}

func (h *Header) Equal(d comparable) bool {
	p, ok := d.(*Header)
	if !ok {
		return false
	}
	return *h == *p
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
		&GetPeersMessage{},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000009  12345678     01         00000000              ",
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
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000039  12345678      02         0000002c      0b9ebfaf   00000005 8e5d2579 00001ad4 344d6fdb 00001acf 341c42d9 00001acf 341e2f43 00001acf 34335cb6 00001acf",
	},
	{
		&PeersMessage{[]PeerInfo{{net.IPv4(1, 2, 3, 4), 0x8888}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000019  12345678      02         0000000c         648fa8c8     00000001 01020304 00008888",
	},
	{
		&GetSignaturesMessage{[]crypto.Signature{{0x01}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000051  12345678      14         00000044      5474fb17   00000001 01000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&SignaturesMessage{[]crypto.Signature{{0x13}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000051  12345678      15         00000044         5e0c8bee    00000001 13000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&GetBlockMessage{NewBlockIDFromSignature(crypto.Signature{0x15, 0x12})},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"0000004d  12345678      16         00000040          01d5a895   15120000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
	{
		&BlockMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"0000000f  12345678       17         00000002      c2426c62   6642",
	},
	{
		&ScoreMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"0000000f  12345678       18         00000002      c2426c62   6642",
	},
	{
		&ScoreMessage{[]byte{0x01, 0x47, 0x02, 0x0e, 0x5b, 0x00, 0x75, 0x7a, 0xbe}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000016  12345678       18         00000009      74580717   01 47 02 0e 5b 00 75 7a be",
	},
	{
		&TransactionMessage{[]byte{0x66, 0x42}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"0000000f  12345678       19         00000002      c2426c62   6642",
	},
	{
		&CheckPointMessage{[]CheckpointItem{{0xdeadbeef, crypto.Signature{0x10, 0x11}}}},
		//P. Len |    Magic | ContentID | Payload Length | PayloadChecksum | Payload
		"00000059  12345678       64         0000004c      fcb6b02a   00000001 00000000 deadbeef 10110000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000",
	},
}

func TestProtocolMarshalling(t *testing.T) {
	for _, v := range tests {
		t.Run(fmt.Sprintf("%T", v.testMessage), func(t *testing.T) {
			trimmed := strings.Replace(v.testEncoded, " ", "", -1)
			decoded, err := hex.DecodeString(trimmed)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			_, err = v.testMessage.WriteTo(buf)
			require.NoError(t, err)
			require.Equal(t, decoded, buf.Bytes())

			m := v.testMessage
			_, err = m.ReadFrom(buf)
			require.NoError(t, err)
			require.Equal(t, v.testMessage, m)
		})
	}
}

func TestTransactionMessageUnmarshalBinary(t *testing.T) {
	p := TransactionMessage{
		Transaction: []byte("transaction"),
	}

	bts, err := p.MarshalBinary()
	require.NoError(t, err)

	otherBts := make([]byte, len(bts)+100)
	copy(otherBts, bts)

	p2 := TransactionMessage{}
	err = p2.UnmarshalBinary(otherBts)
	require.NoError(t, err)
	assert.Equal(t, []byte("transaction"), p2.Transaction)
}

func TestPeerInfoMarshalJSON(t *testing.T) {
	p := PeerInfo{
		Addr: net.ParseIP("8.8.8.8"),
		Port: 80,
	}
	js, err := json.Marshal(p)
	require.Nil(t, err)
	assert.Equal(t, `"8.8.8.8:80"`, string(js))

	// test incorrect struct
	p = PeerInfo{}
	_, err = json.Marshal(p)
	require.NotNil(t, err)
}

func TestNewPeerInfoFromString(t *testing.T) {
	rs, err := NewPeerInfoFromString("34.253.153.4:6868")
	require.NoError(t, err)
	assert.Equal(t, "34.253.153.4", rs.Addr.String())
	assert.EqualValues(t, 6868, rs.Port)
}

func TestPeerInfoUnmarshalJSON(t *testing.T) {
	p := new(PeerInfo)
	err := json.Unmarshal([]byte(`"/159.65.239.245:6868"`), p)
	require.Nil(t, err)
	assert.Equal(t, "159.65.239.245", p.Addr.String())
	assert.Equal(t, uint16(6868), p.Port)
}

func TestPeerInfoUnmarshalJSONWithoutSlash(t *testing.T) {
	p := new(PeerInfo)
	err := json.Unmarshal([]byte(`"159.65.239.245:6868"`), p)
	require.Nil(t, err)
	assert.Equal(t, "159.65.239.245", p.Addr.String())
	assert.Equal(t, uint16(6868), p.Port)
}

func TestPeerInfoUnmarshalJSONWithoutPort(t *testing.T) {
	p := new(PeerInfo)
	err := json.Unmarshal([]byte(`"/159.65.239.245"`), p)
	require.Nil(t, err)
	assert.Equal(t, "159.65.239.245", p.Addr.String())
	assert.Equal(t, uint16(0), p.Port)
}

func TestPeerInfoUnmarshalJSONNotAvailable(t *testing.T) {
	p := new(PeerInfo)
	err := json.Unmarshal([]byte(`"N/A"`), p)
	require.Nil(t, err)
	assert.Equal(t, &PeerInfo{}, p)
}

func TestHandshakeReadFrom(t *testing.T) {
	b := []byte{6, 119, 97, 118, 101, 115, 84, 0, 0, 0, 0, 0, 0, 0, 13, 0, 0, 0, 2, 11, 78, 111, 100, 101, 45, 53, 49, 52, 49, 55, 56, 0, 0, 0, 0, 0, 7, 216, 130, 0, 0, 0, 0 /*timestamp*/, 0, 0, 0, 0, 0, 0, 0, 0}
	h := Handshake{}
	_, err := h.ReadFrom(bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, "wavesT", h.AppName)
	assert.Equal(t, NewVersion(0, 13, 2), h.Version)
	assert.Equal(t, "Node-514178", h.NodeName)
	assert.Empty(t, h.DeclaredAddr)
}

func TestHandshakeReadFrom2(t *testing.T) {
	b := []byte{
		6, 119, 97, 118, 101, 115, 84, // app name
		0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 2, // version
		23, 116, 101, 115, 116, 110, 111, 100, 101, 49, 46, 119, 97, 118, 101, 115, 110, 111, 100, 101, 46, 110, 101, 116, // node name
		0, 0, 0, 0, 0, 9, 101, 17, // nonce
		0, 0, 0, 8 /*length*/, 217, 100, 219, 251, 0, 0, 26, 207,
		0, 0, 0, 0, 0, 0, 0, 1, // timestamp
	}
	h := Handshake{}
	_, err := h.ReadFrom(bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, "wavesT", h.AppName)
	assert.Equal(t, NewVersion(0, 15, 2), h.Version)
	assert.Equal(t, "testnode1.wavesnode.net", h.NodeName)
	assert.EqualValues(t, 615697, h.NodeNonce)
	require.NoError(t, err)

	assert.Equal(t, NewHandshakeTCPAddr(net.IPv4(217, 100, 219, 251), 6863), h.DeclaredAddr)
	assert.EqualValues(t, 1, h.Timestamp)

	buf := new(bytes.Buffer)
	_, _ = h.WriteTo(buf)
	require.Equal(t, b, buf.Bytes())
}

func TestHandshakeRoundTrip(t *testing.T) {
	buf := new(bytes.Buffer)

	h1 := Handshake{
		AppName:      "wavesT",
		Version:      NewVersion(0, 15, 2),
		NodeName:     "testnode1.wavesnode.net",
		NodeNonce:    615697,
		DeclaredAddr: NewHandshakeTCPAddr(net.IPv4(217, 100, 219, 251), 6863),
		Timestamp:    222233,
	}

	_, _ = h1.WriteTo(buf)

	h2 := Handshake{}
	_, _ = h2.ReadFrom(buf)
	assert.Equal(t, h1, h2)
}

func TestTransactionMessageMarshalRoundTrip(t *testing.T) {
	bts := []byte{
		0, 0, 1, 42, // total length
		18, 52, 86, 120, // magic
		25,          // transaction marker
		0, 0, 1, 29, // payload length
		208, 57, 41, 65, 4, 119, 220, 26, 37, 147, 197, 72, 109, 170, 147, 83, 220, 218, 17, 212, 125, 39, 185, 131, 203, 69, 8, 149, 185, 215, 35, 33, 52, 201, 186, 41, 33, 5, 224, 50, 154, 110, 14, 167, 44, 2, 106, 176, 54, 15, 65, 224, 128, 42, 203, 173, 248, 58, 234, 2, 226, 79, 100, 91, 156, 240, 21, 122, 6, 4, 136, 194, 176, 221, 33, 193, 126, 39, 31, 18, 42, 194, 241, 210, 179, 65, 245, 146, 6, 241, 229, 173, 11, 254, 121, 119, 248, 63, 231, 108, 128, 69, 1, 252, 128, 182, 134, 205, 22, 113, 112, 222, 246, 195, 232, 27, 191, 145, 230, 69, 162, 55, 112, 210, 135, 135, 126, 165, 69, 100, 184, 192, 145, 75, 122, 1, 252, 128, 182, 134, 205, 22, 113, 112, 222, 246, 195, 232, 27, 191, 145, 230, 69, 162, 55, 112, 210, 135, 135, 126, 165, 69, 100, 184, 192, 145, 75, 122, 0, 0, 1, 105, 6, 165, 214, 102, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 76, 75, 64, 1, 87, 243, 56, 252, 214, 234, 246, 38, 66, 79, 1, 239, 32, 122, 48, 14, 249, 87, 62, 63, 29, 174, 141, 205, 109, 0, 69, 100, 50, 99, 99, 55, 101, 100, 99, 52, 53, 51, 52, 100, 98, 101, 56, 53, 101, 56, 100, 99, 55, 52, 51, 55, 101, 54, 101, 101, 100, 50, 54, 52, 56, 55, 57, 54, 50, 48, 100, 51, 50, 52, 102, 52, 98, 57, 98, 55, 52, 101, 99, 52, 50, 56, 98, 99, 51, 51, 102, 101, 98, 49, 52, 32, 62, 133, 252, 18}

	m := TransactionMessage{}
	require.NoError(t, m.UnmarshalBinary(bts))

	bts2, err := m.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, bts, bts2)
}

func TestU8String(t *testing.T) {
	s := NewU8String("bla bla")

	buf := new(bytes.Buffer)

	_, _ = s.WriteTo(buf)

	s2 := NewU8String("")
	_, _ = s2.ReadFrom(buf)
	assert.Equal(t, s, s2)
}

func TestHandshakeTCPAddr_ReadWrite(t *testing.T) {
	tests := []HandshakeTCPAddr{{}, NewHandshakeTCPAddr(net.IPv4(127, 0, 0, 1), 6868)}
	for _, v := range tests {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			buf := new(bytes.Buffer)
			a := HandshakeTCPAddr{}
			_, err := v.WriteTo(buf)
			require.NoError(t, err)

			_, err = a.ReadFrom(buf)
			require.NoError(t, err)

			require.Equal(t, v, a)
			require.Equal(t, v.Port, a.Port)
		})
	}
}

func TestHandshakeTCPAddr_Empty(t *testing.T) {
	a := HandshakeTCPAddr{}
	require.True(t, a.Empty())
	b := NewHandshakeTCPAddr(net.IPv4(127, 0, 0, 1), 10)
	require.False(t, b.Empty())
}

func TestHandshakeTCPAddrCastToTCPAddr(t *testing.T) {
	ha := NewHandshakeTCPAddr(net.IPv4(1, 1, 1, 1), 8080)
	ca := net.TCPAddr(ha)
	assert.Equal(t, "1.1.1.1:8080", ca.String())
	a := net.Addr(&ca)
	assert.Equal(t, "1.1.1.1:8080", a.String())
	ta, ok := a.(*net.TCPAddr)
	assert.True(t, ok)
	assert.Equal(t, "1.1.1.1:8080", ta.String())
}

func TestNewVersionFromString(t *testing.T) {
	v, err := NewVersionFromString("1.2.3")
	require.NoError(t, err)
	assert.Equal(t, NewVersion(1, 2, 3), v)
	v, err = NewVersionFromString("1.2")
	require.NoError(t, err)
	assert.Equal(t, NewVersion(1, 2, 0), v)
	v, err = NewVersionFromString("1")
	require.NoError(t, err)
	assert.Equal(t, NewVersion(1, 0, 0), v)
	_, err = NewVersionFromString("")
	assert.Error(t, err)
	_, err = NewVersionFromString("1.2.3.4")
	assert.Error(t, err)
	_, err = NewVersionFromString("1234567890.5555555555.9999999999")
	assert.Error(t, err)
	_, err = NewVersionFromString("-1234.-4567.-8900")
	assert.Error(t, err)
}

func TestVersionsSort(t *testing.T) {
	versions := []Version{
		NewVersion(0, 16, 1),
		NewVersion(0, 13, 4),
		NewVersion(0, 16, 5),
		NewVersion(0, 15, 5),
		NewVersion(0, 16, 1),
		NewVersion(1, 0, 0),
		NewVersion(1, 2, 3),
	}
	v := ByVersion(versions)
	sort.Sort(v)
	expected := []Version{
		NewVersion(0, 13, 4),
		NewVersion(0, 15, 5),
		NewVersion(0, 16, 1),
		NewVersion(0, 16, 1),
		NewVersion(0, 16, 5),
		NewVersion(1, 0, 0),
		NewVersion(1, 2, 3),
	}
	assert.Equal(t, expected, []Version(v))
}

func TestGetSignaturesMessageRoundTrip(t *testing.T) {
	msg := GetSignaturesMessage{Signatures: []crypto.Signature{{0x1}, {0x2}}}
	msgBytes, err := msg.MarshalBinary()
	assert.NoError(t, err)
	var res GetSignaturesMessage
	err = res.UnmarshalBinary(msgBytes)
	assert.NoError(t, err)
	assert.Equal(t, res, msg)
}

func TestGetBlockIdsMessageRoundTrip(t *testing.T) {
	id0 := NewBlockIDFromSignature(crypto.Signature{0x1})
	id1 := NewBlockIDFromSignature(crypto.Signature{0x2})
	id2 := NewBlockIDFromDigest(crypto.Digest{0x3})
	msg := GetBlockIdsMessage{Blocks: []BlockID{id0, id1, id2}}
	msgBytes, err := msg.MarshalBinary()
	assert.NoError(t, err)
	var res GetBlockIdsMessage
	err = res.UnmarshalBinary(msgBytes)
	assert.NoError(t, err)
	assert.Equal(t, res, msg)
}

func TestBlockIdsMessageRoundTrip(t *testing.T) {
	id0 := NewBlockIDFromSignature(crypto.Signature{0x1})
	id1 := NewBlockIDFromSignature(crypto.Signature{0x2})
	id2 := NewBlockIDFromDigest(crypto.Digest{0x3})
	msg := BlockIdsMessage{Blocks: []BlockID{id0, id1, id2}}
	msgBytes, err := msg.MarshalBinary()
	assert.NoError(t, err)
	var res BlockIdsMessage
	err = res.UnmarshalBinary(msgBytes)
	assert.NoError(t, err)
	assert.Equal(t, res, msg)
}

func TestVersion_Cmp(t *testing.T) {
	require.Equal(t, 0, NewVersion(1, 2, 1).Cmp(NewVersion(1, 2, 1)))
	require.Equal(t, 1, NewVersion(2, 2, 1).Cmp(NewVersion(1, 2, 1)))
	require.Equal(t, -1, NewVersion(1, 2, 0).Cmp(NewVersion(1, 2, 1)))
}

func TestVersion_CmpMinor(t *testing.T) {
	// Check equals.
	require.Equal(t, 0, NewVersion(1, 2, 0).CmpMinor(NewVersion(1, 2, 0)))
	// Check patch version has no effect.
	require.Equal(t, 0, NewVersion(1, 2, 0).CmpMinor(NewVersion(1, 2, 3)))
	// Check diff only 1 version
	require.Equal(t, 1, NewVersion(1, 2, 0).CmpMinor(NewVersion(1, 1, 0)))
	// Check totally different.
	require.Equal(t, 2, NewVersion(1, 3, 0).CmpMinor(NewVersion(1, 1, 0)))
	// Check major.
	require.Equal(t, 2, NewVersion(0, 1, 0).CmpMinor(NewVersion(1, 1, 0)))
}

func TestGetBlockMessage_MarshalBinary(t *testing.T) {
	t.Run("parse signature from huge byte array", func(t *testing.T) {
		b := GetBlockMessage{
			BlockID: NewBlockIDFromSignature(crypto.MustSignatureFromBase58("2Br31FjkXETUiFanHyeN2GzinXvmN1vkVwRDkjzSHrcbFhU4aYwyvrXxkGkkprQ5h1UuaFeVQC1jexdsYbrqTtS9")),
		}

		rs, err := b.MarshalBinary()
		require.NoError(t, err)

		//
		buf := make([]byte, 1000)
		copy(buf, rs)

		b2 := GetBlockMessage{}
		err = b2.UnmarshalBinary(buf)
		require.NoError(t, err)

		require.Equal(t, b, b2)
	})
	t.Run("parse block id from huge byte array", func(t *testing.T) {
		b := GetBlockMessage{
			BlockID: NewBlockIDFromDigest(crypto.MustDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")),
		}

		rs, err := b.MarshalBinary()
		require.NoError(t, err)

		b2 := GetBlockMessage{}

		buf := make([]byte, 1000)
		copy(buf, rs)

		err = b2.UnmarshalBinary(buf)
		require.NoError(t, err)

		require.Equal(t, b, b2)
	})
}

func withBytes(t *testing.T, b encoding.BinaryMarshaler, f func(bts []byte)) {
	bts, err := b.MarshalBinary()
	require.NoError(t, err)
	f(bts)
}

// check that there is no panic, if payload is less than declared
func TestParsePacked(t *testing.T) {
	b := GetBlockMessage{
		BlockID: NewBlockIDFromDigest(crypto.MustDigestFromBase58("3Janbh2r7ZQjiUM3sWVswVGHWyQB2TPxm348QvuX5v6c")),
	}
	withBytes(t, &b, func(bts []byte) {
		bts = bts[:18:18]
		b2 := GetBlockMessage{}
		err := b2.UnmarshalBinary(bts)
		require.EqualError(t, err, "GetBlockMessage: expected data at least 49, found 18")
	})
}

func TestPeersMessage_Marshalling(t *testing.T) {
	t.Run("Test overflow", func(t *testing.T) {
		p := PeersMessage{}
		for i := uint16(0); i < 1100; i++ {
			p.Peers = append(p.Peers, PeerInfo{
				Addr: net.ParseIP("127.0.0.1"),
				Port: i,
			})
		}

		buf := &bytes.Buffer{}
		_, _ = p.WriteTo(buf)

		p2 := PeersMessage{}
		require.NoError(t, p2.UnmarshalBinary(buf.Bytes()))
	})
}

func TestTCPAddr_ToUint64(t *testing.T) {
	a := NewTCPAddrFromString("127.0.0.1:6868")
	rs := a.ToUint64()
	b := NewTcpAddrFromUint64(rs)

	require.True(t, a.Equal(b))
}
