package state

import (
	"bytes"
	"math/big"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
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
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	_, ok := transaction.(*proto.Genesis)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to genesis transaction")
	}
	return sg.generateSnapshotForGenesisTx(balanceChanges)
}

func (sg *snapshotGenerator) performPayment(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	_, ok := transaction.(*proto.Payment)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to payment transaction")
	}
	return sg.generateSnapshotForPaymentTx(balanceChanges)
}

func (sg *snapshotGenerator) performTransferWithSig(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with sig transaction")
	}
	return sg.generateSnapshotForTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performTransferWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	_, ok := transaction.(*proto.TransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to transfer with proofs transaction")
	}
	return sg.generateSnapshotForTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performIssueWithSig(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performIssueWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performReissueWithSig(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return sg.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (sg *snapshotGenerator) performReissueWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return sg.generateSnapshotForReissueTx(tx.AssetID, tx.Reissuable, tx.Quantity, balanceChanges)
}

func (sg *snapshotGenerator) performBurnWithSig(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return sg.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
}

func (sg *snapshotGenerator) performBurnWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return sg.generateSnapshotForBurnTx(tx.AssetID, tx.Amount, balanceChanges)
}

func (sg *snapshotGenerator) performExchange(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performLeaseWithSig(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return sg.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (sg *snapshotGenerator) performLeaseWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return sg.generateSnapshotForLeaseTx(&tx.Lease, tx.ID, info, balanceChanges)
}

func (sg *snapshotGenerator) performLeaseCancelWithSig(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performLeaseCancelWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performCreateAliasWithSig(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return sg.generateSnapshotForCreateAliasTx(sg.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (sg *snapshotGenerator) performCreateAliasWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return sg.generateSnapshotForCreateAliasTx(sg.scheme, tx.SenderPK, tx.Alias, balanceChanges)
}

func (sg *snapshotGenerator) performMassTransferWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	_, ok := transaction.(*proto.MassTransferWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return sg.generateSnapshotForMassTransferTx(balanceChanges)
}

func (sg *snapshotGenerator) performDataWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performSponsorshipWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	return sg.generateSnapshotForSponsorshipTx(tx.AssetID, tx.MinAssetFee, balanceChanges)
}

func (sg *snapshotGenerator) performSetScriptWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performSetAssetScriptWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) performInvokeScriptWithProofs(
	transaction proto.Transaction,
	info *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	tx, ok := transaction.(*proto.InvokeScriptWithProofs)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	se := info.checkerData.scriptEstimation
	return sg.generateSnapshotForInvokeScript(tx.ScriptRecipient, balanceChanges, se)
}

func (sg *snapshotGenerator) performInvokeExpressionWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return txSnapshot{}, errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	return sg.generateSnapshotForInvokeExpressionTx(balanceChanges)
}

func (sg *snapshotGenerator) performEthereumTransactionWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return txSnapshot{}, errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	kind, err := proto.GuessEthereumTransactionKindType(ethTx.Data())
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to guess ethereum tx kind")
	}
	switch kind {
	case proto.EthereumTransferWavesKindType, proto.EthereumTransferAssetsKindType:
		return sg.generateSnapshotForTransferTx(balanceChanges) // like regular transfer
	case proto.EthereumInvokeKindType:
		return sg.snapshotForInvoke(balanceChanges) // like invoke script
	default:
		return txSnapshot{}, errors.Errorf("unexpected ethereum tx kind (%d)", kind)
	}
}

func (sg *snapshotGenerator) performUpdateAssetInfoWithProofs(
	transaction proto.Transaction,
	_ *performerInfo,
	balanceChanges []balanceChanges,
) (txSnapshot, error) {
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

func (sg *snapshotGenerator) generateSnapshotForGenesisTx(balanceChanges []balanceChanges) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForPaymentTx(balanceChanges []balanceChanges) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForTransferTx(balanceChanges []balanceChanges) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForIssueTx(
	tx *proto.Issue,
	assetID crypto.Digest,
	info *performerInfo,
	balanceChanges []balanceChanges,
	scriptEstimation *scriptEstimation,
	script proto.Script,
) (txSnapshot, error) {
	// Create new asset.
	blockHeight := info.blockHeight()
	senderPK := tx.SenderPK
	ai := assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:        proto.DigestTail(assetID),
			Issuer:      senderPK,
			Decimals:    tx.Decimals,
			IssueHeight: blockHeight,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}
	if err := ai.initIsNFTFlag(sg.stor.features); err != nil {
		return txSnapshot{}, errors.Wrapf(err, "failed to initialize NFT flag for issued asset %s", assetID.String())
	}

	addrWavesBalanceDiff, addrAssetBalanceDiff, err := sg.balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Just issued Asset IDs are not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots, unless we specify full asset ids from uncertain info
	justIssuedAssets := justIssuedAssetsIDsToTails{
		proto.AssetIDFromDigest(assetID): proto.DigestTail(assetID),
	}

	var snapshot txSnapshot

	issueStaticInfoSnapshot := &proto.NewAssetSnapshot{
		AssetID:         assetID,
		IssuerPublicKey: senderPK,
		Decimals:        ai.Decimals,
		IsNFT:           ai.IsNFT,
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
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff, justIssuedAssets)
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

func (sg *snapshotGenerator) generateSnapshotForReissueTx(
	assetID crypto.Digest,
	isReissuableFromTx bool,
	quantity uint64,
	balanceChanges []balanceChanges,
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
	// We should combine reissuable flag from the transaction and from the storage.
	// For more info see 'settings.FunctionalitySettings.CanReissueNonReissueablePeriod'
	isReissuable := isReissuableFromTx && assetInfo.reissuable
	assetReissuability := &proto.AssetVolumeSnapshot{
		AssetID:       assetID,
		TotalQuantity: *resQuantity,
		IsReissuable:  isReissuable,
	}
	snapshot.regular = append(snapshot.regular, assetReissuability)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForBurnTx(assetID crypto.Digest, newQuantity uint64,
	balanceChanges []balanceChanges) (txSnapshot, error) {
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
	balanceChanges []balanceChanges) (txSnapshot, error) {
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
	balanceChanges []balanceChanges,
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
	balanceChanges []balanceChanges,
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
	balanceChanges []balanceChanges,
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
		Alias:   alias.Alias,
	}
	snapshot.regular = append(snapshot.regular, aliasSnapshot)
	return snapshot, nil
}

func (sg *snapshotGenerator) generateSnapshotForMassTransferTx(balanceChanges []balanceChanges) (txSnapshot, error) {
	return sg.generateBalancesSnapshot(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForDataTx(senderAddress proto.WavesAddress,
	entries []proto.DataEntry, balanceChanges []balanceChanges) (txSnapshot, error) {
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
	minAssetFee uint64, balanceChanges []balanceChanges) (txSnapshot, error) {
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
	scriptEstimation scriptEstimation, balanceChanges []balanceChanges) (txSnapshot, error) {
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
	balanceChanges []balanceChanges, scriptEstimation scriptEstimation) (txSnapshot, error) {
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
		fullAssetID := proto.ReconstructDigest(assetID, infoAsset.assetInfo.Tail)
		// order of snapshots here is important: static info snapshot should be first
		if infoAsset.wasJustIssued {
			issueStaticInfoSnapshot := &proto.NewAssetSnapshot{
				AssetID:         fullAssetID,
				IssuerPublicKey: infoAsset.assetInfo.Issuer,
				Decimals:        infoAsset.assetInfo.Decimals,
				IsNFT:           infoAsset.assetInfo.IsNFT, // NFT flag is set in
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
	balanceChanges []balanceChanges,
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

type justIssuedAssetsIDsToTails map[proto.AssetID][proto.AssetIDTailSize]byte // asset id -> asset id tail

func (sg *snapshotGenerator) snapshotForInvoke(balanceChanges []balanceChanges) (txSnapshot, error) {
	var snapshot txSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := sg.balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	// Just issued Asset IDs are not in the storage yet,
	// so can't be processed with generateBalancesAtomicSnapshots, unless we specify full asset ids from uncertain info
	justIssuedAssets := justIssuedAssetsIDsToTails{}
	for key := range addrAssetBalanceDiff {
		uncertainAssets := sg.stor.assets.uncertainAssetInfo
		if uncertainInfo, ok := uncertainAssets[key.asset]; ok {
			if !uncertainInfo.wasJustIssued {
				continue
			}
			justIssuedAssets[key.asset] = uncertainInfo.assetInfo.Tail
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
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff, justIssuedAssets)
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

func (sg *snapshotGenerator) generateSnapshotForInvokeExpressionTx(
	balanceChanges []balanceChanges) (txSnapshot, error) {
	return sg.snapshotForInvoke(balanceChanges)
}

func (sg *snapshotGenerator) generateSnapshotForUpdateAssetInfoTx(
	assetID crypto.Digest,
	assetName string,
	assetDescription string,
	balanceChanges []balanceChanges,
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

func (sg *snapshotGenerator) createInitialBlockSnapshot(minerAndRewardChanges []balanceChanges) (txSnapshot, error) {
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := sg.balanceDiffFromTxDiff(minerAndRewardChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	var snapshot txSnapshot
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff, nil)
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

func (sg *snapshotGenerator) generateBalancesSnapshot(balanceChanges []balanceChanges) (txSnapshot, error) {
	var snapshot txSnapshot
	addrWavesBalanceDiff, addrAssetBalanceDiff, err := sg.balanceDiffFromTxDiff(balanceChanges, sg.scheme)
	if err != nil {
		return txSnapshot{}, errors.Wrap(err, "failed to create balance diff from tx diff")
	}
	wavesBalancesSnapshot, assetBalancesSnapshot, leaseBalancesSnapshot, err :=
		sg.generateBalancesAtomicSnapshots(addrWavesBalanceDiff, addrAssetBalanceDiff, nil)
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
	addrAssetBalanceDiff addressAssetBalanceDiff, justIssuedAssets justIssuedAssetsIDsToTails) (
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

	assetBalanceSnapshot, err := sg.assetBalanceSnapshotFromBalanceDiff(addrAssetBalanceDiff, justIssuedAssets)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to construct asset balance snapshot")
	}
	return wavesBalanceSnapshot, assetBalanceSnapshot, leaseBalanceSnapshot, nil
}

func (sg *snapshotGenerator) addAssetBalanceDiffFromTxDiff(change balanceDiff, assetKey []byte, scheme proto.Scheme,
	addrAssetBalanceDiff addressAssetBalanceDiff) error {
	if change.balance == 0 {
		return nil
	}
	assetBalanceK := &assetBalanceKey{}
	if err := assetBalanceK.unmarshal(assetKey); err != nil {
		return errors.Errorf("failed to unmarshal asset balance key: %v", err)
	}
	asset := assetBalanceK.asset
	address, cnvrtErr := assetBalanceK.address.ToWavesAddress(scheme)
	if cnvrtErr != nil {
		return errors.Wrap(cnvrtErr, "failed to convert address id to waves address")
	}
	assetBalKey := assetBalanceDiffKey{address: address, asset: asset}
	addrAssetBalanceDiff[assetBalKey] = change.balance
	return nil
}

func (sg *snapshotGenerator) addWavesBalanceDiffFromTxDiff(change balanceDiff, wavesKey []byte, scheme proto.Scheme,
	addrWavesBalanceDiff addressWavesBalanceDiff) error {
	if change.balance == 0 && change.leaseOut == 0 && change.leaseIn == 0 {
		return nil
	}
	wavesBalanceK := &wavesBalanceKey{}
	if err := wavesBalanceK.unmarshal(wavesKey); err != nil {
		return errors.Errorf("failed to unmarshal waves balance key: %v", err)
	}
	address, cnvrtErr := wavesBalanceK.address.ToWavesAddress(scheme)
	if cnvrtErr != nil {
		return errors.Wrap(cnvrtErr, "failed to convert address id to waves address")
	}
	addrWavesBalanceDiff[address] = change
	return nil
}

func (sg *snapshotGenerator) balanceDiffFromTxDiff(balanceChanges []balanceChanges,
	scheme proto.Scheme) (addressWavesBalanceDiff, addressAssetBalanceDiff, error) {
	addrAssetBalanceDiff := make(addressAssetBalanceDiff)
	addrWavesBalanceDiff := make(addressWavesBalanceDiff)
	for _, balanceChange := range balanceChanges {
		if l := len(balanceChange.balanceDiffs); l != 1 {
			return nil, nil, errors.Errorf(
				"invalid balance diff count for the same address in the same block: want 1, got %d", l,
			)
		}
		switch len(balanceChange.key) {
		case wavesBalanceKeySize:
			err := sg.addWavesBalanceDiffFromTxDiff(balanceChange.balanceDiffs[0], balanceChange.key,
				scheme, addrWavesBalanceDiff)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to add waves balance from tx diff")
			}
		case assetBalanceKeySize:
			err := sg.addAssetBalanceDiffFromTxDiff(balanceChange.balanceDiffs[0], balanceChange.key,
				scheme, addrAssetBalanceDiff)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to add asset balance from tx diff")
			}
		default:
			return nil, nil,
				errors.Errorf("wrong key size to calculate balance diff from tx diff, key size is %d",
					len(balanceChange.key))
		}
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
			newBalance, bErr := common.AddInt(int64(fullBalance.balance), diffAmount.balance) // sum & sanity check
			if bErr != nil {
				return nil, nil, errors.Wrapf(bErr,
					"failed to calculate waves balance for addr %q: failed to add %d to %d",
					wavesAddress.String(), diffAmount.balance, fullBalance.balance,
				)
			}
			if newBalance < 0 { // sanity check
				return nil, nil, errors.Errorf("negative waves balance for addr %q", wavesAddress.String())
			}
			newBalanceSnapshot := proto.WavesBalanceSnapshot{
				Address: wavesAddress,
				Balance: uint64(newBalance),
			}
			wavesBalances = append(wavesBalances, newBalanceSnapshot)
		}
		if diffAmount.leaseIn != 0 || diffAmount.leaseOut != 0 {
			// Don't check for overflow & negative leaseIn/leaseOut because overflowed addresses
			// See `balances.generateLeaseBalanceSnapshotsForLeaseOverflows` for details
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
	diff addressAssetBalanceDiff,
	justIssuedAssets justIssuedAssetsIDsToTails,
) ([]proto.AssetBalanceSnapshot, error) {
	assetBalances := make([]proto.AssetBalanceSnapshot, 0, len(diff))
	// add miner address to the diff

	for key, diffAmount := range diff {
		balance, balErr := sg.stor.balances.newestAssetBalance(key.address.ID(), key.asset)
		if balErr != nil {
			return nil, errors.Wrapf(balErr, "failed to receive sender's %q waves balance", key.address.String())
		}
		assetIDTail, ok := justIssuedAssets[key.asset]
		if !ok { // asset is not just issued, looking in the storage
			constInfo, err := sg.stor.assets.newestConstInfo(key.asset)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get newest asset info for short assetID %q",
					key.asset.String(),
				)
			}
			assetIDTail = constInfo.Tail
		}
		fullAssetID := key.asset.Digest(assetIDTail)
		newBalance, err := common.AddInt(int64(balance), diffAmount) // sum & sanity check
		if err != nil {
			return nil, errors.Wrapf(err, "failed to calculate asset %q balance: failed to add %d to %d",
				fullAssetID.String(), diffAmount, balance,
			)
		}
		if newBalance < 0 { // sanity check
			return nil, errors.Errorf("negative asset %q balance %d", fullAssetID.String(), newBalance)
		}
		newBalanceSnapshot := proto.AssetBalanceSnapshot{
			Address: key.address,
			AssetID: fullAssetID,
			Balance: uint64(newBalance),
		}
		assetBalances = append(assetBalances, newBalanceSnapshot)
	}
	return assetBalances, nil
}
