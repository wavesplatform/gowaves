package ride

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type dataEntryKey struct {
	key     string
	address proto.WavesAddress
}

func (d *dataEntryKey) String() string {
	return fmt.Sprintf("%s|%s", d.key, d.address.String())
}

type lease struct {
	sender    proto.WavesAddress
	recipient proto.WavesAddress
	active    bool
	amount    int64
}

type diffBalance struct {
	balance         int64
	leaseIn         int64
	leaseOut        int64
	stateGenerating int64
}

func (db *diffBalance) addBalance(amount int64) error {
	b, err := common.AddInt64(db.balance, amount)
	if err != nil {
		return err
	}
	db.balance = b
	return nil
}

func (db *diffBalance) addLeaseIn(amount int64) error {
	b, err := common.AddInt64(db.leaseIn, amount)
	if err != nil {
		return err
	}
	db.leaseIn = b
	return nil
}

func (db *diffBalance) addLeaseOut(amount int64) error {
	b, err := common.AddInt64(db.leaseOut, amount)
	if err != nil {
		return err
	}
	db.leaseOut = b
	return nil
}

func (db *diffBalance) spendableBalance() (int64, error) {
	b, err := common.AddInt64(db.balance, -db.leaseOut)
	if err != nil {
		return 0, err
	}
	return b, nil
}

func (db *diffBalance) checkedRegularBalance() (uint64, error) {
	if db.balance < 0 {
		return 0, errors.New("negative regular balance")
	}
	return uint64(db.balance), nil
}

func (db *diffBalance) checkedSpendableBalance() (uint64, error) {
	b, err := common.AddInt64(db.balance, -db.leaseOut)
	if err != nil {
		return 0, err
	}
	if b < 0 {
		return 0, errors.New("negative spendable balance")
	}
	return uint64(b), nil
}

func (db *diffBalance) effectiveBalance() (int64, error) {
	v1, err := common.AddInt64(db.balance, db.leaseIn)
	if err != nil {
		return 0, err
	}
	v2, err := common.AddInt64(v1, -db.leaseOut)
	if err != nil {
		return 0, err
	}
	return v2, nil
}

func (db *diffBalance) toFullWavesBalance() (*proto.FullWavesBalance, error) {
	eff, err := db.effectiveBalance()
	if err != nil {
		return nil, err
	}
	if eff < 0 {
		return nil, errors.New("negative effective balance")
	}
	spb, err := db.spendableBalance()
	if err != nil {
		return nil, err
	}
	if spb < 0 {
		return nil, errors.New("negative spendable balance")
	}
	gen := eff
	if db.stateGenerating < gen {
		gen = db.stateGenerating
	}
	return &proto.FullWavesBalance{
		Regular:    uint64(db.balance),
		Generating: uint64(gen),
		Available:  uint64(spb),
		Effective:  uint64(eff),
		LeaseIn:    uint64(db.leaseIn),
		LeaseOut:   uint64(db.leaseOut),
	}, nil
}

type diffSponsorship struct {
	minFee int64
}

type diffNewAssetInfo struct {
	dAppIssuer  proto.WavesAddress
	name        string
	description string
	quantity    int64
	decimals    int32
	reissuable  bool
	script      []byte
	nonce       int64
}

type diffOldAssetInfo struct {
	diffQuantity int64
}

type assetBalanceKey struct {
	id    proto.AddressID
	asset crypto.Digest
}

type assetBalance int64

func (b assetBalance) add(amount int64) (assetBalance, error) {
	r, err := common.AddInt64(int64(b), amount)
	if err != nil {
		return 0, err
	}
	return assetBalance(r), nil
}

func (b assetBalance) checked() (uint64, error) {
	if b < 0 {
		return 0, errors.New("negative asset balance")
	}
	return uint64(b), nil
}

type changedAccountKey struct {
	account proto.AddressID
	asset   proto.OptionalAsset
}

type changedAccounts map[changedAccountKey]struct{}

func (b changedAccounts) addWavesBalanceChange(account proto.AddressID) {
	key := changedAccountKey{
		account: account,
		asset:   proto.NewOptionalAssetWaves(),
	}
	b[key] = struct{}{}
}

func (b changedAccounts) addAssetBalanceChange(account proto.AddressID, asset crypto.Digest) {
	key := changedAccountKey{
		account: account,
		asset:   *proto.NewOptionalAssetFromDigest(asset),
	}
	b[key] = struct{}{}
}

type diffState struct {
	state           types.SmartState
	data            map[dataEntryKey]proto.DataEntry
	wavesBalances   map[proto.AddressID]diffBalance
	assetBalances   map[assetBalanceKey]assetBalance
	sponsorships    map[crypto.Digest]diffSponsorship
	newAssetsInfo   map[crypto.Digest]diffNewAssetInfo
	oldAssetsInfo   map[crypto.Digest]diffOldAssetInfo
	leases          map[crypto.Digest]lease
	changedAccounts changedAccounts
}

func newDiffState(state types.SmartState) diffState {
	return diffState{
		state:           state,
		data:            map[dataEntryKey]proto.DataEntry{},
		wavesBalances:   map[proto.AddressID]diffBalance{},
		assetBalances:   map[assetBalanceKey]assetBalance{},
		sponsorships:    map[crypto.Digest]diffSponsorship{},
		newAssetsInfo:   map[crypto.Digest]diffNewAssetInfo{},
		oldAssetsInfo:   map[crypto.Digest]diffOldAssetInfo{},
		leases:          map[crypto.Digest]lease{},
		changedAccounts: make(changedAccounts),
	}
}

func (ds *diffState) replaceChangedAccounts(new changedAccounts) changedAccounts {
	old := ds.changedAccounts
	ds.changedAccounts = new
	return old
}

func (ds *diffState) loadWavesBalance(id proto.AddressID) (diffBalance, error) {
	// Look up for local diff for the account
	if diff, ok := ds.wavesBalances[id]; ok {
		return diff, nil
	}
	// In case of no balance diff found make new one from a full Waves balance from state
	profile, err := ds.state.WavesBalanceProfile(id)
	if err != nil {
		return diffBalance{}, errors.Wrap(err, "failed to get full Waves balance from state")
	}
	diff := diffBalance{
		balance:         int64(profile.Balance),
		leaseIn:         profile.LeaseIn,
		leaseOut:        profile.LeaseOut,
		stateGenerating: int64(profile.Generating),
	}
	// Store new diff locally
	ds.wavesBalances[id] = diff
	return diff, nil
}

func (ds *diffState) addWavesBalance(key proto.AddressID, amount int64) error {
	ds.changedAccounts.addWavesBalanceChange(key)

	if diff, ok := ds.wavesBalances[key]; ok {
		err := diff.addBalance(amount)
		if err != nil {
			return err // Int64 overflow error
		}
		ds.wavesBalances[key] = diff
		return nil
	}
	return errors.New("diff not found")
}

func (ds *diffState) loadAssetBalance(key assetBalanceKey) (assetBalance, error) {
	// Look up for local diff for the account
	if b, ok := ds.assetBalances[key]; ok {
		return b, nil
	}
	// In case of no balance diff found make new one from a full Waves balance from state
	balance, err := ds.state.NewestAssetBalanceByAddressID(key.id, key.asset)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get asset balance from state")
	}
	b := assetBalance(balance)
	// Store new diff locally
	ds.assetBalances[key] = b
	return b, nil
}

func (ds *diffState) addAssetBalance(key assetBalanceKey, amount int64) error {
	ds.changedAccounts.addAssetBalanceChange(key.id, key.asset)

	if b, ok := ds.assetBalances[key]; ok {
		r, err := b.add(amount)
		if err != nil {
			return err
		}
		ds.assetBalances[key] = r
		return nil
	}
	return errors.New("diff not found")
}

func (ds *diffState) reissueNewAsset(assetID crypto.Digest, quantity int64, reissuable bool) {
	assetInfo := ds.newAssetsInfo[assetID]
	assetInfo.reissuable = reissuable
	assetInfo.quantity += quantity
	ds.newAssetsInfo[assetID] = assetInfo
}

func (ds *diffState) burnNewAsset(assetID crypto.Digest, quantity int64) {
	assetInfo := ds.newAssetsInfo[assetID]
	assetInfo.quantity -= quantity
	ds.newAssetsInfo[assetID] = assetInfo
}

func (ds *diffState) loadLease(leaseID crypto.Digest) (lease, error) {
	// Look up for local lease for the account
	if l, ok := ds.leases[leaseID]; ok {
		return l, nil
	}
	// In case of no lease found load it from state
	leaseFromStore, err := ds.state.NewestLeasingInfo(leaseID)
	if err != nil {
		return lease{}, err
	}
	l := lease{
		active:    leaseFromStore.IsActive,
		recipient: leaseFromStore.Recipient,
		sender:    leaseFromStore.Sender,
		amount:    int64(leaseFromStore.LeaseAmount),
	}
	ds.leases[leaseID] = l
	return l, nil
}

// changeLeaseBalances changes sender's leaseOut and receiver's leaseIn by leasing amount
func (ds *diffState) changeLeaseBalances(sender, receiver proto.AddressID, amount int64) error {
	ds.changedAccounts.addWavesBalanceChange(sender)
	ds.changedAccounts.addWavesBalanceChange(receiver)

	senderDiff, err := ds.loadWavesBalance(sender)
	if err != nil {
		return err
	}
	// changes sender's leaseOut by amount
	if err := senderDiff.addLeaseOut(amount); err != nil {
		return err
	}
	ds.wavesBalances[sender] = senderDiff
	receiverDiff, err := ds.loadWavesBalance(receiver)
	if err != nil {
		return err
	}
	// changes receiver's leaseIn by amount
	if err := receiverDiff.addLeaseIn(amount); err != nil {
		return err
	}
	ds.wavesBalances[receiver] = receiverDiff
	return nil
}

// lease increases sender's leaseOut and receiver's leaseIn by leasing amount
func (ds *diffState) lease(sender, receiver proto.WavesAddress, amount int64, leaseID crypto.Digest) error {
	if err := ds.changeLeaseBalances(sender.ID(), receiver.ID(), amount); err != nil {
		return errors.Wrapf(err, "failed to change lease balances for lease '%s'", leaseID.String())
	}
	// add new lease
	l := lease{
		active:    true,
		recipient: receiver,
		sender:    sender,
		amount:    amount,
	}
	ds.leases[leaseID] = l
	return nil
}

// cancelLease decreases sender's leaseOut and receiver's leaseIn by cancelled leasing amount
func (ds *diffState) cancelLease(sender, receiver proto.WavesAddress, amount int64, leaseID crypto.Digest) error {
	if err := ds.changeLeaseBalances(sender.ID(), receiver.ID(), -amount); err != nil {
		return errors.Wrapf(err, "failed to change lease balances for lease '%s'", leaseID.String())
	}
	// deactivate lease
	l, err := ds.loadLease(leaseID)
	if err != nil {
		return errors.Wrapf(err, "failed to load lease by leaseID '%s' for cancel", leaseID.String())
	}
	l.active = false
	ds.leases[leaseID] = l
	return nil
}

func (ds *diffState) findIntFromDataEntryByKey(key string, address proto.WavesAddress) *proto.IntegerDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := ds.data[k]; ok {
		if te, ok := e.(*proto.IntegerDataEntry); ok {
			return te
		}
	}
	return nil
}

func (ds *diffState) findBoolFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BooleanDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := ds.data[k]; ok {
		if te, ok := e.(*proto.BooleanDataEntry); ok {
			return te
		}
	}
	return nil
}

func (ds *diffState) findStringFromDataEntryByKey(key string, address proto.WavesAddress) *proto.StringDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := ds.data[k]; ok {
		if te, ok := e.(*proto.StringDataEntry); ok {
			return te
		}
	}
	return nil
}

func (ds *diffState) findBinaryFromDataEntryByKey(key string, address proto.WavesAddress) *proto.BinaryDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := ds.data[k]; ok {
		if te, ok := e.(*proto.BinaryDataEntry); ok {
			return te
		}
	}
	return nil
}

func (ds *diffState) findDeleteFromDataEntryByKey(key string, address proto.WavesAddress) *proto.DeleteDataEntry {
	k := dataEntryKey{key, address}
	if e, ok := ds.data[k]; ok {
		if te, ok := e.(*proto.DeleteDataEntry); ok {
			return te
		}
	}
	return nil
}

func (ds *diffState) putDataEntry(entry proto.DataEntry, address proto.WavesAddress) {
	k := dataEntryKey{entry.GetKey(), address}
	ds.data[k] = entry
}

func (ds *diffState) findSponsorship(assetID crypto.Digest) *int64 {
	if sponsorship, ok := ds.sponsorships[assetID]; ok {
		return &sponsorship.minFee
	}
	return nil
}

func (ds *diffState) findNewAsset(assetID crypto.Digest) *diffNewAssetInfo {
	if newAsset, ok := ds.newAssetsInfo[assetID]; ok {
		return &newAsset
	}
	return nil
}

func (ds *diffState) findOldAsset(assetID crypto.Digest) *diffOldAssetInfo {
	if oldAsset, ok := ds.oldAssetsInfo[assetID]; ok {
		return &oldAsset
	}
	return nil
}

func (ds *diffState) wavesTransfer(sender, recipient proto.AddressID, amount int64) error {
	if _, err := ds.loadWavesBalance(sender); err != nil {
		return err
	}
	if err := ds.addWavesBalance(sender, -amount); err != nil {
		return err
	}
	if _, err := ds.loadWavesBalance(recipient); err != nil {
		return err
	}
	if err := ds.addWavesBalance(recipient, amount); err != nil {
		return err
	}
	return nil
}

func (ds *diffState) assetTransfer(sender, recipient proto.AddressID, asset crypto.Digest, amount int64) error {
	senderBalanceKey := assetBalanceKey{id: sender, asset: asset}
	if _, err := ds.loadAssetBalance(senderBalanceKey); err != nil {
		return err
	}
	if err := ds.addAssetBalance(senderBalanceKey, -amount); err != nil {
		return err
	}
	recipientBalanceKey := assetBalanceKey{id: recipient, asset: asset}
	if _, err := ds.loadAssetBalance(recipientBalanceKey); err != nil {
		return err
	}
	if err := ds.addAssetBalance(recipientBalanceKey, amount); err != nil {
		return err
	}
	return nil
}
