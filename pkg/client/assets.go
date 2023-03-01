package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Assets struct {
	options Options
}

// NewAssets creates new Assets
func NewAssets(options Options) *Assets {
	return &Assets{
		options: options,
	}
}

type AssetsBalances struct {
	Address  proto.WavesAddress `json:"address"`
	Balances []AssetsBalance    `json:"balances"`
}

type AssetsBalance struct {
	AssetId              crypto.Digest      `json:"assetId"`
	Balance              uint64             `json:"balance"`
	Reissuable           bool               `json:"reissuable"`
	MinSponsoredAssetFee uint64             `json:"minSponsoredAssetFee"`
	SponsorBalance       uint64             `json:"sponsorBalance"`
	Quantity             uint64             `json:"quantity"`
	IssueTransaction     proto.IssueWithSig `json:"issueTransaction"`
}

// BalanceByAddress provides detailed information about given asset
func (a *Assets) BalanceByAddress(ctx context.Context, address proto.WavesAddress) (*AssetsBalances, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/assets/balance/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AssetsBalances)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsBalanceAndAsset struct {
	Address proto.WavesAddress `json:"address"`
	AssetId crypto.Digest      `json:"assetId"`
	Balance uint64             `json:"balance"`
}

// BalanceByAddressAndAsset returns account's balance by given asset.
func (a *Assets) BalanceByAddressAndAsset(ctx context.Context, address proto.WavesAddress, assetId crypto.Digest) (*AssetsBalanceAndAsset, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/assets/balance/%s/%s", address.String(), assetId.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AssetsBalanceAndAsset)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsDetail struct {
	AssetId              crypto.Digest      `json:"assetId"`
	IssueHeight          uint64             `json:"issueHeight"`
	IssueTimestamp       uint64             `json:"issueTimestamp"`
	Issuer               proto.WavesAddress `json:"issuer"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Decimals             uint64             `json:"decimals"`
	Reissuable           bool               `json:"reissuable"`
	Quantity             uint64             `json:"quantity"`
	MinSponsoredAssetFee uint64             `json:"minSponsoredAssetFee"`
}

// Details provides detailed information about given asset.
func (a *Assets) Details(ctx context.Context, assetId crypto.Digest) (*AssetsDetail, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/assets/details/%s", assetId.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AssetsDetail)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsDistributionAtHeight struct {
	HasNext  bool                          `json:"hasNext"`
	LastItem proto.WavesAddress            `json:"lastItem"`
	Items    map[proto.WavesAddress]uint64 `json:"items"`
}

// DistributionAtHeight gets asset balance distribution by an account at provided height.
// Result records are limited by limit param. after param is optional and used for pagination.
func (a *Assets) DistributionAtHeight(ctx context.Context, assetId crypto.Digest, height, limit uint64, after *proto.WavesAddress) (*AssetsDistributionAtHeight, *Response, error) {
	var rawPath string
	if after != nil {
		rawPath = fmt.Sprintf("/assets/%s/distribution/%d/limit/%d?after=%s", assetId.String(), height, limit, after.String())
	} else {
		rawPath = fmt.Sprintf("/assets/%s/distribution/%d/limit/%d", assetId.String(), height, limit)
	}
	url, err := joinUrl(a.options.BaseUrl, rawPath)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(AssetsDistributionAtHeight)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsDistribution map[string]uint64

// Distribution gets asset balance distribution by account.
// Deprecated: use DistributionAtHeight method.
func (a *Assets) Distribution(ctx context.Context, assetId crypto.Digest) (AssetsDistribution, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/assets/%s/distribution", assetId.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := make(AssetsDistribution)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
