package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type RewardInfo struct {
	Height              proto.Height        `json:"height"`
	TotalWavesAmount    uint64              `json:"totalWavesAmount"`
	CurrentReward       uint64              `json:"currentReward"`
	MinIncrement        uint64              `json:"minIncrement"`
	Term                uint64              `json:"term"`
	NextCheck           uint64              `json:"nextCheck"`
	VotingIntervalStart uint64              `json:"votingIntervalStart"`
	VotingInterval      uint64              `json:"votingInterval"`
	VotingThreshold     uint64              `json:"votingThreshold"`
	Votes               proto.RewardVotes   `json:"votes"`
	DAOAddress          *proto.WavesAddress `json:"daoAddress,omitempty"`
	XTNBuybackAddress   *proto.WavesAddress `json:"xtnBuybackAddress,omitempty"`
}

type Blockchain struct {
	options Options
}

func NewBlockchain(options Options) *Blockchain {
	return &Blockchain{
		options: options,
	}
}

// Rewards returns info about rewards on top of blockchain.
func (a *Blockchain) Rewards(ctx context.Context) (*RewardInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, "/blockchain/rewards")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(RewardInfo)
	response, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// RewardsAtHeight returns info about rewards at height.
func (a *Blockchain) RewardsAtHeight(ctx context.Context, height proto.Height) (*RewardInfo, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/blockchain/rewards/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := new(RewardInfo)
	response, err := doHTTP(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}
