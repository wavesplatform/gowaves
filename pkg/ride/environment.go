package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"unicode/utf16"
)

func (wrappedSt *WrappedState) AddingBlockHeight() (uint64, error) {
	return wrappedSt.Diff.state.AddingBlockHeight()
}

func (wrappedSt *WrappedState) NewestLeasingInfo(id crypto.Digest, filter bool) (*proto.LeaseInfo, error) {
	return wrappedSt.Diff.state.NewestLeasingInfo(id, filter)
}

func (wrappedSt *WrappedState) NewestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error) {
	return wrappedSt.Diff.state.NewestScriptPKByAddr(addr, filter)
}
func (wrappedSt *WrappedState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	return wrappedSt.Diff.state.NewestTransactionByID(id)
}
func (wrappedSt *WrappedState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	return wrappedSt.Diff.state.NewestTransactionHeightByID(id)
}
func (wrappedSt *WrappedState) GetByteTree(recipient proto.Recipient) (proto.Script, error) {
	return wrappedSt.Diff.state.GetByteTree(recipient)
}
func (wrappedSt *WrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	return wrappedSt.Diff.state.NewestRecipientToAddress(recipient)
}

func (wrappedSt *WrappedState) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	return wrappedSt.Diff.state.NewestAddrByAlias(alias)
}

func (wrappedSt *WrappedState) NewestAccountBalance(account proto.Recipient, assetID []byte) (uint64, error) {
	balance, err := wrappedSt.Diff.state.NewestAccountBalance(account, assetID)
	if err != nil {
		return 0, err
	}

	asset, err := proto.NewOptionalAssetFromBytes(assetID)
	if err != nil {
		return 0, err
	}
	balanceDiff, _, err := wrappedSt.Diff.FindBalance(account, *asset)
	if err != nil {
		return 0, err
	}
	if balanceDiff != nil {
		resBalance := int64(balance) + balanceDiff.regular
		return uint64(resBalance), nil

	}
	return balance, nil
}

func (wrappedSt *WrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	balance, err := wrappedSt.Diff.state.NewestFullWavesBalance(account)
	if err != nil {
		return nil, err
	}
	wavesBalanceDiff, searchAddress, err := wrappedSt.Diff.FindBalance(account, proto.NewOptionalAssetWaves())
	if err != nil {
		return nil, err
	}
	if wavesBalanceDiff != nil {
		resRegular := wavesBalanceDiff.regular + int64(balance.Regular)
		resAvailable := (wavesBalanceDiff.regular - wavesBalanceDiff.leaseOut) + int64(balance.Available)
		resEffective := (wavesBalanceDiff.regular - wavesBalanceDiff.leaseOut + wavesBalanceDiff.leaseIn) + int64(balance.Effective)
		resLeaseIn := wavesBalanceDiff.leaseIn + int64(balance.LeaseIn)
		resLeaseOut := wavesBalanceDiff.leaseOut + int64(balance.LeaseOut)

		err := wrappedSt.Diff.addEffectiveToHistory(searchAddress, resEffective)
		if err != nil {
			return nil, err
		}

		resGenerating := wrappedSt.Diff.findMinGenerating(wrappedSt.Diff.balances[searchAddress].effectiveHistory, int64(balance.Generating))

		return &proto.FullWavesBalance{
			Regular:    uint64(resRegular),
			Generating: uint64(resGenerating),
			Available:  uint64(resAvailable),
			Effective:  uint64(resEffective),
			LeaseIn:    uint64(resLeaseIn),
			LeaseOut:   uint64(resLeaseOut)}, nil

	}
	_, searchAddr := wrappedSt.Diff.createNewWavesBalance(account)
	err = wrappedSt.Diff.addEffectiveToHistory(searchAddr, int64(balance.Effective))
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (wrappedSt *WrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := wrappedSt.Diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.Diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if intDataEntry := wrappedSt.Diff.findIntFromDataEntryByKey(key, address.String()); intDataEntry != nil {
		return intDataEntry, nil
	}

	return wrappedSt.Diff.state.RetrieveNewestIntegerEntry(account, key)
}
func (wrappedSt *WrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	address, err := wrappedSt.Diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.Diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if boolDataEntry := wrappedSt.Diff.findBoolFromDataEntryByKey(key, address.String()); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return wrappedSt.Diff.state.RetrieveNewestBooleanEntry(account, key)
}
func (wrappedSt *WrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	address, err := wrappedSt.Diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.Diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if stringDataEntry := wrappedSt.Diff.findStringFromDataEntryByKey(key, address.String()); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return wrappedSt.Diff.state.RetrieveNewestStringEntry(account, key)
}
func (wrappedSt *WrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	address, err := wrappedSt.Diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.Diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if binaryDataEntry := wrappedSt.Diff.findBinaryFromDataEntryByKey(key, address.String()); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return wrappedSt.Diff.state.RetrieveNewestBinaryEntry(account, key)
}
func (wrappedSt *WrappedState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	if cost := wrappedSt.Diff.findSponsorship(assetID); cost != nil {
		if *cost == 0 {
			return false, nil
		}
		return true, nil
	}
	return wrappedSt.Diff.state.NewestAssetIsSponsored(assetID)
}
func (wrappedSt *WrappedState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	searchNewAsset := wrappedSt.Diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.Diff.state.NewestAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := wrappedSt.Diff.findOldAsset(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			assetFromStore.Quantity = uint64(quantity)
			return assetFromStore, nil
		}

		return assetFromStore, nil
	}

	issuerPK, err := wrappedSt.NewestScriptPKByAddr(searchNewAsset.dAppIssuer, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}

	sponsored, err := wrappedSt.NewestAssetIsSponsored(assetID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}

	return &proto.AssetInfo{
		ID:              assetID,
		Quantity:        uint64(searchNewAsset.quantity),
		Decimals:        uint8(searchNewAsset.decimals),
		Issuer:          searchNewAsset.dAppIssuer,
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}
func (wrappedSt *WrappedState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	searchNewAsset := wrappedSt.Diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.Diff.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := wrappedSt.Diff.findOldAsset(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			if quantity >= 0 {
				assetFromStore.Quantity = uint64(quantity)
				return assetFromStore, nil
			}

			return nil, errors.Errorf("quantity of the asset is negative")
		}

		return assetFromStore, nil
	}

	issuerPK, err := wrappedSt.NewestScriptPKByAddr(searchNewAsset.dAppIssuer, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}

	sponsored, err := wrappedSt.NewestAssetIsSponsored(assetID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}

	assetInfo := proto.AssetInfo{
		ID:              assetID,
		Quantity:        uint64(searchNewAsset.quantity),
		Decimals:        uint8(searchNewAsset.decimals),
		Issuer:          searchNewAsset.dAppIssuer,
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}
	scriptInfo := proto.ScriptInfo{
		Bytes: searchNewAsset.script,
	}

	sponsorshipCost := int64(0)
	if sponsorship := wrappedSt.Diff.findSponsorship(assetID); sponsorship != nil {
		sponsorshipCost = *sponsorship
	}

	return &proto.FullAssetInfo{
		AssetInfo:       assetInfo,
		Name:            searchNewAsset.name,
		Description:     searchNewAsset.description,
		ScriptInfo:      scriptInfo,
		SponsorshipCost: uint64(sponsorshipCost),
	}, nil
}

func (wrappedSt *WrappedState) NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return wrappedSt.Diff.state.NewestHeaderByHeight(height)
}
func (wrappedSt *WrappedState) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	return wrappedSt.Diff.state.BlockVRF(blockHeader, height)
}

func (wrappedSt *WrappedState) EstimatorVersion() (int, error) {
	return wrappedSt.Diff.state.EstimatorVersion()
}
func (wrappedSt *WrappedState) IsNotFound(err error) bool {
	return wrappedSt.Diff.state.IsNotFound(err)
}

func (wrappedSt *WrappedState) validateTransferAction(otherActionsCount *int, res *proto.TransferScriptAction, restrictions proto.ActionsValidationRestrictions, senderAddress proto.Address) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Amount < 0 {
		return errors.New("negative transfer amount")
	}
	if res.InvalidAsset {
		return errors.New("invalid asset")
	}
	if restrictions.DisableSelfTransfers {
		senderAddress := restrictions.ScriptAddress
		if res.SenderPK() != nil {
			var err error
			senderAddress, err = proto.NewAddressFromPublicKey(restrictions.Scheme, *res.SenderPK())
			if err != nil {
				return errors.Wrap(err, "failed to validate TransferScriptAction")
			}
		}
		if res.Recipient.Address.Eq(senderAddress) {
			return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
		}
	}
	sender := proto.NewRecipientFromAddress(senderAddress)
	balance, err := wrappedSt.NewestAccountBalance(sender, res.Asset.ID.Bytes())
	if err != nil {
		return err
	}

	if balance < uint64(res.Amount) {
		return errors.New("Not enough money in the DApp")
	}

	return nil
}

func (wrappedSt *WrappedState) validateDataEntryAction(dataEntriesCount *int, dataEntriesSize *int, res *proto.DataEntryScriptAction, restrictions proto.ActionsValidationRestrictions) error {
	*dataEntriesCount++
	if *dataEntriesCount > proto.MaxDataEntryScriptActions {
		return errors.Errorf("number of data entries produced by script is more than allowed %d", proto.MaxDataEntryScriptActions)
	}
	switch restrictions.KeySizeValidationVersion {
	case 1:
		if len(utf16.Encode([]rune(res.Entry.GetKey()))) > proto.MaxKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	default:
		if len([]byte(res.Entry.GetKey())) > proto.MaxPBKeySize {
			return errs.NewTooBigArray("key is too large")
		}
	}

	*dataEntriesSize += res.Entry.BinarySize()
	if *dataEntriesSize > proto.MaxDataEntryScriptActionsSizeInBytes {
		return errors.Errorf("total size of data entries produced by script is more than %d bytes", proto.MaxDataEntryScriptActionsSizeInBytes)
	}
	return nil
}

func (wrappedSt *WrappedState) validateIssueAction(otherActionsCount *int, res *proto.IssueScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Quantity < 0 {
		return errors.New("negative quantity")
	}
	if res.Decimals < 0 || res.Decimals > proto.MaxDecimals {
		return errors.New("invalid decimals")
	}
	if l := len(res.Name); l < proto.MinAssetNameLen || l > proto.MaxAssetNameLen {
		return errors.New("invalid asset's name")
	}
	if l := len(res.Description); l > proto.MaxDescriptionLen {
		return errors.New("invalid asset's description")
	}
	return nil
}

func (wrappedSt *WrappedState) validateReissueAction(otherActionsCount *int, res *proto.ReissueScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Quantity < 0 {
		return errors.New("negative quantity")
	}

	assetInfo, err := wrappedSt.NewestAssetInfo(res.AssetID)
	if err != nil {
		return err
	}

	if !assetInfo.Reissuable {
		return errors.New("failed to reissue asset as it's not reissuable anymore")
	}

	return nil
}

func (wrappedSt *WrappedState) validateBurnAction(otherActionsCount *int, res *proto.BurnScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Quantity < 0 {
		return errors.New("negative quantity")
	}
	assetInfo, err := wrappedSt.NewestAssetInfo(res.AssetID)
	if err != nil {
		return err
	}

	if assetInfo.Quantity < uint64(res.Quantity) {
		return errors.New("quantity of asset is less than what was tried to burn")
	}

	return nil
}

func (wrappedSt *WrappedState) validateSponsorshipAction(otherActionsCount *int, res *proto.SponsorshipScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.MinFee < 0 {
		return errors.New("negative minimal fee")
	}

	return nil
}

func (wrappedSt *WrappedState) validateLeaseAction(otherActionsCount *int, res *proto.LeaseScriptAction, restrictions proto.ActionsValidationRestrictions) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Amount < 0 {
		return errors.New("negative leasing amount")
	}
	senderAddress := restrictions.ScriptAddress
	if res.SenderPK() != nil {
		var err error
		senderAddress, err = proto.NewAddressFromPublicKey(restrictions.Scheme, *res.SenderPK())
		if err != nil {
			return errors.Wrap(err, "failed to validate TransferScriptAction")
		}
	}
	if res.Recipient.Address.Eq(senderAddress) {
		return errors.New("leasing to DApp itself is forbidden")
	}

	currentDappAddress := proto.Address(wrappedSt.EnvThis)
	currentDapp := proto.NewRecipientFromAddress(currentDappAddress)
	balance, err := wrappedSt.NewestFullWavesBalance(currentDapp)
	if err != nil {
		return err
	}

	if balance.Available < uint64(res.Amount) {
		return errors.New("not enough money on the available balance of the account")
	}
	return nil
}

func (wrappedSt *WrappedState) validateLeaseCancelAction(otherActionsCount *int) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	return nil
}

func (wrappedSt *WrappedState) getLibVersion() (int, error) {
	currentDAppAddress := proto.Address(wrappedSt.EnvThis)
	currentDapp := proto.NewRecipientFromAddress(currentDAppAddress)
	dappScript, err := wrappedSt.GetByteTree(currentDapp)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get script by recipient")
	}
	tree, err := Parse(dappScript)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get tree by script")
	}
	return tree.LibVersion, nil
}

func (wrappedSt *WrappedState) ApplyToState(actions []proto.ScriptAction) ([]proto.ScriptAction, error) {
	dataEntriesCount := 0
	dataEntriesSize := 0
	otherActionsCount := 0
	libVersion, err := wrappedSt.getLibVersion()
	if err != nil {
		return nil, err
	}

	disableSelfTransfers := libVersion >= 4
	var keySizeValidationVersion byte = 1
	if libVersion >= 4 {
		keySizeValidationVersion = 2
	}
	restrictions := proto.ActionsValidationRestrictions{
		DisableSelfTransfers:     disableSelfTransfers,
		KeySizeValidationVersion: keySizeValidationVersion,
	}

	for _, action := range actions {
		switch res := action.(type) {

		case *proto.DataEntryScriptAction:
			err := wrappedSt.validateDataEntryAction(&dataEntriesCount, &dataEntriesSize, res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of data entry action")
			}

			switch dataEntry := res.Entry.(type) {

			case *proto.IntegerDataEntry:
				intEntry := *dataEntry
				addr := proto.Address(wrappedSt.EnvThis)

				wrappedSt.Diff.dataEntries.diffInteger[dataEntry.Key+addr.String()] = intEntry

				senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.StringDataEntry:
				stringEntry := *dataEntry
				addr := proto.Address(wrappedSt.EnvThis)

				wrappedSt.Diff.dataEntries.diffString[dataEntry.Key+addr.String()] = stringEntry

				senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.BooleanDataEntry:
				boolEntry := *dataEntry
				addr := proto.Address(wrappedSt.EnvThis)

				wrappedSt.Diff.dataEntries.diffBool[dataEntry.Key+addr.String()] = boolEntry

				senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.BinaryDataEntry:
				binaryEntry := *dataEntry
				addr := proto.Address(wrappedSt.EnvThis)

				wrappedSt.Diff.dataEntries.diffBinary[dataEntry.Key+addr.String()] = binaryEntry

				senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.DeleteDataEntry:
				deleteEntry := *dataEntry
				addr := proto.Address(wrappedSt.EnvThis)

				wrappedSt.Diff.dataEntries.diffDDelete[dataEntry.Key+addr.String()] = deleteEntry

				senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK
			default:

			}

		case *proto.TransferScriptAction:
			var senderAddress proto.Address
			var senderPK crypto.PublicKey
			if res.Sender != nil {
				senderPK = *res.Sender
				var err error
				senderAddress, err = proto.NewAddressFromPublicKey(wrappedSt.envScheme, senderPK)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get address  by public key")
				}
			} else {
				pk, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				senderPK = pk

				senderAddress = proto.Address(wrappedSt.EnvThis)
			}

			err := wrappedSt.validateTransferAction(&otherActionsCount, res, restrictions, senderAddress)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of transfer action or attached payments")
			}

			searchBalance, searchAddr, err := wrappedSt.Diff.FindBalance(res.Recipient, res.Asset)
			if err != nil {
				return nil, err
			}
			err = wrappedSt.Diff.ChangeBalance(searchBalance, searchAddr, res.Amount, res.Asset.ID, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderRecipient := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := wrappedSt.Diff.FindBalance(senderRecipient, res.Asset)
			if err != nil {
				return nil, err
			}

			err = wrappedSt.Diff.ChangeBalance(senderSearchBalance, senderSearchAddr, -res.Amount, res.Asset.ID, senderRecipient)
			if err != nil {
				return nil, err
			}

			res.Sender = &senderPK

		case *proto.SponsorshipScriptAction:
			err := wrappedSt.validateSponsorshipAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			var sponsorship diffSponsorship
			sponsorship.MinFee = res.MinFee

			wrappedSt.Diff.sponsorships[res.AssetID.String()] = sponsorship

			senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.IssueScriptAction:
			err := wrappedSt.validateIssueAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			var assetInfo diffNewAssetInfo
			assetInfo.dAppIssuer = proto.Address(wrappedSt.EnvThis)
			assetInfo.name = res.Name
			assetInfo.description = res.Description
			assetInfo.quantity = res.Quantity
			assetInfo.decimals = res.Decimals
			assetInfo.reissuable = res.Reissuable
			assetInfo.script = res.Script
			assetInfo.nonce = res.Nonce

			wrappedSt.Diff.newAssetsInfo[res.ID.String()] = assetInfo

			senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.ReissueScriptAction:
			err := wrappedSt.validateReissueAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchNewAsset := wrappedSt.Diff.findNewAsset(res.AssetID)
			if searchNewAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += res.Quantity

				wrappedSt.Diff.oldAssetsInfo[res.AssetID.String()] = assetInfo
				break
			}
			wrappedSt.Diff.reissueNewAsset(res.AssetID, res.Quantity, res.Reissuable)

			senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.BurnScriptAction:
			err := wrappedSt.validateBurnAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchAsset := wrappedSt.Diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += -res.Quantity

				wrappedSt.Diff.oldAssetsInfo[res.AssetID.String()] = assetInfo

				break
			}
			wrappedSt.Diff.burnNewAsset(res.AssetID, res.Quantity)

			senderPK, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.LeaseScriptAction:
			err := wrappedSt.validateLeaseAction(&otherActionsCount, res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			senderAddress := proto.Address(wrappedSt.EnvThis)

			recipientSearchBalance, recipientSearchAddress, err := wrappedSt.Diff.FindBalance(res.Recipient, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			err = wrappedSt.Diff.changeLeaseIn(recipientSearchBalance, recipientSearchAddress, res.Amount, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderAccount := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := wrappedSt.Diff.FindBalance(senderAccount, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}

			err = wrappedSt.Diff.changeLeaseOut(senderSearchBalance, senderSearchAddr, res.Amount, senderAccount)
			if err != nil {
				return nil, err
			}

			pk, err := wrappedSt.Diff.state.NewestScriptPKByAddr(senderAddress, false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			wrappedSt.Diff.addNewLease(res.Recipient, senderAccount, res.Amount, res.ID)

			res.Sender = &pk
		case *proto.LeaseCancelScriptAction:
			err := wrappedSt.validateLeaseCancelAction(&otherActionsCount)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchLease, err := wrappedSt.Diff.findLeaseByIDForCancel(res.LeaseID)
			if err != nil {
				return nil, errors.Errorf("failed to find lease by leaseID")
			}
			if searchLease == nil {
				return nil, errors.Errorf("there is no lease to cancel")
			}

			recipientBalance, recipientSearchAddress, err := wrappedSt.Diff.FindBalance(searchLease.Recipient, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			if recipientBalance == nil {
				_, recipientSearchAddress = wrappedSt.Diff.createNewWavesBalance(searchLease.Recipient)
			}

			senderBalance, senderSearchAddress, err := wrappedSt.Diff.FindBalance(searchLease.Sender, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			if senderBalance == nil {
				_, senderSearchAddress = wrappedSt.Diff.createNewWavesBalance(searchLease.Sender)
			}

			wrappedSt.Diff.cancelLease(*searchLease, senderSearchAddress, recipientSearchAddress)

			pk, err := wrappedSt.Diff.state.NewestScriptPKByAddr(proto.Address(wrappedSt.EnvThis), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			res.Sender = &pk
		default:
		}
	}
	return actions, nil
}

type WrappedState struct {
	Diff      diffState
	EnvThis   rideAddress
	envScheme proto.Scheme
}

type Environment struct {
	Sch         proto.Scheme
	St          types.SmartState
	act         []proto.ScriptAction
	h           rideInt
	tx          rideObject
	id          rideType
	Th          rideType
	b           rideObject
	check       func(int) bool
	inv         rideObject
	invokeCount uint64
	//isInternalPmnt bool
}

func NewWrappedState(state types.SmartState, envThis rideType, scheme proto.Scheme) types.SmartState {

	var dataEntries diffDataEntries

	dataEntries.diffInteger = map[string]proto.IntegerDataEntry{}
	dataEntries.diffBool = map[string]proto.BooleanDataEntry{}
	dataEntries.diffString = map[string]proto.StringDataEntry{}
	dataEntries.diffBinary = map[string]proto.BinaryDataEntry{}
	dataEntries.diffDDelete = map[string]proto.DeleteDataEntry{}

	balances := map[string]diffBalance{}
	sponsorships := map[string]diffSponsorship{}
	newAssetInfo := map[string]diffNewAssetInfo{}
	oldAssetInfo := map[string]diffOldAssetInfo{}
	leases := map[string]lease{}

	diffSt := &diffState{state: state, dataEntries: dataEntries, balances: balances, sponsorships: sponsorships, newAssetsInfo: newAssetInfo, oldAssetsInfo: oldAssetInfo, leases: leases}
	wrappedSt := WrappedState{Diff: *diffSt, EnvThis: envThis.(rideAddress), envScheme: scheme}

	return &wrappedSt
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState) (*Environment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}

	return &Environment{
		Sch:         scheme,
		St:          state,
		act:         nil,
		h:           rideInt(height),
		tx:          nil,
		id:          nil,
		Th:          nil,
		b:           nil,
		check:       func(int) bool { return true },
		inv:         nil,
		invokeCount: 0,
		//isInternalPmnt: false,
	}, nil
}

func (e *Environment) ChooseSizeCheck(v int) {
	if v > 2 {
		e.check = func(l int) bool {
			return l <= maxMessageLength
		}
	}
}

func (e *Environment) SetThisFromFullAssetInfo(info *proto.FullAssetInfo) {
	e.Th = fullAssetInfoToObject(info)
}

func (e *Environment) SetThisFromAssetInfo(info *proto.AssetInfo) {
	e.Th = assetInfoToObject(info)
}

func (e *Environment) SetThisFromAddress(addr proto.Address) {
	e.Th = rideAddress(addr)
}

func (e *Environment) SetLastBlock(info *proto.BlockInfo) {
	e.b = blockInfoToObject(info)
}

func (e *Environment) SetTransactionFromScriptTransfer(transfer *proto.FullScriptTransfer) {
	e.id = rideBytes(transfer.ID.Bytes())
	e.tx = scriptTransferToObject(transfer)
}

func (e *Environment) SetTransactionWithoutProofs(tx proto.Transaction) error {

	err := e.SetTransaction(tx)
	if err != nil {
		return err
	}
	e.tx["proofs"] = rideUnit{}
	return nil
}

func (e *Environment) SetTransactionFromScriptAction(action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, ts uint64) error {
	obj, err := scriptActionToObject(e.Sch, action, pk, id, ts)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetTransaction(tx proto.Transaction) error {
	id, err := tx.GetID(e.Sch)
	if err != nil {
		return err
	}
	e.id = rideBytes(id)
	obj, err := transactionToObject(e.Sch, tx)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetTransactionFromOrder(order proto.Order) error {
	obj, err := orderToObject(e.Sch, order)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetInvoke(tx *proto.InvokeScriptWithProofs, v int) error {
	obj, err := invocationToObject(v, e.Sch, tx)
	if err != nil {
		return err
	}
	e.inv = obj
	return nil
}

func (e *Environment) scheme() byte {
	return e.Sch
}

func (e *Environment) height() rideInt {
	return e.h
}

func (e *Environment) transaction() rideObject {
	return e.tx
}

func (e *Environment) this() rideType {
	return e.Th
}

func (e *Environment) block() rideObject {
	return e.b
}

func (e *Environment) txID() rideType {
	return e.id
}

func (e *Environment) state() types.SmartState {
	return e.St
}

func (e *Environment) actions() []proto.ScriptAction {
	return e.act
}

func (e *Environment) setNewDAppAddress(address proto.Address) {
	e.SetThisFromAddress(address)
}

func (e *Environment) applyToState(actions []proto.ScriptAction) ([]proto.ScriptAction, error) {
	return e.St.ApplyToState(actions)
}

func (e *Environment) appendActions(actions []proto.ScriptAction) {
	e.act = append(e.act, actions...)
}

func (e *Environment) smartAppendActions(actions []proto.ScriptAction) error {
	_, ok := e.St.(*WrappedState)
	if !ok {
		wrappedSt := NewWrappedState(e.state(), e.this(), e.Sch)
		e.St = wrappedSt
	}

	modifiedActions, err := e.applyToState(actions)
	if err != nil {
		return err
	}
	e.appendActions(modifiedActions)
	return nil
}

func (e *Environment) checkMessageLength(l int) bool {
	return e.check(l)
}

func (e *Environment) invocation() rideObject {
	return e.inv
}

func (e *Environment) SetInvocation(inv rideObject) {
	e.inv = inv
}

func (e *Environment) invCount() uint64 {
	return e.invokeCount
}

func (e *Environment) incrementInvCount() {
	e.invokeCount++
}
