package metamask

import "github.com/semrush/zenrpc/v2"

//go:generate zenrpc

type MetaMask struct{ zenrpc.Service }

/* Returns the number of most recent block */
func (as MetaMask) Eth_blockNumber() int {

	// returns integer of the current block number the client is on
	return 5
}

/* Returns the current network id */
func (as MetaMask) Net_version() int {
	return 3
}

/* Returns the balance of the account of given address
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (as MetaMask) Eth_getBalance(address string, blockNumber int) int {
	return 100
}

/* Returns information about a block by block number.
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
   - filter: if true it returns the full transaction objects, if false only the hashes of the transactions */
func (as MetaMask) Eth_getBlockByNumber(block int, filter bool) string {
	return "blockContent"
}

/* Returns the current price per gas in wei */
func (as MetaMask) Eth_gasPrice() int {
	return 1
}

/* Returns the compiled smart contract code, if any, at a given address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (as MetaMask) Eth_getCode(address string, block int) string {
	return ""
}

/* Returns the number of transactions sent from an address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (as MetaMask) Eth_getTransactionCount(address string, block string) string {
	return "0x1"
}

/* Creates new message call transaction or a contract creation for signed transactions.
   - signedTxData: The signed transaction data. */
func (as MetaMask) Eth_sendrawtransaction(signedTxData string) string {
	// returns 32 Bytes - the transaction hash, or the zero hash if the transaction is not yet available.
	return ""
}
