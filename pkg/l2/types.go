package l2

import (
	"bytes"
	"math/big"
	"strconv"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	EmptyFeeRecipient  = "0x0000000000000000000000000000000000000000"
	EmptyPrevRandaoHex = "0x0000000000000000000000000000000000000000000000000000000000000000"
	EmptyRootHashHex   = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"

	HexBase = 16
)

const (
	ValidStatus   = "VALID"
	SyncingStatus = "SYNCING"
)

func EmptyPrevRandaoEthHash() (proto.EthereumHash, error) {
	emptyPrevRandaoBytes, err := proto.DecodeFromHexString(EmptyPrevRandaoHex)
	if err != nil {
		return proto.EthereumHash{}, err
	}
	return proto.BytesToEthereumHash(emptyPrevRandaoBytes), nil
}

func emptyFeeRecipient() (proto.EthereumAddress, error) {
	return proto.NewEthereumAddressFromHexString(EmptyFeeRecipient)
}

func EmptyRootHash() (proto.EthereumHash, error) {
	emptyRootHashBytes, err := proto.DecodeFromHexString(EmptyRootHashHex)
	if err != nil {
		return proto.EthereumHash{}, err
	}
	return proto.BytesToEthereumHash(emptyRootHashBytes), nil
}

type ForkChoiceStateV1 struct {
	HeadBlockHash      proto.EthereumHash `json:"headBlockHash"`
	SafeBlockHash      proto.EthereumHash `json:"safeBlockHash"`
	FinalizedBlockHash proto.EthereumHash `json:"finalizedBlockHash"`
}

type PayloadAttributes struct {
	Timestamp             Quantity              `json:"timestamp"`
	Random                proto.EthereumHash    `json:"prevRandao"`
	SuggestedFeeRecipient proto.EthereumAddress `json:"suggestedFeeRecipient"`
	Withdrawals           []Withdrawal          `json:"withdrawals"`
	BeaconRoot            *proto.EthereumHash   `json:"parentBeaconBlockRoot"`
}

type Withdrawal struct {
	Index     Quantity              `json:"index"`
	Validator Quantity              `json:"validatorIndex"`
	Address   proto.EthereumAddress `json:"address"`
	Amount    Quantity              `json:"amount"`
}

type ForkChoiceResponse struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *PayloadID      `json:"payloadId"`
}

type PayloadStatusV1 struct {
	Status          string              `json:"status"`
	LatestValidHash *proto.EthereumHash `json:"latestValidHash"`
	ValidationError *string             `json:"validationError"`
}

type PayloadID [8]byte

func (b PayloadID) String() string {
	return proto.EncodeToHexString(b[:])
}

type Quantity uint64

func (h Quantity) MarshalJSON() ([]byte, error) {
	s := strconv.FormatUint(uint64(h), HexBase)
	var sb bytes.Buffer
	sb.Grow(2 + len(s))
	sb.WriteString("\"0x")
	sb.WriteString(s)
	sb.WriteRune('"')
	return sb.Bytes(), nil
}

func (h *Quantity) UnmarshalJSON(bytes []byte) error {
	trimmed := strings.TrimPrefix(string(bytes), "0x")
	u, err := strconv.ParseUint(trimmed, HexBase, 64)
	if err != nil {
		return err
	}
	*h = Quantity(u)
	return nil
}

type ExecutablePayloadV3 struct {
	ParentHash    proto.EthereumHash    `json:"parentHash"`
	FeeRecipient  proto.EthereumAddress `json:"feeRecipient"`
	StateRoot     proto.EthereumHash    `json:"stateRoot"`
	ReceiptsRoot  proto.EthereumHash    `json:"receiptsRoot"`
	LogsBloom     proto.HexBytes        `json:"logsBloom"`
	Random        proto.EthereumHash    `json:"prevRandao"`
	Number        Quantity              `json:"blockNumber"`
	GasLimit      Quantity              `json:"gasLimit"`
	GasUsed       Quantity              `json:"gasUsed"`
	Timestamp     Quantity              `json:"timestamp"`
	ExtraData     proto.HexBytes        `json:"extraData"`
	BaseFeePerGas *string               `json:"baseFeePerGas"`
	BlockHash     proto.EthereumHash    `json:"blockHash"`
	Transactions  []proto.HexBytes      `json:"transactions"`
	Withdrawals   []*Withdrawal         `json:"withdrawals"`
	BlobGasUsed   *Quantity             `json:"blobGasUsed"`
	ExcessBlobGas *Quantity             `json:"excessBlobGas"`
}

type BlobsBundleV1 struct {
	Commitments []proto.HexBytes `json:"commitments"`
	Proofs      []proto.HexBytes `json:"proofs"`
	Blobs       []proto.HexBytes `json:"blobs"`
}

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutablePayloadV3 `json:"executionPayload"`
	BlockValue       *big.Int             `json:"blockValue"`
	BlobsBundle      *BlobsBundleV1       `json:"blobsBundle"`
	Override         bool                 `json:"shouldOverrideBuilder"`
}

type ExecutionPayloadBodyV1 struct {
	TransactionData []proto.HexBytes `json:"transactions"`
	Withdrawals     []*Withdrawal    `json:"withdrawals"`
}
