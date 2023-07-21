package api

import (
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type RewardInfo struct {
	Height              proto.Height      `json:"height"`
	TotalWavesAmount    uint64            `json:"totalWavesAmount"`
	CurrentReward       uint64            `json:"currentReward"`
	MinIncrement        uint64            `json:"minIncrement"`
	Term                uint64            `json:"term"`
	NextCheck           uint64            `json:"nextCheck"`
	VotingIntervalStart uint64            `json:"votingIntervalStart"`
	VotingInterval      uint64            `json:"votingInterval"`
	VotingThreshold     uint64            `json:"votingThreshold"`
	Votes               state.RewardVotes `json:"votes"`
	DaoAddress          string            `json:"daoAddress"`
	XtnBuybackAddress   string            `json:"xtnBuybackAddress"`
}

func (a *NodeApi) rewardAtHeight(height proto.Height) (RewardInfo, error) {
	blockRewardsActiivated, err := a.state.IsActiveAtHeight(int16(settings.BlockReward), height)
	if err != nil {
		return RewardInfo{}, err
	}
	if !blockRewardsActiivated || height == 1 {
		return RewardInfo{}, errors.Wrap(err, "Block reward feature is not activated yet")
	}

	cappedRewardsActivated, err := a.state.IsActiveAtHeight(int16(settings.CappedRewards), height)
	if err != nil {
		return RewardInfo{}, err
	}
	set, err := a.state.BlockchainSettings()
	if err != nil {
		return RewardInfo{}, err
	}
	blockRewardHeight, err := a.state.ActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return RewardInfo{}, err
	}

	var diff = height - blockRewardHeight + 1
	var modifiedTerm uint64
	if cappedRewardsActivated {
		modifiedTerm = set.BlockRewardTermAfter20
	} else {
		modifiedTerm = set.BlockRewardTerm
	}
	var mul = uint64(math.Ceil(float64(diff) / float64(modifiedTerm)))
	nextCheck := blockRewardHeight + mul*modifiedTerm - 1

	var term uint64
	if cappedRewardsActivated {
		term = set.BlockRewardTermAfter20
	} else {
		term = set.BlockRewardTerm
	}

	reward, err := a.state.RewardAtHeight(height)
	if err != nil {
		return RewardInfo{}, err
	}

	blockRewardDistributionActivated, err := a.state.IsActiveAtHeight(int16(settings.BlockRewardDistribution), height)
	if err != nil {
		return RewardInfo{}, err
	}
	daoAddress := ""
	xtnBuybackAddress := ""
	if blockRewardDistributionActivated && len(set.RewardAddresses) > 0 {
		if len(set.RewardAddresses) >= 1 {
			daoAddress = set.RewardAddresses[0].String()
		}
		if len(set.RewardAddresses) >= 2 {
			xtnBuybackAddress = set.RewardAddresses[1].String()
		}
	}

	votes, err := a.state.RewardVotes()
	if err != nil {
		return RewardInfo{}, err
	}
	resp := RewardInfo{
		Height:              height,
		TotalWavesAmount:    0,
		CurrentReward:       reward,
		MinIncrement:        set.BlockRewardIncrement,
		Term:                term,
		NextCheck:           nextCheck,
		VotingIntervalStart: nextCheck - set.BlockRewardVotingPeriod + 1,
		VotingInterval:      set.BlockRewardVotingPeriod,
		VotingThreshold:     set.BlockRewardVotingPeriod/2 + 1,
		Votes:               votes,
		DaoAddress:          daoAddress,
		XtnBuybackAddress:   xtnBuybackAddress,
	}
	return resp, nil
}

func (a *NodeApi) blockchainRewards(w http.ResponseWriter, _ *http.Request) error {
	h, err := a.state.Height()
	if err != nil {
		return err
	}
	res, err := a.rewardAtHeight(h)
	if err != nil {
		return err
	}
	if err = trySendJson(w, res); err != nil {
		return errors.Wrap(err, "BlockchainRewards")
	}
	return nil
}

func (a *NodeApi) blockchainRewardsAtHeight(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return &BadRequestError{err}
	}
	res, err := a.rewardAtHeight(height)
	if err != nil {
		return err
	}
	if err = trySendJson(w, res); err != nil {
		return errors.Wrap(err, "BlockchainRewards")
	}
	return nil
}
