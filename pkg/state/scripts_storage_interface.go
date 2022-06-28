package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

//go:generate moq -out scripts_storage_moq_test.go . scriptStorageState:mockScriptStorageState
type scriptStorageState interface {
	commitUncertain(blockID proto.BlockID) error
	dropUncertain()
	setAssetScriptUncertain(fullAssetID crypto.Digest, script proto.Script, pk crypto.PublicKey)
	setAssetScript(assetID crypto.Digest, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestIsSmartAsset(assetID proto.AssetID, filter bool) (bool, error)
	isSmartAsset(assetID proto.AssetID, filter bool) (bool, error)
	newestScriptByAsset(assetID proto.AssetID, filter bool) (*ast.Tree, error)
	scriptByAsset(assetID proto.AssetID, filter bool) (*ast.Tree, error)
	scriptBytesByAsset(assetID proto.AssetID, filter bool) (proto.Script, error)
	newestScriptBytesByAsset(assetID proto.AssetID, filter bool) (proto.Script, error)
	newestScriptBytesByAddr(addr proto.WavesAddress, filter bool) (proto.Script, error)
	setAccountScript(addr proto.WavesAddress, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestAccountHasVerifier(addr proto.WavesAddress, filter bool) (bool, error)
	accountHasVerifier(addr proto.WavesAddress, filter bool) (bool, error)
	newestAccountHasScript(addr proto.WavesAddress, filter bool) (bool, error)
	accountHasScript(addr proto.WavesAddress, filter bool) (bool, error)
	newestScriptByAddr(addr proto.WavesAddress, filter bool) (*ast.Tree, error)
	newestScriptPKByAddr(addr proto.WavesAddress, filter bool) (crypto.PublicKey, error)
	scriptByAddr(addr proto.WavesAddress, filter bool) (*ast.Tree, error)
	scriptBytesByAddr(addr proto.WavesAddress, filter bool) (proto.Script, error)
	clearCache() error
	prepareHashes() error
	reset()
	getAccountScriptsHasher() *stateHasher
	getAssetScriptsHasher() *stateHasher
}
