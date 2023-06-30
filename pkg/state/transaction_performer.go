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
	checkerInfo         *checkerInfo
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

type assetBalanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.AssetID
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff
type addressAssetBalanceDiff map[assetBalanceDiffKey]int64

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
			assetBalKey := assetBalanceDiffKey{address: address, asset: asset}
			addrAssetBalanceDiff[assetBalKey] = diffAmount.balance
			continue
		}
		address, err := wavesBalanceKey.address.ToWavesAddress(scheme)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert address id to waves address")
		}
		// if the waves balance diff is 0, it means it did not change. Though the record might occur when LeaseIn and LeaseOut change,
		// but they are handled separately in snapshots
		if diffAmount.balance == 0 {
			continue
		}
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

// from txDiff and fees. no validation needed at this point
func (tp *transactionPerformer) constructWavesBalanceSnapshotFromDiff(diff addressWavesBalanceDiff) ([]WavesBalanceSnapshot, error) {
	var wavesBalances []WavesBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {

		fullBalance, err := tp.stor.balances.newestWavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		newBalance := WavesBalanceSnapshot{
			Address: wavesAddress,
			Balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
		}
		wavesBalances = append(wavesBalances, newBalance)
	}
	return wavesBalances, nil
}

func (tp *transactionPerformer) constructAssetBalanceSnapshotFromDiff(diff addressAssetBalanceDiff) ([]AssetBalanceSnapshot, error) {
	var assetBalances []AssetBalanceSnapshot
	// add miner address to the diff

	for key, diffAmount := range diff {
		balance, err := tp.stor.balances.newestAssetBalance(key.address.ID(), key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		assetInfo, err := tp.stor.assets.newestAssetInfo(key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get newest asset info")
		}

		newBalance := AssetBalanceSnapshot{
			Address: key.address,
			AssetID: key.asset.Digest(assetInfo.tail),
			Balance: uint64(int64(balance) + diffAmount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return assetBalances, nil
}

func (tp *transactionPerformer) generateBalancesAtomicSnapshots(addrWavesBalanceDiff addressWavesBalanceDiff, addrAssetBalanceDiff addressAssetBalanceDiff) ([]WavesBalanceSnapshot, []AssetBalanceSnapshot, error) {
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

func (tp *transactionPerformer) generateBalancesSnapshot(applicationRes *applicationResult) (TransactionSnapshot, error) {
	var transactionSnapshot TransactionSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, err := tp.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &wavesBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		transactionSnapshot = append(transactionSnapshot, &assetBalancesSnapshot[i])
	}
	return transactionSnapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForGenesisTx(applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	return tp.generateBalancesSnapshot(applicationRes)
}

func (tp *transactionPerformer) performGenesis(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return nil, errors.New("failed to convert interface to genesis transaction")
	}
	return tp.generateSnapshotForGenesisTx(applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForPaymentTx(applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	return tp.generateBalancesSnapshot(applicationRes)
}

func (tp *transactionPerformer) performPayment(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return nil, errors.New("failed to convert interface to payment transaction")
	}
	return tp.generateSnapshotForPaymentTx(applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForTransferTx(applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	return tp.generateBalancesSnapshot(applicationRes)
}

func (tp *transactionPerformer) performTransfer(applicationRes *applicationResult) (TransactionSnapshot, error) {
	return tp.generateSnapshotForTransferTx(applicationRes)
}

func (tp *transactionPerformer) performTransferWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) performTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return tp.performTransfer(applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForIssueTx(assetID crypto.Digest, txID crypto.Digest, senderPK crypto.PublicKey, assetInfo assetInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	var snapshot TransactionSnapshot

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet, so can't be processed with generateBalancesAtomicSnapshots
	var specialAssetSnapshot *AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		if key.asset == proto.AssetIDFromDigest(assetID) {
			// remove the element from the array

			delete(addrAssetBalanceDiff, key)
			specialAssetSnapshot = &AssetBalanceSnapshot{
				Address: key.address,
				AssetID: assetID,
				Balance: uint64(diffAmount),
			}
		}
	}

	issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
		AssetID:             assetID,
		IssuerPublicKey:     senderPK,
		SourceTransactionID: txID,
		Decimals:            assetInfo.decimals,
		IsNFT:               assetInfo.isNFT(),
	}

	assetDescription := &AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetInfo.name,
		AssetDescription: assetInfo.description,
		ChangeHeight:     assetInfo.lastNameDescChangeHeight,
	}

	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		IsReissuable:  assetInfo.reissuable,
		TotalQuantity: assetInfo.quantity,
	}
	snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)

	wavesBalancesSnapshot, assetBalancesSnapshot, err := tp.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}

	for i := range wavesBalancesSnapshot {
		snapshot = append(snapshot, &wavesBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot = append(snapshot, &assetBalancesSnapshot[i])
	}
	if specialAssetSnapshot != nil {
		snapshot = append(snapshot, specialAssetSnapshot)
	}

	return snapshot, nil
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, txID crypto.Digest, assetID crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
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

	snapshot, err := tp.generateSnapshotForIssueTx(assetID, txID, tx.SenderPK, *assetInfo, applicationRes)
	if err != nil {
		return nil, err
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
	return tp.performIssue(&tx.Issue, assetID, assetID, info, applicationRes)
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
	return tp.performIssue(&tx.Issue, assetID, assetID, info, applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForReissueTx(assetID crypto.Digest, change assetReissueChange, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	quantityDiff := big.NewInt(change.diff)
	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, err
	}
	resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)

	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  change.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}

	snapshot, err := tp.generateSnapshotForReissueTx(tx.AssetID, *change, applicationRes)
	if err != nil {
		return nil, err
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

func (tp *transactionPerformer) generateSnapshotForBurnTx(assetID crypto.Digest, change assetBurnChange, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	quantityDiff := big.NewInt(change.diff)
	assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, err
	}
	resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)

	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInfo.reissuable,
	}
	snapshot = append(snapshot, assetReissuability)
	return snapshot, nil
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}

	snapshot, err := tp.generateSnapshotForBurnTx(tx.AssetID, *change, applicationRes)
	if err != nil {
		return nil, err
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

func (tp *transactionPerformer) generateOrderAtomicSnapshot(orderID []byte, volume uint64, fee uint64) (*FilledVolumeFeeSnapshot, error) {
	newestFilledAmount, newestFilledFee, err := tp.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return nil, err
	}
	orderIdDigset, err := crypto.NewDigestFromBytes(orderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	orderSnapshot := &FilledVolumeFeeSnapshot{
		OrderID:      orderIdDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}
	return orderSnapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForExchangeTx(sellOrder proto.Order, sellFee uint64, buyOrder proto.Order, buyFee uint64, volume uint64, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}

	sellOrderID, err := sellOrder.GetID()
	if err != nil {
		return nil, err
	}
	sellOrderAtomicSnapshot, err := tp.generateOrderAtomicSnapshot(sellOrderID, volume, sellFee)
	if err != nil {
		return nil, err
	}
	buyOrderID, err := buyOrder.GetID()
	if err != nil {
		return nil, err
	}
	buyOrderAtomicSnapshot, err := tp.generateOrderAtomicSnapshot(buyOrderID, volume, buyFee)
	if err != nil {
		return nil, err
	}

	snapshot = append(snapshot, sellOrderAtomicSnapshot, buyOrderAtomicSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) increaseOrderVolume(order proto.Order, fee uint64, volume uint64, info *performerInfo) error {
	orderID, err := order.GetID()
	if err != nil {
		return err
	}
	return tp.stor.ordersVolumes.increaseFilled(orderID, volume, fee, info.blockID)
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return nil, errors.New("failed to convert interface to Exchange transaction")
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no sell order")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return nil, errors.Wrap(err, "no buy order")
	}
	volume := tx.GetAmount()
	sellFee := tx.GetSellMatcherFee()
	buyFee := tx.GetBuyMatcherFee()

	// snapshot must be generated before the state with orders is changed
	snapshot, err := tp.generateSnapshotForExchangeTx(sellOrder, sellFee, buyOrder, buyFee, volume, applicationRes)
	if err != nil {
		return nil, err
	}

	err = tp.increaseOrderVolume(sellOrder, sellFee, volume, info)
	if err != nil {
		return nil, err
	}
	err = tp.increaseOrderVolume(buyOrder, buyFee, volume, info)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (tp *transactionPerformer) generateLeaseAtomicSnapshots(leaseID crypto.Digest, l leasing, originalTxID crypto.Digest,
	senderAddress proto.WavesAddress, receiverAddress proto.WavesAddress, amount int64) (*LeaseStateSnapshot, *LeaseBalanceSnapshot, *LeaseBalanceSnapshot, error) {
	leaseStatusSnapshot := &LeaseStateSnapshot{
		LeaseID: leaseID,
		Status: LeaseStateStatus{
			Value: l.Status,
		},
		Amount:              l.Amount,
		Sender:              l.Sender,
		Recipient:           l.Recipient,
		OriginTransactionID: &originalTxID,
		Height:              l.Height,
	}

	senderBalanceProfile, err := tp.stor.balances.newestWavesBalance(senderAddress.ID())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to receive sender's waves balance")
	}
	senderLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  senderAddress,
		LeaseIn:  uint64(senderBalanceProfile.leaseIn),
		LeaseOut: uint64(senderBalanceProfile.leaseOut + amount),
	}

	receiverBalanceProfile, err := tp.stor.balances.newestWavesBalance(receiverAddress.ID())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to receive recipient's waves balance")
	}
	recipientLeaseBalanceSnapshot := &LeaseBalanceSnapshot{
		Address:  receiverAddress,
		LeaseIn:  uint64(receiverBalanceProfile.leaseIn + amount),
		LeaseOut: uint64(receiverBalanceProfile.leaseOut),
	}

	return leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForLeaseTx(lease leasing, leaseID crypto.Digest, originalTxID crypto.Digest, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	var err error
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	amount := int64(lease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := tp.generateLeaseAtomicSnapshots(leaseID, lease, originalTxID, lease.Sender, lease.Recipient, amount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease transaction")
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, txID crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
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
		Sender:    senderAddr,
		Recipient: recipientAddr,
		Amount:    tx.Amount,
		Height:    info.height,
		Status:    LeaseActive,
	}
	snapshot, err := tp.generateSnapshotForLeaseTx(*l, txID, txID, applicationRes)
	if err != nil {
		return nil, nil
	}

	if err := tp.stor.leases.addLeasing(txID, l, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to add leasing")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, *tx.ID, info, applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForLeaseCancelTx(txID *crypto.Digest, oldLease leasing, leaseID crypto.Digest, originalTxID crypto.Digest, cancelHeight uint64, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	var err error
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	negativeAmount := -int64(oldLease.Amount)
	leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := tp.generateLeaseAtomicSnapshots(leaseID, oldLease, originalTxID, oldLease.Sender, oldLease.Recipient, negativeAmount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate snapshots for a lease transaction")
	}
	leaseStatusSnapshot.Status = LeaseStateStatus{
		Value:               LeaseCanceled,
		CancelHeight:        cancelHeight,
		CancelTransactionID: txID,
	}

	snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo, applicationRes *applicationResult) (TransactionSnapshot, error) {
	oldLease, err := tp.stor.leases.newestLeasingInfo(tx.LeaseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receiver leasing info")
	}

	snapshot, err := tp.generateSnapshotForLeaseCancelTx(txID, *oldLease, tx.LeaseID, *oldLease.OriginTransactionID, info.height, applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, info.height, txID); err != nil {
		return nil, errors.Wrap(err, "failed to cancel leasing")
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

	var snapshot TransactionSnapshot
	if applicationRes != nil {
		var err error
		snapshot, err = tp.generateBalancesSnapshot(applicationRes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
		}
		aliasSnapshot := &AliasSnapshot{
			Address: senderAddr,
			Alias:   tx.Alias,
		}
		snapshot = append(snapshot, aliasSnapshot)

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

func (tp *transactionPerformer) generateSnapshotForMassTransferTx(applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	return tp.generateBalancesSnapshot(applicationRes)
}

func (tp *transactionPerformer) performMassTransferWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.generateSnapshotForMassTransferTx(applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForDataTx(senderAddress proto.WavesAddress, entries []proto.DataEntry, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, err
	}
	dataEntriesSnapshot := &DataEntriesSnapshot{
		Address:     senderAddress,
		DataEntries: entries,
	}
	snapshot = append(snapshot, dataEntriesSnapshot)
	return snapshot, nil
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

	snapshot, err := tp.generateSnapshotForDataTx(senderAddr, tx.Entries, applicationRes)
	if err != nil {
		return nil, err
	}
	for _, entry := range tx.Entries {
		if err := tp.stor.accountsDataStor.appendEntry(senderAddr, entry, info.blockID); err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForSponsorshipTx(assetID crypto.Digest, minAssetFee uint64, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &SponsorshipSnapshot{
		AssetID:         assetID,
		MinSponsoredFee: minAssetFee,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}

	snapshot, err := tp.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.sponsoredAssets.sponsorAsset(tx.AssetID, tx.MinAssetFee, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to sponsor asset")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForSetScriptTx(senderAddress proto.WavesAddress, senderPK crypto.PublicKey, script proto.Script, estimatorVersion int, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, err
	}
	// the complexity was saved before when evaluated in checker
	treeEstimation, err := tp.stor.scriptsComplexity.newestScriptComplexityByAddr(senderAddress, estimatorVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get verifier complexity from storage")
	}
	complexity := treeEstimation.Verifier

	sponsorshipSnapshot := &AccountScriptSnapshot{
		SenderPublicKey:    senderPK,
		Script:             script,
		VerifierComplexity: uint64(complexity),
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForSetScriptTx(senderAddr, tx.SenderPK, tx.Script, info.checkerInfo.estimatorVersion(), applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.scriptsStorage.setAccountScript(senderAddr, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set account script")
	}
	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForSetAssetScriptTx(assetID crypto.Digest, script proto.Script, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, err
	}
	// the complexity was saved before when evaluated in checker
	treeEstimation, err := tp.stor.scriptsComplexity.newestScriptComplexityByAsset(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get verifier complexity from storage")
	}
	complexity := treeEstimation.Verifier

	sponsorshipSnapshot := &AssetScriptSnapshot{
		AssetID:    assetID,
		Script:     script,
		Complexity: uint64(complexity),
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
	return snapshot, nil
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction, info *performerInfo, _ *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return nil, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	snapshot, err := tp.generateSnapshotForSetAssetScriptTx(tx.AssetID, tx.Script, applicationRes)
	if err != nil {
		return nil, err
	}

	if err := tp.stor.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to set asset script")
	}
	return snapshot, nil
}

func addToWavesBalanceDiff(addrWavesBalanceDiff addressWavesBalanceDiff,
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	amount int64) {
	if _, ok := addrWavesBalanceDiff[senderAddress]; ok {
		prevBalance := addrWavesBalanceDiff[senderAddress]
		prevBalance.balance -= amount
		addrWavesBalanceDiff[senderAddress] = prevBalance
	} else {
		addrWavesBalanceDiff[senderAddress] = balanceDiff{balance: amount}
	}

	if _, ok := addrWavesBalanceDiff[recipientAddress]; ok {
		prevRecipientBalance := addrWavesBalanceDiff[recipientAddress]
		prevRecipientBalance.balance += amount
		addrWavesBalanceDiff[recipientAddress] = prevRecipientBalance
	} else {
		addrWavesBalanceDiff[recipientAddress] = balanceDiff{balance: amount}
	}
}

// subtracts the amount from the sender's balance and add it to the recipient's balane
func addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff addressAssetBalanceDiff,
	senderAddress proto.WavesAddress,
	recipientAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}
	keyRecipient := assetBalanceDiffKey{address: recipientAddress, asset: asset}

	if _, ok := addrAssetBalanceDiff[keySender]; ok {
		prevSenderBalance := addrAssetBalanceDiff[keySender]
		prevSenderBalance -= amount
		addrAssetBalanceDiff[keySender] = prevSenderBalance
	} else {
		addrAssetBalanceDiff[keySender] = amount
	}

	if _, ok := addrAssetBalanceDiff[keyRecipient]; ok {
		prevRecipientBalance := addrAssetBalanceDiff[keyRecipient]
		prevRecipientBalance += amount
		addrAssetBalanceDiff[keyRecipient] = prevRecipientBalance
	} else {
		addrAssetBalanceDiff[keyRecipient] = amount
	}
}

// adds/subtracts the amount to the sender balance
func addSenderToAssetBalanceDiff(addrAssetBalanceDiff addressAssetBalanceDiff,
	senderAddress proto.WavesAddress,
	asset proto.AssetID,
	amount int64) {
	keySender := assetBalanceDiffKey{address: senderAddress, asset: asset}

	if _, ok := addrAssetBalanceDiff[keySender]; ok {
		prevSenderBalance := addrAssetBalanceDiff[keySender]
		prevSenderBalance += amount
		addrAssetBalanceDiff[keySender] = prevSenderBalance
	} else {
		addrAssetBalanceDiff[keySender] = amount
	}

}

// TODO optimize this
func (tp *transactionPerformer) generateInvokeSnapshot(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {

	blockHeight := info.height + 1

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := addressBalanceDiffFromTxDiff(applicationRes.changes.diff, tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	var snapshot TransactionSnapshot
	var dataEntries = make(map[proto.WavesAddress]proto.DataEntries)
	if invocationRes != nil {

		for _, action := range invocationRes.actions {

			switch a := action.(type) {
			case *proto.DataEntryScriptAction:
				senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, err
				}
				// construct the map first and create the snapshot later for convenience
				if _, ok := dataEntries[senderAddr]; ok {
					entries := dataEntries[senderAddr]
					entries = append(entries, a.Entry)
					dataEntries[senderAddr] = entries
				} else {
					dataEntries[senderAddr] = proto.DataEntries{a.Entry}
				}

			case *proto.AttachedPaymentScriptAction:
				senderAddress, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				recipientAddress, err := recipientToAddress(a.Recipient, tp.stor.aliases)
				if err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
				// No balance validation done below
				if a.Asset.Present { // Update asset balance
					addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff, senderAddress, recipientAddress, proto.AssetIDFromDigest(a.Asset.ID), a.Amount)
				} else { // Update Waves balance
					addToWavesBalanceDiff(addrWavesBalanceDiff, senderAddress, recipientAddress, a.Amount)
				}
			case *proto.TransferScriptAction:
				senderAddress, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				recipientAddress, err := recipientToAddress(a.Recipient, tp.stor.aliases)
				if err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
				// No balance validation done below
				if a.Asset.Present { // Update asset balance
					addSenderRecipientToAssetBalanceDiff(addrAssetBalanceDiff, senderAddress, recipientAddress, proto.AssetIDFromDigest(a.Asset.ID), a.Amount)
				} else { // Update Waves balance
					addToWavesBalanceDiff(addrWavesBalanceDiff, senderAddress, recipientAddress, a.Amount)
				}
			case *proto.SponsorshipScriptAction:
				sponsorshipSnapshot := &SponsorshipSnapshot{
					AssetID:         a.AssetID,
					MinSponsoredFee: uint64(a.MinFee),
				}
				snapshot = append(snapshot, sponsorshipSnapshot)
			case *proto.IssueScriptAction:
				assetInfo := assetInfo{
					assetConstInfo: assetConstInfo{
						tail:                 proto.DigestTail(a.ID),
						issuer:               *a.Sender,
						decimals:             uint8(a.Decimals),
						issueHeight:          blockHeight,
						issueSequenceInBlock: info.stateActionsCounter.NextIssueActionNumber(),
					},
					assetChangeableInfo: assetChangeableInfo{
						quantity:                 *big.NewInt(a.Quantity),
						name:                     a.Name,
						description:              a.Description,
						lastNameDescChangeHeight: blockHeight,
						reissuable:               a.Reissuable,
					},
				}
				issuerAddress, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}

				issueStaticInfoSnapshot := &StaticAssetInfoSnapshot{
					AssetID:             a.ID,
					IssuerPublicKey:     *a.Sender,
					SourceTransactionID: txID,
					Decimals:            assetInfo.decimals,
					IsNFT:               assetInfo.isNFT(),
				}

				assetDescription := &AssetDescriptionSnapshot{
					AssetID:          a.ID,
					AssetName:        assetInfo.name,
					AssetDescription: assetInfo.description,
					ChangeHeight:     assetInfo.lastNameDescChangeHeight,
				}

				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.ID,
					IsReissuable:  assetInfo.reissuable,
					TotalQuantity: assetInfo.quantity,
				}
				snapshot = append(snapshot, issueStaticInfoSnapshot, assetDescription, assetReissuability)

				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issuerAddress, proto.AssetIDFromDigest(a.ID), a.Quantity)

			case *proto.ReissueScriptAction:

				assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(a.AssetID))
				if err != nil {
					return nil, err
				}
				quantityDiff := big.NewInt(a.Quantity)
				resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)
				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.AssetID,
					TotalQuantity: *resQuantity,
					IsReissuable:  a.Reissuable,
				}

				issueAddress, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issueAddress, proto.AssetIDFromDigest(a.AssetID), a.Quantity)
				snapshot = append(snapshot, assetReissuability)

			case *proto.BurnScriptAction:
				assetInfo, err := tp.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(a.AssetID))
				if err != nil {
					return nil, err
				}
				quantityDiff := big.NewInt(a.Quantity)
				resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)
				assetReissuability := &AssetVolumeSnapshot{
					AssetID:       a.AssetID,
					TotalQuantity: *resQuantity,
					IsReissuable:  assetInfo.reissuable,
				}

				issueAddress, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get an address from a public key")
				}
				addSenderToAssetBalanceDiff(addrAssetBalanceDiff, issueAddress, proto.AssetIDFromDigest(a.AssetID), -a.Quantity)
				snapshot = append(snapshot, assetReissuability)
			case *proto.LeaseScriptAction:
				senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, *a.Sender)
				if err != nil {
					return nil, err
				}
				var recipientAddr proto.WavesAddress
				if addr := a.Recipient.Address(); addr == nil {
					recipientAddr, err = tp.stor.aliases.newestAddrByAlias(a.Recipient.Alias().Alias)
					if err != nil {
						return nil, errors.Errorf("invalid alias: %v\n", err)
					}
				} else {
					recipientAddr = *addr
				}
				l := &leasing{
					Sender:    senderAddr,
					Recipient: recipientAddr,
					Amount:    uint64(a.Amount),
					Height:    info.height,
					Status:    LeaseActive,
				}
				var amount = int64(l.Amount)
				leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := tp.generateLeaseAtomicSnapshots(a.ID, *l, txID, senderAddr, recipientAddr, amount)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate snapshots for a lease action")
				}
				snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
			case *proto.LeaseCancelScriptAction:
				l, err := tp.stor.leases.leasingInfo(a.LeaseID)
				if err != nil {
					return nil, errors.Wrap(err, "failed to receiver leasing info")
				}

				var amount = -int64(l.Amount)
				leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot, err := tp.generateLeaseAtomicSnapshots(a.LeaseID, *l, txID, l.Sender, l.Recipient, amount)
				if err != nil {
					return nil, errors.Wrap(err, "failed to generate snapshots for a lease cancel action")
				}
				snapshot = append(snapshot, leaseStatusSnapshot, senderLeaseBalanceSnapshot, recipientLeaseBalanceSnapshot)
			default:
				return nil, errors.Errorf("unknown script action type %T", a)
			}
		}

		for address, entries := range dataEntries {
			dataEntrySnapshot := &DataEntriesSnapshot{Address: address, DataEntries: entries}
			snapshot = append(snapshot, dataEntrySnapshot)
		}

	}

	wavesBalancesSnapshot, assetBalancesSnapshot, err := tp.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}

	for i := range wavesBalancesSnapshot {
		snapshot = append(snapshot, &wavesBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot = append(snapshot, &assetBalancesSnapshot[i])
	}

	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForInvokeScriptTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	return tp.generateInvokeSnapshot(txID, info, invocationRes, applicationRes)
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeScriptWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForInvokeScriptTx(txID, info, invocationRes, applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot for an invoke transaction")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForInvokeExpressionTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	return tp.generateInvokeSnapshot(txID, info, invocationRes, applicationRes)
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return nil, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to commit invoke changes")
	}
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	return tp.generateSnapshotForInvokeExpressionTx(txID, info, invocationRes, applicationRes)
}

func (tp *transactionPerformer) generateSnapshotForEthereumInvokeScriptTx(txID crypto.Digest, info *performerInfo, invocationRes *invocationResult, applicationRes *applicationResult) (TransactionSnapshot, error) {
	return tp.generateInvokeSnapshot(txID, info, invocationRes, applicationRes)
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
	txIDBytes, err := transaction.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	txID, err := crypto.NewDigestFromBytes(txIDBytes)
	if err != nil {
		return nil, err
	}

	snapshot, err := tp.generateSnapshotForEthereumInvokeScriptTx(txID, info, invocationRes, applicationRes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a snapshot for an invoke transaction")
	}

	return snapshot, nil
}

func (tp *transactionPerformer) generateSnapshotForUpdateAssetInfoTx(assetID crypto.Digest, assetName string, assetDescription string, changeHeight proto.Height, applicationRes *applicationResult) (TransactionSnapshot, error) {
	if applicationRes == nil {
		return nil, nil
	}
	snapshot, err := tp.generateBalancesSnapshot(applicationRes)
	if err != nil {
		return nil, err
	}
	sponsorshipSnapshot := &AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetName,
		AssetDescription: assetDescription,
		ChangeHeight:     changeHeight,
	}
	snapshot = append(snapshot, sponsorshipSnapshot)
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

	snapshot, err := tp.generateSnapshotForUpdateAssetInfoTx(tx.AssetID, tx.Name, tx.Description, blockHeight, applicationRes)
	if err != nil {
		return nil, err
	}
	if err := tp.stor.assets.updateAssetInfo(tx.AssetID, ch, info.blockID); err != nil {
		return nil, errors.Wrap(err, "failed to update asset info")
	}
	return snapshot, nil
}
