package metamask

import (
	"math/big"
	"strconv"
	"strings"
)

func hexUintToUint64(s string) (uint64, error) {
	trimmed := strings.TrimPrefix(s, "0x")
	u, err := strconv.ParseUint(trimmed, 16, 64)
	if err != nil {
		return 0, err
	}
	return u, nil
}

func bigIntToHexString(n *big.Int) string {
	return "0x" + n.Text(16)
}

func uint64ToHexString(n uint64) string {
	return "0x" + strconv.FormatUint(n, 16)
}
