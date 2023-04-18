package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// TODO validation of the transactions should be before or after?
func (s *SnapshotManager) TxSnapshotFromTx(tx proto.Transaction, scheme proto.Scheme) (TransactionSnapshot, error) {
	var snapshots []AtomicSnapshot

	switch tx.GetTypeInfo().Type {
	case proto.GenesisTransaction: // 1
		genesisTx, ok := tx.(*proto.Genesis)
		if !ok {
			// ...
		}
		wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
			{address: &genesisTx.Recipient, balance: genesisTx.Amount}},
		}
		snapshots = append(snapshots, wavesBalancesSnapshot)
		return snapshots, nil
	case proto.PaymentTransaction: // 2
		paymentTx, ok := tx.(*proto.Payment)
		if !ok {
			// ...
		}
		senderAddress, err := proto.NewAddressFromPublicKey(scheme, paymentTx.SenderPK)
		if err != nil {
			//...
		}
		senderBalance, err := s.wavesBalanceByAddress(senderAddress)
		if err != nil {
			//...
		}
		recipientBalance, err := s.wavesBalanceByAddress(paymentTx.Recipient)
		if err != nil {
			//...
		}

		// TODO validate balances

		wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
			{address: &senderAddress, balance: senderBalance - paymentTx.Amount - paymentTx.Fee},
			{address: &paymentTx.Recipient, balance: recipientBalance + paymentTx.Amount}},
		}
		snapshots = append(snapshots, wavesBalancesSnapshot)
		return snapshots, nil
	case proto.IssueTransaction: // 3
		var issue proto.Issue
		switch i := tx.(type) {
		case *proto.IssueWithSig:
			issue = i.Issue
		case *proto.IssueWithProofs:
			issue = i.Issue
		default:
			// return err
		}
		senderAddress, err := proto.NewAddressFromPublicKey(scheme, issue.SenderPK)
		if err != nil {
			//...
		}
		senderBalance, err := s.wavesBalanceByAddress(senderAddress)
		if err != nil {
			//...
		}
		// TODO validate balances
		wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
			{address: &senderAddress, balance: senderBalance - issue.Fee}},
		}

		assetsSnapshot := &AssetDescriptionSnapshot{
			// TODO generate asset id and fill change height
			assetID:          proto.AssetID{},
			assetName:        &issue.Name,
			assetDescription: issue.Description,
			changeHeight:     0,
		}

		snapshots = append(snapshots, wavesBalancesSnapshot, assetsSnapshot)
		return snapshots, nil
	case proto.TransferTransaction: // 4
		var transfer proto.Transfer
		switch t := tx.(type) {
		case *proto.TransferWithSig:
			transfer = t.Transfer
		case *proto.TransferWithProofs:
			transfer = t.Transfer
		default:
			// return err
		}
		if transfer.AmountAsset.Present {

		} else {
			senderAddress, err := proto.NewAddressFromPublicKey(scheme, transfer.SenderPK)
			if err != nil {
				//...
			}
			// TODO handle alias
			recipientAddress := transfer.Recipient.Address()
			senderBalance, err := s.wavesBalanceByAddress(senderAddress)
			if err != nil {
				//...
			}
			recipientBalance, err := s.wavesBalanceByAddress(*recipientAddress)
			if err != nil {
				//...
			}
			wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
				{address: &senderAddress, balance: senderBalance - transfer.Amount - transfer.Fee},
				{address: recipientAddress, balance: recipientBalance + transfer.Amount}},
			}
			snapshots = append(snapshots, wavesBalancesSnapshot)
		}

		if transfer.FeeAsset.Present {

		} else {

		}

		// TODO merge different arrays of wavesBalances and assetBalances for the same addresses
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

//func (s *SnapshotManager) balanceByPublicKey(pk crypto.PublicKey, scheme proto.Scheme) uint64 {
//	address, err := proto.NewAddressFromPublicKey(scheme, pk)
//	if err != nil {
//		//...
//	}
//	balanceProfile, err := s.stor.balances.wavesBalance(address.ID())
//	if err != nil {
//		//...
//	}
//	return balanceProfile.balance
//}

func (s SnapshotManager) wavesBalanceSnapshotAmountFee(sender proto.WavesAddress, recipient proto.WavesAddress,
	amount uint64, fee uint64) (*WavesBalancesSnapshot, error) {
	senderBalance, err := s.wavesBalanceByAddress(sender)
	if err != nil {
		//...
	}
	recipientBalance, err := s.wavesBalanceByAddress(recipient)
	if err != nil {
		//...
	}
	wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
		{address: &sender, balance: senderBalance - amount - fee},
		{address: recipient, balance: recipientBalance + amount}},
	}
	return wavesBalancesSnapshot, nil
}

func (s SnapshotManager) wavesBalanceSnapshotFee(sender proto.WavesAddress, fee uint64) (*WavesBalancesSnapshot, error) {
	senderBalance, err := s.wavesBalanceByAddress(sender)
	if err != nil {
		//...
	}
	wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
		{address: &sender, balance: senderBalance - fee}},
	}
	return wavesBalancesSnapshot, nil
}

func (s SnapshotManager) assetBalanceSnapshotAmount(sender proto.WavesAddress, recipient proto.WavesAddress, assetID proto.AssetID,
	amount uint64) (*AssetBalancesSnapshot, error) {
	senderAssetBalance, err := s.stor.balances.assetBalance(sender.ID(), assetID)
	if err != nil {
		//...
	}
	recipientAssetBalance, err := s.stor.balances.assetBalance(recipient.ID(), assetID)
	if err != nil {
		//...
	}
	wavesBalancesSnapshot := &AssetBalancesSnapshot{assetBalances: []balanceAsset{
		{address: &sender, balance: senderAssetBalance - amount},
		{address: recipient, balance: recipientAssetBalance + amount}},
	}
	return wavesBalancesSnapshot, nil
}

func (s *SnapshotManager) wavesBalanceByAddress(address proto.WavesAddress) (uint64, error) {
	recipientWavesBalanceProfile, err := s.stor.balances.wavesBalance(address.ID())
	if err != nil {
		//...
	}
	return recipientWavesBalanceProfile.balance, nil
}