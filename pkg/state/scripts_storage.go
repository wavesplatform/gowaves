package state

import (
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

type scriptRecord struct {
	script proto.Script
}

func (r *scriptRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, len(r.script))
	copy(res, r.script)
	return res, nil
}

func (r *scriptRecord) unmarshalBinary(data []byte) error {
	scriptBytes := make([]byte, len(data))
	copy(scriptBytes, data)
	r.script = proto.Script(scriptBytes)
	return nil
}

type scriptsStorage struct {
	hs    *historyStorage
	cache *lru
}

func newScriptsStorage(hs *historyStorage) (*scriptsStorage, error) {
	cache, err := newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return nil, err
	}
	return &scriptsStorage{hs, cache}, nil
}

func (ss *scriptsStorage) setScript(scriptType blockchainEntity, key []byte, record scriptRecord, blockID crypto.Signature) error {
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

func (ss *scriptsStorage) scriptAstFromRecordBytes(recordBytes []byte) (ast.Script, error) {
	var record scriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, err
	}
	if len(record.script) == 0 {
		// Empty script = no script.
		return ast.Script{}, proto.ErrNotFound
	}
	return scriptBytesToAst(record.script)
}

func (ss *scriptsStorage) newestScriptAstByKey(key []byte, filter bool) (ast.Script, error) {
	recordBytes, err := ss.hs.freshLatestEntryData(key, filter)
	if err != nil {
		return ast.Script{}, err
	}
	return ss.scriptAstFromRecordBytes(recordBytes)
}

func (ss *scriptsStorage) scriptAstByKey(key []byte, filter bool) (ast.Script, error) {
	recordBytes, err := ss.hs.latestEntryData(key, filter)
	if err != nil {
		return ast.Script{}, err
	}
	return ss.scriptAstFromRecordBytes(recordBytes)
}

func (ss *scriptsStorage) setAssetScript(assetID crypto.Digest, script proto.Script, blockID crypto.Signature) error {
	key := assetScriptKey{assetID}
	record := scriptRecord{script}
	return ss.setScript(assetScript, key.bytes(), record, blockID)
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
	return len(recordBytes) != 0, nil
}

func (ss *scriptsStorage) isSmartAsset(assetID crypto.Digest, filter bool) (bool, error) {
	key := assetScriptKey{assetID}
	recordBytes, err := ss.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) != 0, nil
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

func (ss *scriptsStorage) setAccountScript(addr proto.Address, script proto.Script, blockID crypto.Signature) error {
	key := accountScriptKey{addr}
	record := scriptRecord{script}
	return ss.setScript(accountScript, key.bytes(), record, blockID)
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
	return len(recordBytes) != 0, nil
}

func (ss *scriptsStorage) accountHasScript(addr proto.Address, filter bool) (bool, error) {
	key := accountScriptKey{addr}
	recordBytes, err := ss.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return false, nil
	}
	return len(recordBytes) != 0, nil
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

func (ss *scriptsStorage) scriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr}
	return ss.scriptAstByKey(key.bytes(), filter)
}

func (ss *scriptsStorage) clear() error {
	var err error
	ss.cache, err = newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return err
	}
	return nil
}
