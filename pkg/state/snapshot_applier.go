package state

import (
	stderrs "errors"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

type txSnapshotContext struct {
	initialized   bool
	validatingUTX bool
	applyingTx    proto.Transaction
}

type blockSnapshotsApplier struct {
	info *blockSnapshotsApplierInfo
	stor snapshotApplierStorages

	txSnapshotContext txSnapshotContext

	issuedAssets   []crypto.Digest
	scriptedAssets map[crypto.Digest]struct{}

	newLeases       []crypto.Digest
	cancelledLeases map[crypto.Digest]struct{}

	// used for legacy SH
	balanceRecordsContext BalanceRecordsContext
}

func (a *blockSnapshotsApplier) BeforeTxSnapshotApply(tx proto.Transaction, validatingUTX bool) error {
	a.txSnapshotContext = txSnapshotContext{
		initialized:   true,
		validatingUTX: validatingUTX,
		applyingTx:    tx,
	}
	a.issuedAssets = []crypto.Digest{}
	if len(a.scriptedAssets) != 0 {
		a.scriptedAssets = make(map[crypto.Digest]struct{})
	}
	a.newLeases = []crypto.Digest{}
	if len(a.cancelledLeases) != 0 {
		a.cancelledLeases = make(map[crypto.Digest]struct{})
	}
	return nil
}

func (a *blockSnapshotsApplier) AfterTxSnapshotApply() error {
	for _, assetID := range a.issuedAssets {
		if _, ok := a.scriptedAssets[assetID]; ok { // don't set an empty script for scripted assets or script updates
			continue
		}
		emptyAssetScriptSnapshot := proto.AssetScriptSnapshot{
			AssetID: assetID,
			Script:  proto.Script{},
		}
		// need for compatibility with legacy state hashes
		// for issued asset without script we have to apply empty asset script snapshot
		err := emptyAssetScriptSnapshot.Apply(a)
		if err != nil {
			return errors.Wrapf(err, "failed to apply empty asset scipt snapshot for asset %q", assetID)
		}
	}
	for _, leaseID := range a.newLeases { // need for compatibility with legacy state hashes
		if _, ok := a.cancelledLeases[leaseID]; ok { // skip cancelled leases
			continue
		}
		if err := a.stor.leases.pushStateHash(leaseID, true, a.info.BlockID()); err != nil {
			return errors.Wrapf(err, "failed to push state hash for new lease %q", leaseID)
		}
	}
	for cancelledLeaseID := range a.cancelledLeases {
		if err := a.stor.leases.pushStateHash(cancelledLeaseID, false, a.info.BlockID()); err != nil {
			return errors.Wrapf(err, "failed to push state hash for cancelled lease %q", cancelledLeaseID)
		}
	}

	a.txSnapshotContext = txSnapshotContext{} // reset to default
	return nil
}

func newBlockSnapshotsApplier(info *blockSnapshotsApplierInfo, stor snapshotApplierStorages) blockSnapshotsApplier {
	return blockSnapshotsApplier{
		info:                  info,
		stor:                  stor,
		issuedAssets:          []crypto.Digest{},
		scriptedAssets:        make(map[crypto.Digest]struct{}),
		newLeases:             []crypto.Digest{},
		cancelledLeases:       make(map[crypto.Digest]struct{}),
		balanceRecordsContext: NewBalanceRecordsContext(),
	}
}

type BalanceRecordsContext struct {
	// used for legacy state hashes to filter out statehash temporary records with 0 change in a block.
	wavesBalanceRecords  wavesBalanceRecords
	assetBalanceRecords  assetBalanceRecords
	leasesBalanceRecords leaseBalanceRecords
}

func NewBalanceRecordsContext() BalanceRecordsContext {
	return BalanceRecordsContext{
		wavesBalanceRecords:  wavesBalanceRecords{make(map[string]balanceRecordInBlock)},
		assetBalanceRecords:  assetBalanceRecords{make(map[string]balanceRecordInBlock)},
		leasesBalanceRecords: leaseBalanceRecords{make(map[string]leaseRecordsInBlock)},
	}
}

func (a *blockSnapshotsApplier) addWavesBalanceRecordLegacySH(address proto.Address, balance int64) error {
	if !a.stor.balances.calculateHashes {
		return nil
	}
	key := wavesBalanceKey{address: address.ID()}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	return a.addWavesBalanceRecord(keyStr, address, balance)
}

func (a *blockSnapshotsApplier) addAssetBalanceRecordLegacySH(address proto.Address, asset proto.AssetID, balance int64) error {
	if !a.stor.balances.calculateHashes {
		return nil
	}
	key := assetBalanceKey{address: address.ID(), asset: asset}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	return a.addAssetBalanceRecord(keyStr, address, asset, balance)
}

func (a *blockSnapshotsApplier) addLeasesBalanceRecordLegacySH(address proto.Address, leaseIn int64, leaseOut int64) error {
	if !a.stor.balances.calculateHashes {
		return nil
	}
	key := wavesBalanceKey{address: address.ID()}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	return a.addLeaseBalanceRecord(keyStr, address, leaseIn, leaseOut)
}

func (a *blockSnapshotsApplier) filterZeroWavesDiffRecords(blockID proto.BlockID) {
	// comparing the final balance to the initial one
	for key, balanceRecord := range a.balanceRecordsContext.wavesBalanceRecords.wavesRecords {
		if balanceRecord.current == balanceRecord.initial { // this means the diff is 0 in block
			temporarySHRecords, ok := a.stor.balances.wavesHashesState[blockID]
			if ok && temporarySHRecords != nil {
				temporarySHRecords.remove(key)
				a.stor.balances.wavesHashesState[blockID] = temporarySHRecords
			}
		}
	}
}

func (a *blockSnapshotsApplier) filterZeroAssetDiffRecords(blockID proto.BlockID) {
	// comparing the final balance to the initial one
	for key, balanceRecord := range a.balanceRecordsContext.assetBalanceRecords.assetRecords {
		if balanceRecord.current == balanceRecord.initial { // this means the diff is 0 in block
			temporarySHRecords, ok := a.stor.balances.assetsHashesState[blockID]
			if ok && temporarySHRecords != nil {
				temporarySHRecords.remove(key)
				a.stor.balances.assetsHashesState[blockID] = temporarySHRecords
			}
		}
	}
}

func (a *blockSnapshotsApplier) filterZeroLeasingDiffRecords(blockID proto.BlockID) {
	// comparing the final balance to the initial one
	for key, balanceRecord := range a.balanceRecordsContext.leasesBalanceRecords.leaseRecords {
		if balanceRecord.currentLeaseIn == balanceRecord.initialLeaseIn &&
			balanceRecord.currentLeaseOut == balanceRecord.initialLeaseOut { // this means the diff is 0 in block
			temporarySHRecords, ok := a.stor.balances.leaseHashesState[blockID]
			if ok && temporarySHRecords != nil {
				temporarySHRecords.remove(key)
				a.stor.balances.leaseHashesState[blockID] = temporarySHRecords
			}
		}
	}
}

func (a *blockSnapshotsApplier) filterZeroDiffsSHOut(blockID proto.BlockID) {
	if !a.stor.balances.calculateHashes {
		return
	}
	a.filterZeroWavesDiffRecords(blockID)
	a.filterZeroAssetDiffRecords(blockID)
	a.filterZeroLeasingDiffRecords(blockID)

	a.balanceRecordsContext.wavesBalanceRecords.reset()
	a.balanceRecordsContext.assetBalanceRecords.reset()
	a.balanceRecordsContext.leasesBalanceRecords.reset()
}

type balanceRecordInBlock struct {
	initial int64
	current int64
}

type wavesBalanceRecords struct {
	wavesRecords map[string]balanceRecordInBlock
}

func (a *blockSnapshotsApplier) addWavesBalanceRecord(keyStr string, address proto.Address, balance int64) error {
	prevRec, ok := a.balanceRecordsContext.wavesBalanceRecords.wavesRecords[keyStr]
	if ok {
		prevRec.current = balance
		a.balanceRecordsContext.wavesBalanceRecords.wavesRecords[keyStr] = prevRec
	} else {
		initialBalance, err := a.stor.balances.newestWavesBalance(address.ID())
		if err != nil {
			return errors.Wrapf(err, "failed to gen initial balance for address %s", address.String())
		}
		a.balanceRecordsContext.wavesBalanceRecords.wavesRecords[keyStr] = balanceRecordInBlock{initial: int64(initialBalance.balance), current: balance}
	}
	return nil
}

func (w *wavesBalanceRecords) reset() {
	w.wavesRecords = make(map[string]balanceRecordInBlock)
}

type assetBalanceRecords struct {
	assetRecords map[string]balanceRecordInBlock
}

func (a *blockSnapshotsApplier) addAssetBalanceRecord(keyStr string, address proto.Address, assetID proto.AssetID, balance int64) error {
	prevRec, ok := a.balanceRecordsContext.assetBalanceRecords.assetRecords[keyStr]
	if ok {
		prevRec.current = balance
		a.balanceRecordsContext.assetBalanceRecords.assetRecords[keyStr] = prevRec
	} else {
		initialBalance, err := a.stor.balances.newestAssetBalance(address.ID(), assetID)
		if err != nil {
			return errors.Wrapf(err, "failed to gen initial balance for address %s", address.String())
		}
		a.balanceRecordsContext.assetBalanceRecords.assetRecords[keyStr] = balanceRecordInBlock{initial: int64(initialBalance), current: balance}
	}
	return nil
}

func (w *assetBalanceRecords) reset() {
	w.assetRecords = make(map[string]balanceRecordInBlock)
}

type leaseRecordsInBlock struct {
	initialLeaseIn  int64
	initialLeaseOut int64
	currentLeaseIn  int64
	currentLeaseOut int64
}

type leaseBalanceRecords struct {
	leaseRecords map[string]leaseRecordsInBlock
}

func (a *blockSnapshotsApplier) addLeaseBalanceRecord(keyStr string, address proto.Address, leaseIn int64, leaseOut int64) error {
	prevLeaseInOut, ok := a.balanceRecordsContext.leasesBalanceRecords.leaseRecords[keyStr]
	if ok {
		prevLeaseInOut.currentLeaseIn = leaseIn
		prevLeaseInOut.currentLeaseOut = leaseOut
		a.balanceRecordsContext.leasesBalanceRecords.leaseRecords[keyStr] = prevLeaseInOut
	} else {
		initialBalance, err := a.stor.balances.newestWavesBalance(address.ID())
		if err != nil {
			return errors.Wrapf(err, "failed to gen initial balance for address %s", address.String())
		}
		a.balanceRecordsContext.leasesBalanceRecords.leaseRecords[keyStr] = leaseRecordsInBlock{
			initialLeaseIn:  initialBalance.leaseIn,
			initialLeaseOut: initialBalance.leaseOut,
			currentLeaseIn:  leaseIn,
			currentLeaseOut: leaseOut}
	}
	return nil
}

func (w *leaseBalanceRecords) reset() {
	w.leaseRecords = make(map[string]leaseRecordsInBlock)
}

type snapshotApplierStorages struct {
	rw                *blockReadWriter
	balances          *balances
	aliases           *aliases
	assets            *assets
	scriptsStorage    scriptStorageState
	scriptsComplexity *scriptsComplexity
	sponsoredAssets   *sponsoredAssets
	ordersVolumes     *ordersVolumes
	accountsDataStor  *accountsDataStorage
	leases            *leases
}

func newSnapshotApplierStorages(stor *blockchainEntitiesStorage, rw *blockReadWriter) snapshotApplierStorages {
	return snapshotApplierStorages{
		rw:                rw,
		balances:          stor.balances,
		aliases:           stor.aliases,
		assets:            stor.assets,
		scriptsStorage:    stor.scriptsStorage,
		scriptsComplexity: stor.scriptsComplexity,
		sponsoredAssets:   stor.sponsoredAssets,
		ordersVolumes:     stor.ordersVolumes,
		accountsDataStor:  stor.accountsDataStor,
		leases:            stor.leases,
	}
}

type blockSnapshotsApplierInfo struct {
	ci                  *checkerInfo
	scheme              proto.Scheme
	stateActionsCounter *proto.StateActionsCounter
}

func newBlockSnapshotsApplierInfo(ci *checkerInfo, scheme proto.Scheme,
	counter *proto.StateActionsCounter) *blockSnapshotsApplierInfo {
	return &blockSnapshotsApplierInfo{
		ci:                  ci,
		scheme:              scheme,
		stateActionsCounter: counter,
	}
}

func (s blockSnapshotsApplierInfo) BlockID() proto.BlockID {
	return s.ci.blockID
}

func (s blockSnapshotsApplierInfo) BlockchainHeight() proto.Height {
	return s.ci.blockchainHeight
}

func (s blockSnapshotsApplierInfo) CurrentBlockHeight() proto.Height {
	return s.BlockchainHeight() + 1
}

func (s blockSnapshotsApplierInfo) EstimatorVersion() int {
	return s.ci.estimatorVersion()
}

func (s blockSnapshotsApplierInfo) Scheme() proto.Scheme {
	return s.scheme
}

func (s blockSnapshotsApplierInfo) StateActionsCounter() *proto.StateActionsCounter {
	return s.stateActionsCounter
}

func (a *blockSnapshotsApplier) SetApplierInfo(info *blockSnapshotsApplierInfo) {
	a.info = info
}

func (a *blockSnapshotsApplier) ApplyWavesBalance(snapshot proto.WavesBalanceSnapshot) error {
	// for compatibility with the legacy state hashes
	err := a.addWavesBalanceRecordLegacySH(snapshot.Address, int64(snapshot.Balance))
	if err != nil {
		return err
	}
	addrID := snapshot.Address.ID()
	profile, err := a.stor.balances.newestWavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.balance = snapshot.Balance
	value := newWavesValue(profile, newProfile)
	if err = a.stor.balances.setWavesBalance(addrID, value, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyLeaseBalance(snapshot proto.LeaseBalanceSnapshot) error {
	err := a.addLeasesBalanceRecordLegacySH(snapshot.Address, int64(snapshot.LeaseIn), int64(snapshot.LeaseOut))
	if err != nil {
		return err
	}

	addrID := snapshot.Address.ID()
	profile, err := a.stor.balances.newestWavesBalance(addrID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest waves balance profile for address %q", snapshot.Address.String())
	}
	newProfile := profile
	newProfile.leaseIn = int64(snapshot.LeaseIn)
	newProfile.leaseOut = int64(snapshot.LeaseOut)
	value := newWavesValue(profile, newProfile)
	if err = a.stor.balances.setWavesBalance(addrID, value, a.info.BlockID()); err != nil {
		return errors.Wrapf(err, "failed to get set balance profile for address %q", snapshot.Address.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyAssetBalance(snapshot proto.AssetBalanceSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	// for compatibility with the legacy state hashes
	err := a.addAssetBalanceRecordLegacySH(snapshot.Address, assetID, int64(snapshot.Balance))
	if err != nil {
		return err
	}
	addrID := snapshot.Address.ID()
	err = a.stor.balances.setAssetBalance(addrID, assetID, snapshot.Balance, a.info.BlockID())
	if err != nil {
		return errors.Wrapf(err, "failed to set asset balance profile for address %q", snapshot.Address.String())
	}
	// for compatibility with the legacy state hashes
	return nil
}

func (a *blockSnapshotsApplier) ApplyAlias(snapshot proto.AliasSnapshot) error {
	return a.stor.aliases.createAlias(snapshot.Alias.Alias, snapshot.Address, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyNewAsset(snapshot proto.NewAssetSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	height := a.info.CurrentBlockHeight()

	assetFullInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:                 proto.DigestTail(snapshot.AssetID),
			issuer:               snapshot.IssuerPublicKey,
			decimals:             snapshot.Decimals,
			issueHeight:          height,
			issueSequenceInBlock: a.info.StateActionsCounter().NextIssueActionNumber(),
		},
		assetChangeableInfo: assetChangeableInfo{},
	}
	err := a.stor.assets.issueAsset(assetID, assetFullInfo, a.info.BlockID())
	if err != nil {
		return errors.Wrapf(err, "failed to issue asset %q", snapshot.AssetID.String())
	}
	a.issuedAssets = append(a.issuedAssets, snapshot.AssetID)
	return nil
}

func (a *blockSnapshotsApplier) ApplyAssetDescription(snapshot proto.AssetDescriptionSnapshot) error {
	change := &assetInfoChange{
		newName:        snapshot.AssetName,
		newDescription: snapshot.AssetDescription,
		newHeight:      a.info.CurrentBlockHeight(),
	}
	return a.stor.assets.updateAssetInfo(snapshot.AssetID, change, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetVolume(snapshot proto.AssetVolumeSnapshot) error {
	assetID := proto.AssetIDFromDigest(snapshot.AssetID)
	assetFullInfo, err := a.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return errors.Wrapf(err, "failed to get newest asset info for asset %q", snapshot.AssetID.String())
	}
	assetFullInfo.assetChangeableInfo.reissuable = snapshot.IsReissuable
	assetFullInfo.assetChangeableInfo.quantity = snapshot.TotalQuantity
	return a.stor.assets.storeAssetInfo(assetID, assetFullInfo, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAssetScript(snapshot proto.AssetScriptSnapshot) error {
	err := a.stor.scriptsStorage.setAssetScript(snapshot.AssetID, snapshot.Script, a.info.BlockID())
	if err != nil {
		return errors.Wrapf(err, "failed to apply asset script for asset %q", snapshot.AssetID)
	}
	if !snapshot.Script.IsEmpty() {
		a.scriptedAssets[snapshot.AssetID] = struct{}{}
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplySponsorship(snapshot proto.SponsorshipSnapshot) error {
	return a.stor.sponsoredAssets.sponsorAsset(snapshot.AssetID, snapshot.MinSponsoredFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyAccountScript(snapshot proto.AccountScriptSnapshot) error {
	addr, err := proto.NewAddressFromPublicKey(a.info.Scheme(), snapshot.SenderPublicKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create address from scheme %d and PK %q",
			a.info.Scheme(), snapshot.SenderPublicKey.String())
	}
	// In case of verifier, there are no functions. If it is a full DApp,
	// the complexity 'functions' will be stored through the internal snapshot InternalDAppComplexitySnapshot.
	treeEstimation := ride.TreeEstimation{
		Estimation: int(snapshot.VerifierComplexity),
		Verifier:   int(snapshot.VerifierComplexity),
		Functions:  nil,
	}
	setErr := a.stor.scriptsStorage.setAccountScript(addr, snapshot.Script, snapshot.SenderPublicKey, a.info.BlockID())
	if setErr != nil {
		return setErr
	}
	se := scriptEstimation{
		currentEstimatorVersion: 0, // 0 means unknown estimator version, script will be re-estimated in full node mode
		scriptIsEmpty:           snapshot.Script.IsEmpty(),
		estimation:              treeEstimation,
	}
	if cmplErr := a.stor.scriptsComplexity.saveComplexitiesForAddr(addr, se, a.info.BlockID()); cmplErr != nil {
		return errors.Wrapf(cmplErr, "failed to store account script estimation for addr %q",
			addr.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyFilledVolumeAndFee(snapshot proto.FilledVolumeFeeSnapshot) error {
	return a.stor.ordersVolumes.storeFilled(snapshot.OrderID.Bytes(),
		snapshot.FilledVolume, snapshot.FilledFee, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyDataEntries(snapshot proto.DataEntriesSnapshot) error {
	blockID := a.info.BlockID()
	for _, entry := range snapshot.DataEntries {
		if err := a.stor.accountsDataStor.appendEntry(snapshot.Address, entry, blockID); err != nil {
			return errors.Wrapf(err, "failed to add entry (%T) for address %q", entry, snapshot.Address)
		}
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyNewLease(snapshot proto.NewLeaseSnapshot) error {
	l := &leasing{
		SenderPK:      snapshot.SenderPK,
		RecipientAddr: snapshot.RecipientAddr,
		Amount:        snapshot.Amount,
		Status:        LeaseActive,
	}
	err := a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
	if err != nil {
		return errors.Wrapf(err, "failed to apply new lease %q", snapshot.LeaseID)
	}
	a.newLeases = append(a.newLeases, snapshot.LeaseID)
	return nil
}

func (a *blockSnapshotsApplier) ApplyCancelledLease(snapshot proto.CancelledLeaseSnapshot) error {
	l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get leasing info by id '%s' for cancelling", snapshot.LeaseID)
	}
	l.Status = LeaseCancelled
	err = a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
	if err != nil {
		return errors.Wrapf(err, "failed to cancel lease %q", snapshot.LeaseID)
	}
	a.cancelledLeases[snapshot.LeaseID] = struct{}{}
	return nil
}

func (a *blockSnapshotsApplier) ApplyTransactionsStatus(snapshot proto.TransactionStatusSnapshot) error {
	if !a.txSnapshotContext.initialized { // sanity check
		return errors.New("failed to apply transaction status snapshot: transaction is not set")
	}
	var (
		status        = snapshot.Status
		tx            = a.txSnapshotContext.applyingTx
		validatingUTX = a.txSnapshotContext.validatingUTX
	)
	var err error
	if validatingUTX {
		// Save transaction to in-mem storage.
		err = a.stor.rw.writeTransactionToMem(tx, status)
	} else {
		// Save transaction to in-mem storage and persistent storage.
		err = a.stor.rw.writeTransaction(tx, status)
	}
	if err != nil {
		txID, idErr := tx.GetID(a.info.Scheme())
		if idErr != nil {
			return errors.Wrapf(stderrs.Join(err, idErr),
				"failed to write transaction to storage, validatingUTX=%t", validatingUTX,
			)
		}
		return errors.Wrapf(err, "failed to write transaction %q to storage, validatingUTX=%t",
			base58.Encode(txID), validatingUTX,
		)
	}
	a.txSnapshotContext = txSnapshotContext{} // reset to default because transaction status should be applied only once
	return nil
}

func (a *blockSnapshotsApplier) ApplyDAppComplexity(snapshot InternalDAppComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Save full complexity of both callable and verifier when the script is set first time
	if setErr := a.stor.scriptsComplexity.saveComplexitiesForAddr(snapshot.ScriptAddress,
		scriptEstimation, a.info.BlockID()); setErr != nil {
		return errors.Wrapf(setErr, "failed to save script complexities for addr %q",
			snapshot.ScriptAddress.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyDAppUpdateComplexity(snapshot InternalDAppUpdateComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Update full complexity of both callable and verifier when the script is set first time
	if scErr := a.stor.scriptsComplexity.updateCallableComplexitiesForAddr(
		snapshot.ScriptAddress,
		scriptEstimation, a.info.BlockID()); scErr != nil {
		return errors.Wrapf(scErr, "failed to save complexity for addr %q",
			snapshot.ScriptAddress,
		)
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyAssetScriptComplexity(snapshot InternalAssetScriptComplexitySnapshot) error {
	scriptEstimation := scriptEstimation{currentEstimatorVersion: a.info.EstimatorVersion(),
		scriptIsEmpty: snapshot.ScriptIsEmpty, estimation: snapshot.Estimation}
	// Save complexity of verifier when the script is set first time
	if setErr := a.stor.scriptsComplexity.saveComplexitiesForAsset(snapshot.AssetID,
		scriptEstimation, a.info.BlockID()); setErr != nil {
		return errors.Wrapf(setErr, "failed to save script complexities for asset ID %q",
			snapshot.AssetID.String())
	}
	return nil
}

func (a *blockSnapshotsApplier) ApplyNewLeaseInfo(snapshot InternalNewLeaseInfoSnapshot) error {
	l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get leasing info by id '%s' for adding active info", snapshot.LeaseID)
	}
	l.OriginHeight = snapshot.OriginHeight
	l.OriginTransactionID = snapshot.OriginTransactionID
	return a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
}

func (a *blockSnapshotsApplier) ApplyCancelledLeaseInfo(snapshot InternalCancelledLeaseInfoSnapshot) error {
	l, err := a.stor.leases.newestLeasingInfo(snapshot.LeaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to get leasing info by id '%s' for adding cancel info", snapshot.LeaseID)
	}
	l.CancelHeight = snapshot.CancelHeight
	l.CancelTransactionID = snapshot.CancelTransactionID
	return a.stor.leases.rawWriteLeasing(snapshot.LeaseID, l, a.info.BlockID())
}
