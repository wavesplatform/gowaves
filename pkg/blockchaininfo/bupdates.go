package blockchaininfo

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BlockchainUpdatesExtension struct {
	l2ContractAddress        proto.WavesAddress
	blockchainExtensionState *BUpdatesExtensionState
	lock                     sync.Mutex
	makeExtensionReadyFunc   func()
	obsolescencePeriod       time.Duration
	ntpTime                  types.Time
}

func NewBlockchainUpdatesExtension(
	l2ContractAddress proto.WavesAddress,
	blockchainExtensionState *BUpdatesExtensionState,
	makeExtensionReadyFunc func(),
	obsolescencePeriod time.Duration,
	ntpTime types.Time,
) *BlockchainUpdatesExtension {
	return &BlockchainUpdatesExtension{
		l2ContractAddress:        l2ContractAddress,
		blockchainExtensionState: blockchainExtensionState,
		makeExtensionReadyFunc:   makeExtensionReadyFunc,
		obsolescencePeriod:       obsolescencePeriod,
		ntpTime:                  ntpTime,
	}
}

func (e *BlockchainUpdatesExtension) L2ContractAddress() proto.WavesAddress {
	return e.l2ContractAddress
}

func (e *BlockchainUpdatesExtension) MarkExtensionReady() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.makeExtensionReadyFunc()
}

func (e *BlockchainUpdatesExtension) ClearPreviousState() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.blockchainExtensionState.PreviousState = nil
}

func (e *BlockchainUpdatesExtension) RunBlockchainUpdatesPublisher(ctx context.Context,
	scheme proto.Scheme,
	updatesChannel <-chan proto.BUpdatesInfo) error {
	opts := &server.Options{
		MaxPayload: natsMaxPayloadSize,
		Host:       hostDefault,
		Port:       portDefault,
		NoSigs:     true,
	}
	s, err := server.NewServer(opts)
	if err != nil {
		return errors.Wrap(err, "failed to create NATS server")
	}
	go s.Start()
	defer func() {
		s.Shutdown()
		s.WaitForShutdown()
	}()
	if !s.ReadyForConnections(NatsConnectionsTimeoutDefault) {
		return errors.New("NATS server is not ready for connections")
	}

	slog.Info("NATS Server is running on port", slog.Any("port", portDefault))

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to NATS server")
	}
	defer nc.Close()

	reqErr := e.requestConstantKeys(nc) // Blocking.
	if reqErr != nil {
		return errors.Wrap(reqErr, "failed to request constant keys from the client")
	}
	e.MarkExtensionReady()
	updatesPublisher := UpdatesPublisher{l2ContractAddress: e.l2ContractAddress.String(),
		obsolescencePeriod: e.obsolescencePeriod, ntpTime: e.ntpTime}
	// Publish the first 100 history entries for the rollback functionality.
	publishHistoryBlocks(ctx, e, scheme, nc, updatesPublisher)

	receiverErr := runReceiver(nc, e)
	if receiverErr != nil {
		return receiverErr
	}
	runPublisher(ctx, e, scheme, nc, updatesPublisher, updatesChannel)
	return nil
}

func (e *BlockchainUpdatesExtension) requestConstantKeys(nc *nats.Conn) error {
	sub, err := nc.SubscribeSync(ConstantKeys)
	if err != nil {
		return err
	}
	defer func(sub *nats.Subscription) {
		unSubErr := sub.Unsubscribe()
		if unSubErr != nil {
			slog.Error("failed to unsubscribe from constant keys topic", logging.Error(unSubErr))
		}
	}(sub)
	// Block until first message or timeout.
	msg, err := sub.NextMsg(NatsConnectionsTimeoutDefault)
	if err != nil {
		return errors.Wrap(err, "timeout or error waiting for constant keys")
	}
	constantKeys, err := DeserializeConstantKeys(msg.Data)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize constant keys")
	}
	e.blockchainExtensionState.constantContractKeys = constantKeys
	return nil
}
