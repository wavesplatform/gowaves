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
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 1000, "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid Address checksum"},
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
				b, err := tx.bodyMarshalBinary()
				assert.NoError(t, err)
				var at Payment
				err = at.bodyUnmarshalBinary(b)
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
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 10, 10, "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid Address checksum"},
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
				if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
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
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
		}
	}
}

func TestIssueV1ToJSON(t *testing.T) {
	if s, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"); assert.NoError(t, err) {
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
			b, err := tx.bodyMarshalBinary()
			assert.NoError(t, err)
			var at IssueV1
			if err := at.bodyUnmarshalBinary(b); assert.NoError(t, err) {
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

func TestTransferV1Validations(t *testing.T) {
	tests := []struct {
		addr   string
		aa     string
		fa     string
		amount uint64
		fee    uint64
		att    string
		err    string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 0, 10, "The attachment", "amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 0, "The attachment", "fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, strings.Repeat("The attachment", 100), "attachment too long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, "The attachment", "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid Address checksum"},
	}

	spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	for _, tc := range tests {
		addr, _ := NewAddressFromString(tc.addr)
		aa, _ := NewOptionalAssetFromString(tc.aa)
		fa, _ := NewOptionalAssetFromString(tc.fa)
		_, err := NewUnsignedTransferV1(spk, *aa, *fa, 0, tc.amount, tc.fee, addr, tc.att)
		assert.EqualError(t, err, tc.err, "No expected error '%s'", tc.err)
	}
}

func TestTransferV1SigningRoundTrip(t *testing.T) {
	tests := []struct {
		scheme      byte
		amountAsset string
		feeAsset    string
	}{
		{'T', "WAVES", "WAVES"},
		{'W', "", ""},
		{'T', "WAVES", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"},
		{'C', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"},
		{'D', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "WAVES"},
		{'W', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", ""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().Unix() * 1000)
		rcp, _ := NewAddressFromPublicKey(tc.scheme, pk)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV1(pk, *aa, *fa, ts, 100000000, 100000, rcp, "test attachment"); assert.NoError(t, err) {
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
		}
	}
}

func TestTransferV1FromMainNet(t *testing.T) {
	tests := []struct {
		id          string
		sig         string
		pk          string
		rcp         string
		amountAsset string
		amount      uint64
		feeAsset    string
		fee         uint64
		timestamp   uint64
		attachment  string
	}{
		{"4YQm1esnp8kXpcqLnknN3Exwfv4m8is3y4Cq1LJA8tVu", "F1s3jXWjVDX4kddHKQXHGHP6CR7Ej3b2RAWGeDimGF1i98uZR9iDbBM5VNZtGwwWJUxDitf58agLVKz6TUwhz7c", "14UoRJcmaMWPsiFjd9EzvKfvAJqAmuT7WVuVuC5PhpCH", "3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", 23000000, "", 100000, 1535033341000, ""},
		{"7RVF6fzmHSFj196bXnXvp3sbp9f7QGYiQynZh4yadkSm", "4h8SwGVTtPjKQLUMZ8wYhcdUBxXRRFtysdobDjE2sQSbJoQ7md9fcCvNSQd3Z4vDT2hgaz4PPdUaN4MHGhRvFfHL", "4ejFC3eqyGEtSVZXdx8ZKqr7m8n795qouxybBJEnVuAt", "3PKV21HqNWG2HbVSExVq7fedoA9utnW6Xbz", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1800000000, "", 100000, 1526522805052, "Doado para Filomena"},
		{"BiW2UdYVhJd1TzBAVShrBWRF1jgELwzjMF38MX2S1JeF", "4fEduvpD4fSWjJxCorwnYnyBK5o1ubdkGmZM927AeT9q1AjiPhDzXARhavNXh8Szbs8fsqwgGxduXFcU6xinTDhA", "C7hkUaAT2R1f1WUNxgR2xqpuKsNtKabJH7WSRn9dY8Pp", "3PEXG4bHcvFs2F3o99N3REuVVxjzEYTPXU8", "", 200000000, "", 100000, 1526522806022, ""},
	}
	for _, tc := range tests {
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		pk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		rcp, _ := NewAddressFromString(tc.rcp)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV1(pk, *aa, *fa, tc.timestamp, tc.amount, tc.fee, rcp, tc.attachment); assert.NoError(t, err) {
			tx.Signature = &sig
			tx.ID = &id
			b, err := tx.bodyMarshalBinary()
			assert.NoError(t, err)
			h, _ := crypto.FastHash(b)
			assert.Equal(t, *tx.ID, h)
			if r, err := tx.Verify(pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
	}
}

func TestTransferToJSON(t *testing.T) {
	tests := []struct {
		amountAsset         string
		expectedAmountAsset string
		feeAsset            string
		expectedFeeAsset    string
		attachment          string
		expectedAttachment  string
	}{
		{"", "null", "", "null", "", ""},
		{"", "null", "", "null", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
		{"", "null", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
		{"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	rcp, _ := NewAddressFromString("3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu")
	ts := uint64(time.Now().Unix() * 1000)
	for _, tc := range tests {
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV1(pk, *aa, *fa, ts, 100000000, 100000, rcp, tc.attachment); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":4,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
				assert.Equal(t, ej, string(j))
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if j, err := json.Marshal(tx); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"type\":4,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}
