package proto

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type ScriptActionGroupType byte

const (
	DataScriptActionGroupType = iota + 1
	AttachedPaymentScriptActionGroupType
	AssetScriptActionGroupType
	BalanceScriptActionGroupType
)

// ScriptAction common interface of script invocation actions.
type ScriptAction interface {
	scriptAction()
	GroupType() ScriptActionGroupType
	SenderPK() *crypto.PublicKey
}

// DataEntryScriptAction is an action to manipulate account data state.
type DataEntryScriptAction struct {
	Sender *crypto.PublicKey
	Entry  DataEntry
}

func (a *DataEntryScriptAction) scriptAction() {}

func (a *DataEntryScriptAction) GroupType() ScriptActionGroupType {
	return DataScriptActionGroupType
}

func (a *DataEntryScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *DataEntryScriptAction) ToProtobuf() *g.DataTransactionData_DataEntry {
	return a.Entry.ToProtobuf()
}

type AttachedPaymentScriptAction struct {
	Sender    *crypto.PublicKey
	Recipient Recipient
	Amount    int64
	Asset     OptionalAsset
}

func (a *AttachedPaymentScriptAction) scriptAction() {}

func (a *AttachedPaymentScriptAction) GroupType() ScriptActionGroupType {
	return AttachedPaymentScriptActionGroupType
}

func (a *AttachedPaymentScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

// TransferScriptAction is an action to emit transfer of asset.
type TransferScriptAction struct {
	Sender    *crypto.PublicKey
	Recipient Recipient
	Amount    int64
	Asset     OptionalAsset
}

func (a *TransferScriptAction) scriptAction() {}

func (a *TransferScriptAction) GroupType() ScriptActionGroupType {
	return BalanceScriptActionGroupType
}

func (a *TransferScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *TransferScriptAction) ToProtobuf() (*g.InvokeScriptResult_Payment, error) {
	amount := &g.Amount{
		AssetId: a.Asset.ToID(),
		Amount:  a.Amount,
	}
	addrBody := a.Recipient.Address.Body()
	return &g.InvokeScriptResult_Payment{
		Address: addrBody,
		Amount:  amount,
	}, nil
}

// IssueScriptAction is an action to issue a new asset as a result of script invocation.
type IssueScriptAction struct {
	Sender      *crypto.PublicKey
	ID          crypto.Digest // calculated field
	Name        string        // name
	Description string        // description
	Quantity    int64         // quantity
	Decimals    int32         // decimals
	Reissuable  bool          // isReissuable
	Script      []byte        // compiledScript //TODO: reversed for future use
	Nonce       int64         // nonce
}

func (a *IssueScriptAction) scriptAction() {}

func (a *IssueScriptAction) GroupType() ScriptActionGroupType {
	return AssetScriptActionGroupType
}

func (a *IssueScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *IssueScriptAction) ToProtobuf() *g.InvokeScriptResult_Issue {
	return &g.InvokeScriptResult_Issue{
		AssetId:     a.ID.Bytes(),
		Name:        a.Name,
		Description: a.Description,
		Amount:      a.Quantity,
		Decimals:    a.Decimals,
		Reissuable:  a.Reissuable,
		Script:      nil, //TODO: in V4 is not used
		Nonce:       a.Nonce,
	}
}

// GenerateIssueScriptActionID implements ID generation used in RIDE to create new ID of Issue.
func GenerateIssueScriptActionID(name, description string, decimals, quantity int64, reissuable bool, nonce int64, txID crypto.Digest) crypto.Digest {
	nl := len(name)
	dl := len(description)
	buf := make([]byte, 4+nl+4+dl+4+8+2+8+crypto.DigestSize)
	pos := 0
	PutStringWithUInt32Len(buf[pos:], name)
	pos += 4 + nl
	PutStringWithUInt32Len(buf[pos:], description)
	pos += 4 + dl
	binary.BigEndian.PutUint32(buf[pos:], uint32(decimals))
	pos += 4
	binary.BigEndian.PutUint64(buf[pos:], uint64(quantity))
	pos += 8
	if reissuable {
		binary.BigEndian.PutUint16(buf[pos:], 1)
	} else {
		binary.BigEndian.PutUint16(buf[pos:], 0)
	}
	pos += 2
	binary.BigEndian.PutUint64(buf[pos:], uint64(nonce))
	pos += 8
	copy(buf[pos:], txID[:])
	return crypto.MustFastHash(buf)
}

// ReissueScriptAction is an action to emit Reissue transaction as a result of script invocation.
type ReissueScriptAction struct {
	Sender     *crypto.PublicKey
	AssetID    crypto.Digest // assetId
	Quantity   int64         // quantity
	Reissuable bool          // isReissuable
}

func (a *ReissueScriptAction) scriptAction() {}

func (a *ReissueScriptAction) GroupType() ScriptActionGroupType {
	return AssetScriptActionGroupType
}

func (a *ReissueScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *ReissueScriptAction) ToProtobuf() *g.InvokeScriptResult_Reissue {
	return &g.InvokeScriptResult_Reissue{
		AssetId:      a.AssetID.Bytes(),
		Amount:       a.Quantity,
		IsReissuable: a.Reissuable,
	}
}

// BurnScriptAction is an action to burn some assets in response to script invocation.
type BurnScriptAction struct {
	Sender   *crypto.PublicKey
	AssetID  crypto.Digest // assetId
	Quantity int64         // quantity
}

func (a *BurnScriptAction) scriptAction() {}

func (a *BurnScriptAction) GroupType() ScriptActionGroupType {
	return AssetScriptActionGroupType
}

func (a *BurnScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *BurnScriptAction) ToProtobuf() *g.InvokeScriptResult_Burn {
	return &g.InvokeScriptResult_Burn{
		AssetId: a.AssetID.Bytes(),
		Amount:  a.Quantity,
	}
}

// SponsorshipScriptAction is an action to set sponsorship for given asset in response to script invocation.
type SponsorshipScriptAction struct {
	Sender  *crypto.PublicKey
	AssetID crypto.Digest // assetId
	MinFee  int64         // minSponsoredAssetFee
}

func (a *SponsorshipScriptAction) scriptAction() {}

func (a *SponsorshipScriptAction) GroupType() ScriptActionGroupType {
	return AssetScriptActionGroupType
}

func (a *SponsorshipScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *SponsorshipScriptAction) ToProtobuf() *g.InvokeScriptResult_SponsorFee {
	return &g.InvokeScriptResult_SponsorFee{
		MinFee: &g.Amount{
			AssetId: a.AssetID.Bytes(),
			Amount:  a.MinFee,
		},
	}
}

// LeaseScriptAction is an action to lease Waves to given account.
type LeaseScriptAction struct {
	Sender    *crypto.PublicKey
	ID        crypto.Digest
	Recipient Recipient
	Amount    int64
	Nonce     int64
}

func (a *LeaseScriptAction) scriptAction() {}

func (a *LeaseScriptAction) GroupType() ScriptActionGroupType {
	return BalanceScriptActionGroupType
}

func (a *LeaseScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *LeaseScriptAction) ToProtobuf() (*g.InvokeScriptResult_Lease, error) {
	rcp, err := a.Recipient.ToProtobuf()
	if err != nil {
		return nil, err
	}
	return &g.InvokeScriptResult_Lease{
		Recipient: rcp,
		Amount:    a.Amount,
		Nonce:     a.Nonce,
		LeaseId:   a.ID.Bytes(),
	}, nil
}

// GenerateLeaseScriptActionID implements ID generation used in RIDE to create new ID for a Lease action.
func GenerateLeaseScriptActionID(recipient Recipient, amount int64, nonce int64, txID crypto.Digest) crypto.Digest {
	rl := WavesAddressSize
	if recipient.Alias != nil {
		rl = 4 + len(recipient.Alias.Alias)
	}
	buf := make([]byte, rl+crypto.DigestSize+8+8)
	pos := 0
	if recipient.Alias != nil {
		PutStringWithUInt32Len(buf[pos:], recipient.Alias.Alias)
	} else {
		copy(buf[pos:], recipient.Address[:])
	}
	pos += rl
	copy(buf[pos:], txID[:])
	pos += crypto.DigestSize
	binary.BigEndian.PutUint64(buf[pos:], uint64(nonce))
	pos += 8
	binary.BigEndian.PutUint64(buf[pos:], uint64(amount))
	return crypto.MustFastHash(buf)
}

// LeaseCancelScriptAction is an action that cancels previously created lease.
type LeaseCancelScriptAction struct {
	Sender  *crypto.PublicKey
	LeaseID crypto.Digest
}

func (a *LeaseCancelScriptAction) scriptAction() {}

func (a *LeaseCancelScriptAction) GroupType() ScriptActionGroupType {
	return BalanceScriptActionGroupType
}

func (a *LeaseCancelScriptAction) SenderPK() *crypto.PublicKey {
	return a.Sender
}

func (a *LeaseCancelScriptAction) ToProtobuf() *g.InvokeScriptResult_LeaseCancel {
	return &g.InvokeScriptResult_LeaseCancel{
		LeaseId: a.LeaseID.Bytes(),
	}
}

type ScriptErrorMessage struct {
	Code TxFailureReason
	Text string
}

func (msg *ScriptErrorMessage) ToProtobuf() *g.InvokeScriptResult_ErrorMessage {
	return &g.InvokeScriptResult_ErrorMessage{
		Code: int32(msg.Code),
		Text: msg.Text,
	}
}

type ScriptResult struct {
	DataEntries  []*DataEntryScriptAction
	Transfers    []*TransferScriptAction
	Issues       []*IssueScriptAction
	Reissues     []*ReissueScriptAction
	Burns        []*BurnScriptAction
	Sponsorships []*SponsorshipScriptAction
	Leases       []*LeaseScriptAction
	LeaseCancels []*LeaseCancelScriptAction
	ErrorMsg     ScriptErrorMessage
}

// NewScriptResult creates correct representation of invocation actions for storage and API.
func NewScriptResult(actions []ScriptAction, msg ScriptErrorMessage) (*ScriptResult, []*AttachedPaymentScriptAction, error) {
	entries := make([]*DataEntryScriptAction, 0)
	transfers := make([]*TransferScriptAction, 0)
	attachedPayments := make([]*AttachedPaymentScriptAction, 0)
	issues := make([]*IssueScriptAction, 0)
	reissues := make([]*ReissueScriptAction, 0)
	burns := make([]*BurnScriptAction, 0)
	sponsorships := make([]*SponsorshipScriptAction, 0)
	leases := make([]*LeaseScriptAction, 0)
	leaseCancels := make([]*LeaseCancelScriptAction, 0)

	for _, a := range actions {
		switch ta := a.(type) {
		case *DataEntryScriptAction:
			entries = append(entries, ta)
		case *TransferScriptAction:
			transfers = append(transfers, ta)
		case *AttachedPaymentScriptAction:
			attachedPayments = append(attachedPayments, ta)
		case *IssueScriptAction:
			issues = append(issues, ta)
		case *ReissueScriptAction:
			reissues = append(reissues, ta)
		case *BurnScriptAction:
			burns = append(burns, ta)
		case *SponsorshipScriptAction:
			sponsorships = append(sponsorships, ta)
		case *LeaseScriptAction:
			leases = append(leases, ta)
		case *LeaseCancelScriptAction:
			leaseCancels = append(leaseCancels, ta)
		default:
			return nil, nil, errors.Errorf("unsupported action type '%T'", a)
		}
	}
	return &ScriptResult{
		DataEntries:  entries,
		Transfers:    transfers,
		Issues:       issues,
		Reissues:     reissues,
		Burns:        burns,
		Sponsorships: sponsorships,
		Leases:       leases,
		LeaseCancels: leaseCancels,
		ErrorMsg:     msg,
	}, attachedPayments, nil
}

func (sr *ScriptResult) ToProtobuf() (*g.InvokeScriptResult, error) {
	data := make([]*g.DataTransactionData_DataEntry, len(sr.DataEntries))
	for i, e := range sr.DataEntries {
		data[i] = e.ToProtobuf()
	}
	transfers := make([]*g.InvokeScriptResult_Payment, len(sr.Transfers))
	var err error
	for i := range sr.Transfers {
		transfers[i], err = sr.Transfers[i].ToProtobuf()
		if err != nil {
			return nil, err
		}
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
	sponsorships := make([]*g.InvokeScriptResult_SponsorFee, len(sr.Sponsorships))
	for i := range sr.Sponsorships {
		sponsorships[i] = sr.Sponsorships[i].ToProtobuf()
	}
	leases := make([]*g.InvokeScriptResult_Lease, len(sr.Leases))
	for i := range sr.Leases {
		leases[i], err = sr.Leases[i].ToProtobuf()
		if err != nil {
			return nil, err
		}
	}
	leaseCancels := make([]*g.InvokeScriptResult_LeaseCancel, len(sr.LeaseCancels))
	for i := range sr.LeaseCancels {
		leaseCancels[i] = sr.LeaseCancels[i].ToProtobuf()
	}
	return &g.InvokeScriptResult{
		Data:         data,
		Transfers:    transfers,
		Issues:       issues,
		Reissues:     reissues,
		Burns:        burns,
		SponsorFees:  sponsorships,
		Leases:       leases,
		LeaseCancels: leaseCancels,
		ErrorMessage: sr.ErrorMsg.ToProtobuf(),
	}, nil
}

func (sr *ScriptResult) FromProtobuf(scheme byte, msg *g.InvokeScriptResult) error {
	if msg == nil {
		return errors.New("empty protobuf message")
	}
	c := ProtobufConverter{FallbackChainID: scheme}
	data := make([]*DataEntryScriptAction, len(msg.Data))
	for i, e := range msg.Data {
		de, err := c.Entry(e)
		if err != nil {
			return err
		}
		data[i] = &DataEntryScriptAction{Entry: de}
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
	sr.Sponsorships, err = c.SponsorshipScriptActions(msg.SponsorFees)
	if err != nil {
		return err
	}
	sr.Leases, err = c.LeaseScriptActions(scheme, msg.Leases)
	if err != nil {
		return err
	}
	sr.LeaseCancels, err = c.LeaseCancelScriptActions(msg.LeaseCancels)
	if err != nil {
		return err
	}
	errMsg, err := c.ErrorMessage(msg.ErrorMessage)
	if err != nil {
		return err
	}
	sr.ErrorMsg = *errMsg
	return nil
}

type ActionsValidationRestrictions struct {
	DisableSelfTransfers  bool
	ScriptAddress         WavesAddress
	IsUTF16KeyLen         bool
	IsProtobufTransaction bool
	MaxDataEntriesSize    int
	Scheme                byte
}

type ActionsCountValidator struct {
	dataScriptActionsCounter            int
	assetScriptActionsCounter           int
	balanceScriptActionsCounter         int
	attachedPaymentScriptActionsCounter int
}

func NewScriptActionsCountValidator() ActionsCountValidator {
	return ActionsCountValidator{
		dataScriptActionsCounter:            0,
		assetScriptActionsCounter:           0,
		balanceScriptActionsCounter:         0,
		attachedPaymentScriptActionsCounter: 0,
	}
}

func (v *ActionsCountValidator) CountAction(action ScriptAction, libVersion int, isRideV6Activated bool) error {
	switch groupType := action.GroupType(); groupType {
	case DataScriptActionGroupType:
		v.dataScriptActionsCounter++
		return v.validateDataEntryGroup()
	case AssetScriptActionGroupType:
		v.assetScriptActionsCounter++
		return v.validateAssetActionsGroup(libVersion)
	case BalanceScriptActionGroupType:
		v.balanceScriptActionsCounter++
		return v.validateBalanceActionsGroup(libVersion)
	case AttachedPaymentScriptActionGroupType:
		v.attachedPaymentScriptActionsCounter++
		return v.validateAttachedPaymentActionGroup(isRideV6Activated)
	default:
		return errors.Errorf("unknown script action group type (%d)", groupType)
	}
}

func (v *ActionsCountValidator) validateDataEntryGroup() error {
	if v.dataScriptActionsCounter > MaxDataEntryScriptActions {
		return errors.Errorf("number of data entries (%d) produced by script is more than allowed %d",
			v.dataScriptActionsCounter, MaxDataEntryScriptActions,
		)
	}
	return nil
}

func (v *ActionsCountValidator) validateAssetActionsGroup(libVersion int) error {
	switch {
	case libVersion < 5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV1 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV1,
			)
		}
	case libVersion == 5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV2 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV2,
			)
		}
	case libVersion > 5:
		if v.assetScriptActionsCounter > MaxAssetScriptActionsV3 {
			return errors.Errorf("number of issue group actions (%d) produced by script is more than allowed %d",
				v.assetScriptActionsCounter, MaxAssetScriptActionsV3,
			)
		}
	default:
		panic("ActionsCountValidator.validateAssetActionsGroup: unreachable point reached")
	}
	return nil
}

func (v *ActionsCountValidator) validateBalanceActionsGroup(libVersion int) error {
	switch {
	case libVersion < 5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV1 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV1,
			)
		}
	case libVersion == 5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV2 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV2,
			)
		}
	case libVersion > 5:
		if v.balanceScriptActionsCounter > MaxBalanceScriptActionsV3 {
			return errors.Errorf("number of transfer group actions (%d) produced by script is more than allowed %d",
				v.balanceScriptActionsCounter, MaxBalanceScriptActionsV3,
			)
		}
	default:
		panic("ActionsCountValidator.validateBalanceActionsGroup: unreachable point reached")
	}
	return nil
}

func (v *ActionsCountValidator) validateAttachedPaymentActionGroup(isRideV6Activated bool) error {
	if isRideV6Activated && v.attachedPaymentScriptActionsCounter > MaxAttachedPaymentsScriptActions {
		return errors.Errorf("number of attached payments (%d) produced by script is more than allowed %d",
			v.attachedPaymentScriptActionsCounter, MaxAttachedPaymentsScriptActions,
		)
	}
	return nil
}

func ValidateActions(actions []ScriptAction, restrictions ActionsValidationRestrictions, isRideV6Activated bool, libVersion int, validatePayments bool) error {
	var (
		dataEntriesSize       = 0
		actionsCountValidator = NewScriptActionsCountValidator()
	)
	for _, a := range actions {
		if err := actionsCountValidator.CountAction(a, libVersion, isRideV6Activated); err != nil {
			return errors.Wrap(err, "failed to validate actions count")
		}
		switch ta := a.(type) {
		case *DataEntryScriptAction:
			if err := ta.Entry.Valid(restrictions.IsProtobufTransaction, restrictions.IsUTF16KeyLen); err != nil {
				return err
			}
			if isRideV6Activated {
				dataEntriesSize += ta.Entry.PayloadSize()
			} else {
				dataEntriesSize += ta.Entry.BinarySize()
			}
			if dataEntriesSize > restrictions.MaxDataEntriesSize {
				return errors.Errorf("total size of data entries produced by script is more than %d bytes", restrictions.MaxDataEntriesSize)
			}

		case *TransferScriptAction:
			if ta.Amount < 0 {
				return errors.New("negative transfer amount")
			}
			if restrictions.DisableSelfTransfers {
				senderAddress := restrictions.ScriptAddress
				if ta.SenderPK() != nil {
					var err error
					senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *ta.SenderPK())
					if err != nil {
						return errors.Wrap(err, "failed to validate TransferScriptAction")
					}
				}
				if ta.Recipient.Address.Equal(senderAddress) {
					return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
				}
			}
		case *AttachedPaymentScriptAction:
			if validatePayments && ta.Amount < 0 {
				return errors.New("negative transfer amount")
			}
			if restrictions.DisableSelfTransfers {
				senderAddress := restrictions.ScriptAddress
				if ta.SenderPK() != nil {
					var err error
					senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *ta.SenderPK())
					if err != nil {
						return errors.Wrap(err, "failed to validate TransferScriptAction")
					}
				}
				if ta.Recipient.Address.Equal(senderAddress) {
					return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
				}
			}
		case *IssueScriptAction:
			if ta.Quantity < 0 {
				return errors.New("negative quantity")
			}
			if ta.Decimals < 0 || ta.Decimals > MaxDecimals {
				return errors.New("invalid decimals")
			}
			if l := len(ta.Name); l < MinAssetNameLen || l > MaxAssetNameLen {
				return errors.New("invalid asset's name")
			}
			if l := len(ta.Description); l > MaxDescriptionLen {
				return errors.New("invalid asset's description")
			}

		case *ReissueScriptAction:
			if ta.Quantity < 0 {
				return errors.New("negative quantity")
			}

		case *BurnScriptAction:
			if ta.Quantity < 0 {
				return errors.New("negative quantity")
			}

		case *SponsorshipScriptAction:
			if ta.MinFee < 0 {
				return errors.New("negative minimal fee")
			}

		case *LeaseScriptAction:
			if ta.Amount < 0 {
				return errors.New("negative leasing amount")
			}
			senderAddress := restrictions.ScriptAddress
			if ta.SenderPK() != nil {
				var err error
				senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *ta.SenderPK())
				if err != nil {
					return errors.Wrap(err, "failed to validate TransferScriptAction")
				}
			}
			if ta.Recipient.Address.Equal(senderAddress) {
				return errors.New("leasing to DApp itself is forbidden")
			}

		case *LeaseCancelScriptAction:
			// no-op
		default:
			return errors.Errorf("unsupported script action type '%T'", a)
		}
	}
	return nil
}
