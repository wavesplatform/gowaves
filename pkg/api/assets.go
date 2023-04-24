package api

import (
	"github.com/pkg/errors"
	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type ScriptDetails struct {
	ScriptComplexity uint64         `json:"scriptComplexity"`
	Script           proto.B64Bytes `json:"script"`
}

type AssetDetails struct {
	AssetId              crypto.Digest      `json:"assetId"`
	IssueHeight          proto.Height       `json:"issueHeight"`
	IssueTimestamp       proto.Timestamp    `json:"issueTimestamp,omitempty"`
	Issuer               proto.WavesAddress `json:"issuer"`
	IssuerPublicKey      crypto.PublicKey   `json:"issuerPublicKey"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Decimals             byte               `json:"decimals"`
	Reissuable           bool               `json:"reissuable"`
	Quantity             uint64             `json:"quantity"`
	Scripted             bool               `json:"scripted"`
	MinSponsoredAssetFee *uint64            `json:"minSponsoredAssetFee"`
	OriginTransactionId  proto.B58Bytes     `json:"originTransactionId,omitempty"`
	SequenceInBlock      uint32             `json:"sequenceInBlock"`
	ScriptDetails        *ScriptDetails     `json:"scriptDetails,omitempty"`
}

func (a *App) AssetsDetailsByID(fullAssetID crypto.Digest, full bool) (*AssetDetails, error) {
	details, err := a.assetsDetailsByID(fullAssetID, full)
	if err != nil {
		return nil, err
	}
	return &details, err
}

func (a *App) assetsDetailsByID(fullAssetID crypto.Digest, full bool) (AssetDetails, error) {
	assetID := proto.AssetIDFromDigest(fullAssetID)
	assetInfo, err := a.state.EnrichedFullAssetInfo(assetID)
	if err != nil {
		return AssetDetails{}, errors.Wrap(err, "failed to get info about asset")
	}
	var (
		txID []byte
		ts   uint64
	)
	if tx := assetInfo.IssueTransaction; tx != nil {
		txID, err = tx.GetID(a.services.Scheme)
		if err != nil {
			return AssetDetails{}, errors.Wrap(err, "failed to get txID for asset")
		}
		ts = tx.GetTimestamp()
	}
	var minSponsoredAssetFee *uint64
	if assetInfo.SponsorshipCost != 0 {
		cost := assetInfo.SponsorshipCost
		minSponsoredAssetFee = &cost
	}
	assetDetails := AssetDetails{
		AssetId:              assetInfo.ID,
		IssueHeight:          assetInfo.IssueHeight,
		IssueTimestamp:       ts,
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
		SequenceInBlock:      assetInfo.SequenceInBlock,
		ScriptDetails:        nil,
	}
	if assetInfo.Scripted && full {
		scriptInfo, err := a.state.ScriptInfoByAsset(assetID)
		if err != nil {
			return AssetDetails{}, errors.Wrap(err, "failed to get script info for scripted asset")
		}
		assetDetails.ScriptDetails = &ScriptDetails{
			ScriptComplexity: scriptInfo.Complexity,
			Script:           scriptInfo.Bytes,
		}
	}
	return assetDetails, nil
}

func (a *App) AssetsDetails(fullAssetsIDs []crypto.Digest, full bool) ([]AssetDetails, error) {
	if limit := a.settings.AssetDetailsLimit; len(fullAssetsIDs) > limit {
		return nil, apiErrs.NewTooBigArrayAllocationError(limit)
	}

	assetDetails := make([]AssetDetails, len(fullAssetsIDs))
	for i, fullAssetsID := range fullAssetsIDs {
		details, err := a.assetsDetailsByID(fullAssetsID, full)
		if err != nil {
			if errors.Is(err, errs.UnknownAsset{}) {
				return nil, a.generateAssetsDoesNotExistError(fullAssetsIDs[i:])
			}
			return nil, errors.Wrapf(err, "failed to get asset details by assetID=%q", fullAssetsID)
		}
		assetDetails[i] = details
	}
	return assetDetails, nil
}

func (a *App) generateAssetsDoesNotExistError(fullAssetsIDs []crypto.Digest) error {
	var notFoundAssets []string
	for _, fullAssetsID := range fullAssetsIDs {
		exist, err := a.state.IsAssetExist(proto.AssetIDFromDigest(fullAssetsID))
		if err != nil {
			return errors.Wrapf(err, "failed to check asset=%q whether it exists or not", fullAssetsID)
		}
		if !exist {
			notFoundAssets = append(notFoundAssets, fullAssetsID.String())
		}
	}
	if len(notFoundAssets) != 0 {
		return apiErrs.NewAssetsDoesNotExistError(notFoundAssets)
	}
	return nil
}
