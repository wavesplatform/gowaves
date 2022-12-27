package types

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
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
}

// SmartState is a part of state used by smart contracts.

type SmartState interface {
	NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error)
	AddingBlockHeight() (uint64, error)
	NewestTransactionByID([]byte) (proto.Transaction, error)
	NewestTransactionHeightByID([]byte) (uint64, error)
	NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error)
	NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error)
	NewestRecipientToAddress(recipient proto.Recipient) (*proto.WavesAddress, error)
	NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error)
	NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error)
	IsStateUntouched(account proto.Recipient) (bool, error)
	NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error)
	NewestWavesBalance(account proto.Recipient) (uint64, error)
	NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error)
	RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)
	NewestAssetIsSponsored(assetID crypto.Digest) (bool, error)
	NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
	NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error)
	NewestScriptByAsset(assetID crypto.Digest) (*ast.Tree, error)
	NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error)
	BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error)

	EstimatorVersion() (int, error)
	IsNotFound(err error) bool

	// WavesBalanceProfile returns WavesBalanceProfile structure retrieved by proto.AddressID of an account.
	// This function always returns the newest available state of Waves balance of account.
	WavesBalanceProfile(id proto.AddressID) (*WavesBalanceProfile, error)

	// NewestAssetBalanceByAddressID returns the most actual asset balance by given proto.AddressID and
	// assets crypto.Digest.
	NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error)

	// NewestScriptVersionByAddressID returns library version of the script on the account with given proto.AddressID.
	// In case of no script on account an error is returned.
	NewestScriptVersionByAddressID(id proto.AddressID) (ast.LibraryVersion, error)

	//TODO: The last 3 functions intended to be used only in wrapped state. Extract separate interface for such functions.
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
