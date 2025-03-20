package blockchaininfo

import "github.com/nats-io/nats.go"

func SendRestartSignal(nc *nats.Conn) (*nats.Msg, error) {
	message := []byte(RequestRestartSubTopic)
	msg, err := nc.Request(L2RequestsTopic, message, ConnectionsTimeoutDefault)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
