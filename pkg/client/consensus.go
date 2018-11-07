package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Consensus struct {
	options Options
}

// creates new consensus api section
func NewConsensus(options Options) *Consensus {
	return &Consensus{
		options: options,
	}
}

type ConsensusGeneratingBalance struct {
	Address proto.Address `json:"address"`
	Balance uint64        `json:"balance"`
}

// Account's generating balance(the same as balance atm)
func (a Consensus) GeneratingBalance(ctx context.Context, address proto.Address) (*ConsensusGeneratingBalance, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/generatingbalance/%s", a.options.BaseUrl, address.String()),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(ConsensusGeneratingBalance)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Generation signature of a block with specified id
func (a *Consensus) GenerationSignatureByBlock(ctx context.Context, blockID string) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/generationsignature/%s", a.options.BaseUrl, blockID),
		nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["generationSignature"], response, nil
}

// Base target of a block with specified id
func (a *Consensus) BaseTargetByBlock(ctx context.Context, blockID string) (uint64, *Response, error) {
	if a.options.ApiKey == "" {
		return 0, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/basetarget/%s", a.options.BaseUrl, blockID),
		nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]uint64)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return 0, response, err
	}

	return out["baseTarget"], response, nil
}

type ConsensusBaseTarget struct {
	BaseTarget uint64 `json:"baseTarget"`
	Score      string `json:"score"`
}

// Base target of a last block
func (a *Consensus) BaseTarget(ctx context.Context) (*ConsensusBaseTarget, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/basetarget", a.options.BaseUrl),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(ConsensusBaseTarget)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Shows which consensus algo being using
func (a *Consensus) Algo(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/algo", a.options.BaseUrl),
		nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["consensusAlgo"], response, nil
}

// Generation signature of a last block
func (a *Consensus) GenerationSignature(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/consensus/generationsignature", a.options.BaseUrl),
		nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["generationSignature"], response, nil
}
