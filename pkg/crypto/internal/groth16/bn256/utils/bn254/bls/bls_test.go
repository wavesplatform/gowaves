//go:build groth16
// +build groth16

package bls

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestKeyPairBytes(t *testing.T) {
	e0, err := NewKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := e0.secret
	b := e0.ToBytes()
	e1, err := NewKeyPairFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e0.Public.ToBytes(), e1.Public.ToBytes()) {
		t.Fatal("bad key enc/dec, a")
	}
	if !bytes.Equal(e0.secret[:], e1.secret[:]) {
		t.Fatal("bad key enc/dec, b")
	}
	e2, err := NewKeyPairFromSecret(secret[:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e0.Public.ToBytes(), e2.Public.ToBytes()) {
		t.Fatal("bad key enc/dec, c")
	}
	if !bytes.Equal(e0.secret[:], e2.secret[:]) {
		t.Fatal("bad key enc/dec, d")
	}
}

func TestVerify(t *testing.T) {
	domain := []byte{0x00, 0x00, 0x00, 0x00}
	message := []byte{0x10, 0x11, 0x12, 0x13}
	account, err := NewKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := account.Public
	signer := NewBLSSigner(domain, account)
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewBLSVerifier(domain)
	verified, err := verifier.Verify(message, signature, publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatalf("signature is not verified")
	}
	message2 := []byte{0xff, 0xff, 0xff, 0xff}
	verified, err = verifier.Verify(message2, signature, publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatalf("signature is verified with broken message")
	}
}

func TestVerifyAggregatedCommon(t *testing.T) {
	domain := []byte{0x00, 0x00, 0x00, 0x00}
	message := []byte{0x10, 0x11, 0x12, 0x13}
	signerSize := 1000
	publicKeys := make([]*PublicKey, signerSize)
	signatures := make([]*Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		account, err := NewKeyPair(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := NewBLSSigner(domain, account)
		publicKeys[i] = account.Public
		signature, err := signer.Sign(message)
		if err != nil {
			t.Fatal(err)
		}
		signatures[i] = signature
	}
	verifier := NewBLSVerifier(domain)
	aggregatedSignature := verifier.AggregateSignatures(signatures)
	verified, err := verifier.VerifyAggregateCommon(message, publicKeys, aggregatedSignature)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatalf("signature is not verified")
	}
	account, err := NewKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKeys[signerSize-1] = account.Public
	verified, err = verifier.VerifyAggregateCommon(message, publicKeys, aggregatedSignature)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatalf("signature is verified with unrelated pubkey")
	}
}

func TestVerifyAggregated(t *testing.T) {
	domain := []byte{0x00, 0x00, 0x00, 0x00}
	signerSize := 1000
	publicKeys := make([]*PublicKey, signerSize)
	messages := make([]Message, signerSize)
	signatures := make([]*Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		message := make([]byte, 4)
		_, err := rand.Read(message)
		if err != nil {
			t.Fatal(err)
		}
		account, err := NewKeyPair(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := NewBLSSigner(domain, account)

		signature, err := signer.Sign(message)
		if err != nil {
			t.Fatal(err)
		}
		messages[i] = message
		publicKeys[i] = account.Public
		signatures[i] = signature
	}
	verifier := NewBLSVerifier(domain)
	aggregatedSignature := verifier.AggregateSignatures(signatures)
	verified, err := verifier.VerifyAggregate(messages, publicKeys, aggregatedSignature)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatalf("signature is not verified")
	}
}

func BenchmarkVerifyAggregated24ByteMsgSHA256(t *testing.B) {
	domain := []byte{0x00, 0x00, 0x00, 0x00}
	signerSize := 1000
	publicKeys := make([]*PublicKey, signerSize)
	messages := make([]Message, signerSize)
	signatures := make([]*Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		message := make([]byte, 20)
		_, err := rand.Read(message)
		if err != nil {
			t.Fatal(err)
		}
		account, err := NewKeyPair(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := NewBLSSigner(domain, account)

		signature, err := signer.Sign(message)
		if err != nil {
			t.Fatal(err)
		}
		messages[i] = message
		publicKeys[i] = account.Public
		signatures[i] = signature
	}
	verifier := NewBLSVerifier(domain)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		aggregatedSignature := verifier.AggregateSignatures(signatures)
		_, _ = verifier.VerifyAggregate(messages, publicKeys, aggregatedSignature)
	}
}

func BenchmarkVerifyAggregated24ByteMsgKeccak256(t *testing.B) {
	domain := []byte{0x00, 0x00, 0x00, 0x00}
	signerSize := 1000
	publicKeys := make([]*PublicKey, signerSize)
	messages := make([]Message, signerSize)
	signatures := make([]*Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		message := make([]byte, 20)
		_, err := rand.Read(message)
		if err != nil {
			t.Fatal(err)
		}
		account, err := NewKeyPair(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := NewBLSSigner(domain, account)
		signature, err := signer.Sign(message)
		if err != nil {
			t.Fatal(err)
		}
		messages[i] = message
		publicKeys[i] = account.Public
		signatures[i] = signature
	}
	verifier := NewBLSVerifier(domain)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		aggregatedSignature := verifier.AggregateSignatures(signatures)
		_, _ = verifier.VerifyAggregate(messages, publicKeys, aggregatedSignature)
	}
}
