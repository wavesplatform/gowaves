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

func (a *Alias) Get(ctx context.Context, alias string) (proto.Address, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/alias/by-alias/%s", a.options.BaseUrl, alias),
		nil)
	if err != nil {
		return proto.Address{}, nil, err
	}

	out := struct {
		Address proto.Address `json:"address"`
	}{}
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return proto.Address{}, response, err
	}

	return out.Address, response, nil
}

func (a *Alias) GetByAddress(ctx context.Context, address proto.Address) ([]*proto.Alias, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/alias/by-address/%s", a.options.BaseUrl, address.String()),
		nil)
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
	Sender    proto.Address `json:"sender"`
	Alias     string        `json:"alias"`
	Fee       uint64        `json:"fee"`
	Timestamp uint64        `json:"timestamp,omitempty"`
}

type CreateAliasV1 struct {
	Type      proto.TransactionType `json:"type"`
	Version   byte                  `json:"version,omitempty"`
	ID        *crypto.Digest        `json:"id,omitempty"`
	Signature *crypto.Signature     `json:"signature,omitempty"`
	SenderPK  crypto.PublicKey      `json:"senderPublicKey"`
	Alias     string                `json:"alias"`
	Fee       uint64                `json:"fee"`
	Timestamp uint64                `json:"timestamp,omitempty"`
}

func (a *Alias) Create(ctx context.Context, createReq AliasCreateReq) (*CreateAliasV1, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	bts, err := json.Marshal(createReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/alias/create", a.options.BaseUrl),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(CreateAliasV1)
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

func (a *Alias) Broadcast(ctx context.Context, broadcastReq AliasBroadcastReq) (*CreateAliasV1, *Response, error) {
	bts, err := json.Marshal(broadcastReq)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/alias/broadcast/create", a.options.BaseUrl),
		bytes.NewReader(bts))
	if err != nil {
		return nil, nil, err
	}

	out := new(CreateAliasV1)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
