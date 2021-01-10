package types

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Scheduler interface {
	Reschedule()
}

// Abstract handler that called when event happens
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

// state for smart contracts
type SmartState interface {
	NewestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error)
	AddingBlockHeight() (uint64, error)
	NewestTransactionByID([]byte) (proto.Transaction, error)
	NewestTransactionHeightByID([]byte) (uint64, error)
	GetByteTree(recipient proto.Recipient) (proto.Script, error)
	NewestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error)
	NewestAddrByAlias(alias proto.Alias) (proto.Address, error)

	// NewestAccountBalance retrieves balance of address in specific currency, asset is asset's ID.
	// nil asset = Waves.
	NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error)
	NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error)
	RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)
	NewestAssetIsSponsored(assetID crypto.Digest) (bool, error)
	NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
	NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error)
	//
	NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error)
	BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error)

	ApplyToState(actions []proto.ScriptAction) ([]proto.ScriptAction, error)
	EstimatorVersion() (int, error)
	IsNotFound(err error) bool
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
	Seeds() [][]byte
}
