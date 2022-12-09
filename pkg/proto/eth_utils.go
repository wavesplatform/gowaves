package proto

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
)

const (
	ethereumWei            uint64 = 1
	ethereumGWei                  = 1e9 * ethereumWei
	ethereumEther                 = 1e9 * ethereumGWei
	waveletToWeiMultiplier        = ethereumEther / PriceConstant
)

func WaveletToEthereumWei(waveletAmount uint64) *big.Int {
	return new(big.Int).Mul(
		new(big.Int).SetUint64(waveletAmount),
		new(big.Int).SetUint64(waveletToWeiMultiplier),
	)
}

func EthereumWeiToWavelet(weiAmount *big.Int) (int64, error) {
	wavelets := new(big.Int).Div(weiAmount, new(big.Int).SetUint64(waveletToWeiMultiplier))
	if !wavelets.IsInt64() {
		return 0, errors.Errorf("too many wavelets=%d", wavelets)
	}
	return wavelets.Int64(), nil
}

func unmarshalTransactionToFieldFastRLP(value *fastrlp.Value) (*EthereumAddress, error) {
	toBytes, err := value.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TO bytes")
	}
	addrTo := &EthereumAddress{}
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

func unmarshalSignatureValuesFastRLP(vValue, rValue, sValue *fastrlp.Value) (v, r, s big.Int, err error) {
	if getErr := vValue.GetBigInt(&v); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'V'")
	}

	if getErr := rValue.GetBigInt(&r); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'R'")
	}

	if getErr := sValue.GetBigInt(&s); getErr != nil {
		return big.Int{}, big.Int{}, big.Int{}, errors.Wrap(getErr, "failed to parse signature value 'S'")
	}

	return v, r, s, nil
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
