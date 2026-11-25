package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// recordingSmartState is a view over types.EnrichedSmartState that records the access key of every state
// entity read through it. The collected read set is the actual read set of a script evaluation.
//
// A key is recorded before the call is delegated to the wrapped state, so negative lookups and failed calls
// are recorded as well: a later write may turn "not found" into "found".
//
// Methods that read nothing but data that is constant within a block (heights, block infos, estimator version)
// record nothing.
//
// The view is intended to be used by a single goroutine and has no internal locking.
type recordingSmartState struct {
	inner types.EnrichedSmartState
	reads accessSet
}

var _ types.EnrichedSmartState = (*recordingSmartState)(nil)

func newRecordingSmartState(inner types.EnrichedSmartState) *recordingSmartState {
	return &recordingSmartState{inner: inner, reads: newAccessSet()}
}

// readSet returns the set of keys read so far. The set is owned by the view: it must not be modified by
// the caller and it keeps growing while the view is in use.
func (r *recordingSmartState) readSet() accessSet { return r.reads }

// recordRecipient records the key built by the given constructor for the account behind the recipient.
// For an alias recipient the alias key is recorded first, then the alias is resolved through the wrapped
// state and the key of the resolved address is recorded. If the alias can not be resolved only the alias key
// is recorded: the following delegated call fails on the same resolution, and any transaction that makes
// the alias resolvable writes the alias key.
func (r *recordingSmartState) recordRecipient(recipient proto.Recipient, key func(proto.AddressID) accessKey) {
	if addr := recipient.Address(); addr != nil {
		r.reads.add(key(addr.ID()))
		return
	}
	alias := recipient.Alias()
	if alias == nil {
		return // Invalid recipient, nothing is read; the delegated call reports the error.
	}
	r.reads.add(aliasAccessKey(alias.Alias))
	addr, err := r.inner.NewestAddrByAlias(*alias)
	if err != nil {
		return
	}
	r.reads.add(key(addr.ID()))
}

func (r *recordingSmartState) NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error) {
	r.reads.add(accountScriptAccessKey(addr.ID()))
	return r.inner.NewestScriptPKByAddr(addr)
}

func (r *recordingSmartState) AddingBlockHeight() (uint64, error) {
	return r.inner.AddingBlockHeight()
}

func (r *recordingSmartState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	r.reads.add(txAccessKey(id))
	return r.inner.NewestTransactionByID(id)
}

func (r *recordingSmartState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	r.reads.add(txAccessKey(id))
	return r.inner.NewestTransactionHeightByID(id)
}

func (r *recordingSmartState) NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error) {
	r.recordRecipient(account, accountScriptAccessKey)
	return r.inner.NewestScriptByAccount(account)
}

func (r *recordingSmartState) NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error) {
	r.recordRecipient(account, accountScriptAccessKey)
	return r.inner.NewestScriptBytesByAccount(account)
}

// NewestRecipientToAddress records the alias key for an alias recipient. An address recipient is returned
// by the wrapped state as is, without reading anything.
func (r *recordingSmartState) NewestRecipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error) {
	if alias := recipient.Alias(); alias != nil {
		r.reads.add(aliasAccessKey(alias.Alias))
	}
	return r.inner.NewestRecipientToAddress(recipient)
}

func (r *recordingSmartState) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	r.reads.add(aliasAccessKey(alias.Alias))
	return r.inner.NewestAddrByAlias(alias)
}

func (r *recordingSmartState) NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error) {
	r.reads.add(leaseAccessKey(id))
	return r.inner.NewestLeasingInfo(id)
}

func (r *recordingSmartState) IsStateUntouched(account proto.Recipient) (bool, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.IsStateUntouched(account)
}

func (r *recordingSmartState) NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
	r.recordRecipient(account, accountBalanceAccessKey)
	return r.inner.NewestAssetBalance(account, assetID)
}

func (r *recordingSmartState) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	r.recordRecipient(account, accountBalanceAccessKey)
	return r.inner.NewestWavesBalance(account)
}

func (r *recordingSmartState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	r.recordRecipient(account, accountBalanceAccessKey)
	return r.inner.NewestFullWavesBalance(account)
}

func (r *recordingSmartState) RetrieveNewestIntegerEntry(
	account proto.Recipient, key string,
) (*proto.IntegerDataEntry, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.RetrieveNewestIntegerEntry(account, key)
}

func (r *recordingSmartState) RetrieveNewestBooleanEntry(
	account proto.Recipient, key string,
) (*proto.BooleanDataEntry, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.RetrieveNewestBooleanEntry(account, key)
}

func (r *recordingSmartState) RetrieveNewestStringEntry(
	account proto.Recipient, key string,
) (*proto.StringDataEntry, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.RetrieveNewestStringEntry(account, key)
}

func (r *recordingSmartState) RetrieveNewestBinaryEntry(
	account proto.Recipient, key string,
) (*proto.BinaryDataEntry, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.RetrieveNewestBinaryEntry(account, key)
}

func (r *recordingSmartState) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	r.recordRecipient(account, accountDataAccessKey)
	return r.inner.RetrieveEntries(account)
}

func (r *recordingSmartState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	r.reads.add(assetAccessKey(assetID))
	return r.inner.NewestAssetIsSponsored(assetID)
}

func (r *recordingSmartState) NewestAssetConstInfo(assetID proto.AssetID) (*proto.AssetConstInfo, error) {
	r.reads.add(shortAssetAccessKey(assetID))
	return r.inner.NewestAssetConstInfo(assetID)
}

func (r *recordingSmartState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	r.reads.add(assetAccessKey(assetID))
	return r.inner.NewestAssetInfo(assetID)
}

// NewestFullAssetInfo records the asset key and, for a sponsored asset, the balance key of its issuer:
// the full info of a sponsored asset includes the Waves balance of the sponsor. The issuer is known only
// from the result, so its key is the only one recorded after the delegated call. Nothing is missed when
// the call fails: the issuer and the sponsorship flag are parts of the asset, which is already recorded.
func (r *recordingSmartState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	r.reads.add(assetAccessKey(assetID))
	info, err := r.inner.NewestFullAssetInfo(assetID)
	if info != nil && info.Sponsored {
		r.reads.add(accountBalanceAccessKey(info.Issuer.ID()))
	}
	return info, err
}

func (r *recordingSmartState) NewestScriptByAsset(assetID crypto.Digest) (*ast.Tree, error) {
	r.reads.add(assetAccessKey(assetID))
	return r.inner.NewestScriptByAsset(assetID)
}

func (r *recordingSmartState) NewestBlockInfoByHeight(height proto.Height) (*proto.BlockInfo, error) {
	return r.inner.NewestBlockInfoByHeight(height)
}

func (r *recordingSmartState) EstimatorVersion() (int, error) {
	return r.inner.EstimatorVersion()
}

func (r *recordingSmartState) IsNotFound(err error) bool {
	return r.inner.IsNotFound(err)
}

func (r *recordingSmartState) WavesBalanceProfile(id proto.AddressID) (*types.WavesBalanceProfile, error) {
	r.reads.add(accountBalanceAccessKey(id))
	return r.inner.WavesBalanceProfile(id)
}

func (r *recordingSmartState) NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error) {
	r.reads.add(accountBalanceAccessKey(id))
	return r.inner.NewestAssetBalanceByAddressID(id, asset)
}
