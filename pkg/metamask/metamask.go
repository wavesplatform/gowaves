package metamask

import (
	"fmt"

	"github.com/semrush/zenrpc/v2"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

//go:generate zenrpc

type RPCService struct {
	zenrpc.Service
	services *services.Services
}

// TODO(nickeskov): create error type

func NewRPCService(state *services.Services) RPCService {
	return RPCService{
		services: state,
	}
}

/* Returns the number of most recent block */
func (s RPCService) Eth_blockNumber() (string, error) {
	// returns integer of the current block number the client is on
	height, err := s.services.State.Height()
	if err != nil {
		// todo(nickeskov): convert to RPC API error with corresponding code
		return "", err
	}
	return fmt.Sprintf("0x%x", height), nil
}

/* Returns the current network id */
func (s RPCService) Net_version() int {
	// TODO(nickeskov): change it
	return 1
}

/* Returns the chain id */
func (s RPCService) Eth_chainId() string {
	return fmt.Sprintf("0x%x", s.services.Scheme)
	//return "0x1" - show real currency price
}

/* Returns the balance of the account of given address
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getBalance(address, blockOrTag string) (string, error) {
	zap.S().Infof("Eth_getBalance was called: address %q, blockOrTag %q", address, blockOrTag)

	// return balance in wei. 1 ether is equivalent to 1 x 10^18 wei (
	ethAddr, err := proto.NewEthereumAddressFromHexString(address)
	if err != nil {
		// todo
		return "", err
	}
	amount, err := s.accountBalance(ethAddr)
	if err != nil {
		// todo
		return "", err
	}
	return fmt.Sprintf("0x%x", proto.WaveletToEthereumWei(amount)), nil // 0.159
}

/* Returns information about a block by block number.
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
   - filter: if true it returns the full transaction objects, if false only the hashes of the transactions */
func (s RPCService) Eth_getBlockByNumber(blockOrTag string, filter bool) map[string]string {
	zap.S().Infof("Eth_getBlockByNumber was called: blockOrTag %q, filter \"%t\"", blockOrTag, filter)
	// TODO(nickeskov): scala crunch...
	return map[string]string{
		"number": blockOrTag,
	}
}

/* Returns the current price per gas in wei */
func (s RPCService) Eth_gasPrice() string {
	return fmt.Sprintf("0x%x", proto.EthereumGasPrice)
}

func (s RPCService) Eth_estimateGas(gas string) {
	// TODO(nickeskov):
}

//type callParams struct {
//	To   proto.EthereumAddress `json:"to"`
//	Data string                `json:"data"`
//}
//
//var (
//	erc20SymbolSelector   = ethabi.Signature("symbol()").Selector()           // "0x95d89b41"
//	erc20DecimalsSelector = ethabi.Signature("decimals()").Selector()         // "0x313ce567"
//	erc20BalanceSelector  = ethabi.Signature("balanceOf(address)").Selector() // "0x70a08231"
//)
//
//func (s RPCService) Eth_call(params callParams) (string, error) {
//	zap.S().Infof("Eth_call was called: params %+v", params)
//
//	contractAddress := params.To
//	hexCallData := params.Data
//
//	callData, err := proto.DecodeFromHexString(hexCallData)
//	if err != nil {
//		return "", errors.Wrapf(err, "failed to decode 'data' parameter as hex")
//	}
//	var selector ethabi.Selector
//	copy(selector[:], callData)
//
//	switch selector {
//	case erc20SymbolSelector:
//	case erc20DecimalsSelector:
//
//	case erc20BalanceSelector:
//		if len(callData) != ethabi.SelectorSize+proto.EthereumAddressSize {
//			return "", errors.Errorf(
//				"invalid call data for \"balanceOf(address)\" ERC20 function, call data %q", hexCallData,
//			)
//		}
//		addr, err := proto.NewEthereumAddressFromBytes(callData[ethabi.SelectorSize:])
//		if err != nil {
//			// todo
//			return "", err
//		}
//		amount, err := s.accountBalance(addr)
//		if err != nil {
//			// todo
//			return "", err
//		}
//		return fmt.Sprintf("0x%x", proto.WaveletToEthereumWei(amount)), nil // 0.159
//	default:
//		return "", errors.Errorf(
//			"unexpected call (selector %q) for %q at %q",
//			selector.String(), hexCallData, contractAddress.String(),
//		)
//	}
//}

/* Returns the compiled smart contract code, if any, at a given address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getCode(address, blockOrTag string) (string, error) {
	zap.S().Infof("Eth_getCode was called: address %q, blockOrTag %q", address, blockOrTag)

	ethAddr, err := proto.NewEthereumAddressFromHexString(address)
	if err != nil {
		// todo
		return "", err
	}
	wavesAddr, err := ethAddr.ToWavesAddress(s.services.Scheme)
	if err != nil {
		// todo
		return "", err
	}
	_, err = s.services.State.ScriptInfoByAccount(proto.Recipient{Address: &wavesAddr})
	switch {
	case state.IsNotFound(err):
		// it's not a DApp
		return "0x", nil
	case err != nil:
		// todo
		return "", err
	}
	// it's a DApp
	return "0xff", nil
}

/* Returns the number of transactions sent from an address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getTransactionCount(address, blockOrTag string) string {
	zap.S().Infof("Eth_getTransactionCount was called: address %q, blockOrTag %q", address, blockOrTag)
	return "0x1"
}

/* Creates new message call transaction or a contract creation for signed transactions.
   - signedTxData: The signed transaction data. */
func (s RPCService) Eth_sendrawtransaction(signedTxData string) string {

	data, err := proto.DecodeFromHexString(signedTxData)
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

func (s RPCService) accountBalance(addr proto.Address) (uint64, error) {
	wavesAddr, err := addr.ToWavesAddress(s.services.Scheme)
	if err != nil {
		return 0, err
	}
	amount, err := s.services.State.AccountBalance(proto.Recipient{Address: &wavesAddr}, nil)
	if err != nil {
		return 0, err
	}
	return amount, nil
}
