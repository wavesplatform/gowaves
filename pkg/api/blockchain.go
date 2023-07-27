package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type rewardInfoResponse struct {
	Height              proto.Height      `json:"height"`
	TotalWavesAmount    uint64            `json:"totalWavesAmount"`
	CurrentReward       uint64            `json:"currentReward"`
	MinIncrement        uint64            `json:"minIncrement"`
	Term                uint64            `json:"term"`
	NextCheck           uint64            `json:"nextCheck"`
	VotingIntervalStart uint64            `json:"votingIntervalStart"`
	VotingInterval      uint64            `json:"votingInterval"`
	VotingThreshold     uint64            `json:"votingThreshold"`
	Votes               proto.RewardVotes `json:"votes"`
	DAOAddress          string            `json:"daoAddress,omitempty"`
	XTNBuybackAddress   string            `json:"xtnBuybackAddress,omitempty"`
}

func (a *NodeApi) rewardAtHeight(height proto.Height) (rewardInfoResponse, error) {
	blockRewardsActivated, err := a.state.IsActiveAtHeight(int16(settings.BlockReward), height)
	if err != nil {
		return rewardInfoResponse{}, err
	}
	if !blockRewardsActivated || height == 1 {
		return rewardInfoResponse{}, errors.Wrap(err, "Block reward feature is not activated yet")
	}

	cappedRewardsActivated, err := a.state.IsActiveAtHeight(int16(settings.CappedRewards), height)
	if err != nil {
		return rewardInfoResponse{}, err
	}
	set, err := a.state.BlockchainSettings()
	if err != nil {
		return rewardInfoResponse{}, err
	}
	blockRewardHeight, err := a.state.ActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return rewardInfoResponse{}, err
	}

	nextCheck := set.NextRewardTerm(height, blockRewardHeight, cappedRewardsActivated)

	reward, err := a.state.RewardAtHeight(height)
	if err != nil {
		return rewardInfoResponse{}, err
	}

	blockRewardDistributionActivated, err := a.state.IsActiveAtHeight(int16(settings.BlockRewardDistribution), height)
	if err != nil {
		return rewardInfoResponse{}, err
	}
	xtnBuyBackCessation, err := a.state.IsActiveAtHeight(int16(settings.XTNBuyBackCessation), height)
	if err != nil {
		return rewardInfoResponse{}, err
	}

	var daoAddress string
	var xtnBuybackAddress string
	if blockRewardDistributionActivated && len(set.CurrentRewardAddresses(xtnBuyBackCessation)) > 0 {
		daoAddress = set.DAOAddress(xtnBuyBackCessation).String()
		xtnBuybackAddress = set.XTNBuybackAddress(xtnBuyBackCessation).String()
	}

	votes, err := a.state.RewardVotes()
	if err != nil {
		return rewardInfoResponse{}, err
	}
	totalAmount, err := a.state.TotalWavesAmount(height)
	if err != nil {
		return rewardInfoResponse{}, err
	}
	return rewardInfoResponse{
		Height:              height,
		TotalWavesAmount:    totalAmount,
		CurrentReward:       reward,
		MinIncrement:        set.BlockRewardIncrement,
		Term:                set.CurrentBlockRewardTerm(cappedRewardsActivated),
		NextCheck:           nextCheck - 1,
		VotingIntervalStart: nextCheck - set.BlockRewardVotingPeriod,
		VotingInterval:      set.BlockRewardVotingPeriod,
		VotingThreshold:     set.BlockRewardVotingThreshold(),
		Votes:               votes,
		DAOAddress:          daoAddress,
		XTNBuybackAddress:   xtnBuybackAddress,
	}, nil
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
