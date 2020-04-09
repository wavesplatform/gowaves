package state

import (
	"bytes"
	"errors"
	"io"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

const (
	maxCacheSize = 100000
	// Can't evaluate real script size, so we use 1 per each.
	scriptSize    = 1
	maxCacheBytes = maxCacheSize * scriptSize
)

func scriptBytesToAst(script proto.Script) (ast.Script, error) {
	scriptAst, err := ast.BuildScript(reader.NewBytesReader(script[:]))
	if err != nil {
		return ast.Script{}, err
	}
	return *scriptAst, nil
}

type accountScripRecordForHashes struct {
	addr   *proto.Address
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
	asset  []byte
	script proto.Script
}

func (as *assetScripRecordForHashes) writeTo(w io.Writer) error {
	if _, err := w.Write(as.asset); err != nil {
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
	return bytes.Compare(as.asset, as2.asset) == -1
}

type scriptRecord struct {
	pk     crypto.PublicKey
	script proto.Script
}

func (r *scriptRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, crypto.KeySize+len(r.script))
	copy(res, r.pk[:])
	copy(res[crypto.KeySize:], r.script)
	return res, nil
}

func (r *scriptRecord) unmarshalBinary(data []byte) error {
	if len(data) < crypto.KeySize {
		return errors.New("insufficient data for scriptRecord")
	}
	pk, err := crypto.NewPublicKeyFromBytes(data[:crypto.KeySize])
	if err != nil {
		return err
	}
	r.pk = pk
	scriptBytes := make([]byte, len(data)-crypto.KeySize)
	copy(scriptBytes, data[crypto.KeySize:])
	r.script = proto.Script(scriptBytes)
	return nil
}

type scriptsStorage struct {
	hs    *historyStorage
	cache *lru

	accountScriptsHasher *stateHasher
	assetScriptsHasher   *stateHasher
	calculateHashes      bool
}

func newScriptsStorage(hs *historyStorage, calcHashes bool) (*scriptsStorage, error) {
	cache, err := newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return nil, err
	}
	return &scriptsStorage{
		hs:                   hs,
		cache:                cache,
		accountScriptsHasher: newStateHasher(),
		assetScriptsHasher:   newStateHasher(),
		calculateHashes:      calcHashes,
	}, nil
}

func (ss *scriptsStorage) setScript(scriptType blockchainEntity, key []byte, record scriptRecord, blockID proto.BlockID) error {
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := ss.hs.addNewEntry(scriptType, key, recordBytes, blockID); err != nil {
		return err
	}
	if len(record.script) == 0 {
		// There is no AST for empty script.
		ss.cache.deleteIfExists(key)
		return nil
	}
	scriptAst, err := scriptBytesToAst(record.script)
	if err != nil {
		return err
	}
	ss.cache.set(key, scriptAst, scriptSize)
	return nil
}

func (ss *scriptsStorage) scriptBytesByKey(key []byte, filter bool) (proto.Script, error) {
	recordBytes, err := ss.hs.latestEntryData(key, filter)
	if err != nil {
		return proto.Script{}, err
	}
	var record scriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return proto.Script{}, err
	}
	return record.script, nil
}

func (ss *scriptsStorage) scriptAstFromRecordBytes(recordBytes []byte) (ast.Script, crypto.PublicKey, error) {
	var record scriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, crypto.PublicKey{}, err
	}
	if len(record.script) == 0 {
		// Empty script = no script.
		return ast.Script{}, crypto.PublicKey{}, proto.ErrNotFound
	}
	script, err := scriptBytesToAst(record.script)
	return script, record.pk, err
}

func (ss *scriptsStorage) newestScriptAstByKey(key []byte, filter bool) (ast.Script, error) {
	recordBytes, err := ss.hs.freshLatestEntryData(key, filter)
	if err != nil {
		return ast.Script{}, err
	}
	script, _, err := ss.scriptAstFromRecordBytes(recordBytes)
	return script, err
}

func (ss *scriptsStorage) scriptAstByKey(key []byte, filter bool) (ast.Script, error) {
	recordBytes, err := ss.hs.latestEntryData(key, filter)
	if err != nil {
		return ast.Script{}, err
	}
	script, _, err := ss.scriptAstFromRecordBytes(recordBytes)
	return script, err
}

func (ss *scriptsStorage) setAssetScript(assetID crypto.Digest, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error {
	key := assetScriptKey{assetID}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := scriptRecord{pk: pk, script: script}
	if ss.calculateHashes {
		as := &assetScripRecordForHashes{
			asset:  assetID[:],
			script: script,
		}
		if err := ss.assetScriptsHasher.push(keyStr, as, blockID); err != nil {
			return err
		}
	}
	return ss.setScript(assetScript, keyBytes, record, blockID)
}

func (ss *scriptsStorage) newestIsSmartAsset(assetID crypto.Digest, filter bool) (bool, error) {
	key := assetScriptKey{assetID}
	keyBytes := key.bytes()
	if _, has := ss.cache.get(keyBytes); has {
		return true, nil
	}
	recordBytes, err := ss.hs.freshLatestEntryData(keyBytes, filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) > crypto.KeySize, nil
}

func (ss *scriptsStorage) isSmartAsset(assetID crypto.Digest, filter bool) (bool, error) {
	key := assetScriptKey{assetID}
	recordBytes, err := ss.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) > crypto.KeySize, nil
}

func (ss *scriptsStorage) newestScriptByAsset(assetID crypto.Digest, filter bool) (ast.Script, error) {
	key := assetScriptKey{assetID}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return script, nil
	}
	script, err := ss.newestScriptAstByKey(keyBytes, filter)
	if err != nil {
		return ast.Script{}, err
	}
	ss.cache.set(keyBytes, script, scriptSize)
	return script, nil
}

func (ss *scriptsStorage) scriptByAsset(assetID crypto.Digest, filter bool) (ast.Script, error) {
	key := assetScriptKey{assetID}
	return ss.scriptAstByKey(key.bytes(), filter)
}

func (ss *scriptsStorage) scriptBytesByAsset(assetID crypto.Digest, filter bool) (proto.Script, error) {
	key := assetScriptKey{assetID}
	return ss.scriptBytesByKey(key.bytes(), filter)
}

func (ss *scriptsStorage) setAccountScript(addr proto.Address, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error {
	key := accountScriptKey{addr}
	keyBytes := key.bytes()
	keyStr := string(keyBytes)
	record := scriptRecord{pk: pk, script: script}
	if ss.calculateHashes {
		ac := &accountScripRecordForHashes{
			addr:   &addr,
			script: script,
		}
		if err := ss.accountScriptsHasher.push(keyStr, ac, blockID); err != nil {
			return err
		}
	}
	return ss.setScript(accountScript, keyBytes, record, blockID)
}

func (ss *scriptsStorage) newestAccountHasVerifier(addr proto.Address, filter bool) (bool, error) {
	key := accountScriptKey{addr}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return script.HasVerifier(), nil
	}
	script, err := ss.newestScriptAstByKey(keyBytes, filter)
	if err != nil {
		return false, nil
	}
	accountHasVerifier := script.HasVerifier()
	return accountHasVerifier, nil
}

func (ss *scriptsStorage) accountHasVerifier(addr proto.Address, filter bool) (bool, error) {
	script, err := ss.scriptByAddr(addr, filter)
	if err != nil {
		return false, nil
	}
	accountHasVerifier := script.HasVerifier()
	return accountHasVerifier, nil
}

func (ss *scriptsStorage) newestAccountHasScript(addr proto.Address, filter bool) (bool, error) {
	key := accountScriptKey{addr}
	keyBytes := key.bytes()
	if _, has := ss.cache.get(keyBytes); has {
		return true, nil
	}
	recordBytes, err := ss.hs.freshLatestEntryData(keyBytes, filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) > crypto.KeySize, nil
}

func (ss *scriptsStorage) accountHasScript(addr proto.Address, filter bool) (bool, error) {
	key := accountScriptKey{addr}
	recordBytes, err := ss.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) > crypto.KeySize, nil
}

func (ss *scriptsStorage) newestScriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr}
	keyBytes := key.bytes()
	if script, has := ss.cache.get(keyBytes); has {
		return script, nil
	}
	script, err := ss.newestScriptAstByKey(keyBytes, filter)
	if err != nil {
		return ast.Script{}, err
	}
	ss.cache.set(keyBytes, script, scriptSize)
	return script, nil
}

func (ss *scriptsStorage) newestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error) {
	key := accountScriptKey{addr}
	recordBytes, err := ss.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return crypto.PublicKey{}, err
	}
	_, pk, err := ss.scriptAstFromRecordBytes(recordBytes)
	return pk, err
}

func (ss *scriptsStorage) scriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr}
	return ss.scriptAstByKey(key.bytes(), filter)
}

func (ss *scriptsStorage) scriptBytesByAddr(addr proto.Address, filter bool) (proto.Script, error) {
	key := accountScriptKey{addr}
	return ss.scriptBytesByKey(key.bytes(), filter)
}

func (ss *scriptsStorage) clear() error {
	var err error
	ss.cache, err = newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return err
	}
	return nil
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
