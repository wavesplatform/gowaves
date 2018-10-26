package client

import (
	"context"
	"fmt"
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
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/blocks/height", a.options.BaseUrl), nil)
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
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/blocks/height/%s", a.options.BaseUrl, signature),
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
	Version          uint64       `json:"version"`
	Timestamp        uint64       `json:"timestamp"`
	Reference        string       `json:"reference"`
	NxtConsensus     NxtConsensus `json:"nxt-consensus"`
	Features         []uint64     `json:"features"`
	Generator        string       `json:"generator"`
	Signature        string       `json:"signature"`
	Blocksize        uint64       `json:"blocksize"`
	TransactionCount uint64       `json:"transactionCount"`
	Height           uint64       `json:"height"`
}

type NxtConsensus struct {
	BaseTarget          uint64 `json:"base-target"`
	GenerationSignature string `json:"generation-signature"`
}

func (a *Blocks) HeadersAt(ctx context.Context, height uint64) (*Headers, *Response, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/blocks/headers/at/%d", a.options.BaseUrl, height),
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
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/blocks/headers/last", a.options.BaseUrl),
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
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/blocks/headers/seq/%d/%d", a.options.BaseUrl, from, to),
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
