package fourbyte

import "math/big"

var (
	ABIPadding = 32
)

var (
	Big1  = big.NewInt(1)
	Big32 = big.NewInt(32)
	// MaxUint256 is the maximum value that can be represented by a uint256.
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(Big1, 256), Big1)
)
