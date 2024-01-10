package state

import "github.com/wavesplatform/gowaves/pkg/proto"

// transactionPerformer is a temporary interface for compatibility with legacy code.
// It builds txSnapshot for each proto.Transaction type.
type transactionPerformer interface {
	performGenesis(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performPayment(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performTransferWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performTransferWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performIssueWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performIssueWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performReissueWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performReissueWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performBurnWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performBurnWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performExchange(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performLeaseWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performLeaseWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performLeaseCancelWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performLeaseCancelWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performCreateAliasWithSig(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performCreateAliasWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performMassTransferWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performDataWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performSponsorshipWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performSetScriptWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performSetAssetScriptWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performInvokeScriptWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performInvokeExpressionWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performEthereumTransactionWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
	performUpdateAssetInfoWithProofs(proto.Transaction, *performerInfo, *invocationResult, txDiff) (txSnapshot, error)
}

type performerInfo struct {
	blockchainHeight    proto.Height
	blockID             proto.BlockID
	currentMinerAddress proto.WavesAddress
	checkerData         txCheckerData
}

func (i *performerInfo) blockHeight() proto.Height { return i.blockchainHeight + 1 }

func newPerformerInfo(
	blockchainHeight proto.Height,
	blockID proto.BlockID,
	currentMinerAddress proto.WavesAddress,
	checkerData txCheckerData,
) *performerInfo {
	return &performerInfo{ // all fields must be initialized
		blockchainHeight,
		blockID,
		currentMinerAddress,
		checkerData,
	}
}
