package proto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

func TestIssueV2Validations(t *testing.T) {
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
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedIssueV2('T', spk, tc.name, tc.desc, tc.quantity, tc.decimals, false, []byte{}, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

func TestIssueV2FromMainNet(t *testing.T) {
	tests := []struct {
		pk         string
		sig        string
		id         string
		name       string
		desc       string
		quantity   uint64
		reissuable bool
		decimals   byte
		fee        uint64
		timestamp  uint64
	}{
		{"H2WC7vaqRYfTgqQjFj9MgN2GK4VdLDqGX3pzwVmstt2S", "42aGhLG57MBVK1fmGyzw1Hpr9d5Ms9oHTqyWJJPkXVYfZNoXzZYK72EdqzdFV6UVVhDcktbMLjzMeHbG8sTK9ZG8", "5VNzokhFHssa1JeiyvHcyt7ndUSLc9NseSAwXTse6iSj", "Holacoin", "shhshs", 4555555500000000, true, 8, 100000000, 1541463569575},
		{"E24KEJzmRZSBXUAMs3WSZvA5MwYYTYhuYU19N2iMQKMb", "43Qt3QP2ssFVTG9u9urq27AinaTrD5SyYquGV141Va2cS3cYJFqHB2WZqy3h4b4UEu6Q2zvcAVcnSoegm6awi8iQ", "4G6tscsy4MWaCV4AS8FG1R44wSQdJjzPspxrTV89TvAF", "watriom", "Purchase of goods and services in the Internet network by making a transaction in the shortest time possible and the lowest fee.", 100000000000000000, true, 8, 100000000, 1541495473339},
		{"CLrzCSLyUkwQRvEJLGUx6eqHeXMabjsN3h9b2hYy4bSV", "3EWzu49xP7d36kTbceE1p9qZivaQK53nWAX5nnbq4ZPzLTZRmAuXEdNe3RDAWg7tLvFv317pjQHStpfwn99Aua4E", "2kpvCwZoNp93YyDBYAeY8YvzgqUG8U8jT1UjEKnUZoEY", "HitlerCoin", "國家社會主義國際所發行之貨幣\n由國家社會主義學會發行，營運\n\nThe digital currency of National Socialism International will be distributed and serviced \nby the National Socialism lnstitut", 1000000000000000000, false, 4, 100000000, 1541522692952},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		if tx, err := NewUnsignedIssueV2('W', spk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, []byte{}, tc.timestamp, tc.fee); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestIssueV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chain      byte
		name       string
		desc       string
		quantity   uint64
		decimals   byte
		reissuable bool
		script     string
		fee        uint64
	}{
		{'T', "TOKEN", "This is a valid description for the token", 12345, 4, true, "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 100000},
		{'W', "TOKEN", "This is a valid description for the token", 100000, 8, false, "", 100000},
		{'X', "TOKEN", "This is a valid description for the token", 9876543210, 2, true, "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 123456},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		if tx, err := NewUnsignedIssueV2(tc.chain, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx IssueV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.ChainID, atx.ChainID)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.Name, atx.Name)
					assert.Equal(t, tx.Description, atx.Description)
					assert.Equal(t, tx.Quantity, atx.Quantity)
					assert.Equal(t, tx.Decimals, atx.Decimals)
					assert.Equal(t, tx.Reissuable, atx.Reissuable)
					assert.ElementsMatch(t, tx.Script, atx.Script)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx IssueV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, tc.chain, atx.ChainID)
					assert.Equal(t, tc.name, atx.Name)
					assert.Equal(t, tc.desc, atx.Description)
					assert.Equal(t, tc.quantity, atx.Quantity)
					assert.Equal(t, tc.decimals, atx.Decimals)
					assert.Equal(t, tc.reissuable, atx.Reissuable)
					assert.Equal(t, tc.script, base64.StdEncoding.EncodeToString(atx.Script))
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestIssueV2ToJSON(t *testing.T) {
	tests := []struct {
		chain      byte
		name       string
		desc       string
		quantity   uint64
		decimals   byte
		reissuable bool
		script     string
		fee        uint64
	}{
		{'T', "TOKEN", "This is a valid description for the token", 12345, 4, true, "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 100000},
		{'W', "SHMOKEN", "This is a valid description for the token", 100000, 8, false, "", 100000},
		{'X', "POKEN", "This is a valid description for the token", 9876543210, 2, true, "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 123456},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		if tx, err := NewUnsignedIssueV2(tc.chain, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":3,\"version\":2,\"senderPublicKey\":\"%s\",\"name\":\"%s\",\"description\":\"%s\",\"quantity\":%d,\"decimals\":%d,\"reissuable\":%v,\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
					base58.Encode(pk[:]), tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, tc.script, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":3,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"name\":\"%s\",\"description\":\"%s\",\"quantity\":%d,\"decimals\":%d,\"reissuable\":%v,\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, tc.script, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
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
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 0, 10, "The attachment", "failed to create TransferV1 transaction: amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 0, "The attachment", "failed to create TransferV1 transaction: fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, strings.Repeat("The attachment", 100), "failed to create TransferV1 transaction: attachment too long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, "The attachment", "failed to create TransferV1 transaction: invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid Address checksum"},
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

func TestTransferV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme              byte
		amountAsset         string
		expectedAmountAsset string
		feeAsset            string
		expectedFeeAsset    string
		amount              uint64
		fee                 uint64
		attachment          string
	}{
		{'T', "WAVES", "WAVES", "WAVES", "WAVES", 1234, 56789, "test attachment"},
		{'W', "", "WAVES", "", "WAVES", 1234567890, 9876543210, ""},
		{'T', "WAVES", "WAVES", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", 121212121, 343434, "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"},
		{'C', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", 12345, 67890, ""},
		{'D', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "WAVES", "WAVES", 567890, 1234, "xxx"},
		{'W', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "", "WAVES", 10, 20, ""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		rcp, _ := NewAddressFromPublicKey(tc.scheme, pk)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV1(pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, tc.attachment); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx TransferV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.AmountAsset.Present, atx.AmountAsset.Present)
					assert.ElementsMatch(t, tx.AmountAsset.ID, atx.AmountAsset.ID)
					assert.Equal(t, tx.FeeAsset.Present, atx.FeeAsset.Present)
					assert.ElementsMatch(t, tx.FeeAsset.ID, atx.FeeAsset.ID)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
					assert.Equal(t, tx.Attachment.String(), atx.Attachment.String())
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx TransferV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.Equal(t, tc.expectedAmountAsset, atx.AmountAsset.String())
					assert.Equal(t, tc.expectedFeeAsset, atx.FeeAsset.String())
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
					assert.Equal(t, tc.attachment, atx.Attachment.String())
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

func TestTransferV1ToJSON(t *testing.T) {
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

func TestTransferV2Validations(t *testing.T) {
	tests := []struct {
		addr   string
		aa     string
		fa     string
		amount uint64
		fee    uint64
		att    string
		err    string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 0, 10, "The attachment", "failed to create TransferV2 transaction: amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 0, "The attachment", "failed to create TransferV2 transaction: fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, strings.Repeat("The attachment", 100), "failed to create TransferV2 transaction: attachment too long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", "4UY7UjzhRxKYyLh6mtiPkZpC73HFLE9DFNGKs7ju6Ai3", 1000, 10, "The attachment", "failed to create TransferV2 transaction: invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid Address checksum"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	for _, tc := range tests {
		addr, _ := NewAddressFromString(tc.addr)
		aa, _ := NewOptionalAssetFromString(tc.aa)
		fa, _ := NewOptionalAssetFromString(tc.fa)
		_, err := NewUnsignedTransferV2(spk, *aa, *fa, 0, tc.amount, tc.fee, addr, tc.att)
		assert.EqualError(t, err, tc.err, "No expected error '%s'", tc.err)
	}
}

func TestTransferV2FromMainNet(t *testing.T) {
	tests := []struct {
		id          string
		sig         string
		spk         string
		rcp         string
		amountAsset string
		amount      uint64
		feeAsset    string
		fee         uint64
		timestamp   uint64
		attachment  string
	}{
		{"93H1i2jgP21Eh4Q5uzwmCYCVfGHZcAMzpC6PPbwvCSTs", "4jbfvXGiqsaKkZso6ykNMQZDARewvBjKxgaz55jF4g6tBVPgxv5qChSvBYdHRGHjUdXbG3CZ3PUNBBK3eoiuRfVt", "6tbTkJukCZ4qX13ucXVeUV2aN88t9ypi1MADZb9PfFQD", "3P5mTiUpUnb1eM19udtM8QyNLBGf6VjS19j", "GryqKQBmTZGZnbZ4efrQvNGNpeLM83djWSNJBWuhZg5H", 566, "9PVyxDPUjauYafvq83JTXvHQ8nPnxwKA7siUFcqthCDJ", 1000000000, 1541593367281, ""},
		{"HE7jA4xjRiqdNVEP4jSXAY8FEy412MGKTD1hqWtpnYrZ", "33VU7yYd6bLrHf5VBCtR5iMpxDhMwWWdfJMSzBUw38WsSjCsKhfjerBiatDtJNpPPx8FK8cqX4kKeb5XiUhSv7ev", "6cDEgFTH9mZwuJjRubE4ZzSjrCvofzn24M7jmw5oTu5p", "3PKi8kvBCMUZnPFVRDBMzYaY49wLf7TurEe", "FGJQGTG13wKXSaYB4JJ6But7Ui3iRq5ZA9DTFsNTYJvt", 9788200000000, "", 100000, 1541593585115, "0x09F7f8d4f0e4BCC89073318759179EB1e5cFC500"},
		{"ERhAQmKArX6Yy2iC5N9S9aV9xPhnabyhwMBXufSZkgEw", "36u5TudkkRDE6V67jydUCAE3Vh8xf7Yv5FM8M53DJHtyhjdEPJtFqLBAnEFaXUEyyV2s7qPV8DyL3nyhDJ7YBCcH", "9zMXKmq3tWJJmezrkaYjmpiD3LZkFbhiwy9AP2r4CBnC", "3PGxhF7LtybhRdZfErxBTN4ZDDJjLUQW8Rb", "", 6500000, "", 100000, 1541593775634, "Send"},
	}
	for _, tc := range tests {
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		spk, _ := crypto.NewPublicKeyFromBase58(tc.spk)
		rcp, _ := NewAddressFromString(tc.rcp)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV2(spk, *aa, *fa, tc.timestamp, tc.amount, tc.fee, rcp, tc.attachment); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestTransferV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme              byte
		amountAsset         string
		expectedAmountAsset string
		feeAsset            string
		expectedFeeAsset    string
		amount              uint64
		fee                 uint64
		attachment          string
	}{
		{'T', "WAVES", "WAVES", "WAVES", "WAVES", 1234, 56789, "test attachment"},
		{'W', "", "WAVES", "", "WAVES", 1234567890, 9876543210, ""},
		{'T', "WAVES", "WAVES", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", 121212121, 343434, "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"},
		{'C', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", 12345, 67890, ""},
		{'D', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "WAVES", "WAVES", 567890, 1234, "xxx"},
		{'W', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "", "WAVES", 10, 20, ""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		rcp, _ := NewAddressFromPublicKey(tc.scheme, pk)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV2(pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, tc.attachment); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx TransferV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.AmountAsset.Present, atx.AmountAsset.Present)
					assert.ElementsMatch(t, tx.AmountAsset.ID, atx.AmountAsset.ID)
					assert.Equal(t, tx.FeeAsset.Present, atx.FeeAsset.Present)
					assert.ElementsMatch(t, tx.FeeAsset.ID, atx.FeeAsset.ID)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
					assert.Equal(t, tx.Attachment.String(), atx.Attachment.String())
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx TransferV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, pk, atx.SenderPK)
					assert.Equal(t, tc.expectedAmountAsset, atx.AmountAsset.String())
					assert.Equal(t, tc.expectedFeeAsset, atx.FeeAsset.String())
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
					assert.Equal(t, tc.attachment, atx.Attachment.String())
				}
			}
		}
	}
}

func TestTransferV2ToJSON(t *testing.T) {
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
	ts := uint64(time.Now().UnixNano() / 1000000)
	for _, tc := range tests {
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		if tx, err := NewUnsignedTransferV2(pk, *aa, *fa, ts, 100000000, 100000, rcp, tc.attachment); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":4,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
				assert.Equal(t, ej, string(j))
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if j, err := json.Marshal(tx); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"type\":4,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
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
		{0, 100000, "failed to create ReissueV1 transaction: quantity should be positive"},
		{100000, 0, "failed to create ReissueV1 transaction: fee should be positive"},
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
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx ReissueV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.AssetID, atx.AssetID)
					assert.Equal(t, tx.Reissuable, atx.Reissuable)
					assert.Equal(t, tx.Quantity, atx.Quantity)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx ReissueV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, aid, atx.AssetID)
					assert.Equal(t, tc.quantity, atx.Quantity)
					assert.Equal(t, tc.reissuable, atx.Reissuable)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
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

func TestReissueV2Validations(t *testing.T) {
	tests := []struct {
		quantity uint64
		fee      uint64
		err      string
	}{
		{0, 100000, "failed to create ReissueV2 transaction: quantity should be positive"},
		{100000, 0, "failed to create ReissueV2 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedReissueV2('T', spk, aid, tc.quantity, false, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

func TestReissueV2FromMainNetAndTestNet(t *testing.T) {
	tests := []struct {
		chain      byte
		pk         string
		sig        string
		id         string
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
		timestamp  uint64
	}{
		{'T', "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "QMWAKh61ZecMLZoEFGCd3PsdnEyQLARtyArHqb6tWu9GURBdR7X7QYt7t1EK7kgUKEkt3mqAZEgke4xDcW7Quyt", "Jrj5KTbHCpiViTaJJDcPuCqAxnporJMbmPCz9qTXm5e", "HGY85s7fygwNJLfpbXC97YL4uVAsMc5U6B6WuQTdJ7TQ", 2, true, 100000000, 1541508749585},
		{'T', "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "5ZfqE2LCom6CHqG1RQP1ftkQvt68gipVfs74itnCbWxwidWPeRz6QCcppVKSXcrELqMBNjPtXnHKyZtWzhMCiG99", "BFdx9xUxXaaV3Uk31xibaCMdX9Ygh8aHC2QtZZWf2QEA", "B7REqjQshrX3n9aGQnAJtQ5NCtVZVc7KXXU4tTZGahJV", 20, true, 100000000, 1541509020176},
		{'T', "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "5rqyi1s8duEtfJebiNFcKCHWVWF4Mc8dGcC8aNDqHRBh1TrDeuwtCr2hHFGvZpE79hLFmeZdLtkvszCkaF393xrT", "UEED7c4F7rfkxtYKev5hHkRbfn4eoCuy7u5stB8WmAa", "74mTmy8ugd21bmQTd56KGT3nJHUrHyy1C4en3RGgyiuC", 10000, true, 100400000, 1541598574945},
		{'T', "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "5ZqseZWDgHsTwTBP61WT66GhSPBwQc8vSdi6KUFsMECEMAuAad3ZC8cSbcpo8qRzFcBsHcypcEuDTmg1g4ZanUbn", "42DgLrFq9Pnvaw3dwH5gym1Sn3i7FkpKHWZVaZH9YwGC", "AKd2QRLsaSFzq3JJnDo6iedUtp4CianMDJucf5M3M1Jx", 10000, true, 100400000, 1541584022814},
		{'T', "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "2dE15W7CqQJa8SLx7kFskM6JZrzYhLXv1hz6mDTU7StYB9JsTAcPdDe3VQtKtjyoURPXpqrwemBbJzqYPzf3eyQG", "7SugP79gdR2q3G2Lw4eXtuAjAxy1bDzHXhN4tJUqu8Y4", "VE6fUNmTYwkr9FABhsXvETtcejgQAEVAVcMGvp2V5Kv", 100, true, 100000000, 1541516491149},
		{'T', "PYLg2tgZFWvcDNA9xnroZWxSwjNbc1TM6Bik8Mq2qjw", "h5snYP7ppP3xogBvaAq4a1tC6i9A5Jaf7rRYDjF8iSyrZej52E3DBhvpaP1qqAsAX36hgtMuqHRGSgyYx32YfyW", "GMkDMTujtKFz72JZLmWqwkWrhFX8AH6tuCxnLvwzDcFj", "FvzAYgre224geHwECqCk7EmiQckaSNVLiCzkKAvNtwLV", 100, true, 100400000, 1541575029371},
		{'W', "F8ZGP4Xf5cjfHpPJHznt9BtXCrVYRcdjKHrfsu6psUGe", "5jWNY19Zc4Db1sb9oMBt6VR9xcu4U6pKJgT2c8iMPzuheDk9nJ4VrSXaKQgqhAp8vi4cEBFKopvc65JMKq65ctEH", "9wKSt9ij2YYiWgCYd84UcQiVuL6r9YurhVJ8Nikp2URy", "edgxQYrr8usWtBXKQ1Q4Nv712F5fcx6fWRqmuW3eEKz", 95000, true, 100000000, 1541512609669},
		{'W', "7kGS1HeJ2gkfWz8FhxwAEjdTRzkJJiFGgAeNVxq1K7TZ", "AqcuR4MzuTVeKwMVxbymAeWUryGByZfxDFFGfp8GnXxoqo5h5VdbdL2Zv4R92py8T1jeCpgHqErKF938iVmhnsf", "UFVrNKhFQnZDGKLrbu6S1oPsYMhgU6fZVXSxCRyUfgY", "5siW5tqvYLnfgeXttU7F9dZ3xrE8UHVtpqT9Ye71GBuJ", 20000, true, 100000000, 1541522867378},
		{'W', "GshF9vnYfAm2fYCTJj9rRGc2VyC1579Kq91xL2Rr8KDU", "GzTJgnicvCkJ2UHuE1h2upgiwU2FU2JfHpRJKwPNLLthsxyF3Xvv7ZAbGgXgBGMp2vcNkMrPGea7K2zERiRacKy", "HCBy6qMfDLgnbMokyDGYezttp9zdgBC8SjUjtqtuBnXv", "65UMyqN6yBmnSg8xjb3RBDsxoW5iYYc7rJeyL5rKQJ5i", 9000000000000, false, 100000000, 1541613498199},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedReissueV2(tc.chain, spk, aid, tc.quantity, tc.reissuable, tc.timestamp, tc.fee); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestReissueV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chain      byte
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
	}{
		{'T', "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, false, 1234567890},
		{'W', "6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, true, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedReissueV2(tc.chain, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx ReissueV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.ChainID, atx.ChainID)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.AssetID, atx.AssetID)
					assert.Equal(t, tx.Reissuable, atx.Reissuable)
					assert.Equal(t, tx.Quantity, atx.Quantity)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx ReissueV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, aid, atx.AssetID)
					assert.Equal(t, tc.quantity, atx.Quantity)
					assert.Equal(t, tc.reissuable, atx.Reissuable)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestReissueV2ToJSON(t *testing.T) {
	tests := []struct {
		chain      byte
		asset      string
		quantity   uint64
		reissuable bool
		fee        uint64
	}{
		{'T', "8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, false, 1234567890},
		{'W', "6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, true, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		if tx, err := NewUnsignedReissueV2(tc.chain, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":5,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":5,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
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
		{0, 100000, "failed to create BurnV1 transaction: amount should be positive"},
		{100000, 0, "failed to create BurnV1 transaction: fee should be positive"},
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
					assert.Equal(t, aid, atx.AssetID)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestBurnV1ToJSON(t *testing.T) {
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

func TestBurnV2Validations(t *testing.T) {
	tests := []struct {
		chain  byte
		amount uint64
		fee    uint64
		err    string
	}{
		{'T', 0, 100000, "failed to create BurnV2 transaction: amount should be positive"},
		{'W', 100000, 0, "failed to create BurnV2 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedBurnV2(tc.chain, spk, aid, tc.amount, 0, tc.fee)
		assert.EqualError(t, err, tc.err)
	}
}

func TestBurnV2FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		asset     string
		amount    uint64
		fee       uint64
		timestamp uint64
	}{
		{"GkBGp7A4PAwixahc8Uo7oiapAZuqZURhfs6HHRBr4wfC", "4qxJSzjffE5Bqhff69ZkNHFHwbh3spgrRQhzAprGmFGSUmR74PrUSrvVkkqvDAEncbgGb2wwMDZr57cVuH4jvxei", "Huikmb3ZL25RZWua427RLo2Sqq7u18nu8gpgbkDvpqBV", "BETU9FPZL4yYL7rgTL8ZjXV6QHoFrSELkhdetiykC1Wc", 935066866613, 100000, 1541550501417},
		{"GkBGp7A4PAwixahc8Uo7oiapAZuqZURhfs6HHRBr4wfC", "3gavc4q9VoeRdu7xxM42PsTG9kurh2BJtMBtYyb1ndhFHBbzhRFnSSkRhEQyPvomb5fKAN4YH2t1rMqvBr5fH1ie", "EHeGW2T3bYnv6kvWEE4BH4zg51FYk4PJbvHiTSTQtM33", "7q8DjdQw2tpc27mos3LZJNpmCcNbXDsrqLdSpZgdq1tA", 1600363277644, 100000, 1541551098417},
		{"B3f8VFh6T2NGT26U7rHk2grAxn5zi9iLkg4V9uxG6C8q", "2Y9T9E4xZjPZGZnEfVWPeQdTR7L9pxebuif9NmVf6q6Tc5hiByTT9PuPHDGVLZWtMiCrmih5RfUxLQ71dUY4PF4g", "Bm3QwrxZau51JSuYa8ckVSv26biAYwfeEMfwVqtnMjmH", "XjgUdDr36kHFYBCi4J3Eh3f4v4zXwquYnUGNYgRjLVX", 1, 100000, 1541675830387},
		{"HCDzebMWj8cmKyEq4BhD6TgTnEqMbdiSwh76zrAx7j7D", "3dGMKjDiJAWUp3KG87tdnTGJAdgs99dWwhdKUyV3dq4xdsBDh9Rf1gwA64rfPyA2UVBdARRKP9bcSQv9vixzfRTe", "FsBLfdkCCGz9wDY9WmAmu1HP244Nhc1QtnzwBjCJj2bN", "A7t6CtfSLbqhgM93oz2gbUzE8MxGEqCFDYVHEMxvN17i", 47038, 100000, 1541669733714},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedBurnV2('W', spk, aid, tc.amount, tc.timestamp, tc.fee); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestBurnV2BinaryRoundTrip(t *testing.T) {
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
		if tx, err := NewUnsignedBurnV2('T', pk, aid, tc.amount, ts, tc.fee); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx BurnV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.ChainID, atx.ChainID)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.AssetID, atx.AssetID)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx BurnV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, aid, atx.AssetID)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestBurnV2ToJSON(t *testing.T) {
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
		ts := uint64(time.Now().Unix() * 1000)
		if tx, err := NewUnsignedBurnV2('T', pk, aid, tc.amount, ts, tc.fee); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":6,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":6,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
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
	bo, _ := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, 10, 100, 0, 0, 3)
	sbo, _ := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, 10, 100, 0, 0, 3)
	sbo.ID = &id
	sbo.Signature = &sig
	so, _ := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, 9, 50, 0, 0, 3)
	sso, _ := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, 9, 50, 0, 0, 3)
	sso.ID = &id
	sso.Signature = &sig
	tests := []struct {
		buy     OrderV1
		sell    OrderV1
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
		buyID          string
		buySender      string
		buySig         string
		buyPrice       uint64
		buyAmount      uint64
		buyTs          uint64
		buyExp         uint64
		buyFee         uint64
		sellID         string
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
		bo, _ := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, tc.buyTs, tc.buyExp, tc.buyFee)
		bID, _ := crypto.NewDigestFromBase58(tc.buyID)
		bSig, _ := crypto.NewSignatureFromBase58(tc.buySig)
		bo.ID = &bID
		bo.Signature = &bSig
		so, _ := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, tc.sellTs, tc.sellExp, tc.sellFee)
		sID, _ := crypto.NewDigestFromBase58(tc.sellID)
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
	bo, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	bo.Sign(sk)
	so, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	so.Sign(sk)
	tests := []struct {
		buy     OrderV1
		sell    OrderV1
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
				if _, err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
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
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
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
		bo, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, ts, exp, tc.fee)
		bo.Sign(sk)
		boj, _ := json.Marshal(bo)
		so, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, ts, exp, tc.fee)
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

func TestExchangeV2Validations(t *testing.T) {
	buySender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	sellSender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	mpk, _ := crypto.NewPublicKeyFromBase58("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	id, _ := crypto.NewDigestFromBase58("AkYY8M2iEts8xc21JEzwkMSmuJtH9ABGzEYeau4xWC5R")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	sbo, _ := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, 10, 100, 0, 0, 3)
	sbo.ID = &id
	sbo.Signature = &sig
	sso, _ := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, 9, 50, 0, 0, 3)
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
	}
	for _, tc := range tests {
		_, err := NewUnsignedExchangeV2(tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, 0)
		assert.EqualError(t, err, tc.err, fmt.Sprintf("expected error: %s", tc.err))
	}
}

func TestExchangeV2FromTestNet(t *testing.T) {
	tests := []struct {
		matcher        string
		sig            string
		id             string
		amountAsset    string
		priceAsset     string
		buyID          string
		buySender      string
		buySig         string
		buyPrice       uint64
		buyAmount      uint64
		buyTs          uint64
		buyExp         uint64
		buyFee         uint64
		sellID         string
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
		{"8QUAqtTckM5B8gvcuP7mMswat9SjKUuafJMusEoSn1Gy", "4xTXXWYfjYkeTqrodaNqDhWjRSJSfTWNi7dPXWbKi4BtqS9vQotr9ovYT2g67aBYsgKtHaWfup6AaPV1BGZaihPn", "H8rykXmgXeaP9CEPdkRd5iAThwhGQxpNnou1cvNBmhjw", "DWgwcZTMhSvnyYCoWLRUXXSH1RSkzThXLJhww9gwkqdn", "WAVES",
			"AMunMc338taaSC5hEbVwy4FCQp7BJnDzDj1h9prX2ftW", "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "5THpubijxUjMchUBnMN6xzDjLZLbu2PTkLvM5imLqqxCtxhnG4bdQsxnscdjBut4uhi4HWV8h948aYmtegAApqx6", 1400000000, 100000, 1542025349897, 1544530949898, 300000,
			"Gu1GTAyz3mxZwNrcMkS8j742gb4iHg9yUj8Y6QRNfG4F", "FB5ErjREo817duEBBQUqUdkgoPctQJEYuG3mU7w3AYjc", "2EcB3XkoHsUTdALPvhRhTHHj3eJBfESeYRAQ2wJmEpahCMRXyMY5KLUR4gf5YtHCKSmhEXJxrwa2V2JA8esZArtt", 1400000000, 100000, 1542025338886, 1544530938886, 300000,
			1400000000, 100000, 300000, 300000, 300000, 1542025350044},
	}
	for _, tc := range tests {
		buySender, _ := crypto.NewPublicKeyFromBase58(tc.buySender)
		sellSender, _ := crypto.NewPublicKeyFromBase58(tc.sellSender)
		mpk, _ := crypto.NewPublicKeyFromBase58(tc.matcher)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		pa, _ := NewOptionalAssetFromString(tc.priceAsset)
		bo, _ := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, tc.buyTs, tc.buyExp, tc.buyFee)
		bID, _ := crypto.NewDigestFromBase58(tc.buyID)
		bSig, _ := crypto.NewSignatureFromBase58(tc.buySig)
		bo.ID = &bID
		bo.Signature = &bSig
		so, _ := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, tc.sellTs, tc.sellExp, tc.sellFee)
		sID, _ := crypto.NewDigestFromBase58(tc.sellID)
		sSig, _ := crypto.NewSignatureFromBase58(tc.sellSig)
		so.ID = &sID
		so.Signature = &sSig
		if tx, err := NewUnsignedExchangeV2(*bo, *so, tc.price, tc.amount, tc.buyMatcherFee, tc.sellMatcherFee, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(mpk, sig, b))
			}
		}
	}
}

func TestExchangeV2BinaryRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seedA)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk := crypto.GenerateKeyPair(seedB)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo1, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	bo1.Sign(sk)
	so1, _ := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	so1.Sign(sk)
	bo2, _ := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	bo2.Sign(sk)
	so2, _ := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	so2.Sign(sk)
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{*bo1, *so1, 123, 456, 789, 987, 654},
		{*bo2, *so2, 987654321, 544321, 9876, 8765, 13245},
		{*bo1, *so2, 123, 456, 789, 987, 654},
		{*bo2, *so1, 987654321, 544321, 9876, 8765, 13245},
	}
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedExchangeV2(tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx ExchangeV2
				if _, err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
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
				var atx ExchangeV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.Equal(t, mpk, atx.SenderPK)
					assert.Equal(t, tc.buy, atx.BuyOrder)
					assert.Equal(t, tc.sell, atx.SellOrder)
					assert.Equal(t, tc.price, atx.Price)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.buyFee, atx.BuyMatcherFee)
					assert.Equal(t, tc.sellFee, atx.SellMatcherFee)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestExchangeV2ToJSON(t *testing.T) {
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
		bo, _ := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, ts, exp, tc.fee)
		bo.Sign(sk)
		boj, _ := json.Marshal(bo)
		so, _ := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, ts, exp, tc.fee)
		so.Sign(sk)
		soj, _ := json.Marshal(so)
		if tx, err := NewUnsignedExchangeV2(*bo, *so, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":7,\"version\":2,\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
					base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(msk); assert.NoError(t, err) {
					if j, err := json.Marshal(tx); assert.NoError(t, err) {
						ej := fmt.Sprintf("{\"type\":7,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
						assert.Equal(t, ej, string(j))
					}
				}
			}
		}
	}
}

func TestLeaseV1Validations(t *testing.T) {
	tests := []struct {
		address string
		amount  uint64
		fee     uint64
		err     string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, 100000, "failed to create LeaseV1 transaction: amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 100000, 0, "failed to create LeaseV1 transaction: fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 100000, 100000, "failed to create LeaseV1 transaction: failed to create new unsigned LeaseV1 transaction: invalid Address checksum"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		rcp, _ := NewAddressFromString(tc.address)
		_, err := NewUnsignedLeaseV1(spk, rcp, tc.amount, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		recipient string
		amount    uint64
		fee       uint64
		timestamp uint64
	}{
		{"fv36AUvNhn4vNRdvA1jfkUmEu25HtoG8vo3bQTgAFQx", "4W28LUXxhF6QmH8rhSFm8v14U1RzU3bnmV6DdbrhNhvqjeMHRtqtY5Hsy9enAXNxoKzrX1L2wSBNWkzjJ4WLASaG", "58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", "3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 111500000000, 100000, 1537728236926},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		rcp, _ := NewAddressFromString(tc.recipient)
		if tx, err := NewUnsignedLeaseV1(spk, rcp, tc.amount, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestLeaseV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAddressFromString(tc.recipient)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseV1(pk, a, tc.amount, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx LeaseV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, *tx.Recipient.Address, *atx.Recipient.Address)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx LeaseV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, a, *atx.Recipient.Address)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestLeaseV1ToJSON(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAddressFromString(tc.recipient)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseV1(pk, a, tc.amount, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":8,\"version\":1,\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":8,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestLeaseV2Validations(t *testing.T) {
	tests := []struct {
		address string
		amount  uint64
		fee     uint64
		err     string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, 100000, "failed to create LeaseV2 transaction: amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 100000, 0, "failed to create LeaseV2 transaction: fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 100000, 100000, "failed to create LeaseV2 transaction: failed to create new unsigned LeaseV1 transaction: invalid Address checksum"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		rcp, _ := NewAddressFromString(tc.address)
		_, err := NewUnsignedLeaseV2(spk, rcp, tc.amount, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseV2FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		recipient string
		amount    uint64
		fee       uint64
		timestamp uint64
	}{
		{"rpQk8UkmB3cKZaS4NsGovw2Rg2HfuoW1FYXx5cmudSs", "4uU5oNDu9zTE4BUdUEtTGcW5cuaLpfFPjV8y1uiCRAkNYaumnBSZVN1YZaLZshQrtztVn4dvYztRa9FPsL879gi6", "8CrVEbYrcrwB8BujfmtFwqEzCr2FKmw5SHC9vL49Jrjf", "3P9DEDP5VbyXQyKtXDUt2crRPn5B7gs6ujc", 50000000000, 100000, 1541665254893},
		{"3mMU3zgPdQY56u5oqjqESZNyVuh7RUzPBRc8jtuRvH5m", "JzJZ9HDFPwL4deFR48jYxohQk6oeouU1tPmcHEqe77QW3P2S3RLY9f12Tq3KFcNf4MWD8fqTutsXh4S8MWzvvhC", "5gzRtg5M5cbF2QzreRpwJt3TmFRHCBmtWdJDYM1g7ks4", "3P3PfgFKpfisSW6RCsbmgWXtwUH8fHAESw4", 11158462347, 100000, 1541681562021},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		rcp, _ := NewAddressFromString(tc.recipient)
		if tx, err := NewUnsignedLeaseV2(spk, rcp, tc.amount, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestLeaseV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAddressFromString(tc.recipient)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseV2(pk, a, tc.amount, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx LeaseV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, *tx.Recipient.Address, *atx.Recipient.Address)
					assert.Equal(t, tx.Amount, atx.Amount)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx LeaseV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, a, *atx.Recipient.Address)
					assert.Equal(t, tc.amount, atx.Amount)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestLeaseV2ToJSON(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAddressFromString(tc.recipient)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseV2(pk, a, tc.amount, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":8,\"version\":2,\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":8,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestLeaseCancelV1Validations(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
		err   string
	}{
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", 0, "failed to create LeaseCancelV1 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		_, err := NewUnsignedLeaseCancelV1(spk, l, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseCancelV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		lease     string
		fee       uint64
		timestamp uint64
	}{
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", "5cEX4Ljm6qY2ZL83uNiTmmbFcEuvm9SFsshYozbwUNQNzLGKHDqYWjiJUoBfLMMGPJcVBsi7YV7DRYKSX4h6EvGT", "6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", "EB3HTJyQb95mbURq7RUC4WEaJGTNgopFvHgH9XkmuMt6", 100000, 1537789773418},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		if tx, err := NewUnsignedLeaseCancelV1(spk, l, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestLeaseCancelV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseCancelV1(pk, l, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx LeaseCancelV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.LeaseID, atx.LeaseID)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx LeaseCancelV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, l, atx.LeaseID)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestLeaseCancelV1ToJSON(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseCancelV1(pk, l, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":9,\"version\":1,\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.lease, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":9,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.lease, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestLeaseCancelV2Validations(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
		err   string
	}{
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", 0, "failed to create LeaseCancelV2 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		_, err := NewUnsignedLeaseCancelV2('T', spk, l, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseCancelV2FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		lease     string
		fee       uint64
		timestamp uint64
	}{
		{"B41okNfzZkFCc2djmbGe9EGoXnRqioXCQxUPJtZm1F74", "FLnMw7hYAaWihBVVjMzcfZWgWPTcsUNWf5bLmsihsLuXksvFtyEkNCEpfCKb7LU4PPxMH1jXVGUvLxMrFS4Pjdy", "5NUzjmfpEiXbD53Y66HpuqTis5VdcGfXj7W4jZxShrZX", "HUhpiDum9NxPwrgLwEowBCdDV7kydJ8sjNqhRSmfZVhe", 100000, 1541730662836},
		{"GxzVYRGDT8XDvBug7f73xhkS6Grz91FDY2DHHqHTdUZL", "4Yjdwyq6unhyCsvt56wE6cQvg4Akauvi7YaQbpCe66i67ggFfgLydWXSfodhYSS3swop1Nn6PP7v3JtWfhSAJU2j", "9zeGFec5Ybiv1xUk8MVC16he67eRD6G41DXkEhReLft7", "HvPgHtKPsDhmGuT9sGM65UDSdetnwQzt4g65ffRCtFiq", 100000, 1541710975166},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		if tx, err := NewUnsignedLeaseCancelV2('W', spk, l, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestLeaseCancelV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseCancelV2('T', pk, l, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx LeaseCancelV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.LeaseID, atx.LeaseID)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx LeaseCancelV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, l, atx.LeaseID)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestLeaseCancelV2ToJSON(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedLeaseCancelV2('T', pk, l, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":9,\"version\":2,\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.lease, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":9,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.lease, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestCreateAliasV1Validations(t *testing.T) {
	tests := []struct {
		alias string
		fee   uint64
		err   string
	}{
		{"something", 0, "failed to create CreateAliasV1 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := NewAlias('W', tc.alias)
		_, err := NewUnsignedCreateAliasV1(spk, *a, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestCreateAliasV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		scheme    byte
		alias     string
		fee       uint64
		timestamp uint64
	}{
		{"6e5rbqXt5UVYFqaQnuGeLjYSFwgmddmEAkYZWdBjPgAF", "3XfjLrk8HZt1mhXnAAaBLEcpbUb4xvsEEgf4AzRqn1bZv3uQ88LisjQn2NzApDYGDvmm1VV4gJxifREjyqDKxRLc", "BEwr6WzmzWT2DRTsmojipT6RoqPkZx64cbfQu4fjuEne", 'W', "stonescissors", 100000, 1537786658492},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		a, _ := NewAlias(tc.scheme, tc.alias)
		if tx, err := NewUnsignedCreateAliasV1(spk, *a, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := tx.id(); assert.NoError(t, err) {
					assert.Equal(t, id, *h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestCreateAliasV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewAlias(tc.scheme, tc.alias)
		if tx, err := NewUnsignedCreateAliasV1(pk, *a, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx CreateAliasV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.Alias, atx.Alias)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx CreateAliasV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.Equal(t, tc.scheme, atx.Alias.Scheme)
					assert.Equal(t, tc.alias, atx.Alias.Alias)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestCreateAliasV1ToJSON(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "alice", 1234567890},
		{'T', "peter", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAlias(tc.scheme, tc.alias)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedCreateAliasV1(pk, *a, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":10,\"version\":1,\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), a.String(), tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":10,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), a.String(), tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestCreateAliasV2Validations(t *testing.T) {
	tests := []struct {
		alias string
		fee   uint64
		err   string
	}{
		{"something", 0, "failed to create CreateAliasV1 transaction: fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := NewAlias('W', tc.alias)
		_, err := NewUnsignedCreateAliasV2(spk, *a, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestCreateAliasV2FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		scheme    byte
		alias     string
		fee       uint64
		timestamp uint64
	}{
		{"7F6CNQTH1CfjKtVV47RbYi6eDctyPjCNNveBP8R3xi63", "dei21qjMJtHtuvZcUcHTV5K4dxVi94SYbY13tuwPaCEkUJeduNmBBnB3MQmkDW43Tm68JtMW58eEBLMUDNQhkxy", "8FYM87cwuFwgdncwAp9qE5FYR4HryJQmyF1g6QhqbiLn", 'W', "bits_cop", 100000, 1541740849873},
		{"B3f8VFh6T2NGT26U7rHk2grAxn5zi9iLkg4V9uxG6C8q", "56FXJ8VtXjJVv1T9QcJHUcBjaQRyxPmnAeDR9JzgSddfkx7rC6barZFSXGDxTv667ehUFxRWuhcEnd3PohkTAoYU", "9SXwLHgyxpBhU5mHc21QWe3C4cjbEbLMAZDYNyjHzGCK", 'W', "pigeon-test", 100000, 1541674542064},
		{"91MhZvyJGnhZy9pfEMUGNKj9j7KUokeJoQQwsyqknDNG", "5J6oXXDQAEWWtRtGVfzWnb42W5mHMCB2cHNg1Zn17jgCLwkU8nshy9hhrjYf86CLF4YnWtsu1uZnpjfc2NkwSKtH", "5QmYAjGXqhym59qxNjSUYswWRvT5v321CXPZsfwnFeiw", 'W', "sexx", 100000, 1541711270800},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		a, _ := NewAlias(tc.scheme, tc.alias)
		if tx, err := NewUnsignedCreateAliasV2(spk, *a, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := tx.id(); assert.NoError(t, err) {
					assert.Equal(t, id, *h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestCreateAliasV2BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewAlias(tc.scheme, tc.alias)
		if tx, err := NewUnsignedCreateAliasV2(pk, *a, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx CreateAliasV2
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.Alias, atx.Alias)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx CreateAliasV2
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.ElementsMatch(t, *tx.ID, *atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
					assert.ElementsMatch(t, pk, atx.SenderPK)
					assert.Equal(t, tc.scheme, atx.Alias.Scheme)
					assert.Equal(t, tc.alias, atx.Alias.Alias)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestCreateAliasV2ToJSON(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "alice", 1234567890},
		{'T', "peter", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		a, _ := NewAlias(tc.scheme, tc.alias)
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedCreateAliasV2(pk, *a, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":10,\"version\":2,\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), a.String(), tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":10,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), a.String(), tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestMassTransferV1Validations(t *testing.T) {
	addr, _ := NewAddressFromString("3PB1Y84BGdEXE4HKaExyJ5cHP36nEw8ovaE")
	tests := []struct {
		asset      string
		transfers  []MassTransferEntry
		fee        uint64
		attachment string
		err        string
	}{
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 100}}, 0, "", "fee should be positive"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{}, 10, "", "empty transfers"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 0}, {NewRecipientFromAddress(addr), 20}}, 20, "", "at least one of the transfers has non-positive amount"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 10}, {NewRecipientFromAddress(addr), 20}}, 30, strings.Repeat("blah-blah", 30), "attachment too long"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := NewOptionalAssetFromString(tc.asset)
		_, err := NewUnsignedMassTransferV1(spk, *a, tc.transfers, tc.fee, 0, tc.attachment)
		assert.EqualError(t, err, tc.err)
	}
}

func TestMassTransferV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk         string
		sig        string
		id         string
		asset      string
		addresses  []string
		amounts    []uint64
		fee        uint64
		timestamp  uint64
		attachment string
	}{
		{"CZtGUoC8hcE4xYsDNGNfzzbhn2Kg67AJdQ9LuMmyYkvr", "4xDmcaiEm7CcsHu6TLaWXEpN5G1sy5Bf4hTHEZXkwRZYZb1wcDCb1c2oN9pbco2g2oLjY2q9bLGwvMn2KWH3tnDg", "HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", "34mkLgLeX6XDsqcafaNWNraQrLUiyGN8vP6TPLuyNKMs", []string{"3PB1Y84BGdEXE4HKaExyJ5cHP36nEw8ovaE", "3PCDiutJwRATXKhFHmadqVVVVA8WS81Sdgn", "3PJcqT9Avo2nf5fQG5FsSMSMyP9ZZhzExAk"}, []uint64{2, 2, 2}, 300000, 1537873402548, ""},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		a, _ := NewOptionalAssetFromString(tc.asset)
		transfers := make([]MassTransferEntry, len(tc.addresses))
		for i, as := range tc.addresses {
			addr, _ := NewAddressFromString(as)
			amount := tc.amounts[i]
			transfers[i] = MassTransferEntry{NewRecipientFromAddress(addr), amount}
		}
		if tx, err := NewUnsignedMassTransferV1(spk, *a, transfers, tc.fee, tc.timestamp, tc.attachment); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestMassTransferV1BinaryRoundTrip(t *testing.T) {
	addr, _ := NewAddressFromString("3PB1Y84BGdEXE4HKaExyJ5cHP36nEw8ovaE")
	tests := []struct {
		asset      string
		transfers  []MassTransferEntry
		fee        uint64
		attachment string
	}{
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 9876543210}}, 1234567890, "this is the attachment"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 12345}, {NewRecipientFromAddress(addr), 67890}}, 987654321, ""},
		{"WAVES", []MassTransferEntry{{NewRecipientFromAddress(addr), 12345}, {NewRecipientFromAddress(addr), 67890}}, 987654321, ""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		if tx, err := NewUnsignedMassTransferV1(pk, *a, tc.transfers, tc.fee, ts, tc.attachment); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx MassTransferV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.Asset, atx.Asset)
					assert.ElementsMatch(t, tx.Transfers, atx.Transfers)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
					assert.Equal(t, tx.Attachment, atx.Attachment)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx MassTransferV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.Equal(t, len(tx.Proofs.Proofs), len(atx.Proofs.Proofs))
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, tc.transfers, atx.Transfers)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
					assert.Equal(t, tc.attachment, atx.Attachment.String())
				}
			}
		}
	}
}

func TestMassTransferV1ToJSON(t *testing.T) {
	addr, _ := NewAddressFromString("3PB1Y84BGdEXE4HKaExyJ5cHP36nEw8ovaE")
	tests := []struct {
		asset              string
		expectedAsset      string
		transfers          []MassTransferEntry
		fee                uint64
		attachment         string
		expectedAttachment string
	}{
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", "\"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK\"", []MassTransferEntry{{NewRecipientFromAddress(addr), 9876543210}}, 1234567890, "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", "\"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK\"", []MassTransferEntry{{NewRecipientFromAddress(addr), 12345}, {NewRecipientFromAddress(addr), 67890}}, 987654321, "", ""},
		{"", "null", []MassTransferEntry{{NewRecipientFromAddress(addr), 12345}, {NewRecipientFromAddress(addr), 67890}}, 987654321, "", ""},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		if tx, err := NewUnsignedMassTransferV1(pk, *a, tc.transfers, tc.fee, ts, tc.attachment); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				var sb strings.Builder
				for i, t := range tc.transfers {
					if i != 0 {
						sb.WriteRune(',')
					}
					sb.WriteRune('{')
					sb.WriteString("\"recipient\":")
					sb.WriteRune('"')
					sb.WriteString(t.Recipient.String())
					sb.WriteRune('"')
					sb.WriteRune(',')
					sb.WriteString("\"amount\":")
					sb.WriteString(strconv.Itoa(int(t.Amount)))
					sb.WriteRune('}')
				}
				ej := fmt.Sprintf("{\"type\":11,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":%s,\"transfers\":[%s],\"timestamp\":%d,\"fee\":%d%s}", base58.Encode(pk[:]), tc.expectedAsset, sb.String(), ts, tc.fee, tc.expectedAttachment)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":11,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":%s,\"transfers\":[%s],\"timestamp\":%d,\"fee\":%d%s}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.expectedAsset, sb.String(), ts, tc.fee, tc.expectedAttachment)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestDataV1Validations(t *testing.T) {
	tests := []struct {
		fee uint64
		err string
	}{
		{0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		_, err := NewUnsignedData(spk, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestDataV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		keys      []string
		values    []string
		fee       uint64
		timestamp uint64
	}{
		{"BDKjPZTcVizRirHhd6u1VJJuUzQUkMctXU9cBNRRxs8k", "4JMP6WwpP78EVYZzG9CKQKDUTPUdvMCYGKVNn4G3VdHmW5mZKNXbvHvuvA8Nj6p39k8htY9VkM6uSf5ombFzETJq", "B7WAhQEM95LvpnKSxNVCCv1WrAzjtAVcKX9NqeCPLK46", []string{"pseudo_random_data", "based_on_height", "timestamp"}, []string{"86CuZqazFdH8cepfkdTv1Qo84khWcVeboRnqdzgEnjFA", "1176855", "Mon Sep 17 18:31:49 EEST 2018"}, 100000, 1537198309819},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		if tx, err := NewUnsignedData(spk, tc.fee, tc.timestamp); assert.NoError(t, err) {
			for i, k := range tc.keys {
				e := StringDataEntry{k, tc.values[i]}
				tx.AppendEntry(e)
			}
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestDataV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		keys   []string
		types  []byte
		values []string
		fee    uint64
	}{
		{[]string{"int-val", "bool-val", "bin-val", "string-val"}, []byte{0, 1, 2, 3}, []string{"1234567890", "true", "4JMP6WwpP78EVYZzG9CKQKDUTPUdvMCYGKVNn4G3VdHmW5mZKNXbvHvuvA8Nj6p39k8htY9VkM6uSf5ombFzETJq", "some string"}, 1234567890},
		{[]string{"int-val", "bool-val", "bin-val", "string-val"}, []byte{0, 1, 2, 3}, []string{"987654321", "false", "B7WAhQEM95LvpnKSxNVCCv1WrAzjtAVcKX9NqeCPLK46", ""}, 1234567890},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedData(pk, tc.fee, ts); assert.NoError(t, err) {
			for i, k := range tc.keys {
				var e DataEntry
				switch ValueType(tc.types[i]) {
				case Integer:
					v, _ := strconv.Atoi(tc.values[i])
					e = IntegerDataEntry{k, int64(v)}
				case Boolean:
					v, _ := strconv.ParseBool(tc.values[i])
					e = BooleanDataEntry{k, v}
				case Binary:
					v, _ := base58.Decode(tc.values[i])
					e = BinaryDataEntry{k, v}
				case String:
					e = StringDataEntry{k, tc.values[i]}
				}
				err := tx.AppendEntry(e)
				assert.NoError(t, err)
			}
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx DataV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.Entries, atx.Entries)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx DataV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.Equal(t, len(tx.Proofs.Proofs), len(atx.Proofs.Proofs))
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, pk, atx.SenderPK)
					assert.ElementsMatch(t, tx.Entries, atx.Entries)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestDataV1ToJSON(t *testing.T) {
	tests := []struct {
		keys   []string
		types  []byte
		values []string
		fee    uint64
	}{
		{[]string{"int-val", "bool-val", "bin-val", "string-val"}, []byte{0, 1, 2, 3}, []string{"1234567890", "true", "4JMP6WwpP78EVYZzG9CKQKDUTPUdvMCYGKVNn4G3VdHmW5mZKNXbvHvuvA8Nj6p39k8htY9VkM6uSf5ombFzETJq", "some string"}, 1234567890},
		{[]string{"int-val", "bool-val", "bin-val", "string-val"}, []byte{0, 1, 2, 3}, []string{"987654321", "false", "B7WAhQEM95LvpnKSxNVCCv1WrAzjtAVcKX9NqeCPLK46", ""}, 1234567890},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		if tx, err := NewUnsignedData(pk, tc.fee, ts); assert.NoError(t, err) {
			var sb strings.Builder
			for i, k := range tc.keys {
				if i != 0 {
					sb.WriteRune(',')
				}
				sb.WriteRune('{')
				sb.WriteString("\"key\":")
				sb.WriteRune('"')
				sb.WriteString(k)
				sb.WriteRune('"')
				sb.WriteString(",\"type\":")
				var e DataEntry
				switch ValueType(tc.types[i]) {
				case Integer:
					v, _ := strconv.Atoi(tc.values[i])
					e = IntegerDataEntry{k, int64(v)}
					sb.WriteRune('"')
					sb.WriteString("integer")
					sb.WriteRune('"')
					sb.WriteString(",\"value\":")
					sb.WriteString(tc.values[i])
				case Boolean:
					v, _ := strconv.ParseBool(tc.values[i])
					e = BooleanDataEntry{k, v}
					sb.WriteRune('"')
					sb.WriteString("boolean")
					sb.WriteRune('"')
					sb.WriteString(",\"value\":")
					sb.WriteString(tc.values[i])
				case Binary:
					v, _ := base58.Decode(tc.values[i])
					e = BinaryDataEntry{k, v}
					sb.WriteRune('"')
					sb.WriteString("binary")
					sb.WriteRune('"')
					sb.WriteString(",\"value\":")
					sb.WriteRune('"')
					sb.WriteString(base64.StdEncoding.EncodeToString(v))
					sb.WriteRune('"')
				case String:
					e = StringDataEntry{k, tc.values[i]}
					sb.WriteRune('"')
					sb.WriteString("string")
					sb.WriteRune('"')
					sb.WriteString(",\"value\":")
					sb.WriteRune('"')
					sb.WriteString(tc.values[i])
					sb.WriteRune('"')
				}
				sb.WriteRune('}')
				err := tx.AppendEntry(e)
				assert.NoError(t, err)
			}
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":12,\"version\":1,\"senderPublicKey\":\"%s\",\"data\":[%s],\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), sb.String(), tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":12,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"data\":[%s],\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), sb.String(), tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestSetScriptV1Validations(t *testing.T) {
	tests := []struct {
		script string
		fee    uint64
		err    string
	}{
		{"something", 0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		s, _ := base58.Decode(tc.script)
		_, err := NewUnsignedSetScriptV1('W', spk, s, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestSetScriptV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		scheme    byte
		script    string
		fee       uint64
		timestamp uint64
	}{
		{"3LZmDK7vuSBsDmFLxJ4qihZynUz8JF9e88dNu5fsus5p", "V45jPG1nuEnwaYb9jTKQCJpRskJQvtkBcnZ45WjZUbVdNTi1KijVikJkDfMNcEdSBF8oGDYZiWpVTdLSn76mV57", "8Nwjd2tcQWff3S9WAhBa7vLRNpNnigWqrTbahvyfMVrU", 'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 2082496, 1537973512182},
		{"2M25DqL2W4rGFLCFadgATboS8EPqyWAN3DjH12AH5Kdr", "2WwvBAosGg1WN8g8f2xXqxXt8rz8Yzgdh1cFEYPrks674ryEtqKXMT8YmBrTLHGSSNgGaP5y19A1XGcLd7L5UCMb", "6uZcgYxmC33ziqUAKo1uyxhFARoQEWczf6jgM8Ns8jZa", 'W', "", 1400000, 1539693546199},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		if tx, err := NewUnsignedSetScriptV1(tc.scheme, spk, s, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestSetScriptV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		if tx, err := NewUnsignedSetScriptV1(tc.chainID, pk, s, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx SetScriptV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.Equal(t, tx.ChainID, atx.ChainID)
					assert.ElementsMatch(t, tx.Script, atx.Script)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx SetScriptV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, pk, atx.SenderPK)
					assert.Equal(t, tc.chainID, atx.ChainID)
					assert.Equal(t, tc.script, base64.StdEncoding.EncodeToString(atx.Script))
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestSetScriptV1ToJSON(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		if tx, err := NewUnsignedSetScriptV1(tc.chainID, pk, s, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":13,\"version\":1,\"senderPublicKey\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.script, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":13,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.script, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}

func TestSponsorshipV1Validations(t *testing.T) {
	tests := []struct {
		fee uint64
		err string
	}{
		{0, "fee should be positive"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := crypto.NewDigestFromBase58("8Nwjd2tcQWff3S9WAhBa7vLRNpNnigWqrTbahvyfMVrU")
		_, err := NewUnsignedSponsorshipV1(spk, a, 0, tc.fee, 0)
		assert.EqualError(t, err, tc.err)
	}
}

func TestSponsorshipV1FromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		asset     string
		assetFee  uint64
		fee       uint64
		timestamp uint64
	}{
		{"9hdJNctMBwRf4aWS7GMwZrZAi4PndLqG8GRjL8tZiwTm", "4WucVTUE6NusuZL6SNFtoYfGbZVab8MQcwk4xtrnkQgJFxQrQNVXRCgXXgMu7o5C6EC6iNHfGztQNKnmp3cphEEu", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 10000000, 100000000, 1537922072905},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedSponsorshipV1(spk, a, tc.assetFee, tc.fee, tc.timestamp); assert.NoError(t, err) {
			if b, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				if h, err := crypto.FastHash(b); assert.NoError(t, err) {
					assert.Equal(t, id, h)
				}
				assert.True(t, crypto.Verify(spk, sig, b))
			}
		}
	}
}

func TestSponsorshipV1BinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedSponsorshipV1(pk, a, tc.assetFee, tc.fee, ts); assert.NoError(t, err) {
			if bb, err := tx.bodyMarshalBinary(); assert.NoError(t, err) {
				var atx SponsorshipV1
				if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, atx.Type)
					assert.Equal(t, tx.Version, atx.Version)
					assert.Equal(t, tx.SenderPK, atx.SenderPK)
					assert.ElementsMatch(t, tx.AssetID, atx.AssetID)
					assert.Equal(t, tx.MinAssetFee, atx.MinAssetFee)
					assert.Equal(t, tx.Fee, atx.Fee)
					assert.Equal(t, tx.Timestamp, atx.Timestamp)
				}
			}
			if err := tx.Sign(sk); assert.NoError(t, err) {
				if r, err := tx.Verify(pk); assert.NoError(t, err) {
					assert.True(t, r)
				}
			}
			if b, err := tx.MarshalBinary(); assert.NoError(t, err) {
				var atx SponsorshipV1
				if err := atx.UnmarshalBinary(b); assert.NoError(t, err) {
					assert.Equal(t, tx.ID, atx.ID)
					assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
					assert.Equal(t, pk, atx.SenderPK)
					assert.Equal(t, tc.asset, atx.AssetID.String())
					assert.Equal(t, tc.assetFee, atx.MinAssetFee)
					assert.Equal(t, tc.fee, atx.Fee)
					assert.Equal(t, ts, atx.Timestamp)
				}
			}
		}
	}
}

func TestSponsorshipV1ToJSON(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk := crypto.GenerateKeyPair(seed)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		if tx, err := NewUnsignedSponsorshipV1(pk, a, tc.assetFee, tc.fee, ts); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":14,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"minSponsoredAssetFee\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.asset, tc.assetFee, tc.fee, ts)
				assert.Equal(t, ej, string(j))
				if err := tx.Sign(sk); assert.NoError(t, err) {
					if sj, err := json.Marshal(tx); assert.NoError(t, err) {
						esj := fmt.Sprintf("{\"type\":14,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"minSponsoredAssetFee\":%d,\"fee\":%d,\"timestamp\":%d}",
							base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.assetFee, tc.fee, ts)
						assert.Equal(t, esj, string(sj))
					}
				}
			}
		}
	}
}
