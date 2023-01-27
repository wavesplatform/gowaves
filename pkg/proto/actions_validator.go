package proto

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

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

func (v *ActionsCountValidator) CountAction(action ScriptAction, libVersion ast.LibraryVersion, isRideV6Activated bool) error {
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

func (v *ActionsCountValidator) validateAssetActionsGroup(libVersion ast.LibraryVersion) error {
	switch {
	case libVersion < ast.LibV5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV1 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV1,
			)
		}
	case libVersion == ast.LibV5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV2 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV2,
			)
		}
	case libVersion > ast.LibV5:
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

func (v *ActionsCountValidator) validateBalanceActionsGroup(libVersion ast.LibraryVersion) error {
	switch {
	case libVersion < ast.LibV5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV1 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV1,
			)
		}
	case libVersion == ast.LibV5:
		if actionsCount := v.assetScriptActionsCounter + v.balanceScriptActionsCounter; actionsCount > MaxScriptActionsV2 {
			return errors.Errorf("number of actions (%d) produced by script is more than allowed %d",
				actionsCount, MaxScriptActionsV2,
			)
		}
	case libVersion > ast.LibV5:
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

func (v *ActionsCountValidator) ValidateCounts(libVersion ast.LibraryVersion, isRideV6Activated bool) error {
	err := v.validateDataEntryGroup()
	if err != nil {
		return err
	}
	err = v.validateBalanceActionsGroup(libVersion)
	if err != nil {
		return err
	}
	err = v.validateAssetActionsGroup(libVersion)
	if err != nil {
		return err
	}
	return v.validateAttachedPaymentActionGroup(isRideV6Activated)
}

func (v *ActionsCountValidator) validateAttachedPaymentActionGroup(isRideV6Activated bool) error {
	if isRideV6Activated && v.attachedPaymentScriptActionsCounter > MaxAttachedPaymentsScriptActions {
		return errors.Errorf("number of attached payments (%d) produced by script is more than allowed %d",
			v.attachedPaymentScriptActionsCounter, MaxAttachedPaymentsScriptActions,
		)
	}
	return nil
}

func ValidateAttachedPaymentScriptAction(action *AttachedPaymentScriptAction, restrictions ActionsValidationRestrictions, validatePayments bool) error {
	if validatePayments && action.Amount < 0 {
		return errors.New("negative transfer amount")
	}
	if restrictions.DisableSelfTransfers {
		senderAddress := restrictions.ScriptAddress
		if action.SenderPK() != nil {
			var err error
			senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *action.SenderPK())
			if err != nil {
				return errors.Wrap(err, "failed to validate AttachedPaymentScriptAction")
			}
		}
		eq, err := action.Recipient.EqAddr(senderAddress)
		if err != nil {
			return errors.Wrap(err, "failed to compare recipient with sender addr")
		}
		if eq {
			return errors.New("payments to DApp itself are forbidden since activation of RIDE V4")
		}
	}
	return nil
}

func ValidateIssueScriptAction(action *IssueScriptAction) error {
	if action.Quantity < 0 {
		return errors.New("negative quantity")
	}
	if action.Decimals < 0 || action.Decimals > MaxDecimals {
		return errors.New("invalid decimals")
	}
	if l := len(action.Name); l < MinAssetNameLen || l > MaxAssetNameLen {
		return errors.New("invalid asset's name")
	}
	if l := len(action.Description); l > MaxDescriptionLen {
		return errors.New("invalid asset's description")
	}
	return nil
}

func ValidateDataEntryScriptAction(action *DataEntryScriptAction, restrictions ActionsValidationRestrictions, isRideV6Activated bool, dataEntriesSize int) (int, error) {
	if err := action.Entry.Valid(restrictions.IsProtobufTransaction, restrictions.IsUTF16KeyLen); err != nil {
		return 0, err
	}
	if isRideV6Activated {
		dataEntriesSize += action.Entry.PayloadSize()
	} else {
		dataEntriesSize += action.Entry.BinarySize()
	}
	if dataEntriesSize > restrictions.MaxDataEntriesSize {
		return 0, errors.Errorf("total size of data entries produced by script is more than %d bytes", restrictions.MaxDataEntriesSize)
	}
	return dataEntriesSize, nil
}

func ValidateTransferScriptAction(action *TransferScriptAction, restrictions ActionsValidationRestrictions) error {
	if action.Amount < 0 {
		return errors.New("negative transfer amount")
	}
	if restrictions.DisableSelfTransfers {
		senderAddress := restrictions.ScriptAddress
		if action.SenderPK() != nil {
			var err error
			senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *action.SenderPK())
			if err != nil {
				return errors.Wrap(err, "failed to validate TransferScriptAction")
			}
		}
		eq, err := action.Recipient.EqAddr(senderAddress)
		if err != nil {
			return errors.Wrap(err, "failed to compare recipient with sender addr")
		}
		if eq {
			return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
		}
	}
	return nil
}

func ValidateReissueScriptAction(action *ReissueScriptAction) error {
	if action.Quantity < 0 {
		return errors.New("negative quantity")
	}
	return nil
}

func ValidateBurnScriptAction(action *BurnScriptAction) error {
	if action.Quantity < 0 {
		return errors.New("negative quantity")
	}
	return nil
}

func ValidateSponsorshipScriptAction(action *SponsorshipScriptAction) error {
	if action.MinFee < 0 {
		return errors.New("negative minimal fee")
	}
	return nil
}

func ValidateLeaseScriptAction(action *LeaseScriptAction, restrictions ActionsValidationRestrictions) error {
	if action.Amount < 0 {
		return errors.New("negative leasing amount")
	}
	senderAddress := restrictions.ScriptAddress
	if action.SenderPK() != nil {
		var err error
		senderAddress, err = NewAddressFromPublicKey(restrictions.Scheme, *action.SenderPK())
		if err != nil {
			return errors.Wrap(err, "failed to validate LeaseScriptAction")
		}
	}
	eq, err := action.Recipient.EqAddr(senderAddress)
	if err != nil {
		return errors.Wrap(err, "failed to compare recipient with sender addr")
	}
	if eq {
		return errors.New("leasing to DApp itself is forbidden")
	}
	return nil
}

func ValidateActions(
	actions []ScriptAction,
	restrictions ActionsValidationRestrictions,
	isRideV6Activated bool,
	libVersion ast.LibraryVersion,
	validatePayments bool,
) error {
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
			newSize, err := ValidateDataEntryScriptAction(ta, restrictions, isRideV6Activated, dataEntriesSize)
			if err != nil {
				return err
			}
			dataEntriesSize = newSize
		case *TransferScriptAction:
			if err := ValidateTransferScriptAction(ta, restrictions); err != nil {
				return err
			}
		case *AttachedPaymentScriptAction:
			if err := ValidateAttachedPaymentScriptAction(ta, restrictions, validatePayments); err != nil {
				return err
			}
		case *IssueScriptAction:
			if err := ValidateIssueScriptAction(ta); err != nil {
				return err
			}
		case *ReissueScriptAction:
			if err := ValidateReissueScriptAction(ta); err != nil {
				return err
			}
		case *BurnScriptAction:
			if err := ValidateBurnScriptAction(ta); err != nil {
				return err
			}
		case *SponsorshipScriptAction:
			if err := ValidateSponsorshipScriptAction(ta); err != nil {
				return err
			}
		case *LeaseScriptAction:
			if err := ValidateLeaseScriptAction(ta, restrictions); err != nil {
				return err
			}
		case *LeaseCancelScriptAction:
			// no-op
		default:
			return errors.Errorf("unsupported script action type '%T'", a)
		}
	}
	return nil
}
