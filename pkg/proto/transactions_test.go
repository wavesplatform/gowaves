package proto

import (
	"encoding/json"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"strings"
	"testing"
	"time"
)

func TestIssueV1FromMainNet(t *testing.T) {
	const (
		senderPK = "BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"
		sig      = "6JAr35fMADxhhK5jEXCKBzZAMCBoXBPcW4D9iaBDnhATxQ7Dk5EgJKBSWCeauqftSUVWgY79bMjdxqomCRxafFd"
		id       = "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS"
	)
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	if assert.NoError(t, err) {
		tx, err := NewUnsignedIssueV1(spk, "WBTC", "Bitcoin Token", 2100000000000000, 8, false, 1480690876160, 100000000)
		if assert.NoError(t, err) {
			b := tx.marshalBody()
			h, err := crypto.FastHash(b)
			if assert.NoError(t, err) {
				assert.Equal(t, id, base58.Encode(h[:]))
			}
			s, err := crypto.NewSignatureFromBase58(sig)
			if assert.NoError(t, err) {
				assert.True(t, crypto.Verify(spk, s, b))

			}
		}
	}
}

func TestIssueV1Validations(t *testing.T) {
	const (
		senderPK    = "BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"
		name        = "TOKEN"
		description = "This is a valid description for the token"
		quantity    = 1000000
		fee         = 100000
	)
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	if assert.NoError(t, err) {
		_, err = NewUnsignedIssueV1(spk, "TKN", description, quantity, 2, false, 0, fee)
		assert.EqualError(t, err, "incorrect number of bytes in the asset's name")
		d := strings.Repeat("x", 1010)
		_, err = NewUnsignedIssueV1(spk, name, d, quantity, 2, false, 0, fee)
		assert.EqualError(t, err, "incorrect number of bytes in the asset's description")
		_, err = NewUnsignedIssueV1(spk, name, description, 0, 2, false, 0, fee)
		assert.EqualError(t, err, "quantity should be positive")
		_, err = NewUnsignedIssueV1(spk, name, description, 10, 12, false, 0, fee)
		assert.EqualError(t, err, fmt.Sprintf("incorrect decimals, should be no more then %d", maxDecimals))
		_, err = NewUnsignedIssueV1(spk, name, description, 10, 2, false, 0, 0)
		assert.EqualError(t, err, "fee should be positive")
	}
}

func TestIssueV1SigningRoundTrip(t *testing.T) {
	const (
		seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	)
	s, err := base58.Decode(seed)
	if assert.NoError(t, err) {
		sk, pk := crypto.GenerateKeyPair(s)
		ts := uint64(time.Now().Unix() * 1000)
		tx, err := NewUnsignedIssueV1(pk, "TOKEN", "", 1000, 0, false, ts, 100000)
		if assert.NoError(t, err) {
			err := tx.Sign(sk)
			if assert.NoError(t, err) {
				assert.True(t, tx.Verify(pk))
			}
		}
	}
}

func TestIssueV1ToJSON(t *testing.T) {
	const (
		seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	)

	if s, err := base58.Decode(seed); assert.NoError(t, err) {
		sk, pk := crypto.GenerateKeyPair(s)
		ts := uint64(time.Now().Unix() * 1000)
		tx, _ := NewUnsignedIssueV1(pk, "TOKEN", "", 1000, 0, false, ts, 100000)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":3,\"version\":1,\"senderPublicKey\":\"%s\",\"name\":\"TOKEN\",\"description\":\"\",\"quantity\":1000,\"decimals\":0,\"reissuable\":false,\"timestamp\":%d,\"fee\":100000}", base58.Encode(pk[:]), ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":3,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"name\":\"TOKEN\",\"description\":\"\",\"quantity\":1000,\"decimals\":0,\"reissuable\":false,\"timestamp\":%d,\"fee\":100000}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestIssueV1BinaryRoundTrip(t *testing.T) {
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	assert.NoError(t, err)
	sk, pk := crypto.GenerateKeyPair(seed)
	tests := []struct {
		name       string
		desc       string
		quantity   uint64
		decimals   byte
		reissuable bool
		ts         uint64
		fee        uint64
	}{
		{"TOKEN", "", 1000000000, 2, false, uint64(time.Now().UnixNano() / 1000000), 100000},
		{"TOKN", "Some long description of TOKN", 1000000000, 8, true, uint64(time.Now().UnixNano() / 1000000), 100000000},
		{"abcd", "Some long description of TOKN", 123456789012345, 8, true, uint64(time.Now().UnixNano() / 1000000), 100000000},
	}
	for _, tc := range tests {
		tx, err := NewUnsignedIssueV1(pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, tc.ts, tc.fee)
		if assert.NoError(t, err) {
			b := tx.marshalBody()
			var at IssueV1
			if err := at.unmarshalBody(b); assert.NoError(t, err) {
				assert.Equal(t, *tx, at)
				assert.Equal(t, pk, at.SenderPK)
				assert.Equal(t, tc.name, at.Name)
				assert.Equal(t, tc.desc, at.Description)
				assert.Equal(t, tc.quantity, at.Quantity)
				assert.Equal(t, tc.decimals, at.Decimals)
				assert.Equal(t, tc.reissuable, at.Reissuable)
				assert.Equal(t, tc.ts, at.Timestamp)
				assert.Equal(t, tc.fee, at.Fee)
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
					var at IssueV1
					if err = at.UnmarshalBinary(b); assert.NoError(t, err) {
						assert.Equal(t, *tx, at)
						assert.Equal(t, pk, at.SenderPK)
						assert.Equal(t, tc.name, at.Name)
						assert.Equal(t, tc.desc, at.Description)
						assert.Equal(t, tc.quantity, at.Quantity)
						assert.Equal(t, tc.decimals, at.Decimals)
						assert.Equal(t, tc.reissuable, at.Reissuable)
						assert.Equal(t, tc.ts, at.Timestamp)
						assert.Equal(t, tc.fee, at.Fee)
					}
				}
			}
		}
	}
}
