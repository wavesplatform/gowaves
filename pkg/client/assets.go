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

func NewAssets(options Options) *Assets {
	return &Assets{
		options: options,
	}
}

type AssetsBalances struct {
	Address  proto.Address   `json:"address"`
	Balances []AssetsBalance `json:"balances"`
}

type AssetsBalance struct {
	AssetId              crypto.Digest `json:"assetId"`
	Balance              uint64        `json:"balance"`
	Reissuable           bool          `json:"reissuable"`
	MinSponsoredAssetFee uint64        `json:"minSponsoredAssetFee"`
	SponsorBalance       uint64        `json:"sponsorBalance"`
	Quantity             uint64        `json:"quantity"`
	IssueTransaction     proto.IssueV1 `json:"issueTransaction"`
}

// Provides detailed information about given asset
func (a *Assets) BalanceByAddress(ctx context.Context, address proto.Address) (*AssetsBalances, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/assets/balance/%s", a.options.BaseUrl, address.String()),
		nil)
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
	Address proto.Address `json:"address"`
	AssetId crypto.Digest `json:"assetId"`
	Balance uint64        `json:"balance"`
}

// Account's balance by given asset
func (a *Assets) BalanceByAddressAndAsset(ctx context.Context, address proto.Address, assetId crypto.Digest) (*AssetsBalanceAndAsset, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/assets/balance/%s/%s", a.options.BaseUrl, address.String(), assetId.String()),
		nil)
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
	AssetId              crypto.Digest `json:"assetId"`
	IssueHeight          uint64        `json:"issueHeight"`
	IssueTimestamp       uint64        `json:"issueTimestamp"`
	Issuer               proto.Address `json:"issuer"`
	Name                 string        `json:"name"`
	Description          string        `json:"description"`
	Decimals             uint64        `json:"decimals"`
	Reissuable           bool          `json:"reissuable"`
	Quantity             uint64        `json:"quantity"`
	MinSponsoredAssetFee uint64        `json:"minSponsoredAssetFee"`
}

// Provides detailed information about given asset
func (a *Assets) Details(ctx context.Context, assetId crypto.Digest) (*AssetsDetail, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/assets/details/%s", a.options.BaseUrl, assetId.String()),
		nil)
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

type AssetsDistribution map[string]uint64

func (a *Assets) Distribution(ctx context.Context, assetId crypto.Digest) (AssetsDistribution, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/assets/%s/distribution", a.options.BaseUrl, assetId.String()),
		nil)
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
