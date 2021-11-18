package metamask

import (
	"encoding/hex"
	"github.com/semrush/zenrpc/v2"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"strings"
)

//go:generate zenrpc

type MetaMask struct {
	zenrpc.Service
	state state.State
}

// Eth_BlockNumber returns the number of most recent block
func (s MetaMask) Eth_BlockNumber() int {

	// returns integer of the current block number the client is on
	return 5
}

// Net_Version returns the current network id
func (s MetaMask) Net_Version() int {
	return 3
}

// Eth_ChainId returns the chain id
func (s MetaMask) Eth_ChainId() string {
	return "0x5d"
}

// Eth_GetBalance returns the balance of the account of given address
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s MetaMask) Eth_GetBalance(address string, blockNumber int) string {
	// return balance in wei. 1 ether is equivalent to 1 x 10^18 wei (
	return "0x0234c8a3397aab58" // 0.159
}

// Eth_GetBlockByNumber returns information about a block by block number.
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
//   - filter: if true it returns the full transaction objects, if false only the hashes of the transactions */
func (s MetaMask) Eth_GetBlockByNumber(block int, filter bool) string {
	return "blockContent"
}

// Eth_GasPrice returns the current price per gas in wei
func (s MetaMask) Eth_GasPrice() int {
	return 1
}

// Eth_GetCode returns the compiled smart contract code, if any, at a given address.
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s MetaMask) Eth_GetCode(address string, block int) string {
	return ""
}

// Eth_GetTransactionCount returns the number of transactions sent from an address.
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s MetaMask) Eth_GetTransactionCount(address string, block string) string {
	return "0x1"
}

// Eth_SendRawTransaction creates new message call transaction or a contract creation for signed transactions.
//   - signedTxData: The signed transaction data.
func (s MetaMask) Eth_SendRawTransaction(signedTxData string) string {

	encodedTx := strings.TrimPrefix(signedTxData, "0x")

	data, err := hex.DecodeString(encodedTx)
	if err != nil {
		zap.S().Errorf("failed to decode tx: %v", err)
	}

	var tx proto.EthereumTransaction
	err = tx.DecodeCanonical(data)
	if err != nil {
		zap.S().Errorf("failed to unmarshal rlp encoded ethereum transaction: %v", err)
	}

	signer := proto.MakeEthereumSigner(tx.ChainId())
	sender, err := proto.ExtractEthereumSender(signer, &tx)
	if err != nil {
		zap.S().Errorf("failed to get sender")
	}
	zap.S().Infof("Receiver is %s\n", tx.To().String())
	zap.S().Infof("Sender is %s\n", sender.Hex())

	return ""
}
