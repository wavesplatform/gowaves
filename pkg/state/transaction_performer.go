package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerInfo struct {
	height              uint64
	blockID             proto.BlockID
	currentMinerAddress proto.WavesAddress
	stateActionsCounter *proto.StateActionsCounter
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

type assetBalanceDiff struct {
	asset  proto.AssetID
	amount int64
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff
type addressAssetBalanceDiff map[proto.WavesAddress]assetBalanceDiff

func addressBalanceDiffFromTxDiff(diff txDiff, scheme proto.Scheme) (addressWavesBalanceDiff, addressAssetBalanceDiff, error) {
	addrWavesBalanceDiff := make(addressWavesBalanceDiff)
	addrAssetBalanceDiff := make(addressAssetBalanceDiff)
	for balanceKeyString, diffAmount := range diff {

		// construct address from key
		wavesBalanceKey := &wavesBalanceKey{}
		err := wavesBalanceKey.unmarshal([]byte(balanceKeyString))
		if err != nil {
			assetBalanceKey := &assetBalanceKey{}
			err := assetBalanceKey.unmarshal([]byte(balanceKeyString))
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert balance key to asset balance key")
			}
			asset := assetBalanceKey.asset
			address, err := assetBalanceKey.address.ToWavesAddress(scheme)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
			}
			addrAssetBalanceDiff[address] = assetBalanceDiff{asset: asset, amount: diffAmount.balance}
			continue
		}
		address, err := wavesBalanceKey.address.ToWavesAddress(scheme)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
		}
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

//func (firstAddressBalanceDiff addressWavesBalanceDiff) mergeWithAnotherDiff(secondDiff addressWavesBalanceDiff) error {
//	for address, diffBalance := range secondDiff {
//		if _, ok := firstAddressBalanceDiff[address]; ok {
//			oldBalance := firstAddressBalanceDiff[address]
//			err := oldBalance.addCommon(&diffBalance)
//			if err != nil {
//				return errors.Wrap(err, "failed to merge two balance diffs")
//			}
//		}
//	}
//	return nil
//}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

// from txDiff and fees. no validation needed at this point
func (tp *transactionPerformer) constructWavesBalanceSnapshotFromDiff(diff addressWavesBalanceDiff) (*WavesBalancesSnapshot, error) {
	var wavesBalances []balanceWaves
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {

		fullBalance, err := tp.stor.balances.wavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		newBalance := balanceWaves{
			address: wavesAddress,
			balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
		}
		wavesBalances = append(wavesBalances, newBalance)
	}
	return &WavesBalancesSnapshot{wavesBalances: wavesBalances}, nil
}

func (tp *transactionPerformer) constructAssetBalanceSnapshotFromDiff(diff addressAssetBalanceDiff) (*AssetBalancesSnapshot, error) {
	var assetBalances []balanceAsset
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		balance, err := tp.stor.balances.assetBalance(wavesAddress.ID(), diffAmount.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		assetInfo, err := tp.stor.assets.newestAssetInfo(diffAmount.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get newest asset info")
		}
		newBalance := balanceAsset{
			address: wavesAddress,
			assetID: diffAmount.asset.Digest(assetInfo.tail),
			balance: uint64(int64(balance) + diffAmount.amount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return &AssetBalancesSnapshot{assetBalances: assetBalances}, nil
}

func (tp *transactionPerformer) constructBalancesSnapshotFromDiff(diff txDiff) (*WavesBalancesSnapshot, *AssetBalancesSnapshot, error) {
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(diff, tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalanceSnapshot, err := tp.constructWavesBalanceSnapshotFromDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, nil
	}

	assetBalanceSnapshot, err := tp.constructAssetBalanceSnapshotFromDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, nil
}

func (tp *transactionPerformer) performGenesis(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performTransfer(applicationRes *applicationResult) (TransactionSnapshot, error) {

	var snapshot TransactionSnapshot
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, assetID crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	blockHeight := info.height + 1
	// Create new asset.
	assetInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(assetID),
			issuer:               tx.SenderPK,
			decimals:             tx.Decimals,
			issueHeight:          blockHeight,
			issueSequenceInBlock: info.stateActionsCounter.NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}

	sender := proto.MustAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	var snapshot TransactionSnapshot
	issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
		assetID:  assetID,
		issuer:   sender,
		decimals: int8(assetInfo.decimals),
		isNFT:    assetInfo.isNFT(),
	}

	assetDescription := &AssetDescriptionSnapshot{
		assetID:          assetID,
		assetName:        assetInfo.name,
		assetDescription: assetInfo.description,
		changeHeight:     assetInfo.lastNameDescChangeHeight,
	}

	assetReissuability := &AssetReissuabilitySnapshot{
		assetID:      assetID,
		isReissuable: assetInfo.reissuable,
	}
	snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.assets.issueAsset(proto.AssetIDFromDigest(assetID), assetInfo, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to issue asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, proto.Script{}, tx.SenderPK, info.blockID); err != nil {
		return nil, err
	}
	return tp.performIssue(&tx.Issue, assetID, info, applicationRes)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, err
	}
	return tp.performIssue(&tx.Issue, assetID, info, applicationRes)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}

	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return nil, err
	}

	quantityDiff := big.NewInt(change.diff)
	resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)
	assetReissuability := &AssetReissuabilitySnapshot{
		assetID:       tx.AssetID,
		totalQuantity: *resQuantity,
		isReissuable:  change.reissuable,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, assetReissuability)

	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.assets.reissueAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to reissue asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.performReissue(&tx.Reissue, info, applicationRes)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.performReissue(&tx.Reissue, info, applicationRes)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}

	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(tx.AssetID))
	if err != nil {
		return nil, err
	}
	quantityDiff := big.NewInt(change.diff)
	resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)
	assetReissuability := &AssetReissuabilitySnapshot{
		assetID:       tx.AssetID,
		totalQuantity: *resQuantity,
		isReissuable:  assetInfo.reissuable,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, assetReissuability)

	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.assets.burnAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to burn asset")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.performBurn(&tx.Burn, info, applicationRes)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.performBurn(&tx.Burn, info, applicationRes)
}

func (tp *transactionPerformer) increaseOrderVolume(order proto.Order, tx proto.Exchange, info *performerInfo) (*FilledVolumeFeeSnapshot, error) {
	orderId, err := order.GetID()
	if err != nil {
		return nil, err
	}
	fee := tx.GetBuyMatcherFee()
	if order.GetOrderType() == proto.Sell {
		fee = tx.GetSellMatcherFee()
	}
	volume := tx.GetAmount()

	newestFilledFee, err := tp.stor.ordersVolumes.newestFilledFee(orderId)
	if err != nil {
		return nil, err
	}
	newestFilledAmount, err := tp.stor.ordersVolumes.newestFilledAmount(orderId)
	if err != nil {
		return nil, err
	}
	orderSnapshot := &FilledVolumeFeeSnapshot{
		orderID:      orderId,
		filledFee:    newestFilledFee + fee,
		filledVolume: newestFilledAmount + volume,
	}

	if err := tp.stor.ordersVolumes.increaseFilledFee(orderId, fee, info.blockID); err != nil {
		return nil, err
	}
	if err := tp.stor.ordersVolumes.increaseFilledAmount(orderId, volume, info.blockID); err != nil {
		return nil, err
	}

	return orderSnapshot, nil
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert interface to Exchange transaction")
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no sell order")
	}
	sellOrderSnapshot, err := tp.increaseOrderVolume(so, tx, info)
	if err != nil {
		return nil, err
	}
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no buy order")
	}
	buyOrderSnapshot, err := tp.increaseOrderVolume(bo, tx, info)
	if err != nil {
		return nil, err
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sellOrderSnapshot, buyOrderSnapshot)

	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, id *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	var recipientAddr proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return nil, errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = *addr
	}
	// Add leasing to lease state.
	l := &leasing{
		Sender:         senderAddr,
		Recipient:      recipientAddr,
		Amount:         tx.Amount,
		Height:         info.height,
		Status:         LeaseActive,
		RecipientAlias: tx.Recipient.Alias(),
	}
	leaseStatusSnapshot := &LeaseStatusSnapshot{
		leaseID:  *id,
		isActive: true,
	}

	senderBalanceProfile, err := tp.stor.balances.wavesBalance(senderAddr.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		address:  senderAddr,
		leaseIn:  senderBalanceProfile.leaseIn,
		leaseOut: senderBalanceProfile.leaseOut + int64(tx.Amount),
	}

	receiverBalanceProfile, err := tp.stor.balances.wavesBalance(recipientAddr.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		address:  recipientAddr,
		leaseIn:  receiverBalanceProfile.leaseIn + int64(tx.Amount),
		leaseOut: receiverBalanceProfile.leaseOut,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.leases.addLeasing(*id, l, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to add leasing")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, info.height, txID); err != nil {
		return nil, errors.Wrap(err, "failed to cancel leasing")
	}
	leaseStatusSnapshot := &LeaseStatusSnapshot{
		leaseID:  tx.LeaseID,
		isActive: false,
	}

	leasingInfo, err := tp.stor.leases.leasingInfo(tx.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	// TODO check if the balance will be updated immediately after the leasing
	senderBalanceProfile, err := tp.stor.balances.wavesBalance(leasingInfo.Sender.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		address:  leasingInfo.Sender,
		leaseIn:  senderBalanceProfile.leaseIn,
		leaseOut: senderBalanceProfile.leaseOut,
	}

	receiverBalanceProfile, err := tp.stor.balances.wavesBalance(leasingInfo.Recipient.ID())
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		address:  leasingInfo.Recipient,
		leaseIn:  receiverBalanceProfile.leaseIn,
		leaseOut: receiverBalanceProfile.leaseOut,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	aliasSnapshot := &AliasSnapshot{
		address: senderAddr,
		alias:   tx.Alias,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, aliasSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.aliases.createAlias(tx.Alias.Alias, senderAddr, info.blockID); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, applicationRes)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info, applicationRes)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	dataEntriesSnapshot := &DataEntriesSnapshot{
		address:     senderAddr,
		dataEntries: tx.Entries,
	}

	var snapshot TransactionSnapshot
	snapshot = append(snapshot, dataEntriesSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	for _, entry := range tx.Entries {
		if err := tp.stor.accountsDataStor.appendEntry(senderAddr, entry, info.blockID); err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	sponsorshipSnapshot := &SponsorshipSnapshot{
		assetID:         tx.AssetID,
		minSponsoredFee: tx.MinAssetFee,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sponsorshipSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}

	}

	if err := tp.stor.sponsoredAssets.sponsorAsset(tx.AssetID, tx.MinAssetFee, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to sponsor asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	var snapshot TransactionSnapshot

	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}
	if applicationRes != nil {
		sponsorshipSnapshot := &AccountScriptSnapshot{
			address: senderAddr,
			script:  tx.Script,
		}
		snapshot = append(snapshot, sponsorshipSnapshot)

		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.scriptsStorage.setAccountScript(senderAddr, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set account script")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	var snapshot TransactionSnapshot
	sponsorshipSnapshot := &AssetScriptSnapshot{
		assetID: tx.AssetID,
		script:  tx.Script,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)

	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set asset script")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeScriptWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	// TODO
	if applicationRes != nil {
		for _, action := range invocationRes.actions {

			switch a := action.(type) {
			case *proto.DataEntryScriptAction:

			case *proto.AttachedPaymentScriptAction:

			case *proto.TransferScriptAction:

			case *proto.SponsorshipScriptAction:

			case *proto.IssueScriptAction:

			case *proto.ReissueScriptAction:

			case *proto.BurnScriptAction:

			case *proto.LeaseScriptAction:

			case *proto.LeaseCancelScriptAction:

			default:
				return nil, errors.Errorf("unknown script action type %T", a)
			}
		}
	}
	return nil, nil
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	// TODO
	return nil, nil
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return nil, errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	if _, ok := ethTx.TxKind.(*proto.EthereumInvokeScriptTxKind); ok {
		if err := tp.stor.commitUncertain(info.blockID); err != nil {
			return nil, errors.Wrap(err, "failed to commit invoke changes")
		}
	}
	var snapshot TransactionSnapshot
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	// nothing to do for proto.EthereumTransferWavesTxKind and proto.EthereumTransferAssetsErc20TxKind
	return snapshot, nil
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	blockHeight := info.height + 1
	ch := &assetInfoChange{
		newName:        tx.Name,
		newDescription: tx.Description,
		newHeight:      blockHeight,
	}

	sponsorshipSnapshot := &AssetDescriptionSnapshot{
		assetID:          tx.AssetID,
		assetName:        tx.Name,
		assetDescription: tx.Description,
		changeHeight:     blockHeight,
	}
	var snapshot TransactionSnapshot
	snapshot = append(snapshot, sponsorshipSnapshot)
	if applicationRes != nil {
		wavesBalanceSnapshot, assetBalanceSnapshot, err := tp.constructBalancesSnapshotFromDiff(applicationRes.changes.diff)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
		}
		snapshot = append(snapshot, wavesBalanceSnapshot)

		if assetBalanceSnapshot != nil {
			snapshot = append(snapshot, assetBalanceSnapshot)
		}
	}

	if err := tp.stor.assets.updateAssetInfo(tx.AssetID, ch, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to update asset info")
	}
	return snapshot, nil
}
