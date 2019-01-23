package data

import "github.com/wavesplatform/gowaves/pkg/crypto"

var (
	WavesID        = crypto.Digest{}
	WavesAssetInfo = AssetInfo{ID: WavesID, Name: "WAVES", Issuer: crypto.PublicKey{}, Decimals: 8, Reissuable: false, Supply: 10000000000000000}
)

type AssetInfo struct {
	ID         crypto.Digest
	Name       string
	Issuer     crypto.PublicKey
	Decimals   uint8
	Reissuable bool
	Supply     uint64
}
