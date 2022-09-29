// Code generated by zenrpc; DO NOT EDIT.

package metamask

import (
	"context"
	"encoding/json"

	"github.com/semrush/zenrpc/v2"
	"github.com/semrush/zenrpc/v2/smd"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

var RPC = struct {
	RPCService struct{ Eth_BlockNumber, Net_Version, Eth_ChainId, Eth_GetBalance, Eth_GetBlockByNumber, Eth_GasPrice, Eth_EstimateGas, Eth_Call, Eth_GetCode, Eth_GetTransactionCount, Eth_SendRawTransaction, Eth_GetTransactionReceipt string }
}{
	RPCService: struct{ Eth_BlockNumber, Net_Version, Eth_ChainId, Eth_GetBalance, Eth_GetBlockByNumber, Eth_GasPrice, Eth_EstimateGas, Eth_Call, Eth_GetCode, Eth_GetTransactionCount, Eth_SendRawTransaction, Eth_GetTransactionReceipt string }{
		Eth_BlockNumber:           "eth_blocknumber",
		Net_Version:               "net_version",
		Eth_ChainId:               "eth_chainid",
		Eth_GetBalance:            "eth_getbalance",
		Eth_GetBlockByNumber:      "eth_getblockbynumber",
		Eth_GasPrice:              "eth_gasprice",
		Eth_EstimateGas:           "eth_estimategas",
		Eth_Call:                  "eth_call",
		Eth_GetCode:               "eth_getcode",
		Eth_GetTransactionCount:   "eth_gettransactioncount",
		Eth_SendRawTransaction:    "eth_sendrawtransaction",
		Eth_GetTransactionReceipt: "eth_gettransactionreceipt",
	},
}

func (RPCService) SMD() smd.ServiceInfo {
	return smd.ServiceInfo{
		Description: ``,
		Methods: map[string]smd.Service{
			"Eth_BlockNumber": {
				Description: `Eth_BlockNumber returns the number of most recent block`,
				Parameters:  []smd.JSONSchema{},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Net_Version": {
				Description: `Net_Version returns the current network id`,
				Parameters:  []smd.JSONSchema{},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_ChainId": {
				Description: `Eth_ChainId returns the chain id`,
				Parameters:  []smd.JSONSchema{},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_GetBalance": {
				Description: `Eth_GetBalance returns the balance of the account of given address
- address: 20 Bytes - address to check for balance
- block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending" */`,
				Parameters: []smd.JSONSchema{
					{
						Name:        "address",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
					{
						Name:        "blockOrTag",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_GetBlockByNumber": {
				Description: `Eth_GetBlockByNumber returns information about a block by block number.
- block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"
- filterTxObj: if true it returns the full transaction objects, if false only the hashes of the transactions */`,
				Parameters: []smd.JSONSchema{
					{
						Name:        "blockOrTag",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
					{
						Name:        "filterTxObj",
						Optional:    false,
						Description: ``,
						Type:        smd.Boolean,
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.Object,
					Properties: map[string]smd.Property{
						"number": {
							Description: ``,
							Type:        smd.String,
						},
					},
				},
			},
			"Eth_GasPrice": {
				Description: `Eth_GasPrice returns the current price per gas in wei`,
				Parameters:  []smd.JSONSchema{},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_EstimateGas": {
				Description: ``,
				Parameters: []smd.JSONSchema{
					{
						Name:        "req",
						Optional:    false,
						Description: ``,
						Type:        smd.Object,
						Properties: map[string]smd.Property{
							"to": {
								Description: ``,
								Ref:         "#/definitions/proto.EthereumAddress",
								Type:        smd.Object,
							},
							"value": {
								Description: ``,
								Type:        smd.String,
							},
							"data": {
								Description: ``,
								Type:        smd.String,
							},
						},
						Definitions: map[string]smd.Definition{
							"proto.EthereumAddress": {
								Type:       "object",
								Properties: map[string]smd.Property{},
							},
						},
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_Call": {
				Description: ``,
				Parameters: []smd.JSONSchema{
					{
						Name:        "params",
						Optional:    false,
						Description: ``,
						Type:        smd.Object,
						Properties: map[string]smd.Property{
							"to": {
								Description: ``,
								Ref:         "#/definitions/proto.EthereumAddress",
								Type:        smd.Object,
							},
							"data": {
								Description: ``,
								Type:        smd.String,
							},
						},
						Definitions: map[string]smd.Definition{
							"proto.EthereumAddress": {
								Type:       "object",
								Properties: map[string]smd.Property{},
							},
						},
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_GetCode": {
				Description: `Eth_GetCode returns the compiled smart contract code, if any, at a given address.
- address: 20 Bytes - address to check for balance
- block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"`,
				Parameters: []smd.JSONSchema{
					{
						Name:        "address",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
					{
						Name:        "blockOrTag",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_GetTransactionCount": {
				Description: `Eth_GetTransactionCount returns the number of transactions sent from an address.
- address: 20 Bytes - address to check for balance
- block: QUANTITY|TAG - integer block number, or the string "latest", "earliest" or "pending"`,
				Parameters: []smd.JSONSchema{
					{
						Name:        "address",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
					{
						Name:        "blockOrTag",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.String,
				},
			},
			"Eth_SendRawTransaction": {
				Description: `Eth_SendRawTransaction creates new message call transaction or a contract creation for signed transactions.
- signedTxData: The signed transaction data.`,
				Parameters: []smd.JSONSchema{
					{
						Name:        "signedTxData",
						Optional:    false,
						Description: ``,
						Type:        smd.String,
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    false,
					Type:        smd.Object,
					Properties:  map[string]smd.Property{},
				},
			},
			"Eth_GetTransactionReceipt": {
				Description: ``,
				Parameters: []smd.JSONSchema{
					{
						Name:        "ethTxID",
						Optional:    false,
						Description: ``,
						Type:        smd.Object,
						Properties:  map[string]smd.Property{},
					},
				},
				Returns: smd.JSONSchema{
					Description: ``,
					Optional:    true,
					Type:        smd.Object,
					Properties: map[string]smd.Property{
						"transactionHash": {
							Description: ``,
							Ref:         "#/definitions/proto.EthereumHash",
							Type:        smd.Object,
						},
						"transactionIndex": {
							Description: ``,
							Type:        smd.String,
						},
						"blockHash": {
							Description: ``,
							Type:        smd.String,
						},
						"blockNumber": {
							Description: ``,
							Type:        smd.String,
						},
						"from": {
							Description: ``,
							Ref:         "#/definitions/proto.EthereumAddress",
							Type:        smd.Object,
						},
						"to": {
							Description: ``,
							Ref:         "#/definitions/proto.EthereumAddress",
							Type:        smd.Object,
						},
						"cumulativeGasUsed": {
							Description: ``,
							Type:        smd.String,
						},
						"gasUsed": {
							Description: ``,
							Type:        smd.String,
						},
						"contractAddress": {
							Description: ``,
							Ref:         "#/definitions/proto.EthereumAddress",
							Type:        smd.Object,
						},
						"logs": {
							Description: ``,
							Type:        smd.Array,
							Items: map[string]string{
								"type": smd.String,
							},
						},
						"logsBloom": {
							Description: ``,
							Ref:         "#/definitions/proto.EthereumHash",
							Type:        smd.Object,
						},
						"status": {
							Description: ``,
							Type:        smd.String,
						},
					},
					Definitions: map[string]smd.Definition{
						"proto.EthereumHash": {
							Type:       "object",
							Properties: map[string]smd.Property{},
						},
						"proto.EthereumAddress": {
							Type:       "object",
							Properties: map[string]smd.Property{},
						},
					},
				},
			},
		},
	}
}

// Invoke is as generated code from zenrpc cmd
func (s RPCService) Invoke(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
	resp := zenrpc.Response{}
	var err error

	switch method {
	case RPC.RPCService.Eth_BlockNumber:
		resp.Set(s.Eth_BlockNumber())

	case RPC.RPCService.Net_Version:
		resp.Set(s.Net_Version())

	case RPC.RPCService.Eth_ChainId:
		resp.Set(s.Eth_ChainId())

	case RPC.RPCService.Eth_GetBalance:
		var args = struct {
			Address    string `json:"address"`
			BlockOrTag string `json:"blockOrTag"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"address", "blockOrTag"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_GetBalance(args.Address, args.BlockOrTag))

	case RPC.RPCService.Eth_GetBlockByNumber:
		var args = struct {
			BlockOrTag  string `json:"blockOrTag"`
			FilterTxObj bool   `json:"filterTxObj"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"blockOrTag", "filterTxObj"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_GetBlockByNumber(args.BlockOrTag, args.FilterTxObj))

	case RPC.RPCService.Eth_GasPrice:
		resp.Set(s.Eth_GasPrice())

	case RPC.RPCService.Eth_EstimateGas:
		var args = struct {
			Req estimateGasRequest `json:"req"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"req"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_EstimateGas(args.Req))

	case RPC.RPCService.Eth_Call:
		var args = struct {
			Params ethCallParams `json:"params"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"params"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_Call(args.Params))

	case RPC.RPCService.Eth_GetCode:
		var args = struct {
			Address    string `json:"address"`
			BlockOrTag string `json:"blockOrTag"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"address", "blockOrTag"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_GetCode(args.Address, args.BlockOrTag))

	case RPC.RPCService.Eth_GetTransactionCount:
		var args = struct {
			Address    string `json:"address"`
			BlockOrTag string `json:"blockOrTag"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"address", "blockOrTag"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_GetTransactionCount(args.Address, args.BlockOrTag))

	case RPC.RPCService.Eth_SendRawTransaction:
		var args = struct {
			SignedTxData string `json:"signedTxData"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"signedTxData"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_SendRawTransaction(args.SignedTxData))

	case RPC.RPCService.Eth_GetTransactionReceipt:
		var args = struct {
			EthTxID proto.EthereumHash `json:"ethTxID"`
		}{}

		if zenrpc.IsArray(params) {
			if params, err = zenrpc.ConvertToObject([]string{"ethTxID"}, params); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		if len(params) > 0 {
			if err := json.Unmarshal(params, &args); err != nil {
				return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "", err.Error())
			}
		}

		resp.Set(s.Eth_GetTransactionReceipt(args.EthTxID))

	default:
		resp = zenrpc.NewResponseError(nil, zenrpc.MethodNotFound, "", nil)
	}

	return resp
}
