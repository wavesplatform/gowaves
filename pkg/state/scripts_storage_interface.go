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
	setAssetScriptUncertain(fullAssetID crypto.Digest, script proto.Script, pk crypto.PublicKey) error
	setAssetScript(assetID crypto.Digest, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestIsSmartAsset(assetID proto.AssetID) (bool, error)
	isSmartAsset(assetID proto.AssetID) (bool, error)
	newestScriptByAsset(assetID proto.AssetID) (*ast.Tree, error)
	scriptByAsset(assetID proto.AssetID) (*ast.Tree, error)
	scriptBytesByAsset(assetID proto.AssetID) (proto.Script, error)
	newestScriptBytesByAsset(assetID proto.AssetID) (proto.Script, error)
	newestScriptBytesByAddr(addr proto.WavesAddress) (proto.Script, error)
	setAccountScript(addr proto.WavesAddress, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestAccountIsDApp(addr proto.WavesAddress) (bool, error)
	accountIsDApp(addr proto.WavesAddress) (bool, error)
	newestAccountHasVerifier(addr proto.WavesAddress) (bool, error)
	accountHasVerifier(addr proto.WavesAddress) (bool, error)
	newestAccountHasScript(addr proto.WavesAddress) (bool, error)
	accountHasScript(addr proto.WavesAddress) (bool, error)
	newestScriptByAddr(addr proto.WavesAddress) (*ast.Tree, error)
	newestScriptBasicInfoByAddressID(addressID proto.AddressID) (scriptBasicInfoRecord, error)
	scriptBasicInfoByAddressID(addressID proto.AddressID) (scriptBasicInfoRecord, error)
	scriptByAddr(addr proto.WavesAddress) (*ast.Tree, error)
	scriptBytesByAddr(addr proto.WavesAddress) (proto.Script, error)
	clearCache() error
	prepareHashes() error
	reset()
	getAccountScriptsHasher() *stateHasher
	getAssetScriptsHasher() *stateHasher
}
