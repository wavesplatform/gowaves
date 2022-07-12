package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	errDeletedEntry = errors.New("entry has been deleted")
)

type WrappedState struct {
	diff                      diffState
	cle                       rideAddress
	scheme                    proto.Scheme
	act                       []proto.ScriptAction
	blocklist                 []proto.WavesAddress
	invocationCount           int
	totalComplexity           int
	dataEntriesSize           int
	rootScriptLibVersion      ast.LibraryVersion
	rootActionsCountValidator proto.ActionsCountValidator
}

func newWrappedState(env *EvaluationEnvironment, rootScriptLibVersion ast.LibraryVersion) *WrappedState {
	return &WrappedState{
		diff:                      newDiffState(env.st),
		cle:                       env.th.(rideAddress),
		scheme:                    env.sch,
		rootScriptLibVersion:      rootScriptLibVersion,
		rootActionsCountValidator: proto.NewScriptActionsCountValidator(),
	}
}

func (ws *WrappedState) appendActions(actions []proto.ScriptAction) {
	ws.act = append(ws.act, actions...)
}

func (ws *WrappedState) callee() proto.WavesAddress {
	return proto.WavesAddress(ws.cle)
}

func (ws *WrappedState) smartAppendActions(
	actions []proto.ScriptAction,
	env environment,
	localActionsCountValidator *proto.ActionsCountValidator,
) error {
	modifiedActions, err := ws.ApplyToState(actions, env, localActionsCountValidator)
	if err != nil {
		return err
	}
	ws.appendActions(modifiedActions)
	return nil
}

func (ws *WrappedState) AddingBlockHeight() (uint64, error) {
	return ws.diff.state.AddingBlockHeight()
}

func (ws *WrappedState) NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error) {
	return ws.diff.state.NewestLeasingInfo(id)
}

func (ws *WrappedState) NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error) {
	return ws.diff.state.NewestScriptPKByAddr(addr)
}

func (ws *WrappedState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	return ws.diff.state.NewestTransactionByID(id)
}

func (ws *WrappedState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	return ws.diff.state.NewestTransactionHeightByID(id)
}

func (ws *WrappedState) NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error) {
	return ws.diff.state.NewestScriptByAccount(account)
}

func (ws *WrappedState) NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error) {
	return ws.diff.state.NewestScriptBytesByAccount(account)
}

func (ws *WrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.WavesAddress, error) {
	return ws.diff.state.NewestRecipientToAddress(recipient)
}

func (ws *WrappedState) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	return ws.diff.state.NewestAddrByAlias(alias)
}

func (ws *WrappedState) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	id, err := ws.recipientToAddressID(account)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get full Waves balance from wrapped state")
	}
	b, err := ws.diff.loadWavesBalance(id)
	if err != nil {
		return 0, err
	}
	return b.checkedSpendableBalance()
}

func (ws *WrappedState) NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
	id, err := ws.recipientToAddressID(account)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get asset balance from wrapped state")
	}
	key := assetBalanceKey{id: id, asset: assetID}
	b, err := ws.diff.loadAssetBalance(key)
	if err != nil {
		return 0, err
	}
	return b.checked()
}

func (ws *WrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	id, err := ws.recipientToAddressID(account)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get full Waves balance from wrapped state")
	}
	b, err := ws.diff.loadWavesBalance(id)
	if err != nil {
		return nil, err
	}
	return b.toFullWavesBalance()
}

func (ws *WrappedState) IsStateUntouched(account proto.Recipient) (bool, error) {
	return ws.diff.state.IsStateUntouched(account)
}

func (ws *WrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, *address) {
		return nil, errDeletedEntry
	}

	if intDataEntry := ws.diff.findIntFromDataEntryByKey(key, *address); intDataEntry != nil {
		return intDataEntry, nil
	}

	return ws.diff.state.RetrieveNewestIntegerEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, *address) {
		return nil, errDeletedEntry
	}

	if boolDataEntry := ws.diff.findBoolFromDataEntryByKey(key, *address); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBooleanEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, *address) {
		return nil, errDeletedEntry
	}

	if stringDataEntry := ws.diff.findStringFromDataEntryByKey(key, *address); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestStringEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, *address) {
		return nil, errDeletedEntry
	}

	if binaryDataEntry := ws.diff.findBinaryFromDataEntryByKey(key, *address); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBinaryEntry(account, key)
}

func (ws *WrappedState) isNewestDataEntryDeleted(key string, address proto.WavesAddress) bool {
	deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address)
	return deletedDataEntry != nil
}

func (ws *WrappedState) NewestAssetIsSponsored(asset crypto.Digest) (bool, error) {
	if cost := ws.diff.findSponsorship(asset); cost != nil {
		if *cost == 0 {
			return false, nil
		}
		return true, nil
	}
	return ws.diff.state.NewestAssetIsSponsored(asset)
}

func (ws *WrappedState) NewestAssetInfo(asset crypto.Digest) (*proto.AssetInfo, error) {
	searchNewAsset := ws.diff.findNewAsset(asset)
	if searchNewAsset == nil {
		assetFromStore, err := ws.diff.state.NewestAssetInfo(asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}
		if oldAssetFromDiff := ws.diff.findOldAsset(asset); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			assetFromStore.Quantity = uint64(quantity)
			return assetFromStore, nil
		}
		return assetFromStore, nil
	}
	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}
	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}
	sponsored, err := ws.NewestAssetIsSponsored(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}
	return &proto.AssetInfo{
		ID:              asset,
		Quantity:        uint64(searchNewAsset.quantity),
		Decimals:        uint8(searchNewAsset.decimals),
		Issuer:          searchNewAsset.dAppIssuer,
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

func (ws *WrappedState) NewestFullAssetInfo(asset crypto.Digest) (*proto.FullAssetInfo, error) {
	searchNewAsset := ws.diff.findNewAsset(asset)

	if searchNewAsset == nil {
		assetFromStore, err := ws.diff.state.NewestFullAssetInfo(asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}
		if oldAssetFromDiff := ws.diff.findOldAsset(asset); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			if quantity >= 0 {
				assetFromStore.Quantity = uint64(quantity)
				return assetFromStore, nil
			}

			return nil, errors.Errorf("quantity of the asset is negative")
		}
		return assetFromStore, nil
	}

	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := false
	if searchNewAsset.script != nil {
		scripted = true
	}

	sponsored, err := ws.NewestAssetIsSponsored(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}

	assetInfo := proto.AssetInfo{
		ID:              asset,
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
	if sponsorship := ws.diff.findSponsorship(asset); sponsorship != nil {
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

func (ws *WrappedState) NewestScriptByAsset(asset crypto.Digest) (*ast.Tree, error) {
	return ws.diff.state.NewestScriptByAsset(asset)
}

func (ws *WrappedState) WavesBalanceProfile(id proto.AddressID) (*types.WavesBalanceProfile, error) {
	return ws.diff.state.WavesBalanceProfile(id)
}

func (ws *WrappedState) NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error) {
	return ws.diff.state.NewestAssetBalanceByAddressID(id, asset)
}

func (ws *WrappedState) validateAsset(action proto.ScriptAction, asset proto.OptionalAsset, env environment) (bool, error) {
	if !asset.Present {
		return true, nil
	}
	assetInfo, err := ws.NewestAssetInfo(asset.ID)
	if err != nil {
		return false, err
	}
	if !assetInfo.Scripted {
		return true, nil
	}
	txID, err := crypto.NewDigestFromBytes(env.txID().(rideBytes))
	if err != nil {
		return false, err
	}

	timestamp := env.timestamp()

	localEnv, err := NewEnvironment(env.scheme(), env.state(), env.internalPaymentsValidationHeight(), env.blockV5Activated(), env.rideV6Activated())
	if err != nil {
		return false, err
	}

	switch res := action.(type) {

	case *proto.TransferScriptAction:
		sender, err := proto.NewAddressFromPublicKey(localEnv.scheme(), *res.Sender)
		if err != nil {
			return false, err
		}

		fullTr := &proto.FullScriptTransfer{
			Amount:    uint64(res.Amount),
			Asset:     res.Asset,
			Recipient: res.Recipient,
			Sender:    sender,
			Timestamp: timestamp,
			ID:        &txID,
		}
		localEnv.SetTransactionFromScriptTransfer(fullTr)
	case *proto.AttachedPaymentScriptAction:
		sender, err := proto.NewAddressFromPublicKey(localEnv.scheme(), *res.Sender)
		if err != nil {
			return false, err
		}

		fullTr := &proto.FullScriptTransfer{
			Amount:    uint64(res.Amount),
			Asset:     res.Asset,
			Recipient: res.Recipient,
			Sender:    sender,
			Timestamp: timestamp,
			ID:        &txID,
		}
		localEnv.SetTransactionFromScriptTransfer(fullTr)

	case *proto.ReissueScriptAction, *proto.BurnScriptAction:
		err = localEnv.SetTransactionFromScriptAction(action, *action.SenderPK(), txID, timestamp)
		if err != nil {
			return false, err
		}

	}

	tree, err := ws.NewestScriptByAsset(asset.ID)
	if err != nil {
		return false, err
	}
	localEnv.ChooseSizeCheck(tree.LibVersion)
	switch tree.LibVersion {
	case ast.LibV1, ast.LibV2, ast.LibV3:
		assetInfo, err := ws.NewestAssetInfo(asset.ID)
		if err != nil {
			return false, err
		}
		localEnv.SetThisFromAssetInfo(assetInfo)
	default:
		assetInfo, err := ws.NewestFullAssetInfo(asset.ID)
		if err != nil {
			return false, err
		}
		localEnv.SetThisFromFullAssetInfo(assetInfo)
	}

	localEnv.ChooseTakeString(true)
	localEnv.ChooseMaxDataEntriesSize(true)

	r, err := CallVerifier(localEnv, tree)
	if err != nil {
		return false, errs.NewTransactionNotAllowedByScript(err.Error(), asset.ID.Bytes())
	}
	if !r.Result() {
		return false, errs.NewTransactionNotAllowedByScript("Script returned False", asset.ID.Bytes())
	}

	return true, nil
}

func (ws *WrappedState) validatePaymentAction(res *proto.AttachedPaymentScriptAction, sender proto.WavesAddress, env environment, restrictions proto.ActionsValidationRestrictions) error {
	assetResult, err := ws.validateAsset(res, res.Asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	if err := proto.ValidateAttachedPaymentScriptAction(res, restrictions, env.validateInternalPayments(), env.rideV6Activated()); err != nil {
		return err
	}
	senderRcp := proto.NewRecipientFromAddress(sender)
	var balance uint64
	if res.Asset.Present {
		balance, err = ws.NewestAssetBalance(senderRcp, res.Asset.ID)
	} else {
		if env.rideV6Activated() {
			allBalance, err := ws.NewestFullWavesBalance(senderRcp)
			if err != nil {
				return err
			}
			balance = allBalance.Available
		} else {
			balance, err = ws.NewestWavesBalance(senderRcp)
		}
	}
	if err != nil {
		return err
	}
	if (env.validateInternalPayments() || env.rideV6Activated()) && balance < uint64(res.Amount) {
		return errors.Errorf("not enough money in the DApp, balance of DApp with address %s is %d and it tried to transfer asset %s to %s, amount of %d",
			sender.String(), balance, res.Asset.String(), res.Recipient.Address.String(), res.Amount)
	}
	return nil
}

func (ws *WrappedState) validateTransferAction(res *proto.TransferScriptAction, restrictions proto.ActionsValidationRestrictions, sender proto.WavesAddress, env environment) error {
	assetResult, err := ws.validateAsset(res, res.Asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	if err := proto.ValidateTransferScriptAction(res, restrictions); err != nil {
		return err
	}
	var (
		balance   uint64
		senderRcp = proto.NewRecipientFromAddress(sender)
	)
	if res.Asset.Present {
		balance, err = ws.NewestAssetBalance(senderRcp, res.Asset.ID)
	} else {
		if env.rideV6Activated() {
			allBalance, err := ws.NewestFullWavesBalance(senderRcp)
			if err != nil {
				return err
			}
			balance = allBalance.Available
		} else {
			balance, err = ws.NewestWavesBalance(senderRcp)
		}
	}
	if err != nil {
		return err
	}
	if env.rideV6Activated() {
		if balance < uint64(res.Amount) {
			return errors.Errorf("not enough money in the DApp, balance of DApp with address %s is %d and it tried to transfer asset %s to %s, amount of %d",
				sender.String(), balance, res.Asset.String(), res.Recipient.Address.String(), res.Amount)
		}
	}
	return nil
}

func (ws *WrappedState) validateDataEntryAction(
	res *proto.DataEntryScriptAction,
	restrictions proto.ActionsValidationRestrictions,
	isRideV6Activated bool,
) error {
	newSize, err := proto.ValidateDataEntryScriptAction(res, restrictions, isRideV6Activated, ws.dataEntriesSize)
	if err != nil {
		return err
	}
	ws.dataEntriesSize = newSize
	return nil
}

func (ws *WrappedState) validateIssueAction(res *proto.IssueScriptAction) error {
	return proto.ValidateIssueScriptAction(res)
}

func (ws *WrappedState) validateReissueAction(res *proto.ReissueScriptAction, env environment) error {
	asset := proto.NewOptionalAssetFromDigest(res.AssetID)
	assetResult, err := ws.validateAsset(res, *asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	if err := proto.ValidateReissueScriptAction(res); err != nil {
		return err
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

func (ws *WrappedState) validateBurnAction(res *proto.BurnScriptAction, env environment) error {
	asset := proto.NewOptionalAssetFromDigest(res.AssetID)
	assetResult, err := ws.validateAsset(res, *asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	if err := proto.ValidateBurnScriptAction(res); err != nil {
		return err
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

func (ws *WrappedState) validateSponsorshipAction(res *proto.SponsorshipScriptAction) error {
	return proto.ValidateSponsorshipScriptAction(res)
}

func (ws *WrappedState) validateLeaseAction(res *proto.LeaseScriptAction, restrictions proto.ActionsValidationRestrictions) error {
	if err := proto.ValidateLeaseScriptAction(res, restrictions); err != nil {
		return err
	}
	balance, err := ws.NewestFullWavesBalance(proto.NewRecipientFromAddress(ws.callee()))
	if err != nil {
		return err
	}
	if balance.Available < uint64(res.Amount) {
		return errors.New("not enough money on the available balance of the account")
	}
	return nil
}

func (ws *WrappedState) getLibVersion() (ast.LibraryVersion, error) {
	tree, err := ws.NewestScriptByAccount(proto.NewRecipientFromAddress(ws.callee()))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get script by recipient")
	}
	return tree.LibVersion, nil
}

func (ws *WrappedState) invCount() int {
	return ws.invocationCount
}

func (ws *WrappedState) incrementInvCount() {
	ws.invocationCount++
}

func (ws *WrappedState) countActionTotal(action proto.ScriptAction, libVersion ast.LibraryVersion, isRideV6Activated bool) error {
	return ws.rootActionsCountValidator.CountAction(action, libVersion, isRideV6Activated)
}

func (ws *WrappedState) validateBalances(rideV6Activated bool) error {
	for id, diff := range ws.diff.wavesBalances {
		if diff.balance < 0 {
			addr, err := id.ToWavesAddress(ws.scheme)
			if err != nil {
				return errors.Wrap(err, "failed to validate balances")
			}
			return errors.Errorf("the Waves balance of address %s is %d which is negative", addr.String(), diff.balance)
		}
		if rideV6Activated { // After activation of RideV6 we check that spendable balance is not negative
			_, err := diff.checkedSpendableBalance()
			if err != nil {
				addr, err2 := id.ToWavesAddress(ws.scheme)
				if err2 != nil {
					return errors.Wrap(err, "failed to validate balances")
				}
				return errors.Wrapf(err, "failed validation of address %s", addr.String())
			}
		}
	}
	for k, b := range ws.diff.assetBalances {
		if _, err := b.checked(); err != nil {
			addr, err2 := k.id.ToWavesAddress(ws.scheme)
			if err2 != nil {
				return errors.Wrap(err, "failed to validate balances")
			}
			return errors.Wrapf(err, "failed validation of address %s of asset %s", addr.String(), k.asset.String())
		}
	}
	return nil
}

func (ws *WrappedState) ApplyToState(
	actions []proto.ScriptAction,
	env environment,
	localActionsCountValidator *proto.ActionsCountValidator,
) ([]proto.ScriptAction, error) {
	currentLibVersion, err := ws.getLibVersion()
	if err != nil {
		return nil, err
	}
	disableSelfTransfers := currentLibVersion >= ast.LibV4 // it's OK, this flag depends on library version, not feature
	restrictions := proto.ActionsValidationRestrictions{
		DisableSelfTransfers:  disableSelfTransfers,
		IsUTF16KeyLen:         !env.blockV5Activated(), // if RideV4 isn't activated,
		IsProtobufTransaction: env.isProtobufTx(),
		MaxDataEntriesSize:    env.maxDataEntriesSize(),
	}
	for _, action := range actions {
		if err := localActionsCountValidator.CountAction(action, currentLibVersion, env.rideV6Activated()); err != nil {
			return nil, errors.Wrap(err, "failed to validate local actions count")
		}
		if err := ws.countActionTotal(action, currentLibVersion, env.rideV6Activated()); err != nil {
			return nil, errors.Wrap(err, "failed to validate total actions count")
		}
		switch a := action.(type) {
		case *proto.DataEntryScriptAction:
			err := ws.validateDataEntryAction(a, restrictions, env.rideV6Activated())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of data entry action")
			}
			addr := ws.callee()
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK
			ws.diff.putDataEntry(a.Entry, addr)

		case *proto.AttachedPaymentScriptAction:
			senderAddress, err := proto.NewAddressFromPublicKey(ws.scheme, *a.Sender)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			err = ws.validatePaymentAction(a, senderAddress, env, restrictions) // TODO: Optimize double balance check inside this function and outside
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			recipientID, err := ws.recipientToAddressID(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			if a.Asset.Present { // Update asset balance
				if err := ws.diff.assetTransfer(senderAddress.ID(), recipientID, a.Asset.ID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
			} else { // Update Waves balance
				if err := ws.diff.wavesTransfer(senderAddress.ID(), recipientID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
			}

		case *proto.TransferScriptAction:
			// Update sender's Public Key in the action
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK
			senderAddress := ws.callee()
			if err = ws.validateTransferAction(a, restrictions, senderAddress, env); err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of transfer action")
			}
			recipientID, err := ws.recipientToAddressID(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply transfer action")
			}
			if a.Asset.Present { // Update asset balance
				if err := ws.diff.assetTransfer(senderAddress.ID(), recipientID, a.Asset.ID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply transfer action")
				}
			} else { // Update Waves balance
				if err := ws.diff.wavesTransfer(senderAddress.ID(), recipientID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply transfer action")
				}
			}

		case *proto.SponsorshipScriptAction:
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK

			err = ws.validateSponsorshipAction(a)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			sponsorship := diffSponsorship{
				minFee: a.MinFee,
			}
			ws.diff.sponsorships[a.AssetID] = sponsorship

		case *proto.IssueScriptAction:
			if err := ws.validateIssueAction(a); err != nil {
				return nil, errors.Wrapf(err, "failed to validate Issue action before application")
			}

			assetInfo := diffNewAssetInfo{
				dAppIssuer:  ws.callee(),
				name:        a.Name,
				description: a.Description,
				quantity:    a.Quantity,
				decimals:    a.Decimals,
				reissuable:  a.Reissuable,
				script:      a.Script,
				nonce:       a.Nonce,
			}
			ws.diff.newAssetsInfo[a.ID] = assetInfo

			// Update sender's Public Key in the action
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK

			key := assetBalanceKey{id: ws.callee().ID(), asset: a.ID}
			if _, err := ws.diff.loadAssetBalance(key); err != nil {
				return nil, errors.Wrap(err, "failed to apply Issue action")
			}
			if err = ws.diff.addAssetBalance(key, a.Quantity); err != nil {
				return nil, errors.Wrap(err, "failed to apply Issue action")
			}

		case *proto.ReissueScriptAction:
			// Update sender's Public Key in the action
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK

			err = ws.validateReissueAction(a, env)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of reissue action")
			}

			key := assetBalanceKey{id: ws.callee().ID(), asset: a.AssetID}
			if _, err := ws.diff.loadAssetBalance(key); err != nil {
				return nil, errors.Wrap(err, "failed to apply Reissue action")
			}
			if err := ws.diff.addAssetBalance(key, a.Quantity); err != nil {
				return nil, errors.Wrap(err, "failed to apply Reissue action")
			}

			// Update asset info
			// TODO: Simplify following logic, get rid of separate local storages for two kinds of asset info (old and new)
			if searchNewAsset := ws.diff.findNewAsset(a.AssetID); searchNewAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAsset(a.AssetID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity += a.Quantity
					ws.diff.oldAssetsInfo[a.AssetID] = *oldAssetFromDiff
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity += a.Quantity
				ws.diff.oldAssetsInfo[a.AssetID] = assetInfo
				break
			}
			ws.diff.reissueNewAsset(a.AssetID, a.Quantity, a.Reissuable)

		case *proto.BurnScriptAction:
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &senderPK

			if err = ws.validateBurnAction(a, env); err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of burn action")
			}

			key := assetBalanceKey{id: ws.callee().ID(), asset: a.AssetID}
			if _, err := ws.diff.loadAssetBalance(key); err != nil {
				return nil, errors.Wrap(err, "failed to apply Burn action")
			}
			if err := ws.diff.addAssetBalance(key, -a.Quantity); err != nil {
				return nil, errors.Wrap(err, "failed to apply Burn action")
			}

			// Update asset's info
			// TODO: Simplify following logic, get rid of two separate storages of asset infos
			if searchAsset := ws.diff.findNewAsset(a.AssetID); searchAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAsset(a.AssetID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity -= a.Quantity
					ws.diff.oldAssetsInfo[a.AssetID] = *oldAssetFromDiff
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity -= a.Quantity
				ws.diff.oldAssetsInfo[a.AssetID] = assetInfo
				break
			}
			ws.diff.burnNewAsset(a.AssetID, a.Quantity)

		case *proto.LeaseScriptAction:
			senderAddress := ws.callee()
			pk, err := ws.diff.state.NewestScriptPKByAddr(senderAddress)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &pk

			if err = ws.validateLeaseAction(a, restrictions); err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of lease action")
			}

			senderID := senderAddress.ID()
			receiverID, err := ws.recipientToAddressID(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}
			if _, err := ws.diff.loadWavesBalance(senderAddress.ID()); err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}
			if _, err := ws.diff.loadWavesBalance(receiverID); err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}
			if err := ws.diff.lease(senderID, receiverID, a.Amount); err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}

			ws.diff.addNewLease(a.Recipient, proto.NewRecipientFromAddress(senderAddress), a.Amount, a.ID)

		case *proto.LeaseCancelScriptAction:
			pk, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &pk

			searchLease, err := ws.diff.findLeaseByIDForCancel(a.LeaseID)
			if err != nil {
				return nil, errors.Errorf("failed to find lease by leaseID")
			}
			if searchLease == nil {
				return nil, errors.Errorf("there is no lease to cancel")
			}

			senderID, err := ws.recipientToAddressID(searchLease.Sender)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}
			receiverID, err := ws.recipientToAddressID(searchLease.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}
			if _, err := ws.diff.loadWavesBalance(senderID); err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}
			if _, err := ws.diff.loadWavesBalance(receiverID); err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}
			if err := ws.diff.cancelLease(senderID, receiverID, searchLease.leasedAmount); err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}

		default:
			return nil, errors.Errorf("unknown script action type %T", a)
		}
	}

	return actions, nil
}

func (ws *WrappedState) recipientToAddressID(recipient proto.Recipient) (proto.AddressID, error) {
	addr, err := ws.diff.state.NewestRecipientToAddress(recipient)
	if err != nil {
		return proto.AddressID{}, err
	}
	return addr.ID(), nil
}

type EvaluationEnvironment struct {
	sch                   proto.Scheme
	st                    types.SmartState
	h                     rideInt
	tx                    rideObject
	id                    rideType
	th                    rideType
	time                  uint64
	b                     rideObject
	check                 func(int) bool
	takeStr               func(s string, n int) rideString
	inv                   rideObject
	ver                   ast.LibraryVersion
	validatePaymentsAfter uint64
	isBlockV5Activated    bool
	isRideV6Activated     bool
	isProtobufTransaction bool
	mds                   int
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState, internalPaymentsValidationHeight uint64, blockV5, rideV6 bool) (*EvaluationEnvironment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	return &EvaluationEnvironment{
		sch:                   scheme,
		st:                    state,
		h:                     rideInt(height),
		check:                 func(int) bool { return true }, // By default, for versions below 2 there was no check, always ok.
		takeStr:               func(s string, n int) rideString { panic("function 'takeStr' was not initialized") },
		validatePaymentsAfter: internalPaymentsValidationHeight,
		isBlockV5Activated:    blockV5,
		isRideV6Activated:     rideV6,
	}, nil
}

func NewEnvironmentWithWrappedState(
	env *EvaluationEnvironment,
	payments proto.ScriptPayments,
	sender proto.WavesAddress,
	isProtobufTransaction bool,
	rootScriptLibVersion ast.LibraryVersion,
) (*EvaluationEnvironment, error) {
	recipient := proto.WavesAddress(env.th.(rideAddress))
	st := newWrappedState(env, rootScriptLibVersion)
	for i, payment := range payments {
		var (
			senderBalance uint64
			err           error
			callerRcp     = proto.NewRecipientFromAddress(sender)
		)
		// TODO: Move validation after application
		if payment.Asset.Present {
			senderBalance, err = st.NewestAssetBalance(callerRcp, payment.Asset.ID)
		} else {
			if env.isRideV6Activated {
				allBalance, err := st.NewestFullWavesBalance(callerRcp)
				if err != nil {
					return nil, err
				}
				senderBalance = allBalance.Available
			} else {
				senderBalance, err = st.NewestWavesBalance(callerRcp)
			}
		}
		if err != nil {
			return nil, err
		}
		if senderBalance < payment.Amount {
			return nil, errors.Errorf("not enough money for tx attached payment #%d of asset '%s' with amount %d",
				i+1, payment.Asset.String(), payment.Amount)
		}
		if payment.Asset.Present {
			senderKey := assetBalanceKey{id: sender.ID(), asset: payment.Asset.ID}
			if err := st.diff.addAssetBalance(senderKey, -int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
			recipientKey := assetBalanceKey{id: recipient.ID(), asset: payment.Asset.ID}
			if _, err := st.diff.loadAssetBalance(recipientKey); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
			if err := st.diff.addAssetBalance(recipientKey, int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
		} else {
			if err := st.diff.addWavesBalance(sender.ID(), -int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
			if _, err := st.diff.loadWavesBalance(recipient.ID()); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
			if err := st.diff.addWavesBalance(recipient.ID(), int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
			}
		}
	}

	return &EvaluationEnvironment{
		sch:                   env.sch,
		st:                    st,
		h:                     env.h,
		tx:                    env.tx,
		id:                    env.id,
		th:                    env.th,
		b:                     env.b,
		check:                 env.check,
		takeStr:               env.takeStr,
		inv:                   env.inv,
		validatePaymentsAfter: env.validatePaymentsAfter,
		mds:                   env.mds,
		isBlockV5Activated:    env.isBlockV5Activated,
		isRideV6Activated:     env.isRideV6Activated,
		isProtobufTransaction: isProtobufTransaction,
	}, nil
}

func (e *EvaluationEnvironment) rideV6Activated() bool {
	return e.isRideV6Activated
}

func (e *EvaluationEnvironment) blockV5Activated() bool {
	return e.isBlockV5Activated
}

func (e *EvaluationEnvironment) ChooseTakeString(isRideV5 bool) {
	e.takeStr = takeRideString
	if !isRideV5 {
		e.takeStr = takeRideStringWrong
	}
}

func (e *EvaluationEnvironment) ChooseSizeCheck(v ast.LibraryVersion) {
	e.ver = v
	if v > ast.LibV2 {
		e.check = func(l int) bool {
			return l <= maxMessageLength
		}
	}
}

func (e *EvaluationEnvironment) ChooseMaxDataEntriesSize(isRideV5 bool) {
	e.mds = proto.MaxDataEntriesScriptActionsSizeInBytesV1
	if isRideV5 {
		e.mds = proto.MaxDataEntriesScriptActionsSizeInBytesV2
	}
}

func (e *EvaluationEnvironment) SetThisFromFullAssetInfo(info *proto.FullAssetInfo) {
	e.th = fullAssetInfoToObject(info)
}

func (e *EvaluationEnvironment) SetTimestamp(timestamp uint64) {
	e.time = timestamp
}

func (e *EvaluationEnvironment) SetThisFromAssetInfo(info *proto.AssetInfo) {
	e.th = assetInfoToObject(info)
}

func (e *EvaluationEnvironment) SetThisFromAddress(addr proto.WavesAddress) {
	e.th = rideAddress(addr)
}

func (e *EvaluationEnvironment) SetLastBlock(info *proto.BlockInfo) {
	e.b = blockInfoToObject(info)
}

func (e *EvaluationEnvironment) SetTransactionFromScriptTransfer(transfer *proto.FullScriptTransfer) {
	e.id = rideBytes(transfer.ID.Bytes())
	e.tx = scriptTransferToTransferTransactionObject(transfer)
}

func (e *EvaluationEnvironment) SetTransactionWithoutProofs(tx proto.Transaction) error {
	err := e.SetTransaction(tx)
	if err != nil {
		return err
	}
	e.tx[proofsField] = rideUnit{}
	return nil
}

func (e *EvaluationEnvironment) SetTransactionFromScriptAction(action proto.ScriptAction, pk crypto.PublicKey, id crypto.Digest, ts uint64) error {
	obj, err := scriptActionToObject(e.sch, action, pk, id, ts)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *EvaluationEnvironment) SetTransaction(tx proto.Transaction) error {
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

func (e *EvaluationEnvironment) SetTransactionFromOrder(order proto.Order) error {
	obj, err := orderToObject(e.sch, order)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *EvaluationEnvironment) SetInvoke(tx proto.Transaction, v ast.LibraryVersion) error {
	obj, err := invocationToObject(v, e.sch, tx)
	if err != nil {
		return err
	}
	e.inv = obj

	return nil
}

func (e *EvaluationEnvironment) SetEthereumInvoke(tx *proto.EthereumTransaction, v ast.LibraryVersion, payments []proto.ScriptPayment) error {
	obj, err := ethereumInvocationToObject(v, e.sch, tx, payments)
	if err != nil {
		return err
	}
	e.inv = obj

	return nil
}

func (e *EvaluationEnvironment) timestamp() uint64 {
	return e.time
}

func (e *EvaluationEnvironment) scheme() byte {
	return e.sch
}

func (e *EvaluationEnvironment) height() rideInt {
	return e.h
}

func (e *EvaluationEnvironment) transaction() rideObject {
	return e.tx
}

func (e *EvaluationEnvironment) this() rideType {
	return e.th
}

func (e *EvaluationEnvironment) block() rideObject {
	return e.b
}

func (e *EvaluationEnvironment) txID() rideType {
	return e.id
}

func (e *EvaluationEnvironment) state() types.SmartState {
	return e.st
}

func (e *EvaluationEnvironment) setNewDAppAddress(address proto.WavesAddress) {
	ws, ok := e.st.(*WrappedState)
	if !ok {
		panic("not a WrappedState")
	}
	ws.cle = rideAddress(address)
	e.SetThisFromAddress(address)
}

func (e *EvaluationEnvironment) checkMessageLength(l int) bool {
	return e.check(l)
}

func (e *EvaluationEnvironment) takeString(s string, n int) rideString {
	return e.takeStr(s, n)
}

func (e *EvaluationEnvironment) invocation() rideObject {
	return e.inv
}

func (e *EvaluationEnvironment) setInvocation(inv rideObject) {
	e.inv = inv
}

func (e *EvaluationEnvironment) libVersion() ast.LibraryVersion {
	return e.ver
}

func (e *EvaluationEnvironment) validateInternalPayments() bool {
	return int(e.h) > int(e.validatePaymentsAfter)
}

func (e *EvaluationEnvironment) internalPaymentsValidationHeight() uint64 {
	return e.validatePaymentsAfter
}

func (e *EvaluationEnvironment) maxDataEntriesSize() int {
	return e.mds
}

func (e *EvaluationEnvironment) isProtobufTx() bool {
	return e.isProtobufTransaction
}
