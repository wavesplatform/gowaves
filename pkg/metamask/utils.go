package metamask

import (
	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"math/big"
)

func unmarshalTransactionToFieldFastRLP(value *fastrlp.Value) (*EthAddress, error) {
	toBytes, err := value.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TO bytes")
	}
	addrTo := &EthAddress{}
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
