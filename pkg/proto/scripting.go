package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/client"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
)

// ScriptAction common interface of script invocation actions.
type ScriptAction interface {
	scriptAction()
}

// DataEntryScriptAction is an action to manipulate account data state.
type DataEntryScriptAction struct {
	Entry DataEntry
}

func (a *DataEntryScriptAction) scriptAction() {}

func (a *DataEntryScriptAction) ToProtobuf() *g.DataTransactionData_DataEntry {
	return a.Entry.ToProtobuf()
}

// TransferScriptAction is an action to emit transfer of asset.
type TransferScriptAction struct {
	Recipient Recipient
	Amount    int64
	Asset     OptionalAsset
}

func (a TransferScriptAction) scriptAction() {}

func (a *TransferScriptAction) ToProtobuf() *g.InvokeScriptResult_Payment {
	amount := &g.Amount{
		AssetId: a.Asset.ToID(),
		Amount:  a.Amount,
	}
	return &g.InvokeScriptResult_Payment{
		Address: a.Recipient.Address.Bytes(),
		Amount:  amount,
	}
}

// IssueScriptAction is an action to issue a new asset as a result of script invocation.
type IssueScriptAction struct {
	ID          crypto.Digest // calculated field
	Name        string        // name
	Description string        // description
	Quantity    int64         // quantity
	Decimals    int32         // decimals
	Reissuable  bool          // isReissuable
	Script      []byte        // compiledScript //TODO: reversed for future use
	Timestamp   int64         // nonce
}

func (a IssueScriptAction) scriptAction() {}

func (a *IssueScriptAction) ToProtobuf() *g.InvokeScriptResult_Issue {
	return &g.InvokeScriptResult_Issue{
		AssetId:     a.ID.Bytes(),
		Name:        a.Name,
		Description: a.Description,
		Amount:      a.Quantity,
		Decimals:    a.Decimals,
		Reissuable:  a.Reissuable,
		Script:      nil, //TODO: in V4 is not used
		Nonce:       a.Timestamp,
	}
}

// ReissueScriptAction is an action to emit Reissue transaction as a result of script invocation.
type ReissueScriptAction struct {
	AssetID    crypto.Digest // assetId
	Quantity   int64         // quantity
	Reissuable bool          // isReissuable
}

func (a ReissueScriptAction) scriptAction() {}

func (a *ReissueScriptAction) ToProtobuf() *g.InvokeScriptResult_Reissue {
	return &g.InvokeScriptResult_Reissue{
		AssetId:      a.AssetID.Bytes(),
		Amount:       a.Quantity,
		IsReissuable: a.Reissuable,
	}
}

// BurnScriptAction is an action to burn some assets in response to script invocation.
type BurnScriptAction struct {
	AssetID  crypto.Digest // assetId
	Quantity int64         // quantity
}

func (a BurnScriptAction) scriptAction() {}

func (a *BurnScriptAction) ToProtobuf() *g.InvokeScriptResult_Burn {
	return &g.InvokeScriptResult_Burn{
		AssetId: a.AssetID.Bytes(),
		Amount:  a.Quantity,
	}
}

type ScriptResult struct {
	DataEntries []DataEntryScriptAction
	Transfers   []TransferScriptAction
	Issues      []IssueScriptAction
	Reissues    []ReissueScriptAction
	Burns       []BurnScriptAction
}

// NewScriptResult creates correct representation of invocation actions for storage and API.
func NewScriptResult(actions []ScriptAction) (*ScriptResult, error) {
	entries := make([]DataEntryScriptAction, 0)
	transfers := make([]TransferScriptAction, 0)
	issues := make([]IssueScriptAction, 0)
	reissues := make([]ReissueScriptAction, 0)
	burns := make([]BurnScriptAction, 0)
	for _, a := range actions {
		switch ta := a.(type) {
		case DataEntryScriptAction:
			entries = append(entries, ta)
		case TransferScriptAction:
			transfers = append(transfers, ta)
		case IssueScriptAction:
			issues = append(issues, ta)
		case ReissueScriptAction:
			reissues = append(reissues, ta)
		case BurnScriptAction:
			burns = append(burns, ta)
		default:
			return nil, errors.Errorf("unsupported action type '%T'", a)
		}
	}
	return &ScriptResult{
		DataEntries: entries,
		Transfers:   transfers,
		Issues:      issues,
		Reissues:    reissues,
		Burns:       burns,
	}, nil
}

func (sr *ScriptResult) ToProtobuf() (*g.InvokeScriptResult, error) {
	data := make([]*g.DataTransactionData_DataEntry, len(sr.DataEntries))
	for i, e := range sr.DataEntries {
		data[i] = e.ToProtobuf()
	}
	transfers := make([]*g.InvokeScriptResult_Payment, len(sr.Transfers))
	for i, t := range sr.Transfers {
		transfers[i] = t.ToProtobuf()
	}
	issues := make([]*g.InvokeScriptResult_Issue, len(sr.Issues))
	for i, x := range sr.Issues {
		issues[i] = x.ToProtobuf()
	}
	reissues := make([]*g.InvokeScriptResult_Reissue, len(sr.Reissues))
	for i, r := range sr.Reissues {
		reissues[i] = r.ToProtobuf()
	}
	burns := make([]*g.InvokeScriptResult_Burn, len(sr.Burns))
	for i, b := range sr.Burns {
		burns[i] = b.ToProtobuf()
	}
	return &g.InvokeScriptResult{
		Data:      data,
		Transfers: transfers,
		Issues:    issues,
		Reissues:  reissues,
		Burns:     burns,
	}, nil
}

func (sr *ScriptResult) FromProtobuf(scheme byte, msg *g.InvokeScriptResult) error {
	if msg == nil {
		return errors.New("empty protobuf message")
	}
	c := client.SafeConverter{}
	data := make([]DataEntryScriptAction, len(msg.Data))
	for i, e := range msg.Data {
		de, err := c.Entry(e)
		if err != nil {
			return err
		}
		data[i] = DataEntryScriptAction{Entry: de}
	}
	sr.DataEntries = data
	var err error
	sr.Transfers, err = c.TransferScriptActions(scheme, msg.Transfers)
	if err != nil {
		return err
	}
	sr.Issues, err = c.IssueScriptActions(msg.Issues)
	if err != nil {
		return err
	}
	sr.Reissues, err = c.ReissueScriptActions(msg.Reissues)
	if err != nil {
		return err
	}
	sr.Burns, err = c.BurnScriptActions(msg.Burns)
	if err != nil {
		return err
	}
	return nil
}
