package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Blocks struct {
	options Options
}

func NewBlocks(options Options) *Blocks {
	return &Blocks{
		options: options,
	}
}

type BlocksHeight struct {
	Height uint64 `json:"height"`
}

func (a *Blocks) Height(ctx context.Context) (*BlocksHeight, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blocks/height")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	out := new(BlocksHeight)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}
	return out, response, nil
}

func (a *Blocks) HeightBySignature(ctx context.Context, signature string) (*BlocksHeight, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/height/%s", signature))
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(BlocksHeight)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type Headers struct {
	Version          uint64           `json:"version"`
	Timestamp        uint64           `json:"timestamp"`
	Reference        crypto.Signature `json:"reference"`
	NxtConsensus     NxtConsensus     `json:"nxt-consensus"`
	Features         []uint64         `json:"features"`
	Generator        proto.Address    `json:"generator"`
	Signature        crypto.Signature `json:"signature"`
	Blocksize        uint64           `json:"blocksize"`
	TransactionCount uint64           `json:"transactionCount"`
	Height           uint64           `json:"height"`
}

type NxtConsensus struct {
	BaseTarget          uint64 `json:"base-target"`
	GenerationSignature string `json:"generation-signature"`
}

func (a *Blocks) HeadersAt(ctx context.Context, height uint64) (*Headers, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/headers/at/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Headers)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) HeadersLast(ctx context.Context) (*Headers, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blocks/headers/last")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Headers)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) HeadersSeq(ctx context.Context, from uint64, to uint64) ([]*Headers, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/headers/seq/%d/%d", from, to))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(
		"GET",
		url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*Headers
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type Block struct {
	Version          uint64       `json:"version"`
	Timestamp        uint64       `json:"timestamp"`
	Reference        string       `json:"reference"`
	NxtConsensus     NxtConsensus `json:"nxt-consensus"`
	Features         []uint64     `json:"features"`
	Generator        string       `json:"generator"`
	Signature        string       `json:"signature"`
	Blocksize        uint64       `json:"blocksize"`
	TransactionCount uint64       `json:"transactionCount"`
	Fee              uint64       `json:"fee"`
	Height           uint64       `json:"height"`
}
