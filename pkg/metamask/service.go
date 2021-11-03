package metamask

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/semrush/zenrpc/v2"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

func RunMetaMaskService(ctx context.Context, address string, service RPCService, enableRpcServiceLog bool) error {
	// TODO(nickeskov): what about `BatchMaxLen` option?
	rpc := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true, AllowCORS: true})
	rpc.Register("", service) // public

	if enableRpcServiceLog {
		rpc.Use(zenrpcZapLoggerMiddleware)
	}

	http.Handle("/eth", rpc)

	server := &http.Server{Addr: address, Handler: nil}

	go func() {
		<-ctx.Done()
		zap.S().Info("shutting down metamask service...")
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("failed to shutdown metamask service: %v", err)
		}
	}()
	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

//go:generate zenrpc

type RPCService struct {
	zenrpc.Service
	nodeRPCApp nodeRPCApp
}

// TODO(nickeskov): create error type

func NewRPCService(nodeServices *services.Services) RPCService {
	return RPCService{
		nodeRPCApp: nodeRPCApp{nodeServices},
	}
}

/* Returns the number of most recent block */
func (s RPCService) Eth_blockNumber() (string, error) {
	// returns integer of the current block number the client is on
	height, err := s.nodeRPCApp.State.Height()
	if err != nil {
		// todo(nickeskov): convert to RPC API error with corresponding code
		return "", err
	}
	return uint64ToHexString(height), nil
}

/* Returns the current network id */
func (s RPCService) Net_version() string {
	return s.Eth_chainId()
}

/* Returns the chain id */
func (s RPCService) Eth_chainId() string {
	return uint64ToHexString(uint64(s.nodeRPCApp.Scheme))
	//return "0x1" - show real currency price
}

/* Returns the balance of the account of given address
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getBalance(address, blockOrTag string) (string, error) {
	zap.S().Debugf("Eth_getBalance was called: address %q, blockOrTag %q", address, blockOrTag)

	// return balance in wei. 1 ether is equivalent to 1 x 10^18 wei (
	ethAddr, err := proto.NewEthereumAddressFromHexString(address)
	if err != nil {
		// todo log err
		return "", err
	}
	wavesAddr, err := ethAddr.ToWavesAddress(s.nodeRPCApp.Scheme)
	if err != nil {
		// todo log err
		return "", err
	}
	amount, err := s.nodeRPCApp.State.AccountBalance(proto.Recipient{Address: &wavesAddr}, nil)
	if err != nil {
		// todo log err
		return "", err
	}
	return bigIntToHexString(proto.WaveletToEthereumWei(amount)), nil // 0.159
}

type GetBlockByNumberResponse struct {
	Number string `json:"number"`
}

/* Returns information about a block by block number.
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
   - filter: if true it returns the full transaction objects, if false only the hashes of the transactions */
func (s RPCService) Eth_getBlockByNumber(blockOrTag string, filter bool) GetBlockByNumberResponse {
	zap.S().Debugf("Eth_getBlockByNumber was called: blockOrTag %q, filter \"%t\"", blockOrTag, filter)
	// TODO(nickeskov): scala crunch...
	return GetBlockByNumberResponse{
		Number: blockOrTag,
	}
}

/* Returns the current price per gas in wei */
func (s RPCService) Eth_gasPrice() string {
	return uint64ToHexString(proto.EthereumGasPrice)
}

type EstimateGasRequest struct {
	To    *proto.EthereumAddress `json:"to"`
	Value *string                `json:"value"`
	Data  *string                `json:"data"`
}

func (s RPCService) Eth_estimateGas(req EstimateGasRequest) (string, error) {
	var (
		value = new(big.Int)
		data  []byte
	)
	if req.To == nil {
		zap.S().Debug("Eth_estimateGas: trying estimate gas for set dApp transaction")
		return "", errors.New("gas estimation for set dApp transaction is not permitted")
	}
	if req.Value != nil {
		var ok bool
		_, ok = value.SetString(strings.TrimPrefix(*req.Value, "0x"), 16)
		if !ok {
			zap.S().Debugf("Eth_estimateGas: failed decode from hex 'value'=%q as big.Int", *req.Value)
			return "", errors.New("invalid 'value' field")
		}
	}
	if req.Data != nil {
		var err error
		data, err = proto.DecodeFromHexString(*req.Data)
		if err != nil {
			zap.S().Debugf("Eth_estimateGas: failed to decode from hex 'data'=%q as bytes", *req.Data)
			return "", errors.Errorf("invalid 'data' field, %v", err)
		}
	}

	txKind, err := state.GuessEthereumTransactionKind(data, req.To, s.nodeRPCApp.State.AssetInfoByID)
	if err != nil {
		return "", errors.Errorf("failed to guess ethereum tx kind, %v", err)
	}
	switch txKind {
	case state.EthereumTransferWavesKind:
		return strconv.Itoa(proto.MinFee), nil
	case state.EthereumTransferAssetsKind:
		fee := proto.MinFee
		assetID := (*proto.AssetID)(req.To)

		asset, err := s.nodeRPCApp.State.AssetInfoByID(*assetID, true)
		if err != nil {
			return "", errors.Errorf("failed to get asset info, %v", err)
		}
		if asset.Scripted {
			fee += proto.MinFeeScriptedAsset
		}
		return strconv.Itoa(fee), nil
	case state.EthereumInvokeKind:
		//fee := proto.MinFeeInvokeScript
		//
		//scriptAddr, err := req.To.ToWavesAddress(s.nodeRPCApp.Scheme)
		//if err != nil {
		//	return "", err
		//}
		//tree, err := s.nodeRPCApp..scriptsStorage.newestScriptByAddr(scriptAddr, !params.initialisation)
		//if err != nil {
		//	return nil, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
		//}
		//db, err := ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
		//if err != nil {
		//	return nil, err
		//}
		//decodedData, err := db.ParseCallDataRide(ethTx.Data())
		//if err != nil {
		//	return nil, errors.Errorf("failed to parse ethereum data, %v", err)
		//}
		return "", nil
	default:
		return "", errors.Errorf("unexpected ethereum tx kind")
	}
}

type ethCallParams struct {
	To   proto.EthereumAddress `json:"to"`
	Data string                `json:"data"`
}

func (c ethCallParams) String() string {
	return fmt.Sprintf("Eth_callParams(to=%s,data=%s)", c.To, c.Data)
}

var (
	erc20SymbolSelector   = ethabi.Signature("symbol()").Selector()           // "0x95d89b41"
	erc20DecimalsSelector = ethabi.Signature("decimals()").Selector()         // "0x313ce567"
	erc20BalanceSelector  = ethabi.Signature("balanceOf(address)").Selector() // "0x70a08231"
)

func (s RPCService) Eth_call(params ethCallParams) (string, error) {
	// TODO(nickeskov): what this method should send in case of error?
	zap.S().Debugf("Eth_call was called with %s", params.String())

	callData, err := proto.DecodeFromHexString(params.Data)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode 'data' parameter as hex")
	}
	var selector ethabi.Selector
	copy(selector[:], callData)

	shortAssetID := proto.AssetID(params.To)

	switch selector {
	case erc20SymbolSelector:
		fullInfo, err := s.nodeRPCApp.State.FullAssetInfo(shortAssetID)
		if err != nil {
			zap.S().Errorf("Eth_call: failed to fetch full asset info, %s: %v", params.String(), err)
			return "", err
		}
		return fullInfo.Name, nil
	case erc20DecimalsSelector:
		info, err := s.nodeRPCApp.State.AssetInfo(shortAssetID)
		if err != nil {
			zap.S().Errorf("Eth_call: failed to fetch asset info, %s: %v", params.String(), err)
			return "", err
		}
		return fmt.Sprintf("%d", info.Decimals), nil

	case erc20BalanceSelector:
		if len(callData) != ethabi.SelectorSize+proto.EthereumAddressSize {
			err := errors.Errorf(
				"invalid call data for %q ERC20 method, call data %q",
				erc20BalanceSelector.String(), params.Data,
			)
			zap.S().Debugf("Eth_call: %v", err)
			return "", err
		}
		ethAddr, err := proto.NewEthereumAddressFromBytes(callData[ethabi.SelectorSize:])
		if err != nil {
			return "", err
		}
		wavesAddr, err := ethAddr.ToWavesAddress(s.nodeRPCApp.Scheme)
		if err != nil {
			return "", err
		}
		info, err := s.nodeRPCApp.State.AssetInfo(shortAssetID)
		if err != nil {
			zap.S().Errorf("Eth_call: failed to fetch asset info, %s: %v", params.String(), err)
			return "", err
		}
		accountBalance, err := s.nodeRPCApp.State.AccountBalance(proto.Recipient{Address: &wavesAddr}, info.ID.Bytes())
		if err != nil {
			zap.S().Errorf("Eth_call: failed to fetch account balance for addr=%q, %s: %v",
				wavesAddr.String(), params.String(), err,
			)
			return "", err
		}
		return bigIntToHexString(proto.WaveletToEthereumWei(accountBalance)), nil
	default:
		return "", errors.Errorf("unexpected call, %s", params.String())
	}
}

/* Returns the compiled smart contract code, if any, at a given address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getCode(address, blockOrTag string) (string, error) {
	// TODO(nickeskov): what this method should send in case of error?

	zap.S().Debugf("Eth_getCode was called: address %q, blockOrTag %q", address, blockOrTag)

	ethAddr, err := proto.NewEthereumAddressFromHexString(address)
	if err != nil {
		return "", err
	}
	wavesAddr, err := ethAddr.ToWavesAddress(s.nodeRPCApp.Scheme)
	if err != nil {
		return "", err
	}
	_, err = s.nodeRPCApp.State.ScriptInfoByAccount(proto.Recipient{Address: &wavesAddr})
	switch {
	case state.IsNotFound(err):
		// it's not a DApp
		return "0x", nil
	case err != nil:
		zap.S().Errorf("Eth_getCode: failed to get script info by account, addr=%q: %v", wavesAddr.String(), err)
		return "", err
	default:
		// it's a DApp
		return "0xff", nil
	}
}

/* Returns the number of transactions sent from an address.
   - address: 20 Bytes - address to check for balance
   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */
func (s RPCService) Eth_getTransactionCount(address, blockOrTag string) string {
	zap.S().Debugf("Eth_getTransactionCount was called: address %q, blockOrTag %q", address, blockOrTag)
	return int64ToHexString(common.UnixMillisFromTime(s.nodeRPCApp.Time.Now()))
}

/* Creates new message call transaction or a contract creation for signed transactions.
   - signedTxData: The signed transaction data. */
func (s RPCService) Eth_sendrawtransaction(signedTxData string) (proto.EthereumHash, error) {
	// TODO(nickeskov): what this method should send in case of error?

	data, err := proto.DecodeFromHexString(signedTxData)
	if err != nil {
		zap.S().Errorf("Eth_sendrawtransaction: failed to decode ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}

	// TODO(nickeskov): check max payload size

	var tx proto.EthereumTransaction
	err = tx.DecodeCanonical(data)
	if err != nil {
		zap.S().Errorf("Eth_sendrawtransaction: failed to unmarshal rlp encoded ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}

	txID, err := tx.GetID(s.nodeRPCApp.Scheme)
	if err != nil {
		zap.S().Errorf("Eth_sendrawtransaction: failed to get ID of ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}
	ethTxID := proto.BytesToEthereumHash(txID)
	to := tx.To()
	from, err := tx.From()
	if err != nil {
		zap.S().Errorf(
			"Eth_sendrawtransaction: failed to get sender of ethereum transaction (ethTxID=%q, to=%q): %v",
			ethTxID.String(), to.String(), err,
		)
		return proto.EthereumHash{}, err
	}

	if err := s.nodeRPCApp.UtxPool.Add(&tx); err != nil {
		zap.S().Warnf(
			"Eth_sendrawtransaction: failed to add ethereum transaction (ethTxID=%q, to=%q, from=%q) to UTXPool: %v",
			ethTxID.String(), to.String(), from.String(), err,
		)
		// TODO(nickeskov): what is correct response?
		return proto.EthereumHash{}, err
	}

	respCh := make(chan error, 1)
	// TODO(nickeskov): add context?
	s.nodeRPCApp.InternalChannel <- messages.NewBroadcastTransaction(respCh, &tx)

	select {
	case <-time.After(5 * time.Second):
		zap.S().Errorf(
			"Eth_sendrawtransaction: timeout waiting response from internal FSM for ethereum tx (ethTxID=%q, to=%q, from=%q)",
			ethTxID.String(), to.String(), from.String(),
		)
		return proto.EthereumHash{}, errors.New("timeout waiting response from internal FSM")
	case err := <-respCh:
		if err != nil {
			zap.S().Errorf("Eth_sendrawtransaction: error from internal FSM for ethereum tx (ethTxID=%q, to=%q, from=%q): %v",
				ethTxID.String(), to.String(), from.String(), err,
			)
			return proto.EthereumHash{}, err
		}
		return ethTxID, nil
	}
}

type GetTransactionReceiptResponse struct {
	TransactionHash   proto.EthereumHash     `json:"transactionHash"`
	TransactionIndex  string                 `json:"transactionIndex"`
	BlockHash         string                 `json:"blockHash"`
	BlockNumber       string                 `json:"blockNumber"`
	From              proto.EthereumAddress  `json:"from"`
	To                *proto.EthereumAddress `json:"to"`
	CumulativeGasUsed string                 `json:"cumulativeGasUsed"`
	GasUsed           string                 `json:"gasUsed"`
	ContractAddress   *proto.EthereumAddress `json:"contractAddress"`
	Logs              []string               `json:"logs"`
	LogsBloom         proto.EthereumHash     `json:"logsBloom"`
	Status            string                 `json:"status"`
}

func (s RPCService) Eth_getTransactionReceipt(ethTxID proto.EthereumHash) (GetTransactionReceiptResponse, error) {
	txID := crypto.Digest(ethTxID)
	tx, txIsFailed, err := s.nodeRPCApp.State.TransactionByIDWithStatus(txID.Bytes())
	if state.IsNotFound(err) {
		zap.S().Debugf("Eth_getTransactionReceipt: transaction with ID=%q or ethID=%q cannot be found",
			txID, ethTxID,
		)
		return GetTransactionReceiptResponse{}, errors.Errorf("transaction with ethID=%q not found", ethTxID)
	}
	ethTx, ok := tx.(*proto.EthereumTransaction)
	if !ok {
		zap.S().Debugf(
			"Eth_getTransactionReceipt: transaction with ID=%q or ethID=%q is not 'EthereumTransaction'",
			txID, ethTxID,
		)
		// according to the scala node implementation
		return GetTransactionReceiptResponse{}, nil
	}

	to := ethTx.To()
	from, err := ethTx.From()
	if err != nil {
		zap.S().Errorf(
			"Eth_getTransactionReceipt: failed to get sender (from) for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return GetTransactionReceiptResponse{}, errors.New("failed to get sender from tx")
	}

	blockHeight, err := s.nodeRPCApp.State.TransactionHeightByID(txID.Bytes())
	if err != nil {
		zap.S().Errorf(
			"Eth_getTransactionReceipt: failed to get block height for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return GetTransactionReceiptResponse{}, errors.New("failed to get blockNumber for transaction")
	}

	blockHeader, err := s.nodeRPCApp.State.HeaderByHeight(blockHeight)
	if err != nil {
		zap.S().Errorf(
			"Eth_getTransactionReceipt: failed to get block header for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return GetTransactionReceiptResponse{}, errors.New("failed to get blockHeader for transaction")
	}
	txStatus := "0x1"
	if txIsFailed {
		txStatus = "0x0"
	}
	gasLimit := uint64ToHexString(tx.GetFee())

	return GetTransactionReceiptResponse{
		TransactionHash:   ethTxID,
		TransactionIndex:  "0x01",                                          // according to the scala node implementation
		BlockHash:         proto.EncodeToHexString(blockHeader.ID.Bytes()), // should be always 32bytes
		BlockNumber:       uint64ToHexString(blockHeight),
		From:              from,
		To:                to,
		CumulativeGasUsed: gasLimit,
		GasUsed:           gasLimit,
		ContractAddress:   nil,
		Logs:              []string{},
		LogsBloom:         proto.EthereumHash{},
		Status:            txStatus,
	}, nil
}
