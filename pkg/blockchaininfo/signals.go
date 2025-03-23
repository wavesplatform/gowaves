package blockchaininfo

import (
	"strings"

	"go.uber.org/zap"

	"github.com/nats-io/nats.go"
)

const (
	TokenID       = "tokenId"
	AllMiners     = "allMiners"
	LastChainID   = "lastChainId"
	MinerReward   = "minerReward"
	Chain0Height  = "chain0Height"
	ThisEpochData = "thisEpochData"
)

func ConstantContractKeys() []string {
	return []string{
		TokenID,
		AllMiners,
		LastChainID,
		MinerReward,
		Chain0Height,
		ThisEpochData,
	}
}

func SendRestartSignal(nc *nats.Conn) (*nats.Msg, error) {
	message := []byte(RequestRestartSubTopic)
	msg, err := nc.Request(L2RequestsTopic, message, ConnectionsTimeoutDefault)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func SendConstantKeys(nc *nats.Conn) error {
	_, subErr := nc.Subscribe(ConstantKeys, func(request *nats.Msg) {
		constantKeys := strings.Join(ConstantContractKeys(), ",")
		err := request.Respond([]byte(constantKeys))
		if err != nil {
			zap.S().Errorf("failed to respond to a restart signal, %v", err)
			return
		}
	})
	return subErr
}
