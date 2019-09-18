package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
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
	return ast.BuildAst(reader.NewBytesReader(script[:]))
}

type accountScriptRecord struct {
	script proto.Script
}

func (r *accountScriptRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, len(r.script))
	copy(res, r.script)
	return res, nil
}

func (r *accountScriptRecord) unmarshalBinary(data []byte) error {
	scriptBytes := make([]byte, len(data))
	copy(scriptBytes, data)
	r.script = proto.Script(scriptBytes)
	return nil
}

type accountsScripts struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage

	cache *lru
}

func newAccountsScripts(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage) (*accountsScripts, error) {
	cache, err := newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return nil, err
	}
	return &accountsScripts{
		db:      db,
		dbBatch: dbBatch,
		hs:      hs,
		cache:   cache,
	}, nil
}

func (as *accountsScripts) setScript(addr proto.Address, script proto.Script, blockID crypto.Signature) error {
	key := accountScriptKey{addr}
	record := accountScriptRecord{script}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := as.hs.addNewEntry(accountScript, key.bytes(), recordBytes, blockID); err != nil {
		return err
	}
	if len(script) == 0 {
		// There is no AST for empty script.
		as.cache.deleteIfExists(addr)
		return nil
	}
	scriptAst, err := scriptBytesToAst(record.script)
	if err != nil {
		return err
	}
	as.cache.set(addr, scriptAst, scriptSize)
	return nil
}

func (as *accountsScripts) newestScriptAstFromAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr: addr}
	recordBytes, err := as.hs.freshLatestEntryData(key.bytes(), filter)
	if err != nil {
		return ast.Script{}, err
	}
	var record accountScriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, err
	}
	if len(record.script) == 0 {
		// Empty script = no script.
		return ast.Script{}, proto.ErrNotFound
	}
	return scriptBytesToAst(record.script)
}

func (as *accountsScripts) newestHasVerifier(addr proto.Address, filter bool) (bool, error) {
	if script, has := as.cache.get(addr); has {
		return (script.Verifier != nil), nil
	}
	script, err := as.newestScriptAstFromAddr(addr, filter)
	if err != nil {
		return false, nil
	}
	hasVerifier := (script.Verifier != nil)
	return hasVerifier, nil
}

func (as *accountsScripts) hasVerifier(addr proto.Address, filter bool) (bool, error) {
	script, err := as.scriptByAddr(addr, filter)
	if err != nil {
		return false, nil
	}
	hasVerifier := (script.Verifier != nil)
	return hasVerifier, nil
}

func (as *accountsScripts) newestHasScript(addr proto.Address, filter bool) (bool, error) {
	if _, has := as.cache.get(addr); has {
		return true, nil
	}
	if _, err := as.newestScriptAstFromAddr(addr, filter); err == nil {
		return true, nil
	}
	return false, nil
}

func (as *accountsScripts) hasScript(addr proto.Address, filter bool) (bool, error) {
	if _, err := as.scriptByAddr(addr, filter); err == nil {
		return true, nil
	}
	return false, nil
}

func (as *accountsScripts) newestScriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	if script, has := as.cache.get(addr); has {
		return script, nil
	}
	script, err := as.newestScriptAstFromAddr(addr, filter)
	if err != nil {
		return ast.Script{}, err
	}
	as.cache.set(addr, script, scriptSize)
	return script, nil
}

func (as *accountsScripts) scriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr: addr}
	recordBytes, err := as.hs.latestEntryData(key.bytes(), filter)
	if err != nil {
		return ast.Script{}, err
	}
	var record accountScriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, err
	}
	if len(record.script) == 0 {
		// Empty script = no script.
		return ast.Script{}, proto.ErrNotFound
	}
	return scriptBytesToAst(record.script)
}

func (as *accountsScripts) clear() error {
	var err error
	as.cache, err = newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return err
	}
	return nil
}
