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

func TestGenesisFromMainNet(t *testing.T) {
	tests := []struct {
		sig       string
		timestamp uint64
		recipient string
		amount    uint64
	}{
		{"2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8", 1465742577614, "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9999999500000000},
		{"2TsxPS216SsZJAiep7HrjZ3stHERVkeZWjMPFcvMotrdGpFa6UCCmoFiBGNizx83Ks8DnP3qdwtJ8WFcN9J4exa3", 1465742577614, "3P8JdJGYc7vaLu4UXUZc1iRLdzrkGtdCyJM", 100000000},
		{"3gF8LFjhnZdgEVjP7P6o1rvwapqdgxn7GCykCo8boEQRwxCufhrgqXwdYKEg29jyPWthLF5cFyYcKbAeFvhtRNTc", 1465742577614, "3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy", 100000000},
		{"5hjSPLDyqic7otvtTJgVv73H3o6GxgTBqFMTY2PqAFzw2GHAnoQddC4EgWWFrAiYrtPadMBUkoepnwFHV1yR6u6g", 1465742577614, "3P9o3ZYwtHkaU1KxsKkFjJqJKS3dLHLC9oF", 100000000},
		{"ivP1MzTd28yuhJPkJsiurn2rH2hovXqxr7ybHZWoRGUYKazkfaL9MYoTUym4sFgwW7WB5V252QfeFTsM6Uiz3DM", 1465742577614, "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3", 100000000},
		{"29gnRjk8urzqc9kvqaxAfr6niQTuTZnq7LXDAbd77nydHkvrTA4oepoMLsiPkJ8wj2SeFB5KXASSPmbScvBbfLiV", 1465742577614, "3PBWXDFUc86N2EQxKJmW8eFco65xTyMZx6J", 100000000},
	}
	for _, tc := range tests {
		id, _ := base58.Decode(tc.sig)
		if rcp, err := NewAddressFromString(tc.recipient); assert.NoError(t, err) {
			if tx, err := NewUnsignedGenesis(rcp, tc.amount, tc.timestamp); assert.NoError(t, err) {
				if err := tx.GenerateSigID(); assert.NoError(t, err) {
					assert.Equal(t, id, tx.ID[:])
					assert.Equal(t, tc.amount, tx.Amount)
					assert.Equal(t, tc.recipient, tx.Recipient.String())
					assert.Equal(t, tc.timestamp, tx.Timestamp)
					b, err := tx.MarshalBinary()
					assert.NoError(t, err)
					var at Genesis
					err = at.UnmarshalBinary(b)
					assert.NoError(t, err)
					assert.Equal(t, *tx, at)
				}
			}
		}
	}
}

func TestGenesisValidations(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		err       string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, "amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 1000, "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ: invalid Address checksum"},
	}
	for _, tc := range tests {
		rcp, _ := NewAddressFromString(tc.recipient)
		_, err := NewUnsignedGenesis(rcp, tc.amount, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestGenesisToJSON(t *testing.T) {
	const addr = "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ"
	if rcp, err := NewAddressFromString(addr); assert.NoError(t, err) {
		ts := uint64(time.Now().Unix() * 1000)
		if tx, err := NewUnsignedGenesis(rcp, 1000, ts); assert.NoError(t, err) {
			tx.GenerateSigID()
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":1,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"timestamp\":%d,\"recipient\":\"%s\",\"amount\":1000}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), ts, tx.Recipient.String())
				assert.Equal(t, ej, string(j))
			}
		}
	}
}

func TestPaymentFromMainNet(t *testing.T) {
	tests := []struct {
		sig       string
		timestamp uint64
		spk       string
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"2ZojhAw3r8DhiHD6gRJ2dXNpuErAd4iaoj5NSWpfYrqppxpYkcXBHzSAWTkAGX5d3EeuAUS8rZ4vnxnDSbJU8MkM", 1465754870341, "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7", 20999990, 1},
		{"5cQLvZVUZqYcC75u5vXydpPoxKeazyiNtKgtz4DSyQboDSyefxcQEihwN9er772DbFDuaBRDLQHbT9CJiezk8sba", 1465825839722, "vAyFRfGG225MjUXo2VXhLfh2F6utsGkG782HuKi5fRp", "3P9v6SjRKUZPZMG1aL2oTznGZHBvNr21EQS", 99999999, 1},
		{"396pxC3YjVMjYQF7S9Xk3ntCjEJz4ip91ckux6ni4qpNEHbkyzqhSeYzyiVaUUM94uc21nGe8qwurGFDdzynrCHZ", 1466531340683, "2DAbbF2XuQPc3ePzKdxncsdMUzjSjEGC4nHx7kA3s1jm", "3PFrwqFZpoTzwKYq8NUALrtALP1oDvixt8z", 49310900000000, 1},
	}
	for _, tc := range tests {
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		spk, _ := crypto.NewPublicKeyFromBase58(tc.spk)
		if rcp, err := NewAddressFromString(tc.recipient); assert.NoError(t, err) {
			if tx, err := NewUnsignedPayment(spk, rcp, tc.amount, tc.fee, tc.timestamp); assert.NoError(t, err) {
				assert.Equal(t, tc.spk, base58.Encode(tx.SenderPK[:]))
				assert.Equal(t, tc.amount, tx.Amount)
				assert.Equal(t, tc.recipient, tx.Recipient.String())
				assert.Equal(t, tc.timestamp, tx.Timestamp)
				assert.Equal(t, tc.fee, tx.Fee)
				b := tx.marshalBody()
				var at Payment
				err = at.unmarshalBody(b)
				assert.NoError(t, err)
				assert.Equal(t, *tx, at)
				tx.Signature = &sig
				tx.ID = &sig
				tx.Verify(spk)
				b, _ = tx.MarshalBinary()
				err = at.UnmarshalBinary(b)
				assert.NoError(t, err)
				assert.Equal(t, *tx, at)
			}
		}
	}
}

func TestPaymentValidations(t *testing.T) {
	tests := []struct {
		spk       string
		recipient string
		amount    uint64
		fee       uint64
		err       string
	}{
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, 10, "amount should be positive"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 10, 0, "fee should be positive"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 10, 10, "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ: invalid Address checksum"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.spk)
		rcp, _ := NewAddressFromString(tc.recipient)
		_, err := NewUnsignedPayment(spk, rcp, tc.amount, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestPaymentToJSON(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(s)
	rcp, _ := NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	ts := uint64(time.Now().Unix() * 1000)
	if tx, err := NewUnsignedPayment(pk, rcp, 1000, 10, ts); assert.NoError(t, err) {
		err = tx.Sign(sk)
		assert.NoError(t, err)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":2,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":1000,\"fee\":10,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(tx.SenderPK[:]), tx.Recipient.String(), ts)
			assert.Equal(t, ej, string(j))
		}
	}
}

func TestIssueV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk  string
		sig string
		id  string
	}{
		{"BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6", "6JAr35fMADxhhK5jEXCKBzZAMCBoXBPcW4D9iaBDnhATxQ7Dk5EgJKBSWCeauqftSUVWgY79bMjdxqomCRxafFd", "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58(tc.pk)
		if assert.NoError(t, err) {
			tx, err := NewUnsignedIssueV1(spk, "WBTC", "Bitcoin Token", 2100000000000000, 8, false, 1480690876160, 100000000)
			if assert.NoError(t, err) {
				b := tx.marshalBody()
				h, err := crypto.FastHash(b)
				if assert.NoError(t, err) {
					assert.Equal(t, tc.id, base58.Encode(h[:]))
				}
				s, err := crypto.NewSignatureFromBase58(tc.sig)
				if assert.NoError(t, err) {
					assert.True(t, crypto.Verify(spk, s, b))

				}
			}
		}
	}
}

func TestIssueV1Validations(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		quantity uint64
		decimals byte
		fee      uint64
		err      string
	}{
		{"TKN", "This is a valid description for the token", 1000000, 2, 100000, "incorrect number of bytes in the asset's name"},
		{"TOKEN", strings.Repeat("x", 1010), 1000000, 2, 100000, "incorrect number of bytes in the asset's description"},
		{"TOKEN", "This is a valid description for the token", 0, 2, 100000, "quantity should be positive"},
		{"TOKEN", "This is a valid description for the token", 100000, 12, 100000, fmt.Sprintf("incorrect decimals, should be no more then %d", maxDecimals)},
		{"TOKEN", "This is a valid description for the token", 100000, 2, 0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		if assert.NoError(t, err) {
			_, err = NewUnsignedIssueV1(spk, tc.name, tc.desc, tc.quantity, tc.decimals, false, 0, tc.fee)
			assert.EqualError(t, err, tc.err)
		}
	}
}

func TestIssueV1SigningRoundTrip(t *testing.T) {
	const seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
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
	const seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
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
