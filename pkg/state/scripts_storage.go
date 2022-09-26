package state

import (
	"bytes"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
)

const (
	maxCacheSize = 100000
	// Can't evaluate real script size, so we use 1 per each.
	scriptSize    = 1
	maxCacheBytes = maxCacheSize * scriptSize
)

var errEmptyScript = errors.New("empty script")

func scriptBytesToTree(script proto.Script) (*ast.Tree, error) {
	tree, err := serialization.Parse(script)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

type accountScripRecordForHashes struct {
	addr   *proto.WavesAddress
	script proto.Script
}

func (ac *accountScripRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(ac.addr[:]); err != nil {
		return err
	}
	if len(ac.script) != 0 {
		if _, err := w.Write(ac.script[:]); err != nil {
			return err
		}
	}
	return nil
}

func (ac *accountScripRecordForHashes) less(other stateComponent) bool {
	ac2 := other.(*accountScripRecordForHashes)
	return bytes.Compare(ac.addr[:], ac2.addr[:]) == -1
}

type assetScripRecordForHashes struct {
	asset  crypto.Digest
	script proto.Script
}

func (as *assetScripRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(as.asset[:]); err != nil {
		return err
	}
	if len(as.script) != 0 {
		if _, err := w.Write(as.script[:]); err != nil {
			return err
		}
	}
	return nil
}

func (as *assetScripRecordForHashes) less(other stateComponent) bool {
	as2 := other.(*assetScripRecordForHashes)
	return bytes.Compare(as.asset[:], as2.asset[:]) == -1
}

type scriptBasicInfoRecord struct {
	PK             crypto.PublicKey   `cbor:"0,keyasint,omitemtpy"`
	ScriptLen      uint32             `cbor:"1,keyasint,omitemtpy"`
	LibraryVersion ast.LibraryVersion `cbor:"2,keyasint,omitemtpy"`
	HasVerifier    bool               `cbor:"3,keyasint,omitemtpy"`
	IsDApp         bool               `cbor:"4,keyasint,omitemtpy"`
}

func newScriptBasicInfoRecord(pk crypto.PublicKey, script proto.Script) (scriptBasicInfoRecord, *ast.Tree, error) {
	scriptLen := uint32(len(script))
	if scriptLen == 0 {
		return scriptBasicInfoRecord{PK: pk, ScriptLen: scriptLen}, nil, nil
	}
	tree, err := scriptBytesToTree(script)
	if err != nil {
		return scriptBasicInfoRecord{}, nil, errors.Wrapf(err, "failed to parse script bytes to tree for pk %q", pk.String())
	}
	info := scriptBasicInfoRecord{
		PK:             pk,
		ScriptLen:      scriptLen,
		LibraryVersion: tree.LibVersion,
		HasVerifier:    tree.HasVerifier(),
		IsDApp:         tree.IsDApp(),
	}
	return info, tree, nil
}

func (r *scriptBasicInfoRecord) scriptExists() bool {
	return r.ScriptLen != 0
}

func (r *scriptBasicInfoRecord) marshalBinary() ([]byte, error) {
	return cbor.Marshal(r)
}

func (r *scriptBasicInfoRecord) unmarshalBinary(data []byte) error {
	return cbor.Unmarshal(data, r)
}

type scriptDBItem struct {
	script proto.Script
	tree   *ast.Tree
	info   scriptBasicInfoRecord
}

func newScriptDBItem(pk crypto.PublicKey, script proto.Script) (scriptDBItem, error) {
	info, tree, err := newScriptBasicInfoRecord(pk, script)
	if err != nil {
		return scriptDBItem{}, errors.Wrap(err, "failed to create new script basic info record")
	}
	dbItem := scriptDBItem{
		script: script,
		tree:   tree,
		info:   info,
	}
	return dbItem, nil
}

type assetScriptRecordWithAssetIDTail struct {
	scriptDBItem scriptDBItem
	assetIDTail  [proto.AssetIDTailSize]byte // this field doesn't have to be stored to db, because it is used only for state hash calculation
}

// TODO: LRU cache for script ASTs here only makes sense at the import stage.
// It shouldn't be used at all when the node does rollbacks or validates UTX,
// because it has to be cleared after each rollback or UTX validation,
// which makes it inefficient.
type scriptsStorage struct {
	hs    *historyStorage
	cache *lru

	accountScriptsHasher *stateHasher
	assetScriptsHasher   *stateHasher
	calculateHashes      bool
	scheme               proto.Scheme

	uncertainAssetScripts map[proto.AssetID]assetScriptRecordWithAssetIDTail
}

func newScriptsStorage(hs *historyStorage, scheme proto.Scheme, calcHashes bool) (*scriptsStorage, error) {
	cache, err := newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return nil, err
	}
	return &scriptsStorage{
		hs:                    hs,
		cache:                 cache,
		accountScriptsHasher:  newStateHasher(),
		assetScriptsHasher:    newStateHasher(),
		calculateHashes:       calcHashes,
		scheme:                scheme,
		uncertainAssetScripts: make(map[proto.AssetID]assetScriptRecordWithAssetIDTail),
	}, nil
}

func (ss *scriptsStorage) setScript(scriptType blockchainEntity, key scriptKey, dbItem scriptDBItem, blockID proto.BlockID) error {
	scriptBasicInfoRecordBytes, err := dbItem.info.marshalBinary()
	if err != nil {
		return err
	}
	scriptKeyBytes := key.bytes()
	if err := ss.hs.addNewEntry(scriptType, scriptKeyBytes, dbItem.script, blockID); err != nil {
		return err
	}
	scriptBasicInfoKeyBytes := (&scriptBasicInfoKey{scriptKey: key}).bytes()
	if err := ss.hs.addNewEntry(scriptBasicInfo, scriptBasicInfoKeyBytes, scriptBasicInfoRecordBytes, blockID); err != nil {
		return err
	}
	if dbItem.script.IsEmpty() {
		// There is no AST for empty script.
		ss.cache.deleteIfExists(scriptKeyBytes)
		return nil
	}
	ss.cache.set(scriptKeyBytes, *dbItem.tree, scriptSize)
	return nil
}

func (ss *scriptsStorage) scriptBytesByKey(key []byte) (proto.Script, error) {
	script, err := ss.hs.topEntryData(key)
	if err != nil {
		return proto.Script{}, err
	}
	return script, nil
}

func (ss *scriptsStorage) newestScriptBytesByKey(key []byte) (proto.Script, error) {
	script, err := ss.hs.newestTopEntryData(key)
	if err != nil {
		return proto.Script{}, err
	}
	return script, nil
}

func (ss *scriptsStorage) scriptAstFromRecordBytes(script proto.Script) (*ast.Tree, error) {
	if script.IsEmpty() {
		// Empty script = no script.
		return nil, proto.ErrNotFound
	}
	return scriptBytesToTree(script)
}

func (ss *scriptsStorage) newestScriptAstByKey(key []byte) (*ast.Tree, error) {
	script, err := ss.hs.newestTopEntryData(key)
	if err != nil {
		return nil, err
	}
	return ss.scriptAstFromRecordBytes(script) // Possible errors `proto.ErrNotFound` and parsing errors.
}

func (ss *scriptsStorage) scriptTreeByKey(key []byte) (*ast.Tree, error) {
	script, err := ss.hs.topEntryData(key)
	if err != nil {
		return nil, err // Possible errors are `keyvalue.ErrNotFoundHere` and untyped "empty history"
	}
	return ss.scriptAstFromRecordBytes(script) // Possible errors `proto.ErrNotFound` and parsing errors.
}

func (ss *scriptsStorage) commitUncertain(blockID proto.BlockID) error {
	for assetID, r := range ss.uncertainAssetScripts {
		digest := proto.ReconstructDigest(assetID, r.assetIDTail)
		if err := ss.setAssetScript(digest, r.scriptDBItem.script, r.scriptDBItem.info.PK, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (ss *scriptsStorage) dropUncertain() {
	ss.uncertainAssetScripts = make(map[proto.AssetID]assetScriptRecordWithAssetIDTail)
}

func (ss *scriptsStorage) setAssetScriptUncertain(fullAssetID crypto.Digest, script proto.Script, pk crypto.PublicKey) error {
	// NOTE: we use fullAssetID (crypto.Digest) only for state hashes compatibility
	var (
		assetID     = proto.AssetIDFromDigest(fullAssetID)
		assetIDTail = proto.DigestTail(fullAssetID)
	)
	dbItem, err := newScriptDBItem(pk, script)
	if err != nil {
		return errors.Wrapf(err, "failed to set uncertain asset script for asset %q with pk %q",
			fullAssetID.String(), pk.String(),
		)
	}
	ss.uncertainAssetScripts[assetID] = assetScriptRecordWithAssetIDTail{
		assetIDTail:  assetIDTail,
		scriptDBItem: dbItem,
	}
	return nil
}

func (ss *scriptsStorage) setAssetScript(fullAssetID crypto.Digest, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error {
	// NOTE: we use fullAssetID (crypto.Digest) only for state hashes compatibility
	key := assetScriptKey{assetID: proto.AssetIDFromDigest(fullAssetID)}
	if ss.calculateHashes {
		as := &assetScripRecordForHashes{
			asset:  fullAssetID,
			script: script,
		}
		keyStr := string(key.bytes())
		if err := ss.assetScriptsHasher.push(keyStr, as, blockID); err != nil {
			return err
		}
	}
	dbItem, err := newScriptDBItem(pk, script)
	if err != nil {
		return errors.Wrapf(err, "failed to set asset script for asset %q with pk %q on block %q",
			fullAssetID.String(), pk.String(), blockID.String(),
		)
	}
	return ss.setScript(assetScript, &key, dbItem, blockID)
}

func (ss *scriptsStorage) newestIsSmartAsset(assetID proto.AssetID) (bool, error) {
	if r, ok := ss.uncertainAssetScripts[assetID]; ok {
		return !r.scriptDBItem.script.IsEmpty(), nil
	}
	key := assetScriptKey{assetID}
	if _, has := ss.cache.get(key.bytes()); has {
		return true, nil
	}
	infoKey := scriptBasicInfoKey{scriptKey: &key}
	recordBytes, err := ss.hs.newestTopEntryData(infoKey.bytes())
	if err != nil { // TODO: check error type
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	return info.scriptExists(), nil
}

func (ss *scriptsStorage) isSmartAsset(assetID proto.AssetID) (bool, error) {
	key := scriptBasicInfoKey{scriptKey: &assetScriptKey{assetID}}
	recordBytes, err := ss.hs.topEntryData(key.bytes())
	if err != nil { // TODO: check error type
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	return info.scriptExists(), nil
}

func (ss *scriptsStorage) newestScriptByAsset(assetID proto.AssetID) (*ast.Tree, error) {
	if r, ok := ss.uncertainAssetScripts[assetID]; ok {
		return ss.scriptAstFromRecordBytes(r.scriptDBItem.script) // Possible errors `proto.ErrNotFound` and parsing errors.
	}
	key := assetScriptKey{assetID}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return &script, nil
	}
	tree, err := ss.newestScriptAstByKey(keyBytes)
	if err != nil {
		return nil, err
	}
	ss.cache.set(keyBytes, *tree, scriptSize)
	return tree, nil
}

func (ss *scriptsStorage) scriptByAsset(assetID proto.AssetID) (*ast.Tree, error) {
	key := assetScriptKey{assetID}
	return ss.scriptTreeByKey(key.bytes())
}

func (ss *scriptsStorage) scriptBytesByAsset(assetID proto.AssetID) (proto.Script, error) {
	key := assetScriptKey{assetID}
	return ss.scriptBytesByKey(key.bytes())
}

func (ss *scriptsStorage) newestScriptBytesByAsset(assetID proto.AssetID) (proto.Script, error) {
	key := assetScriptKey{assetID}
	return ss.newestScriptBytesByKey(key.bytes())
}

func (ss *scriptsStorage) newestScriptBytesByAddr(addr proto.WavesAddress) (proto.Script, error) {
	key := accountScriptKey{addr.ID()}
	return ss.newestScriptBytesByKey(key.bytes())
}

func (ss *scriptsStorage) setAccountScript(addr proto.WavesAddress, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error {
	key := accountScriptKey{addr.ID()}
	if ss.calculateHashes {
		ac := &accountScripRecordForHashes{
			addr:   &addr,
			script: script,
		}
		keyStr := string(key.bytes())
		if err := ss.accountScriptsHasher.push(keyStr, ac, blockID); err != nil {
			return err
		}
	}
	dbItem, err := newScriptDBItem(pk, script)
	if err != nil {
		return errors.Wrapf(err, "failed to set account script for account %q with pk %q on block %q",
			addr.String(), pk.String(), blockID.String(),
		)
	}
	return ss.setScript(accountScript, &key, dbItem, blockID)
}

// newestAccountIsDApp checks that account is DApp.
// Note that only real proto.WavesAddress account can be a DApp.
func (ss *scriptsStorage) newestAccountIsDApp(addr proto.WavesAddress) (bool, error) {
	key := accountScriptKey{addr.ID()}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return script.IsDApp(), nil
	}
	infoKey := scriptBasicInfoKey{scriptKey: &key}
	recordBytes, err := ss.hs.newestTopEntryData(infoKey.bytes())
	if err != nil { // TODO: Check errors type, all NotFound like errors must be suppressed
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	if !info.scriptExists() { // Script doesn't exist, so account is not DApp
		return false, nil
	}
	return info.IsDApp, nil
}

func (ss *scriptsStorage) accountIsDApp(addr proto.WavesAddress) (bool, error) {
	key := scriptBasicInfoKey{scriptKey: &accountScriptKey{addr.ID()}}
	recordBytes, err := ss.hs.topEntryData(key.bytes())
	if err != nil { // TODO: Check errors type, all NotFound like errors must be suppressed
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	if !info.scriptExists() { // Script doesn't exist, so account is not DApp
		return false, nil
	}
	return info.IsDApp, nil
}

// newestAccountHasVerifier checks that account has verifier.
// Note that only real proto.WavesAddress account can have a verifier.
func (ss *scriptsStorage) newestAccountHasVerifier(addr proto.WavesAddress) (bool, error) {
	key := accountScriptKey{addr.ID()}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return script.HasVerifier(), nil
	}
	infoKey := scriptBasicInfoKey{scriptKey: &key}
	recordBytes, err := ss.hs.newestTopEntryData(infoKey.bytes())
	if err != nil { // TODO: Check errors type, all NotFound like errors must be suppressed
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	if !info.scriptExists() { // Script doesn't exist, so account also doesn't have verifier
		return false, nil
	}
	return info.HasVerifier, nil
}

func (ss *scriptsStorage) accountHasVerifier(addr proto.WavesAddress) (bool, error) {
	key := scriptBasicInfoKey{scriptKey: &accountScriptKey{addr.ID()}}
	recordBytes, err := ss.hs.topEntryData(key.bytes())
	if err != nil { // TODO: Check errors type, all NotFound like errors must be suppressed
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	if !info.scriptExists() { // Script doesn't exist, so account also doesn't have verifier
		return false, nil
	}
	return info.HasVerifier, nil
}

func (ss *scriptsStorage) newestAccountHasScript(addr proto.WavesAddress) (bool, error) {
	key := accountScriptKey{addr.ID()}
	if _, has := ss.cache.get(key.bytes()); has {
		return true, nil
	}
	infoKey := scriptBasicInfoKey{scriptKey: &key}
	recordBytes, err := ss.hs.newestTopEntryData(infoKey.bytes())
	if err != nil { // TODO: check error type
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	return info.scriptExists(), nil
}

func (ss *scriptsStorage) accountHasScript(addr proto.WavesAddress) (bool, error) {
	key := scriptBasicInfoKey{scriptKey: &accountScriptKey{addr.ID()}}
	recordBytes, err := ss.hs.topEntryData(key.bytes())
	if err != nil { // TODO: check error type
		return false, nil
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return false, err
	}
	return info.scriptExists(), nil
}

func (ss *scriptsStorage) newestScriptByAddr(addr proto.WavesAddress) (*ast.Tree, error) {
	key := accountScriptKey{addr.ID()}
	keyBytes := key.bytes()
	if tree, has := ss.cache.get(keyBytes); has {
		return &tree, nil
	}
	tree, err := ss.newestScriptAstByKey(keyBytes)
	if err != nil {
		return nil, err
	}
	ss.cache.set(keyBytes, *tree, scriptSize)
	return tree, nil
}

func (ss *scriptsStorage) newestScriptBasicInfoByAddressID(addressID proto.AddressID) (scriptBasicInfoRecord, error) {
	key := scriptBasicInfoKey{scriptKey: &accountScriptKey{addressID}}
	recordBytes, err := ss.hs.newestTopEntryData(key.bytes())
	if err != nil {
		return scriptBasicInfoRecord{}, err
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return scriptBasicInfoRecord{}, err
	}
	if !info.scriptExists() {
		return scriptBasicInfoRecord{}, errEmptyScript
	}
	return info, nil
}

func (ss *scriptsStorage) scriptBasicInfoByAddressID(addressID proto.AddressID) (scriptBasicInfoRecord, error) {
	key := scriptBasicInfoKey{scriptKey: &accountScriptKey{addressID}}
	recordBytes, err := ss.hs.topEntryData(key.bytes())
	if err != nil {
		return scriptBasicInfoRecord{}, err
	}
	var info scriptBasicInfoRecord
	if err := info.unmarshalBinary(recordBytes); err != nil {
		return scriptBasicInfoRecord{}, err
	}
	if !info.scriptExists() {
		return scriptBasicInfoRecord{}, errEmptyScript
	}
	return info, nil
}

// scriptByAddr returns script of corresponding proto.WavesAddress.
// Note that only real proto.WavesAddress account can have a scripts.
func (ss *scriptsStorage) scriptByAddr(addr proto.WavesAddress) (*ast.Tree, error) {
	key := accountScriptKey{addr: addr.ID()}
	return ss.scriptTreeByKey(key.bytes())
}

// scriptBytesByAddr returns script bytes of corresponding proto.WavesAddress.
// Note that only real proto.WavesAddress account can have a scripts.
func (ss *scriptsStorage) scriptBytesByAddr(addr proto.WavesAddress) (proto.Script, error) {
	key := accountScriptKey{addr: addr.ID()}
	return ss.scriptBytesByKey(key.bytes())
}

func (ss *scriptsStorage) clearCache() error {
	var err error
	ss.cache, err = newLru(maxCacheSize, maxCacheBytes)
	return err
}

func (ss *scriptsStorage) prepareHashes() error {
	if err := ss.accountScriptsHasher.stop(); err != nil {
		return err
	}
	if err := ss.assetScriptsHasher.stop(); err != nil {
		return err
	}
	return nil
}

func (ss *scriptsStorage) reset() {
	if !ss.calculateHashes {
		return
	}
	ss.assetScriptsHasher.reset()
	ss.accountScriptsHasher.reset()
}

func (ss *scriptsStorage) getAccountScriptsHasher() *stateHasher {
	return ss.accountScriptsHasher
}

func (ss *scriptsStorage) getAssetScriptsHasher() *stateHasher {
	return ss.assetScriptsHasher
}
