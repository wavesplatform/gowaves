package proto

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestMicroBlock_Marshaling(t *testing.T) {
	txBytes := []byte{0, 0, 0, 152, 4, 76, 252, 177, 12, 123, 169, 56, 92, 8, 85, 82, 118, 1, 166, 228, 57, 52, 84, 161, 19, 144, 247, 9, 93, 114, 88, 198, 123, 123, 210, 188, 95, 177, 170, 229, 15, 176, 248, 128, 112, 121, 201, 53, 221, 15, 55, 231, 118, 113, 192, 201, 113, 251, 55, 6, 95, 207, 47, 24, 71, 240, 162, 206, 6, 4, 236, 89, 77, 54, 8, 236, 240, 30, 10, 87, 121, 139, 23, 7, 114, 121, 45, 177, 69, 50, 132, 55, 119, 224, 172, 245, 68, 95, 44, 28, 243, 4, 0, 0, 0, 0, 1, 107, 137, 11, 41, 201, 0, 0, 0, 10, 247, 96, 247, 0, 0, 0, 0, 0, 0, 1, 134, 160, 1, 87, 126, 90, 125, 49, 243, 210, 18, 83, 195, 130, 223, 30, 209, 178, 95, 17, 186, 108, 63, 172, 209, 224, 228, 138, 0, 0}
	txs, err := NewTransactionsFromBytes(txBytes, 1, TestNetScheme)
	require.NoError(t, err)
	refSig := crypto.MustSignatureFromBase58("37ex9gonRZtUddDHgSzSes5Ds9UeQyS74DyAXtGFrDpJnEg7sjGdi2ncaV4rVpZnLboQmid3whcbZUWS49FV3ZCs")
	ref := NewBlockIDFromSignature(refSig)
	m := MicroBlock{
		VersionField:          3,
		Reference:             ref,
		TotalResBlockSigField: crypto.MustSignatureFromBase58("3ta68P5LdLHWKuKcDvASsjcCMEQsm1ySrpxYZwqmzCHiAWHgrYJE1ZmaTsh3ytPqY73545EUPDaGfVdrguTqVTHg"),
		SenderPK:              crypto.MustPublicKeyFromBase58("adBBo1RCATFZYX114g8xDRpzKqRCVwckuTP6rcgYmA6"),
		Transactions:          txs,
		TransactionCount:      1,
		Signature:             crypto.MustSignatureFromBase58("56Un9HE6UnG2ut3srow7tGrQ9pMKyKqhbpJBjwJ7oV2rpr58iaPYG5G3QmZqVo169GN4bNHNHwhDykgPbQknD3Nv"),
	}

	buf := &bytes.Buffer{}
	_, _ = m.WriteTo(TestNetScheme, buf)

	m2 := MicroBlock{}
	_ = m2.UnmarshalBinary(buf.Bytes(), TestNetScheme)

	require.Equal(t, m, m2)

	rs, err := m.VerifySignature(TestNetScheme)
	require.NoError(t, err)
	require.True(t, rs)
}

func TestMicroBlockProtobufRoundTrip(t *testing.T) {
	txBytes := []byte{0, 0, 0, 152, 4, 76, 252, 177, 12, 123, 169, 56, 92, 8, 85, 82, 118, 1, 166, 228, 57, 52, 84, 161, 19, 144, 247, 9, 93, 114, 88, 198, 123, 123, 210, 188, 95, 177, 170, 229, 15, 176, 248, 128, 112, 121, 201, 53, 221, 15, 55, 231, 118, 113, 192, 201, 113, 251, 55, 6, 95, 207, 47, 24, 71, 240, 162, 206, 6, 4, 236, 89, 77, 54, 8, 236, 240, 30, 10, 87, 121, 139, 23, 7, 114, 121, 45, 177, 69, 50, 132, 55, 119, 224, 172, 245, 68, 95, 44, 28, 243, 4, 0, 0, 0, 0, 1, 107, 137, 11, 41, 201, 0, 0, 0, 10, 247, 96, 247, 0, 0, 0, 0, 0, 0, 1, 134, 160, 1, 87, 126, 90, 125, 49, 243, 210, 18, 83, 195, 130, 223, 30, 209, 178, 95, 17, 186, 108, 63, 172, 209, 224, 228, 138, 0, 0}
	txs, err := NewTransactionsFromBytes(txBytes, 1, MainNetScheme)
	require.NoError(t, err)
	refSig := crypto.MustSignatureFromBase58("37ex9gonRZtUddDHgSzSes5Ds9UeQyS74DyAXtGFrDpJnEg7sjGdi2ncaV4rVpZnLboQmid3whcbZUWS49FV3ZCs")
	ref := NewBlockIDFromSignature(refSig)
	m := MicroBlock{
		VersionField:          3,
		Reference:             ref,
		TotalResBlockSigField: crypto.MustSignatureFromBase58("3ta68P5LdLHWKuKcDvASsjcCMEQsm1ySrpxYZwqmzCHiAWHgrYJE1ZmaTsh3ytPqY73545EUPDaGfVdrguTqVTHg"),
		SenderPK:              crypto.MustPublicKeyFromBase58("adBBo1RCATFZYX114g8xDRpzKqRCVwckuTP6rcgYmA6"),
		Transactions:          txs,
		TransactionCount:      1,
		Signature:             crypto.MustSignatureFromBase58("56Un9HE6UnG2ut3srow7tGrQ9pMKyKqhbpJBjwJ7oV2rpr58iaPYG5G3QmZqVo169GN4bNHNHwhDykgPbQknD3Nv"),
	}

	pbBytes, err := m.MarshalToProtobuf(MainNetScheme)
	require.NoError(t, err)
	var m2 MicroBlock
	err = m2.UnmarshalFromProtobuf(pbBytes)
	require.NoError(t, err)
	require.Equal(t, m, m2)
}

func TestMicroBlock_WriteTo(t *testing.T) {
	id := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("rBA7qj1nvXCnD8puLzWBWDoyHVkm3TzooDJgwbiaum9oV3vGhxGs45DfqwoM9qAyu4xfP6j8gQL6avub1wrB2zX"))
	ref := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("2UwZrKyjx7Bs4RYkEk5SLCdtr9w6GR1EDbpS3TH9DGJKcxSCuQP4nivk4YPFpQTqWmoXXPPUiy6riF3JwhikbSQu"))
	c1 := &MicroBlockInv{
		PublicKey:    crypto.MustPublicKeyFromBase58("adBBo1RCATFZYX114g8xDRpzKqRCVwckuTP6rcgYmA6"),
		TotalBlockID: id,
		Reference:    ref,
		Signature:    crypto.MustSignatureFromBase58("Cmf4sk7RZkCLn5w8LjSxZGTVNKAchE2w7eBcnFbbC6suntuuY8ieWZNxp82ZTXb1ZADwpfjWuSQ3bNQ61veoRPB")}

	b := bytes.Buffer{}

	_, _ = c1.WriteTo(&b)

	c2 := &MicroBlockInv{}
	_ = c2.UnmarshalBinary(b.Bytes())
	require.Equal(t, c1, c2)
}

func TestMicroBlockInv_SignVerify(t *testing.T) {
	sec, pub, err := crypto.GenerateKeyPair([]byte("test1"))
	require.NoError(t, err)
	id := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("rBA7qj1nvXCnD8puLzWBWDoyHVkm3TzooDJgwbiaum9oV3vGhxGs45DfqwoM9qAyu4xfP6j8gQL6avub1wrB2zX"))
	ref := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("2UwZrKyjx7Bs4RYkEk5SLCdtr9w6GR1EDbpS3TH9DGJKcxSCuQP4nivk4YPFpQTqWmoXXPPUiy6riF3JwhikbSQu"))
	c1 := &MicroBlockInv{
		PublicKey:    pub,
		TotalBlockID: id,
		Reference:    ref,
	}

	require.NoError(t, c1.Sign(sec, TestNetScheme))
	rs, err := c1.Verify(TestNetScheme)
	require.NoError(t, err)
	require.True(t, rs)
}

func TestMicroBlockInvMessage_WriteTo_And_Marshal(t *testing.T) {
	id := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("rBA7qj1nvXCnD8puLzWBWDoyHVkm3TzooDJgwbiaum9oV3vGhxGs45DfqwoM9qAyu4xfP6j8gQL6avub1wrB2zX"))
	ref := NewBlockIDFromSignature(crypto.MustSignatureFromBase58("2UwZrKyjx7Bs4RYkEk5SLCdtr9w6GR1EDbpS3TH9DGJKcxSCuQP4nivk4YPFpQTqWmoXXPPUiy6riF3JwhikbSQu"))
	c1 := &MicroBlockInv{
		PublicKey:    crypto.MustPublicKeyFromBase58("adBBo1RCATFZYX114g8xDRpzKqRCVwckuTP6rcgYmA6"),
		TotalBlockID: id,
		Reference:    ref,
		Signature:    crypto.MustSignatureFromBase58("Cmf4sk7RZkCLn5w8LjSxZGTVNKAchE2w7eBcnFbbC6suntuuY8ieWZNxp82ZTXb1ZADwpfjWuSQ3bNQ61veoRPB"),
	}

	bts, _ := c1.MarshalBinary()
	m := MicroBlockInvMessage{
		Body: bts,
	}

	rs1, err := m.MarshalBinary()
	require.NoError(t, err)

	rs2 := new(bytes.Buffer)
	_, _ = m.WriteTo(rs2)

	require.Equal(t, rs1, rs2.Bytes())

	m2 := MicroBlockInvMessage{}
	err = m2.UnmarshalBinary(rs1)
	require.NoError(t, err)

	require.Equal(t, m, m2)
}

func TestMicroBlockRequestMessage_Marshaling(t *testing.T) {
	sig := crypto.MustSignatureFromBase58("rBA7qj1nvXCnD8puLzWBWDoyHVkm3TzooDJgwbiaum9oV3vGhxGs45DfqwoM9qAyu4xfP6j8gQL6avub1wrB2zX")

	mess := MicroBlockRequestMessage{
		TotalBlockSig: sig.Bytes(),
	}

	bts, err := mess.MarshalBinary()
	require.NoError(t, err)

	mess2 := &MicroBlockRequestMessage{}
	err = mess2.UnmarshalBinary(bts)
	require.NoError(t, err)

	require.Equal(t, mess.TotalBlockSig, mess2.TotalBlockSig)
}

func TestMicroBlockV5VerifySignature(t *testing.T) {
	b, err := base64.StdEncoding.DecodeString("CqMCCAUSINDAl9MzGkepj6WsZ+1NZv0grSgzJogVswMTP7+ug6LoGkDJBEdWcsDc/bat2ljrW74o9l7Y+Pcp07ra2VRe/HLU/Oq6cT7xJqyqZ7xoXP5tLKcnq4hF/5FtS/NYx5zX3RMPIiDk9FCvGrDyyqy8jzX1qe6cEdv5NRXv+hSH+BMnnFtqBSqYAQpUCFQSIBjkoQIwpcrsWlpsgLJVOBo27loBDODD+h473uYYaxMSGgQQoI0GIOvZ5ffzLigDwgYeChYKFCCTgv+auCSYevJXZ7mkKyv2/dkoEgQQoI0GEkD5K5E+HKr3IXYhnwLZaWVsIF+tJdbvV4LFjksWIeLoopDf46TTE2XXXb64R2ZsbWV0QJpQ3cNqTnKXGcB2DesIEkA+/2wKSB07Tg2uBH9OGuIXLBH7FzKPLllyjn7TlvYTLZrohyNSBAIQ3sM9UwPQkUDSC1NGYBFwRHRdF+gPfQcDGiAzlpLCohmCR1KXVnxw5AVO7Xq60gorXfInMXSiS3Qf9Q==")
	require.NoError(t, err)
	micro := new(MicroBlock)
	err = micro.UnmarshalFromProtobuf(b)
	require.NoError(t, err)
	ok, err := micro.VerifySignature('T')
	require.NoError(t, err)
	assert.True(t, ok)
}
