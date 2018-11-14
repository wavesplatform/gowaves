package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
	"time"
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
	Address proto.Address `json:"address"`
	AssetId crypto.Digest `json:"assetId"`
	Balance uint64        `json:"balance"`
}

// Account's balance by given asset
func (a *Assets) BalanceByAddressAndAsset(ctx context.Context, address proto.Address, assetId crypto.Digest) (*AssetsBalanceAndAsset, *Response, error) {
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

type AssetsDistribution map[string]uint64

// Asset balance distribution by account
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

type AssetsIssueReq struct {
	Sender      proto.Address `json:"sender"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Quantity    uint64        `json:"quantity"`
	Decimals    uint8         `json:"decimals"`
	Reissuable  bool          `json:"reissuable"`
	Fee         uint64        `json:"fee"`
	Timestamp   uint64        `json:"timestamp"`
}

type AssetsIssue struct {
	Sender      proto.Address `json:"sender"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Quantity    uint64        `json:"quantity"`
	Decimals    uint8         `json:"decimals"`
	Reissuable  bool          `json:"reissuable"`
	Fee         uint64        `json:"fee"`
	Timestamp   uint64        `json:"timestamp"`
}

// Issue new Asset
func (a *Assets) Issue(ctx context.Context, issueReq AssetsIssueReq) (*AssetsIssue, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/assets/issue")
	if err != nil {
		return nil, nil, err
	}

	if issueReq.Timestamp == 0 {
		issueReq.Timestamp = NewTimestampFromTime(time.Now())
	}

	bts, err := json.Marshal(issueReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(AssetsIssue)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsMassTransfersReq struct {
	Version    uint8                   `json:"version"`
	AssetId    crypto.Digest           `json:"asset_id"`
	Sender     proto.Address           `json:"sender"`
	Transfers  []AssetsMassTransferReq `json:"transfers"`
	Fee        uint64                  `json:"fee"`
	Attachment proto.Attachment        `json:"attachment"`
	Timestamp  uint64                  `json:"timestamp"`
}

type AssetsMassTransferReq struct {
	Recipient proto.Address `json:"recipient"`
	Amount    uint64        `json:"amount"`
}

// Mass transfer of assets
func (a *Assets) MassTransfer(ctx context.Context, transfersReq AssetsMassTransfersReq) (*proto.MassTransferV1, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/assets/masstransfer")
	if err != nil {
		return nil, nil, err
	}

	if transfersReq.Timestamp == 0 {
		transfersReq.Timestamp = NewTimestampFromTime(time.Now())
	}
	if transfersReq.Version == 0 {
		transfersReq.Version = 1
	}

	bts, err := json.Marshal(transfersReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(proto.MassTransferV1)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsSponsorReq struct {
	Sender               proto.Address `json:"sender"`
	AssetId              crypto.Digest `json:"assetId"`
	MinSponsoredAssetFee uint64        `json:"minSponsoredAssetFee"`
	Fee                  uint64        `json:"fee"`
	Version              uint8         `json:"version"`
}

// Sponsor provided asset
func (a *Assets) Sponsor(ctx context.Context, sponsorReq AssetsSponsorReq) (*proto.SponsorshipV1, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/assets/sponsor")
	if err != nil {
		return nil, nil, err
	}

	bts, err := json.Marshal(sponsorReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(proto.SponsorshipV1)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsTransferReq struct {
	Version    uint8            `json:"version"`
	AssetId    crypto.Digest    `json:"assetId"`
	Amount     uint64           `json:"amount"`
	FeeAssetId crypto.Digest    `json:"feeAssetId"`
	Fee        uint64           `json:"fee"`
	Sender     proto.Address    `json:"sender"`
	Attachment proto.Attachment `json:"attachment"`
	Recipient  proto.Address    `json:"recipient"`
	Timestamp  uint64           `json:"timestamp"`
}

// Transfer asset to new address
func (a *Assets) Transfer(ctx context.Context, transferReq AssetsTransferReq) (*proto.TransferV2, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/assets/transfer")
	if err != nil {
		return nil, nil, err
	}

	bts, err := json.Marshal(transferReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(proto.TransferV2)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AssetsBurnReq struct {
	Sender    proto.Address `json:"sender"`
	AssetId   crypto.Digest `json:"assetId"`
	Quantity  uint64        `json:"quantity"`
	Fee       uint64        `json:"fee"`
	Timestamp uint64        `json:"timestamp"`
}

// Burn some of your assets
func (a *Assets) Burn(ctx context.Context, burnReq AssetsBurnReq) (*proto.BurnV1, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	bts, err := json.Marshal(burnReq)
	if err != nil {
		return nil, nil, err
	}

	url, err := joinUrl(a.options.BaseUrl, "/assets/burn")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(proto.BurnV1)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
