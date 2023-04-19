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
		senderAddress, err := proto.NewAddressFromPublicKey(scheme, transfer.SenderPK)
		if err != nil {
			//...
		}
		// TODO handle alias
		recipientAddress := transfer.Recipient.Address()
		assetBalancesSnapshot, wavesBalancesSnapshot, err := s.optionalAssetBalanceSnapshotTransfer(senderAddress, *recipientAddress, transfer.AmountAsset, transfer.Amount, transfer.FeeAsset, transfer.Fee)
		if err != nil {
			//...
		}
		if assetBalancesSnapshot != nil {
			snapshots = append(snapshots, assetBalancesSnapshot)
		}
		if wavesBalancesSnapshot != nil {
			snapshots = append(snapshots, wavesBalancesSnapshot)
		}
		return snapshots, nil
		// TODO should be a snapshot about the quantity
	case proto.ReissueTransaction: // 5
		var reissue proto.Reissue
		switch t := tx.(type) {
		case *proto.ReissueWithSig:
			reissue = t.Reissue
		case *proto.ReissueWithProofs:
			reissue = t.Reissue
		default:
			// return err
		}
		assetInfo, err := s.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(reissue.AssetID))
		if err != nil {
			// ...
		}
		senderAddress, err := proto.NewAddressFromPublicKey(scheme, reissue.SenderPK)
		if err != nil {
			//...
		}
		senderBalance, err := s.wavesBalanceByAddress(senderAddress)
		if err != nil {
			//...
		}
		// TODO validate balances
		wavesBalancesSnapshot := &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
			{address: &senderAddress, balance: senderBalance - reissue.Fee}},
		}
		snapshots = append(snapshots, wavesBalancesSnapshot)
		// TODO can you make an asset reissuable again?
		if assetInfo.reissuable != reissue.Reissuable {
			assetReissuabilitySnapshot := &AssetReissuabilitySnapshot{
				assetID:      proto.AssetIDFromDigest(reissue.AssetID),
				isReissuable: reissue.Reissuable,
			}
			snapshots = append(snapshots, assetReissuabilitySnapshot)
		}
		return snapshots, nil
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

func (s SnapshotManager) optionalAssetBalanceSnapshotTransfer(sender proto.WavesAddress, recipient proto.WavesAddress, amountAsset proto.OptionalAsset,
	amount uint64, feeAsset proto.OptionalAsset, fee uint64) (*AssetBalancesSnapshot, *WavesBalancesSnapshot, error) {

	if amountAsset.Present {
		senderAssetBalance, err := s.stor.balances.assetBalance(sender.ID(), proto.AssetIDFromDigest(amountAsset.ID))
		if err != nil {
			//...
		}
		recipientAssetBalance, err := s.stor.balances.assetBalance(recipient.ID(), proto.AssetIDFromDigest(amountAsset.ID))
		if err != nil {
			//...
		}
		if feeAsset.Present {
			return &AssetBalancesSnapshot{assetBalances: []balanceAsset{
				{address: sender, assetID: proto.AssetIDFromDigest(feeAsset.ID), balance: senderAssetBalance - amount - fee},
				{address: recipient, assetID: proto.AssetIDFromDigest(feeAsset.ID), balance: recipientAssetBalance + amount},
			}}, nil, nil
		}

		senderWavesBalance, err := s.stor.balances.wavesBalance(sender.ID())
		if err != nil {
			//...
		}
		return &AssetBalancesSnapshot{assetBalances: []balanceAsset{
				{address: sender, assetID: proto.AssetIDFromDigest(feeAsset.ID), balance: senderAssetBalance - amount},
				{address: recipient, assetID: proto.AssetIDFromDigest(feeAsset.ID), balance: recipientAssetBalance + amount},
			}}, &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
				{address: sender, balance: senderWavesBalance.balance - fee},
			}}, nil
	}

	senderWavesBalance, err := s.stor.balances.wavesBalance(sender.ID())
	if err != nil {
		//...
	}
	recipientWavesBalance, err := s.stor.balances.wavesBalance(recipient.ID())
	if err != nil {
		//...
	}
	if feeAsset.Present {
		senderAssetBalance, err := s.stor.balances.assetBalance(sender.ID(), proto.AssetIDFromDigest(amountAsset.ID))
		if err != nil {
			//...
		}
		return &AssetBalancesSnapshot{assetBalances: []balanceAsset{
				{address: sender, assetID: proto.AssetIDFromDigest(feeAsset.ID), balance: senderAssetBalance - fee},
			}}, &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
				{address: sender, balance: senderWavesBalance.balance - amount},
				{address: recipient, balance: recipientWavesBalance.balance + amount},
			}}, nil
	}

	return nil, &WavesBalancesSnapshot{wavesBalances: []balanceWaves{
		{address: sender, balance: senderWavesBalance.balance - fee - amount},
		{address: recipient, balance: senderWavesBalance.balance + amount},
	}}, nil
}

func (s *SnapshotManager) wavesBalanceByAddress(address proto.WavesAddress) (uint64, error) {
	recipientWavesBalanceProfile, err := s.stor.balances.wavesBalance(address.ID())
	if err != nil {
		//...
	}
	return recipientWavesBalanceProfile.balance, nil
}
