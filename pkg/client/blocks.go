package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
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

	req, err := http.NewRequest("GET", url.String(), nil)
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

	req, err := http.NewRequest("GET", url.String(), nil)
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

	req, err := http.NewRequest("GET", url.String(), nil)
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
	Headers
	Fee          uint64            `json:"fee"`
	Transactions TransactionsField `json:"transactions"`
}

type TransactionsField []proto.Transaction

func (b *TransactionsField) UnmarshalJSON(data []byte) error {
	var tt []*proto.TransactionTypeVersion
	err := json.Unmarshal(data, &tt)
	if err != nil {
		return errors.Wrap(err, "TransactionTypeVersion unmarshal")
	}

	transactions := make([]proto.Transaction, len(tt))
	for i, row := range tt {
		realType, err := proto.GuessTransactionType(row)
		if err != nil {
			return err
		}
		transactions[i] = realType
	}

	err = json.Unmarshal(data, &transactions)
	if err != nil {
		return errors.Wrap(err, "transactions list unmarshal")
	}
	*b = transactions

	return nil
}

// Get block at specified height
func (a *Blocks) At(ctx context.Context, height uint64) (*Block, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/at/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Block)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	if err != nil {
		return nil, response, &ParseError{Err: err}
	}
	return out, response, nil
}

func (a *Blocks) Delay(ctx context.Context, signature crypto.Signature, blockNum uint64) (uint64, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/delay/%s/%d", signature.String(), blockNum))
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return 0, nil, err
	}

	out := struct {
		Delay uint64 `json:"delay"`
	}{}

	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return 0, response, err
	}

	return out.Delay, response, nil
}

// Get block by its signature
func (a *Blocks) Signature(ctx context.Context, signature crypto.Signature) (*Block, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/signature/%s", signature.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Block)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) Child(ctx context.Context, signature crypto.Signature) (*Block, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/child/%s", signature.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Block)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) First(ctx context.Context) (*Block, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blocks/first")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Block)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) Last(ctx context.Context) (*Block, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blocks/last")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(Block)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) Seq(ctx context.Context, from, to uint64) ([]*Block, *Response, error) {
	if from > to {
		return nil, nil, errors.New("invalid arguments")
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/seq/%d/%d", from, to))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*Block
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

func (a *Blocks) Address(ctx context.Context, addr proto.Address, from, to uint64) ([]*Block, *Response, error) {
	if from > to {
		return nil, nil, errors.New("invalid arguments")
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/address/%s/%d/%d", addr.String(), from, to))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*Block
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
