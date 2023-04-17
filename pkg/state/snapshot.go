package state

import "github.com/wavesplatform/gowaves/pkg/proto"

func (SnapshotManager) TxSnapshotFromTx(tx proto.Transaction) TransactionSnapshot {
	var snapshots []AtomicSnapshot

	switch tx.GetTypeInfo().Type {
	case proto.GenesisTransaction: // 1
		out = &GenesisTransactionInfo{}
	case proto.PaymentTransaction: // 2
		out = &PaymentTransactionInfo{}
	case proto.IssueTransaction: // 3
		if t.Version >= 2 {
			out = &IssueWithProofsTransactionInfo{}
		} else {
			out = &IssueWithSigTransactionInfo{}
		}
	case proto.TransferTransaction: // 4
		if t.Version >= 2 {
			out = &TransferWithProofsTransactionInfo{}
		} else {
			out = &TransferWithSigTransactionInfo{}
		}
	case proto.ReissueTransaction: // 5
		if t.Version >= 2 {
			out = &ReissueWithProofsTransactionInfo{}
		} else {
			out = &ReissueWithSigTransactionInfo{}
		}
	case proto.BurnTransaction: // 6
		if t.Version >= 2 {
			out = &BurnWithProofsTransactionInfo{}
		} else {
			out = &BurnWithSigTransactionInfo{}
		}
	case proto.ExchangeTransaction: // 7
		if t.Version >= 2 {
			out = &ExchangeWithProofsTransactionInfo{}
		} else {
			out = &ExchangeWithSigTransactionInfo{}
		}
	case proto.LeaseTransaction: // 8
		if t.Version >= 2 {
			out = &LeaseWithProofsTransactionInfo{}
		} else {
			out = &LeaseWithSigTransactionInfo{}
		}
	case proto.LeaseCancelTransaction: // 9
		if t.Version >= 2 {
			out = &LeaseCancelWithProofsTransactionInfo{}
		} else {
			out = &LeaseCancelWithSigTransactionInfo{}
		}
	case proto.CreateAliasTransaction: // 10
		if t.Version >= 2 {
			out = &CreateAliasWithProofsTransactionInfo{}
		} else {
			out = &CreateAliasWithSigTransactionInfo{}
		}
	case proto.MassTransferTransaction: // 11
		out = &MassTransferTransactionInfo{}
	case proto.DataTransaction: // 12
		out = &DataTransactionInfo{}
	case proto.SetScriptTransaction: // 13
		out = &SetScriptTransactionInfo{}
	case proto.SponsorshipTransaction: // 14
		out = &SponsorshipTransactionInfo{}
	case proto.SetAssetScriptTransaction: // 15
		out = &SetAssetScriptTransactionInfo{}
	case proto.InvokeScriptTransaction: // 16
		out = &InvokeScriptTransactionInfo{}
	case proto.UpdateAssetInfoTransaction: // 17
		out = &UpdateAssetInfoTransactionInfo{}
	case proto.EthereumMetamaskTransaction: // 18
		out = &EthereumTransactionInfo{}
	}
	return snapshots
}
