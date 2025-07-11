package scheduler

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/cancellable"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"github.com/wavesplatform/gowaves/pkg/wallet"
)

type Emit struct {
	Timestamp    uint64
	KeyPair      proto.KeyPair
	GenSignature []byte
	VRF          []byte
	BaseTarget   types.BaseTarget
	Parent       proto.BlockID
}

type Default struct {
	seeder         seeder
	mine           chan Emit
	cancel         []func()
	settings       *settings.BlockchainSettings
	mu             sync.Mutex
	internal       internal
	emits          []Emit
	storage        state.State
	tm             types.Time
	consensus      types.MinerConsensus
	obsolescence   time.Duration
	generateInPast bool
}

type internal interface {
	schedule(
		state state.StateInfo,
		keyPairs []proto.KeyPair,
		settings *settings.BlockchainSettings,
		confirmedBlock *proto.Block,
		confirmedBlockHeight uint64,
		tm types.Time,
		generateInPast bool,
	) ([]Emit, error)
}

type internalImpl struct{}

func (a internalImpl) schedule(
	storage state.StateInfo,
	keyPairs []proto.KeyPair,
	blockchainSettings *settings.BlockchainSettings,
	confirmedBlock *proto.Block,
	confirmedBlockHeight uint64,
	tm types.Time,
	generateInPast bool,
) ([]Emit, error) {
	vrfActivated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return nil, errors.Wrap(err, "failed get vrfActivated")
	}
	if vrfActivated {
		return a.scheduleWithVrf(storage, keyPairs, blockchainSettings, confirmedBlock, confirmedBlockHeight,
			tm, generateInPast)
	}
	return a.scheduleWithoutVrf(storage, keyPairs, blockchainSettings, confirmedBlock, confirmedBlockHeight, tm,
		generateInPast)
}

func (a internalImpl) prepareDataForSchedule(
	storage state.StateInfo,
	confirmedBlockHeight uint64,
	blockchainSettings *settings.BlockchainSettings,
) (proto.Timestamp, bool, consensus.PosCalculator, error) {
	var greatGrandParentTimestamp proto.Timestamp = 0
	if confirmedBlockHeight > 2 {
		greatGrandParentHeight := confirmedBlockHeight - 2
		greatGrandParent, err := storage.HeaderByHeight(greatGrandParentHeight)
		if err != nil {
			slog.Error("Scheduler: Failed to get blockID by height",
				"height", greatGrandParentHeight, "error", err)
			return 0, false, nil, err
		}
		greatGrandParentTimestamp = greatGrandParent.Timestamp
	}
	fairPosActivated, err := storage.IsActiveAtHeight(int16(settings.FairPoS), confirmedBlockHeight)
	if err != nil {
		return 0, false, nil, errors.Wrap(err, "failed get fairPosActivated")
	}
	blockV5Activated, err := storage.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return 0, false, nil, errors.Wrap(err, "failed get blockV5Activated")
	}
	pos := consensus.NXTPosCalculator
	if fairPosActivated {
		if blockV5Activated {
			pos = consensus.NewFairPosCalculator(blockchainSettings.DelayDelta, blockchainSettings.MinBlockTime)
		} else {
			pos = consensus.FairPosCalculatorV1
		}
	}
	return greatGrandParentTimestamp, blockV5Activated, pos, nil
}

func (a internalImpl) scheduleWithVrf(
	storage state.StateInfo,
	keyPairs []proto.KeyPair,
	blockchainSettings *settings.BlockchainSettings,
	confirmedBlock *proto.Block,
	confirmedBlockHeight uint64,
	tm types.Time,
	generateInPast bool,
) ([]Emit, error) {
	greatGrandParentTimestamp, blockV5Activated, pos, err := a.prepareDataForSchedule(storage, confirmedBlockHeight,
		blockchainSettings,
	)
	if err != nil {
		return nil, err
	}

	gsp := consensus.NXTGenerationSignatureProvider
	if blockV5Activated {
		gsp = consensus.VRFGenerationSignatureProvider
	}

	heightForHit := pos.HeightForHit(confirmedBlockHeight)
	hitSourceAtHeight, err := storage.HitSourceAtHeight(heightForHit)
	if err != nil {
		slog.Error("Scheduler: Failed to get hit source at height", "height", heightForHit, "error", err)
		return nil, err
	}
	slog.Debug("Scheduler: Top block", "BlockID", confirmedBlock.BlockID().String(),
		"GenSig", confirmedBlock.GenSignature, "height", confirmedBlockHeight)

	var out []Emit
	for _, keyPair := range keyPairs {
		sk := keyPair.Secret
		genSig, err := gsp.GenerationSignature(sk, hitSourceAtHeight)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining, can't get generation signature at height",
				"height", heightForHit, "error", err)
			continue
		}
		source, err := gsp.HitSource(sk, hitSourceAtHeight)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to get hit source at height",
				"height", heightForHit, "error", err)
			continue
		}
		var vrf []byte
		if blockV5Activated {
			vrf = source
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to generate hit from source", "error", err)
			continue
		}

		addr, err := keyPair.Addr(blockchainSettings.AddressSchemeCharacter)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to create address from PK", "error", err)
			continue
		}

		generatingBalance, err := storage.GeneratingBalance(proto.NewRecipientFromAddress(addr), confirmedBlockHeight)
		if err != nil {
			slog.Debug("Scheduler: Failed to get generating balance for address", "address", addr.String(),
				"height", confirmedBlockHeight, "error", err)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BaseTarget, generatingBalance)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining for address %q, failed to calculate delay",
				"address", addr.String(), "error", err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(
			blockchainSettings.AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			delay+confirmedBlock.Timestamp,
		)
		if err != nil {
			slog.Error("Scheduler: Failed to schedule mining for address, failed to calculate base target",
				"address", addr.String(), "error", err)
			continue
		}
		ts := confirmedBlock.Timestamp + delay // Maybe in past or future.
		if !generateInPast {
			now := proto.NewTimestampFromTime(tm.Now())
			if ts < now {
				ts = now
			}
		}
		sts, err := safecast.ToInt64(ts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to cast timestamp to int64")
		}
		slog.Debug("Scheduled generation", "address", addr.String(),
			"time", time.UnixMilli(sts).Format("2006-01-02 15:04:05.000 MST"))
		out = append(out, Emit{
			Timestamp:    ts,
			KeyPair:      keyPair,
			GenSignature: genSig,
			VRF:          vrf,
			BaseTarget:   baseTarget,
			Parent:       confirmedBlock.BlockID(),
		})
	}
	return out, nil
}

func (a internalImpl) scheduleWithoutVrf(
	storage state.StateInfo,
	keyPairs []proto.KeyPair,
	blockchainSettings *settings.BlockchainSettings,
	confirmedBlock *proto.Block,
	confirmedBlockHeight uint64,
	tm types.Time,
	generateInPast bool,
) ([]Emit, error) {
	greatGrandParentTimestamp, _, pos, err := a.prepareDataForSchedule(storage, confirmedBlockHeight, blockchainSettings)
	if err != nil {
		return nil, err
	}

	gsp := consensus.NXTGenerationSignatureProvider

	heightForHit := pos.HeightForHit(confirmedBlockHeight)
	hitSourceHeader, err := storage.HeaderByHeight(heightForHit)
	if err != nil {
		slog.Error("Scheduler: Failed to get header by height", "height", heightForHit, "error", err)
		return nil, err
	}

	slog.Debug("Scheduling generation", "height", confirmedBlockHeight,
		"BlockID", confirmedBlock.BlockID().String(), "BlockTS", confirmedBlock.Timestamp, "BlockTime",
		time.UnixMilli(int64(confirmedBlock.Timestamp)), // #nosec: used only for logging
		"BaseTarget", confirmedBlock.BaseTarget)
	var out []Emit
	for _, keyPair := range keyPairs {
		pk := keyPair.Public
		genSigBlock := confirmedBlock.BlockHeader
		genSig, err := gsp.GenerationSignature(pk, genSigBlock.GenSignature)
		if err != nil {
			slog.Error("Scheduler: Failed to get generation signature for PK", "PK", pk.String(), "error", err)
			continue
		}
		source, err := gsp.HitSource(pk, hitSourceHeader.GenSignature)
		if err != nil {
			slog.Error("Scheduler: Failed to generate hit source for PK", "PK", pk.String(), "error", err)
			continue
		}
		hit, err := consensus.GenHit(source)
		if err != nil {
			slog.Error("Scheduler: Failed to generate hit for PK", "PK", pk.String(), "error", err)
			continue
		}

		addr, err := proto.NewAddressFromPublicKey(blockchainSettings.AddressSchemeCharacter, pk)
		if err != nil {
			slog.Error("Scheduler: Failed to create new address from PK", "PK", pk.String(), "error", err)
			continue
		}

		generatingBalance, err := storage.GeneratingBalance(proto.NewRecipientFromAddress(addr), confirmedBlockHeight)
		if err != nil {
			slog.Debug("Scheduler: Failed to get generating balance", "address", addr.String(),
				"height", confirmedBlockHeight, "error", err)
			continue
		}

		delay, err := pos.CalculateDelay(hit, confirmedBlock.BaseTarget, generatingBalance)
		if err != nil {
			slog.Error("Scheduler: Failed to calculate delay for address", "address", addr,
				"balance", generatingBalance, "error", err)
			continue
		}

		baseTarget, err := pos.CalculateBaseTarget(
			blockchainSettings.AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			delay+confirmedBlock.Timestamp,
		)
		if err != nil {
			slog.Error("Scheduler: Failed to calculate base target", "address", addr.String(), "error", err)
			continue
		}
		ts := confirmedBlock.Timestamp + delay // Maybe in past or future.
		if !generateInPast {
			now := proto.NewTimestampFromTime(tm.Now())
			if ts < now {
				ts = now
			}
		}
		slog.Debug("Schedule generation", "address", addr.String(), "pk", pk.String(), "hit", hit.String(),
			"hitSource", base58.Encode(source), "generationBalance", generatingBalance, "delay", delay,
			"timestamp", ts, "time", common.UnixMillisToTime(int64(ts)).String()) // #nosec: used only for logging
		out = append(out, Emit{
			Timestamp:    ts,
			KeyPair:      keyPair,
			GenSignature: genSig,
			VRF:          nil, // because without VRF
			BaseTarget:   baseTarget,
			Parent:       confirmedBlock.BlockID(),
		})
	}
	return out, nil
}

type seeder interface {
	AccountSeeds() [][]byte
}

func NewScheduler(
	state state.State,
	seeder seeder,
	settings *settings.BlockchainSettings,
	tm types.Time,
	consensus types.MinerConsensus,
	minerDelay time.Duration,
	generateInPast bool) (*Default, error) {
	if minerDelay <= 0 {
		return nil, errors.New("minerDelay must be positive")
	}
	return newScheduler(internalImpl{}, state, seeder, settings, tm, consensus, minerDelay, generateInPast), nil
}

func newScheduler(internal internal, state state.State, seeder seeder, settings *settings.BlockchainSettings,
	tm types.Time, consensus types.MinerConsensus, minerDelay time.Duration, generateInPast bool) *Default {
	if seeder == nil {
		seeder = wallet.NewWallet()
	}
	return &Default{
		seeder:         seeder,
		mine:           make(chan Emit, 1),
		settings:       settings,
		internal:       internal,
		storage:        state,
		mu:             sync.Mutex{},
		tm:             tm,
		consensus:      consensus,
		obsolescence:   minerDelay,
		generateInPast: generateInPast,
	}
}

func (a *Default) Mine() chan Emit {
	return a.mine
}

func (a *Default) Reschedule() {
	if len(a.seeder.AccountSeeds()) == 0 {
		slog.Debug("Scheduler: Mining is not possible because no seeds registered")
		return
	}

	slog.Debug("Scheduler: Trying to mine", "accounts", len(a.seeder.AccountSeeds()))

	if !a.consensus.IsMiningAllowed() {
		slog.Debug("Scheduler: Mining is not allowed because of lack of connected nodes")
		return
	}

	now := a.tm.Now()
	obsolescenceTime := now.Add(-a.obsolescence)
	lastBlock := a.storage.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	if obsolescenceTime.After(lastBlockTime) {
		slog.Debug("Scheduler: Mining is not allowed because last block is before the obsolescence time",
			"BlockID", lastBlock.ID, "BlockTime", lastBlockTime, "ObsolescenceTime", obsolescenceTime)
		return
	}

	h, err := a.storage.Height()
	if err != nil {
		slog.Error("Scheduler: Failed to get state height", "error", err)
		return
	}

	block, err := a.storage.BlockByHeight(h)
	if err != nil {
		slog.Error("Scheduler: Failed to get block by height", "height", h, "error", err)
		return
	}

	a.reschedule(block, h)
}

func (a *Default) reschedule(confirmedBlock *proto.Block, confirmedBlockHeight uint64) {
	if len(a.seeder.AccountSeeds()) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// stop previous timeouts
	for _, cancel := range a.cancel {
		cancel()
	}
	a.cancel = nil
	a.emits = nil

	keyPairs, err := makeKeyPairs(a.seeder.AccountSeeds())
	if err != nil {
		slog.Error("Scheduler: Failed to make key pairs from seeds", "error", err)
		return
	}

	rs, err := a.storage.MapR(func(info state.StateInfo) (i interface{}, err error) {
		return a.internal.schedule(info, keyPairs, a.settings, confirmedBlock, confirmedBlockHeight, a.tm,
			a.generateInPast)
	})
	if err != nil {
		slog.Error("Scheduler: Failed to schedule", "error", err)
	}
	emits := rs.([]Emit)

	a.emits = emits
	now := proto.NewTimestampFromTime(a.tm.Now())
	for _, emit := range emits {
		if emit.Timestamp > now { // timestamp in future
			timeout := emit.Timestamp - now
			emit_ := emit
			cancel := cancellable.After(time.Duration(timeout)*time.Millisecond, func() {
				// hack for integrations tests
				common.EnsureTimeout(a.tm, emit_.Timestamp)
				select {
				case a.mine <- emit_:
				default:
					slog.Debug("Scheduler: cannot emit a.mine, chan is full")
				}
			})
			a.cancel = append(a.cancel, cancel)
		} else {
			select {
			case a.mine <- emit:
			default:
				slog.Debug("Scheduler: cannot emit a.mine, chan is full")
			}
		}
	}
}

func (a *Default) Emits() []Emit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.emits
}

func makeKeyPairs(seeds [][]byte) ([]proto.KeyPair, error) {
	var out []proto.KeyPair
	for _, bts := range seeds {
		kp, err := proto.NewKeyPair(bts)
		if err != nil {
			return nil, err
		}
		out = append(out, kp)
	}
	return out, nil
}
