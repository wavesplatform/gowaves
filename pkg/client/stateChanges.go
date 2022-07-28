package client

import (
	"encoding/json"

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
	Reissuable     bool          `json:"isReissuable"`
	CompiledScript string        `json:"compiledScript"`
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
	MinSponsoredAssetFee int64         `json:"minSponsoredAssetFee"`
}

type LeaseAction struct {
	ID                  crypto.Digest    `json:"id"`
	OriginTransactionId crypto.Digest    `json:"originTransactionId"`
	Sender              crypto.PublicKey `json:"sender"`
	Recipient           proto.Recipient  `json:"recipient"`
	Amount              int32            `json:"amount"`
	Height              int32            `json:"height"`
	Status              LeaseStatus      `json:"status"`
	CancelHeight        int32            `json:"cancelHeight,omitempty"`
	CancelTransactionId crypto.Digest    `json:"cancelTransactionId,omitempty"`
}

type LeaseStatus byte

const (
	LeaseActiveStatus LeaseStatus = iota
	LeaseCanceledStatus
)

func (s *LeaseStatus) UnmarshalJSON(data []byte) error {
	var stringStatus string
	if err := json.Unmarshal(data, &stringStatus); err != nil {
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

type LeaseCancelAction struct {
	LeaseID crypto.Digest `json:"leaseId"`
}

type InvokeAction struct {
	DApp         proto.Recipient    `json:"dApp"`
	Call         proto.FunctionCall `json:"call"`
	Payments     []*proto.Payment   `json:"payment"`
	StateChanges StateChanges       `json:"stateChanges"`
}

type StateChanges struct {
	Data         *DataEntries         `json:"data"`
	Transfers    []*TransferAction    `json:"transfers"`
	Issues       []*IssueAction       `json:"issues"`
	Reissues     []*ReissueAction     `json:"reissues"`
	Burns        []*BurnAction        `json:"burns"`
	SponsorFees  []*SponsorFeeAction  `json:"sponsorFees"`
	Leases       []*LeaseAction       `json:"leases"`
	LeaseCancels []*LeaseCancelAction `json:"leaseCancel"`
	Invokes      []*InvokeAction      `json:"invokes"`
}
