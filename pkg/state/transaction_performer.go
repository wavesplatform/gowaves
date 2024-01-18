package state

import "github.com/wavesplatform/gowaves/pkg/proto"

// transactionPerformer is a temporary interface for compatibility with legacy code.
// It builds txSnapshot for each proto.Transaction type.
type transactionPerformer interface {
	performGenesis(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performPayment(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performTransferWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performTransferWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performIssueWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performIssueWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performReissueWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performReissueWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performBurnWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performBurnWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performExchange(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performLeaseWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performLeaseWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performLeaseCancelWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performLeaseCancelWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performCreateAliasWithSig(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performCreateAliasWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performMassTransferWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performDataWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performSponsorshipWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performSetScriptWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performSetAssetScriptWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performInvokeScriptWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performInvokeExpressionWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performEthereumTransactionWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)
	performUpdateAssetInfoWithProofs(proto.Transaction, *performerInfo,
		*invocationResult, []balanceChanges) (txSnapshot, error)

	createInitialBlockSnapshot(minerAndRewardChanges []balanceChanges) (txSnapshot, error)
	// used for creating snapshots from failed changes
	generateBalancesSnapshot(balanceChanges []balanceChanges) (txSnapshot, error)
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
