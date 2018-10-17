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

func TestReissueV1Validations(t *testing.T) {
	tests := []struct {
		quantity uint64
		fee      uint64
		err      string
	}{
		{0, 100000, "quantity should be positive"},
		{100000, 0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedReissueV1(spk, aid, tc.quantity, false, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

func TestReissueV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk         string
		sig        string
		id         string
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
		timestamp  uint64
	}{
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", "5pgvcwYtkEhHzc7dEqCgxWk26z6diBgTdM5PBdFKD9e4mtYHyZSaRmWsn9j9HtQRDUaF3NzQXiVbfU4QLu4QLQY1", "9njXqFMRo7M3gvNWnHSjV3gpKd39Ex2sWgbCTPHJXvds", "2bkjzFqTMM3cQpbgGYKE8r7J73SrXFH8YfxFBRBterLt", 1000000000000000, false, 100000000, 1537436430564},
		{"3fnpXfr9dmMBCrbKuTX7T8LAjkhDuVv4TYDeJ8GjR6Ci", "5pBKr5qALAdUxctEmBNsgMQNYH2zJXqvYQmUfsaEF5vV4SjbKczJSxmy8gRgTrfFiiMT6FcRazTuBB95rW2MiUYo", "6SGeUizNdhLx8jEVcAtEsE7MGPHGYyvL2chdmPxDh51K", "Embs5w5pnVn9fdrieq8pYTMSjc72tagDKWT2Tytkn6KN", 100000000000000000, false, 100000000, 1529031382299},
		{"E6hZnnfqkLXFiqu7wGkMCJHDVyyftRQsdsdoBnD8qhJT", "5xz6MRgQegYQAxvjZJzw1rgkbt752ZvMUcQcSPUyzhXLGJM9UWNUGRXgtxvq1zysC6jFWv64rFZ5wCKnMyDz3odE", "ASj5Z2NBRGHqhfN2SRELaZpw3WW7tXcb3NGWR4TCDgS1", "5WLqNPkA3oDp1hTFCeUukTL1qvFnk9Ew7DXTtCzvoCxi", 10000000, true, 100000000, 1529051550982},
		{"3fnpXfr9dmMBCrbKuTX7T8LAjkhDuVv4TYDeJ8GjR6Ci", "2vQKaDfaLJ8uJDWux7JDDNzmpVmmoPNk1hLfUzbeFNpTfeNseBVio33TMiV96iA9GMDjBBSFFrGTKUsYEoJby7y2", "3DhpxLxUrotfXHcWKr4ivvLNVQUueJTSJL5AG4qB2E7U", "E5s6bxcRGMidPDW3QnyctDtRtxAdf5RzJE7DHvJrorCj", 99999999993000000, false, 100000000, 1529054217679},
		{"EUpjLEaJoaM2wR6QEP5FcSfCT59EUDGVsVsRgirsvUDs", "4EuP371DiBgMSwCC5VNQdQUawy8pL85UBUd4y92QjkWhbWsKcccMaJdjdZGk9HNUctnYzgNpU5ziHUibj8Z5XAmR", "8GhZFK3kZ7N7XHwJHRgxmdyLs57TrQQP8eyWPR5Bv8g1", "ANdLVFpTmpxPsCwMZq7hHMfikSVz8LBZNykziPgnZ7sn", 7000000000000, true, 100000000, 1529071786238},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedReissueV1(spk, aid, tc.quantity, tc.reissuable, tc.timestamp, tc.fee); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestReissueV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, false, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, true, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedReissueV1(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee); assert.NoError(t, err) {
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx ReissueV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.Equal(t, tx.Signature, atx.Signature)
					assert.Equal(t, pk, atx.SenderPK)
					assert.Equal(t, aid, atx.AssetId)
					assert.Equal(t, tc.quantity, atx.Quantity)
					assert.Equal(t, tc.reissuable, tx.Reissuable)
					assert.Equal(t, tc.fee, tx.Fee)
					assert.Equal(t, ts, tx.Timestamp)
				}
			}
		}
	}
}

func TestReissueV1ToJSON(t *testing.T) {
	tests := []struct {
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, false, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, true, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		if tx, err := NewUnsignedReissueV1(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":5,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":5,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestBurnV1Validations(t *testing.T) {
	tests := []struct {
		amount uint64
		fee    uint64
		err    string
	}{
		{0, 100000, "amount should be positive"},
		{100000, 0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedBurnV1(spk, aid, tc.amount, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

func TestBurnV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		asset     string
		amount    uint64
		fee       uint64
		timestamp uint64
	}{
		{"DNZgzeNN1FPaaNfbshPkQSPMpHUayBuFfpYRT7j4TyeL", "5SffFVQrUrzXjAi95C3zpW4kQiTssRShXLrAaSke1u84ud2Zf61aExUN8XUFVncBm3NB7ofDCyZ6JZ8zs2mjXTdo", "r4yYamJFy4SemyMRUHqp1B7ofXL5sqmpLTrHmLaGKt2", "8gbB78vBCcgRbtw4vE1kU9fPRMG8agYKQAMPN5DJATqa", 10037433739, 100000, 1537476952020},
		{"7SSfeSCderQcfk1p6FrGnpJxrc8JbWpe4fmcNFwA9VE8", "4gDNKM8e4ibdzhnSXXdoFEgJ1ErPV5d1yofAoB6cgEFVnNBHFQbccsNXii5WvAnWKKCG8VvEaU9SxruUBXx516Px", "3LBdgPHtT6wmATiVN5qVy6B4cUSC1UoXeh4N39B3Aawe", "J9BFrBzftppntpaXcM1XvXVZAh57KYv9hJkDh5cu1Cwi", 196498, 100000, 1537852476185},
		{"7SSfeSCderQcfk1p6FrGnpJxrc8JbWpe4fmcNFwA9VE8", "3UUrHKJP49AkBYg5D7AppGoduMWymkPWZQeKJ3MStzTfSiFdwvzZPeYkopeGFSra7UpsvYj8FBAJUTpCZVrd5RrK", "6yDvzJkZcFef7n5iVdnM5wCKeLiKLRBDNw8QWbdN1oHk", "AgEYFYvkwmQTgu7YZ7SsYiQFjAZNE1hWTbQdpveydQq1", 15570205360672, 100000, 1537852548440},
		{"DEiZqRdX3aYM6NX3B5YXBFGMZZoqw24BdAQgnTD84C2E", "2LKrYg9jaXHm9YArWeuanfpecVQfKdmkazfzqzx4jSYkturKiTDxrF2ZeSyQTeMkxj9gbokac9ZdAd7xPTKBAv2S", "5tfJxcra5cG7G2EMoBhuSHVhWUE9jf1bQaDZ1qUHAfHP", "FqhrCrn3nR6ggbnMa8HqVjPEpuAUrBkAqt3ZJa7QcvEG", 199997300000000000, 100000, 1537896381539},
		{"adbKWqVb8Sez8Fm9NqCkPiBv955rEpyDSaXP8hVJB3R", "4dnZQKJTEwDFYXStq6ja5DJEiD26DayYSXFtpx2kpSeeAaDjH1W9xx3pAXLLd9xs5AmzczeJMuEt1RccqTzvWKaH", "3fSRG7KZw2P26qbBRsEwrTQj3J3kVCbPNgmjn3GMZnmm", "9GGTr8sRMbyb8wWi6dcJGDQR5qChdJxJqgzreMTAf716", 200100000000, 100000, 1537909994643},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedBurnV1(spk, aid, tc.amount, tc.timestamp, tc.fee); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestBurnV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedBurnV1(pk, aid, tc.amount, ts, tc.fee); assert.NoError(t, err) {
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx BurnV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.Equal(t, tx.Signature, atx.Signature)
					assert.Equal(t, pk, atx.SenderPK)
					assert.Equal(t, aid, atx.AssetId)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, tx.Fee)
					assert.Equal(t, ts, tx.Timestamp)
				}
			}
		}
	}
}

func TestBurnV1ToJSON(t *testing.T) {
	tests := []struct {
		asset      string
		amount     uint64
		reissuable bool
		fee        uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, false, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, true, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		if tx, err := NewUnsignedBurnV1(pk, aid, tc.amount, ts, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":6,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":6,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestExchangeV1Validations(t *testing.T) {
	buySender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	sellSender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	mpk, _ := crypto.NewPublicKeyFromBase58("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	id, _ := crypto.NewDigestFromBase58("AkYY8M2iEts8xc21JEzwkMSmuJtH9ABGzEYeau4xWC5R")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo, _ := NewUnsignedOrder(buySender, mpk, *aa, *pa, Buy, 10, 100, 0, 0, 3)
	sbo, _ := NewUnsignedOrder(buySender, mpk, *aa, *pa, Buy, 10, 100, 0, 0, 3)
	sbo.ID = &id
	sbo.Signature = &sig
	so, _ := NewUnsignedOrder(sellSender, mpk, *aa, *pa, Sell, 9, 50, 0, 0, 3)
	sso, _ := NewUnsignedOrder(sellSender, mpk, *aa, *pa, Sell, 9, 50, 0, 0, 3)
	sso.ID = &id
	sso.Signature = &sig
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
		err     string
	}{
		{*sbo, *sso, 0, 456, 789, 987, 654, "price should be positive"},
		{*sbo, *sso, 123, 0, 789, 987, 654, "amount should be positive"},
		{*sbo, *sso, 123, 456, 0, 987, 654, "buy matcher's fee should be positive"},
		{*sbo, *sso, 123, 456, 789, 0, 654, "sell matcher's fee should be positive"},
		{*sbo, *sso, 123, 456, 789, 987, 0, "fee should be positive"},
		{*bo, *sso, 123, 456, 789, 987, 654, "buy order should be signed"},
		{*sbo, *so, 123, 456, 789, 987, 654, "sell order should be signed"},
	}
	for _, tc := range tests {
		_, err := NewUnsignedExchangeV1(tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestExchangeV1FromMainNet(t *testing.T) {
	tests := []struct {
		matcher        string
		sig            string
		id             string
		amountAsset    string
		priceAsset     string
		buyId          string
		buySender      string
		buySig         string
		buyPrice       uint64
		buyAmount      uint64
		buyTs          uint64
		buyExp         uint64
		buyFee         uint64
		sellId         string
		sellSender     string
		sellSig        string
		sellPrice      uint64
		sellAmount     uint64
		sellTs         uint64
		sellExp        uint64
		sellFee        uint64
		price          uint64
		amount         uint64
		buyMatcherFee  uint64
		sellMatcherFee uint64
		fee            uint64
		timestamp      uint64
	}{
		{"7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "YT5y1vAdvjcKaznbnNNH2Ep9hZwWrtHP7ue4vzDksbo3sp6A2STvy4fTBMutRkBwcBPgm78WQ6rFbGRG3NFWNW2", "3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N", "FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2", "WAVES",
			"AkYY8M2iEts8xc21JEzwkMSmuJtH9ABGzEYeau4xWC5R", "E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ", "5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L", 6278200, 6700000000, 1537776540542, 1540368240542, 300000,
			"DXnD6PaRWSpTKpd4PTBU5UyWX6hfJA24EguDsNpgJJ8a", "E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ", "5WfFq2jZ65dFmdmCRwkgze5h1MwisPyYY8u2KRs7Go3M4cZTajhFfAwNaLFzScWr846SajLiZsx1i7FJTMYcFjXE", 6278200, 150000000000, 1537776523784, 1540368223784, 300000,
			6278200, 6700000000, 300000, 13400, 300000, 1537776540342},
		{"7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy", "28h3szGRoGNsMAyhbPMBmnYLUUNXUssvzHnEgFUyDkY3JR8FFo4rWJe1kXzQpmCHCqeguzJbjNZRECQ9E1jS3G2G", "AafRqbsudHeeDHMPLfsk49ZioHwmFhEguQt71XXGsnQt", "8ewyQ64YgpaXdqXyfQbp2FGFJVGdGuGgc9qvrKUrCuGV", "WAVES",
			"5NyBm1CfcuuhbyQhknawhkm6u1bjNv6Avry7GFe3KQdf", "8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9", "3Dhf4jJz2SsmwkHfQyvgYRNfo2KViSSJM7DbVBprssxCYC1cpvUpSQr8nk7WQk56xCohLMfJcvDgk5bG8tW5TbVz", 7229657, 18245044292, 1539773859527, 1539774159527, 300000,
			"D1Sbzhit6F7KaMuHKwXpKiU7eGnF5NkzkdLKaSUjVzHp", "8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9", "2GS9aRvuwiKLQgKFYUup4zfXkhkKwuiLwTKV4XDCFidP96wwFiXiva51q1YqG6dNFNGtSJ2h5gHEhPPATuSGMgbb", 7229657, 5417386295, 1539773858626, 1539774158626, 300000,
			7229657, 5417386289, 89077, 299999, 300000, 1539773859535},
	}
	for _, tc := range tests {
		buySender, _ := crypto.NewPublicKeyFromBase58(tc.buySender)
		sellSender, _ := crypto.NewPublicKeyFromBase58(tc.sellSender)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		bo, _ := NewUnsignedOrder(buySender, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, tc.buyTs, tc.buyExp, tc.buyFee)
		bID, _ := crypto.NewDigestFromBase58(tc.buyId)
		bSig, _ := crypto.NewSignatureFromBase58(tc.buySig)
		bo.ID = &bID
		bo.Signature = &bSig
		so, _ := NewUnsignedOrder(sellSender, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, tc.sellTs, tc.sellExp, tc.sellFee)
		sID, _ := crypto.NewDigestFromBase58(tc.sellId)
		sSig, _ := crypto.NewSignatureFromBase58(tc.sellSig)
		so.ID = &sID
		so.Signature = &sSig
		if tx, err := NewUnsignedExchangeV1(*bo, *so, tc.price, tc.amount, tc.buyMatcherFee, tc.sellMatcherFee, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(mpk, sig, b))
			}
		}
	}
}

func TestExchangeV1BinaryRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seedA)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk := crypto.GenerateKeyPair(seedB)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo, _ := NewUnsignedOrder(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	bo.Sign(sk)
	so, _ := NewUnsignedOrder(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	so.Sign(sk)
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{*bo, *so, 123, 456, 789, 987, 654},
		{*bo, *so, 987654321, 544321, 9876, 8765, 13245},
	}
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedExchangeV1(tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx ExchangeV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.BuyOrder, atx.BuyOrder)
					assert.Equal(t, tx.SellOrder, atx.SellOrder)
					assert.Equal(t, tx.Price, atx.Price)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.BuyMatcherFee, atx.BuyMatcherFee)
					assert.Equal(t, tx.SellMatcherFee, atx.SellMatcherFee)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(msk); assert.NoError(t, err) {
				if r, err := tx.Verify(mpk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx ExchangeV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.ID, atx.ID)
					assert.Equal(t, tx.Signature, atx.Signature)
					assert.Equal(t, mpk, atx.SenderPK)
					assert.Equal(t, bo.ID, atx.BuyOrder.ID)
					assert.Equal(t, so.ID, atx.SellOrder.ID)
					assert.Equal(t, tc.price, atx.Price)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.buyFee, atx.BuyMatcherFee)
					assert.Equal(t, tc.sellFee, atx.SellMatcherFee)
					assert.Equal(t, tc.fee, tx.Fee)
					assert.Equal(t, ts, tx.Timestamp)
				}
			}
		}
	}
}

func TestExchangeV1ToJSON(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seedA)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk := crypto.GenerateKeyPair(seedB)
	tests := []struct {
		amountAsset string
		priceAsset  string
		buyPrice    uint64
		buyAmount   uint64
		sellPrice   uint64
		sellAmount  uint64
		price       uint64
		amount      uint64
		buyFee      uint64
		sellFee     uint64
		fee         uint64
	}{
		{"3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N", "FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2", 100, 10, 110, 20, 110, 10, 30000, 15000, 30000},
		{"3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N", "WAVES", 100, 10, 110, 20, 110, 10, 30000, 15000, 30000},
		{"FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2", "WAVES", 100, 10, 110, 20, 110, 10, 30000, 15000, 30000},
	}
	for _, tc := range tests {
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		exp := ts + 100*1000
		bo, _ := NewUnsignedOrder(pk, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, ts, exp, tc.fee)
		bo.Sign(sk)
		boj, _ := json.Marshal(bo)
		so, _ := NewUnsignedOrder(pk, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, ts, exp, tc.fee)
		so.Sign(sk)
		soj, _ := json.Marshal(so)
		if tx, err := NewUnsignedExchangeV1(*bo, *so, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":7,\"version\":1,\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
					base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(msk); assert.NoError(t, err) {
					if j, err := json.Marshal(tx); assert.NoError(t, err) {
						ej := fmt.Sprintf("{\"type\":7,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
						assert.Equal(t, ej, string(j))
					}
				}
			}
		}
	}
}
