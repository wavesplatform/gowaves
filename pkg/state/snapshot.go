package state

import "github.com/wavesplatform/gowaves/pkg/proto"

func (s *SnapshotManager) TxSnapshotFromTx(tx proto.Transaction, scheme proto.Scheme) (TransactionSnapshot, error) {
	var snapshots []AtomicSnapshot

	switch tx.GetTypeInfo().Type {
	case proto.GenesisTransaction: // 1
		out = &Genesis{}
	case proto.PaymentTransaction: // 2
		paymentTx := tx.(*proto.Payment)
		senderAddress, err := proto.NewAddressFromPublicKey(scheme, paymentTx.SenderPK)
		if err != nil {
			//...
		}
		senderBalanceProfile, err := s.stor.balances.wavesBalance(senderAddress.ID())
		recipientBalanceProfile, err := s.stor.balances.wavesBalance(paymentTx.Recipient.ID())

		wavesBalanaceSnapshot := WavesBalancesSnapshot{wavesBalances: balanceWaves{address: *senderAddress}}

	case proto.IssueTransaction: // 3
		if t.Version >= 2 {
			out = &IssueWithProofs{}
		} else {
			out = &IssueWithSig{}
		}
	case proto.TransferTransaction: // 4
		if t.Version >= 2 {
			out = &TransferWithProofs{}
		} else {
			out = &TransferWithSig{}
		}
	case proto.ReissueTransaction: // 5
		if t.Version >= 2 {
			out = &ReissueWithProofs{}
		} else {
			out = &ReissueWithSig{}
		}
	case proto.BurnTransaction: // 6
		if t.Version >= 2 {
			out = &BurnWithProofs{}
		} else {
			out = &BurnWithSig{}
		}
	case proto.ExchangeTransaction: // 7
		if t.Version >= 2 {
			out = &ExchangeWithProofs{}
		} else {
			out = &ExchangeWithSig{}
		}
	case proto.LeaseTransaction: // 8
		if t.Version >= 2 {
			out = &LeaseWithProofs{}
		} else {
			out = &LeaseWithSig{}
		}
	case proto.LeaseCancelTransaction: // 9
		if t.Version >= 2 {
			out = &LeaseCancelWithProofs{}
		} else {
			out = &LeaseCancelWithSig{}
		}
	case proto.CreateAliasTransaction: // 10
		if t.Version >= 2 {
			out = &CreateAliasWithProofs{}
		} else {
			out = &CreateAliasWithSig{}
		}
	case proto.MassTransferTransaction: // 11
		out = &MassTransferWithProofs{}
	case proto.DataTransaction: // 12
		out = &DataWithProofs{}
	case proto.SetScriptTransaction: // 13
		out = &SetScriptWithProofs{}
	case proto.SponsorshipTransaction: // 14
		out = &SponsorshipWithProofs{}
	case proto.SetAssetScriptTransaction: // 15
		out = &SetAssetScriptWithProofs{}
	case proto.InvokeScriptTransaction: // 16
		out = &InvokeScriptWithProofs{}
	case proto.UpdateAssetInfoTransaction: // 17
		out = &UpdateAssetInfoWithProofs{}
	case proto.EthereumMetamaskTransaction: // 18
		out = &EthereumTransaction{}
	}
	return snapshots
}
