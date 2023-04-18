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
			senderAddress, err := proto.NewAddressFromPublicKey(scheme, transfer.SenderPK)
			if err != nil {
				//...
			}
			// TODO handle alias
			recipientAddress := transfer.Recipient.Address()
			assetBalanceSnapshotFromAmount, err := s.assetBalanceSnapshotTransfer(senderAddress, *recipientAddress, proto.AssetIDFromDigest(transfer.AmountAsset.ID), transfer.Amount)
			if err != nil {
				//...
			}
			snapshots = append(snapshots, assetBalanceSnapshotFromAmount)
		} else {
			senderAddress, err := proto.NewAddressFromPublicKey(scheme, transfer.SenderPK)
			if err != nil {
				//...
			}
			// TODO handle alias
			recipientAddress := transfer.Recipient.Address()

			wavesBalanceSnapshotFromAmount, err := s.wavesBalanceSnapshotTransfer(senderAddress, *recipientAddress, transfer.Amount)
			if err != nil {
				//...
			}
			snapshots = append(snapshots, wavesBalanceSnapshotFromAmount)
		}

		if transfer.FeeAsset.Present {

		} else {

		}

		// TODO merge different arrays of wavesBalances and assetBalances for the same addresses
	case proto.ReissueTransaction: // 5

	case proto.BurnTransaction: // 6

	case proto.ExchangeTransaction: // 7

	case proto.LeaseTransaction: // 8

	case proto.LeaseCancelTransaction: // 9

	case proto.CreateAliasTransaction: // 10

	case proto.MassTransferTransaction: // 11
	case proto.DataTransaction: // 12
	case proto.SetScriptTransaction: // 13
	case proto.SponsorshipTransaction: // 14
	case proto.SetAssetScriptTransaction: // 15
	case proto.InvokeScriptTransaction: // 16
	case proto.UpdateAssetInfoTransaction: // 17
	case proto.EthereumMetamaskTransaction: // 18
	}
	return snapshots, nil
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

func (s SnapshotManager) wavesBalanceSnapshotTransfer(sender proto.WavesAddress, recipient proto.WavesAddress, amount uint64) (*WavesBalancesSnapshot, error) {
	senderBalance, err := s.wavesBalanceByAddress(sender)
	if err != nil {
		//...
	}
	recipientBalance, err := s.wavesBalanceByAddress(recipient)
	if err != nil {
		//...
	}
	wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
		{address: &sender, balance: senderBalance - amount},
		{address: recipient, balance: recipientBalance + amount}},
	}
	return wavesBalancesSnapshot, nil
}

func (s SnapshotManager) assetBalanceSnapshotTransfer(sender proto.WavesAddress, recipient proto.WavesAddress, assetID proto.AssetID,
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