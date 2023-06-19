package state_fsm

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/state_fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	askPeersInterval = 5 * time.Minute
)

// Set args types for events
// First arg is Async - return value of event handler
var (
	eventsArgsTypes = map[stateless.Trigger][]reflect.Type{
		ConnectedPeerEvent:     {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem()},
		DisconnectedPeerEvent:  {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem()},
		ConnectedBestPeerEvent: {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem()},
		StopMiningEvent:        {reflect.TypeOf(&Async{})},
		ScoreEvent:             {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.Score{})},
		BlockEvent:             {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.Block{})},
		MinedBlockEvent:        {reflect.TypeOf(&Async{}), reflect.TypeOf(&proto.Block{}), reflect.TypeOf(proto.MiningLimits{}), reflect.TypeOf(proto.KeyPair{}), reflect.TypeOf([]byte{})},
		BlockIDsEvent:          {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf([]proto.BlockID{})},
		TaskEvent:              {reflect.TypeOf(&Async{}), reflect.TypeOf(tasks.AsyncTask{})},
		MicroBlockEvent:        {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.MicroBlock{})},
		MicroBlockInvEvent:     {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.MicroBlockInv{})},
		TransactionEvent:       {reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf((*proto.Transaction)(nil)).Elem()},
		HaltEvent:              {reflect.TypeOf(&Async{})},
	}
)

func syncWithNewPeer(state State, baseInfo BaseInfo, p peer.Peer) (State, Async, error) {
	lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(baseInfo.storage)
	if err != nil {
		return state, nil, err
	}
	internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, baseInfo.scheme), lastSignatures)
	c := conf{
		peerSyncWith: p,
		timeout:      30 * time.Second,
	}
	zap.S().Debugf("[%s] Starting synchronization with peer '%s'", state.String(), p.ID())
	return &SyncState{
		baseInfo: baseInfo,
		conf:     c.Now(baseInfo.tm),
		internal: internal,
	}, nil, nil
}

func tryBroadcastTransaction(fsm State, baseInfo BaseInfo, p peer.Peer, t proto.Transaction) (_ State, _ Async, err error) {
	defer func() {
		if err != nil {
			err = fsm.Errorf(proto.NewInfoMsg(err))
		}
	}()
	if zap.S().Level() <= zap.DebugLevel {
		defer func() {
			if genIDErr := t.GenerateID(baseInfo.scheme); genIDErr != nil {
				zap.S().Errorf("[%s] Failed to generate ID for transaction: %v", fsm.String(), genIDErr)
				return
			}
			txIDBytes, getIDErr := t.GetID(baseInfo.scheme)
			if getIDErr != nil {
				zap.S().Errorf("[%s] Failed to get ID for transaction: %v", fsm.String(), getIDErr)
				return
			}
			txID := base58.Encode(txIDBytes)
			if err != nil {
				err = errors.Wrapf(err, "Failed to broadcast transaction %q", txID)
			} else {
				zap.S().Debugf("[%s] Transaction %q broadcasted successfuly", fsm.String(), txID)
			}
		}()
	}
	if _, err := t.Validate(baseInfo.scheme); err != nil {
		err = errors.Wrap(err, "Failed to validate transaction")
		if p != nil {
			baseInfo.peers.AddToBlackList(p, time.Now(), err.Error())
		}
		return fsm, nil, err
	}

	if err := baseInfo.utx.Add(t); err != nil {
		err = errors.Wrap(err, "Failed to add transaction to utx")
		return fsm, nil, err
	}
	baseInfo.BroadcastTransaction(t, p)
	return fsm, nil, nil
}

func fsmErrorf(state State, err error) error {
	switch e := err.(type) {
	case *proto.InfoMsg:
		return proto.NewInfoMsg(errors.Errorf("[%s] %s", state.String(), e.Error()))
	default:
		return errors.Errorf("[%s] %s", state.String(), e.Error())
	}
}

func createPermitDynamicCallback(event stateless.Trigger, state *StateData, actionFunc func(...interface{}) (State, Async, error)) stateless.DestinationSelectorFunc {
	return func(ctx context.Context, args ...interface{}) (stateless.State, error) {
		validateEventArgs(event, args...)
		newState, asyncNew, err := actionFunc(args[1:]...)
		async := args[0].(*Async)
		*async = asyncNew
		state.State = newState
		state.Name = newState.String()
		return newState.String(), err
	}
}

func convertToInterface[T any](arg interface{}) T {
	var res T
	if arg == nil {
		return res
	}
	return arg.(T)
}

func isCanBeNil(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func, reflect.Ptr:
		return true
	default:
		return false
	}
}

func validateEventArgs(event stateless.Trigger, args ...interface{}) {
	if len(args) != len(eventsArgsTypes[event]) {
		panic(fmt.Sprintf("Invalid number of arguments for event %q: expected %d, got %d", event, len(eventsArgsTypes[event]), len(args)))
	}

	for i, arg := range args {
		want := eventsArgsTypes[event][i]
		tp := reflect.TypeOf(arg)
		if tp == nil && isCanBeNil(want) {
			continue
		}
		if !tp.ConvertibleTo(want) {
			panic(fmt.Sprintf("The argument in position '%d' for event %s is of type '%v' but must be convertible to '%v'.", i, event, tp, want))
		}
	}
}
