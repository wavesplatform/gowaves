package ride

import (
	"unicode/utf16"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type WrappedState struct {
	diff diffState
	env  RideEnvironment
}

func newWrappedState(env *Environment) *WrappedState {
	dataEntries := diffDataEntries{
		diffInteger: map[string]proto.IntegerDataEntry{},
		diffBool:    map[string]proto.BooleanDataEntry{},
		diffString:  map[string]proto.StringDataEntry{},
		diffBinary:  map[string]proto.BinaryDataEntry{},
		diffDDelete: map[string]proto.DeleteDataEntry{},
	}
	diffSt := diffState{
		state:         env.st,
		dataEntries:   dataEntries,
		balances:      map[string]diffBalance{},
		sponsorships:  map[string]diffSponsorship{},
		newAssetsInfo: map[string]diffNewAssetInfo{},
		oldAssetsInfo: map[string]diffOldAssetInfo{},
		leases:        map[string]lease{}}

	return &WrappedState{diff: diffSt, env: env}
}

func (ws *WrappedState) AddingBlockHeight() (uint64, error) {
	return ws.diff.state.AddingBlockHeight()
}

func (ws *WrappedState) NewestLeasingInfo(id crypto.Digest, filter bool) (*proto.LeaseInfo, error) {
	return ws.diff.state.NewestLeasingInfo(id, filter)
}

func (ws *WrappedState) NewestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error) {
	return ws.diff.state.NewestScriptPKByAddr(addr, filter)
}
func (ws *WrappedState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	return ws.diff.state.NewestTransactionByID(id)
}
func (ws *WrappedState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	return ws.diff.state.NewestTransactionHeightByID(id)
}
func (ws *WrappedState) GetByteTree(recipient proto.Recipient) (proto.Script, error) {
	return ws.diff.state.GetByteTree(recipient)
}
func (ws *WrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	return ws.diff.state.NewestRecipientToAddress(recipient)
}

func (ws *WrappedState) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	return ws.diff.state.NewestAddrByAlias(alias)
}

func (ws *WrappedState) NewestAccountBalance(account proto.Recipient, assetID []byte) (uint64, error) {
	balance, err := ws.diff.state.NewestAccountBalance(account, assetID)
	if err != nil {
		return 0, err
	}
	var asset *proto.OptionalAsset

	if isAssetWaves(assetID) {
		waves := proto.NewOptionalAssetWaves()
		asset = &waves
	} else {
		asset, err = proto.NewOptionalAssetFromBytes(assetID)
		if err != nil {
			return 0, err
		}
	}

	balanceDiff, _, err := ws.diff.FindBalance(account, *asset)
	if err != nil {
		return 0, err
	}
	if balanceDiff != nil {
		resBalance := int64(balance) + balanceDiff.regular
		return uint64(resBalance), nil

	}
	return balance, nil
}

func (ws *WrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	balance, err := ws.diff.state.NewestFullWavesBalance(account)
	if err != nil {
		return nil, err
	}
	wavesBalanceDiff, searchAddress, err := ws.diff.FindBalance(account, proto.NewOptionalAssetWaves())
	if err != nil {
		return nil, err
	}
	if wavesBalanceDiff != nil {
		resRegular := wavesBalanceDiff.regular + int64(balance.Regular)
		resAvailable := (wavesBalanceDiff.regular - wavesBalanceDiff.leaseOut) + int64(balance.Available)
		resEffective := (wavesBalanceDiff.regular - wavesBalanceDiff.leaseOut + wavesBalanceDiff.leaseIn) + int64(balance.Effective)
		resLeaseIn := wavesBalanceDiff.leaseIn + int64(balance.LeaseIn)
		resLeaseOut := wavesBalanceDiff.leaseOut + int64(balance.LeaseOut)

		err := ws.diff.addEffectiveToHistory(searchAddress, resEffective)
		if err != nil {
			return nil, err
		}

		resGenerating := ws.diff.findMinGenerating(ws.diff.balances[searchAddress].effectiveHistory, int64(balance.Generating))

		return &proto.FullWavesBalance{
			Regular:    uint64(resRegular),
			Generating: uint64(resGenerating),
			Available:  uint64(resAvailable),
			Effective:  uint64(resEffective),
			LeaseIn:    uint64(resLeaseIn),
			LeaseOut:   uint64(resLeaseOut)}, nil

	}
	_, searchAddr := ws.diff.createNewWavesBalance(account)
	err = ws.diff.addEffectiveToHistory(searchAddr, int64(balance.Effective))
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (ws *WrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if intDataEntry := ws.diff.findIntFromDataEntryByKey(key, address.String()); intDataEntry != nil {
		return intDataEntry, nil
	}

	return ws.diff.state.RetrieveNewestIntegerEntry(account, key)
}
func (ws *WrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if boolDataEntry := ws.diff.findBoolFromDataEntryByKey(key, address.String()); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBooleanEntry(account, key)
}
func (ws *WrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if stringDataEntry := ws.diff.findStringFromDataEntryByKey(key, address.String()); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestStringEntry(account, key)
}
func (ws *WrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if binaryDataEntry := ws.diff.findBinaryFromDataEntryByKey(key, address.String()); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBinaryEntry(account, key)
}
func (ws *WrappedState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	if cost := ws.diff.findSponsorship(assetID); cost != nil {
		if *cost == 0 {
			return false, nil
		}
		return true, nil
	}
	return ws.diff.state.NewestAssetIsSponsored(assetID)
}
func (ws *WrappedState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	searchNewAsset := ws.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := ws.diff.state.NewestAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := ws.diff.findOldAsset(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			assetFromStore.Quantity = uint64(quantity)
			return assetFromStore, nil
		}

		return assetFromStore, nil
	}

	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}

	sponsored, err := ws.NewestAssetIsSponsored(assetID)
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
func (ws *WrappedState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	searchNewAsset := ws.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := ws.diff.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := ws.diff.findOldAsset(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			if quantity >= 0 {
				assetFromStore.Quantity = uint64(quantity)
				return assetFromStore, nil
			}

			return nil, errors.Errorf("quantity of the asset is negative")
		}

		return assetFromStore, nil
	}

	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}

	sponsored, err := ws.NewestAssetIsSponsored(assetID)
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
	if sponsorship := ws.diff.findSponsorship(assetID); sponsorship != nil {
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

func (ws *WrappedState) NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return ws.diff.state.NewestHeaderByHeight(height)
}
func (ws *WrappedState) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	return ws.diff.state.BlockVRF(blockHeader, height)
}

func (ws *WrappedState) EstimatorVersion() (int, error) {
	return ws.diff.state.EstimatorVersion()
}
func (ws *WrappedState) IsNotFound(err error) bool {
	return ws.diff.state.IsNotFound(err)
}

func (ws *WrappedState) validateTransferAction(otherActionsCount *int, res *proto.TransferScriptAction, restrictions proto.ActionsValidationRestrictions, sender proto.Address) error {
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
	senderRcp := proto.NewRecipientFromAddress(sender)
	balance, err := ws.NewestAccountBalance(senderRcp, res.Asset.ID.Bytes())
	if err != nil {
		return err
	}

	if balance < uint64(res.Amount) {
		return errors.Errorf("not enough money in the DApp. balance of DApp with address %s is %d and it tried to transfer asset %s to %s, amount of %d",
			sender.String(), balance, res.Asset.ID.String(), res.Recipient.Address.String(), res.Amount)
	}

	return nil
}

func (ws *WrappedState) validateDataEntryAction(dataEntriesCount *int, dataEntriesSize *int, res *proto.DataEntryScriptAction, restrictions proto.ActionsValidationRestrictions) error {
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

func (ws *WrappedState) validateIssueAction(otherActionsCount *int, res *proto.IssueScriptAction) error {
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

func (ws *WrappedState) validateReissueAction(otherActionsCount *int, res *proto.ReissueScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Quantity < 0 {
		return errors.New("negative quantity")
	}

	assetInfo, err := ws.NewestAssetInfo(res.AssetID)
	if err != nil {
		return err
	}

	if !assetInfo.Reissuable {
		return errors.New("failed to reissue asset as it's not reissuable anymore")
	}

	return nil
}

func (ws *WrappedState) validateBurnAction(otherActionsCount *int, res *proto.BurnScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.Quantity < 0 {
		return errors.New("negative quantity")
	}
	assetInfo, err := ws.NewestAssetInfo(res.AssetID)
	if err != nil {
		return err
	}

	if assetInfo.Quantity < uint64(res.Quantity) {
		return errors.New("quantity of asset is less than what was tried to burn")
	}

	return nil
}

func (ws *WrappedState) validateSponsorshipAction(otherActionsCount *int, res *proto.SponsorshipScriptAction) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	if res.MinFee < 0 {
		return errors.New("negative minimal fee")
	}

	return nil
}

func (ws *WrappedState) validateLeaseAction(otherActionsCount *int, res *proto.LeaseScriptAction, restrictions proto.ActionsValidationRestrictions) error {
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

	balance, err := ws.NewestFullWavesBalance(proto.NewRecipientFromAddress(ws.env.callee()))
	if err != nil {
		return err
	}
	if balance.Available < uint64(res.Amount) {
		return errors.New("not enough money on the available balance of the account")
	}
	return nil
}

func (ws *WrappedState) validateLeaseCancelAction(otherActionsCount *int) error {
	*otherActionsCount++
	if *otherActionsCount > proto.MaxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", proto.MaxScriptActions)
	}
	return nil
}

func (ws *WrappedState) getLibVersion() (int, error) {
	script, err := ws.GetByteTree(proto.NewRecipientFromAddress(ws.env.callee()))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get script by recipient")
	}
	tree, err := Parse(script)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get tree by script")
	}
	return tree.LibVersion, nil
}

func (ws *WrappedState) ApplyToState(actions []proto.ScriptAction) ([]proto.ScriptAction, error) {
	dataEntriesCount := 0
	dataEntriesSize := 0
	otherActionsCount := 0
	libVersion, err := ws.getLibVersion()
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
			err := ws.validateDataEntryAction(&dataEntriesCount, &dataEntriesSize, res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of data entry action")
			}

			switch dataEntry := res.Entry.(type) {

			case *proto.IntegerDataEntry:
				intEntry := *dataEntry
				addr := ws.env.callee()

				ws.diff.dataEntries.diffInteger[dataEntry.Key+addr.String()] = intEntry

				senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.StringDataEntry:
				stringEntry := *dataEntry
				addr := ws.env.callee()

				ws.diff.dataEntries.diffString[dataEntry.Key+addr.String()] = stringEntry

				senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.BooleanDataEntry:
				boolEntry := *dataEntry
				addr := ws.env.callee()

				ws.diff.dataEntries.diffBool[dataEntry.Key+addr.String()] = boolEntry

				senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.BinaryDataEntry:
				binaryEntry := *dataEntry
				addr := ws.env.callee()

				ws.diff.dataEntries.diffBinary[dataEntry.Key+addr.String()] = binaryEntry

				senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr, false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				res.Sender = &senderPK

			case *proto.DeleteDataEntry:
				deleteEntry := *dataEntry
				addr := ws.env.callee()

				ws.diff.dataEntries.diffDDelete[dataEntry.Key+addr.String()] = deleteEntry

				senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr, false)
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
				senderAddress, err = proto.NewAddressFromPublicKey(ws.env.scheme(), senderPK)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get address  by public key")
				}
			} else {
				pk, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get public key by address")
				}
				senderPK = pk
				senderAddress = ws.env.callee()
			}

			err = ws.validateTransferAction(&otherActionsCount, res, restrictions, senderAddress)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of transfer action or attached payments")
			}

			searchBalance, searchAddr, err := ws.diff.FindBalance(res.Recipient, res.Asset)
			if err != nil {
				return nil, err
			}
			err = ws.diff.ChangeBalance(searchBalance, searchAddr, res.Amount, res.Asset.ID, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderRecipient := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := ws.diff.FindBalance(senderRecipient, res.Asset)
			if err != nil {
				return nil, err
			}

			err = ws.diff.ChangeBalance(senderSearchBalance, senderSearchAddr, -res.Amount, res.Asset.ID, senderRecipient)
			if err != nil {
				return nil, err
			}

			res.Sender = &senderPK

		case *proto.SponsorshipScriptAction:
			err := ws.validateSponsorshipAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			var sponsorship diffSponsorship
			sponsorship.MinFee = res.MinFee

			ws.diff.sponsorships[res.AssetID.String()] = sponsorship

			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.IssueScriptAction:
			err := ws.validateIssueAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			var assetInfo diffNewAssetInfo
			assetInfo.dAppIssuer = ws.env.callee()
			assetInfo.name = res.Name
			assetInfo.description = res.Description
			assetInfo.quantity = res.Quantity
			assetInfo.decimals = res.Decimals
			assetInfo.reissuable = res.Reissuable
			assetInfo.script = res.Script
			assetInfo.nonce = res.Nonce

			ws.diff.newAssetsInfo[res.ID.String()] = assetInfo

			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			//TODO: Issue should create a diff
			//senderAddress := proto.Address(wrappedSt.EnvThis)
			//senderRecipient := proto.NewRecipientFromAddress(senderAddress)
			//asset := proto.NewOptionalAssetFromDigest(res.ID)
			//
			//searchBalance, searchAddr, err := wrappedSt.Diff.FindBalance(senderRecipient, *asset)
			//if err != nil {
			//	return nil, err
			//}
			//err = wrappedSt.Diff.ChangeBalance(searchBalance, searchAddr, res.Quantity, asset.ID, senderRecipient)
			//if err != nil {
			//	return nil, err
			//}

		case *proto.ReissueScriptAction:
			err := ws.validateReissueAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchNewAsset := ws.diff.findNewAsset(res.AssetID)
			if searchNewAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += res.Quantity

				ws.diff.oldAssetsInfo[res.AssetID.String()] = assetInfo
				break
			}
			ws.diff.reissueNewAsset(res.AssetID, res.Quantity, res.Reissuable)

			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.BurnScriptAction:
			err := ws.validateBurnAction(&otherActionsCount, res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchAsset := ws.diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += -res.Quantity

				ws.diff.oldAssetsInfo[res.AssetID.String()] = assetInfo

				break
			}
			ws.diff.burnNewAsset(res.AssetID, res.Quantity)

			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

		case *proto.LeaseScriptAction:
			err := ws.validateLeaseAction(&otherActionsCount, res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			senderAddress := ws.env.callee()

			recipientSearchBalance, recipientSearchAddress, err := ws.diff.FindBalance(res.Recipient, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeLeaseIn(recipientSearchBalance, recipientSearchAddress, res.Amount, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderAccount := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := ws.diff.FindBalance(senderAccount, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}

			err = ws.diff.changeLeaseOut(senderSearchBalance, senderSearchAddr, res.Amount, senderAccount)
			if err != nil {
				return nil, err
			}

			pk, err := ws.diff.state.NewestScriptPKByAddr(senderAddress, false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			ws.diff.addNewLease(res.Recipient, senderAccount, res.Amount, res.ID)

			res.Sender = &pk
		case *proto.LeaseCancelScriptAction:
			err := ws.validateLeaseCancelAction(&otherActionsCount)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			searchLease, err := ws.diff.findLeaseByIDForCancel(res.LeaseID)
			if err != nil {
				return nil, errors.Errorf("failed to find lease by leaseID")
			}
			if searchLease == nil {
				return nil, errors.Errorf("there is no lease to cancel")
			}

			recipientBalance, recipientSearchAddress, err := ws.diff.FindBalance(searchLease.Recipient, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			if recipientBalance == nil {
				_, recipientSearchAddress = ws.diff.createNewWavesBalance(searchLease.Recipient)
			}

			senderBalance, senderSearchAddress, err := ws.diff.FindBalance(searchLease.Sender, proto.NewOptionalAssetWaves())
			if err != nil {
				return nil, err
			}
			if senderBalance == nil {
				_, senderSearchAddress = ws.diff.createNewWavesBalance(searchLease.Sender)
			}

			ws.diff.cancelLease(*searchLease, senderSearchAddress, recipientSearchAddress)

			pk, err := ws.diff.state.NewestScriptPKByAddr(ws.env.callee(), false)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			res.Sender = &pk
		default:
		}
	}
	return actions, nil
}

type Environment struct {
	sch         proto.Scheme
	st          types.SmartState
	act         []proto.ScriptAction
	h           rideInt
	tx          rideObject
	id          rideType
	th          rideType
	cle         proto.Address
	b           rideObject
	check       func(int) bool
	inv         rideObject
	invokeCount uint64
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState) (*Environment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}

	return &Environment{
		sch:   scheme,
		st:    state,
		h:     rideInt(height),
		check: func(int) bool { return true },
	}, nil
}

func NewEnvironmentWithWrappedState(env *Environment, payments proto.ScriptPayments, callerPK crypto.PublicKey) (*Environment, error) {
	caller, err := proto.NewAddressFromPublicKey(env.sch, callerPK)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
	}
	recipient := proto.NewRecipientFromAddress(env.callee())

	st := newWrappedState(env)

	for _, payment := range payments {
		//senderBalance, err := wrappedSt.NewestAccountBalance(proto.NewRecipientFromAddress(caller), payment.Asset.ID.Bytes())
		//if err != nil {
		//	return err
		//}
		//if senderBalance < payment.Amount {
		//	return errors.New("not enough money for tx attached payments")
		//}

		searchBalance, searchAddr, err := st.diff.FindBalance(recipient, payment.Asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}
		err = st.diff.ChangeBalance(searchBalance, searchAddr, int64(payment.Amount), payment.Asset.ID, recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}

		callerRcp := proto.NewRecipientFromAddress(caller)
		senderSearchBalance, senderSearchAddr, err := st.diff.FindBalance(callerRcp, payment.Asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}

		err = st.diff.ChangeBalance(senderSearchBalance, senderSearchAddr, -int64(payment.Amount), payment.Asset.ID, callerRcp)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}
	}

	return &Environment{
		sch:         env.sch,
		st:          st,
		act:         env.act,
		h:           env.h,
		tx:          env.tx,
		id:          env.id,
		th:          env.th,
		cle:         env.callee(),
		b:           env.b,
		check:       env.check,
		inv:         env.inv,
		invokeCount: env.invokeCount,
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
	e.th = fullAssetInfoToObject(info)
}

func (e *Environment) SetThisFromAssetInfo(info *proto.AssetInfo) {
	e.th = assetInfoToObject(info)
}

func (e *Environment) SetThisFromAddress(addr proto.Address) {
	e.cle = addr
	e.th = rideAddress(addr)
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
	obj, err := scriptActionToObject(e.sch, action, pk, id, ts)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetTransaction(tx proto.Transaction) error {
	id, err := tx.GetID(e.sch)
	if err != nil {
		return err
	}
	e.id = rideBytes(id)
	obj, err := transactionToObject(e.sch, tx)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetTransactionFromOrder(order proto.Order) error {
	obj, err := orderToObject(e.sch, order)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *Environment) SetInvoke(tx *proto.InvokeScriptWithProofs, v int) error {
	obj, err := invocationToObject(v, e.sch, tx)
	if err != nil {
		return err
	}
	e.inv = obj
	return nil
}

func (e *Environment) scheme() byte {
	return e.sch
}

func (e *Environment) height() rideInt {
	return e.h
}

func (e *Environment) transaction() rideObject {
	return e.tx
}

func (e *Environment) this() rideType {
	return e.th
}

func (e *Environment) block() rideObject {
	return e.b
}

func (e *Environment) txID() rideType {
	return e.id
}

func (e *Environment) state() types.SmartState {
	return e.st
}

func (e *Environment) callee() proto.Address {
	return e.cle
}

func (e *Environment) actions() []proto.ScriptAction {
	return e.act
}

func (e *Environment) setNewDAppAddress(address proto.Address) {
	e.SetThisFromAddress(address)
}

func (e *Environment) applyToState(actions []proto.ScriptAction) ([]proto.ScriptAction, error) {
	return e.st.ApplyToState(actions)
}

func (e *Environment) appendActions(actions []proto.ScriptAction) {
	e.act = append(e.act, actions...)
}

func (e *Environment) smartAppendActions(actions []proto.ScriptAction) error {
	_, ok := e.st.(*WrappedState)
	if !ok {
		e.st = newWrappedState(e)
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

func isAssetWaves(assetID []byte) bool {
	wavesAsset := crypto.Digest{}
	if len(wavesAsset) != len(assetID) {
		return false
	}
	for i := range assetID {
		if assetID[i] != wavesAsset[i] {
			return false
		}
	}
	return true
}
