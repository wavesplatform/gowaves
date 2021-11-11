package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Alias struct {
	options Options
}

func NewAlias(options Options) *Alias {
	return &Alias{
		options: options,
	}
}

func (a *Alias) Get(ctx context.Context, alias string) (proto.WavesAddress, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/alias/by-alias/%s", alias))
	if err != nil {
		return proto.WavesAddress{}, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return proto.WavesAddress{}, nil, err
	}

	out := struct {
		Address proto.WavesAddress `json:"address"`
	}{}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return proto.WavesAddress{}, response, err
	}

	return out.Address, response, nil
}

func (a *Alias) GetByAddress(ctx context.Context, address proto.WavesAddress) ([]*proto.Alias, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/alias/by-address/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*proto.Alias
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AliasCreateReq struct {
	Sender    proto.WavesAddress `json:"sender"`
	Alias     string             `json:"alias"`
	Fee       uint64             `json:"fee"`
	Timestamp uint64             `json:"timestamp,omitempty"`
}

type CreateAliasWithSig struct {
	Type      proto.TransactionType `json:"type"`
	Version   byte                  `json:"version,omitempty"`
	ID        *crypto.Digest        `json:"id,omitempty"`
	Signature *crypto.Signature     `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey      `json:"senderPublicKey"`
	Alias     string                `json:"alias"`
	Fee       uint64                `json:"fee"`
	Timestamp uint64                `json:"timestamp,omitempty"`
}

func (a *Alias) Create(ctx context.Context, createReq AliasCreateReq) (*CreateAliasWithSig, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/alias/create")
	if err != nil {
		return nil, nil, err
	}

	bts, err := json.Marshal(createReq)
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

	out := new(CreateAliasWithSig)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type AliasBroadcastReq struct {
	SenderPublicKey crypto.PublicKey `json:"senderPublicKey"`
	Fee             uint64           `json:"fee"`
	Timestamp       uint64           `json:"timestamp"`
	Signature       crypto.Signature `json:"signature"`
	Alias           string           `json:"alias"`
}

func (a *Alias) Broadcast(ctx context.Context, broadcastReq AliasBroadcastReq) (*CreateAliasWithSig, *Response, error) {
	bts, err := json.Marshal(broadcastReq)
	if err != nil {
		return nil, nil, err
	}

	url, err := joinUrl(a.options.BaseUrl, "/alias/broadcast/create")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST", url.String(),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	out := new(CreateAliasWithSig)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
