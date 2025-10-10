package fsm

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"

	"github.com/wavesplatform/gowaves/pkg/libs/signatures"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/sync_internal"
	"github.com/wavesplatform/gowaves/pkg/node/fsm/tasks"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer/extension"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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
		return []reflect.Type{reflect.TypeFor[*Async]()}
	case TaskEvent:
		return []reflect.Type{reflect.TypeFor[*Async](), reflect.TypeFor[tasks.AsyncTask]()}
	case ChangeSyncPeerEvent:
		return []reflect.Type{reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer]()}
	case ScoreEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[*proto.Score](),
		}
	case BlockEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[*proto.Block](),
		}
	case MinedBlockEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[*proto.Block](), reflect.TypeFor[proto.MiningLimits](),
			reflect.TypeFor[proto.KeyPair](), reflect.TypeFor[[]byte](),
		}
	case BlockIDsEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[[]proto.BlockID](),
		}
	case MicroBlockEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[*proto.MicroBlock](),
		}
	case MicroBlockInvEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[*proto.MicroBlockInv](),
		}
	case TransactionEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](),
			reflect.TypeFor[proto.Transaction](),
		}
	case BlockSnapshotEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[proto.BlockID](),
			reflect.TypeFor[proto.BlockSnapshot](),
		}
	case MicroBlockSnapshotEvent:
		return []reflect.Type{
			reflect.TypeFor[*Async](), reflect.TypeFor[peer.Peer](), reflect.TypeFor[proto.BlockID](),
			reflect.TypeFor[proto.BlockSnapshot](),
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
	internal := sync_internal.InternalFromLastSignatures(
		extension.NewPeerExtension(p, baseInfo.scheme, baseInfo.netLogger),
		lastSignatures,
		baseInfo.enableLightMode,
	)
	c := conf{
		peerSyncWith: p,
		timeout:      defaultSyncTimeout,
	}
	baseInfo.logger.Debug("Starting synchronization with peer", "state", state.String(), "peer", p.ID())
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
	if baseInfo.logger.Enabled(context.Background(), slog.LevelDebug) {
		defer func() {
			if genIDErr := t.GenerateID(baseInfo.scheme); genIDErr != nil {
				slog.Error("Failed to generate ID for transaction", slog.String("state", fsm.String()),
					logging.Error(genIDErr))
				return
			}
			txIDBytes, getIDErr := t.GetID(baseInfo.scheme)
			if getIDErr != nil {
				slog.Error("Failed to get ID for transaction", slog.String("state", fsm.String()),
					logging.Error(getIDErr))
				return
			}
			txID := base58.Encode(txIDBytes)
			if err != nil {
				err = errors.Wrapf(err, "failed to broadcast transaction %q", txID)
			} else {
				baseInfo.logger.Debug("Transaction broadcasted successfully", "state", fsm.String(), "txID", txID)
			}
		}()
	}
	lightNodeActivated, err := baseInfo.storage.IsActivated(int16(settings.LightNode))
	if err != nil {
		return fsm, nil, errors.Wrap(err, "failed to check if LightNode feature is activated")
	}
	params := proto.TransactionValidationParams{Scheme: baseInfo.scheme, CheckVersion: lightNodeActivated}
	if _, err = t.Validate(params); err != nil {
		err = errors.Wrap(err, "failed to validate transaction")
		if p != nil {
			baseInfo.peers.AddToBlackList(p, time.Now(), err.Error())
		}
		return fsm, nil, err
	}
	if utxErr := baseInfo.AddToUtx(t); utxErr != nil {
		return fsm, nil, utxErr
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
	event stateless.Trigger, state *StateData, actionFunc func(...any) (State, Async, error),
) stateless.DestinationSelectorFunc {
	return func(_ context.Context, args ...any) (stateless.State, error) {
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

func convertToInterface[T any](arg any) T {
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

func validateEventArgs(event stateless.Trigger, args ...any) {
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

func broadcastMicroBlockInv(info BaseInfo, inv *proto.MicroBlockInv) error {
	invBts, err := inv.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal binary '%T'", inv)
	}
	var (
		cnt int
		msg = &proto.MicroBlockInvMessage{
			Body: invBts,
		}
	)
	info.peers.EachConnected(func(p peer.Peer, _ *proto.Score) {
		p.SendMessage(msg)
		cnt++
	})
	info.invRequester.Add2Cache(inv.TotalBlockID) // prevent further unnecessary microblock request
	info.logger.Debug("Network message sent to peers", logging.Type(msg), slog.Int("count", cnt),
		slog.Any("blockID", inv.TotalBlockID), slog.Any("ref", inv.Reference))
	return nil
}

func processScoreAfterApplyingOrReturnToNG(
	state State,
	baseInfo BaseInfo,
	scores []ReceivedScore,
	cache blockStatesCache,
) (State, Async, error) {
	for _, s := range scores {
		if err := baseInfo.peers.UpdateScore(s.Peer, s.Score); err != nil {
			info := proto.NewInfoMsg(err)
			baseInfo.logger.Debug("Failed to update score", slog.String("state", state.String()),
				logging.Error(info))
			continue
		}
		nodeScore, err := baseInfo.storage.CurrentScore()
		if err != nil {
			info := proto.NewInfoMsg(err)
			baseInfo.logger.Debug("Failed to get current score", slog.String("state", state.String()),
				logging.Error(info))
			continue
		}
		if s.Score.Cmp(nodeScore) == 1 {
			// received score is larger than local score
			newS, task, errS := syncWithNewPeer(state, baseInfo, s.Peer)
			if errS != nil {
				se := state.Errorf(errS)
				slog.Error("Failed to sync with peer", slog.String("state", state.String()), logging.Error(se))
				continue
			}
			if newSName := newS.String(); newSName != SyncStateName { // sanity check
				return newS, task, errors.Errorf("unexpected state %q after sync with peer, want %q",
					newSName, SyncStateName,
				)
			}
			return newS, task, nil
		}
	}
	return newNGStateWithCache(baseInfo, cache), nil, nil
}
