package client

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/api"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net/http"
)

type Blockchain struct {
	options Options
}

func NewBlockchain(options Options) *Blockchain {
	return &Blockchain{
		options: options,
	}
}

// Rewards returns info about rewards on top of blockchain.
func (a *Blockchain) Rewards(ctx context.Context) (*api.RewardInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blockchain/rewards")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(api.RewardInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// RewardsAtHeight returns info about rewards at height.
func (a *Blockchain) RewardsAtHeight(ctx context.Context, height proto.Height) (*api.RewardInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blockchain/rewards/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(api.RewardInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
