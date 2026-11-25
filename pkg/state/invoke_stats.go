package state

import (
	"cmp"
	"container/list"
	"encoding/binary"
	"encoding/hex"
	"log/slog"
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// Invoke access statistics.
//
// After every execution of an invoke the sets of state entities it actually read and modified are stored per
// (dApp address, function name). Later the statistics are used to predict what the next invocation of the same
// function is going to touch. Statistics are advisory: they only decide what is worth evaluating speculatively
// and never affect the result of the block application. That is why they are not consensus data: they are
// written directly to the key-value database, are not part of state hashes, are not versioned by blocks and are
// not rolled back.

// statRole tells that the address of a statistics key is not absolute, but is taken from the invoke transaction.
type statRole byte

const (
	// statRoleNone marks an absolute key.
	statRoleNone statRole = iota
	// statRoleSender marks a key of the account that sent the invoke transaction.
	statRoleSender
	// statRoleDApp marks a key of the invoked dApp.
	statRoleDApp
)

// statKey is an access key stored in statistics. If the role is not statRoleNone, the address of the key is
// replaced by the role and the payload of the key is zero.
type statKey struct {
	role statRole
	key  accessKey
}

func (k statKey) compare(other statKey) int {
	if c := cmp.Compare(k.role, other.role); c != 0 {
		return c
	}
	return k.key.compare(other.key)
}

// isAccountKind reports whether the payload of keys of the kind is an address ID, so it can be replaced by a role.
func (k accessKind) isAccountKind() bool {
	return k == accessKindAccountData || k == accessKindAccountBalance || k == accessKindAccountScript
}

// storedPayloadSize returns the number of meaningful leading bytes of the payload of keys of the kind,
// the rest of the payload is always zero. The second result is false for an unknown kind.
func (k accessKind) storedPayloadSize() (int, bool) {
	switch k {
	case accessKindAccountData, accessKindAccountBalance, accessKindAccountScript:
		return proto.AddressIDSize, true
	case accessKindAsset:
		return proto.AssetIDSize, true
	case accessKindAlias, accessKindLease, accessKindTx:
		return accessKeyPayloadSize, true
	default:
		return 0, false
	}
}

// normalize converts the actual access set of an invoke to role-normalised statistics keys: keys of the sender
// of the transaction and of the dApp are replaced by roles, other keys stay absolute.
// If the sender is the dApp itself, both roles are produced for its keys, because it is unknown which of them
// is the reason of the access. Instantiation with the same sender and dApp gives the original set anyway.
// Transaction keys are dropped: a transaction is written only once, by itself, so a key of this kind can
// conflict only inside a single block and has no predictive value.
// The result is sorted.
func normalize(actual accessSet, sender, dApp proto.AddressID) []statKey {
	res := make([]statKey, 0, len(actual))
	for k := range actual {
		if k.kind == accessKindTx {
			continue
		}
		if !k.kind.isAccountKind() {
			res = append(res, statKey{role: statRoleNone, key: k})
			continue
		}
		isSender := k == newAccessKey(k.kind, sender.Bytes())
		isDApp := k == newAccessKey(k.kind, dApp.Bytes())
		if isSender {
			res = append(res, statKey{role: statRoleSender, key: accessKey{kind: k.kind}})
		}
		if isDApp {
			res = append(res, statKey{role: statRoleDApp, key: accessKey{kind: k.kind}})
		}
		if !isSender && !isDApp {
			res = append(res, statKey{role: statRoleNone, key: k})
		}
	}
	slices.SortFunc(res, statKey.compare)
	return res
}

func (k statKey) instantiate(sender, dApp proto.AddressID) accessKey {
	switch k.role {
	case statRoleSender:
		return newAccessKey(k.key.kind, sender.Bytes())
	case statRoleDApp:
		return newAccessKey(k.key.kind, dApp.Bytes())
	case statRoleNone:
		return k.key
	default:
		return k.key
	}
}

// statKeySet is a set of statistics keys with the number of the last execution each key was seen in.
type statKeySet map[statKey]uint64

func (s statKeySet) instantiate(sender, dApp proto.AddressID) accessSet {
	res := make(accessSet, len(s))
	for k := range s {
		res.add(k.instantiate(sender, dApp))
	}
	return res
}

// merge adds keys seen in the execution with the given number. If the set grows above the soft cap, the least
// recently seen keys are evicted. Keys of the current execution are never evicted.
// It reports whether the set of keys has changed.
func (s statKeySet) merge(keys []statKey, execution uint64) bool {
	changed := false
	for _, k := range keys {
		if _, ok := s[k]; !ok {
			changed = true
		}
		s[k] = execution
	}
	excess := len(s) - invokeStatsKeysSoftCap
	if excess <= 0 {
		return changed
	}
	type candidate struct {
		key      statKey
		lastSeen uint64
	}
	candidates := make([]candidate, 0, len(s))
	for k, seen := range s {
		if seen != execution {
			candidates = append(candidates, candidate{key: k, lastSeen: seen})
		}
	}
	// Ties are broken by the key to keep the eviction deterministic.
	slices.SortFunc(candidates, func(a, b candidate) int {
		if c := cmp.Compare(a.lastSeen, b.lastSeen); c != 0 {
			return c
		}
		return a.key.compare(b.key)
	})
	evicted := candidates[:min(excess, len(candidates))]
	for _, c := range evicted {
		delete(s, c.key)
	}
	return changed || len(evicted) != 0
}

func (s statKeySet) sorted() []statKey {
	keys := make([]statKey, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, statKey.compare)
	return keys
}

// invokeStatsKey identifies a callable function of a dApp.
type invokeStatsKey struct {
	dApp     proto.AddressID
	function string
}

func (k invokeStatsKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressIDSize+len(k.function))
	buf[0] = invokeStatsKeyPrefix
	copy(buf[1:], k.dApp[:])
	copy(buf[1+proto.AddressIDSize:], k.function)
	return buf
}

// invokeStats is the access statistics of a callable function.
type invokeStats struct {
	// reads and writes are the role-normalised keys the function was seen to read and to modify.
	reads, writes statKeySet
	// wide is set once a single execution touches more keys than the hard cap. Keys of a wide entry are not
	// tracked and such invokes must not be speculated together with predecessors. The flag is never cleared.
	wide bool
	// executions is the number of recorded executions, it is also the number of the last execution.
	executions uint64
	// mispredictions is the number of executions for which a speculative result had to be discarded.
	mispredictions uint64
}

func newInvokeStats() *invokeStats {
	return &invokeStats{reads: make(statKeySet), writes: make(statKeySet)}
}

// instantiate converts statistics to the concrete predicted read and write sets of the invoke with the given
// sender and dApp. Nil statistics produce empty sets.
func instantiate(stats *invokeStats, sender, dApp proto.AddressID) (accessSet, accessSet) {
	if stats == nil {
		return newAccessSet(), newAccessSet()
	}
	return stats.reads.instantiate(sender, dApp), stats.writes.instantiate(sender, dApp)
}

// update records one more execution with the given role-normalised actual read and write keys.
// It reports whether the predictions made from the statistics have changed: the keys or the wide flag.
func (s *invokeStats) update(actualReads, actualWrites []statKey, mispredicted bool) bool {
	s.executions++
	if mispredicted {
		s.mispredictions++
	}
	if s.wide {
		return false
	}
	if len(actualReads) > invokeStatsKeysHardCap || len(actualWrites) > invokeStatsKeysHardCap {
		s.wide = true
		clear(s.reads)
		clear(s.writes)
		return true
	}
	readsChanged := s.reads.merge(actualReads, s.executions)
	writesChanged := s.writes.merge(actualWrites, s.executions)
	return readsChanged || writesChanged
}

const (
	invokeStatsVersion byte = 1

	invokeStatsWideFlag byte = 1

	// A statistics key is stored as a byte of the kind and the role followed by the payload of absolute keys.
	statKeyKindMask byte = 0x3f
)

const statKeyRoleShift = 6

// marshalBinary serializes statistics in the following format:
//
//	version (1 byte) | flags (1 byte) | executions (uvarint) | mispredictions (uvarint) | reads | writes
//
// where reads and writes are
//
//	number of keys (uvarint) | keys
//
// and every key is
//
//	role << 6 | kind (1 byte) | payload (only for absolute keys, size depends on the kind) | last seen (uvarint)
//
// Keys are sorted, so the encoding is deterministic.
func (s *invokeStats) marshalBinary() ([]byte, error) {
	const headerSize = 2 + 4*binary.MaxVarintLen64
	const keySizeEstimate = 1 + accessKeyPayloadSize + binary.MaxVarintLen32
	buf := make([]byte, 0, headerSize+(len(s.reads)+len(s.writes))*keySizeEstimate)
	var flags byte
	if s.wide {
		flags |= invokeStatsWideFlag
	}
	buf = append(buf, invokeStatsVersion, flags)
	buf = binary.AppendUvarint(buf, s.executions)
	buf = binary.AppendUvarint(buf, s.mispredictions)
	for _, set := range []statKeySet{s.reads, s.writes} {
		buf = binary.AppendUvarint(buf, uint64(len(set)))
		for _, k := range set.sorted() {
			size, ok := k.key.kind.storedPayloadSize()
			if !ok {
				return nil, errors.Errorf("unknown access kind %d", byte(k.key.kind))
			}
			buf = append(buf, byte(k.role)<<statKeyRoleShift|byte(k.key.kind))
			if k.role == statRoleNone {
				buf = append(buf, k.key.payload[:size]...)
			}
			buf = binary.AppendUvarint(buf, set[k])
		}
	}
	return buf, nil
}

func (s *invokeStats) unmarshalBinary(data []byte) error {
	const minSize = 2
	if len(data) < minSize {
		return errInvalidDataSize
	}
	if v := data[0]; v != invokeStatsVersion {
		return errors.Errorf("unsupported version %d of invoke statistics", v)
	}
	if data[1]&^invokeStatsWideFlag != 0 {
		return errors.Errorf("unknown flags %#x of invoke statistics", data[1])
	}
	res := invokeStats{wide: data[1]&invokeStatsWideFlag != 0}
	data = data[minSize:]
	var err error
	if res.executions, data, err = readUvarint(data); err != nil {
		return err
	}
	if res.mispredictions, data, err = readUvarint(data); err != nil {
		return err
	}
	if res.reads, data, err = unmarshalStatKeySet(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal read keys")
	}
	if res.writes, data, err = unmarshalStatKeySet(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal write keys")
	}
	if len(data) != 0 {
		return errors.Errorf("%d trailing bytes after invoke statistics", len(data))
	}
	*s = res
	return nil
}

func readUvarint(data []byte) (uint64, []byte, error) {
	v, n := binary.Uvarint(data)
	if n <= 0 {
		return 0, nil, errInvalidDataSize
	}
	return v, data[n:], nil
}

func unmarshalStatKeySet(data []byte) (statKeySet, []byte, error) {
	count, data, err := readUvarint(data)
	if err != nil {
		return nil, nil, err
	}
	if count > invokeStatsKeysHardCap {
		return nil, nil, errors.Errorf("too many keys %d", count)
	}
	set := make(statKeySet, count)
	for range count {
		if len(data) == 0 {
			return nil, nil, errInvalidDataSize
		}
		k := statKey{
			role: statRole(data[0] >> statKeyRoleShift),
			key:  accessKey{kind: accessKind(data[0] & statKeyKindMask)},
		}
		data = data[1:]
		size, ok := k.key.kind.storedPayloadSize()
		if !ok {
			return nil, nil, errors.Errorf("unknown access kind %d", byte(k.key.kind))
		}
		switch k.role {
		case statRoleNone:
			if len(data) < size {
				return nil, nil, errInvalidDataSize
			}
			copy(k.key.payload[:], data[:size])
			data = data[size:]
		case statRoleSender, statRoleDApp:
			if !k.key.kind.isAccountKind() {
				return nil, nil, errors.Errorf("role %d is not applicable to access kind %s", k.role, k.key.kind)
			}
		default:
			return nil, nil, errors.Errorf("unknown role %d", k.role)
		}
		var seen uint64
		if seen, data, err = readUvarint(data); err != nil {
			return nil, nil, err
		}
		if _, dup := set[k]; dup {
			return nil, nil, errors.New("duplicate key")
		}
		set[k] = seen
	}
	return set, data, nil
}

type invokeStatsCacheEntry struct {
	key invokeStatsKey
	// stats is nil if it is known that there are no statistics for the key.
	stats *invokeStats
	dirty bool
}

// invokeStatsStorage is the storage of invoke access statistics: a bounded in-memory cache over the records of
// the key-value database. Entries are loaded from the database lazily, updated in memory and written by flush.
//
// The storage is not safe for concurrent use, it must be used from the block appender goroutine only.
type invokeStatsStorage struct {
	db      keyvalue.KeyValue
	dbBatch keyvalue.Batch

	maxEntries int
	entries    map[invokeStatsKey]*list.Element // Elements hold *invokeStatsCacheEntry.
	order      *list.List                       // The most recently used entry is at the front.
	dirty      int
	// revision is changed every time the predictions made from the statistics of any function may have
	// changed, so the predictions made at the same revision can be reused.
	revision uint64
}

func newInvokeStatsStorage(db keyvalue.KeyValue, dbBatch keyvalue.Batch, maxEntries int) *invokeStatsStorage {
	return &invokeStatsStorage{
		db:         db,
		dbBatch:    dbBatch,
		maxEntries: maxEntries,
		entries:    make(map[invokeStatsKey]*list.Element),
		order:      list.New(),
	}
}

func (s *invokeStatsStorage) entry(key invokeStatsKey) (*invokeStatsCacheEntry, error) {
	if el, ok := s.entries[key]; ok {
		s.order.MoveToFront(el)
		return el.Value.(*invokeStatsCacheEntry), nil //nolint:errcheck // The list holds only entries.
	}
	e := &invokeStatsCacheEntry{key: key}
	data, err := s.db.Get(key.bytes())
	switch {
	case errors.Is(err, keyvalue.ErrNotFound):
		// No statistics yet.
	case err != nil:
		return nil, errors.Wrap(err, "failed to load invoke statistics")
	default:
		stats := new(invokeStats)
		if umErr := stats.unmarshalBinary(data); umErr != nil {
			// Statistics are advisory, so an unreadable record is the same as no record. It is going to be
			// overwritten after the next execution of the function.
			slog.Debug("Ignoring unreadable invoke statistics record", slog.String("dApp", hex.EncodeToString(key.dApp[:])),
				slog.String("function", key.function), logging.Error(umErr))
		} else {
			e.stats = stats
		}
	}
	s.entries[key] = s.order.PushFront(e)
	s.trim()
	return e, nil
}

// trim evicts the least recently used entries above the limit. Dirty entries are kept until flush, so the cache
// can temporarily exceed the limit. The most recently used entry is never evicted, because it is the one
// the caller is working with.
func (s *invokeStatsStorage) trim() {
	for el := s.order.Back(); el != nil && el != s.order.Front() && len(s.entries) > s.maxEntries; {
		prev := el.Prev()
		if e := el.Value.(*invokeStatsCacheEntry); !e.dirty { //nolint:errcheck // The list holds only entries.
			s.order.Remove(el)
			delete(s.entries, e.key)
		}
		el = prev
	}
}

// stats returns statistics of the function, the second result is false if there are no statistics yet.
// The returned statistics are owned by the storage and must not be modified by the caller.
func (s *invokeStatsStorage) stats(key invokeStatsKey) (*invokeStats, bool, error) {
	e, err := s.entry(key)
	if err != nil {
		return nil, false, err
	}
	return e.stats, e.stats != nil, nil
}

// update records an execution of the function by the given sender with the actual read and write sets.
func (s *invokeStatsStorage) update(
	key invokeStatsKey, sender proto.AddressID, actualReads, actualWrites accessSet, mispredicted bool,
) error {
	e, err := s.entry(key)
	if err != nil {
		return err
	}
	created := e.stats == nil
	if created {
		e.stats = newInvokeStats()
	}
	changed := e.stats.update(
		normalize(actualReads, sender, key.dApp), normalize(actualWrites, sender, key.dApp), mispredicted,
	)
	if created || changed {
		s.revision++
	}
	if !e.dirty {
		e.dirty = true
		s.dirty++
	}
	return nil
}

// flush adds updated entries to the database batch. The batch is written to the database by the caller.
// If the write of the batch fails afterwards, the cache stays ahead of the database, which is acceptable
// for advisory data.
func (s *invokeStatsStorage) flush() error {
	if s.dirty == 0 {
		return nil
	}
	for el := s.order.Front(); el != nil; el = el.Next() {
		e := el.Value.(*invokeStatsCacheEntry) //nolint:errcheck // The list holds only entries.
		if !e.dirty {
			continue
		}
		data, err := e.stats.marshalBinary()
		if err != nil {
			return errors.Wrap(err, "failed to marshal invoke statistics")
		}
		s.dbBatch.Put(e.key.bytes(), data)
		e.dirty = false
		s.dirty--
	}
	s.trim()
	return nil
}

// reset drops entries updated since the last flush, so the cache holds only what was passed to the database.
// It is called when the application of blocks fails and the database batch is discarded.
func (s *invokeStatsStorage) reset() {
	if s.dirty == 0 {
		return
	}
	for el := s.order.Front(); el != nil; {
		next := el.Next()
		if e := el.Value.(*invokeStatsCacheEntry); e.dirty { //nolint:errcheck // The list holds only entries.
			s.order.Remove(el)
			delete(s.entries, e.key)
		}
		el = next
	}
	s.dirty = 0
	s.revision++
}
