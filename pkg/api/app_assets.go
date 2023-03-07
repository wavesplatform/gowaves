package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ScriptDetails struct {
	ScriptComplexity uint64         `json:"scriptComplexity"`
	Script           proto.B64Bytes `json:"script"`
}

type AssetDetails struct {
	AssetId              crypto.Digest      `json:"assetId"`
	IssueHeight          proto.Height       `json:"issueHeight"`
	IssueTimestamp       proto.Timestamp    `json:"issueTimestamp"`
	Issuer               proto.WavesAddress `json:"issuer"`
	IssuerPublicKey      crypto.PublicKey   `json:"issuerPublicKey"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Decimals             byte               `json:"decimals"`
	Reissuable           bool               `json:"reissuable"`
	Quantity             uint64             `json:"quantity"`
	Scripted             bool               `json:"scripted"`
	MinSponsoredAssetFee *uint64            `json:"minSponsoredAssetFee"`
	OriginTransactionId  proto.B58Bytes     `json:"originTransactionId"`
	ScriptDetails        *ScriptDetails     `json:"scriptDetails,omitempty"`
}

func (a *App) AssetsDetailsByID(fullAssetID crypto.Digest, full bool) (*AssetDetails, error) {
	assetID := proto.AssetIDFromDigest(fullAssetID)
	assetInfo, err := a.state.FullAssetInfo(assetID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get info about asset")
	}
	txID, err := assetInfo.IssueTransaction.GetID(a.services.Scheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get txID for asset")
	}
	var minSponsoredAssetFee *uint64
	if assetInfo.SponsorshipCost != 0 {
		cost := assetInfo.SponsorshipCost
		minSponsoredAssetFee = &cost
	}
	assetDetails := &AssetDetails{
		AssetId:              assetInfo.ID,
		IssueHeight:          0, // TODO(nickeskov): add issue height to asset info
		IssueTimestamp:       assetInfo.IssueTransaction.GetTimestamp(),
		Issuer:               assetInfo.Issuer,
		IssuerPublicKey:      assetInfo.IssuerPublicKey,
		Name:                 assetInfo.Name,
		Description:          assetInfo.Description,
		Decimals:             assetInfo.Decimals,
		Reissuable:           assetInfo.Reissuable,
		Quantity:             assetInfo.Quantity,
		Scripted:             assetInfo.Scripted,
		MinSponsoredAssetFee: minSponsoredAssetFee,
		OriginTransactionId:  txID,
		ScriptDetails:        nil,
	}
	if assetInfo.Scripted && full {
		scriptInfo, err := a.state.ScriptInfoByAsset(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get script info for scripted asset")
		}
		assetDetails.ScriptDetails = &ScriptDetails{
			ScriptComplexity: scriptInfo.Complexity,
			Script:           scriptInfo.Bytes,
		}
	}
	return assetDetails, nil
}
