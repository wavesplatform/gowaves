package fsm

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
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	askPeersInterval   = 5 * time.Minute
	defaultSyncTimeout = 30 * time.Second
)

// Set args types for events.
// First arg is Async - return value of event handler.
func eventArgsTypes(event stateless.Trigger) []reflect.Type {
	switch event {
	case StartMiningEvent, StopSyncEvent, StopMiningEvent, HaltEvent:
		return []reflect.Type{reflect.TypeOf(&Async{})}
	case TaskEvent:
		return []reflect.Type{reflect.TypeOf(&Async{}), reflect.TypeOf(tasks.AsyncTask{})}
	case ChangeSyncPeerEvent:
		return []reflect.Type{reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem()}
	case ScoreEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.Score{}),
		}
	case BlockEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.Block{}),
		}
	case MinedBlockEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf(&proto.Block{}), reflect.TypeOf(proto.MiningLimits{}),
			reflect.TypeOf(proto.KeyPair{}), reflect.TypeOf([]byte{}),
		}
	case BlockIDsEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf([]proto.BlockID{}),
		}
	case MicroBlockEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.MicroBlock{}),
		}
	case MicroBlockInvEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(), reflect.TypeOf(&proto.MicroBlockInv{}),
		}
	case TransactionEvent:
		return []reflect.Type{
			reflect.TypeOf(&Async{}), reflect.TypeOf((*peer.Peer)(nil)).Elem(),
			reflect.TypeOf((*proto.Transaction)(nil)).Elem(),
		}
	default:
		return nil
	}
}

func syncWithNewPeer(state State, baseInfo BaseInfo, p peer.Peer) (State, Async, error) {
	// TODO: LastBlockIDs can be a function.
	lastSignatures, err := signatures.LastSignaturesImpl{}.LastBlockIDs(baseInfo.storage)
	if err != nil {
		return state, nil, err
	}
	internal := sync_internal.InternalFromLastSignatures(extension.NewPeerExtension(p, baseInfo.scheme), lastSignatures)
	c := conf{
		peerSyncWith: p,
		timeout:      defaultSyncTimeout,
	}
	zap.S().Named(logging.FSMNamespace).Debugf("[%s] Starting synchronization with peer '%s'",
		state.String(), p.ID())
	baseInfo.syncPeer.SetPeer(p)
	return &SyncState{
		baseInfo: baseInfo,
		conf:     c.Now(baseInfo.tm),
		internal: internal,
	}, nil, nil
}

func tryBroadcastTransaction(
	fsm State, baseInfo BaseInfo, p peer.Peer, t proto.Transaction,
) (_ State, _ Async, err error) {
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
				err = errors.Wrapf(err, "failed to broadcast transaction %q", txID)
			} else {
				zap.S().Named(logging.FSMNamespace).Debugf("[%s] Transaction %q broadcasted successfuly",
					fsm.String(), txID)
			}
		}()
	}
	if _, err = t.Validate(baseInfo.scheme); err != nil {
		err = errors.Wrap(err, "failed to validate transaction")
		if p != nil {
			baseInfo.peers.AddToBlackList(p, time.Now(), err.Error())
		}
		return fsm, nil, err
	}

	if err = baseInfo.utx.Add(t); err != nil {
		err = errors.Wrap(err, "failed to add transaction to utx")
		return fsm, nil, err
	}
	baseInfo.BroadcastTransaction(t, p)
	return fsm, nil, nil
}

func fsmErrorf(state State, err error) error {
	infoMsg := &proto.InfoMsg{}
	if errors.As(err, &infoMsg) {
		return proto.NewInfoMsg(errors.Errorf("[%s] %s", state.String(), err.Error()))
	}
	return errors.Errorf("[%s] %s", state.String(), err.Error())
}

func createPermitDynamicCallback(
	event stateless.Trigger, state *StateData, actionFunc func(...interface{}) (State, Async, error),
) stateless.DestinationSelectorFunc {
	return func(_ context.Context, args ...interface{}) (stateless.State, error) {
		validateEventArgs(event, args...)
		newState, asyncNew, err := actionFunc(args[1:]...)
		async, ok := args[0].(*Async)
		if !ok {
			return nil, errors.Errorf("unexpected type '%T', expected '*Async'", args[0])
		}
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
	return t.Kind() == reflect.Map || t.Kind() == reflect.Slice || t.Kind() == reflect.Interface ||
		t.Kind() == reflect.Chan || t.Kind() == reflect.Func || t.Kind() == reflect.Ptr
}

func validateEventArgs(event stateless.Trigger, args ...interface{}) {
	if len(args) != len(eventArgsTypes(event)) {
		panic(fmt.Sprintf("Invalid number of arguments for event %q: expected %d, got %d", event,
			len(eventArgsTypes(event)), len(args)),
		)
	}

	want := eventArgsTypes(event)
	for i, arg := range args {
		tp := reflect.TypeOf(arg)
		if tp == nil && isCanBeNil(want[i]) {
			continue
		}
		if !tp.ConvertibleTo(want[i]) {
			panic(fmt.Sprintf(
				"The argument in position '%d' for event %s is of type '%v' but must be convertible to '%v'.",
				i, event, tp, want),
			)
		}
	}
}
