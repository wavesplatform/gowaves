package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Consensus struct {
	options Options
}

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
func (a *Consensus) GenerationSignature(ctx context.Context, blockID string) (string, *Response, error) {
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
