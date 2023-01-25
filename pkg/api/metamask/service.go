package metamask

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/semrush/zenrpc/v2"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
)

type nodeRPCApp struct {
	*services.Services
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

// Eth_BlockNumber returns the number of most recent block
func (s RPCService) Eth_BlockNumber() (string, error) {
	// returns integer of the current block number the client is on
	height, err := s.nodeRPCApp.State.Height()
	if err != nil {
		// todo(nickeskov): convert to RPC API error with corresponding code
		return "", err
	}
	return uint64ToHexString(height), nil
}

// Net_Version returns the current network id
func (s RPCService) Net_Version() string {
	return s.Eth_ChainId()
}

// Eth_ChainId returns the chain id
func (s RPCService) Eth_ChainId() string {
	return uint64ToHexString(uint64(s.nodeRPCApp.Scheme))
}

// Eth_GetBalance returns the balance in wei of the account of given address. 1 ether is equivalent to 1 x 10^18 wei
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s RPCService) Eth_GetBalance(ethAddr proto.EthereumAddress, blockOrTag string) (string, error) {
	zap.S().Debugf("Eth_GetBalance was called: ethAddr %q, blockOrTag %q", ethAddr, blockOrTag)
	wavesAddr, err := ethAddr.ToWavesAddress(s.nodeRPCApp.Scheme)
	if err != nil {
		// todo log err
		return "", errors.Wrapf(err, "failed to convert ethereum address %q to waves address", ethAddr)
	}
	amount, err := s.nodeRPCApp.State.WavesBalance(proto.NewRecipientFromAddress(wavesAddr))
	if err != nil {
		// todo log err
		return "", errors.Wrapf(err, "failed to get waves balance for address %q", wavesAddr)
	}
	return bigIntToHexString(proto.WaveletToEthereumWei(amount)), nil
}

type GetBlockByNumberResponse struct {
	Number *string `json:"number"`
}

// Eth_GetBlockByNumber returns information about a block by block number.
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
//   - filterTxObj: if true it returns the full transaction objects, if false only the hashes of the transactions
func (s RPCService) Eth_GetBlockByNumber(blockOrTag string, filterTxObj bool) (GetBlockByNumberResponse, error) {
	zap.S().Debugf("Eth_GetBlockByNumber was called: blockOrTag %q, filter \"%t\"", blockOrTag, filterTxObj)
	var n proto.Height
	switch blockOrTag {
	case "earliest":
		n = 1
	case "latest":
		h, err := s.nodeRPCApp.State.Height()
		if err != nil {
			return GetBlockByNumberResponse{}, err
		}
		n = h
	case "pending":
		return GetBlockByNumberResponse{Number: nil}, nil
	default:
		u, err := hexUintToUint64(blockOrTag)
		if err != nil {
			return GetBlockByNumberResponse{}, errors.New("Request parameter is not number nor supported tag")
		}
		n = u
	}
	out := uint64ToHexString(n)
	return GetBlockByNumberResponse{
		Number: &out,
	}, nil
}

type GetBlockByHashResponse struct {
	BaseFeePerGas string `json:"baseFeePerGas"`
}

// Eth_GetBlockByHash returns block by provided blockID.
//   - blockIDBytes: block id in hexadecimal notation.
//   - filterTxObj: if true it returns the full transaction objects, if false only the hashes of the transactions.
func (s RPCService) Eth_GetBlockByHash(blockIDBytes proto.HexBytes, filterTxObj bool) (*GetBlockByHashResponse, error) {
	zap.S().Debugf("Eth_GetBlockByHash was called: blockIDBytes %q, filter \"%t\"", blockIDBytes, filterTxObj)
	blockID, err := proto.NewBlockIDFromBytes(blockIDBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse blockID from blockIDBytes %q", blockIDBytes.String())
	}
	_, err = s.nodeRPCApp.State.BlockIDToHeight(blockID)
	switch {
	case state.IsNotFound(err):
		return nil, nil // according to the scala node implementation
	case err != nil:
		return nil, errors.Wrapf(err, "failed to fetch heigh of block by blockID %q", blockID.String())
	default:
		return &GetBlockByHashResponse{BaseFeePerGas: "0x0"}, nil
	}
}

// Eth_GasPrice returns the current price per gas in wei
func (s RPCService) Eth_GasPrice() string {
	return uint64ToHexString(proto.EthereumGasPrice)
}

type estimateGasRequest struct {
	To    *proto.EthereumAddress `json:"to"`
	Value *string                `json:"value"`
	Data  *string                `json:"data"`
}

func (s RPCService) Eth_EstimateGas(req estimateGasRequest) (string, error) {
	var (
		value = new(big.Int)
		data  []byte
	)
	if req.To == nil {
		zap.S().Debug("Eth_EstimateGas: trying estimate gas for set dApp transaction")
		return "", errors.New("gas estimation for set dApp transaction is not permitted")
	}
	if req.Value != nil {
		var _, ok = value.SetString(strings.TrimPrefix(*req.Value, "0x"), 16)
		if !ok {
			zap.S().Debugf("Eth_EstimateGas: failed decode from hex 'value'=%q as big.Int", *req.Value)
			return "", errors.New("invalid 'value' field")
		}
	}
	if req.Data != nil {
		var err error
		data, err = proto.DecodeFromHexString(*req.Data)
		if err != nil {
			zap.S().Debugf("Eth_EstimateGas: failed to decode from hex 'data'=%q as bytes", *req.Data)
			return "", errors.Errorf("invalid 'data' field, %v", err)
		}
	}

	txKind, err := state.GuessEthereumTransactionKind(data)
	if err != nil {
		return "", errors.Errorf("failed to guess ethereum tx kind, %v", err)
	}
	switch txKind {
	case state.EthereumTransferWavesKind:
		return uint64ToHexString(proto.MinFee), nil
	case state.EthereumTransferAssetsKind:
		fee := proto.MinFee
		assetID := (*proto.AssetID)(req.To)

		asset, err := s.nodeRPCApp.State.AssetInfo(*assetID)
		if err != nil {
			return "", errors.Errorf("failed to get asset info, %v", err)
		}
		if asset.Scripted {
			fee += proto.MinFeeScriptedAsset
		}
		return uint64ToHexString(uint64(fee)), nil
	case state.EthereumInvokeKind:
		return uint64ToHexString(proto.MinFeeInvokeScript), nil
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
	erc20SymbolSelector            = ethabi.Signature("symbol()").Selector()                  // "0x95d89b41"
	erc20DecimalsSelector          = ethabi.Signature("decimals()").Selector()                // "0x313ce567"
	erc20BalanceSelector           = ethabi.Signature("balanceOf(address)").Selector()        // "0x70a08231"
	erc20SupportsInterfaceSelector = ethabi.Signature("supportsInterface(bytes4)").Selector() // "0x01ffc9a7"
)

// Eth_Call returns information about assets.
//   - params: the tx call object
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s RPCService) Eth_Call(params ethCallParams, blockOrTag string) (string, error) {
	zap.S().Debugf("Eth_Call was called: params %q, blockOrTag %q", params, blockOrTag)
	abiVal, err := ethCall(s.nodeRPCApp.State, s.nodeRPCApp.Scheme, params)
	if err != nil {
		zap.S().Debugf("Eth_Call: %v", err)
		return "0x", err
	}
	return proto.EncodeToHexString(abiVal), nil
}

func ethCall(state state.State, scheme proto.Scheme, params ethCallParams) ([]byte, error) {

	callData, err := proto.DecodeFromHexString(params.Data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode 'data' parameter as hex")
	}
	if l := len(callData); l < ethabi.SelectorSize {
		return nil, errors.Errorf("insufficient call data size: wanted at least %d, got %d",
			ethabi.SelectorSize, l,
		)
	}
	selector, err := ethabi.NewSelectorFromBytes(callData[:ethabi.SelectorSize])
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse selector from call data")
	}

	var (
		shortAssetID = proto.AssetID(params.To)
	)
	switch selector {
	case erc20SymbolSelector:
		fullInfo, err := state.FullAssetInfo(shortAssetID)
		if err != nil {
			zap.S().Debugf("Eth_Call: failed to fetch full asset info, %s: %v", params.String(), err)
			return nil, err
		}
		return ethabi.String(fullInfo.Name).EncodeToABI(), nil
	case erc20DecimalsSelector:
		info, err := state.AssetInfo(shortAssetID)
		if err != nil {
			zap.S().Debugf("Eth_Call: failed to fetch asset info, %s: %v", params.String(), err)
			return nil, err
		}
		return ethabi.Int(info.Decimals).EncodeToABI(), nil
	case erc20BalanceSelector:
		const (
			// ethabi.SelectorSize + 4 bytes padding = 16
			// example value from metamask: "0x70a082310000000000000000000000007fd3a8438edf428eeb1dafe75afd5f64dd5017bf"
			selectorSizeWithPadding = 16
		)
		if len(callData) != selectorSizeWithPadding+proto.EthereumAddressSize {
			return nil, errors.Errorf("invalid call data for %q ERC20 method, call data %q",
				erc20BalanceSelector.String(), params.Data,
			)
		}
		ethAddr, err := proto.NewEthereumAddressFromBytes(callData[selectorSizeWithPadding:])
		if err != nil {
			return nil, err
		}
		wavesAddr, err := ethAddr.ToWavesAddress(scheme)
		if err != nil {
			return nil, err
		}
		accountBalance, err := state.AssetBalance(proto.NewRecipientFromAddress(wavesAddr), shortAssetID)
		if err != nil {
			zap.S().Errorf("Eth_Call: failed to fetch account balance for addr=%q, %s: %v",
				wavesAddr.String(), params.String(), err,
			)
			return nil, err
		}
		return ethabi.Int(accountBalance).EncodeToABI(), nil
	case erc20SupportsInterfaceSelector:
		return ethabi.Bool(false).EncodeToABI(), nil
	default:
		return nil, nil // according to the scala node implementation ("0x" in the result will be returned)
	}
}

// Eth_GetCode returns the compiled smart contract code, if any, at a given address.
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s RPCService) Eth_GetCode(ethAddr proto.EthereumAddress, blockOrTag string) (string, error) {
	// TODO(nickeskov): what this method should send in case of error?
	zap.S().Debugf("Eth_GetCode was called: ethAddr %q, blockOrTag %q", ethAddr, blockOrTag)

	wavesAddr, err := ethAddr.ToWavesAddress(s.nodeRPCApp.Scheme)
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert ethereum address %q to waves address", ethAddr)
	}

	si, err := s.nodeRPCApp.State.ScriptBasicInfoByAccount(proto.NewRecipientFromAddress(wavesAddr))
	switch {
	case state.IsNotFound(err):
		// account has no script, trying fetch data as asset
		assetID := proto.AssetID(ethAddr)
		_, err := s.nodeRPCApp.State.AssetInfo(assetID)
		switch {
		case errors.Is(err, errs.UnknownAsset{}):
			// address has no script and it's not an asset
			return "0x", nil
		case err != nil:
			zap.S().Errorf("Eth_GetCode: failed to get asset info by assetID=%q: %v", assetID.String(), err)
			return "", err
		default:
			// it's an asset
			return "0xff", nil
		}
	case err != nil:
		zap.S().Errorf("Eth_GetCode: failed to get script info by account, addr=%q: %v", wavesAddr.String(), err)
		return "", err
	case si.IsDApp:
		// it's a DApp
		return "0xff", nil
	default:
		// account has an expression script, but it's not a DApp
		return "0x", nil
	}
}

// Eth_GetTransactionCount returns the number of transactions sent from an address.
//   - address: 20 Bytes - address to check for balance
//   - block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
func (s RPCService) Eth_GetTransactionCount(address, blockOrTag string) string {
	zap.S().Debugf("Eth_GetTransactionCount was called: address %q, blockOrTag %q", address, blockOrTag)
	return uint64ToHexString(uint64(common.UnixMillisFromTime(s.nodeRPCApp.Time.Now())))
}

// Eth_SendRawTransaction creates new message call transaction or a contract creation for signed transactions.
//   - signedTxData: The signed transaction data.
func (s RPCService) Eth_SendRawTransaction(signedTxData string) (proto.EthereumHash, error) {
	// TODO(nickeskov): what this method should return in case of error?
	const broadcastTimeout = 5 * time.Second

	data, err := proto.DecodeFromHexString(signedTxData)
	if err != nil {
		zap.S().Debugf("Eth_SendRawTransaction: failed to decode ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}

	// TODO(nickeskov): check max payload size

	var tx proto.EthereumTransaction
	err = tx.DecodeCanonical(data)
	if err != nil {
		zap.S().Debugf("Eth_SendRawTransaction: failed to unmarshal rlp encoded ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}

	txID, err := tx.GetID(s.nodeRPCApp.Scheme)
	if err != nil {
		zap.S().Errorf("Eth_SendRawTransaction: failed to get ID of ethereum transaction: %v", err)
		return proto.EthereumHash{}, err
	}
	ethTxID := proto.BytesToEthereumHash(txID)
	to := tx.To()
	from, err := tx.From()
	if err != nil {
		zap.S().Debugf(
			"Eth_SendRawTransaction: failed to get sender of ethereum transaction (ethTxID=%q, to=%q): %v",
			ethTxID.String(), to.String(), err,
		)
		return proto.EthereumHash{}, err
	}

	respCh := make(chan error, 1)
	// TODO(nickeskov): add context?
	s.nodeRPCApp.InternalChannel <- messages.NewBroadcastTransaction(respCh, &tx)

	timer := time.NewTimer(broadcastTimeout)
	select {
	case <-timer.C:
		zap.S().Errorf(
			"Eth_SendRawTransaction: timeout waiting response from internal FSM for ethereum tx (ethTxID=%q, to=%q, from=%q)",
			ethTxID.String(), to.String(), from.String(),
		)
		return proto.EthereumHash{}, errors.New("timeout waiting response from internal FSM")
	case err := <-respCh:
		if !timer.Stop() {
			<-timer.C
		}
		if err != nil {
			zap.S().Debugf("Eth_SendRawTransaction: error from internal FSM for ethereum tx (ethTxID=%q, to=%q, from=%q): %v",
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

func (s RPCService) Eth_GetTransactionReceipt(ethTxID proto.EthereumHash) (*GetTransactionReceiptResponse, error) {
	txID := crypto.Digest(ethTxID)
	tx, txIsFailed, err := s.nodeRPCApp.State.TransactionByIDWithStatus(txID.Bytes())
	if state.IsNotFound(err) {
		zap.S().Debugf("Eth_GetTransactionReceipt: transaction with ID=%q or ethID=%q cannot be found",
			txID, ethTxID,
		)
		return nil, errors.Errorf("transaction with ethID=%q is not found", ethTxID)
	}
	ethTx, ok := tx.(*proto.EthereumTransaction)
	if !ok {
		zap.S().Debugf(
			"Eth_GetTransactionReceipt: transaction with ID=%q or ethID=%q is not 'EthereumTransaction'",
			txID, ethTxID,
		)
		// according to the scala node implementation
		return nil, nil
	}

	to := ethTx.To()
	from, err := ethTx.From()
	if err != nil {
		zap.S().Errorf(
			"Eth_GetTransactionReceipt: failed to get sender (from) for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return nil, errors.New("failed to get sender from tx")
	}

	blockHeight, err := s.nodeRPCApp.State.TransactionHeightByID(txID.Bytes())
	if err != nil {
		zap.S().Errorf(
			"Eth_GetTransactionReceipt: failed to get block height for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return nil, errors.New("failed to get blockNumber for transaction")
	}

	lastBlockHeader := s.nodeRPCApp.State.TopBlock()
	txStatus := "0x1"
	if txIsFailed {
		txStatus = "0x0"
	}
	gasLimit := uint64ToHexString(tx.GetFee())

	resp := &GetTransactionReceiptResponse{
		TransactionHash:   ethTxID,
		TransactionIndex:  "0x01",                                              // according to the scala node implementation
		BlockHash:         proto.EncodeToHexString(lastBlockHeader.ID.Bytes()), // should be always 32bytes
		BlockNumber:       uint64ToHexString(blockHeight),
		From:              from,
		To:                to,
		CumulativeGasUsed: gasLimit,
		GasUsed:           gasLimit,
		ContractAddress:   nil,
		Logs:              []string{},
		LogsBloom:         proto.EthereumHash{},
		Status:            txStatus,
	}
	return resp, nil
}

type GetTransactionByHashResponse struct {
	Hash             proto.EthereumHash      `json:"hash"`
	Nonce            string                  `json:"nonce"`
	BlockHash        string                  `json:"blockHash"`
	BlockNumber      string                  `json:"blockNumber"`
	TransactionIndex string                  `json:"transactionIndex"`
	From             proto.EthereumAddress   `json:"from"`
	To               *proto.EthereumAddress  `json:"to"`
	Value            string                  `json:"value"`
	GasPrice         string                  `json:"gasPrice"`
	Gas              string                  `json:"gas"`
	Input            string                  `json:"input"`
	V                string                  `json:"v"`
	StandardV        string                  `json:"standardV"`
	R                string                  `json:"r"`
	Raw              string                  `json:"raw"`
	PublicKey        proto.EthereumPublicKey `json:"publickey"`
}

func (s RPCService) Eth_GetTransactionByHash(ethTxID proto.EthereumHash) (*GetTransactionByHashResponse, error) {
	txID := crypto.Digest(ethTxID)
	tx, err := s.nodeRPCApp.State.TransactionByID(txID.Bytes())
	if state.IsNotFound(err) {
		zap.S().Debugf("Eth_GetTransactionByHash: transaction with ID=%q or ethID=%q cannot be found",
			txID, ethTxID,
		)
		return nil, errors.Errorf("transaction with ethID=%q is not found", ethTxID)
	}
	ethTx, ok := tx.(*proto.EthereumTransaction)
	if !ok {
		zap.S().Debugf(
			"Eth_GetTransactionByHash: transaction with ID=%q or ethID=%q is not 'EthereumTransaction'",
			txID, ethTxID,
		)
		// according to the scala node implementation
		return nil, nil
	}

	to := ethTx.To()
	fromPK, err := ethTx.FromPK()
	if err != nil {
		zap.S().Errorf(
			"Eth_GetTransactionByHash: failed to get sender (from) public key for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return nil, errors.New("failed to get sender from tx")
	}

	blockHeight, err := s.nodeRPCApp.State.TransactionHeightByID(txID.Bytes())
	if err != nil {
		zap.S().Errorf(
			"Eth_GetTransactionByHash: failed to get block height for tx with ID=%q or ethID=%q: %v",
			txID, ethTxID, err,
		)
		return nil, errors.New("failed to get blockNumber for transaction")
	}

	lastBlockHeader := s.nodeRPCApp.State.TopBlock()

	gasLimit := uint64ToHexString(tx.GetFee())

	resp := &GetTransactionByHashResponse{
		Hash:             ethTxID,
		Nonce:            "0x1",
		BlockHash:        proto.EncodeToHexString(lastBlockHeader.ID.Bytes()),
		BlockNumber:      uint64ToHexString(blockHeight),
		TransactionIndex: "0x1",
		From:             fromPK.EthereumAddress(),
		To:               to,
		Value:            "0x10",
		GasPrice:         gasLimit, // according to the scala node implementation
		Gas:              gasLimit,
		Input:            "0x20",
		V:                "0x30",
		StandardV:        "0x40",
		R:                "0x50",
		Raw:              "0x60",
		PublicKey:        *fromPK,
	}
	return resp, nil
}
