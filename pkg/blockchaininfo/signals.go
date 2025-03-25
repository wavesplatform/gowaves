package blockchaininfo

import (
	"strings"

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
	constantKeys := strings.Join(ConstantContractKeys(), ",")
	return nc.Publish(ConstantKeys, []byte(constantKeys))
}
