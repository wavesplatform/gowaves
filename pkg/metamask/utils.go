package metamask

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"math/big"
)

func unmarshalTransactionToFieldFastRLP(value *fastrlp.Value) (*Address, error) {
	toBytes, err := value.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TO bytes")
	}
	addrTo := &Address{}
	switch len(toBytes) {
	case 0:
		addrTo = nil
	case len(addrTo):
		copy(addrTo[:], toBytes)
	default:
		return nil, errors.Errorf("failed to parse TO bytes as address, invalid bytes length %d", len(toBytes))
	}
	return addrTo, nil
}

func unmarshalSignatureValuesFastRLP(vValue, rValue, sValue *fastrlp.Value) (V, R, S big.Int, err error) {
	if getErr := vValue.GetBigInt(&V); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'V'")
	}

	if getErr := rValue.GetBigInt(&R); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'R'")
	}

	if getErr := sValue.GetBigInt(&S); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'S'")
	}

	return V, R, S, nil
}

// copyBytes returns an exact copy of the provided bytes.
func copyBytes(bytes []byte) []byte {
	if bytes == nil {
		return nil
	}
	copiedBytes := make([]byte, len(bytes))
	copy(copiedBytes, bytes)
	return copiedBytes
}

// copyBytes returns an exact copy of the provided big.Int.
func copyBigInt(v *big.Int) *big.Int {
	if v == nil {
		return nil
	}
	return new(big.Int).Set(v)
}

// HexEncodeToBytes encodes b as a hex string with 0x prefix.
func HexEncodeToBytes(b []byte) []byte {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return enc
}

// HexEncodeToString encodes b as a hex string with 0x prefix.
func HexEncodeToString(b []byte) string {
	return string(HexEncodeToBytes(b))
}
