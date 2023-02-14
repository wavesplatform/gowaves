package proto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/libs/serializer"
	protobuf "google.golang.org/protobuf/proto"
)

func TestGuessTransaction_Genesis(t *testing.T) {
	genesisJson := `    {
      "type": 1,
      "id": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "fee": 0,
      "timestamp": 1465742577614,
      "signature": "2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8",
      "recipient": "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ",
      "amount": 9999999500000000
    }`

	buf := bytes.NewBufferString(genesisJson)
	genesis := &Genesis{}
	rs, err := GuessTransactionType(&TransactionTypeVersion{Type: TransactionType(1), Version: 1})
	require.Nil(t, err)
	err = json.NewDecoder(buf).Decode(genesis)
	require.Nil(t, err)
	require.IsType(t, &Genesis{}, rs)
	assert.Equal(t, uint64(9999999500000000), genesis.Amount)
}

func TestGenesisBinarySize(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _, err := crypto.GenerateKeyPair(s)
	require.NoError(t, err)

	tests := []struct {
		timestamp uint64
		recipient string
		amount    uint64
	}{
		{1465742577614, "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9999999500000000},
		{1465742577614, "3P8JdJGYc7vaLu4UXUZc1iRLdzrkGtdCyJM", 100000000},
		{1465742577614, "3PAGPDPqnGkyhcihyjMHe9v36Y4hkAh9yDy", 100000000},
		{1465742577614, "3P9o3ZYwtHkaU1KxsKkFjJqJKS3dLHLC9oF", 100000000},
		{1465742577614, "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3", 100000000},
		{1465742577614, "3PBWXDFUc86N2EQxKJmW8eFco65xTyMZx6J", 100000000},
	}
	for _, tc := range tests {
		if rcp, err := NewAddressFromString(tc.recipient); assert.NoError(t, err) {
			tx := NewUnsignedGenesis(rcp, tc.amount, tc.timestamp)
			err = tx.Sign(MainNetScheme, sk)
			assert.Nil(t, err)
			_, err := tx.Validate(MainNetScheme)
			assert.Nil(t, err)
			txBytes, err := tx.MarshalBinary(MainNetScheme)
			assert.Nil(t, err)
			assert.Equal(t, len(txBytes), tx.BinarySize())
		}
	}
}

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
			tx := NewUnsignedGenesis(rcp, tc.amount, tc.timestamp)
			_, err := tx.Validate(MainNetScheme)
			assert.Nil(t, err)
			if err := tx.GenerateSigID(MainNetScheme); assert.NoError(t, err) {
				assert.Equal(t, id, tx.ID[:])
				assert.Equal(t, tc.amount, tx.Amount)
				assert.Equal(t, tc.recipient, tx.Recipient.String())
				assert.Equal(t, tc.timestamp, tx.Timestamp)
				b, err := tx.MarshalBinary(MainNetScheme)
				assert.NoError(t, err)
				var at Genesis
				err = at.UnmarshalBinary(b, MainNetScheme)
				assert.NoError(t, err)
				err = at.GenerateID(MainNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, at)
			}
		}
	}
}

func TestGenesisProtobufRoundTrip(t *testing.T) {
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
		if rcp, err := NewAddressFromString(tc.recipient); assert.NoError(t, err) {
			tx := NewUnsignedGenesis(rcp, tc.amount, tc.timestamp)
			_, err := tx.Validate(MainNetScheme)
			assert.Nil(t, err)
			err = tx.GenerateID(MainNetScheme)
			assert.Nil(t, err)
			b, err := tx.MarshalToProtobuf(MainNetScheme)
			assert.Nil(t, err)
			var at Genesis
			err = at.UnmarshalFromProtobuf(b)
			assert.Nil(t, err)
			err = at.GenerateID(MainNetScheme)
			assert.NoError(t, err)
			assert.Equal(t, *tx, at)
			if err := tx.GenerateSigID(MainNetScheme); assert.NoError(t, err) {
				b, err := tx.MarshalSignedToProtobuf(MainNetScheme)
				assert.NoError(t, err)
				var at2 Genesis
				err = at2.UnmarshalSignedFromProtobuf(b)
				assert.NoError(t, err)
				err = at2.GenerateID(MainNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, at2)
			}
		}
	}
}

func TestGenesisValidations(t *testing.T) {
	tests := []struct {
		scheme  Scheme
		address string
		amount  uint64
		err     string
	}{
		{MainNetScheme, "3PLrCnhKyX5iFbGDxbqqMvea5VAqxMcinPW", 0, "amount should be positive"},
		{MainNetScheme, "3PLrCnhKyX5iFbGDxbqqMvea5VAqxMcinPW", math.MaxInt64, ""},
		{MainNetScheme, "3PLrCnhKyX5iFbGDxbqqMvea5VAqxMcinPW", math.MaxInt64 + 1, "amount is too big"},
		{TestNetScheme, "3PLrCnhKyX5iFbGDxbqqMvea5VAqxMcinPW", 1000, "invalid recipient address '3PLrCnhKyX5iFbGDxbqqMvea5VAqxMcinPW': invalid scheme 'W', expected 'T'"},
		{MainNetScheme, "3N8qPqNS7PYKd8xohXaqQUGkibf58EACN7g", 1000, "invalid recipient address '3N8qPqNS7PYKd8xohXaqQUGkibf58EACN7g': invalid scheme 'T', expected 'W'"},
		{MainNetScheme, "3PLrCnhKyX5iFbE9t9Fhn7x2Jd2egp68GKE", 1000, "invalid recipient address '3PLrCnhKyX5iFbE9t9Fhn7x2Jd2egp68GKE': invalid WavesAddress checksum"},
		{MainNetScheme, "2JCiMDURsukbNXQddDVHrVefRRXo4kKjLNAp", 1000, "invalid recipient address '2JCiMDURsukbNXQddDVHrVefRRXo4kKjLNAp': unsupported address version 42"},
	}
	for _, tc := range tests {
		addr, err := addressFromString(tc.address)
		require.NoError(t, err)
		tx := NewUnsignedGenesis(addr, tc.amount, 0)
		_, err = tx.Validate(tc.scheme)
		if tc.err != "" {
			assert.EqualError(t, err, tc.err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestGenesisToJSON(t *testing.T) {
	const addr = "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ"
	if rcp, err := NewAddressFromString(addr); assert.NoError(t, err) {
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedGenesis(rcp, 1000, ts)
		err := tx.GenerateSigID(TestNetScheme)
		require.NoError(t, err)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":1,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"timestamp\":%d,\"recipient\":\"%s\",\"amount\":1000}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), ts, tx.Recipient.String())
			assert.Equal(t, ej, string(j))
		}
	}
}

func TestPaymentBinarySize(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(s)
	require.NoError(t, err)
	tests := []struct {
		address string
		amount  uint64
		fee     uint64
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 1, 10},
	}
	for _, tc := range tests {
		addr, err := addressFromString(tc.address)
		require.NoError(t, err)
		tx := NewUnsignedPayment(pk, addr, tc.amount, tc.fee, 1)
		err = tx.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		require.NoError(t, err)
		require.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestPaymentMarshalUnmarshal(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(s)
	require.NoError(t, err)
	tests := []struct {
		address string
		amount  uint64
		fee     uint64
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 1, 10},
	}
	for _, tc := range tests {
		addr, err := addressFromString(tc.address)
		require.NoError(t, err)
		tx := NewUnsignedPayment(pk, addr, tc.amount, tc.fee, 1)
		err = tx.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		var tx1 Payment
		err = tx1.UnmarshalBinary(txBytes, TestNetScheme)
		require.NoError(t, err)
		err = tx1.GenerateID(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, *tx, tx1)
	}
}

func TestPaymentProtobufRoundTrip(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(s)
	require.NoError(t, err)
	tests := []struct {
		address string
		amount  uint64
		fee     uint64
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 1, 10},
	}
	for _, tc := range tests {
		addr, err := addressFromString(tc.address)
		require.NoError(t, err)
		tx := NewUnsignedPayment(pk, addr, tc.amount, tc.fee, 1)
		err = tx.GenerateID(MainNetScheme)
		require.NoError(t, err)
		txBytes, err := tx.MarshalToProtobuf(MainNetScheme)
		assert.NoError(t, err)
		var tx1 Payment
		err = tx1.UnmarshalFromProtobuf(txBytes)
		require.NoError(t, err)
		err = tx1.GenerateID(MainNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, *tx, tx1)
		err = tx.Sign(MainNetScheme, sk)
		require.NoError(t, err)
		txBytes, err = tx.MarshalSignedToProtobuf(MainNetScheme)
		assert.NoError(t, err)
		var tx2 Payment
		err = tx2.UnmarshalSignedFromProtobuf(txBytes)
		require.NoError(t, err)
		err = tx2.GenerateID(MainNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, *tx, tx2)
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
		{"2ZojhAw3r8DhiHD6gRJ2dXNpuErAd4iaoj5NSWpfYrqppxpYkcXBHzSAWTkAGX5d3EeuAUS8rZ4vnxnDSbJU8MkM", 1465754870341, "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3P7NaMWCosRTbVwTfiiU6M6tHpQ6DuNFtYp", 20999990, 1},
		{"5cQLvZVUZqYcC75u5vXydpPoxKeazyiNtKgtz4DSyQboDSyefxcQEihwN9er772DbFDuaBRDLQHbT9CJiezk8sba", 1465825839722, "vAyFRfGG225MjUXo2VXhLfh2F6utsGkG782HuKi5fRp", "3P9v6SjRKUZPZMG1aL2oTznGZHBvNr21EQS", 99999999, 1},
		{"396pxC3YjVMjYQF7S9Xk3ntCjEJz4ip91ckux6ni4qpNEHbkyzqhSeYzyiVaUUM94uc21nGe8qwurGFDdzynrCHZ", 1466531340683, "2DAbbF2XuQPc3ePzKdxncsdMUzjSjEGC4nHx7kA3s1jm", "3PFrwqFZpoTzwKYq8NUALrtALP1oDvixt8z", 49310900000000, 1},
	}
	for _, tc := range tests {
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		spk, _ := crypto.NewPublicKeyFromBase58(tc.spk)
		if rcp, err := NewAddressFromString(tc.recipient); assert.NoError(t, err) {
			tx := NewUnsignedPayment(spk, rcp, tc.amount, tc.fee, tc.timestamp)
			assert.Equal(t, tc.spk, base58.Encode(tx.SenderPK[:]))
			assert.Equal(t, tc.amount, tx.Amount)
			assert.Equal(t, tc.recipient, tx.Recipient.String())
			assert.Equal(t, tc.timestamp, tx.Timestamp)
			assert.Equal(t, tc.fee, tx.Fee)
			b := tx.bodyMarshalBinaryBuffer()
			err = tx.bodyMarshalBinary(b)
			assert.NoError(t, err)
			var at Payment
			err = at.bodyUnmarshalBinary(b)
			assert.NoError(t, err)
			assert.Equal(t, *tx, at)
			tx.Signature = &sig
			tx.ID = &sig
			vr, err := tx.Verify(TestNetScheme, spk)
			require.NoError(t, err)
			assert.True(t, vr)
			b, _ = tx.MarshalBinary(TestNetScheme)
			err = at.UnmarshalBinary(b, TestNetScheme)
			assert.NoError(t, err)
			err = at.GenerateID(TestNetScheme)
			assert.NoError(t, err)
			assert.Equal(t, *tx, at)

			signable, err := tx.BodyMarshalBinary(TestNetScheme)
			require.NoError(t, err)
			require.True(t, crypto.Verify(spk, *tx.Signature, signable))
		}
	}
}

func BenchmarkPayment_MarshalBinary(t *testing.B) {
	t.ReportAllocs()
	tc := struct {
		sig       string
		timestamp uint64
		spk       string
		recipient string
		amount    uint64
		fee       uint64
	}{"2ZojhAw3r8DhiHD6gRJ2dXNpuErAd4iaoj5NSWpfYrqppxpYkcXBHzSAWTkAGX5d3EeuAUS8rZ4vnxnDSbJU8MkM", 1465754870341, "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3P7NaMWCosRTbVwTfiiU6M6tHpQ6DuNFtYp", 20999990, 1}
	var b []byte
	sig, _ := crypto.NewSignatureFromBase58(tc.sig)
	spk, _ := crypto.NewPublicKeyFromBase58(tc.spk)
	rcp, err := NewAddressFromString(tc.recipient)
	require.NoError(t, err)
	tx := NewUnsignedPayment(spk, rcp, tc.amount, tc.fee, tc.timestamp)
	tx.Signature = &sig
	tx.ID = &sig
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		b, err = tx.MarshalBinary(TestNetScheme)
	}
	t.StopTimer()
	if err != nil || len(b) == 0 {
		t.FailNow()
	}
}

func TestPaymentValidations(t *testing.T) {
	tests := []struct {
		spk     string
		address string
		amount  uint64
		fee     uint64
		err     string
	}{
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, 10, "amount should be positive"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 10, 0, "fee should be positive"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 10, 10, "invalid recipient address '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid WavesAddress checksum"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", math.MaxInt64 + 100, 10, "amount is too big"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 10, math.MaxInt64 + 100, "fee is too big"},
		{"AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7", "3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", math.MaxInt64, math.MaxInt64, "sum of amount and fee overflows JVM long"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58(tc.spk)
		require.NoError(t, err)
		addr, err := addressFromString(tc.address)
		require.NoError(t, err)
		tx := NewUnsignedPayment(spk, addr, tc.amount, tc.fee, 0)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestPaymentToJSON(t *testing.T) {
	s, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(s)
	require.NoError(t, err)
	rcp, _ := NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	ts := uint64(time.Now().Unix() * 1000)
	tx := NewUnsignedPayment(pk, rcp, 1000, 10, ts)
	err = tx.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	if j, err := json.Marshal(tx); assert.NoError(t, err) {
		ej := fmt.Sprintf("{\"type\":2,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":1000,\"fee\":10,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(tx.SenderPK[:]), tx.Recipient.String(), ts)
		assert.Equal(t, ej, string(j))
	}
}

func TestIssueWithSigFromMainNet(t *testing.T) {
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
			tx := NewUnsignedIssueWithSig(spk, "WBTC", "Bitcoin Token", 2100000000000000, 8, false, 1480690876160, 100000000)
			if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
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

func TestIssueWithSigValidations(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		quantity uint64
		decimals byte
		fee      uint64
		err      string
	}{
		{"TKN", "This is a valid description for the token", 1000000, 2, 100000, "incorrect number of bytes in the asset's name"},
		{"VERY_LONG_TOKEN_NAME", "This is a valid description for the token", 1000000, 2, 100000, "incorrect number of bytes in the asset's name"},
		{"TOKEN", strings.Repeat("x", 1010), 1000000, 2, 100000, "incorrect number of bytes in the asset's description"},
		{"TOKEN", "This is a valid description for the token", 0, 2, 100000, "quantity should be positive"},
		{"TOKEN", "This is a valid description for the token", math.MaxInt64 + 100, 2, 100000, "quantity is too big"},
		{"TOKEN", "This is a valid description for the token", 100000, 12, 100000, fmt.Sprintf("incorrect decimals, should be no more then %d", MaxDecimals)},
		{"TOKEN", "This is a valid description for the token", 100000, 2, 0, "fee should be positive"},
		{"TOKEN", "This is a valid description for the token", 100000, 2, math.MaxInt64 + 100, "fee is too big"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		if assert.NoError(t, err) {
			tx := NewUnsignedIssueWithSig(spk, tc.name, tc.desc, tc.quantity, tc.decimals, false, 0, tc.fee)
			_, err := tx.Validate(TestNetScheme)
			assert.EqualError(t, err, tc.err)
		}
	}
}

func TestIssueWithSigBinarySize(t *testing.T) {
	const seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	s, err := base58.Decode(seed)
	if assert.NoError(t, err) {
		sk, pk, err := crypto.GenerateKeyPair(s)
		assert.NoError(t, err)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedIssueWithSig(pk, "TOKEN", "description", 1000, 10, false, ts, 100000)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestIssueWithSigSigningRoundTrip(t *testing.T) {
	const seed = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	s, err := base58.Decode(seed)
	if assert.NoError(t, err) {
		sk, pk, err := crypto.GenerateKeyPair(s)
		assert.NoError(t, err)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedIssueWithSig(pk, "TOKEN", "", 1000, 0, false, ts, 100000)
		err = tx.Sign(TestNetScheme, sk)
		if assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
	}
}

func TestIssueWithSigToJSON(t *testing.T) {
	if s, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"); assert.NoError(t, err) {
		sk, pk, err := crypto.GenerateKeyPair(s)
		assert.NoError(t, err)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedIssueWithSig(pk, "TOKEN", "", 1000, 0, false, ts, 100000)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":3,\"version\":1,\"senderPublicKey\":\"%s\",\"name\":\"TOKEN\",\"description\":\"\",\"quantity\":1000,\"decimals\":0,\"reissuable\":false,\"timestamp\":%d,\"fee\":100000}", base58.Encode(pk[:]), ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":3,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"name\":\"TOKEN\",\"description\":\"\",\"quantity\":1000,\"decimals\":0,\"reissuable\":false,\"timestamp\":%d,\"fee\":100000}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestIssueWithSigBinaryRoundTrip(t *testing.T) {
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
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
		tx := NewUnsignedIssueWithSig(pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, tc.ts, tc.fee)
		b, err := tx.BodyMarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		var at IssueWithSig
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
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
				var at IssueWithSig
				if err = at.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
					err = at.GenerateID(TestNetScheme)
					assert.NoError(t, err)
					err = at.GenerateID(TestNetScheme)
					assert.NoError(t, err)
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

func TestIssueWithSigProtobufRoundTrip(t *testing.T) {
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
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
		tx := NewUnsignedIssueWithSig(pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, tc.ts, tc.fee)
		err = tx.GenerateID(TestNetScheme)
		assert.NoError(t, err)
		b, err := tx.MarshalToProtobuf(TestNetScheme)
		assert.NoError(t, err)
		var at IssueWithSig
		if err := at.UnmarshalFromProtobuf(b); assert.NoError(t, err) {
			assert.Equal(t, *tx, at)
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
				var at IssueWithSig
				if err = at.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
					err = at.GenerateID(TestNetScheme)
					assert.NoError(t, err)
					assert.Equal(t, *tx, at)
				}
			}
		}
	}
}

func TestIssueWithProofsValidations(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		quantity uint64
		decimals byte
		fee      uint64
		err      string
	}{
		{"TKN", "This is a valid description for the token", 1000000, 2, 100000, "incorrect number of bytes in the asset's name"},
		{"VERY_LONG_TOKEN_NAME", "This is a valid description for the token", 1000000, 2, 100000, "incorrect number of bytes in the asset's name"},
		{"TOKEN", strings.Repeat("x", 1010), 1000000, 2, 100000, "incorrect number of bytes in the asset's description"},
		{"TOKEN", "This is a valid description for the token", 0, 2, 100000, "quantity should be positive"},
		{"TOKEN", "This is a valid description for the token", math.MaxInt64 + 1, 2, 100000, "quantity is too big"},
		{"TOKEN", "This is a valid description for the token", 100000, 12, 100000, fmt.Sprintf("incorrect decimals, should be no more then %d", MaxDecimals)},
		{"TOKEN", "This is a valid description for the token", 100000, 2, 0, "fee should be positive"},
		{"TOKEN", "This is a valid description for the token", 100000, 2, math.MaxInt64 + 1, "fee is too big"},
		//TODO: add tests on script validation
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedIssueWithProofs(2, spk, tc.name, tc.desc, tc.quantity, tc.decimals, false, []byte{}, 0, tc.fee)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestIssueWithProofsFromMainNet(t *testing.T) {
	const scheme = MainNetScheme

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
		tx := NewUnsignedIssueWithProofs(2, spk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, []byte{}, tc.timestamp, tc.fee)
		if b, err := tx.BodyMarshalBinary(scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestIssueWithProofsBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedIssueWithProofs(2, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee)
		err := tx.Sign(tc.chain, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.chain)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestIssueWithProofsBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedIssueWithProofs(2, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee)
		if bb, err := tx.BodyMarshalBinary(tc.chain); assert.NoError(t, err) {
			var atx IssueWithProofs
			if err := atx.bodyUnmarshalBinary(bb, tc.chain); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
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
		if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chain, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.chain); assert.NoError(t, err) {
			var atx IssueWithProofs
			if err := atx.UnmarshalBinary(b, tc.chain); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
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

func TestIssueWithProofsProtobufRoundTrip(t *testing.T) {
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
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
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedIssueWithProofs(2, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee)
		err = tx.GenerateID(tc.chain)
		assert.NoError(t, err)
		b, err := tx.MarshalToProtobuf(tc.chain)
		assert.NoError(t, err)
		var atx IssueWithProofs
		if err := atx.UnmarshalFromProtobuf(b); assert.NoError(t, err) {
			assert.Equal(t, *tx, atx)
		}
		if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
			if b, err := tx.MarshalSignedToProtobuf(tc.chain); assert.NoError(t, err) {
				var atx IssueWithProofs
				if err = atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
					err = atx.GenerateID(tc.chain)
					assert.NoError(t, err)
					assert.Equal(t, *tx, atx)
				}
			}
		}
	}
}

func TestIssueWithProofsToJSON(t *testing.T) {
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
		{'T', "TOKEN", "This is a valid description for the token", 12345, 4, true, "base64:AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 100000},
		{'W', "SHMOKEN", "This is a valid description for the token", 100000, 8, false, "base64:", 100000},
		{'X', "POKEN", "This is a valid description for the token", 9876543210, 2, true, "base64:AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 123456},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, err := base64.StdEncoding.DecodeString(tc.script[7:])
		require.NoError(t, err)
		tx := NewUnsignedIssueWithProofs(2, pk, tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, s, ts, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":3,\"version\":2,\"script\":\"%s\",\"senderPublicKey\":\"%s\",\"name\":\"%s\",\"description\":\"%s\",\"quantity\":%d,\"decimals\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}",
				tc.script, base58.Encode(pk[:]), tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, ts, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":3,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"script\":\"%s\",\"senderPublicKey\":\"%s\",\"name\":\"%s\",\"description\":\"%s\",\"quantity\":%d,\"decimals\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), tc.script, base58.Encode(pk[:]), tc.name, tc.desc, tc.quantity, tc.decimals, tc.reissuable, ts, tc.fee)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestTransferWithSigValidations(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
		att       string
		err       string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, 10, "The attachment", "amount should be positive"},
		{"alias:T:nickname", 1000, 0, "The attachment", "fee should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", math.MaxInt64 + 10, 1, "The attachment", "amount is too big"},
		{"alias:T:nickname", 1000, math.MaxInt64 + 100, "The attachment", "fee is too big"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", math.MaxInt64, math.MaxInt64, "The attachment", "sum of amount and fee in the same asset overflows JVM long"},
		{"alias:T:nickname", 1000, 10, strings.Repeat("The attachment", 100), "attachment is too long"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGQQgnWN", 1000, 10, "The attachment", "invalid recipient '3MxW8ZFCQUQDg7xagmGQQcwbQDmNGQQgnWN': invalid WavesAddress checksum"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 1000, 10, "The attachment", "invalid recipient '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid scheme 'W', expected 'T'"},
		{"alias:T:прозвище", 1000, 10, "The attachment", "invalid recipient 'alias:T:прозвище': Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz"},
		{"alias:W:invalid-scheme", 1000, 10, "The attachment", "invalid recipient 'alias:W:invalid-scheme': invalid scheme 'W', expected 'T'"},
		{"alias:W:прозвище", 1000, 10, "The attachment", "invalid recipient 'alias:W:прозвище': invalid scheme 'W', expected 'T'"},
	}
	spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	for _, tc := range tests {
		rcp, err := recipientFromString(tc.recipient)
		require.NoError(t, err)
		a, err := NewOptionalAssetFromString("WAVES")
		require.NoError(t, err)
		att := []byte(tc.att)
		tx := NewUnsignedTransferWithSig(spk, *a, *a, 0, tc.amount, tc.fee, rcp, att)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err, "No expected error '%s'", tc.err)
	}
}

func TestTransferWithSigProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithSig(pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		err = tx.GenerateID(tc.scheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.scheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestTransferWithSigBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithSig(pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		err = tx.Sign(tc.scheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.scheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestTransferWithSigBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := Attachment(tc.attachment)
		tx := NewUnsignedTransferWithSig(pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		if bb, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithSig
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
				assert.Equal(t, tx.Attachment, atx.Attachment)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithSig
			if err := atx.UnmarshalBinary(b, tc.scheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
				assert.ElementsMatch(t, pk, atx.SenderPK)
				assert.Equal(t, tc.expectedAmountAsset, atx.AmountAsset.String())
				assert.Equal(t, tc.expectedFeeAsset, atx.FeeAsset.String())
				assert.Equal(t, tc.amount, atx.Amount)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
				att := Attachment(tc.attachment)
				assert.Equal(t, att, atx.Attachment)
			}
		}
		buf := &bytes.Buffer{}
		_, err = tx.WriteTo(buf)
		require.NoError(t, err)
		var atx TransferWithSig
		if err := atx.UnmarshalBinary(buf.Bytes(), tc.scheme); assert.NoError(t, err) {
			assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
			assert.ElementsMatch(t, pk, atx.SenderPK)
			assert.Equal(t, tc.expectedAmountAsset, atx.AmountAsset.String())
			assert.Equal(t, tc.expectedFeeAsset, atx.FeeAsset.String())
			assert.Equal(t, tc.amount, atx.Amount)
			assert.Equal(t, tc.fee, atx.Fee)
			assert.Equal(t, ts, atx.Timestamp)
			att := Attachment(tc.attachment)
			assert.Equal(t, att, atx.Attachment)
		}
	}
}

func TestTransferWithSigFromMainNet(t *testing.T) {
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
		id, err := crypto.NewDigestFromBase58(tc.id)
		require.NoError(t, err)
		sig, err := crypto.NewSignatureFromBase58(tc.sig)
		require.NoError(t, err)
		pk, err := crypto.NewPublicKeyFromBase58(tc.pk)
		require.NoError(t, err)
		addr, err := NewAddressFromString(tc.rcp)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithSig(pk, *aa, *fa, tc.timestamp, tc.amount, tc.fee, rcp, att)
		tx.Signature = &sig
		tx.ID = &id
		b, err := tx.BodyMarshalBinary(TestNetScheme)
		require.NoError(t, err)
		h, _ := crypto.FastHash(b)
		assert.Equal(t, *tx.ID, h)
		if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
			assert.True(t, r)
		}
	}
}

func TestTransferWithSigToJSON(t *testing.T) {
	tests := []struct {
		amountAsset         string
		expectedAmountAsset string
		feeAsset            string
		expectedFeeAsset    string
		attachment          string
		expectedAttachment  string
	}{
		{"", "null", "", "null", "", ``},
		{"", "null", "", "null", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
		{"", "null", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
		{"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "\"B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ\"", "blah-blah-blah", ",\"attachment\":\"dBfDSWhwLmZQy4zr2S3\""},
	}
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	adr, err := NewAddressFromString("3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu")
	require.NoError(t, err)
	rcp := NewRecipientFromAddress(adr)
	ts := uint64(time.Now().Unix() * 1000)
	for _, tc := range tests {
		aa, _ := NewOptionalAssetFromString(tc.amountAsset)
		fa, _ := NewOptionalAssetFromString(tc.feeAsset)
		att := Attachment(tc.attachment)
		tx := NewUnsignedTransferWithSig(pk, *aa, *fa, ts, 100000000, 100000, rcp, att)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":4,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
			assert.Equal(t, ej, string(j))
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":4,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
				assert.Equal(t, ej, string(j))
			}
		}
	}
}

func TestTransferWithProofsValidations(t *testing.T) {
	var (
		w      = NewOptionalAssetWaves()
		a, err = NewOptionalAssetFromString("93H1i2jgP21Eh4Q5uzwmCYCVfGHZcAMzpC6PPbwvCSTs")
	)
	require.NoError(t, err)
	tests := []struct {
		recipient   string
		amount      uint64
		amountAsset OptionalAsset
		fee         uint64
		feeAsset    OptionalAsset
		att         string
		err         string
	}{
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 0, w, 10, w, "The attachment", "amount should be positive"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 1000, w, 0, w, "The attachment", "fee should be positive"},
		{"alias:T:nickname", math.MaxInt64 + 1, w, 10, w, "The attachment", "amount is too big"},
		{"alias:T:nickname", 1000, w, math.MaxInt64 + 1, w, "The attachment", "fee is too big"},
		{"alias:T:nickname", 1000, w, math.MaxInt64, w, "The attachment", "sum of amount and fee in the same asset overflows JVM long"},
		{"alias:T:nickname", math.MaxInt64, w, 1000, w, "The attachment", "sum of amount and fee in the same asset overflows JVM long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 1000, w, 10, w, strings.Repeat("The attachment", 100), "attachment is too long"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGQQgnWN", 1000, w, 10, w, "The attachment", "invalid recipient '3MxW8ZFCQUQDg7xagmGQQcwbQDmNGQQgnWN': invalid WavesAddress checksum"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ", 1000, w, 10, w, "The attachment", "invalid recipient '3PAWwWa6GbwcJaFzwqXQN5KQm7H86Y7SHTQ': invalid scheme 'W', expected 'T'"},
		{"alias:T:прозвище", 1000, w, 10, w, "The attachment", "invalid recipient 'alias:T:прозвище': Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz"},
		{"alias:W:invalid-scheme", 1000, w, 10, w, "The attachment", "invalid recipient 'alias:W:invalid-scheme': invalid scheme 'W', expected 'T'"},
		{"alias:W:прозвище", 1000, w, 10, w, "The attachment", "invalid recipient 'alias:W:прозвище': invalid scheme 'W', expected 'T'"},
		{"alias:T:nickname", 1000, w, math.MaxInt64, *a, "The attachment", ""},
		{"alias:T:nickname", math.MaxInt64, w, math.MaxInt64, *a, "The attachment", ""},
	}
	spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	require.NoError(t, err)
	for i, tc := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			rcp, err := recipientFromString(tc.recipient)
			require.NoError(t, err)
			att := []byte(tc.att)
			tx := NewUnsignedTransferWithProofs(2, spk, tc.amountAsset, tc.feeAsset, 0, tc.amount, tc.fee, rcp, att)
			_, err = tx.Validate(TestNetScheme)
			if tc.err != "" {
				assert.EqualError(t, err, tc.err, "No expected error '%s'", tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransferWithProofsFromMainNet(t *testing.T) {
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
		id, err := crypto.NewDigestFromBase58(tc.id)
		require.NoError(t, err)
		sig, err := crypto.NewSignatureFromBase58(tc.sig)
		require.NoError(t, err)
		spk, err := crypto.NewPublicKeyFromBase58(tc.spk)
		require.NoError(t, err)
		addr, err := NewAddressFromString(tc.rcp)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, spk, *aa, *fa, tc.timestamp, tc.amount, tc.fee, rcp, att)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestTransferWithProofsJSONRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := Attachment(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if js, err := json.Marshal(tx); assert.NoError(t, err) {
				tx2 := &TransferWithProofs{}
				if err := json.Unmarshal(js, tx2); assert.NoError(t, err) {
					assert.Equal(t, tx.Type, tx2.Type)
					assert.Equal(t, tx.Version, tx2.Version)
					assert.Equal(t, tx.SenderPK, tx2.SenderPK)
					assert.Equal(t, tx.Recipient, tx2.Recipient)
					assert.Equal(t, tx.AmountAsset.Present, tx2.AmountAsset.Present)
					assert.ElementsMatch(t, tx.AmountAsset.ID, tx2.AmountAsset.ID)
					assert.Equal(t, tx.FeeAsset.Present, tx2.FeeAsset.Present)
					assert.ElementsMatch(t, tx.FeeAsset.ID, tx2.FeeAsset.ID)
					assert.Equal(t, tx.Amount, tx2.Amount)
					assert.Equal(t, tx.Fee, tx2.Fee)
					assert.Equal(t, tx.Timestamp, tx2.Timestamp)
					assert.True(t, bytes.Equal(tx.Attachment, tx2.Attachment))
					_, err := tx2.MarshalBinary(tc.scheme)
					require.NoError(t, err)
				}
			}
		}
	}
}

func TestTransferWithProofsProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		err = tx.GenerateID(tc.scheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.scheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestTransferWithProofsBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		err = tx.Sign(tc.scheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.scheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestTransferWithProofsBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		addr, err := NewAddressFromPublicKey(tc.scheme, pk)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
		if bb, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithProofs
			if err := atx.BodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Recipient, atx.Recipient)
				assert.Equal(t, tx.AmountAsset.Present, atx.AmountAsset.Present)
				assert.ElementsMatch(t, tx.AmountAsset.ID, atx.AmountAsset.ID)
				assert.Equal(t, tx.FeeAsset.Present, atx.FeeAsset.Present)
				assert.ElementsMatch(t, tx.FeeAsset.ID, atx.FeeAsset.ID)
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
				assert.Equal(t, tx.Attachment, atx.Attachment)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx TransferWithProofs
			if err := atx.UnmarshalBinary(b, tc.scheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
				assert.Equal(t, pk, atx.SenderPK)
				assert.Equal(t, tc.expectedAmountAsset, atx.AmountAsset.String())
				assert.Equal(t, tc.expectedFeeAsset, atx.FeeAsset.String())
				assert.Equal(t, tc.amount, atx.Amount)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
				att := Attachment(tc.attachment)
				assert.Equal(t, att, atx.Attachment)
			}
		}
	}
}

func BenchmarkTransferWithProofsBinary(t *testing.B) {
	tc := struct {
		scheme              byte
		amountAsset         string
		expectedAmountAsset string
		feeAsset            string
		expectedFeeAsset    string
		amount              uint64
		fee                 uint64
		attachment          string
	}{'W', "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "B1u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ", "", "WAVES", 10, 20, ""}

	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	ts := uint64(time.Now().UnixNano() / 1000000)
	addr, err := NewAddressFromPublicKey(tc.scheme, pk)
	require.NoError(t, err)
	rcp := NewRecipientFromAddress(addr)
	aa, err := NewOptionalAssetFromString(tc.amountAsset)
	require.NoError(t, err)
	fa, err := NewOptionalAssetFromString(tc.feeAsset)
	require.NoError(t, err)
	att := []byte(tc.attachment)
	tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, tc.amount, tc.fee, rcp, att)
	if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
		if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
			assert.True(t, r)
		}
	}
	s := serializer.New(io.Discard)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		err = tx.Serialize(s)
	}
	t.StopTimer()
	if err != nil {
		t.FailNow()
	}
}

func TestTransferWithProofsToJSON(t *testing.T) {
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
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	addr, err := NewAddressFromString("3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu")
	require.NoError(t, err)
	rcp := NewRecipientFromAddress(addr)
	ts := uint64(time.Now().UnixNano() / 1000000)
	for _, tc := range tests {
		aa, err := NewOptionalAssetFromString(tc.amountAsset)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		att := []byte(tc.attachment)
		tx := NewUnsignedTransferWithProofs(2, pk, *aa, *fa, ts, 100000000, 100000, rcp, att)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":4,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}", base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
			assert.Equal(t, ej, string(j))
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if j, err := json.Marshal(tx); assert.NoError(t, err) {
				ej := fmt.Sprintf("{\"type\":4,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":%s,\"feeAssetId\":%s,\"timestamp\":%d,\"amount\":100000000,\"fee\":100000,\"recipient\":\"3PDgLyMzNLkHF2cV1y7NhpmyS2HQjd57SWu\"%s}",
					base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.expectedAmountAsset, tc.expectedFeeAsset, ts, tc.expectedAttachment)
				assert.Equal(t, ej, string(j))
			}
		}
	}
}

func TestTransferWithProofsFromJSON(t *testing.T) {
	var js = `{"senderPublicKey":"9uVCXj92oiUdtMWkwSLyKXRnHju81m3aGRzU2ZhJ91nF","recipient":"3FcSgww3tKZ7feQVmcnPFmRxsjqBodYz63x","amount":1,"assetId":null,"fee":100000,"feeAssetId":null,"attachment":"bQbp","timestamp":1549972745180,"proofs":["45yF4TTn9CtyJbH7BPVZYK92DFhuHCCDCn9fuEuFrcDhG7Fa4SbsmHi2ouQKw8u1AxkqsrPbeEPqiNHZfFw35Z3M"],"version":2,"type":4}`
	spk, err := crypto.NewPublicKeyFromBase58("9uVCXj92oiUdtMWkwSLyKXRnHju81m3aGRzU2ZhJ91nF")
	require.NoError(t, err)
	addr, err := NewAddressFromString("3FcSgww3tKZ7feQVmcnPFmRxsjqBodYz63x")
	require.NoError(t, err)
	var tx TransferWithProofs
	err = json.Unmarshal([]byte(js), &tx)
	require.NoError(t, err)
	assert.Equal(t, TransferTransaction, tx.Type)
	assert.Equal(t, 2, int(tx.Version))
	assert.Equal(t, uint64(1549972745180), tx.Timestamp)
	assert.Equal(t, 100000, int(tx.Fee))
	assert.Equal(t, 1, int(tx.Amount))
	assert.False(t, tx.AmountAsset.Present)
	assert.False(t, tx.FeeAsset.Present)
	assert.Equal(t, 1, len(tx.Proofs.Proofs))
	assert.ElementsMatch(t, spk[:], tx.SenderPK[:])
	assert.ElementsMatch(t, addr[:], tx.Recipient.Address()[:])
}

func TestReissueWithSigValidations(t *testing.T) {
	tests := []struct {
		quantity uint64
		fee      uint64
		err      string
	}{
		{0, 100000, "quantity should be positive"},
		{math.MaxInt64 + 1, 100000, "quantity is too big"},
		{100000, 0, "fee should be positive"},
		{100000, math.MaxInt64 + 1, "fee is too big"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		aid, err := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		tx := NewUnsignedReissueWithSig(spk, aid, tc.quantity, false, 0, tc.fee)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestReissueWithSigFromMainNet(t *testing.T) {
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
		tx := NewUnsignedReissueWithSig(spk, aid, tc.quantity, tc.reissuable, tc.timestamp, tc.fee)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestReissueWithSigProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithSig(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ReissueWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ReissueWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestReissueWithSigBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithSig(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestReissueWithSigBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithSig(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ReissueWithSig
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
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ReissueWithSig
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
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

func TestReissueWithSigToJSON(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, err := crypto.NewDigestFromBase58(tc.asset)
		require.NoError(t, err)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedReissueWithSig(pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":5,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":5,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestReissueWithProofsValidations(t *testing.T) {
	tests := []struct {
		quantity uint64
		fee      uint64
		err      string
	}{
		{0, 100000, "quantity should be positive"},
		{math.MaxInt64 + 1, 100000, "quantity is too big"},
		{100000, 0, "fee should be positive"},
		{100000, math.MaxInt64 + 1, "fee is too big"},
		//TODO: add blockchain scheme validation
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedReissueWithProofs(2, spk, aid, tc.quantity, false, 0, tc.fee)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestReissueWithProofsFromMainNetAndTestNet(t *testing.T) {
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
		tx := NewUnsignedReissueWithProofs(2, spk, aid, tc.quantity, tc.reissuable, tc.timestamp, tc.fee)
		if b, err := tx.BodyMarshalBinary(tc.chain); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestReissueWithProofsProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithProofs(2, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		err = tx.GenerateID(tc.chain)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.chain); assert.NoError(t, err) {
			var atx ReissueWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chain, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.chain); assert.NoError(t, err) {
			var atx ReissueWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.chain)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestReissueWithProofsBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithProofs(2, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		err := tx.Sign(tc.chain, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.chain)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestReissueWithProofsBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedReissueWithProofs(2, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		if bb, err := tx.BodyMarshalBinary(tc.chain); assert.NoError(t, err) {
			var atx ReissueWithProofs
			if err := atx.bodyUnmarshalBinary(bb, tc.chain); assert.NoError(t, err) {
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
		if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chain, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.chain); assert.NoError(t, err) {
			var atx ReissueWithProofs
			if err := atx.UnmarshalBinary(b, tc.chain); assert.NoError(t, err) {
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

func TestReissueWithProofsToJSON(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedReissueWithProofs(2, pk, aid, tc.quantity, tc.reissuable, ts, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":5,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":5,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"quantity\":%d,\"reissuable\":%v,\"timestamp\":%d,\"fee\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.quantity, tc.reissuable, ts, tc.fee)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestBurnWithSigValidations(t *testing.T) {
	tests := []struct {
		amount uint64
		fee    uint64
		err    string
	}{
		{math.MaxInt64 + 1, 100000, "amount is too big"},
		{100000, 0, "fee should be positive"},
		{100000, math.MaxInt64 + 1, "fee is too big"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedBurnWithSig(spk, aid, tc.amount, 0, tc.fee)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestBurnWithSigFromMainNet(t *testing.T) {
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
		tx := NewUnsignedBurnWithSig(spk, aid, tc.amount, tc.timestamp, tc.fee)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestBurnWithSigProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithSig(pk, aid, tc.amount, ts, tc.fee)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestBurnWithSigBinarySize(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithSig(pk, aid, tc.amount, ts, tc.fee)
		err := tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestBurnWithSigBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithSig(pk, aid, tc.amount, ts, tc.fee)
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithSig
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
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

func TestBurnWithSigToJSON(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedBurnWithSig(pk, aid, tc.amount, ts, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":6,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":6,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestBurnWithProofsValidations(t *testing.T) {
	tests := []struct {
		chain  byte
		amount uint64
		fee    uint64
		err    string
	}{
		{'T', math.MaxInt64 + 10, 100000, "amount is too big"},
		{'T', 100000, 0, "fee should be positive"},
		{'T', 100000, math.MaxInt64 + 1, "fee is too big"},
		//TODO: add blockchain scheme validation tests
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		aid, _ := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedBurnWithProofs(2, spk, aid, tc.amount, 0, tc.fee)
		_, err := tx.Validate(tc.chain)
		assert.EqualError(t, err, tc.err)
	}
}

func TestBurnWithProofsFromMainNet(t *testing.T) {
	const scheme = MainNetScheme

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
		tx := NewUnsignedBurnWithProofs(2, spk, aid, tc.amount, tc.timestamp, tc.fee)
		if b, err := tx.BodyMarshalBinary(scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestBurnWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithProofs(2, pk, aid, tc.amount, ts, tc.fee)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestBurnWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithProofs(2, pk, aid, tc.amount, ts, tc.fee)
		err := tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		require.NoError(t, err)
		require.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestBurnWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedBurnWithProofs(2, pk, aid, tc.amount, ts, tc.fee)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithProofs
			if err := atx.bodyUnmarshalBinary(bb, TestNetScheme); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, tx.AssetID, atx.AssetID)
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx BurnWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
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

func TestBurnWithProofsToJSON(t *testing.T) {
	tests := []struct {
		asset  string
		amount uint64
		fee    uint64
	}{
		{"8LQW8f7P5d5PZM7GtZEBgaqRPGSzS3DfPuiXrURJ4AJS", 1234567890, 1234567890},
		{"6zf9mSeHUKRzWR6rCBWPmFPTkhg22qvwUZjTBCfxBkGJ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		aid, _ := crypto.NewDigestFromBase58(tc.asset)
		ts := uint64(time.Now().Unix() * 1000)
		tx := NewUnsignedBurnWithProofs(2, pk, aid, tc.amount, ts, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":6,\"version\":2,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":6,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"amount\":%d,\"timestamp\":%d,\"fee\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.amount, ts, tc.fee)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestExchangeWithSigValidations(t *testing.T) {
	buySender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	sellSender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	mpk, _ := crypto.NewPublicKeyFromBase58("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	mpk2, _ := crypto.NewPublicKeyFromBase58("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	aa2, _ := NewOptionalAssetFromString("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	sbo0 := newSignedOrderV1(t, buySender, mpk, *aa, *pa, Buy, 1000000000, 100, 10, 10+MaxOrderTTL, 3)
	sbo1 := newSignedOrderV1(t, buySender, mpk, *aa, *pa, Buy, math.MaxInt64+1, 100, 10, 10+MaxOrderTTL, 3)
	sbo2 := newSignedOrderV1(t, buySender, mpk, *aa2, *pa, Buy, 1000000000, 100, 10, 10+MaxOrderTTL, 3)
	sso0 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, 900000000, 50, 20, 20+MaxOrderTTL, 3)
	sso1 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, math.MaxInt64+1, 50, 20, 20+MaxOrderTTL, 3)
	sso2 := newSignedOrderV1(t, sellSender, mpk2, *aa, *pa, Sell, 900000000, 50, 10, 10+MaxOrderTTL, 3)
	sso3 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, 900000000, 50, 20, 5+MaxOrderTTL, 3)
	tests := []struct {
		buy     OrderV1
		sell    OrderV1
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
		ts      uint64
		err     string
	}{
		{sbo1, sso0, 123, 456, 789, 987, 654, 111, "invalid buy order: price is too big"},
		{sbo0, sso1, 123, 456, 789, 987, 654, 111, "invalid sell order: price is too big"},
		{sbo0, sso0, 0, 456, 789, 987, 654, 111, "price should be positive"},
		{sbo0, sso0, math.MaxInt64 + 1, 456, 789, 987, 654, 111, "price is too big"},
		{sbo0, sso0, 950000000, 0, 789, 987, 654, 111, "amount should be positive"},
		{sbo0, sso0, 950000000, math.MaxInt64 + 1, 789, 987, 654, 111, "amount is too big"},
		{sbo0, sso0, 950000000, 456, math.MaxInt64 + 1, 987, 654, 111, "buy matcher's fee is too big"},
		{sbo0, sso0, 950000000, 456, 789, math.MaxInt64 + 1, 654, 111, "sell matcher's fee is too big"},
		{sbo0, sso0, 950000000, 456, 789, 987, 0, 111, "fee should be positive"},
		{sbo0, sso0, 950000000, 456, 789, 987, math.MaxInt64 + 1, 111, "fee is too big"},
		{sso0, sso0, 950000000, 456, 789, 987, 654, 111, "incorrect order type of buy order"},
		{sbo0, sbo0, 950000000, 456, 789, 987, 654, 111, "incorrect order type of sell order"},
		{sbo0, sso2, 950000000, 456, 789, 987, 654, 111, "unmatched matcher's public keys"},
		{sbo2, sso0, 950000000, 456, 789, 987, 654, 111, "different asset pairs"},
		{sbo0, sso0, 890000000, 456, 789, 987, 654, 111, "invalid price"},
		{sbo0, sso0, 1010000000, 456, 789, 987, 654, 111, "invalid price"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, 1, "buy order expiration should be earlier than 30 days"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, 11, "sell order expiration should be earlier than 30 days"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, MaxOrderTTL + 15, "invalid buy order expiration"},
		{sbo0, sso3, 950000000, 456, 789, 987, 654, MaxOrderTTL + 10, "invalid sell order expiration"},
		{sso0, sbo0, 123, 456, 789, 987, 654, 111, "incorrect order type of buy order"},
		{sbo0, sbo0, 123, 456, 789, 987, 654, 111, "incorrect order type of sell order"},
	}
	for _, tc := range tests {
		tx := NewUnsignedExchangeWithSig(&tc.buy, &tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, tc.ts)
		_, err := tx.Validate(TestNetScheme)
		assert.Error(t, err)
		assert.Regexp(t, tc.err, err.Error(), fmt.Sprintf("expected: %s", tc.err))
	}
}

func newSignedOrderV1(t *testing.T, sender, matcher crypto.PublicKey, amountAsset, priceAsset OptionalAsset, ot OrderType, price, amount, ts, exp, fee uint64) OrderV1 {
	id, err := crypto.NewDigestFromBase58("AkYY8M2iEts8xc21JEzwkMSmuJtH9ABGzEYeau4xWC5R")
	require.NoError(t, err)
	sig, err := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	require.NoError(t, err)
	o := NewUnsignedOrderV1(sender, matcher, amountAsset, priceAsset, ot, price, amount, ts, exp, fee)
	o.ID = &id
	o.Signature = &sig
	return *o
}

func newSignedOrderV4(t *testing.T, sender, matcher crypto.PublicKey, amountAsset, priceAsset OptionalAsset, ot OrderType, price, amount, ts, exp, fee uint64, sID, sSig string, priceMode OrderPriceMode) OrderV4 {
	id, err := crypto.NewDigestFromBase58(sID)
	require.NoError(t, err)
	sig, err := crypto.NewSignatureFromBase58(sSig)
	require.NoError(t, err)
	o := NewUnsignedOrderV4(sender, matcher, amountAsset, priceAsset, ot, price, amount, ts, exp, fee, OptionalAsset{}, priceMode)
	o.ID = &id
	o.Proofs = NewProofsFromSignature(&sig)
	return *o
}

func newEthereumOrderV4(t *testing.T, ethSenderPKHex, ethSignatureHex, matcherPKBase58, amountAssetBase58, priceAssetBase58 string, ot OrderType, price, amount, ts, exp, fee uint64, priceMode OrderPriceMode) EthereumOrderV4 {
	var (
		err       error
		ethSender EthereumPublicKey
		ethSig    EthereumSignature
	)
	if ethSenderPKHex != "" {
		ethSender, err = NewEthereumPublicKeyFromHexString(ethSenderPKHex)
		require.NoError(t, err)
	}
	if ethSignatureHex != "" {
		ethSig, err = NewEthereumSignatureFromHexString(ethSignatureHex)
		require.NoError(t, err)
	}

	matcher, err := crypto.NewPublicKeyFromBase58(matcherPKBase58)
	require.NoError(t, err)
	amountAsset, err := NewOptionalAssetFromString(amountAssetBase58)
	require.NoError(t, err)
	priceAsset, err := NewOptionalAssetFromString(priceAssetBase58)
	require.NoError(t, err)

	ethereumOrderV4 := NewUnsignedEthereumOrderV4(&ethSender, matcher, *amountAsset, *priceAsset, ot, price, amount, ts, exp, fee, OptionalAsset{}, priceMode)
	ethereumOrderV4.Eip712Signature = ethSig
	return *ethereumOrderV4
}

func TestExchangeWithSigFromMainNet(t *testing.T) {
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
		bo := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, tc.buyTs, tc.buyExp, tc.buyFee)
		bID, _ := crypto.NewDigestFromBase58(tc.buyID)
		bSig, _ := crypto.NewSignatureFromBase58(tc.buySig)
		bo.ID = &bID
		bo.Signature = &bSig
		so := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, tc.sellTs, tc.sellExp, tc.sellFee)
		sID, _ := crypto.NewDigestFromBase58(tc.sellID)
		sSig, _ := crypto.NewSignatureFromBase58(tc.sellSig)
		so.ID = &sID
		so.Signature = &sSig
		tx := NewUnsignedExchangeWithSig(bo, so, tc.price, tc.amount, tc.buyMatcherFee, tc.sellMatcherFee, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(mpk, sig, b))
		}
	}
}

func TestExchangeWithSigProtobufRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := NewTimestampFromTime(time.Now())
	exp := ts + 100*1000
	bo := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 98765, 67890, ts, exp, 3)
	err = bo.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	tests := []struct {
		buy     OrderV1
		sell    OrderV1
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{*bo, *so, 98765, 456, 789, 987, 654},
	}
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedExchangeWithSig(&tc.buy, &tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, mpk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestExchangeWithSigBinarySize(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so.Sign(TestNetScheme, sk)
	require.NoError(t, err)
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
		tx := NewUnsignedExchangeWithSig(&tc.buy, &tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		err = tx.Sign(TestNetScheme, msk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestExchangeWithSigBinaryRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so.Sign(TestNetScheme, sk)
	require.NoError(t, err)
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
		tx := NewUnsignedExchangeWithSig(&tc.buy, &tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithSig
			if _, err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Order1, atx.Order1)
				assert.Equal(t, tx.Order2, atx.Order2)
				assert.Equal(t, tx.Price, atx.Price)
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.BuyMatcherFee, atx.BuyMatcherFee)
				assert.Equal(t, tx.SellMatcherFee, atx.SellMatcherFee)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, mpk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithSig
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.Signature, atx.Signature)
				assert.Equal(t, mpk, atx.SenderPK)
				assert.Equal(t, bo.ID, atx.Order1.ID)
				assert.Equal(t, so.ID, atx.Order2.ID)
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

func TestExchangeWithSigToJSON(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
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
		bo := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, ts, exp, tc.fee)
		err := bo.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		boj, _ := json.Marshal(bo)
		so := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, ts, exp, tc.fee)
		err = so.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		soj, _ := json.Marshal(so)
		tx := NewUnsignedExchangeWithSig(bo, so, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":7,\"version\":1,\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
				if j, err := json.Marshal(tx); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"type\":7,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}

func TestExchangeWithProofsValidations(t *testing.T) {
	buySender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	sellSender, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	mpk, _ := crypto.NewPublicKeyFromBase58("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	mpk2, _ := crypto.NewPublicKeyFromBase58("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	aa2, _ := NewOptionalAssetFromString("E7zJzWVn6kwsc6zwDpxZrEFjUu3xszPZ7XcStYNprbSJ")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	sbo0 := newSignedOrderV1(t, buySender, mpk, *aa, *pa, Buy, 1000000000, 100, 10, 10+MaxOrderTTL, 3)
	sbo1 := newSignedOrderV1(t, buySender, mpk, *aa, *pa, Buy, math.MaxInt64+1, 100, 10, 10+MaxOrderTTL, 3)
	sbo2 := newSignedOrderV1(t, buySender, mpk, *aa2, *pa, Buy, 1000000000, 100, 10, 10+MaxOrderTTL, 3)
	sso0 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, 900000000, 50, 20, 20+MaxOrderTTL, 3)
	sso1 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, math.MaxInt64+1, 50, 20, 20+MaxOrderTTL, 3)
	sso2 := newSignedOrderV1(t, sellSender, mpk2, *aa, *pa, Sell, 900000000, 50, 10, 10+MaxOrderTTL, 3)
	sso3 := newSignedOrderV1(t, sellSender, mpk, *aa, *pa, Sell, 900000000, 50, 20, 5+MaxOrderTTL, 3)
	tests := []struct {
		buy     OrderV1
		sell    OrderV1
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
		ts      uint64
		err     string
	}{
		{sbo1, sso0, 123, 456, 789, 987, 654, 111, "invalid first order: price is too big"},
		{sbo0, sso1, 123, 456, 789, 987, 654, 111, "invalid second order: price is too big"},
		{sbo0, sso0, 0, 456, 789, 987, 654, 111, "price should be positive"},
		{sbo0, sso0, math.MaxInt64 + 1, 456, 789, 987, 654, 111, "price is too big"},
		{sbo0, sso0, 950000000, 0, 789, 987, 654, 111, "amount should be positive"},
		{sbo0, sso0, 950000000, math.MaxInt64 + 1, 789, 987, 654, 111, "amount is too big"},
		{sbo0, sso0, 950000000, 456, math.MaxInt64 + 1, 987, 654, 111, "buy matcher's fee is too big"},
		{sbo0, sso0, 950000000, 456, 789, math.MaxInt64 + 1, 654, 111, "sell matcher's fee is too big"},
		{sbo0, sso0, 950000000, 456, 789, 987, 0, 111, "fee should be positive"},
		{sbo0, sso0, 950000000, 456, 789, 987, math.MaxInt64 + 1, 111, "fee is too big"},
		{sso0, sso0, 950000000, 456, 789, 987, 654, 111, "incorrect combination of orders types"},
		{sbo0, sbo0, 950000000, 456, 789, 987, 654, 111, "incorrect combination of orders types"},
		{sbo0, sso2, 950000000, 456, 789, 987, 654, 111, "unmatched matcher's public keys"},
		{sbo2, sso0, 950000000, 456, 789, 987, 654, 111, "different asset pairs"},
		{sbo0, sso0, 890000000, 456, 789, 987, 654, 111, "invalid price"},
		{sbo0, sso0, 1010000000, 456, 789, 987, 654, 111, "invalid price"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, 1, "first order expiration should be earlier than 30 days"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, 11, "second order expiration should be earlier than 30 days"},
		{sbo0, sso0, 950000000, 456, 789, 987, 654, MaxOrderTTL + 15, "invalid first order expiration"},
		{sbo0, sso3, 950000000, 456, 789, 987, 654, MaxOrderTTL + 10, "invalid second order expiration"},
	}
	for _, tc := range tests {
		tx := NewUnsignedExchangeWithProofs(2, &tc.buy, &tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, tc.ts)
		_, err := tx.Validate(TestNetScheme)
		assert.Error(t, err)
		assert.Regexp(t, tc.err, err, fmt.Sprintf("expected error: %s", tc.err))
	}
}

func TestExchangeV3PriceValidation(t *testing.T) {
	buySender, _ := crypto.NewPublicKeyFromBase58("HFw9wjsuaeZ6w8HqJeyxz9RUXJN8krLpgjwtPrqyicr")
	sellSender, _ := crypto.NewPublicKeyFromBase58("BvJEWY79uQEFetuyiZAF5U4yjPioMj9J6ZrF9uTNfe3E")
	mpk, _ := crypto.NewPublicKeyFromBase58("BvJEWY79uQEFetuyiZAF5U4yjPioMj9J6ZrF9uTNfe3E")
	aa, _ := NewOptionalAssetFromString("3JmaWyFqWo8YSA8x3DXCBUW7veesxacvKx19dMv7wTMg")
	pa, _ := NewOptionalAssetFromString("25FEqEjRkqK6yCkiT7Lz6SAYz7gUFCtxfCChnrVFD5AT")
	sbo := newSignedOrderV4(t, buySender, mpk, *aa, *pa, Buy, 1000000, 800000000, 1624445095222, 1626950695222, 300000, "3fdNTCQ7o2TvN8eDV3m7J9aSLxcUitwN2SMZpn1irSXX", "3aKUz8boZingH8r18grL8Rst5RyGVnESaQtuEoV5piUnvJKNf67xFwFpPpmfiuAuud1AAzj94xYNw1MKkmJaBicR", OrderPriceModeDefault)
	sso := newSignedOrderV4(t, sellSender, mpk, *aa, *pa, Sell, 1000000, 800000000, 1624445095267, 1626950695267, 300000, "81Xc8YP1Ev2bqvSLgN5k3ent6Fr7rnEdCg8x2DH5twqX", "4VQmM6QB8yaQ1AChNNkVH5EvVKenS8YG7YqXK9SsjWAnjJm5xvd48kW2akwcEbhgzqqGMDtS2AmeGSfpEcHEMYGU", OrderPriceModeDefault)
	tx := NewUnsignedExchangeWithProofs(3, &sbo, &sso, 100000000, 800000000, 100, 100, 300000, 1624445095293)
	_, err := tx.Validate(TestNetScheme)
	assert.NoError(t, err)
}

func TestExchangeWithProofsFromTestNet(t *testing.T) {
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
		bo := NewUnsignedOrderV1(buySender, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, tc.buyTs, tc.buyExp, tc.buyFee)
		bID, _ := crypto.NewDigestFromBase58(tc.buyID)
		bSig, _ := crypto.NewSignatureFromBase58(tc.buySig)
		bo.ID = &bID
		bo.Signature = &bSig
		so := NewUnsignedOrderV1(sellSender, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, tc.sellTs, tc.sellExp, tc.sellFee)
		sID, _ := crypto.NewDigestFromBase58(tc.sellID)
		sSig, _ := crypto.NewSignatureFromBase58(tc.sellSig)
		so.ID = &sID
		so.Signature = &sSig
		tx := NewUnsignedExchangeWithProofs(2, bo, so, tc.price, tc.amount, tc.buyMatcherFee, tc.sellMatcherFee, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(mpk, sig, b))
		}
	}
}

func TestExchangeWithProofsProtobufRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := NewTimestampFromTime(time.Now())
	exp := ts + 100*1000
	bo1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 12345, 54321, ts, exp, 3)
	err = so1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	bo2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, 12345, 54321, ts, exp, 3)
	err = so2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{bo1, so1, 12345, 456, 789, 987, 654},
		{bo2, so2, 12345, 544321, 9876, 8765, 13245},
		{bo1, so2, 12345, 456, 789, 987, 654},
		{bo2, so1, 12345, 544321, 9876, 8765, 13245},
		{so1, bo1, 12345, 456, 789, 987, 654},
		{so2, bo2, 12345, 544321, 9876, 8765, 13245},
		{so1, bo2, 12345, 456, 789, 987, 654},
		{so2, bo1, 12345, 544321, 9876, 8765, 13245},
	}
	for _, tc := range tests {
		ts := NewTimestampFromTime(time.Now())
		tx := NewUnsignedExchangeWithProofs(2, tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, mpk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestExchangeWithProofsBinarySize(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	bo2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{bo1, so1, 123, 456, 789, 987, 654},
		{bo2, so2, 987654321, 544321, 9876, 8765, 13245},
		{bo1, so2, 123, 456, 789, 987, 654},
		{bo2, so1, 987654321, 544321, 9876, 8765, 13245},
	}
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedExchangeWithProofs(2, tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		err = tx.Sign(TestNetScheme, msk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestExchangeWithProofsBinaryRoundTrip(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
	aa, _ := NewOptionalAssetFromString("3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N")
	pa, _ := NewOptionalAssetFromString("FftTzae2t8r6zZJ2VzEq2pS2Le4Vx9gYGXuDsEFBTYE2")
	ts := uint64(time.Now().UnixNano() / 1000000)
	exp := ts + 100*1000
	bo1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so1 := NewUnsignedOrderV1(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so1.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	bo2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, 12345, 67890, ts, exp, 3)
	err = bo2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	so2 := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, 98765, 54321, ts, exp, 3)
	err = so2.Sign(TestNetScheme, sk)
	require.NoError(t, err)
	tests := []struct {
		buy     Order
		sell    Order
		price   uint64
		amount  uint64
		buyFee  uint64
		sellFee uint64
		fee     uint64
	}{
		{bo1, so1, 123, 456, 789, 987, 654},
		{bo2, so2, 987654321, 544321, 9876, 8765, 13245},
		{bo1, so2, 123, 456, 789, 987, 654},
		{bo2, so1, 987654321, 544321, 9876, 8765, 13245},
	}
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedExchangeWithProofs(2, tc.buy, tc.sell, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithProofs
			if _, err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Order1, atx.Order1)
				assert.Equal(t, tx.Order2, atx.Order2)
				assert.Equal(t, tx.Price, atx.Price)
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.BuyMatcherFee, atx.BuyMatcherFee)
				assert.Equal(t, tx.SellMatcherFee, atx.SellMatcherFee)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, mpk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx ExchangeWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
				assert.Equal(t, mpk, atx.SenderPK)
				assert.EqualValues(t, tc.buy, atx.Order1)
				assert.Equal(t, tc.sell, atx.Order2)
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

func TestExchangeWithProofsToJSON(t *testing.T) {
	seedA, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seedA)
	require.NoError(t, err)
	seedB, _ := base58.Decode("8cLFt3NHL13H5JCBBgbJDkjjcPseZ1YNtqMWnZS1B2n9")
	msk, mpk, err := crypto.GenerateKeyPair(seedB)
	require.NoError(t, err)
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
		bo := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Buy, tc.buyPrice, tc.buyAmount, ts, exp, tc.fee)
		err := bo.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		boj, _ := json.Marshal(bo)
		so := NewUnsignedOrderV2(pk, mpk, *aa, *pa, Sell, tc.sellPrice, tc.sellAmount, ts, exp, tc.fee)
		err = so.Sign(TestNetScheme, sk)
		require.NoError(t, err)
		soj, _ := json.Marshal(so)
		tx := NewUnsignedExchangeWithProofs(2, bo, so, tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":7,\"version\":2,\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, msk); assert.NoError(t, err) {
				if j, err := json.Marshal(tx); assert.NoError(t, err) {
					ej := fmt.Sprintf("{\"type\":7,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"order1\":%s,\"order2\":%s,\"price\":%d,\"amount\":%d,\"buyMatcherFee\":%d,\"sellMatcherFee\":%d,\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(mpk[:]), string(boj), string(soj), tc.price, tc.amount, tc.buyFee, tc.sellFee, tc.fee, ts)
					assert.Equal(t, ej, string(j))
				}
			}
		}
	}
}

func TestExchangeWithProofsFromJSON1(t *testing.T) {
	var js = `
{
      "type": 7,
      "id": "7umRMoUZfYinCM9jFyAmn9FaPL8Pf5D45mDucDJobpmW",
      "sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
      "senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
      "fee": 300000,
      "timestamp": 1548739881830,
      "proofs": [
        "5bCn1xwHp1uWVSTZLjVZDBH2MmA7jVz8uyQ29pECFW1o16CDo3QUX1uYBiB6z7QqaBn2G8sjL3DQuNQpRZRLbU8f"
      ],
      "version": 2,
      "order1": {
        "version": 2,
        "id": "4DAhJFiZzDnFxiQUPpb1kiMkzNbmyYkfnqXCov9JDLnK",
        "sender": "3P2vp33vwNGir7ixeCR4APTj48kRn8PhHpv",
        "senderPublicKey": "BM8y823b3wRqTSakixu6oQ6kw8YypKy8STgirAmPFuTW",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "EZFN36KbtnZTS5TTfDETfcEcjWxU1QguBS9drBRUpDwh",
          "priceAsset": null
        },
        "orderType": "buy",
        "amount": 4000000000000000,
        "price": 105,
        "timestamp": 1548739881077,
        "expiration": 1551245481076,
        "matcherFee": 300000,
        "signature": "4cAxQCehMHzK7acVwBt6NTw6b3buejtwMMRLkWkTMcsKA81LoMmdyTmpBTVt9n9m1zy4Wxh69w2gQ3pbom31R2Zc",
        "proofs": [
          "4cAxQCehMHzK7acVwBt6NTw6b3buejtwMMRLkWkTMcsKA81LoMmdyTmpBTVt9n9m1zy4Wxh69w2gQ3pbom31R2Zc"
        ]
      },
      "order2": {
        "version": 1,
        "id": "CHVi236M3Zmngd3sisHhWSZs5kSy5bmhZsoyaVeqoZrp",
        "sender": "3PNeE51To42hYSUkefzNLQfGdpAqRCbiUnw",
        "senderPublicKey": "DGB3jLytA97M2kYDPNUFtVkpXprmzgEa3kBpGGpkqi3r",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "EZFN36KbtnZTS5TTfDETfcEcjWxU1QguBS9drBRUpDwh",
          "priceAsset": null
        },
        "orderType": "sell",
        "amount": 10000000000000000,
        "price": 105,
        "timestamp": 1548191511217,
        "expiration": 1550697111217,
        "matcherFee": 300000,
        "signature": "5CRoPU8YkGyRddvn2GZifTPaqiw56JAXvy4Jy79SvkyZB7eS6DMEqxsD7eKd4EERhyJQwggTLMN7tdXzbF95apA1",
        "proofs": [
          "5CRoPU8YkGyRddvn2GZifTPaqiw56JAXvy4Jy79SvkyZB7eS6DMEqxsD7eKd4EERhyJQwggTLMN7tdXzbF95apA1"
        ]
      },
      "amount": 2107478007619048,
      "price": 105,
      "buyMatcherFee": 158060,
      "sellMatcherFee": 63224
    }
`
	var tx ExchangeWithProofs
	err := tx.UnmarshalJSON([]byte(js))
	assert.NoError(t, err)
	assert.Equal(t, ExchangeTransaction, tx.Type)
	assert.Equal(t, 2, int(tx.Version))
	assert.Equal(t, 2, int(tx.Order1.GetVersion()))
	assert.Equal(t, 1, int(tx.Order2.GetVersion()))
	bo, ok := tx.Order1.(*OrderV2)
	assert.True(t, ok)
	assert.NotNil(t, bo)
	so, ok := tx.Order2.(*OrderV1)
	assert.True(t, ok)
	assert.NotNil(t, so)
}

func TestExchangeWithProofsFromJSON2(t *testing.T) {
	var js = `
{
      "type": 7,
      "id": "HgmxEboQEgLgEK7tneqoXjg1pY7pWNazzfJ2hN2pKjAd",
      "sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
      "senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
      "fee": 300000,
      "timestamp": 1548739898607,
      "proofs": [
        "uKXSzjvM2Re6iJ1Pg24yYPvakBSfuyde6rW4QpP6SgwEfrNk5mWfMF29n3CHsBGi8VnzB7dsrSVvKVfhtZi9enN"
      ],
      "version": 2,
      "order1": {
        "version": 1,
        "id": "qs2XukcRkodoi2h2RgVq7Z45g7b7DEHkXtoQCvcLAes",
        "sender": "3PJbKNtRUr5HgwoZvSaWjAVbDWKpyetqYES",
        "senderPublicKey": "67JC7CAy46JmdTARj6Z6KxWMyRRZLdkuQbSFQJZm34XU",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "AFKQxw7A5WrzW2LMDoSoJhcSQ2rLGsepZbWMLsKKBQ2K",
          "priceAsset": "474jTeYx2r2Va35794tCScAXWJG9hU2HcgxzMowaZUnu"
        },
        "orderType": "buy",
        "amount": 167410000,
        "price": 220000,
        "timestamp": 1548739898192,
        "expiration": 1551245498192,
        "matcherFee": 300000,
        "signature": "2wJE9MfhBzTEXjG8ioGDJjgUKjKgune63jL8G58QYexNEeX1nP3fzQDD1aZszXUbozSFnsvPgKowohXCmJXhh3iz",
        "proofs": [
          "2wJE9MfhBzTEXjG8ioGDJjgUKjKgune63jL8G58QYexNEeX1nP3fzQDD1aZszXUbozSFnsvPgKowohXCmJXhh3iz"
        ]
      },
      "order2": {
        "version": 1,
        "id": "9znwY8X56WZfgUH27biZUKfi493wVCLJT8c5fc6G5o2C",
        "sender": "3PJbKNtRUr5HgwoZvSaWjAVbDWKpyetqYES",
        "senderPublicKey": "67JC7CAy46JmdTARj6Z6KxWMyRRZLdkuQbSFQJZm34XU",
        "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
        "assetPair": {
          "amountAsset": "AFKQxw7A5WrzW2LMDoSoJhcSQ2rLGsepZbWMLsKKBQ2K",
          "priceAsset": "474jTeYx2r2Va35794tCScAXWJG9hU2HcgxzMowaZUnu"
        },
        "orderType": "sell",
        "amount": 167410000,
        "price": 220000,
        "timestamp": 1548739880836,
        "expiration": 1551245480836,
        "matcherFee": 300000,
        "signature": "uSj6KYx8H5hun4CzRzL4F3iCrekDseLnX5A4EYsokaPtRQF2WvVQENfRX6DpT4pjWrM2oQmghZ9ecb5j4EYXkuq",
        "proofs": [
          "uSj6KYx8H5hun4CzRzL4F3iCrekDseLnX5A4EYsokaPtRQF2WvVQENfRX6DpT4pjWrM2oQmghZ9ecb5j4EYXkuq"
        ]
      },
      "amount": 167410000,
      "price": 220000,
      "buyMatcherFee": 300000,
      "sellMatcherFee": 300000
    }
`
	var tx ExchangeWithProofs
	err := json.Unmarshal([]byte(js), &tx)
	assert.NoError(t, err)
	assert.Equal(t, ExchangeTransaction, tx.Type)
	assert.Equal(t, 2, int(tx.Version))
	bo, ok := tx.Order1.(*OrderV1)
	assert.True(t, ok)
	assert.NotNil(t, bo)
	so, ok := tx.Order2.(*OrderV1)
	assert.True(t, ok)
	assert.NotNil(t, so)
}

func TestExchangeWithProofsFromJSON3(t *testing.T) {
	var js = `
{
  "type": 7,
  "id": "GR7ZDZFU2K7R9zM1qNqJEaC1vgA7hFbD3qFxvsSB9U84",
  "sender": "3PJaDyprvekvPXPuAtxrapacuDJopgJRaU3",
  "senderPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
  "fee": 300000,
  "timestamp": 1559218968473,
  "proofs": [
    "4HdcL9Ppgbf4kKECBvRx28ieSRMtgaFeF97kxSwmB72fLb3FLApkn4KQcKFE4F4pz5UwFcYBP6PB5RqXSrbKLhQM"
  ],
  "version": 2,
  "order1": {
    "version": 1,
    "id": "Du7mcUrKveCyBchxfR8RULZK6Ad21AtfWQcR8uqo3WZq",
    "sender": "3PCdWLg27GMKprpwKcHqcWS2UwXWwQNRwag",
    "senderPublicKey": "6HfBybJc7E4wJYZgWNpDJf9RnZRDvS4WLbcx7FtYBCbN",
    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair": {
      "amountAsset": "9JKjU6U2Ho71U7VWHvr14RB7iLpx2qYBtyUZqLpv6pVB",
      "priceAsset": null
    },
    "orderType": "buy",
    "amount": 139538564044,
    "price": 105,
    "timestamp": 1559218968424,
    "expiration": 1559219033424,
    "matcherFee": 300000,
    "signature": "SrzSabfBaGFyw1Ex6S7X4BH6mtujgwVxBMKNwcPb2wsyzTrkAzipybjAZcyoBdkEhBoUooUAUPGmHqFcffcTaVG",
    "proofs": [
      "SrzSabfBaGFyw1Ex6S7X4BH6mtujgwVxBMKNwcPb2wsyzTrkAzipybjAZcyoBdkEhBoUooUAUPGmHqFcffcTaVG"
    ]
  },
  "order2": {
    "version": 1,
    "id": "8KyKHCgGPYrwco9QNGaNwCbVZgSBvjz8JNW24VxVr5Vb",
    "sender": "3PCdWLg27GMKprpwKcHqcWS2UwXWwQNRwag",
    "senderPublicKey": "6HfBybJc7E4wJYZgWNpDJf9RnZRDvS4WLbcx7FtYBCbN",
    "matcherPublicKey": "7kPFrHDiGw1rCm7LPszuECwWYL3dMf6iMifLRDJQZMzy",
    "assetPair": {
      "amountAsset": "9JKjU6U2Ho71U7VWHvr14RB7iLpx2qYBtyUZqLpv6pVB",
      "priceAsset": null
    },
    "orderType": "sell",
    "amount": 139538564044000,
    "price": 105,
    "timestamp": 1559218958940,
    "expiration": 1559219023940,
    "matcherFee": 300000,
    "signature": "3TSrKc3EnZtnULQKDGBW6fMQqqPFZoRzy4fC7n637dHXhHhs9K61mTwAkmXnq8M5sTV4Y7eG7fq1YFUCJVEWVLjC",
    "proofs": [
      "3TSrKc3EnZtnULQKDGBW6fMQqqPFZoRzy4fC7n637dHXhHhs9K61mTwAkmXnq8M5sTV4Y7eG7fq1YFUCJVEWVLjC"
    ]
  },
  "amount": 139538095239,
  "price": 105,
  "buyMatcherFee": 299998,
  "sellMatcherFee": 299,
  "height": 1549429
}
`
	var tx ExchangeWithProofs
	err := json.Unmarshal([]byte(js), &tx)
	require.NoError(t, err)
	assert.Equal(t, ExchangeTransaction, tx.Type)
	assert.Equal(t, 2, int(tx.Version))
	bo, ok := tx.Order1.(*OrderV1)
	assert.True(t, ok)
	assert.NotNil(t, bo)
	so, ok := tx.Order2.(*OrderV1)
	assert.True(t, ok)
	assert.NotNil(t, so)

	b, err := tx.MarshalBinary(TestNetScheme)
	require.NoError(t, err)
	assert.Equal(t, uint8(0xcb), b[6])
}

func mustFromBase58(t *testing.T, base58String string) []byte {
	data, err := base58.Decode(base58String)
	require.NoError(t, err)
	return data
}

func TestExchangeWithProofsWithEthereumOrdersRoundTrip(t *testing.T) {
	matcherSecretKey, err := crypto.NewSecretKeyFromBase58("FZ97ouxTGpNnmyyfSBxgC2FGHTpvo7mM7LWoMut6gEYx")
	require.NoError(t, err)
	matcherPublicKey, err := crypto.NewPublicKeyFromBase58("9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ")
	require.NoError(t, err)
	require.Equal(t, matcherPublicKey, crypto.GeneratePublicKey(matcherSecretKey))

	tests := []struct {
		scheme                     Scheme
		jsonExchange               string
		order1ProtobufSignedBase58 string
		order2ProtobufSignedBase58 string
		signedTxBytesBase58        string
		txBytesBase58              string
		txIDBase58                 string
	}{
		{ // order1 is an orderV3, order2 is an ethereumOrderV4
			scheme: TestNetScheme,
			jsonExchange: `
				{
				  "type": 7,
				  "id": "3FDtgyStkFCw89G6C6QJDJ3eqobpbaUfXCFb3qqg9qdQ",
				  "fee": 1000000,
				  "feeAssetId": null,
				  "timestamp": 100,
				  "version": 3,
				  "chainId": 84,
				  "sender": "3MtGzgmNa5fMjGCcPi5nqMTdtZkfojyWHL9",
				  "senderPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				  "proofs": [
					"4QSeB7bVUxvdn2jNdwVpYxKk935CQMS2kv2s21a98W6JibqJNq3cgfkjs3tU2t6NNnGWpiWHFiX2ddfyyVwBtbgK"
				  ],
				  "order1": {
					"version": 3,
					"id": "75YqwVQbiQmLMQBE61W1aLcsaAUnWbzM5Udh9Z4mXUBf",
					"sender": "3MtGzgmNa5fMjGCcPi5nqMTdtZkfojyWHL9",
					"senderPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
					"matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
					"assetPair": {
					  "amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset": null
					},
					"orderType": "buy",
					"amount": 1,
					"price": 100,
					"timestamp": 1,
					"expiration": 123,
					"matcherFee": 100000,
					"signature": "2FekVM3s2CUf79uaG92MqdzyCzr9dhrG4jtXzbAQNxW3B9LGWwtFmMdgHCVuKWqhdAgUfV6PTsZwDxFKrejeT4vu",
					"proofs": [
					  "2FekVM3s2CUf79uaG92MqdzyCzr9dhrG4jtXzbAQNxW3B9LGWwtFmMdgHCVuKWqhdAgUfV6PTsZwDxFKrejeT4vu"
					],
					"matcherFeeAssetId": null
				  },
				  "order2": {
					"version": 4,
					"id": "6tXL591oH3mnwgFcbxqQnqHBF1oQ1Cc6hdLuBU6FB6UG",
					"sender": "3Mvrr424JENHdP4wrSFyNWBVEuQTHBDxMVi",
					"senderPublicKey": "4nZcsfxa3mtAg8D2iR8J139CTVm7Y2aTEd3B8J6p45tX6v8sjCT9JGAWnHGa8ZxenQyaSAVu3FPsry1RnXucpcqE",
					"matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
					"assetPair": {
					  "amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset": null
					},
					"orderType": "sell",
					"amount": 1,
					"price": 100,
					"timestamp": 1,
					"expiration": 123,
					"matcherFee": 100000,
					"signature": "",
					"proofs": [],
					"matcherFeeAssetId": null,
					"eip712Signature": "0x6c4385dd5f6f1200b4d0630c9076104f34c801c16a211e505facfd743ba242db4429b966ffa8d2a9aff9037dafda78cfc8f7c5ef1c94493f5954bc7ebdb649281b",
					"priceMode": null
				  },
				  "amount": 1,
				  "price": 100,
				  "buyMatcherFee": 1,
				  "sellMatcherFee": 1
				}`,
			order1ProtobufSignedBase58: "ffTiDB9neoubfGj1UBSFq9H8QcGsWY8HJWpZRSz4fDDbmikNs8aLitAoPhKtQMZUJwHiVFsebz26YwX7TvtnUtmj1aXTGNmzvtG9TuuS7EPCZxZeJw2nKhe4pre1hAFRA1NV2bwp9a34LYPtu9FT91G2zkrZc2J3wKQLEWZxhX7Bah11YaWoo4RGUMWfrPfA8BpC4nbM5X9YxfPxmfQPffvjdXoDWpv864DnGUNDP6v5DrhtE3jrTtsjkigJ8g25",
			order2ProtobufSignedBase58: "AuUJrZEtZPM7wXFkZwmKEqmXzkUNPpK3t1Gz7jtjtWxgT5NcxcMjHnUNpRJVhsPBBkn7ChYvrLp9mR4WsdLa1YBxuSBgkSVAAAUNnuAhrfoHPQRNWkbPNvaACYPJAKrpiAJP1bE2XmcPL9HKwgDQfiwrgjHFbT5wyMdwm8fUPrzji614z41Tzix6UVCr778DZtu3WVHVna1pQa1wTYnuce",
			signedTxBytesBase58:        "215qG5Yy98oCnP24ZfwtRXNagQTThioD4KoKuRmAKpKd2NAzyucVqNcuasoHdVwVmvu92UAPPqhZvDzn1T7ViuwzTEvAsbHxkzxGBoaBSFgyRpia6czPd3tdd9Uk8KZ4snqPEPjYXoQoqqPqM1JynnAhHLzSaZK1dV85RnUys9DLJ4gSLhf9o8joW8wJKfHucR3WaEBazzAFJXesUTA9naeZhTrRfK6pBDcvGtXLbjpEubFxxMn2D2veB6j7uNSkFj6JKPUNnSWD8hAFpYHkzkwJz15oVqrGYKA9RfQW5zBv1rE7GeL3gEDYrHsfzHrHg76fw4Lgi8hrxLmn3MDXJabqAYa3HUi49vuhNy2CsHuQQWtEE82FKG1GU1HtXPaM9ntswQYsjnFfshzJwvAKRng9aK3oMx2vnrjDrkFn5yCitPxg2itxnMrp5zs7S28MA8xyrwL9onK1JpKa6QZe3umLb1vSVkUK4mrkBh9yvooe7TuRG1qkgZkg15GmpXPcQw3bJTv9YbuPPyWjvds5vYjWYRnN2u77Yn28fkxqJeKn4jc7AoeJo8dHnbsCYg5aXMfjVaiqoanW7MT4NkhbidmnzVtc4AWoLpK8kYrBBSUuo8SFe2V4fjw5FcrmcZ6dvP5FvRqNTtQcf",
			txBytesBase58:              "JujQ26EJkTpemn82adxMW595LN3Qw1qTCyfyna8v1MAiHQBFfLWG6yuTUZwMBPX6mFCHzqiqYjVbGiGFaiJrUniUdzbNfHVgPoyA1rX2XuMa9WzRhPHAFq1JtHpjEVA2WSaFsYH91qAW861JYh5RevgfPJduZNSib2r3jFnbEk4Nj9PDmzGDt91scuEC7kwYcV15Q2dTdAW9UEoVYSNXCz6Qx1cAqZAhYSzaqU1CzDZ4LkRFLJgwwitDzMzp6JAcZDeqckvFnTjtkUztDm5cj5suQP8EybpQdeDpCMFny6RpcmJ3di5xFAWWTWRqo9ZjkDwyRaZmaH5BLgKtrMc8T4F6oewTfiajfuDYYbEro2JpV44bdFDqkgUHGhFPGu5UqZSia8hoqB3jpGL739PJVDCLPpuY1zoaj9ueoTpRjNKryiBeXAktFvurCL4JsKjF431tyM6u2RnmEE6LAdsvLDSKj16fJYQowe31ZrY78k3G3PYGddKp8PQ8XwF14FtmQVtvm58fFxskZEJWYfgooQ1Pzk2opCWt69LSZP3BNT6Wgi",
			txIDBase58:                 "3FDtgyStkFCw89G6C6QJDJ3eqobpbaUfXCFb3qqg9qdQ",
		},
		{ // order1 is an ethereumOrderV4, order2 is an ethereumOrderV4
			scheme: TestNetScheme,
			jsonExchange: `
				{
				  "type": 7,
				  "id": "7htGhFdnyuuSvkqpjPYpPhHqVaH8Q6ww3naZXve6Jk9y",
				  "fee": 1000000,
				  "feeAssetId": null,
				  "timestamp": 100,
				  "version": 3,
				  "chainId": 84,
				  "sender": "3MtGzgmNa5fMjGCcPi5nqMTdtZkfojyWHL9",
				  "senderPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
				  "proofs": [
					"5yhe5LmUZMeUZh2tSUvhADkUMoLBCUgPDttPBAiyFGvoKgaNL19W2S1wjRXoBw4PrB2zyRmJso3FcDycWxrYnZ8R"
				  ],
				  "order1": {
					"version": 4,
					"id": "4d8z9G58EmFggv62L8famqEQxXpvGZiGZyn8PDDVoZF9",
					"sender": "3Myw4GzqGpELqVgVMi6zqzoWssJS6Y44rhx",
					"senderPublicKey": "4TKgCQejYe8zVkXTyS3Q2P8Yvaij4RYz67ig6XkPcBu4PEawuB3Rwhy5Lg1vh9y9eaQr6PHUB58BwCAZ6SfRtSg5",
					"matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
					"assetPair": {
					  "amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset": null
					},
					"orderType": "buy",
					"amount": 1,
					"price": 100,
					"timestamp": 1,
					"expiration": 123,
					"matcherFee": 100000,
					"signature": "",
					"proofs": [],
					"matcherFeeAssetId": null,
					"eip712Signature": "0x0a897d382e4e4a066e1d98e5c3c1051864a557c488571ff71e036c0f5a2c7204274cb293cd4aa7ad40f8c2f650e1a2770ecca6aa14a1da883388fa3b5b9fa8b71c",
					"priceMode": null
				  },
				  "order2": {
					"version": 4,
					"id": "6tXL591oH3mnwgFcbxqQnqHBF1oQ1Cc6hdLuBU6FB6UG",
					"sender": "3Mvrr424JENHdP4wrSFyNWBVEuQTHBDxMVi",
					"senderPublicKey": "4nZcsfxa3mtAg8D2iR8J139CTVm7Y2aTEd3B8J6p45tX6v8sjCT9JGAWnHGa8ZxenQyaSAVu3FPsry1RnXucpcqE",
					"matcherPublicKey": "9BUoYQYq7K38mkk61q8aMH9kD9fKSVL1Fib7FbH6nUkQ",
					"assetPair": {
					  "amountAsset": "5fQPsn8hoaVddFG26cWQ5QFdqxWtUPNaZ9zH2E6LYzFn",
					  "priceAsset": null
					},
					"orderType": "sell",
					"amount": 1,
					"price": 100,
					"timestamp": 1,
					"expiration": 123,
					"matcherFee": 100000,
					"signature": "",
					"proofs": [],
					"matcherFeeAssetId": null,
					"eip712Signature": "0x6c4385dd5f6f1200b4d0630c9076104f34c801c16a211e505facfd743ba242db4429b966ffa8d2a9aff9037dafda78cfc8f7c5ef1c94493f5954bc7ebdb649281b",
					"priceMode": null
				  },
				  "amount": 1,
				  "price": 100,
				  "buyMatcherFee": 1,
				  "sellMatcherFee": 1
				}`,
			order1ProtobufSignedBase58: "WVJLeSzmAK1DT2ftcjx5uigpm5YC8uhAY3t63Xb2jp3oppNX2pUHTctYmgT3bq5BdNEL2KzYgjBmCf6ssHzVmcXCXPKLeJoDSUhY3bfJYfqoB4pDcVdYNGM9ni5ST5bFgwJtJFvhMSzYpWpcLNGYAMqdXMyXVmvJQwSZUQo8yptTET2AmTpLw9VEF1V86aPnKQp1ot6n6XC5SXiKCUK",
			order2ProtobufSignedBase58: "AuUJrZEtZPM7wXFkZwmKEqmXzkUNPpK3t1Gz7jtjtWxgT5NcxcMjHnUNpRJVhsPBBkn7ChYvrLp9mR4WsdLa1YBxuSBgkSVAAAUNnuAhrfoHPQRNWkbPNvaACYPJAKrpiAJP1bE2XmcPL9HKwgDQfiwrgjHFbT5wyMdwm8fUPrzji614z41Tzix6UVCr778DZtu3WVHVna1pQa1wTYnuce",
			signedTxBytesBase58:        "nrJERcPdjjdCaxcdD8B8Ejz1u8Rofy6hH2nGSJome1ZGHG4KXxjzdpgiX2tdatq7w82ubWe5JY8aaBf98SEXQDmJXgjBDp71QpaKkjmEmxGiqmDdwhrK6wkG1BjktTmFfd2TQriNB8n37iZFzcmC8QU7jyK9AQzMtYAjUzTucZ7dD6ELKTdHwrejVRCtXTQm6R47jsXm3ouFAc5wNhwhiapDYRm4AezyFA2siP2yxB2fxyXapQkq6QnxiSatd9N66KoUiKQ3jJPHEnbTkAU8v4ELajxok7QwggVwWkEtZ3z3Q4QZTFKdPAPBfrwSoEWZEgwJXG3b4CHjpwVvHZntzeexK1sA5nC91tRNDNurkkN3AvpPkKVJAwJEYV3nURHEdLCiRQXVJ9X419se6Qtjc8ruXX2oSJXdkFYMMUCJMmprLDQWqqEpFKei9xxTdPiFxv6TFyfpGfxDJkufQ7QSpmarmrmqvLNMyPKwYbSsy5eRQQUx3mNekpXueDfKofadauTL8QBnhj47jPAasLqE4UKMtcN1EWSmtnVbZzuZqvy9FXYxjaKijqmB2pDmszXrECi12LwJ5pQdmr3YywpY8KydTW293DP",
			txBytesBase58:              "EfDZepTtTR6GPsU2qT23ktdiP4uNpm1isevYKiBWWnT66nEAVvkeo8iYWLwgf2EzBHL6HdnQdssyNraLpVrnKq9N784ZkavvwwkM7U7F5WowNSxV5g4hno38LEbyBrqzjN5zByKzCtwkuQYNmx6SvkR7k4J4L2Npo8RrnYJmQ9yGgX3aAS32maihBaDfFqvmToncnrndMjMSq389jcFcbPtt8NLcibQZ8qypDYSgcQppqywfnYzk54EUwh7R1vMSC1PHEVZ2qBzWwoEShFJMCVK5xAvwcAjJSMw42FtzfnsokpWKrPpB8H3jCj8YNXo8fmLtUrEdduHD9g8ipkRb6w2yZs89gQKVq7KECiU9QRmRNGdjwLE6irnRpKooX8sHtZG3aSaxVohgiuWzEERkGdX1GJrbVb41Qn7ZuCRPY6xAjC5Sa86Q59yeRfqbnTo2sEdmH8PAQAqvPsePL3SKxQ4jiCpzbcsUwX2Lo3ZAWzH6o5paK33g1RhyorCeVQHhL",
			txIDBase58:                 "7htGhFdnyuuSvkqpjPYpPhHqVaH8Q6ww3naZXve6Jk9y",
		},
	}
	for _, tc := range tests {
		var (
			expectedTxID          = mustFromBase58(t, tc.txIDBase58)
			expectedTxBytes       = mustFromBase58(t, tc.txBytesBase58)
			expectedSignedTxBytes = mustFromBase58(t, tc.signedTxBytesBase58)
			expectedOrder1Bytes   = mustFromBase58(t, tc.order1ProtobufSignedBase58)
			expectedOrder2Bytes   = mustFromBase58(t, tc.order2ProtobufSignedBase58)
		)
		var (
			unmarshalOrder = func(orderBytes []byte) Order {
				var (
					pbConverter        ProtobufConverter
					unmarshaledPBOrder pb.Order
				)
				require.NoError(t, protobuf.Unmarshal(orderBytes, &unmarshaledPBOrder))
				unmarshaledOrder := pbConverter.extractOrder(&unmarshaledPBOrder)
				require.NoError(t, pbConverter.err)
				return unmarshaledOrder
			}
			checkOrderIDs = func(expected, actual Order) {
				expectedID, err := expected.GetID()
				require.NoError(t, err)
				actualID, err := actual.GetID()
				require.NoError(t, err)
				require.Equal(t, expectedID, actualID)
			}
		)
		// check unmarshal tx from json
		fromJsonTx := ExchangeWithProofs{}
		err := json.Unmarshal([]byte(tc.jsonExchange), &fromJsonTx)
		require.NoError(t, err)

		// check orders IDs
		unmarshaledOrder1 := unmarshalOrder(expectedOrder1Bytes)
		checkOrderIDs(unmarshaledOrder1, fromJsonTx.Order1)
		require.Equal(t, unmarshaledOrder1, fromJsonTx.Order1)

		unmarshaledOrder2 := unmarshalOrder(expectedOrder2Bytes)
		checkOrderIDs(unmarshaledOrder2, fromJsonTx.Order2)
		require.Equal(t, unmarshaledOrder2, fromJsonTx.Order2)

		// check orders protobuf
		actualOrder1Bytes, err := fromJsonTx.Order1.ToProtobufSigned(tc.scheme).MarshalVTStrict()
		require.NoError(t, err)
		require.Equal(t, expectedOrder1Bytes, actualOrder1Bytes)

		actualOrder2Bytes, err := fromJsonTx.Order2.ToProtobufSigned(tc.scheme).MarshalVTStrict()
		require.NoError(t, err)
		require.Equal(t, expectedOrder2Bytes, actualOrder2Bytes)

		// check marshal signed tx to protobuf
		actualSignedTxBytes, err := fromJsonTx.MarshalSignedToProtobuf(tc.scheme)
		require.NoError(t, err)
		require.Equal(t, expectedSignedTxBytes, actualSignedTxBytes)

		// check tx id
		actualTxID, err := fromJsonTx.GetID(tc.scheme)
		require.NoError(t, err)
		require.Equal(t, expectedTxID, actualTxID)

		// check unmarshal signed from protobuf
		fromPbTx := ExchangeWithProofs{}
		err = fromPbTx.UnmarshalSignedFromProtobuf(expectedSignedTxBytes)
		require.NoError(t, err)
		require.Equal(t, fromJsonTx, fromPbTx)

		// check marshal unsigned to protobuf
		actualTxBytes, err := fromPbTx.MarshalToProtobuf(tc.scheme)
		require.NoError(t, err)
		require.Equal(t, expectedTxBytes, actualTxBytes)

		// signed tx verify
		valid, err := fromPbTx.Verify(tc.scheme, matcherPublicKey)
		require.NoError(t, err)
		require.True(t, valid)

		// check sign
		unsignedTx := ExchangeWithProofs{}
		err = unsignedTx.UnmarshalFromProtobuf(actualTxBytes)
		require.NoError(t, err)
		err = unsignedTx.Sign(tc.scheme, matcherSecretKey)
		require.NoError(t, err)
		valid, err = unsignedTx.Verify(tc.scheme, matcherPublicKey)
		require.NoError(t, err)
		require.True(t, valid)
	}
}

func TestLeaseWithSigValidations(t *testing.T) {
	spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	require.NoError(t, err)
	tests := []struct {
		recipient string
		senderPk  crypto.PublicKey
		amount    uint64
		fee       uint64
		err       string
	}{

		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, 0, 100000, "amount should be positive"},
		{"alias:T:nickname", spk, math.MaxInt64 + 1, 100000, "amount is too big"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, 100000, 0, "fee should be positive"},
		{"alias:T:nickname", spk, 100000, math.MaxInt64 + 1, "fee is too big"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, math.MaxInt64, math.MaxInt64, "sum of amount and fee overflows JVM long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", spk, 100000, 100000, "invalid recipient: invalid scheme 'W', expected 'T'"},
		{"3MxW8ZFCQUQDg39G3Zbr3LjCLPnWNtjodve", spk, 100000, 100000, "invalid recipient: invalid WavesAddress checksum"},
		{"alias:T:прозвище", spk, 100000, 100000, "invalid recipient: Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz"},
		{MustAddressFromPublicKey(TestNetScheme, spk).String(), spk, 100000, 100000, "addr \"3Mz43r3B8uBX49oY1RCTPa9wScoToJM3mT3\" trying to lease money to itself"},
	}
	for _, tc := range tests {
		rcp, err := recipientFromString(tc.recipient)
		require.NoError(t, err)
		tx := NewUnsignedLeaseWithSig(tc.senderPk, rcp, tc.amount, tc.fee, 0)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseWithSigFromMainNet(t *testing.T) {
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
		spk, err := crypto.NewPublicKeyFromBase58(tc.pk)
		require.NoError(t, err)
		id, err := crypto.NewDigestFromBase58(tc.id)
		require.NoError(t, err)
		sig, err := crypto.NewSignatureFromBase58(tc.sig)
		require.NoError(t, err)
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		tx := NewUnsignedLeaseWithSig(spk, rcp, tc.amount, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestLeaseWithSigProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithSig(pk, rcp, tc.amount, tc.fee, ts)
		err = tx.GenerateID(MainNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx LeaseWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx LeaseWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(MainNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestLeaseWithSigBinarySize(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithSig(pk, rcp, tc.amount, tc.fee, ts)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestLeaseWithSigBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithSig(pk, rcp, tc.amount, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseWithSig
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, *tx.Recipient.Address(), *atx.Recipient.Address())
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseWithSig
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
				assert.ElementsMatch(t, pk, atx.SenderPK)
				assert.ElementsMatch(t, addr, *atx.Recipient.Address())
				assert.Equal(t, tc.amount, atx.Amount)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestLeaseWithSigToJSON(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithSig(pk, rcp, tc.amount, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":8,\"version\":1,\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":8,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestLeaseWithProofsValidations(t *testing.T) {
	spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	require.NoError(t, err)
	tests := []struct {
		recipient string
		senderPK  crypto.PublicKey
		amount    uint64
		fee       uint64
		err       string
	}{
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, 0, 100000, "amount should be positive"},
		{"alias:T:nickname", spk, math.MaxInt64 + 1, 100000, "amount is too big"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, 100000, 0, "fee should be positive"},
		{"alias:T:nickname", spk, 100000, math.MaxInt64 + 1, "fee is too big"},
		{"3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn", spk, math.MaxInt64, math.MaxInt64, "sum of amount and fee overflows JVM long"},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", spk, 100000, 100000, "invalid recipient: invalid scheme 'W', expected 'T'"},
		{"3MxW8ZFCQUQDg39G3Zbr3LjCLPnWNtjodve", spk, 100000, 100000, "invalid recipient: invalid WavesAddress checksum"},
		{"alias:T:прозвище", spk, 100000, 100000, "invalid recipient: Alias should contain only following characters: -.0123456789@_abcdefghijklmnopqrstuvwxyz"},
		{MustAddressFromPublicKey(TestNetScheme, spk).String(), spk, 100000, 100000, "addr \"3Mz43r3B8uBX49oY1RCTPa9wScoToJM3mT3\" trying to lease money to itself"},
	}
	for _, tc := range tests {
		rcp, err := recipientFromString(tc.recipient)
		require.NoError(t, err)
		tx := NewUnsignedLeaseWithProofs(2, tc.senderPK, rcp, tc.amount, tc.fee, 0)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseWithProofsFromMainNet(t *testing.T) {
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
		spk, err := crypto.NewPublicKeyFromBase58(tc.pk)
		require.NoError(t, err)
		id, err := crypto.NewDigestFromBase58(tc.id)
		require.NoError(t, err)
		sig, err := crypto.NewSignatureFromBase58(tc.sig)
		require.NoError(t, err)
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		tx := NewUnsignedLeaseWithProofs(2, spk, rcp, tc.amount, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestLeaseWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithProofs(2, pk, rcp, tc.amount, tc.fee, ts)
		err = tx.GenerateID(MainNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx LeaseWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx LeaseWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(MainNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestLeaseWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithProofs(2, pk, rcp, tc.amount, tc.fee, ts)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestLeaseWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithProofs(2, pk, rcp, tc.amount, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseWithProofs
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, *tx.Recipient.Address(), *atx.Recipient.Address())
				assert.Equal(t, tx.Amount, atx.Amount)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
				assert.ElementsMatch(t, pk, atx.SenderPK)
				assert.ElementsMatch(t, addr, *atx.Recipient.Address())
				assert.Equal(t, tc.amount, atx.Amount)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestLeaseWithProofsToJSON(t *testing.T) {
	tests := []struct {
		recipient string
		amount    uint64
		fee       uint64
	}{
		{"3P23fi1qfVw6RVDn4CH2a5nNouEtWNQ4THs", 1234567890, 1234567890},
		{"3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ", 9876543210, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		addr, err := NewAddressFromString(tc.recipient)
		require.NoError(t, err)
		rcp := NewRecipientFromAddress(addr)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseWithProofs(2, pk, rcp, tc.amount, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":8,\"version\":2,\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":8,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"recipient\":\"%s\",\"amount\":%d,\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.recipient, tc.amount, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestLeaseCancelWithSigValidations(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
		err   string
	}{
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", 0, "fee should be positive"},
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", math.MaxInt64 + 1, "fee is too big"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		tx := NewUnsignedLeaseCancelWithSig(spk, l, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseCancelWithSigFromMainNet(t *testing.T) {
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
		tx := NewUnsignedLeaseCancelWithSig(spk, l, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestLeaseCancelWithSigProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithSig(pk, l, tc.fee, ts)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestLeaseCancelWithSigBinarySize(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithSig(pk, l, tc.fee, ts)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestLeaseCancelWithSigBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithSig(pk, l, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithSig
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, tx.LeaseID, atx.LeaseID)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithSig
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, *tx.Signature, *atx.Signature)
				assert.ElementsMatch(t, pk, atx.SenderPK)
				assert.ElementsMatch(t, l, atx.LeaseID)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestLeaseCancelWithSigToJSON(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithSig(pk, l, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":9,\"version\":1,\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.lease, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":9,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), tc.lease, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestLeaseCancelWithProofsValidations(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
		err   string
	}{
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", 0, "fee should be positive"},
		{"58iiBQ9uonkDpgr3NiAYgec3K9f5KvhHEwLfZTX2k7y3", math.MaxInt64 + 10, "fee is too big"},
		//TODO: add blockchain scheme validation
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		tx := NewUnsignedLeaseCancelWithProofs(2, spk, l, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestLeaseCancelWithProofsFromMainNet(t *testing.T) {
	const scheme = MainNetScheme

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
		tx := NewUnsignedLeaseCancelWithProofs(2, spk, l, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestLeaseCancelWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithProofs(2, pk, l, tc.fee, ts)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestLeaseCancelWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		const scheme = 'T'
		l := crypto.MustDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithProofs(2, pk, l, tc.fee, ts)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(scheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestLeaseCancelWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithProofs(2, pk, l, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithProofs
			if err := atx.bodyUnmarshalBinary(bb, TestNetScheme); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, tx.LeaseID, atx.LeaseID)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx LeaseCancelWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs[0], atx.Proofs.Proofs[0])
				assert.ElementsMatch(t, pk, atx.SenderPK)
				assert.ElementsMatch(t, l, atx.LeaseID)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestLeaseCancelWithProofsToJSON(t *testing.T) {
	tests := []struct {
		lease string
		fee   uint64
	}{
		{"6jkoA3xzdFuowHsV3An1tc7sexsJ9kenSHeKJVCU5qNM", 1234567890},
		{"Bc83cgvtmBbhpWHgqWPvoPMFVJCsUicocAaDReyyuqSX", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		l, _ := crypto.NewDigestFromBase58(tc.lease)
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedLeaseCancelWithProofs(2, pk, l, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":9,\"version\":2,\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.lease, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":9,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"leaseId\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.lease, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestCreateAliasWithSigValidations(t *testing.T) {
	tests := []struct {
		alias string
		fee   uint64
		err   string
	}{
		{"something", 0, "fee should be positive"},
		{"something", math.MaxInt64 + 1, "fee is too big"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a := NewAlias('W', tc.alias)
		tx := NewUnsignedCreateAliasWithSig(spk, *a, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestCreateAliasWithSigFromMainNet(t *testing.T) {
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
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithSig(spk, *a, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			if h, err := tx.id(); assert.NoError(t, err) {
				assert.Equal(t, id, *h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestCreateAliasWithSigProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithSig(pk, *a, tc.fee, ts)
		err = tx.GenerateID(tc.scheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithSig
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithSig
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.scheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestCreateAliasWithSigBinarySize(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithSig(pk, *a, tc.fee, ts)
		err = tx.Sign(tc.scheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.scheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestCreateAliasWithSigBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithSig(pk, *a, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithSig
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.ElementsMatch(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Alias, atx.Alias)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithSig
			if err := atx.UnmarshalBinary(b, tc.scheme); assert.NoError(t, err) {
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

func TestCreateAliasWithSigJSON(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "alice", 1234567890},
		{'T', "peter", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)

	checkUnmarshal := func(t *testing.T, js []byte, expected *CreateAliasWithSig) {
		tx := new(CreateAliasWithSig)
		err := json.Unmarshal(js, tx)
		require.NoError(t, err)
		require.Nil(t, tx.ID)
		require.Zero(t, tx.Alias.Scheme)

		err = expected.GenerateID(expected.Alias.Scheme)
		require.NoError(t, err)
		err = tx.GenerateID(expected.Alias.Scheme)
		require.NoError(t, err)
		require.Equal(t, expected.Alias.Scheme, tx.Alias.Scheme)

		assert.Equal(t, expected, tx)
	}

	for _, tc := range tests {
		a := NewAlias(tc.scheme, tc.alias)
		ts := uint64(time.Now().UnixNano() / 1000000)
		testName := func(base string) string { return fmt.Sprintf("%s-%c-%s", base, tc.scheme, tc.alias) }

		t.Run(testName("Unsigned"), func(t *testing.T) {
			tx := NewUnsignedCreateAliasWithSig(pk, *a, tc.fee, ts)
			ej := fmt.Sprintf("{\"type\":10,\"version\":1,\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(pk[:]), a.Alias, tc.fee, ts)
			j, err := json.Marshal(tx)
			require.NoError(t, err)
			assert.JSONEq(t, ej, string(j))

			checkUnmarshal(t, []byte(ej), tx)
		})

		t.Run(testName("Signed"), func(t *testing.T) {
			tx := NewUnsignedCreateAliasWithSig(pk, *a, tc.fee, ts)
			err := tx.Sign(tc.scheme, sk)
			require.NoError(t, err)
			esj := fmt.Sprintf("{\"type\":10,\"version\":1,\"id\":\"%s\",\"signature\":\"%s\",\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(tx.ID[:]), base58.Encode(tx.Signature[:]), base58.Encode(pk[:]), a.Alias, tc.fee, ts)
			sj, err := json.Marshal(tx)
			require.NoError(t, err)
			assert.JSONEq(t, esj, string(sj))

			checkUnmarshal(t, []byte(esj), tx)
		})
	}
}

func TestCreateAliasWithProofsValidations(t *testing.T) {
	tests := []struct {
		alias string
		fee   uint64
		err   string
	}{
		{"something", 0, "fee should be positive"},
		{"something", math.MaxInt64 + 10, "fee is too big"},
		{"so", 12345, "Alias 'so' length should be between 4 and 30"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a := NewAlias(TestNetScheme, tc.alias)
		tx := NewUnsignedCreateAliasWithProofs(2, spk, *a, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestCreateAliasWithProofsFromMainNet(t *testing.T) {
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
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithProofs(2, spk, *a, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			if h, err := tx.id(); assert.NoError(t, err) {
				assert.Equal(t, id, *h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestCreateAliasWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithProofs(2, pk, *a, tc.fee, ts)
		err = tx.GenerateID(tc.scheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.scheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestCreateAliasWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithProofs(2, pk, *a, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithProofs
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Alias, atx.Alias)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		err = tx.Sign(tc.scheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.scheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestCreateAliasWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "somealias", 1234567890},
		{'T', "testnetalias", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a := NewAlias(tc.scheme, tc.alias)
		tx := NewUnsignedCreateAliasWithProofs(2, pk, *a, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithProofs
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.Alias, atx.Alias)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(tc.scheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.scheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.scheme); assert.NoError(t, err) {
			var atx CreateAliasWithProofs
			if err := atx.UnmarshalBinary(b, tc.scheme); assert.NoError(t, err) {
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

func TestCreateAliasWithProofsJSON(t *testing.T) {
	tests := []struct {
		scheme byte
		alias  string
		fee    uint64
	}{
		{'W', "alice", 1234567890},
		{'T', "peter", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)

	checkUnmarshal := func(t *testing.T, js []byte, expected *CreateAliasWithProofs) {
		tx := new(CreateAliasWithProofs)
		err := json.Unmarshal(js, tx)
		require.NoError(t, err)
		require.Nil(t, tx.ID)
		require.Zero(t, tx.Alias.Scheme)

		err = expected.GenerateID(expected.Alias.Scheme)
		require.NoError(t, err)
		err = tx.GenerateID(expected.Alias.Scheme)
		require.NoError(t, err)
		require.Equal(t, expected.Alias.Scheme, tx.Alias.Scheme)

		assert.Equal(t, expected, tx)
	}

	for _, tc := range tests {
		a := NewAlias(tc.scheme, tc.alias)
		ts := uint64(time.Now().UnixNano() / 1000000)
		testName := func(base string) string { return fmt.Sprintf("%s-%c-%s", base, tc.scheme, tc.alias) }

		t.Run(testName("Unsigned"), func(t *testing.T) {
			tx := NewUnsignedCreateAliasWithProofs(2, pk, *a, tc.fee, ts)
			ej := fmt.Sprintf("{\"type\":10,\"version\":2,\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(pk[:]), a.Alias, tc.fee, ts)
			j, err := json.Marshal(tx)
			require.NoError(t, err)
			assert.JSONEq(t, ej, string(j))

			checkUnmarshal(t, []byte(ej), tx)
		})

		t.Run(testName("Signed"), func(t *testing.T) {
			tx := NewUnsignedCreateAliasWithProofs(2, pk, *a, tc.fee, ts)
			err := tx.Sign(tc.scheme, sk)
			require.NoError(t, err)
			esj := fmt.Sprintf("{\"type\":10,\"version\":2,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"alias\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
				base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), a.Alias, tc.fee, ts)
			sj, err := json.Marshal(tx)
			require.NoError(t, err)
			assert.JSONEq(t, esj, string(sj))

			checkUnmarshal(t, []byte(esj), tx)
		})
	}
}

func TestMassTransferWithProofsValidations(t *testing.T) {
	repeat := func(t MassTransferEntry, n int) []MassTransferEntry {
		r := make([]MassTransferEntry, n)
		for i := 0; i < n; i++ {
			r = append(r, t)
		}
		return r
	}
	addr := MustAddressFromString("3MxW8ZFCQUQDg7xagmGQQcwbQDmNGLfZVAn")
	addrWithOtherScheme := MustAddressFromString("3PB1Y84BGdEXE4HKaExyJ5cHP36nEw8ovaE")
	tests := []struct {
		asset      string
		transfers  []MassTransferEntry
		fee        uint64
		attachment string
		err        string
	}{
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 100}}, 0, "", "fee should be positive"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 100}}, math.MaxInt64 + 10, "", "fee is too big"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", repeat(MassTransferEntry{NewRecipientFromAddress(addr), 100}, 101), 10, "", "Number of transfers 202 is greater than 100"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), math.MaxInt64 + 1}, {NewRecipientFromAddress(addr), 20}}, 20, "", "at least one of the transfers amount is bigger than JVM long"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), math.MaxInt64 / 2}, {NewRecipientFromAddress(addr), math.MaxInt64 / 2}}, 1000, "", "sum of amounts of transfers and transaction fee is bigger than JVM long"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 100}, {NewRecipientFromAddress(addrWithOtherScheme), 100}}, 1000, "", "invalid recipient: invalid scheme 'W', expected 'T'"},
		{"HmNSH2g1SWYHzuX1G4VCjL63TFs7PXDjsTAHzrAhSRCK", []MassTransferEntry{{NewRecipientFromAddress(addr), 10}, {NewRecipientFromAddress(addr), 20}}, 30, strings.Repeat("blah-blah", 30), "attachment too long"},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := NewOptionalAssetFromString(tc.asset)
		att := []byte(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, spk, *a, tc.transfers, tc.fee, 0, att)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestMassTransferWithProofsFromMainNet(t *testing.T) {
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
		att := []byte(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, spk, *a, transfers, tc.fee, tc.timestamp, att)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestMassTransferWithProofsProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		att := []byte(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, pk, *a, tc.transfers, tc.fee, ts, att)
		err := tx.GenerateID(MainNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx MassTransferWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(MainNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(MainNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(MainNetScheme); assert.NoError(t, err) {
			var atx MassTransferWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(MainNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestMassTransferWithProofsBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		att := []byte(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, pk, *a, tc.transfers, tc.fee, ts, att)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestMassTransferWithProofsBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		att := Attachment(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, pk, *a, tc.transfers, tc.fee, ts, att)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx MassTransferWithProofs
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
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx MassTransferWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
				assert.Equal(t, len(tx.Proofs.Proofs), len(atx.Proofs.Proofs))
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
				assert.Equal(t, pk, atx.SenderPK)
				assert.ElementsMatch(t, tc.transfers, atx.Transfers)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
				att := Attachment(tc.attachment)
				assert.Equal(t, att, atx.Attachment)
			}
		}
	}
}

func TestMassTransferWithProofsToJSON(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := NewOptionalAssetFromString(tc.asset)
		att := []byte(tc.attachment)
		tx := NewUnsignedMassTransferWithProofs(1, pk, *a, tc.transfers, tc.fee, ts, att)
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
			ej := fmt.Sprintf(`{"type":11,"version":1,"senderPublicKey":"%s","assetId":%s,"transfers":[%s],"timestamp":%d,"fee":%d%s}`, base58.Encode(pk[:]), tc.expectedAsset, sb.String(), ts, tc.fee, tc.expectedAttachment)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":11,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":%s,\"transfers\":[%s],\"timestamp\":%d,\"fee\":%d%s}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.expectedAsset, sb.String(), ts, tc.fee, tc.expectedAttachment)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestDataWithProofsValidations(t *testing.T) {
	repeat := func(e *BinaryDataEntry, n int) DataEntries {
		r := DataEntries{}
		for i := 0; i < n; i++ {
			ue := &BinaryDataEntry{}
			ue.Key = fmt.Sprintf("%s-%d", e.Key, i)
			ue.Value = e.Value
			r = append(r, ue)
		}
		return r
	}
	ieOk := &IntegerDataEntry{Key: "integer-entry", Value: 12345}
	beOk := &BooleanDataEntry{Key: "boolean-entry", Value: true}
	seOk := &StringDataEntry{Key: "string-entry", Value: "some string value, should be ok"}
	deOk := &BinaryDataEntry{Key: "binary-entry", Value: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}}
	tests := []struct {
		entries DataEntries
		fee     uint64
		err     string
	}{
		{[]DataEntry{ieOk}, 0, "fee should be positive"},
		{[]DataEntry{beOk}, math.MaxInt64 + 10, "fee is too big"},
		{[]DataEntry{seOk, seOk, deOk}, 12345, "duplicate keys?"},
		{repeat(deOk, 120), 12345, "number of DataWithProofs entries is bigger than 100"},
	}
	for i, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		tx := NewUnsignedDataWithProofs(1, spk, tc.fee, 0)
		tx.Entries = tc.entries
		_, err = tx.Validate(TestNetScheme)
		assert.Error(t, err, "#%d", i) //, tc.err, fmt.Sprintf("expected: %s", tc.err))
		//assert.EqualError(t, err, tc.err, fmt.Sprintf("expected: %s", tc.err))
		assert.Regexp(t, tc.err, err.Error(), "#%d", i)
	}
}

func TestDataWithProofsDeleteValidation(t *testing.T) {
	spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
	require.NoError(t, err)
	de := &DeleteDataEntry{Key: "key"}
	tx1 := NewUnsignedDataWithProofs(1, spk, MinFee, 67890)
	tx1.Entries = DataEntries{de}
	_, err = tx1.Validate(TestNetScheme)
	msg := "delete supported only for protobuf transaction"
	assert.EqualError(t, err, msg, fmt.Sprintf("expected: %s", msg))

	tx2 := NewUnsignedDataWithProofs(2, spk, MinFee, 67890)
	tx2.Entries = DataEntries{de}
	_, err = tx2.Validate(TestNetScheme)
	assert.NoError(t, err)
}

func TestDataWithProofsSizeLimit(t *testing.T) {
	repeat := func(e *BinaryDataEntry, n int) DataEntries {
		r := DataEntries{}
		for i := 0; i < n; i++ {
			ue := &BinaryDataEntry{}
			ue.Key = fmt.Sprintf("%s-%d", e.Key, i)
			ue.Value = e.Value
			r = append(r, ue)
		}
		return r
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	deBig := &BinaryDataEntry{Key: "binary-entry", Value: make([]byte, 2000)}
	tests := []struct {
		entries DataEntries
		fee     uint64
		err     string
	}{
		{repeat(deBig, 100), 12345, "total size of DataWithProofs transaction is bigger than 184320 bytes"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		tx := NewUnsignedDataWithProofs(1, spk, tc.fee, 0)
		tx.Entries = tc.entries
		err = tx.Sign(TestNetScheme, sk)
		require.NoError(t, err)

		// Test custom format.
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		var tx2 DataWithProofs
		err = tx2.UnmarshalBinary(txBytes, TestNetScheme)
		assert.EqualError(t, err, tc.err, fmt.Sprintf("expected: %s", tc.err))

		// Test Protobuf.
		txBytes, err = tx.MarshalSignedToProtobuf(TestNetScheme)
		assert.NoError(t, err)
		err = tx2.UnmarshalSignedFromProtobuf(txBytes)
		assert.EqualError(t, err, tc.err, fmt.Sprintf("expected: %s", tc.err))

		// Test protobuf payload size
		payload := tx.ProtoPayload()
		marshaledPayload, err := protobuf.MarshalOptions{Deterministic: true}.Marshal(payload)
		assert.NoError(t, err)
		marshaledPayloadSize := tx.ProtoPayloadSize()
		assert.NoError(t, err)
		assert.Equal(t, len(marshaledPayload), marshaledPayloadSize)
	}
}

func TestDataWithProofsFromMainNet(t *testing.T) {
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
		tx := NewUnsignedDataWithProofs(1, spk, tc.fee, tc.timestamp)
		for i, k := range tc.keys {
			e := &StringDataEntry{k, tc.values[i]}
			err := tx.AppendEntry(e)
			require.NoError(t, err)
		}
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestDataWithProofsProtobufRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedDataWithProofs(1, pk, tc.fee, ts)
		for i, k := range tc.keys {
			var e DataEntry
			switch DataValueType(tc.types[i]) {
			case DataInteger:
				v, _ := strconv.Atoi(tc.values[i])
				e = &IntegerDataEntry{k, int64(v)}
			case DataBoolean:
				v, _ := strconv.ParseBool(tc.values[i])
				e = &BooleanDataEntry{k, v}
			case DataBinary:
				v, _ := base58.Decode(tc.values[i])
				e = &BinaryDataEntry{k, v}
			case DataString:
				e = &StringDataEntry{k, tc.values[i]}
			}
			err := tx.AppendEntry(e)
			assert.NoError(t, err)
		}
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx DataWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx DataWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestDataWithProofsBinarySize(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedDataWithProofs(1, pk, tc.fee, ts)
		for i, k := range tc.keys {
			var e DataEntry
			switch DataValueType(tc.types[i]) {
			case DataInteger:
				v, _ := strconv.Atoi(tc.values[i])
				e = &IntegerDataEntry{k, int64(v)}
			case DataBoolean:
				v, _ := strconv.ParseBool(tc.values[i])
				e = &BooleanDataEntry{k, v}
			case DataBinary:
				v, _ := base58.Decode(tc.values[i])
				e = &BinaryDataEntry{k, v}
			case DataString:
				e = &StringDataEntry{k, tc.values[i]}
			}
			err := tx.AppendEntry(e)
			assert.NoError(t, err)
		}
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestDataWithProofsBinaryRoundTrip(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedDataWithProofs(1, pk, tc.fee, ts)
		for i, k := range tc.keys {
			var e DataEntry
			switch DataValueType(tc.types[i]) {
			case DataInteger:
				v, _ := strconv.Atoi(tc.values[i])
				e = &IntegerDataEntry{k, int64(v)}
			case DataBoolean:
				v, _ := strconv.ParseBool(tc.values[i])
				e = &BooleanDataEntry{k, v}
			case DataBinary:
				v, _ := base58.Decode(tc.values[i])
				e = &BinaryDataEntry{k, v}
			case DataString:
				e = &StringDataEntry{k, tc.values[i]}
			}
			err := tx.AppendEntry(e)
			assert.NoError(t, err)
		}
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx DataWithProofs
			if err := atx.bodyUnmarshalBinary(bb); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, tx.Entries, atx.Entries)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx DataWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
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

func TestDataWithProofsToJSON(t *testing.T) {
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
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		tx := NewUnsignedDataWithProofs(1, pk, tc.fee, ts)
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
			switch DataValueType(tc.types[i]) {
			case DataInteger:
				v, _ := strconv.Atoi(tc.values[i])
				e = &IntegerDataEntry{k, int64(v)}
				sb.WriteRune('"')
				sb.WriteString("integer")
				sb.WriteRune('"')
				sb.WriteString(",\"value\":")
				sb.WriteString(tc.values[i])
			case DataBoolean:
				v, _ := strconv.ParseBool(tc.values[i])
				e = &BooleanDataEntry{k, v}
				sb.WriteRune('"')
				sb.WriteString("boolean")
				sb.WriteRune('"')
				sb.WriteString(",\"value\":")
				sb.WriteString(tc.values[i])
			case DataBinary:
				v, _ := base58.Decode(tc.values[i])
				e = &BinaryDataEntry{k, v}
				sb.WriteRune('"')
				sb.WriteString("binary")
				sb.WriteRune('"')
				sb.WriteString(",\"value\":")
				sb.WriteRune('"')
				sb.WriteString("base64:")
				sb.WriteString(base64.StdEncoding.EncodeToString(v))
				sb.WriteRune('"')
			case DataString:
				e = &StringDataEntry{k, tc.values[i]}
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
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":12,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"data\":[%s],\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), sb.String(), tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestDataWithProofsFromJSON(t *testing.T) {
	var js = `
{
	"type": 12,
	"id": "74r5tx5BuhnYP3YQ5jo3RwDcH89gaDEdEc9bjUKPiSa8",
	"sender": "3P9QNCmT3Q44zRYXBwKN3azBta9azGqrscm",
	"senderPublicKey": "J48ygzZLEdcR2GbWjjy9eFJDs57Poz6ZajGEyygSMV26",
	"fee": 10000000,
	"timestamp": 1548739929686,
	"proofs": [
		"2bB5ysJXYBumJiLMbQ3o2gqxES5gydQ4bni3aWGiXwBaBDvLEpDNFLgKuj6UnhtS4LUS9R6yVoSVFoT94RCBvzo",
		"3PPgSrFX52vYbAtTVrz8nHjmcv3LQhYd3mP"
	],
	"version": 1,
	"data": [
		{
			"key": "lastPayment",
			"type": "string",
			"value": "GenCSKr8UFrZXrbQ8oAG7W8PDgUY7pe7hrbRmJACuMkS"
		},
		{
			"key": "heightToGetMoney",
			"type": "integer",
			"value": 1372374
		},
		{
			"key": "GenCSKr8UFrZXrbQ8oAG7W8PDgUY7pe7hrbRmJACuMkS",
			"type": "string",
			"value": "used"
		}
	]
}
`
	var tx DataWithProofs
	err := json.Unmarshal([]byte(js), &tx)
	assert.NoError(t, err)
	assert.Equal(t, DataTransaction, tx.Type)
	assert.Equal(t, 1, int(tx.Version))
}

func TestSetScriptWithProofsValidations(t *testing.T) {
	tests := []struct {
		script string
		fee    uint64
		err    string
	}{
		{"something", 0, "fee should be positive"},
		{"something", math.MaxInt64 + 123, "fee is too big"},
		//TODO: add blockchain scheme validation
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		s, _ := base58.Decode(tc.script)
		tx := NewUnsignedSetScriptWithProofs(1, spk, s, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestSetScriptWithProofsFromMainNet(t *testing.T) {
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
		tx := NewUnsignedSetScriptWithProofs(1, spk, s, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestSetScriptWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetScriptWithProofs(1, pk, s, tc.fee, ts)
		err = tx.GenerateID(tc.chainID)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx SetScriptWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx SetScriptWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.chainID)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestSetScriptWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetScriptWithProofs(1, pk, s, tc.fee, ts)
		err = tx.Sign(tc.chainID, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.chainID)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestSetScriptWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetScriptWithProofs(1, pk, s, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx SetScriptWithProofs
			if err := atx.bodyUnmarshalBinary(bb, tc.chainID); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.ElementsMatch(t, tx.Script, atx.Script)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx SetScriptWithProofs
			if err := atx.UnmarshalBinary(b, tc.chainID); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
				assert.Equal(t, pk, atx.SenderPK)
				assert.Equal(t, tc.script, base64.StdEncoding.EncodeToString(atx.Script))
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestSetScriptWithProofsToJSON(t *testing.T) {
	tests := []struct {
		chainID byte
		script  string
		fee     uint64
	}{
		{'W', "base64:AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "base64:", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		s, err := base64.StdEncoding.DecodeString(tc.script[7:])
		require.NoError(t, err)
		tx := NewUnsignedSetScriptWithProofs(1, pk, s, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":13,\"version\":1,\"senderPublicKey\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.script, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":13,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.script, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestSponsorshipWithProofsValidations(t *testing.T) {
	tests := []struct {
		minAssetFee uint64
		fee         uint64
		err         string
	}{
		{0, 0, "fee should be positive"},
		{0, math.MaxInt64 + 1, "fee is too big"},
		{math.MaxInt64 + 1, 12345, "min asset fee is too big"},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		a, err := crypto.NewDigestFromBase58("8Nwjd2tcQWff3S9WAhBa7vLRNpNnigWqrTbahvyfMVrU")
		require.NoError(t, err)
		tx := NewUnsignedSponsorshipWithProofs(1, spk, a, tc.minAssetFee, tc.fee, 0)
		_, err = tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestSponsorshipWithProofsFromMainNet(t *testing.T) {
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
		tx := NewUnsignedSponsorshipWithProofs(1, spk, a, tc.assetFee, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(MainNetScheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestSponsorshipWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		tx := NewUnsignedSponsorshipWithProofs(1, pk, a, tc.assetFee, tc.fee, ts)
		err = tx.GenerateID(TestNetScheme)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx SponsorshipWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(TestNetScheme); assert.NoError(t, err) {
			var atx SponsorshipWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(TestNetScheme)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestSponsorshipWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		tx := NewUnsignedSponsorshipWithProofs(1, pk, a, tc.assetFee, tc.fee, ts)
		err = tx.Sign(TestNetScheme, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(TestNetScheme)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestSponsorshipWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		tx := NewUnsignedSponsorshipWithProofs(1, pk, a, tc.assetFee, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx SponsorshipWithProofs
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
		if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(TestNetScheme, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(TestNetScheme); assert.NoError(t, err) {
			var atx SponsorshipWithProofs
			if err := atx.UnmarshalBinary(b, TestNetScheme); assert.NoError(t, err) {
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

func TestSponsorshipWithProofsToJSON(t *testing.T) {
	tests := []struct {
		asset    string
		assetFee uint64
		fee      uint64
	}{
		{"9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 100, 1234567890},
		{"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 0, 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		tx := NewUnsignedSponsorshipWithProofs(1, pk, a, tc.assetFee, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":14,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"minSponsoredAssetFee\":%d,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.asset, tc.assetFee, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(TestNetScheme, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":14,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"minSponsoredAssetFee\":%d,\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.assetFee, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestSetAssetScriptWithProofsValidations(t *testing.T) {
	tests := []struct {
		script string
		fee    uint64
		err    string
	}{
		{"something", 0, "fee should be positive"},
		{"something", math.MaxInt64 + 1, "fee is too big"},
		//TODO: add tests on blockchain scheme validation and script type
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		a, _ := crypto.NewDigestFromBase58("J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR")
		s, _ := base58.Decode(tc.script)
		tx := NewUnsignedSetAssetScriptWithProofs(1, spk, a, s, tc.fee, 0)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestSetAssetScriptWithProofsFromMainNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		scheme    byte
		asset     string
		script    string
		fee       uint64
		timestamp uint64
	}{
		{"AwQYJRHZNd9bvF7C13uwnPiLQfTzvDFJe7DTUXxzrGQS", "nzYhVKmRmd7BiFDDfrFVnY6Yo98xDGsKrBLWentF7ibe4P9cGWg4RtomHum2NEMBhuyZb5yjThcW7vsCLg7F8NQ", "FwYSpmVDbWQ2BA5NCBZ9z5GSjY39PSyfNZzBayDiMA88", 'W', "7qJUQFxniMQx45wk12UdZwknEW9cDgvfoHuAvwDNVjYv", "AQa3b8tH", 100000000, 1547201038106},
		{"AwQYJRHZNd9bvF7C13uwnPiLQfTzvDFJe7DTUXxzrGQS", "23pjWpcgJfBBV8QD6kvL3ZUaiaFFwHTiiPBzd5XLamETgv4gts4Dg8jrH4BqUjEsaRRFrHem8J34SJ3mau8yaqfX", "FvXkKs9x4UndmFSu3RZxBR2huULJPbUfoWRQ2tJvQh4F", 'W', "7qJUQFxniMQx45wk12UdZwknEW9cDgvfoHuAvwDNVjYv", "AQa3b8tH", 100000000, 1547201122606},
		{"AwQYJRHZNd9bvF7C13uwnPiLQfTzvDFJe7DTUXxzrGQS", "2K4BKgLfv47hPxDbhsFDwABiDrwQuYeN3sDN3UaaDUXs6eo37VjUJwsJJFNbySgZmzBNKTuB3msqp3xXLgdbNo2p", "9dPNuoK9hLowH5KPsVRNpMrQUfop6EBg4Dpzdgdh1WL7", 'W', "7qJUQFxniMQx45wk12UdZwknEW9cDgvfoHuAvwDNVjYv", "AQQAAAAQd2hpdGVMaXN0QWNjb3VudAkBAAAAB0FkZHJlc3MAAAABAQAAABoBVy3YfBi6sTVYY0bkC3rJRVVPBcXqnEJojwQAAAAHJG1hdGNoMAUAAAACdHgDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMEAAAACXJlY2lwaWVudAkAAlgAAAABCAkABCQAAAABCAUAAAACdHgAAAAJcmVjaXBpZW50AAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAABnNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAACXJlY2lwaWVudAcDCQAAAQAAAAIFAAAAByRtYXRjaDACAAAAE0V4Y2hhbmdlVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAA9zZWxsT3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAJc2VsbE9yZGVyAAAABnNlbmRlcgAAAAVieXRlcwQAAAAOYnV5T3JkZXJTZW5kZXIJAAJYAAAAAQgICAUAAAACdHgAAAAIYnV5T3JkZXIAAAAGc2VuZGVyAAAABWJ5dGVzAwkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAAD3NlbGxPcmRlclNlbmRlcgkBAAAAB2V4dHJhY3QAAAABCQAEGwAAAAIFAAAAEHdoaXRlTGlzdEFjY291bnQFAAAADmJ1eU9yZGVyU2VuZGVyBwMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAXTWFzc1RyYW5zZmVyVHJhbnNhY3Rpb24EAAAAAnR4BQAAAAckbWF0Y2gwBAAAAAZzZW5kZXIJAAJYAAAAAQgIBQAAAAJ0eAAAAAZzZW5kZXIAAAAFYnl0ZXMJAQAAAAdleHRyYWN0AAAAAQkABBsAAAACBQAAABB3aGl0ZUxpc3RBY2NvdW50BQAAAAZzZW5kZXIGWSftFg==", 100000000, 1547201663356},
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58(tc.pk)
		id, _ := crypto.NewDigestFromBase58(tc.id)
		sig, _ := crypto.NewSignatureFromBase58(tc.sig)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		tx := NewUnsignedSetAssetScriptWithProofs(1, spk, a, s, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestSetAssetScriptWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		chainID byte
		asset   string
		script  string
		fee     uint64
	}{
		{'W', "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetAssetScriptWithProofs(1, pk, a, s, tc.fee, ts)
		err = tx.GenerateID(tc.chainID)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx SetAssetScriptWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx SetAssetScriptWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.chainID)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestSetAssetScriptWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		chainID byte
		asset   string
		script  string
		fee     uint64
	}{
		{'W', "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetAssetScriptWithProofs(1, pk, a, s, tc.fee, ts)
		err = tx.Sign(tc.chainID, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.chainID)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestSetAssetScriptWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chainID byte
		asset   string
		script  string
		fee     uint64
	}{
		{'W', "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", "AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", "", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, _ := crypto.NewDigestFromBase58(tc.asset)
		s, _ := base64.StdEncoding.DecodeString(tc.script)
		tx := NewUnsignedSetAssetScriptWithProofs(1, pk, a, s, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx SetAssetScriptWithProofs
			if err := atx.bodyUnmarshalBinary(bb, tc.chainID); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.AssetID, atx.AssetID)
				assert.ElementsMatch(t, tx.Script, atx.Script)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx SetAssetScriptWithProofs
			if err := atx.UnmarshalBinary(b, tc.chainID); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
				assert.Equal(t, pk, atx.SenderPK)
				assert.Equal(t, a, atx.AssetID)
				assert.Equal(t, tc.script, base64.StdEncoding.EncodeToString(atx.Script))
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestSetAssetScriptWithProofsToJSON(t *testing.T) {
	tests := []struct {
		chainID byte
		asset   string
		script  string
		fee     uint64
	}{
		{'W', "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", "base64:AQQAAAAEaW5hbAIAAAAESW5hbAQAAAAFZWxlbmECAAAAB0xlbnVza2EEAAAABGxvdmUCAAAAC0luYWxMZW51c2thCQAAAAAAAAIJAAEsAAAAAgUAAAAEaW5hbAUAAAAFZWxlbmEFAAAABGxvdmV4ZFt5", 1234567890},
		{'T', "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", "base64:", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := crypto.NewDigestFromBase58(tc.asset)
		require.NoError(t, err)
		s, err := base64.StdEncoding.DecodeString(tc.script[7:])
		require.NoError(t, err)
		tx := NewUnsignedSetAssetScriptWithProofs(1, pk, a, s, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":15,\"version\":1,\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.asset, tc.script, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":15,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"assetId\":\"%s\",\"script\":\"%s\",\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.asset, tc.script, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestInvokeScriptWithProofsValidations(t *testing.T) {
	repeat := func(arg StringArgument, n int) Arguments {
		r := make([]Argument, n)
		for i := 0; i < n; i++ {
			r = append(r, arg)
		}
		return r
	}
	a1, err := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)
	a2, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)

	tests := []struct {
		sps     ScriptPayments
		name    string
		args    Arguments
		fee     uint64
		err     string
		version byte
	}{
		{ScriptPayments{}, "foo", Arguments{IntegerArgument{Value: 1234567890}}, 0, "fee should be positive", 1},
		{ScriptPayments{{12345, *a1}}, "foo", Arguments{StringArgument{Value: "some value should be ok"}}, math.MaxInt64 + 1, "fee is too big", 1},
		{ScriptPayments{{12345, *a1}}, strings.Repeat("foo", 100), Arguments{}, 13245, "function name is too big", 1},
		{ScriptPayments{{12345, *a1}}, "foo", repeat(StringArgument{Value: "some value should be ok"}, 100), 13245, "too many arguments", 1},
		{ScriptPayments{{0, *a1}}, "foo", Arguments{StringArgument{Value: "some value should be ok"}}, 1234, "at least one payment has a non-positive amount", 1},
		{ScriptPayments{{math.MaxInt64 + 123, *a1}}, "foo", Arguments{StringArgument{Value: "some value should be ok"}}, 12345, "at least one payment has a too big amount", 1},
		{ScriptPayments{}, "foo", Arguments{IntegerArgument{Value: 1234567890}}, 1, "unexpected version 128 for InvokeScriptWithProofs", 128},
		//TODO: add test on arguments evaluation
	}
	for _, tc := range tests {
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		ad, _ := NewAddressFromString("3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu")
		fc := FunctionCall{Name: tc.name, Arguments: tc.args}
		tx := NewUnsignedInvokeScriptWithProofs(tc.version, spk, NewRecipientFromAddress(ad), fc, tc.sps, *a2, tc.fee, 12345)
		_, err := tx.Validate(TestNetScheme)
		assert.EqualError(t, err, tc.err)
	}
}

func TestInvokeScriptWithProofsFromTestNet(t *testing.T) {
	tests := []struct {
		pk        string
		sig       string
		id        string
		scheme    byte
		dapp      string
		fc        string
		payments  string
		fee       uint64
		feeAsset  string
		timestamp uint64
	}{
		{"DKGFPozLrsiR8NM4NJzqQaBYC8NyGYjuw2hDYicQVjco", "5eV3WVt96gQDLdk1KFGa23AZhriZcv6Xy2Zvo33Pqgh7SRLumPeToFUoZxSAry5LUbzdndAdpP5Cy1c6wvVXNUyX", "4xgCjQBvxirqts1nBhTqci5C1TLEJs974aAJgmjPF2bz", 'T', "3MpVFGJWgiGyh5LmE1nxNLsjjtSL3Bgh9NV", "{\"function\":\"LiquidExhange\",\"args\":[]}", "[{\"amount\":35000000000,\"assetId\":null}]", 1000000, "WAVES", 1559832028452},
		{"AkMK8AjATGN89Ziiy8U4p7vquPU8qjeVedMpcWgXLovD", "4RnTATsewLTeawZJWRpwiFLGVPP81NHgDxFdXfiJXVtdRrbANAyTHGuKNiLZkRwW5vRMLnrDneGPrNfqxcAcSh59", "FdQwSRUwBzNbLdKu158aELLdPSfmKQ7B3fYd2en3CEyg", 'T', "3ND68eBy9NyJPeq4eRqi42c45hoDAzzRjSm", "{\"function\":\"bet\",\"args\":[{\"type\":\"string\",\"value\":\"1\"}]}", "[{\"amount\":200500000,\"assetId\":null}]", 500000, "WAVES", 1560151320652},
		{"5gwomYMjH2taJyfXjGj2LPVwcVH7Wd86Xh2T2iTENqa4", "4HGuUbp2y2rh5NA5aXKi9xEAr5AYyb89PknKpZo8LdwbHoA2QBS4CYiGh6cJqZZ7DTsx2b1jEciQpPhcxzrLbDLk", "88G72uSqvSMh4qvAFGiE78eJsFXxiAjXZRJJhMGrHh6", 'T', "3N6t7q6vrBQT7CUPjFeDieKvm7be6pBcFLx", "{\"function\":\"test\",\"args\":[{\"type\":\"binary\",\"value\":\"3x91Br\"},{\"type\":\"boolean\",\"value\":true},{\"type\":\"integer\",\"value\":100500},{\"type\":\"string\",\"value\":\"some text\"}]}", "[{\"amount\":10,\"assetId\":\"H3jGkTWJr8Sr4KFay3QqNqmA3zEtgxYAx1ojitNaPkWy\"}]", 9, "H3jGkTWJr8Sr4KFay3QqNqmA3zEtgxYAx1ojitNaPkWy", 1560153418889},
		{"5gwomYMjH2taJyfXjGj2LPVwcVH7Wd86Xh2T2iTENqa4", "4S7aQoeGrtbRsDGRg2yT8eG9ybRsv1GzeYC4zn14LzmwdYXRviqK9XtPjBWANk1VqQEJMQvMRuj19baSGmb65qeu", "DtsKYcGQbobcXboE1FzfzxqBuQJmdwzQS7YTFXYDBj6S", 'T', "3N6t7q6vrBQT7CUPjFeDieKvm7be6pBcFLx", "null", "[]", 900000, "WAVES", 1560153629460},
		{"5gwomYMjH2taJyfXjGj2LPVwcVH7Wd86Xh2T2iTENqa4", "2PrBFHS41nwd2Gq4BL6VuBxX855UbSN8dgHQYcRWLRczUpoTvojzwT6yXfvEH8QEPiREdxgGUfCLiseiPoKPTUY5", "FYo3AFvGX4L5CSBNXSB6Tc8J4vmuY7PTj4FYge5BKmfM", 'T', "alias:T:inv-test", "null", "[]", 900000, "WAVES", 1560161103375},
	}
	for _, tc := range tests {
		spk, err := crypto.NewPublicKeyFromBase58(tc.pk)
		require.NoError(t, err)
		id, err := crypto.NewDigestFromBase58(tc.id)
		require.NoError(t, err)
		sig, err := crypto.NewSignatureFromBase58(tc.sig)
		require.NoError(t, err)
		rcp, err := NewRecipientFromString(tc.dapp)
		require.NoError(t, err)
		fa, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		fc := FunctionCall{}
		err = json.Unmarshal([]byte(tc.fc), &fc)
		require.NoError(t, err)
		fjs, err := json.Marshal(fc)
		require.NoError(t, err)
		assert.Equal(t, tc.fc, string(fjs))
		payments := ScriptPayments{}
		err = json.Unmarshal([]byte(tc.payments), &payments)
		require.NoError(t, err)
		pjs, err := json.Marshal(payments)
		require.NoError(t, err)
		assert.Equal(t, tc.payments, string(pjs))
		tx := NewUnsignedInvokeScriptWithProofs(1, spk, rcp, fc, payments, *fa, tc.fee, tc.timestamp)
		if b, err := tx.BodyMarshalBinary(tc.scheme); assert.NoError(t, err) {
			if h, err := crypto.FastHash(b); assert.NoError(t, err) {
				assert.Equal(t, id, h)
			}
			assert.True(t, crypto.Verify(spk, sig, b))
		}
	}
}

func TestInvokeScriptWithProofsProtobufRoundTrip(t *testing.T) {
	tests := []struct {
		chainID  byte
		address  string
		fc       string
		payments string
		feeAsset string
		fee      uint64
	}{
		{'W', "3PLANf4MgtNN5v6k4NNnyx2m4zKJiw1tF9v", "{\"function\":\"foo\",\"args\":[{\"type\":\"integer\",\"value\":12345}]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 1234567890},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"bar\",\"args\":[{\"type\":\"boolean\",\"value\":true}]}", "[{\"amount\":67890,\"assetId\":null}]", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 9876543210},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"foobar1\",\"args\":[]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "WAVES", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		ad, err := NewAddressFromString(tc.address)
		require.NoError(t, err)
		fc := FunctionCall{}
		err = json.Unmarshal([]byte(tc.fc), &fc)
		require.NoError(t, err)
		sps := ScriptPayments{}
		err = json.Unmarshal([]byte(tc.payments), &sps)
		require.NoError(t, err)
		tx := NewUnsignedInvokeScriptWithProofs(1, pk, NewRecipientFromAddress(ad), fc, sps, *a, tc.fee, ts)
		err = tx.GenerateID(tc.chainID)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx InvokeScriptWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.chainID); assert.NoError(t, err) {
			var atx InvokeScriptWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.chainID)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestInvokeScriptWithProofsBinarySize(t *testing.T) {
	tests := []struct {
		chainID  byte
		address  string
		fc       string
		payments string
		feeAsset string
		fee      uint64
	}{
		{'W', "3PLANf4MgtNN5v6k4NNnyx2m4zKJiw1tF9v", "{\"function\":\"foo\",\"args\":[{\"type\":\"integer\",\"value\":12345}]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 1234567890},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"bar\",\"args\":[{\"type\":\"boolean\",\"value\":true}]}", "[{\"amount\":67890,\"assetId\":null}]", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 9876543210},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"foobar1\",\"args\":[]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "WAVES", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		ad, err := NewAddressFromString(tc.address)
		require.NoError(t, err)
		fc := FunctionCall{}
		err = json.Unmarshal([]byte(tc.fc), &fc)
		require.NoError(t, err)
		sps := ScriptPayments{}
		err = json.Unmarshal([]byte(tc.payments), &sps)
		require.NoError(t, err)
		tx := NewUnsignedInvokeScriptWithProofs(1, pk, NewRecipientFromAddress(ad), fc, sps, *a, tc.fee, ts)
		err = tx.Sign(tc.chainID, sk)
		assert.NoError(t, err)
		txBytes, err := tx.MarshalBinary(tc.chainID)
		assert.NoError(t, err)
		assert.Equal(t, len(txBytes), tx.BinarySize())
	}
}

func TestInvokeScriptWithProofsBinaryRoundTrip(t *testing.T) {
	tests := []struct {
		chainID  byte
		address  string
		fc       string
		payments string
		feeAsset string
		fee      uint64
	}{
		{'W', "3PLANf4MgtNN5v6k4NNnyx2m4zKJiw1tF9v", "{\"function\":\"foo\",\"args\":[{\"type\":\"integer\",\"value\":12345}]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 1234567890},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"bar\",\"args\":[{\"type\":\"boolean\",\"value\":true}]}", "[{\"amount\":67890,\"assetId\":null}]", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 9876543210},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"foobar1\",\"args\":[]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "WAVES", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		ad, err := NewAddressFromString(tc.address)
		require.NoError(t, err)
		fc := FunctionCall{}
		err = json.Unmarshal([]byte(tc.fc), &fc)
		require.NoError(t, err)
		sps := ScriptPayments{}
		err = json.Unmarshal([]byte(tc.payments), &sps)
		require.NoError(t, err)
		tx := NewUnsignedInvokeScriptWithProofs(1, pk, NewRecipientFromAddress(ad), fc, sps, *a, tc.fee, ts)
		if bb, err := tx.BodyMarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx InvokeScriptWithProofs
			if err := atx.bodyUnmarshalBinary(bb, tc.chainID); assert.NoError(t, err) {
				assert.Equal(t, tx.Type, atx.Type)
				assert.Equal(t, tx.Version, atx.Version)
				assert.Equal(t, tx.SenderPK, atx.SenderPK)
				assert.Equal(t, tx.ScriptRecipient, atx.ScriptRecipient)
				assert.Equal(t, tx.FunctionCall, atx.FunctionCall)
				assert.Equal(t, tx.Payments, atx.Payments)
				assert.Equal(t, tx.FeeAsset, atx.FeeAsset)
				assert.Equal(t, tx.Fee, atx.Fee)
				assert.Equal(t, tx.Timestamp, atx.Timestamp)
			}
		}
		if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chainID, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalBinary(tc.chainID); assert.NoError(t, err) {
			var atx InvokeScriptWithProofs
			if err := atx.UnmarshalBinary(b, tc.chainID); assert.NoError(t, err) {
				assert.ElementsMatch(t, tx.Proofs.Proofs, atx.Proofs.Proofs)
				assert.Equal(t, pk, atx.SenderPK)
				assert.Equal(t, *a, atx.FeeAsset)
				assert.Equal(t, NewRecipientFromAddress(ad), atx.ScriptRecipient)
				assert.Equal(t, sps, atx.Payments)
				assert.Equal(t, fc, atx.FunctionCall)
				assert.Equal(t, tc.fee, atx.Fee)
				assert.Equal(t, ts, atx.Timestamp)
			}
		}
	}
}

func TestInvokeScriptWithProofsToJSON(t *testing.T) {
	tests := []struct {
		chainID  byte
		address  string
		fc       string
		payments string
		feeAsset string
		fee      uint64
	}{
		{'W', "3PLANf4MgtNN5v6k4NNnyx2m4zKJiw1tF9v", "{\"function\":\"foo\",\"args\":[{\"type\":\"integer\",\"value\":12345}]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 1234567890},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"bar\",\"args\":[{\"type\":\"boolean\",\"value\":true}]}", "[{\"amount\":67890,\"assetId\":null}]", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 9876543210},
		{'T', "3MrDis17gyNSusZDg8Eo1PuFnm5SQMda3gu", "{\"function\":\"foobar1\",\"args\":[]}", "[{\"amount\":12345,\"assetId\":\"BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD\"}]", "WAVES", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		ad, err := NewAddressFromString(tc.address)
		require.NoError(t, err)
		fc := FunctionCall{}
		err = json.Unmarshal([]byte(tc.fc), &fc)
		require.NoError(t, err)
		sps := ScriptPayments{}
		err = json.Unmarshal([]byte(tc.payments), &sps)
		require.NoError(t, err)
		feeAssetIDJSON := fmt.Sprintf("\"%s\"", tc.feeAsset)
		if tc.feeAsset == "WAVES" {
			feeAssetIDJSON = "null"
		}
		tx := NewUnsignedInvokeScriptWithProofs(1, pk, NewRecipientFromAddress(ad), fc, sps, *a, tc.fee, ts)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf("{\"type\":16,\"version\":1,\"senderPublicKey\":\"%s\",\"dApp\":\"%s\",\"call\":%s,\"payment\":%s,\"feeAssetId\":%s,\"fee\":%d,\"timestamp\":%d}", base58.Encode(pk[:]), tc.address, tc.fc, tc.payments, feeAssetIDJSON, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chainID, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf("{\"type\":16,\"version\":1,\"id\":\"%s\",\"proofs\":[\"%s\"],\"senderPublicKey\":\"%s\",\"dApp\":\"%s\",\"call\":%s,\"payment\":%s,\"feeAssetId\":%s,\"fee\":%d,\"timestamp\":%d}",
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), tc.address, tc.fc, tc.payments, feeAssetIDJSON, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestUpdateAssetInfoWithProofsValidations(t *testing.T) {
	a1, err := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)
	a2, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)

	tests := []struct {
		version     byte
		chain       byte
		name        string
		description string
		err         error
		fee         uint64
		feeAsset    *OptionalAsset
		valid       bool
	}{
		{1, 'W', "newAssetName", "newDescription", nil, 1, a2, true},
		{1, 'T', "newAssetName", "newDescription", nil, 1, a1, true},
		{1, 'T', "newAssetName", "newDescription", errors.New("fee should be positive"), 0, a1, false},
		{0, 'T', "newAssetName", "newDescription", errors.New("unexpected version 0 for UpdateAssetInfoWithProofs"), 1, a1, false},
		{1, 'W', "newAssetName", "newDescription", errors.New("fee is too big"), math.MaxUint64, a1, false},
		{1, 'W', "VERY_LONG_TOKEN_NAME", "This is a valid description for the token", errors.New("incorrect number of bytes in the asset's name"), 1, a1, false},
		{1, 'W', "TOKEN", strings.Repeat("x", 1010), errors.New("incorrect number of bytes in the asset's description"), 1, a1, false},
	}
	for _, tc := range tests {
		aid, err := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedUpdateAssetInfoWithProofs(tc.version, aid, spk, tc.name, tc.description, 12345, *tc.feeAsset, tc.fee)
		_, err = tx.Validate(tc.chain)
		if !tc.valid {
			assert.Equal(t, tc.err.Error(), err.Error())
		}
	}
}

func TestUpdateAssetInfoWithProofsProtobufRoundTrip(t *testing.T) {
	a1, err := NewOptionalAssetFromString("BXBUNddxTGTQc3G4qHYn5E67SBwMj18zLncUr871iuRD")
	require.NoError(t, err)
	a2, err := NewOptionalAssetFromString("WAVES")
	require.NoError(t, err)
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)

	tests := []struct {
		version     byte
		chain       byte
		name        string
		description string
		fee         uint64
		feeAsset    *OptionalAsset
	}{
		{1, 'W', "newAssetName", "newDescription", 1, a2},
		{1, 'T', "NAME", "Description...", 1, a1},
	}
	for _, tc := range tests {
		aid, err := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		spk, _ := crypto.NewPublicKeyFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		tx := NewUnsignedUpdateAssetInfoWithProofs(tc.version, aid, spk, tc.name, tc.description, 12345, *tc.feeAsset, tc.fee)
		err = tx.GenerateID(tc.chain)
		require.NoError(t, err)
		if bb, err := tx.MarshalToProtobuf(tc.chain); assert.NoError(t, err) {
			var atx UpdateAssetInfoWithProofs
			if err := atx.UnmarshalFromProtobuf(bb); assert.NoError(t, err) {
				assert.Equal(t, *tx, atx)
			}
		}
		if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
			if r, err := tx.Verify(tc.chain, pk); assert.NoError(t, err) {
				assert.True(t, r)
			}
		}
		if b, err := tx.MarshalSignedToProtobuf(tc.chain); assert.NoError(t, err) {
			var atx UpdateAssetInfoWithProofs
			if err := atx.UnmarshalSignedFromProtobuf(b); assert.NoError(t, err) {
				err = atx.GenerateID(tc.chain)
				assert.NoError(t, err)
				assert.Equal(t, *tx, atx)
			}
		}
	}
}

func TestUpdateAssetInfoWithProofsToJSON(t *testing.T) {
	tests := []struct {
		chain       byte
		name        string
		description string
		feeAsset    string
		fee         uint64
	}{
		{'W', "AssetName", "description of asset", "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", 1234567890},
		{'T', "NAME", ".....", "9yCRXrptsYKnsfFv6E226MXXjjxSzm3kXKL2oquw3HrX", 9876543210},
		{'T', "noname", "whatever", "WAVES", 9876543210},
	}
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	for _, tc := range tests {
		ts := uint64(time.Now().UnixNano() / 1000000)
		a, err := NewOptionalAssetFromString(tc.feeAsset)
		require.NoError(t, err)
		aid, err := crypto.NewDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6")
		require.NoError(t, err)
		feeAssetIDJSON := fmt.Sprintf("\"%s\"", tc.feeAsset)
		if tc.feeAsset == "WAVES" {
			feeAssetIDJSON = "null"
		}
		tx := NewUnsignedUpdateAssetInfoWithProofs(1, aid, pk, tc.name, tc.description, ts, *a, tc.fee)
		if j, err := json.Marshal(tx); assert.NoError(t, err) {
			ej := fmt.Sprintf(`{"type":17,"version":1,"senderPublicKey":"%s","assetId":"%s","name":"%s","description":"%s","feeAssetId":%s,"fee":%d,"timestamp":%d}`, base58.Encode(pk[:]), aid.String(), tc.name, tc.description, feeAssetIDJSON, tc.fee, ts)
			assert.Equal(t, ej, string(j))
			if err := tx.Sign(tc.chain, sk); assert.NoError(t, err) {
				if sj, err := json.Marshal(tx); assert.NoError(t, err) {
					esj := fmt.Sprintf(`{"type":17,"version":1,"id":"%s","proofs":["%s"],"senderPublicKey":"%s","assetId":"%s","name":"%s","description":"%s","feeAssetId":%s,"fee":%d,"timestamp":%d}`,
						base58.Encode(tx.ID[:]), base58.Encode(tx.Proofs.Proofs[0]), base58.Encode(pk[:]), aid.String(), tc.name, tc.description, feeAssetIDJSON, tc.fee, ts)
					assert.Equal(t, esj, string(sj))
				}
			}
		}
	}
}

func TestUpdateAssetInfoWithProofsFromJSON(t *testing.T) {
	var js = `
		{"type":17,"version":1,"id":"2Eikmt2YrNyXvDkHueHkzvxs7y9RMWyhrUcMpzg4G6dJ","proofs":["NRb2MtFkrJN9kZrCH6smgfMWXiuFiTaPn81ceWosWvL3ByX3YtqosK4iP3yeu2BBFVh1YT2yq3sAwjky5jEiXSo"],"chainId":87,"senderPublicKey":"3qTkgmBYFjdSEtib9C4b3yHiEexyJ59A5ZVjSvXsg569","assetId":"BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6","name":"AssetName","description":"description of asset","feeAssetId":"J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR","fee":1234567890,"timestamp":1583406542756}
	`
	spk, err := crypto.NewPublicKeyFromBase58("3qTkgmBYFjdSEtib9C4b3yHiEexyJ59A5ZVjSvXsg569")
	require.NoError(t, err)
	var tx UpdateAssetInfoWithProofs
	err = json.Unmarshal([]byte(js), &tx)
	require.NoError(t, err)
	assert.Equal(t, UpdateAssetInfoTransaction, tx.Type)
	assert.Equal(t, 1, int(tx.Version))
	assert.Equal(t, uint64(1583406542756), tx.Timestamp)
	assert.Equal(t, 1234567890, int(tx.Fee))
	assert.True(t, tx.FeeAsset.Present)
	assert.Equal(t, 1, len(tx.Proofs.Proofs))
	assert.Equal(t, spk[:], tx.SenderPK[:])
	assert.Equal(t, "AssetName", tx.Name)
	assert.Equal(t, "description of asset", tx.Description)
	assert.Equal(t, crypto.MustDigestFromBase58("BJ3Q8kNPByCWHwJ3RLn55UPzUDVgnh64EwYAU5iCj6z6"), tx.AssetID)
	assert.Equal(t, "J8shEVBrQ4BLqsuYw5j6vQGCFJGMLBxr5nu2XvUWFEAR", tx.FeeAsset.String())
}

func BenchmarkBytesToTransaction_WithReflection(b *testing.B) {
	b.ReportAllocs()
	bts := []byte{0, 4, 2, 132, 79, 148, 251, 4, 38, 180, 107, 148, 225, 225, 107, 146, 125, 26, 243, 25, 35, 202, 83, 226, 142, 64, 8, 106, 72, 250, 228, 237, 132, 90, 16, 0, 0, 0, 0, 1, 104, 225, 147, 43, 220, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 134, 160, 1, 68, 152, 220, 142, 172, 155, 208, 202, 105, 149, 210, 120, 159, 30, 146, 64, 212, 101, 147, 228, 250, 36, 56, 81, 55, 0, 3, 102, 111, 111, 1, 0, 1, 0, 64, 154, 86, 48, 50, 47, 58, 64, 254, 146, 85, 72, 252, 23, 49, 64, 40, 34, 104, 117, 225, 126, 65, 235, 225, 38, 13, 114, 120, 7, 30, 240, 209, 37, 144, 166, 15, 14, 241, 232, 101, 103, 82, 232, 163, 165, 82, 96, 52, 132, 191, 194, 160, 155, 237, 106, 43, 82, 203, 125, 122, 219, 35, 186, 8}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BytesToTransaction(bts, TestNetScheme)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBytesToTransaction_WithoutReflection(b *testing.B) {
	b.ReportAllocs()
	bts := []byte{0, 4, 2, 132, 79, 148, 251, 4, 38, 180, 107, 148, 225, 225, 107, 146, 125, 26, 243, 25, 35, 202, 83, 226, 142, 64, 8, 106, 72, 250, 228, 237, 132, 90, 16, 0, 0, 0, 0, 1, 104, 225, 147, 43, 220, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 134, 160, 1, 68, 152, 220, 142, 172, 155, 208, 202, 105, 149, 210, 120, 159, 30, 146, 64, 212, 101, 147, 228, 250, 36, 56, 81, 55, 0, 3, 102, 111, 111, 1, 0, 1, 0, 64, 154, 86, 48, 50, 47, 58, 64, 254, 146, 85, 72, 252, 23, 49, 64, 40, 34, 104, 117, 225, 126, 65, 235, 225, 38, 13, 114, 120, 7, 30, 240, 209, 37, 144, 166, 15, 14, 241, 232, 101, 103, 82, 232, 163, 165, 82, 96, 52, 132, 191, 194, 160, 155, 237, 106, 43, 82, 203, 125, 122, 219, 35, 186, 8}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := getTransaction(bts, TestNetScheme)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func getTransaction(txb []byte, scheme Scheme) (Transaction, error) {
	switch txb[0] {
	case 0:
		switch txb[1] {
		case byte(IssueTransaction):
			var tx IssueWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(TransferTransaction):
			var tx TransferWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(ReissueTransaction):
			var tx ReissueWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(BurnTransaction):
			var tx BurnWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(ExchangeTransaction):
			var tx ExchangeWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(LeaseTransaction):
			var tx LeaseWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(LeaseCancelTransaction):
			var tx LeaseCancelWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(CreateAliasTransaction):
			var tx CreateAliasWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(DataTransaction):
			var tx DataWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(SetScriptTransaction):
			var tx SetScriptWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		case byte(SponsorshipTransaction):
			var tx SponsorshipWithProofs
			err := tx.UnmarshalBinary(txb, scheme)
			return &tx, err
		default:
			return nil, errors.New("unknown transaction")
		}

	case byte(IssueTransaction):
		var tx IssueWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err

	case byte(TransferTransaction):
		var tx TransferWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(ReissueTransaction):
		var tx ReissueWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(BurnTransaction):
		var tx BurnWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(ExchangeTransaction):
		var tx ExchangeWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(LeaseTransaction):
		var tx LeaseWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(LeaseCancelTransaction):
		var tx LeaseCancelWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(CreateAliasTransaction):
		var tx CreateAliasWithSig
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	case byte(MassTransferTransaction):
		var tx MassTransferWithProofs
		err := tx.UnmarshalBinary(txb, scheme)
		return &tx, err
	}
	return nil, errors.New("unknown transaction")
}

// This function is for tests only! Could produce invalid address.
func addressFromString(s string) (WavesAddress, error) {
	ab, err := base58.Decode(s)
	if err != nil {
		return WavesAddress{}, err
	}
	a := WavesAddress{}
	copy(a[:], ab)
	return a, nil
}

// This function is for tests only! Could produce invalid recipient.
func recipientFromString(s string) (Recipient, error) {
	if strings.HasPrefix(s, AliasPrefix) {
		a, err := NewAliasFromString(s)
		if err != nil {
			return Recipient{}, err
		}
		return NewRecipientFromAlias(*a), nil
	}
	addr, err := addressFromString(s)
	if err != nil {
		return Recipient{}, err
	}
	return NewRecipientFromAddress(addr), nil
}

func TestTransactionTypeInfo_String(t *testing.T) {
	require.NotEmpty(t, TransactionTypeInfo{}.String())
}

func TestIssue_ToProtobufWithInvalidUtf8String(t *testing.T) {
	kp, _ := NewKeyPair([]byte("test"))
	// invalid utf8 string
	s := string([]byte{0xE0, 0x80, 0x80})

	is := NewUnsignedIssueWithSig(kp.Public, s, s, 100, 1, false, 1, 1)
	err := is.Sign(TestNetScheme, kp.Secret)
	require.NoError(t, err)

	bts, err := is.MarshalSignedToProtobuf(TestNetScheme)
	require.NoError(t, err)
	require.NotEmpty(t, bts)
}

func TestWavesTranactionGetSenderAndGetSenderPK(t *testing.T) {
	emptyWavesPK := crypto.PublicKey{}
	emptyWavesAddr, err := NewAddressFromPublicKey(TestNetScheme, emptyWavesPK)
	require.NoError(t, err)

	wavesPK, err := crypto.NewPublicKeyFromBase58("6uQfgnn18ixRGmhc31eoqWqZrac7jWpT2sNNXtvxQy4A")
	require.NoError(t, err)
	wavesAddr, err := NewAddressFromPublicKey(TestNetScheme, wavesPK)
	require.NoError(t, err)

	tests := []struct {
		tx              Transaction
		expectedPKBytes []byte
		expectedAddr    Address
	}{
		{&Genesis{}, emptyWavesPK.Bytes(), emptyWavesAddr},
		{&Payment{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&IssueWithProofs{Issue: Issue{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&IssueWithSig{Issue: Issue{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&TransferWithProofs{Transfer: Transfer{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&TransferWithSig{Transfer: Transfer{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&ReissueWithProofs{Reissue: Reissue{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&ReissueWithSig{Reissue: Reissue{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&BurnWithProofs{Burn: Burn{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&BurnWithSig{Burn: Burn{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&ExchangeWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&ExchangeWithSig{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&ExchangeWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&LeaseWithProofs{Lease: Lease{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&LeaseWithSig{Lease: Lease{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&LeaseCancelWithProofs{LeaseCancel: LeaseCancel{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&LeaseCancelWithSig{LeaseCancel: LeaseCancel{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&CreateAliasWithProofs{CreateAlias: CreateAlias{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&CreateAliasWithSig{CreateAlias: CreateAlias{SenderPK: wavesPK}}, wavesPK.Bytes(), wavesAddr},
		{&MassTransferWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&DataWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&SetScriptWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&SponsorshipWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&SetAssetScriptWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&InvokeScriptWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
		{&UpdateAssetInfoWithProofs{SenderPK: wavesPK}, wavesPK.Bytes(), wavesAddr},
	}
	for _, tc := range tests {
		addr, err := tc.tx.GetSender(TestNetScheme)
		require.NoError(t, err)
		require.Equal(t, tc.expectedAddr, addr)
		type getSenderPK interface {
			GetSenderPK() crypto.PublicKey
		}
		if wavesTx, ok := tc.tx.(getSenderPK); ok {
			pk := wavesTx.GetSenderPK()
			require.Equal(t, tc.expectedPKBytes, pk.Bytes())
		}
	}
}

func TestEthereumGetSenderAndFromPK(t *testing.T) {
	ethereumPK, err := NewEthereumPublicKeyFromHexString("0xc4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	require.NoError(t, err)
	ethereumAddr := ethereumPK.EthereumAddress()

	tx := &EthereumTransaction{}
	tx.threadSafeSetSenderPK(&ethereumPK)

	actualPK, err := tx.FromPK()
	require.NoError(t, err)
	require.Equal(t, &ethereumPK, actualPK)

	actualAddr, err := tx.GetSender(TestNetScheme)
	require.NoError(t, err)
	require.Equal(t, ethereumAddr, actualAddr)
}

func TestShadowedCreateAliasWithProofs_DoesNotImplementJSONMarshaler(t *testing.T) {
	// this test is necessary check for correct shadowing in CreateAliasWithProofs.MarshalJSON method
	type shadowed CreateAliasWithProofs
	var v shadowed
	_, typeImplements := interface{}(v).(json.Marshaler)
	require.False(t, typeImplements, "type must not implement Marshaler")
	_, pointerImlements := interface{}(&v).(json.Marshaler)
	require.False(t, pointerImlements, "pointer must not implement Marshaler")
}

func TestShadowedCreateAliasWithSig_DoesNotImplementJSONMarshaler(t *testing.T) {
	// this test is necessary check for correct shadowing in CreateAliasWithSig.MarshalJSON method
	type shadowed CreateAliasWithSig
	var v shadowed
	_, typeImplements := interface{}(v).(json.Marshaler)
	require.False(t, typeImplements, "type must not implement Marshaler")
	_, pointerImlements := interface{}(&v).(json.Marshaler)
	require.False(t, pointerImlements, "pointer must not implement Marshaler")
}
