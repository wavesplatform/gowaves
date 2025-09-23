package bls

import (
	"encoding/binary"
	"fmt"
)

const PoPMessageSize = PublicKeySize + 4

// BuildPoPMessage constructs the PoP message from the public key and height.
func BuildPoPMessage(pk PublicKey, height uint32) []byte {
	msg := make([]byte, PoPMessageSize)
	copy(msg, pk[:])
	binary.BigEndian.PutUint32(msg[PublicKeySize:], height)
	return msg
}

// ProvePoP creates a proof of possession (PoP) message from the given public key and height. Then the message is
// signed with a given secret key. The function returns the PoP message and its signature.
func ProvePoP(sk SecretKey, pk PublicKey, height uint32) ([]byte, Signature, error) {
	cpk, err := pk.ToCIRCLPublicKey()
	if err != nil {
		return nil, Signature{}, fmt.Errorf("failed to prove PoP, invalid public key: %w", err)
	}
	if !cpk.Validate() {
		return nil, Signature{}, fmt.Errorf("failed to prove PoP, invalid public key")
	}
	msg := BuildPoPMessage(pk, height)
	sig, err := Sign(sk, msg)
	if err != nil {
		return nil, Signature{}, fmt.Errorf("failed to prove PoP: %w", err)
	}
	return msg, sig, nil
}

// VerifyPoP verifies the proof of possession (PoP) signature for the given public key and height.
// It reconstructs the PoP message and verifies the signature against it.
func VerifyPoP(pk PublicKey, height uint32, sig Signature) (bool, error) {
	msg := BuildPoPMessage(pk, height)
	return Verify(pk, msg, sig)
}
