package metamask

import "github.com/semrush/zenrpc/v2"

//go:generate zenrpc

type MetaMask struct{ zenrpc.Service }

func (as MetaMask) Eth_blockNumber() int {
	return 5
}
func (as MetaMask) Net_version() int {
	return 3
}

func (as MetaMask) Eth_getBalance(addr string, blockNumber int) int {
	return 100
}

func (as MetaMask) Eth_getBlockByNumber(blockNumber int, filter bool) string {
	return "blockContent"
}
