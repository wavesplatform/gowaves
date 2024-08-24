package types

import (
	"context"
	"fmt"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type Scheduler interface {
	Reschedule()
}

// Handler is an abstract function that called when an event happens.
type Handler interface {
	Handle()
}

// UtxPool storage interface
type UtxPool interface {
	Add(t proto.Transaction) error
	AddWithBytes(t proto.Transaction, b []byte) error
	Exists(t proto.Transaction) bool
	Pop() *TransactionWithBytes
	AllTransactions() []*TransactionWithBytes
	Count() int
	ExistsByID(id []byte) bool
}

type TransactionWithBytes struct {
	T proto.Transaction
	B []byte
}

//go:generate moq -out ../state/smart_state_moq_test.go -pkg state . SmartState:AnotherMockSmartState

// WavesBalanceProfile contains essential parts of Waves balance and
// must be used to pass this information if SmartState only.
type WavesBalanceProfile struct {
	Balance    uint64
	LeaseIn    int64
	LeaseOut   int64
	Generating uint64
	Challenged bool // if Challenged true, the account considered as challenged at the current height.
}

// EffectiveBalance returns effective balance with checking for account challenging.
// The function MUST be used ONLY in the context where account challenging IS CHECKED.
func (bp *WavesBalanceProfile) EffectiveBalance() (uint64, error) {
	switch {
	case bp.Challenged:
		return 0, nil
	case bp.LeaseIn < 0:
		return 0, fmt.Errorf("negative lease in balance %d", bp.LeaseIn)
	case bp.LeaseOut < 0:
		return 0, fmt.Errorf("negative lease out balance %d", bp.LeaseOut)
	}
	val, err := common.AddInt(bp.Balance, uint64(bp.LeaseIn))
	if err != nil {
		return 0, err
	}
	return common.SubInt(val, uint64(bp.LeaseOut))
}

func (bp *WavesBalanceProfile) SpendableBalance() (uint64, error) {
	if bp.LeaseOut < 0 {
		return 0, fmt.Errorf("negative lease out balance %d", bp.LeaseOut)
	}
	return common.SubInt(bp.Balance, uint64(bp.LeaseOut))
}

func (bp *WavesBalanceProfile) ToFullWavesBalance() (*proto.FullWavesBalance, error) {
	available, err := bp.SpendableBalance()
	if err != nil {
		return nil, err
	}
	effective, err := bp.EffectiveBalance()
	if err != nil {
		return nil, err
	}
	return &proto.FullWavesBalance{
		Regular:    bp.Balance,
		Generating: bp.Generating,
		Available:  available,
		Effective:  effective,
		LeaseIn:    uint64(bp.LeaseIn),  // LeaseIn is always non-negative, because it's checked in EffectiveBalance.
		LeaseOut:   uint64(bp.LeaseOut), // LeaseOut is always non-negative, because it's checked in EffectiveBalance.
	}, nil
}

// SmartState is a part of state used by smart contracts.

type SmartState interface {
	NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error)
	AddingBlockHeight() (uint64, error)
	// NewestTransactionByID returns a transaction, BUT returns error if a transaction exists but failed or elided.
	NewestTransactionByID([]byte) (proto.Transaction, error)
	// NewestTransactionHeightByID returns a transaction height, BUT returns error if a transaction
	//  exists but failed or elided.
	NewestTransactionHeightByID([]byte) (uint64, error)
	NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error)
	NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error)
	NewestRecipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error)
	NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error)
	NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error)
	IsStateUntouched(account proto.Recipient) (bool, error)
	NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error)
	NewestWavesBalance(account proto.Recipient) (uint64, error)
	// NewestFullWavesBalance returns a full Waves balance of account.
	// The method must be used ONLY in the Ride environment.
	// The boundaries of the generating balance are calculated for the current height of applying block,
	// instead of the last block height.
	//
	// For example, for the block validation we are use min effective balance of the account from height 1 to 1000.
	// This function uses heights from 2 to 1001, where 1001 is the height of the applying block.
	// All changes of effective balance during the applying block are affecting the generating balance.
	NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error)
	RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)
	NewestAssetIsSponsored(assetID crypto.Digest) (bool, error)
	NewestAssetConstInfo(assetID proto.AssetID) (*proto.AssetConstInfo, error)
	NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
	NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error)
	NewestScriptByAsset(assetID crypto.Digest) (*ast.Tree, error)
	NewestBlockInfoByHeight(height proto.Height) (*proto.BlockInfo, error)

	EstimatorVersion() (int, error)
	IsNotFound(err error) bool

	// WavesBalanceProfile returns WavesBalanceProfile structure retrieved by proto.AddressID of an account.
	// This function always returns the newest available state of Waves balance of account.
	// Thought, it can't be used during transaction processing, because the state does no hold changes between txs.
	// The method must be used ONLY in the Ride environment for retrieving data from state.
	// The boundaries of the generating balance are calculated for the current height of applying block,
	// instead of the last block height.
	//
	// For example, for the block validation we are use min effective balance of the account from height 1 to 1000.
	// This function uses heights from 2 to 1001, where 1001 is the height of the applying block.
	// All changes of effective balance during the applying block are affecting the generating balance.
	WavesBalanceProfile(id proto.AddressID) (*WavesBalanceProfile, error)

	// NewestAssetBalanceByAddressID returns the most actual asset balance by given proto.AddressID and
	// assets crypto.Digest.
	NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error)

	//TODO: The last 2 functions intended to be used only in wrapped state. Extract separate interface for such functions.
}

type ID interface {
	ID() string
}

type Subscribe interface {
	Subscribe(p ID, responseMessage proto.Message) (chan proto.Message, func(), error)
	Receive(p ID, responseMessage proto.Message) bool
}

type StateSync interface {
	Sync()
	SetEnabled(enabled bool)
	Close()
	Run(ctx context.Context)
}

type MessageSender interface {
	SendMessage(proto.Message)
}

type InvRequester interface {
	Request(MessageSender, []byte)
}

type BaseTarget = uint64

type Miner interface {
	MineKeyBlock(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent proto.BlockID, baseTarget BaseTarget, gs []byte, vrf []byte) (*proto.Block, proto.MiningLimits, error)
}

type Time interface {
	Now() time.Time
}

type ScoreSender interface {
	Priority()
	NonPriority()
}

type MinerConsensus interface {
	IsMiningAllowed() bool
}

type EmbeddedWallet interface {
	SignTransactionWith(pk crypto.PublicKey, tx proto.Transaction) error
	Load(password []byte) error
	AccountSeeds() [][]byte
}
