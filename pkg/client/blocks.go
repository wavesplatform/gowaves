package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

// HeightBySignature returns block height by the given id in base58 encoding. Does the same as HeightByID.
// Deprecated: use HeightByID.
func (a *Blocks) HeightBySignature(ctx context.Context, id string) (*BlocksHeight, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/height/%s", id))
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

// HeightByID returns block height by the given id.
func (a *Blocks) HeightByID(ctx context.Context, id proto.BlockID) (*BlocksHeight, *Response, error) {
	return a.HeightBySignature(ctx, id.String())
}

type Headers struct {
	Version            uint64             `json:"version"`
	Timestamp          uint64             `json:"timestamp"`
	Reference          proto.BlockID      `json:"reference"`
	NxtConsensus       NxtConsensus       `json:"nxt-consensus"`
	TransactionsRoot   string             `json:"transactionsRoot"`
	Features           []uint64           `json:"features"`
	DesiredReward      int64              `json:"desiredReward"`
	Generator          proto.WavesAddress `json:"generator"`
	GeneratorPublicKey string             `json:"generatorPublicKey"`
	Signature          crypto.Signature   `json:"signature"`
	Blocksize          uint64             `json:"blocksize"`
	TransactionCount   uint64             `json:"transactionCount"`
	Height             uint64             `json:"height"`
	TotalFee           int64              `json:"totalFee"`
	Reward             int64              `json:"reward"`
	VRF                string             `json:"VRF"`
	ID                 proto.BlockID      `json:"id"`
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

// At gets block at specified height.
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
	return out, response, nil
}

func (a *Blocks) Delay(ctx context.Context, id proto.BlockID, blockNum uint64) (uint64, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blocks/delay/%s/%d", id.String(), blockNum))
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

func (a *Blocks) Address(ctx context.Context, addr proto.WavesAddress, from, to uint64) ([]*Block, *Response, error) {
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
