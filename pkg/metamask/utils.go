package metamask

import (
	"fmt"
	"math/big"
)

func bigIntToHexString(n *big.Int) string {
	return fmt.Sprintf("0x%x", n)
}

func uint64ToHexString(n uint64) string {
	return fmt.Sprintf("0x%x", n)
}

func int64ToHexString(n int64) string {
	return fmt.Sprintf("0x%x", n)
}
