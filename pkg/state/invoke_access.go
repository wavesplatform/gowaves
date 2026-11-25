package state

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// accessKind is the kind of state entity touched by a transaction.
// Kinds are deliberately finer than a whole account to avoid needless conflicts between transactions.
type accessKind byte

const (
	// accessKindAccountData covers all data entries of an account.
	accessKindAccountData accessKind = iota + 1
	// accessKindAccountBalance covers any balance of an account: Waves, assets, leasing, generating, deposit.
	accessKindAccountBalance
	// accessKindAccountScript covers the script of an account and its complexities.
	accessKindAccountScript
	// accessKindAsset covers asset info, volume, description, script and sponsorship.
	accessKindAsset
	// accessKindAlias covers the alias to address mapping.
	accessKindAlias
	// accessKindLease covers leasing info by lease ID.
	accessKindLease
	// accessKindTx covers the existence, height and status of a transaction by ID.
	accessKindTx
)

func (k accessKind) String() string {
	switch k {
	case accessKindAccountData:
		return "AccountData"
	case accessKindAccountBalance:
		return "AccountBalance"
	case accessKindAccountScript:
		return "AccountScript"
	case accessKindAsset:
		return "Asset"
	case accessKindAlias:
		return "Alias"
	case accessKindLease:
		return "Lease"
	case accessKindTx:
		return "Tx"
	default:
		return fmt.Sprintf("Unknown(%d)", byte(k))
	}
}

// accessKeyPayloadSize is the size of the fixed payload of accessKey. It fits an address ID (20 bytes),
// a digest (32 bytes) and a valid alias (up to 30 bytes) without transformation.
const accessKeyPayloadSize = crypto.DigestSize

// accessKey identifies a state entity read or written by a transaction.
// The type is comparable and is intended to be used as a map key.
//
// Payloads shorter than accessKeyPayloadSize are zero-padded, longer payloads are replaced by their hash.
// Both transformations may only make two different entities share a key, which results in a spurious conflict
// and never in a missed one.
type accessKey struct {
	kind    accessKind
	payload [accessKeyPayloadSize]byte
}

func newAccessKey(kind accessKind, payload []byte) accessKey {
	k := accessKey{kind: kind}
	if len(payload) <= accessKeyPayloadSize {
		copy(k.payload[:], payload)
		return k
	}
	// The error is ignored because FastHash can fail only on a hasher write, which never happens.
	d, _ := crypto.FastHash(payload)
	k.payload = d
	return k
}

func accountDataAccessKey(id proto.AddressID) accessKey {
	return newAccessKey(accessKindAccountData, id.Bytes())
}

func accountBalanceAccessKey(id proto.AddressID) accessKey {
	return newAccessKey(accessKindAccountBalance, id.Bytes())
}

func accountScriptAccessKey(id proto.AddressID) accessKey {
	return newAccessKey(accessKindAccountScript, id.Bytes())
}

// assetAccessKey creates the key of an asset by its full ID. Assets are keyed by the short proto.AssetID,
// the same way they are keyed in the storage, because some readers know only the short ID.
func assetAccessKey(assetID crypto.Digest) accessKey {
	return shortAssetAccessKey(proto.AssetIDFromDigest(assetID))
}

func shortAssetAccessKey(assetID proto.AssetID) accessKey {
	return newAccessKey(accessKindAsset, assetID.Bytes())
}

func aliasAccessKey(alias string) accessKey {
	return newAccessKey(accessKindAlias, []byte(alias))
}

func leaseAccessKey(leaseID crypto.Digest) accessKey {
	return accessKey{kind: accessKindLease, payload: leaseID}
}

func txAccessKey(txID []byte) accessKey {
	return newAccessKey(accessKindTx, txID)
}

func (k accessKey) compare(other accessKey) int {
	if k.kind != other.kind {
		if k.kind < other.kind {
			return -1
		}
		return 1
	}
	return bytes.Compare(k.payload[:], other.payload[:])
}

func (k accessKey) String() string {
	return k.kind.String() + "(" + hex.EncodeToString(k.payload[:]) + ")"
}

// accessSet is a set of access keys. The zero value is ready for reading, use newAccessSet to create a set
// for writing.
type accessSet map[accessKey]struct{}

func newAccessSet(keys ...accessKey) accessSet {
	s := make(accessSet, len(keys))
	for _, k := range keys {
		s.add(k)
	}
	return s
}

func (s accessSet) add(k accessKey) { s[k] = struct{}{} }

func (s accessSet) len() int { return len(s) }

// intersects reports whether the sets have at least one common key.
func (s accessSet) intersects(other accessSet) bool {
	small, big := s, other
	if len(big) < len(small) {
		small, big = big, small
	}
	for k := range small {
		if _, ok := big[k]; ok {
			return true
		}
	}
	return false
}

// union adds all keys of the other set to the receiver.
func (s accessSet) union(other accessSet) {
	for k := range other {
		s[k] = struct{}{}
	}
}

// reset removes all keys from the set keeping the allocated memory.
func (s accessSet) reset() { clear(s) }

// sorted returns keys of the set in the deterministic order: by kind, then by payload.
func (s accessSet) sorted() []accessKey {
	keys := make([]accessKey, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, accessKey.compare)
	return keys
}

// writeSetFromSnapshot returns the set of access keys written by the transaction with the given ID according to
// its snapshot. Both regular and internal atomic snapshots are taken into account.
// The transaction ID is required only for snapshots that contain transaction-bound atomic snapshots
// (transaction status, script result).
func writeSetFromSnapshot(txID []byte, s txSnapshot, scheme proto.Scheme) (accessSet, error) {
	c := writeSetCollector{
		txID:   txID,
		scheme: scheme,
		set:    make(accessSet, len(s.regular)+len(s.internal)),
	}
	for _, rs := range s.regular {
		if err := rs.Apply(&c); err != nil {
			return nil, errors.Wrap(err, "failed to collect write set from regular snapshot")
		}
	}
	for _, is := range s.internal {
		if err := is.ApplyInternal(&c); err != nil {
			return nil, errors.Wrap(err, "failed to collect write set from internal snapshot")
		}
	}
	return c.set, nil
}

// writeSetCollector maps atomic snapshots to access keys. It implements the same visitor interfaces as the real
// snapshot applier, so a new atomic snapshot type does not compile until it is mapped here.
type writeSetCollector struct {
	txID   []byte
	scheme proto.Scheme
	set    accessSet
}

var (
	_ proto.SnapshotApplier   = (*writeSetCollector)(nil)
	_ internalSnapshotApplier = (*writeSetCollector)(nil)
)

func (c *writeSetCollector) addAccountKey(
	pk crypto.PublicKey, key func(proto.AddressID) accessKey,
) error {
	addr, err := proto.NewAddressFromPublicKey(c.scheme, pk)
	if err != nil {
		return errors.Wrapf(err, "failed to create address from scheme %d and PK %q", c.scheme, pk.String())
	}
	c.set.add(key(addr.ID()))
	return nil
}

func (c *writeSetCollector) addTxKey() error {
	if len(c.txID) == 0 {
		return errors.New("transaction ID is not set")
	}
	c.set.add(txAccessKey(c.txID))
	return nil
}

func (c *writeSetCollector) ApplyWavesBalance(snapshot proto.WavesBalanceSnapshot) error {
	c.set.add(accountBalanceAccessKey(snapshot.Address.ID()))
	return nil
}

func (c *writeSetCollector) ApplyLeaseBalance(snapshot proto.LeaseBalanceSnapshot) error {
	c.set.add(accountBalanceAccessKey(snapshot.Address.ID()))
	return nil
}

func (c *writeSetCollector) ApplyAssetBalance(snapshot proto.AssetBalanceSnapshot) error {
	c.set.add(accountBalanceAccessKey(snapshot.Address.ID()))
	return nil
}

func (c *writeSetCollector) ApplyAlias(snapshot proto.AliasSnapshot) error {
	c.set.add(aliasAccessKey(snapshot.Alias))
	return nil
}

func (c *writeSetCollector) ApplyNewAsset(snapshot proto.NewAssetSnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplyAssetDescription(snapshot proto.AssetDescriptionSnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplyAssetVolume(snapshot proto.AssetVolumeSnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplyAssetScript(snapshot proto.AssetScriptSnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplySponsorship(snapshot proto.SponsorshipSnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplyAccountScript(snapshot proto.AccountScriptSnapshot) error {
	return c.addAccountKey(snapshot.SenderPublicKey, accountScriptAccessKey)
}

// ApplyFilledVolumeAndFee adds nothing: order fills are not observable by scripts of other transactions
// (RIDE has no access to them), and exchange transactions are never speculated.
func (c *writeSetCollector) ApplyFilledVolumeAndFee(proto.FilledVolumeFeeSnapshot) error {
	return nil
}

func (c *writeSetCollector) ApplyDataEntries(snapshot proto.DataEntriesSnapshot) error {
	c.set.add(accountDataAccessKey(snapshot.Address.ID()))
	return nil
}

// ApplyNewLease adds only the lease key: changes of leasing balances of the sender and the recipient come
// in separate lease balance snapshots.
func (c *writeSetCollector) ApplyNewLease(snapshot proto.NewLeaseSnapshot) error {
	c.set.add(leaseAccessKey(snapshot.LeaseID))
	return nil
}

func (c *writeSetCollector) ApplyCancelledLease(snapshot proto.CancelledLeaseSnapshot) error {
	c.set.add(leaseAccessKey(snapshot.LeaseID))
	return nil
}

func (c *writeSetCollector) ApplyTransactionsStatus(proto.TransactionStatusSnapshot) error {
	return c.addTxKey()
}

// ApplyCommitToGeneration adds the balance key of the committing account, because the commitment changes
// the deposit part of its balance profile.
func (c *writeSetCollector) ApplyCommitToGeneration(snapshot proto.GenerationCommitmentSnapshot) error {
	return c.addAccountKey(snapshot.SenderPublicKey, accountBalanceAccessKey)
}

// ApplyDAppComplexity adds the account script key: complexities are stored along with the script and are read
// together with it on invocation.
func (c *writeSetCollector) ApplyDAppComplexity(snapshot InternalDAppComplexitySnapshot) error {
	c.set.add(accountScriptAccessKey(snapshot.ScriptAddress.ID()))
	return nil
}

func (c *writeSetCollector) ApplyDAppUpdateComplexity(snapshot InternalDAppUpdateComplexitySnapshot) error {
	c.set.add(accountScriptAccessKey(snapshot.ScriptAddress.ID()))
	return nil
}

func (c *writeSetCollector) ApplyAssetScriptComplexity(snapshot InternalAssetScriptComplexitySnapshot) error {
	c.set.add(assetAccessKey(snapshot.AssetID))
	return nil
}

func (c *writeSetCollector) ApplyNewLeaseInfo(snapshot InternalNewLeaseInfoSnapshot) error {
	c.set.add(leaseAccessKey(snapshot.LeaseID))
	return nil
}

func (c *writeSetCollector) ApplyCancelledLeaseInfo(snapshot InternalCancelledLeaseInfoSnapshot) error {
	c.set.add(leaseAccessKey(snapshot.LeaseID))
	return nil
}

// ApplyScriptResult adds the transaction key: the script result is stored by the ID of the transaction.
func (c *writeSetCollector) ApplyScriptResult(InternalScriptResultSnapshot) error {
	return c.addTxKey()
}
