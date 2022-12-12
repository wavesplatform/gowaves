package client

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type DataEntries = proto.DataEntries

type TransferAction struct {
	Address proto.WavesAddress  `json:"address"`
	Asset   proto.OptionalAsset `json:"asset"`
	Amount  int64               `json:"amount"`
}

type IssueAction struct {
	AssetID        crypto.Digest `json:"assetId"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Decimals       int32         `json:"decimals"`
	Quantity       int64         `json:"quantity"`
	Reissuable     bool          `json:"isReissuable"`
	CompiledScript []byte        `json:"compiledScript"` // optional
	Nonce          int64         `json:"nonce"`
}

type ReissueAction struct {
	AssetID    crypto.Digest `json:"assetId"`
	Reissuable bool          `json:"isReissuable"`
	Quantity   int64         `json:"quantity"`
}

type BurnAction struct {
	AssetID  crypto.Digest `json:"assetId"`
	Quantity int64         `json:"quantity"`
}

type SponsorFeeAction struct {
	AssetID              crypto.Digest `json:"assetId"`
	MinSponsoredAssetFee *int64        `json:"minSponsoredAssetFee"` // optional
}

type LeaseAction struct {
	ID                  crypto.Digest      `json:"id"`
	OriginTransactionId crypto.Digest      `json:"originTransactionId"`
	Sender              proto.WavesAddress `json:"sender"`
	Recipient           proto.Recipient    `json:"recipient"`
	Amount              int64              `json:"amount"`
	Height              uint32             `json:"height"`
	Status              LeaseStatus        `json:"status"`
	CancelHeight        *uint32            `json:"cancelHeight,omitempty"`        // optional
	CancelTransactionId *crypto.Digest     `json:"cancelTransactionId,omitempty"` // optional
}

type LeaseStatus byte

const (
	LeaseActiveStatus LeaseStatus = iota + 1
	LeaseCanceledStatus
)

func (s *LeaseStatus) UnmarshalJSON(data []byte) error {
	stringStatus, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	switch stringStatus {
	case "active":
		*s = LeaseActiveStatus
	case "canceled":
		*s = LeaseCanceledStatus
	default:
		return errors.Errorf("Unknown lease status: '%s'", stringStatus)
	}

	return nil
}

func (s LeaseStatus) String() string {
	switch s {
	case LeaseActiveStatus:
		return "active"
	case LeaseCanceledStatus:
		return "canceled"
	}
	return "unknown"
}

func (s LeaseStatus) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(s.String())), nil
}

type LeaseCancelAction LeaseAction

type InvokeAction struct {
	DApp         proto.WavesAddress    `json:"dApp"`
	Call         proto.FunctionCall    `json:"call"`
	Payments     []proto.ScriptPayment `json:"payment"`
	StateChanges StateChanges          `json:"stateChanges"`
}

type StateChanges struct {
	Data         DataEntries         `json:"data"`
	Transfers    []TransferAction    `json:"transfers"`
	Issues       []IssueAction       `json:"issues"`
	Reissues     []ReissueAction     `json:"reissues"`
	Burns        []BurnAction        `json:"burns"`
	SponsorFees  []SponsorFeeAction  `json:"sponsorFees"`
	Leases       []LeaseAction       `json:"leases"`
	LeaseCancels []LeaseCancelAction `json:"leaseCancels"`
	Invokes      []InvokeAction      `json:"invokes"`
}
