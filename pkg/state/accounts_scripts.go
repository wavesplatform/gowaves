package state

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/parser"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
)

const (
	maxCacheSize = 100000
	// Can't evaluate real script size, so we use 1 per each.
	scriptSize    = 1
	maxCacheBytes = maxCacheSize * scriptSize
)

func scriptBytesToAst(script proto.Script) (ast.Script, error) {
	return parser.BuildAst(reader.NewBytesReader(script[:]))
}

type accountScriptRecord struct {
	script   proto.Script
	blockNum uint32
}

func (r *accountScriptRecord) marshalBinary() ([]byte, error) {
	res := make([]byte, len(r.script)+4)
	copy(res[:len(r.script)], r.script)
	binary.BigEndian.PutUint32(res[len(r.script):len(r.script)+4], r.blockNum)
	return res, nil
}

func (r *accountScriptRecord) unmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return errors.New("invalid data size")
	}
	scriptBytes := make([]byte, len(data)-4)
	copy(scriptBytes, data[:len(data)-4])
	r.script = proto.Script(scriptBytes)
	r.blockNum = binary.BigEndian.Uint32(data[len(data)-4:])
	return nil
}

type accountsScripts struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	hs      *historyStorage
	stateDB *stateDB

	cache *lru
}

func newAccountsScripts(db keyvalue.IterableKeyVal, dbBatch keyvalue.Batch, hs *historyStorage, stateDB *stateDB) (*accountsScripts, error) {
	cache, err := newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return nil, err
	}
	return &accountsScripts{
		db:      db,
		dbBatch: dbBatch,
		hs:      hs,
		stateDB: stateDB,
		cache:   cache,
	}, nil
}

func (as *accountsScripts) setScript(addr proto.Address, script proto.Script, blockID crypto.Signature) error {
	key := accountScriptKey{addr}
	blockNum, err := as.stateDB.blockIdToNum(blockID)
	if err != nil {
		return err
	}
	record := accountScriptRecord{script, blockNum}
	recordBytes, err := record.marshalBinary()
	if err != nil {
		return err
	}
	if err := as.hs.set(accountScript, key.bytes(), recordBytes); err != nil {
		return err
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
	recordBytes, err := as.hs.getFresh(key.bytes(), filter)
	if err != nil {
		return ast.Script{}, err
	}
	var record accountScriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, err
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
	key := accountScriptKey{addr: addr}
	if _, err := as.hs.getFresh(key.bytes(), filter); err == nil {
		return true, nil
	}
	return false, nil
}

func (as *accountsScripts) hasScript(addr proto.Address, filter bool) (bool, error) {
	key := accountScriptKey{addr: addr}
	if _, err := as.hs.get(key.bytes(), filter); err == nil {
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
		return ast.Script{}, nil
	}
	as.cache.set(addr, script, scriptSize)
	return script, nil
}

func (as *accountsScripts) scriptByAddr(addr proto.Address, filter bool) (ast.Script, error) {
	key := accountScriptKey{addr: addr}
	recordBytes, err := as.hs.get(key.bytes(), filter)
	if err != nil {
		return ast.Script{}, err
	}
	var record accountScriptRecord
	if err := record.unmarshalBinary(recordBytes); err != nil {
		return ast.Script{}, err
	}
	return scriptBytesToAst(record.script)
}

func (as *accountsScripts) callVerifier(addr proto.Address, tx proto.Transaction, scope ast.Scope, filter bool) (bool, error) {
	script, err := as.newestScriptByAddr(addr, filter)
	if err != nil {
		return false, err
	}
	if script.Verifier == nil {
		return false, errors.New("script does not have verifier set")
	}
	res, err := script.Verifier.Evaluate(scope)
	if err != nil {
		return false, err
	}
	isTrue, err := res.Eq(ast.NewBoolean(true))
	if err != nil {
		return false, err
	}
	return isTrue, nil
}

func (as *accountsScripts) clear() error {
	var err error
	as.cache, err = newLru(maxCacheSize, maxCacheBytes)
	if err != nil {
		return err
	}
	return nil
}
