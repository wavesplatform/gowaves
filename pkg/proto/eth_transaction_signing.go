package proto

import (
	"math/big"
)

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

// MakeEthereumSigner returns a EthereumSigner based on the given chain config and block number.
func MakeEthereumSigner(chainID *big.Int) EthereumSigner {
	// nickeskov: LondonSigner is a main signer after the London hardfork (hardfork date - 05.08.2021)
	return NewLondonEthereumSigner(chainID)
}

func ExtractEthereumSender(signer EthereumSigner, tx *EthereumTransaction) (EthereumAddress, error) {
	addr, err := signer.Sender(tx)
	if err != nil {
		return EthereumAddress{}, err
	}
	return addr, nil
}
