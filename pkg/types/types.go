package types

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	//"github.com/wavesplatform/gowaves/pkg/state"
)

type Scheduler interface {
	Reschedule()
}

//type BlocksApplier interface {
//	Apply(state state.State, block []*proto.Block) error
//}

//// notify state that it must run synchronization
//type StateHistorySynchronizer interface {
//	Sync()
//}

// Abstract handler that called when event happens
type Handler interface {
	Handle()
}

// UtxPool storage interface
type UtxPool interface {
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
	AddingBlockHeight() (uint64, error)
	NewestTransactionByID([]byte) (proto.Transaction, error)
	NewestTransactionHeightByID([]byte) (uint64, error)

	// NewestAccountBalance retrieves balance of address in specific currency, asset is asset's ID.
	// nil asset = Waves.
	NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error)
	NewestAddrByAlias(alias proto.Alias) (proto.Address, error)
	RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error)
	RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error)
	RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error)
	RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error)
	NewestAssetIsSponsored(assetID crypto.Digest) (bool, error)
	NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error)
	NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error)

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
	Request(MessageSender, *proto.MicroBlockInv)
}

type BaseTarget = uint64

type Miner interface {
	MineKeyBlock(ctx context.Context, t proto.Timestamp, k proto.KeyPair, parent crypto.Signature, baseTarget BaseTarget, GenSignature []byte) (*proto.Block, proto.MiningLimits, error)
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

type BlockCreator interface {
	FromMicroblockRow(seq proto.MicroblockRow) (*proto.Block, error)
}
