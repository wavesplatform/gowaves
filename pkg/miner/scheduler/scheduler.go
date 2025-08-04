package scheduler

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/logging"
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
				slog.Any("height", greatGrandParentHeight), logging.Error(err))
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
		slog.Error("Scheduler: Failed to get hit source at height", slog.Any("height", heightForHit),
			logging.Error(err))
		return nil, err
	}
	slog.Debug("Scheduler: Top block", "BlockID", confirmedBlock.BlockID().String(),
		"GenSig", confirmedBlock.GenSignature, "height", confirmedBlockHeight)

	var out []Emit
	for _, keyPair := range keyPairs {
		sk := keyPair.Secret
		genSig, gsErr := gsp.GenerationSignature(sk, hitSourceAtHeight)
		if gsErr != nil {
			slog.Error("Scheduler: Failed to schedule mining, can't get generation signature at height",
				slog.Any("height", heightForHit), logging.Error(gsErr))
			continue
		}
		source, hsErr := gsp.HitSource(sk, hitSourceAtHeight)
		if hsErr != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to get hit source at height",
				slog.Any("height", heightForHit), logging.Error(hsErr))
			continue
		}
		var vrf []byte
		if blockV5Activated {
			vrf = source
		}
		hit, ghErr := consensus.GenHit(source)
		if ghErr != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to generate hit from source",
				logging.Error(ghErr))
			continue
		}

		addr, aErr := keyPair.Addr(blockchainSettings.AddressSchemeCharacter)
		if aErr != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to create address from PK",
				logging.Error(aErr))
			continue
		}

		generatingBalance, gbErr := storage.GeneratingBalance(proto.NewRecipientFromAddress(addr), confirmedBlockHeight)
		if gbErr != nil {
			slog.Debug("Scheduler: Failed to get generating balance for address",
				slog.String("address", addr.String()), slog.Any("height", confirmedBlockHeight),
				logging.Error(gbErr))
			continue
		}

		delay, cdErr := pos.CalculateDelay(hit, confirmedBlock.BaseTarget, generatingBalance)
		if cdErr != nil {
			slog.Error("Scheduler: Failed to schedule mining, failed to calculate delay",
				slog.String("address", addr.String()), logging.Error(cdErr))
			continue
		}
		ts := adjustTimestamp(confirmedBlock.Timestamp, delay, tm, generateInPast)

		baseTarget, btErr := pos.CalculateBaseTarget(
			blockchainSettings.AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			ts,
		)
		if btErr != nil {
			slog.Error("Scheduler: Failed to schedule mining for address, failed to calculate base target",
				slog.String("address", addr.String()), logging.Error(btErr))
			continue
		}
		sts, err := safecast.ToInt64(ts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to cast timestamp to int64")
		}
		const format = "2006-01-02 15:04:05.000 MST"
		slog.Debug("Scheduled generation", "address", addr.String(), "time", time.UnixMilli(sts).Format(format))
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
		slog.Error("Scheduler: Failed to get header by height", slog.Any("height", heightForHit),
			logging.Error(err))
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
		genSig, gsErr := gsp.GenerationSignature(pk, genSigBlock.GenSignature)
		if gsErr != nil {
			slog.Error("Scheduler: Failed to get generation signature for PK", slog.String("PK", pk.String()),
				logging.Error(gsErr))
			continue
		}
		source, hsErr := gsp.HitSource(pk, hitSourceHeader.GenSignature)
		if hsErr != nil {
			slog.Error("Scheduler: Failed to generate hit source for PK", slog.String("PK", pk.String()),
				logging.Error(hsErr))
			continue
		}
		hit, ghErr := consensus.GenHit(source)
		if ghErr != nil {
			slog.Error("Scheduler: Failed to generate hit for PK", slog.String("PK", pk.String()),
				logging.Error(ghErr))
			continue
		}

		addr, naErr := proto.NewAddressFromPublicKey(blockchainSettings.AddressSchemeCharacter, pk)
		if naErr != nil {
			slog.Error("Scheduler: Failed to create new address from PK", slog.String("PK", pk.String()),
				logging.Error(naErr))
			continue
		}

		generatingBalance, gbErr := storage.GeneratingBalance(proto.NewRecipientFromAddress(addr), confirmedBlockHeight)
		if gbErr != nil {
			slog.Debug("Scheduler: Failed to get generating balance", slog.String("address", addr.String()),
				slog.Any("height", confirmedBlockHeight), logging.Error(gbErr))
			continue
		}

		delay, cdErr := pos.CalculateDelay(hit, confirmedBlock.BaseTarget, generatingBalance)
		if cdErr != nil {
			slog.Error("Scheduler: Failed to calculate delay for address", slog.String("address", addr.String()),
				slog.Any("balance", generatingBalance), logging.Error(cdErr))
			continue
		}
		ts := adjustTimestamp(confirmedBlock.Timestamp, delay, tm, generateInPast)

		baseTarget, btErr := pos.CalculateBaseTarget(
			blockchainSettings.AverageBlockDelaySeconds,
			confirmedBlockHeight,
			confirmedBlock.BaseTarget,
			confirmedBlock.Timestamp,
			greatGrandParentTimestamp,
			ts,
		)
		if btErr != nil {
			slog.Error("Scheduler: Failed to calculate base target", slog.String("address", addr.String()),
				logging.Error(btErr))
			continue
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

	slog.Debug("Scheduler: Trying to mine", "accountsCount", len(a.seeder.AccountSeeds()))

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
		slog.Error("Scheduler: Failed to get state height", logging.Error(err))
		return
	}

	block, err := a.storage.BlockByHeight(h)
	if err != nil {
		slog.Error("Scheduler: Failed to get block by height", slog.Any("height", h), logging.Error(err))
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
		slog.Error("Scheduler: Failed to make key pairs from seeds", logging.Error(err))
		return
	}

	rs, err := a.storage.MapR(func(info state.StateInfo) (any, error) {
		return a.internal.schedule(info, keyPairs, a.settings, confirmedBlock, confirmedBlockHeight, a.tm,
			a.generateInPast)
	})
	if err != nil {
		slog.Error("Scheduler: Failed to schedule", logging.Error(err))
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

func adjustTimestamp(
	confirmedBlockTimestamp proto.Timestamp,
	delay uint64,
	tm types.Time,
	generateInPast bool,
) proto.Timestamp {
	ts := confirmedBlockTimestamp + delay // Maybe in past or future.
	if !generateInPast {
		now := proto.NewTimestampFromTime(tm.Now())
		if ts < now {
			ts = now
		}
	}
	return ts
}
