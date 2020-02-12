package proto

import (
	"unicode/utf16"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

func (a DataEntryScriptAction) scriptAction() {}

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
		Address: a.Recipient.Address.Body(),
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
	for i := range sr.Transfers {
		transfers[i] = sr.Transfers[i].ToProtobuf()
	}
	issues := make([]*g.InvokeScriptResult_Issue, len(sr.Issues))
	for i := range sr.Issues {
		issues[i] = sr.Issues[i].ToProtobuf()
	}
	reissues := make([]*g.InvokeScriptResult_Reissue, len(sr.Reissues))
	for i := range sr.Reissues {
		reissues[i] = sr.Reissues[i].ToProtobuf()
	}
	burns := make([]*g.InvokeScriptResult_Burn, len(sr.Burns))
	for i := range sr.Burns {
		burns[i] = sr.Burns[i].ToProtobuf()
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
	c := ProtobufConverter{}
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

type ActionsValidationRestrictions struct {
	DisableSelfTransfers bool
	ScriptAddress        Address
}

func ValidateActions(actions []ScriptAction, restrictions ActionsValidationRestrictions) error {
	dataEntriesCount := 0
	dataEntriesSize := 0
	otherActionsCount := 0
	for _, a := range actions {
		switch ta := a.(type) {
		case DataEntryScriptAction:
			dataEntriesCount++
			if dataEntriesCount > maxDataEntryScriptActions {
				return errors.Errorf("number of data entries produced by script is more than allowed %d", maxDataEntryScriptActions)
			}
			if len(utf16.Encode([]rune(ta.Entry.GetKey()))) > maxKeySize {
				return errors.New("key is too large")
			}
			dataEntriesSize += ta.Entry.binarySize()
			if dataEntriesSize > maxDataEntryScriptActionsSizeInBytes {
				return errors.Errorf("total size of data entries produced by script is more than %d bytes", maxDataEntryScriptActionsSizeInBytes)
			}

		case TransferScriptAction:
			otherActionsCount++
			if otherActionsCount > maxScriptActions {
				return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
			}
			if ta.Amount < 0 {
				return errors.New("negative transfer amount")
			}
			if restrictions.DisableSelfTransfers {
				if ta.Recipient.Address.Eq(restrictions.ScriptAddress) {
					return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
				}
			}

		case IssueScriptAction:
			otherActionsCount++
			if otherActionsCount > maxScriptActions {
				return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
			}
			if ta.Quantity <= 0 {
				return errors.New("negative or zero quantity")
			}
			if ta.Decimals < 0 || ta.Decimals > maxDecimals {
				return errors.New("invalid decimals")
			}
			if l := len(ta.Name); l < minAssetNameLen || l > maxAssetNameLen {
				return errors.New("invalid asset's name")
			}
			if l := len(ta.Description); l > maxDescriptionLen {
				return errors.New("invalid asset's description")
			}

		case ReissueScriptAction:
			otherActionsCount++
			if otherActionsCount > maxScriptActions {
				return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
			}
			if ta.Quantity <= 0 {
				return errors.New("negative or zero quantity")
			}

		case BurnScriptAction:
			otherActionsCount++
			if otherActionsCount > maxScriptActions {
				return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
			}
			if ta.Quantity <= 0 {
				return errors.New("negative or zero quantity")
			}

		default:
			return errors.Errorf("unsupported script action type '%T'", a)
		}
	}
	return nil
}
