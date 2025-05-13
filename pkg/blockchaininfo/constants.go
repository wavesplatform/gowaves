package blockchaininfo

import "github.com/nats-io/nats-server/v2/server"

const (
	BlockUpdates    = "block_topic"
	ContractUpdates = "contract_topic"
	ConstantKeys    = "constant_keys"
)

const (
	StartPaging = iota
	EndPaging
	NoPaging
)

const NatsConnectionsTimeoutDefault = 10 * server.AUTH_TIMEOUT
const L2RequestsTopic = "l2_requests_topic"
const RequestRestartSubTopic = "restart"
