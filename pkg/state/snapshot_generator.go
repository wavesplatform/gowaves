package state

import (
	"bytes"
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotGenerator struct {
	stor   *blockchainEntitiesStorage
	scheme proto.Scheme
}

func newSnapshotGenerator(stor *blockchainEntitiesStorage, scheme proto.Scheme) *snapshotGenerator {
	return &snapshotGenerator{
		stor:   stor,
		scheme: scheme,
	}
}

func (sg *snapshotGenerator) performGenesis(
	transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to genesis transaction")
	}
	return sg.generateSnapshotForGenesisTx(balanceChanges)
}

func (sg *snapshotGenerator) performPayment(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to payment transaction")
	}
	return sg.generateSnapshotForPaymentTx(balanceChanges)
}

func (sg *snapshotGenerator) performTransferWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return sg.generateSnapshotForTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return sg.generateSnapshotForTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performIssueWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	return sg.generateSnapshotForIssueTx(&tx.Issue, assetID, info, balanceChanges, nil, nil)
}

func (sg *snapshotGenerator) performIssueWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Errorf("failed to get transaction ID: %v", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return txSnapshot{}, err
	}
	se := info.checkerData.scriptEstimation
	return sg.generateSnapshotForIssueTx(&tx.Issue, assetID, info, balanceChanges, se, tx.Script)
}

func (sg *snapshotGenerator) performReissueWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return sg.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (sg *snapshotGenerator) performReissueWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return sg.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (sg *snapshotGenerator) performBurnWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return sg.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
}

func (sg *snapshotGenerator) performBurnWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return sg.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
}

func (sg *snapshotGenerator) performExchange(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to Exchange transaction")
	}
	sellOrder, err := tx.GetSellOrder()
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "no sell order")
	}
	buyOrder, err := tx.GetBuyOrder()
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "no buy order")
	}
	volume := tx.GetAmount()
	sellFee := tx.GetSellMatcherFee()
	buyFee := tx.GetBuyMatcherFee()

	// snapshot must be generated before the state with orders is changed
	return sg.generateSnapshotForExchangeTx(sellOrder, sellFee, buyOrder, buyFee, volume, balanceChanges)
}

func (sg *snapshotGenerator) performLeaseWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return sg.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (sg *snapshotGenerator) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return sg.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (sg *snapshotGenerator) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return sg.generateSnapshotForLeaseCancelTx(
		tx.LeaseCancel.LeaseID,
		tx.ID,
		info.blockHeight(),
		balanceChanges,
	)
}

func (sg *snapshotGenerator) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return sg.generateSnapshotForLeaseCancelTx(
		tx.LeaseCancel.LeaseID,
		tx.ID,
		info.blockHeight(),
		balanceChanges,
	)
}

func (sg *snapshotGenerator) performCreateAliasWithSig(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return sg.generateSnapshotForCreateAliasTx(sg.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (sg *snapshotGenerator) performCreateAliasWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return sg.generateSnapshotForCreateAliasTx(sg.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (sg *snapshotGenerator) performMassTransferWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return sg.generateSnapshotForMassTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performDataWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(sg.scheme, tx.SenderPK)
	if err != nil {
		return txSnapshot{}, err
	}
	return sg.generateSnapshotForDataTx(senderAddr, tx.Entries, balanceChanges)
}

func (sg *snapshotGenerator) performSponsorshipWithProofs(transaction proto.Transaction, _ *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	return sg.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, balanceChanges)
}

func (sg *snapshotGenerator) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo,
	_ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return txSnapshot{}, errors.New("script estimations must be set for SetScriptWithProofs tx")
	}

	snapshot, err := sg.generateSnapshotForSetScriptTx(tx.SenderPK, tx.Script, *se, balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate snapshot for set script tx")
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) performSetAssetScriptWithProofs(transaction proto.Transaction,
	info *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}

	se := info.checkerData.scriptEstimation
	if !se.isPresent() {
		return txSnapshot{}, errors.New("script estimations must be set for SetAssetScriptWithProofs tx")
	}

	snapshot, err := sg.generateSnapshotForSetAssetScriptTx(tx.AssetID, tx.Script, balanceChanges, *se)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate snapshot for set asset script tx")
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) performInvokeScriptWithProofs(transaction proto.Transaction,
	info *performerInfo,
	_ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	se := info.checkerData.scriptEstimation
	return sg.generateSnapshotForInvokeScript(tx.ScriptRecipient, balanceChanges, se)
}

func (sg *snapshotGenerator) performInvokeExpressionWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult,
	balanceChanges txDiff) (txSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	return sg.generateSnapshotForInvokeExpressionTx(balanceChanges)
}

func (sg *snapshotGenerator) performEthereumTransactionWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	_, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	return sg.generateSnapshotForEthereumInvokeScriptTx(balanceChanges)
}

func (sg *snapshotGenerator) performUpdateAssetInfoWithProofs(transaction proto.Transaction,
	_ *performerInfo, _ *invocationResult, balanceChanges txDiff) (txSnapshot, error) {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	return sg.generateSnapshotForUpdateAssetInfoTx(
		tx.AssetID,
		tx.Name,
		tx.Description,
		balanceChanges,
	)
}

type addressWavesBalanceDiff map[proto.WavesAddress]balanceDiff

type assetBalanceDiffKey struct {
	address proto.WavesAddress
	asset   proto.AssetID
}
type addressAssetBalanceDiff map[assetBalanceDiffKey]int64

func (sg *snapshotGenerator) generateSnapshotForGenesisTx(balanceChanges txDiff) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForPaymentTx(balanceChanges txDiff) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForTransferTx(balanceChanges txDiff) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForIssueTx(
	tx *proto.Issue,
	assetID crypto.Digest,
	info *performerInfo,
	balanceChanges txDiff,
	scriptEstimation *scriptEstimation,
	script proto.Script,
) (txSnapshot, error) {
	// Create new asset.
	blockHeight := info.blockHeight()
	senderPK := tx.SenderPK
	ai := assetInfo{
		assetConstInfo: assetConstInfo{
			tail:        proto.DigestTail(assetID),
			issuer:      senderPK,
			decimals:    tx.Decimals,
			issueHeight: blockHeight,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots.
	var specialAssetSnapshot *proto.AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		if key.asset == proto.AssetIDFromDigest(assetID) {
			// remove the element from the array
			delete(addrAssetBalanceDiff, key)
			specialAssetSnapshot = &proto.AssetBalanceSnapshot{
				Address: key.address,
				AssetID: assetID,
				Balance: uint64(diffAmount),
			}
		}
	}

	var snapshot txSnapshot

	issueStaticInfoSnapshot := &proto.NewAssetSnapshot{
		AssetID:         assetID,
		IssuerPublicKey: senderPK,
		Decimals:        ai.decimals,
		IsNFT:           ai.isNFT(),
	}
	assetDescription := &proto.AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        ai.name,
		AssetDescription: ai.description,
	}
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		IsReissuable:  ai.reissuable,
		TotalQuantity: ai.quantity,
	}

	snapshot.regular = append(snapshot.regular,
		issueStaticInfoSnapshot, assetDescription, assetReissuability,
	)

	if !script.IsEmpty() { // generate asset script snapshot only for non-empty script
		assetScriptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  script,
		}
		if scriptEstimation.isPresent() {
			internalComplexitySnapshot := &InternalAssetScriptComplexitySnapshot{
				Estimation: scriptEstimation.estimation, AssetID: assetID,
				ScriptIsEmpty: scriptEstimation.scriptIsEmpty}
			snapshot.internal = append(snapshot.internal, internalComplexitySnapshot)
		}
		snapshot.regular = append(snapshot.regular, assetScriptSnapshot)
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &leaseBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &assetBalancesSnapshot[i])
	}
	if specialAssetSnapshot != nil {
		snapshot.regular = append(snapshot.regular, specialAssetSnapshot)
	}

	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForReissueTx(
	assetID crypto.Digest,
	isReissuable bool,
	quantity uint64,
	balanceChanges txDiff,
) (txSnapshot, error) {
	// Modify asset.

	quantityDiff := new(big.Int).SetUint64(quantity)
	assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return txSnapshot{}, err
	}
	resQuantity := assetInfo.quantity.Add(&assetInfo.quantity, quantityDiff)

	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  isReissuable,
	}
	snapshot.regular = append(snapshot.regular, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForBurnTx(assetID crypto.Digest, newQuantity uint64,
	balanceChanges txDiff) (txSnapshot, error) {
	// Modify asset.

	quantityDiff := new(big.Int).SetUint64(newQuantity)
	assetInfo, err := sg.stor.assets.newestAssetInfo(proto.AssetIDFromDigest(assetID))
	if err != nil {
		return txSnapshot{}, err
	}
	resQuantity := assetInfo.quantity.Sub(&assetInfo.quantity, quantityDiff)

	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  assetInfo.reissuable,
	}
	snapshot.regular = append(snapshot.regular, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForExchangeTx(sellOrder proto.Order, sellFee uint64,
	buyOrder proto.Order, buyFee uint64, volume uint64,
	balanceChanges txDiff) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}

	sellOrderID, err := sellOrder.GetID()
	if err != nil {
		return txSnapshot{}, err
	}
	sellOrderAtomicSnapshot, err := sg.generateOrderAtomicSnapshot(sellOrderID, volume, sellFee)
	if err != nil {
		return txSnapshot{}, err
	}
	buyOrderID, err := buyOrder.GetID()
	if err != nil {
		return txSnapshot{}, err
	}
	buyOrderAtomicSnapshot, err := sg.generateOrderAtomicSnapshot(buyOrderID, volume, buyFee)
	if err != nil {
		return txSnapshot{}, err
	}

	snapshot.regular = append(snapshot.regular, sellOrderAtomicSnapshot, buyOrderAtomicSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseTx(
	tx *proto.Lease,
	txID *crypto.Digest,
	info *performerInfo,
	balanceChanges txDiff,
) (txSnapshot, error) {
	var recipientAddr proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		rcpAddr, err := sg.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return txSnapshot{}, errors.Wrap(err, "invalid alias")
		}
		recipientAddr = rcpAddr
	} else {
		recipientAddr = *addr
	}
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}

	leaseID := *txID
	leaseStatusSnapshot := &proto.NewLeaseSnapshot{
		LeaseID:       leaseID,
		Amount:        tx.Amount,
		SenderPK:      tx.SenderPK,
		RecipientAddr: recipientAddr,
	}
	leaseStatusActiveSnapshot := &InternalNewLeaseInfoSnapshot{
		LeaseID:             leaseID,
		OriginHeight:        info.blockHeight(),
		OriginTransactionID: txID,
	}
	snapshot.regular = append(snapshot.regular, leaseStatusSnapshot)
	snapshot.internal = append(snapshot.internal, leaseStatusActiveSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForLeaseCancelTx(
	leaseID crypto.Digest,
	txID *crypto.Digest,
	cancelHeight proto.Height,
	balanceChanges txDiff,
) (txSnapshot, error) {
	var err error
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate a snapshot based on transaction's diffs")
	}
	cancelledLeaseSnapshot := &proto.CancelledLeaseSnapshot{
		LeaseID: leaseID,
	}
	leaseStatusCancelledSnapshot := &InternalCancelledLeaseInfoSnapshot{
		LeaseID:             leaseID,
		CancelHeight:        cancelHeight,
		CancelTransactionID: txID,
	}
	snapshot.regular = append(snapshot.regular, cancelledLeaseSnapshot)
	snapshot.internal = append(snapshot.internal, leaseStatusCancelledSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForCreateAliasTx(
	scheme proto.Scheme,
	senderPK crypto.PublicKey,
	alias proto.Alias,
	balanceChanges txDiff,
) (txSnapshot, error) {
	senderAddr, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return txSnapshot{}, err
	}
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	aliasSnapshot := &proto.AliasSnapshot{
		Address: senderAddr,
		Alias:   alias,
	}
	snapshot.regular = append(snapshot.regular, aliasSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForMassTransferTx(balanceChanges txDiff) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForDataTx(senderAddress proto.WavesAddress,
	entries []proto.DataEntry, balanceChanges txDiff) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	dataEntriesSnapshot := &proto.DataEntriesSnapshot{
		Address:     senderAddress,
		DataEntries: entries,
	}
	snapshot.regular = append(snapshot.regular, dataEntriesSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSponsorshipTx(assetID crypto.Digest,
	minAssetFee uint64, balanceChanges txDiff) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	sponsorshipSnapshot := &proto.SponsorshipSnapshot{
		AssetID:         assetID,
		MinSponsoredFee: minAssetFee,
	}
	snapshot.regular = append(snapshot.regular, sponsorshipSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetScriptTx(senderPK crypto.PublicKey, script proto.Script,
	scriptEstimation scriptEstimation, balanceChanges txDiff) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}

	// If the script is empty, it will still be stored in the storage.
	accountScriptSnapshot := &proto.AccountScriptSnapshot{
		SenderPublicKey:    senderPK,
		Script:             script,
		VerifierComplexity: uint64(scriptEstimation.estimation.Verifier),
	}

	snapshot.regular = append(snapshot.regular, accountScriptSnapshot)

	scriptAddr, cnvrtErr := proto.NewAddressFromPublicKey(sg.scheme, senderPK)
	if cnvrtErr != nil {
		return txSnapshot{}, errors.Wrap(cnvrtErr, "failed to get sender for SetScriptTX")
	}
	internalComplexitySnapshot := &InternalDAppComplexitySnapshot{
		Estimation:    scriptEstimation.estimation,
		ScriptAddress: scriptAddr,
		ScriptIsEmpty: scriptEstimation.scriptIsEmpty,
	}
	snapshot.internal = append(snapshot.internal, internalComplexitySnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForSetAssetScriptTx(assetID crypto.Digest, script proto.Script,
	balanceChanges txDiff, scriptEstimation scriptEstimation) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	// script here must not be empty, see transaction checker for set asset script transaction
	assetScrptSnapshot := &proto.AssetScriptSnapshot{
		AssetID: assetID,
		Script:  script,
	}
	snapshot.regular = append(snapshot.regular, assetScrptSnapshot)
	internalComplexitySnapshot := &InternalAssetScriptComplexitySnapshot{
		Estimation: scriptEstimation.estimation, AssetID: assetID,
		ScriptIsEmpty: scriptEstimation.scriptIsEmpty}
	snapshot.internal = append(snapshot.internal, internalComplexitySnapshot)
	return snapshot, nil
}

func generateSnapshotsFromAssetsUncertain(
	assetsUncertain map[proto.AssetID]wrappedUncertainInfo,
) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for assetID, infoAsset := range assetsUncertain {
		fullAssetID := proto.ReconstructDigest(assetID, infoAsset.assetInfo.tail)
		// order of snapshots here is important: static info snapshot should be first
		if infoAsset.wasJustIssued {
			issueStaticInfoSnapshot := &proto.NewAssetSnapshot{
				AssetID:         fullAssetID,
				IssuerPublicKey: infoAsset.assetInfo.issuer,
				Decimals:        infoAsset.assetInfo.decimals,
				IsNFT:           infoAsset.assetInfo.isNFT(),
			}

			assetDescription := &proto.AssetDescriptionSnapshot{
				AssetID:          fullAssetID,
				AssetName:        infoAsset.assetInfo.name,
				AssetDescription: infoAsset.assetInfo.description,
			}

			atomicSnapshots = append(atomicSnapshots, issueStaticInfoSnapshot, assetDescription)
		}

		assetReissuability := &proto.AssetVolumeSnapshot{
			AssetID:       fullAssetID,
			IsReissuable:  infoAsset.assetInfo.reissuable,
			TotalQuantity: infoAsset.assetInfo.quantity,
		}

		atomicSnapshots = append(atomicSnapshots, assetReissuability)
	}
	return atomicSnapshots
}

func generateSnapshotsFromDataEntryUncertain(dataEntriesUncertain map[entryId]uncertainAccountsDataStorageEntry,
	scheme proto.Scheme) ([]proto.AtomicSnapshot, error) {
	dataEntries := make(map[proto.WavesAddress]proto.DataEntries)
	for entryID, entry := range dataEntriesUncertain {
		address, errCnvrt := entryID.addrID.ToWavesAddress(scheme)
		if errCnvrt != nil {
			return nil, errors.Wrap(errCnvrt, "failed to convert address id to waves address")
		}
		if _, ok := dataEntries[address]; ok {
			entries := dataEntries[address]
			entries = append(entries, entry.dataEntry)
			dataEntries[address] = entries
		} else {
			dataEntries[address] = proto.DataEntries{entry.dataEntry}
		}
	}
	var atomicSnapshots []proto.AtomicSnapshot
	for address, entries := range dataEntries {
		dataEntrySnapshot := &proto.DataEntriesSnapshot{Address: address, DataEntries: entries}
		atomicSnapshots = append(atomicSnapshots, dataEntrySnapshot)
	}
	return atomicSnapshots, nil
}

func generateSnapshotsFromAssetsScriptsUncertain(
	assetScriptsUncertain map[proto.AssetID]assetScriptRecordWithAssetIDTail) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for assetID, r := range assetScriptsUncertain {
		if r.scriptDBItem.script.IsEmpty() { // don't generate asset script snapshot for empty script
			continue
		}
		digest := proto.ReconstructDigest(assetID, r.assetIDTail)
		assetScrptSnapshot := &proto.AssetScriptSnapshot{
			AssetID: digest,
			Script:  r.scriptDBItem.script,
		}
		atomicSnapshots = append(atomicSnapshots, assetScrptSnapshot)
	}
	return atomicSnapshots
}

func generateSnapshotsFromLeasingsUncertain(
	leasesUncertain map[crypto.Digest]*leasing,
) ([]proto.AtomicSnapshot, []internalSnapshot, error) {
	var (
		atomicSnapshots   []proto.AtomicSnapshot
		internalSnapshots []internalSnapshot
	)
	for lID, l := range leasesUncertain {
		switch status := l.Status; status {
		case LeaseActive:
			newLeaseSnapshot := &proto.NewLeaseSnapshot{
				LeaseID:       lID,
				Amount:        l.Amount,
				SenderPK:      l.SenderPK,
				RecipientAddr: l.RecipientAddr,
			}
			leaseStatusActiveSnapshot := &InternalNewLeaseInfoSnapshot{
				LeaseID:             lID,
				OriginHeight:        l.OriginHeight,
				OriginTransactionID: l.OriginTransactionID,
			}
			atomicSnapshots = append(atomicSnapshots, newLeaseSnapshot)
			internalSnapshots = append(internalSnapshots, leaseStatusActiveSnapshot)
		case LeaseCancelled:
			// the atomic snapshots order is important here, don't change it
			if origTxID := l.OriginTransactionID; origTxID != nil { // can be nil if a node has been worked in the light mode
				cancelTxID := l.CancelTransactionID // can't be nil here because node is working in the full mode
				if bytes.Equal(origTxID[:], cancelTxID[:]) {
					newLeaseSnapshot := &proto.NewLeaseSnapshot{
						LeaseID:       lID,
						Amount:        l.Amount,
						SenderPK:      l.SenderPK,
						RecipientAddr: l.RecipientAddr,
					}
					leaseStatusActiveSnapshot := &InternalNewLeaseInfoSnapshot{
						LeaseID:             lID,
						OriginHeight:        l.OriginHeight,
						OriginTransactionID: l.OriginTransactionID,
					}
					atomicSnapshots = append(atomicSnapshots, newLeaseSnapshot)
					internalSnapshots = append(internalSnapshots, leaseStatusActiveSnapshot)
				}
			}
			cancelledLeaseSnapshot := &proto.CancelledLeaseSnapshot{
				LeaseID: lID,
			}
			leaseStatusCancelledSnapshot := &InternalCancelledLeaseInfoSnapshot{
				LeaseID:             lID,
				CancelHeight:        l.CancelHeight,
				CancelTransactionID: l.CancelTransactionID,
			}
			atomicSnapshots = append(atomicSnapshots, cancelledLeaseSnapshot)
			internalSnapshots = append(internalSnapshots, leaseStatusCancelledSnapshot)
		default:
			return nil, nil, errors.Errorf("invalid lease status value (%d)", status)
		}
	}
	return atomicSnapshots, internalSnapshots, nil
}

func generateSnapshotsFromSponsoredAssetsUncertain(
	sponsoredAssetsUncertain map[proto.AssetID]uncertainSponsoredAsset) []proto.AtomicSnapshot {
	var atomicSnapshots []proto.AtomicSnapshot
	for _, sponsored := range sponsoredAssetsUncertain {
		sponsorshipSnapshot := proto.SponsorshipSnapshot{
			AssetID:         sponsored.assetID,
			MinSponsoredFee: sponsored.assetCost,
		}
		atomicSnapshots = append(atomicSnapshots, sponsorshipSnapshot)
	}
	return atomicSnapshots
}

func (sg *snapshotGenerator) generateSnapshotForInvokeScript(
	scriptRecipient proto.Recipient,
	balanceChanges txDiff,
	scriptEstimation *scriptEstimation,
) (txSnapshot, error) {
	snapshot, err := sg.snapshotForInvoke(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	if scriptEstimation.isPresent() {
		// script estimation is present an not nil
		// we've pulled up an old script which estimation had been done by an old estimator
		// in txChecker we've estimated script with a new estimator
		// this is the place where we have to store new estimation
		scriptAddr, cnvrtErr := recipientToAddress(scriptRecipient, sg.stor.aliases)
		if cnvrtErr != nil {
			return txSnapshot{}, errors.Wrap(cnvrtErr, "failed to get sender for InvokeScriptWithProofs")
		}
		internalSnapshotDAppUpdateCmplx := &InternalDAppUpdateComplexitySnapshot{
			ScriptAddress: scriptAddr,
			Estimation:    scriptEstimation.estimation,
			ScriptIsEmpty: scriptEstimation.scriptIsEmpty,
		}
		snapshot.internal = append(snapshot.internal, internalSnapshotDAppUpdateCmplx)
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) snapshotForInvoke(balanceChanges txDiff) (txSnapshot, error) {
	var snapshot txSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Remove the just issues snapshot from the diff, because it's not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots.
	var specialAssetsSnapshots []proto.AssetBalanceSnapshot
	for key, diffAmount := range addrAssetBalanceDiff {
		uncertainAssets := sg.stor.assets.uncertainAssetInfo
		if _, ok := uncertainAssets[key.asset]; ok {
			// remove the element from the map
			delete(addrAssetBalanceDiff, key)
			fullAssetID := proto.ReconstructDigest(key.asset, uncertainAssets[key.asset].assetInfo.tail)
			specialAssetSnapshot := proto.AssetBalanceSnapshot{
				Address: key.address,
				AssetID: fullAssetID,
				Balance: uint64(diffAmount),
			}
			specialAssetsSnapshots = append(specialAssetsSnapshots, specialAssetSnapshot)
		}
	}

	assetsUncertain := sg.stor.assets.uncertainAssetInfo
	dataEntriesUncertain := sg.stor.accountsDataStor.uncertainEntries
	assetScriptsUncertain := sg.stor.scriptsStorage.uncertainAssetScriptsCopy()
	leasesUncertain := sg.stor.leases.uncertainLeases
	sponsoredAssetsUncertain := sg.stor.sponsoredAssets.uncertainSponsoredAssets

	assetsSnapshots := generateSnapshotsFromAssetsUncertain(assetsUncertain)
	snapshot.regular = append(snapshot.regular, assetsSnapshots...)

	dataEntriesSnapshots, err := generateSnapshotsFromDataEntryUncertain(dataEntriesUncertain, sg.scheme)
	if err != nil {
		return txSnapshot{}, err
	}
	snapshot.regular = append(snapshot.regular, dataEntriesSnapshots...)

	assetsScriptsSnapshots := generateSnapshotsFromAssetsScriptsUncertain(assetScriptsUncertain)
	snapshot.regular = append(snapshot.regular, assetsScriptsSnapshots...)

	leasingSnapshots, leasingInternalSnapshots, err := generateSnapshotsFromLeasingsUncertain(leasesUncertain)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to generate leasing snapshots")
	}
	snapshot.regular = append(snapshot.regular, leasingSnapshots...)
	snapshot.internal = append(snapshot.internal, leasingInternalSnapshots...)

	sponsoredAssetsSnapshots := generateSnapshotsFromSponsoredAssetsUncertain(sponsoredAssetsUncertain)
	snapshot.regular = append(snapshot.regular, sponsoredAssetsSnapshots...)

	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &leaseBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &assetBalancesSnapshot[i])
	}
	for i := range specialAssetsSnapshots {
		snapshot.regular = append(snapshot.regular, &specialAssetsSnapshots[i])
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForInvokeExpressionTx(balanceChanges txDiff) (txSnapshot, error) {
	return sg.snapshotForInvoke(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForEthereumInvokeScriptTx(
	balanceChanges txDiff,
) (txSnapshot, error) {
	return sg.snapshotForInvoke(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(
	assetID crypto.Digest,
	assetName string,
	assetDescription string,
	balanceChanges txDiff,
) (txSnapshot, error) {
	snapshot, err := sg.generateBalancesSnapshot(balanceChanges)
	if err != nil {
		return txSnapshot{}, err
	}
	assetDescriptionSnapshot := &proto.AssetDescriptionSnapshot{
		AssetID:          assetID,
		AssetName:        assetName,
		AssetDescription: assetDescription,
	}
	snapshot.regular = append(snapshot.regular, assetDescriptionSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateOrderAtomicSnapshot(orderID []byte,
	volume uint64, fee uint64) (*proto.FilledVolumeFeeSnapshot, error) {
	newestFilledAmount, newestFilledFee, err := sg.stor.ordersVolumes.newestFilled(orderID)
	if err != nil {
		return nil, err
	}
	orderIDDigset, err := crypto.NewDigestFromBytes(orderID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct digest from order id bytes")
	}
	// TODO must be added to newest filled amounts and fee
	orderSnapshot := &proto.FilledVolumeFeeSnapshot{
		OrderID:      orderIDDigset,
		FilledFee:    newestFilledFee + fee,
		FilledVolume: newestFilledAmount + volume,
	}
	return orderSnapshot, nil
}

func (sg *snapshotGenerator) generateBalancesSnapshot(balanceChanges txDiff) (txSnapshot, error) {
	var snapshot txSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to build a snapshot from a genesis transaction")
	}
	for i := range wavesBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &wavesBalancesSnapshot[i])
	}
	for i := range leaseBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &leaseBalancesSnapshot[i])
	}
	for i := range assetBalancesSnapshot {
		snapshot.regular = append(snapshot.regular, &assetBalancesSnapshot[i])
	}
	return snapshot, nil
}

func (sg *snapshotGenerator) generateBalancesAtomicSnapshots(
	addrWavesBalanceDiff addressWavesBalanceDiff,
	addrAssetBalanceDiff addressAssetBalanceDiff) (
	[]proto.WavesBalanceSnapshot,
	[]proto.AssetBalanceSnapshot,
	[]proto.LeaseBalanceSnapshot, error) {
	wavesBalanceSnapshot, leaseBalanceSnapshot, err := sg.wavesBalanceSnapshotFromBalanceDiff(addrWavesBalanceDiff)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to construct waves balance snapshot")
	}
	if len(addrAssetBalanceDiff) == 0 {
		return wavesBalanceSnapshot, nil, leaseBalanceSnapshot, nil
	}

	assetBalanceSnapshot, err := sg.assetBalanceSnapshotFromBalanceDiff(addrAssetBalanceDiff)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, leaseBalanceSnapshot, nil
}

func balanceDiffFromTxDiff(diff txDiff, scheme proto.Scheme) (addressWavesBalanceDiff, addressAssetBalanceDiff, error) {
	addrWavesBalanceDiff := make(addressWavesBalanceDiff)
	addrAssetBalanceDiff := make(addressAssetBalanceDiff)
	for balanceKeyString, diffAmount := range diff {
		// construct address from key
		wavesBalanceKey := &wavesBalanceKey{}
		err := wavesBalanceKey.unmarshal([]byte(balanceKeyString))
		if err != nil {
			// if the waves balance unmarshal failed, try to marshal into asset balance, and if it fails, then return the error
			assetBalanceKey := &assetBalanceKey{}
			mrshlErr := assetBalanceKey.unmarshal([]byte(balanceKeyString))
			if mrshlErr != nil {
				return nil, nil, errors.Wrap(mrshlErr, "failed to convert balance key to asset balance key")
			}
			asset := assetBalanceKey.asset
			address, cnvrtErr := assetBalanceKey.address.ToWavesAddress(scheme)
			if cnvrtErr != nil {
				return nil, nil, errors.Wrap(cnvrtErr, "failed to convert address id to waves address")
			}
			assetBalKey := assetBalanceDiffKey{address: address, asset: asset}
			addrAssetBalanceDiff[assetBalKey] = diffAmount.balance
			continue
		}
		address, cnvrtErr := wavesBalanceKey.address.ToWavesAddress(scheme)
		if cnvrtErr != nil {
			return nil, nil, errors.Wrap(cnvrtErr, "failed to convert address id to waves address")
		}
		// if the waves balance diff is 0, it means it did not change.
		// The reason for the 0 diff to exist is because of how LeaseIn and LeaseOut are handled in transaction differ.
		addrWavesBalanceDiff[address] = diffAmount
	}
	return addrWavesBalanceDiff, addrAssetBalanceDiff, nil
}

// from txDiff and fees. no validation needed at this point.
func (sg *snapshotGenerator) wavesBalanceSnapshotFromBalanceDiff(
	diff addressWavesBalanceDiff) ([]proto.WavesBalanceSnapshot, []proto.LeaseBalanceSnapshot, error) {
	var wavesBalances []proto.WavesBalanceSnapshot
	var leaseBalances []proto.LeaseBalanceSnapshot
	// add miner address to the diff

	for wavesAddress, diffAmount := range diff {
		fullBalance, err := sg.stor.balances.newestWavesBalance(wavesAddress.ID())
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		if diffAmount.balance != 0 {
			newBalance := proto.WavesBalanceSnapshot{
				Address: wavesAddress,
				Balance: uint64(int64(fullBalance.balance) + diffAmount.balance),
			}
			wavesBalances = append(wavesBalances, newBalance)
		}
		if diffAmount.leaseIn != 0 || diffAmount.leaseOut != 0 {
			newLeaseBalance := proto.LeaseBalanceSnapshot{
				Address:  wavesAddress,
				LeaseIn:  uint64(fullBalance.leaseIn + diffAmount.leaseIn),
				LeaseOut: uint64(fullBalance.leaseOut + diffAmount.leaseOut),
			}
			leaseBalances = append(leaseBalances, newLeaseBalance)
		}
	}
	return wavesBalances, leaseBalances, nil
}

func (sg *snapshotGenerator) assetBalanceSnapshotFromBalanceDiff(
	diff addressAssetBalanceDiff) ([]proto.AssetBalanceSnapshot, error) {
	var assetBalances []proto.AssetBalanceSnapshot
	// add miner address to the diff

	for key, diffAmount := range diff {
		balance, err := sg.stor.balances.newestAssetBalance(key.address.ID(), key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to receive sender's waves balance")
		}
		assetInfo, err := sg.stor.assets.newestAssetInfo(key.asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get newest asset info")
		}

		newBalance := proto.AssetBalanceSnapshot{
			Address: key.address,
			AssetID: key.asset.Digest(assetInfo.tail),
			Balance: uint64(int64(balance) + diffAmount),
		}
		assetBalances = append(assetBalances, newBalance)
	}
	return assetBalances, nil
}
