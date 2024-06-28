package l2

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/ybbus/jsonrpc/v3"
)

type EngineApiClient struct {
	rpcClient jsonrpc.RPCClient
}

type EngineApiOpts struct {
	Address string
	Port    string
}

func NewEngineApiClient(opts EngineApiOpts) *EngineApiClient {
	rpcClient := jsonrpc.NewClient("http://" + opts.Address + ":" + opts.Port)
	return &EngineApiClient{
		rpcClient: rpcClient,
	}
}

func (e *EngineApiClient) forkChoiceUpdate(ctx context.Context, hash proto.EthereumHash) error {
	state := ForkChoiceStateV1{
		HeadBlockHash:      hash,
		SafeBlockHash:      hash,
		FinalizedBlockHash: hash,
	}
	response := ForkChoiceResponse{}
	err := e.rpcClient.CallFor(ctx, &response, "engine_forkchoiceUpdatedV3", state, nil)
	if err != nil {
		return err
	}
	if (response.PayloadStatus.Status == "SYNCING" || response.PayloadStatus.Status == "VALID") && response.PayloadID == nil {
		return nil
	}
	if response.PayloadStatus.ValidationError != nil {
		return fmt.Errorf("payload validation error: %s", *response.PayloadStatus.ValidationError)
	}
	return fmt.Errorf("unexpected payload status %s", response.PayloadStatus.Status)
}

func (e *EngineApiClient) forkChoiceUpdateWithPayloadId(ctx context.Context,
	lastBlockHash proto.EthereumHash,
	unixEpochSeconds uint64,
	suggestedFeeRecipient *proto.EthereumAddress,
	withdrawals []Withdrawal,
) (PayloadID, error) {
	state := ForkChoiceStateV1{
		HeadBlockHash:      lastBlockHash,
		SafeBlockHash:      lastBlockHash,
		FinalizedBlockHash: lastBlockHash,
	}
	feeRecipient, err := emptyFeeRecipient()
	if err != nil {
		return PayloadID{}, err
	}
	if suggestedFeeRecipient != nil {
		feeRecipient = *suggestedFeeRecipient
	}
	emptyPrevRandao, err := emptyPrevRandaoEthHash()
	if err != nil {
		return PayloadID{}, err
	}
	emptyBeaconRootHash, err := emptyRootHash()
	if err != nil {
		return PayloadID{}, err
	}
	attr := PayloadAttributes{
		Timestamp:             Quantity(unixEpochSeconds),
		Random:                emptyPrevRandao,
		SuggestedFeeRecipient: feeRecipient,
		Withdrawals:           withdrawals,
		BeaconRoot:            &emptyBeaconRootHash,
	}
	response := ForkChoiceResponse{}
	err = e.rpcClient.CallFor(ctx, &response, "engine_forkchoiceUpdatedV3", state, attr)
	if err != nil {
		return PayloadID{}, err
	}
	if response.PayloadStatus.Status == "VALID" && response.PayloadID != nil {
		return *response.PayloadID, nil
	}
	if response.PayloadStatus.ValidationError != nil {
		return PayloadID{}, fmt.Errorf("payload validation error: %s", *response.PayloadStatus.ValidationError)
	}
	return PayloadID{}, fmt.Errorf("unexpected payload status for %s: %s", lastBlockHash, response.PayloadStatus.Status)
}

func (e *EngineApiClient) GetPayload(ctx context.Context, id PayloadID) (ExecutablePayloadV3, error) {
	response := ExecutionPayloadEnvelope{}
	err := e.rpcClient.CallFor(ctx, &response, "engine_getPayloadV3", id)
	if err != nil {
		return ExecutablePayloadV3{}, err
	}
	return *response.ExecutionPayload, nil
}

func (e *EngineApiClient) ApplyNewPayload(ctx context.Context, payload ExecutablePayloadV3) (proto.EthereumHash, error) {
	emptyArray := make([]string, 0)
	emptyBeaconRootHash, err := emptyRootHash()
	if err != nil {
		return proto.EthereumHash{}, err
	}
	response := PayloadStatusV1{}
	err = e.rpcClient.CallFor(ctx, &response, "engine_newPayloadV3", payload, emptyArray, emptyBeaconRootHash)
	if err != nil {
		return proto.EthereumHash{}, err
	}
	if response.ValidationError != nil {
		return proto.EthereumHash{}, fmt.Errorf("payload validation error: %s", *response.ValidationError)
	}
	if response.Status == "VALID" && response.LatestValidHash != nil {
		return *response.LatestValidHash, nil
	}
	if response.LatestValidHash == nil {
		return proto.EthereumHash{}, fmt.Errorf("latest valid hash is not defined")
	}
	return proto.EthereumHash{}, fmt.Errorf("unexpected payload status for %s", response.Status)
}

func (e *EngineApiClient) GetPayloadByHash(ctx context.Context, hash proto.EthereumHash) ([]ExecutionPayloadBodyV1, error) {
	var response []ExecutionPayloadBodyV1
	err := e.rpcClient.CallFor(ctx, &response, "engine_getPayloadBodiesByHashV1", [][]proto.EthereumHash{{hash}})
	if err != nil {
		return nil, err
	}
	return response, nil
}
