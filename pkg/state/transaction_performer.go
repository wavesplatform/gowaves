package state

import "github.com/wavesplatform/gowaves/pkg/proto"

// transactionPerformer is a temporary interface for compatibility with legacy code.
// It builds txSnapshot for each proto.Transaction type.
type transactionPerformer interface {
	performGenesis(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performPayment(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performTransferWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performTransferWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performIssueWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performIssueWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performReissueWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performReissueWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performBurnWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performBurnWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performExchange(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performLeaseWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performLeaseWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performLeaseCancelWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performLeaseCancelWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performCreateAliasWithSig(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performCreateAliasWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performMassTransferWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performDataWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performSponsorshipWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performSetScriptWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performSetAssetScriptWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performInvokeScriptWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performInvokeExpressionWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performEthereumTransactionWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)
	performUpdateAssetInfoWithProofs(proto.Transaction, *performerInfo, []balanceChanges) (txSnapshot, error)

	createInitialBlockSnapshot(minerAndRewardChanges []balanceChanges) (txSnapshot, error)
	// used for creating snapshots from failed changes
	generateBalancesSnapshot(balanceChanges []balanceChanges, txIsSuccessfulInvoke bool) (txSnapshot, error)
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
