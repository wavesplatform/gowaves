package ride

import (
	"fmt"
	"math"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

var (
	// errDeletedEntry is returned when the entry was deleted in during the script execution.
	// Wrapped with proto.ErrNotFound to be compatible WrappedState.IsNotFound method.
	errDeletedEntry = errors.Wrap(proto.ErrNotFound, "entry has been deleted")
)

type WrappedState struct {
	diff                      diffState
	cle                       rideAddress
	scheme                    proto.Scheme
	height                    proto.Height
	act                       []proto.ScriptAction
	blocklist                 []proto.WavesAddress
	invocationCount           int
	dataEntriesSize           int
	rootScriptLibVersion      ast.LibraryVersion
	rootActionsCountValidator proto.ActionsCountValidator
	isLightNodeActivated      bool
}

func newWrappedState(
	env environment,
	originalStateForWrappedState types.EnrichedSmartState,
	rootScriptLibVersion ast.LibraryVersion,
) *WrappedState {
	return &WrappedState{
		diff:                      newDiffState(originalStateForWrappedState),
		cle:                       env.this().(rideAddress),
		scheme:                    env.scheme(),
		height:                    proto.Height(env.height()),
		rootScriptLibVersion:      rootScriptLibVersion,
		rootActionsCountValidator: proto.NewScriptActionsCountValidator(),
		isLightNodeActivated:      env.lightNodeActivated(),
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

func (ws *WrappedState) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	return ws.diff.state.RetrieveEntries(account)
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

func (ws *WrappedState) NewestRecipientToAddress(recipient proto.Recipient) (proto.WavesAddress, error) {
	return ws.diff.state.NewestRecipientToAddress(recipient)
}

func (ws *WrappedState) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	return ws.diff.state.NewestAddrByAlias(alias)
}

func (ws *WrappedState) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	addr, err := ws.NewestRecipientToAddress(account)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get full Waves balance from wrapped state")
	}
	b, err := ws.diff.loadWavesBalance(addr.ID())
	if err != nil {
		return 0, err
	}
	return b.checkedRegularBalance()
}

func (ws *WrappedState) uncheckedAssetBalance(addr proto.WavesAddress, assetID crypto.Digest) (int64, error) {
	key := assetBalanceKey{id: addr.ID(), asset: assetID}
	b, err := ws.diff.loadAssetBalance(key)
	if err != nil {
		return 0, err
	}
	return int64(b), nil
}

func (ws *WrappedState) NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
	addr, err := ws.NewestRecipientToAddress(account)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get asset balance from wrapped state")
	}
	key := assetBalanceKey{id: addr.ID(), asset: assetID}
	b, err := ws.diff.loadAssetBalance(key)
	if err != nil {
		return 0, err
	}
	return b.checked()
}

func (ws *WrappedState) uncheckedWavesBalance(addr proto.WavesAddress) (diffBalance, error) {
	return ws.diff.loadWavesBalance(addr.ID())
}

// NewestFullWavesBalance returns a full Waves balance of account.
// The method must be used ONLY in the Ride environment.
// The boundaries of the generating balance are calculated for the current height of applying block,
// instead of the last block height.
//
// For example, for the block validation we are use min effective balance of the account from height 1 to 1000.
// This function uses heights from 2 to 1001, where 1001 is the height of the applying block.
// All changes of effective balance during the applying block are affecting the generating balance.
func (ws *WrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := ws.NewestRecipientToAddress(account)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get full Waves balance from wrapped state")
	}
	b, err := ws.diff.loadWavesBalance(addr.ID())
	if err != nil {
		return nil, err
	}
	return b.toFullWavesBalance(ws.isLightNodeActivated)
}

func (ws *WrappedState) IsStateUntouched(account proto.Recipient) (bool, error) {
	return ws.diff.state.IsStateUntouched(account)
}

func (ws *WrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, address) {
		return nil, errDeletedEntry
	}

	if intDataEntry := ws.diff.findIntFromDataEntryByKey(key, address); intDataEntry != nil {
		return intDataEntry, nil
	}

	return ws.diff.state.RetrieveNewestIntegerEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, address) {
		return nil, errDeletedEntry
	}

	if boolDataEntry := ws.diff.findBoolFromDataEntryByKey(key, address); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBooleanEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, address) {
		return nil, errDeletedEntry
	}

	if stringDataEntry := ws.diff.findStringFromDataEntryByKey(key, address); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestStringEntry(account, key)
}

func (ws *WrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, address) {
		return nil, errDeletedEntry
	}

	if binaryDataEntry := ws.diff.findBinaryFromDataEntryByKey(key, address); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return ws.diff.state.RetrieveNewestBinaryEntry(account, key)
}

func (ws *WrappedState) isNewestDataEntryDeleted(key string, address proto.WavesAddress) bool {
	deletedDataEntry := ws.diff.findDeleteFromDataEntryByKey(key, address)
	return deletedDataEntry != nil
}

func (ws *WrappedState) NewestAssetIsSponsored(asset crypto.Digest) (bool, error) {
	assetID := proto.AssetIDFromDigest(asset)
	if sponsorship, ok := ws.diff.findSponsorshipByAssetID(assetID); ok {
		return sponsorship.minFee != 0, nil
	}
	return ws.diff.state.NewestAssetIsSponsored(asset)
}

func (ws *WrappedState) NewestAssetConstInfo(assetID proto.AssetID) (*proto.AssetConstInfo, error) {
	searchNewAsset := ws.diff.findNewAssetByAssetID(assetID)
	// it's an old asset which has been issued before current tx
	if searchNewAsset == nil {
		assetFromStore, err := ws.diff.state.NewestAssetConstInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get const asset's info from store")
		}
		return assetFromStore, nil
	}
	return &proto.AssetConstInfo{
		ID:          searchNewAsset.asset,
		Issuer:      searchNewAsset.dAppIssuer,
		IssueHeight: ws.height,
		Decimals:    uint8(searchNewAsset.decimals),
	}, nil
}

func (ws *WrappedState) NewestAssetInfo(asset crypto.Digest) (*proto.AssetInfo, error) {
	assetID := proto.AssetIDFromDigest(asset)
	searchNewAsset := ws.diff.findNewAssetByAssetID(assetID)
	// it's an old asset which has been issued before current tx
	if searchNewAsset == nil {
		assetFromStore, err := ws.diff.state.NewestAssetInfo(asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}
		if oldAssetFromDiff := ws.diff.findOldAssetByAssetID(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			assetFromStore.Quantity = uint64(quantity)
			return assetFromStore, nil
		}
		return assetFromStore, nil
	}
	// it's a new asset
	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}
	scripted := searchNewAsset.script != nil
	sponsored, err := ws.NewestAssetIsSponsored(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}
	return &proto.AssetInfo{
		AssetConstInfo: proto.AssetConstInfo{
			ID:          asset,
			IssueHeight: ws.height,
			Issuer:      searchNewAsset.dAppIssuer,
			Decimals:    uint8(searchNewAsset.decimals),
		},
		Quantity:        uint64(searchNewAsset.quantity),
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

func (ws *WrappedState) NewestFullAssetInfo(asset crypto.Digest) (*proto.FullAssetInfo, error) {
	assetID := proto.AssetIDFromDigest(asset)
	searchNewAsset := ws.diff.findNewAssetByAssetID(assetID)
	// it's an old asset which has been issued before current tx
	if searchNewAsset == nil {
		assetFromStore, err := ws.diff.state.NewestFullAssetInfo(asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}
		if oldAssetFromDiff := ws.diff.findOldAssetByAssetID(assetID); oldAssetFromDiff != nil {
			quantity := int64(assetFromStore.Quantity) + oldAssetFromDiff.diffQuantity

			if quantity >= 0 {
				assetFromStore.Quantity = uint64(quantity)
				return assetFromStore, nil
			}

			return nil, errors.Errorf("quantity of the asset is negative")
		}
		return assetFromStore, nil
	}
	// it's a new asset
	issuerPK, err := ws.NewestScriptPKByAddr(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuerPK from address in NewestAssetInfo")
	}

	scripted := searchNewAsset.script != nil

	sponsored, err := ws.NewestAssetIsSponsored(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find out sponsoring of the asset")
	}

	assetInfo := proto.AssetInfo{
		AssetConstInfo: proto.AssetConstInfo{
			ID:          asset,
			IssueHeight: ws.height,
			Issuer:      searchNewAsset.dAppIssuer,
			Decimals:    uint8(searchNewAsset.decimals),
		},
		Quantity:        uint64(searchNewAsset.quantity),
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}
	scriptInfo := proto.ScriptInfo{
		Bytes: searchNewAsset.script,
	}

	sponsorshipCost := int64(0)
	if sponsorship, ok := ws.diff.findSponsorshipByAssetID(assetID); ok {
		sponsorshipCost = sponsorship.minFee
	}

	return &proto.FullAssetInfo{
		AssetInfo:       assetInfo,
		Name:            searchNewAsset.name,
		Description:     searchNewAsset.description,
		ScriptInfo:      scriptInfo,
		SponsorshipCost: uint64(sponsorshipCost),
	}, nil
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

func (ws *WrappedState) NewestBlockInfoByHeight(height proto.Height) (*proto.BlockInfo, error) {
	return ws.diff.state.NewestBlockInfoByHeight(height)
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
	txID, err := crypto.NewDigestFromBytes(env.txID().(rideByteVector))
	if err != nil {
		return false, err
	}

	timestamp := env.timestamp()

	localEnv, err := NewEnvironment(
		env.scheme(),
		env.state(),
		env.internalPaymentsValidationHeight(),
		env.paymentsFixAfterHeight(),
		env.blockV5Activated(),
		env.rideV6Activated(),
		env.consensusImprovementsActivated(),
		env.blockRewardDistributionActivated(),
		env.lightNodeActivated(),
	)
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
	localEnv.setLastBlock(env.block())
	localEnv.SetLimit(MaxAssetVerifierComplexity(tree.LibVersion))
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
		return false, errs.NewTransactionNotAllowedByScript(fmt.Sprintf("asset script: %v", err), asset.ID.Bytes())
	}
	if !r.Result() {
		return false, errs.NewTransactionNotAllowedByScript("Script returned False", asset.ID.Bytes())
	}

	return true, nil
}

func (ws *WrappedState) validatePaymentAction(res *proto.AttachedPaymentScriptAction, env environment, restrictions proto.ActionsValidationRestrictions) error {
	if err := proto.ValidateAttachedPaymentScriptAction(res, restrictions, env.validateInternalPayments()); err != nil {
		return err
	}
	assetResult, err := ws.validateAsset(res, res.Asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	return nil
}

var errNegativeBalanceAfterPaymentsApplication = errors.New("negative balance after payments application")

func (ws *WrappedState) validateBalancesAfterPaymentsApplication(env environment, addr proto.WavesAddress, payments proto.ScriptPayments) error {
	for _, payment := range payments {
		var balance int64
		if payment.Asset.Present {
			var err error
			balance, err = ws.uncheckedAssetBalance(addr, payment.Asset.ID)
			if err != nil {
				return err
			}
		} else {
			fullBalance, err := ws.uncheckedWavesBalance(addr)
			if err != nil {
				return err
			}
			if env.rideV6Activated() {
				balance, err = fullBalance.spendableBalance()
				if err != nil {
					return err
				}
			} else {
				balance = fullBalance.balance
			}
		}
		if (env.validateInternalPayments() || env.rideV6Activated()) && balance < 0 {
			return errors.Wrapf(errNegativeBalanceAfterPaymentsApplication,
				"not enough money in the DApp, balance of asset %s on address %s after payments application is %d",
				payment.Asset.String(), addr.String(), balance)
		}
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
				sender.String(), balance, res.Asset.String(), fmt.Stringer(res.Recipient.Address()), res.Amount)
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

func (ws *WrappedState) invCount() int {
	return ws.invocationCount
}

func (ws *WrappedState) incrementInvCount() {
	ws.invocationCount++
}

func (ws *WrappedState) setInvocationCount(count int) {
	ws.invocationCount = count
}

func (ws *WrappedState) countActionTotal(action proto.ScriptAction, libVersion ast.LibraryVersion, isRideV6Activated bool) error {
	return ws.rootActionsCountValidator.CountAction(action, libVersion, isRideV6Activated)
}

func (ws *WrappedState) validateBalances(rideV6Activated bool) error {
	for changed := range ws.diff.changedAccounts {
		var err error
		if changed.asset.Present {
			err = ws.validateAssetBalance(changed.account, changed.asset.ID)
		} else {
			err = ws.validateWavesBalance(changed.account, rideV6Activated)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *WrappedState) validateWavesBalance(addrID proto.AddressID, rideV6Activated bool) error {
	diff, ok := ws.diff.wavesBalances[addrID]
	if !ok {
		addr, addrErr := addrID.ToWavesAddress(ws.scheme)
		if addrErr != nil {
			return errors.Wrap(addrErr, "failed to transform addrID to WavesAddress")
		}
		return errors.Errorf("failed to find waves balance diff by addr %q", addr)
	}
	if diff.balance < 0 {
		addr, err := addrID.ToWavesAddress(ws.scheme)
		if err != nil {
			return errors.Wrap(err, "failed to validate balances")
		}
		return errors.Errorf("the Waves balance of address %s is %d which is negative", addr.String(), diff.balance)
	}
	if rideV6Activated { // After activation of RideV6 we check that spendable balance is not negative
		_, err := diff.checkedSpendableBalance()
		if err != nil {
			addr, wErr := addrID.ToWavesAddress(ws.scheme)
			if wErr != nil {
				return errors.Wrap(wErr, "failed to validate balances")
			}
			return errors.Wrapf(err, "failed validation of address %s", addr.String())
		}
	}
	return nil
}

func (ws *WrappedState) validateAssetBalance(addrID proto.AddressID, asset crypto.Digest) error {
	key := assetBalanceKey{id: addrID, asset: asset}
	diff, ok := ws.diff.assetBalances[key]
	if !ok {
		addr, addrErr := addrID.ToWavesAddress(ws.scheme)
		if addrErr != nil {
			return errors.Wrap(addrErr, "failed to transform addrID to WavesAddress")
		}
		return errors.Errorf("failed to find asset %q balance diff by addr %q", asset, addr)
	}
	if _, err := diff.checked(); err != nil {
		addr, addrErr := addrID.ToWavesAddress(ws.scheme)
		if addrErr != nil {
			return errors.Wrap(addrErr, "failed to transform addrID to WavesAddress")
		}
		return errors.Wrapf(err, "failed validation of address %s of asset %s", addr.String(), asset.String())
	}
	return nil
}

func (ws *WrappedState) ApplyToState(
	actions []proto.ScriptAction,
	env environment,
	localActionsCountValidator *proto.ActionsCountValidator,
) ([]proto.ScriptAction, error) {
	currentLibVersion, err := env.libVersion() // get current version, for more info see usages of env.setLibVersion
	if err != nil {
		return nil, err
	}
	disableSelfTransfers := currentLibVersion >= ast.LibV4 // it's OK, this flag depends on library version, not feature
	restrictions := proto.ActionsValidationRestrictions{
		DisableSelfTransfers:  disableSelfTransfers,
		IsUTF16KeyLen:         !env.blockV5Activated(), // if RideV4 isn't activated,
		IsProtobufTransaction: env.isProtobufTx(),
		MaxDataEntriesSize:    env.maxDataEntriesSize(),
		Scheme:                ws.scheme,
		ScriptAddress:         ws.callee(),
	}
	var libVersion ast.LibraryVersion
	if env.blockRewardDistributionActivated() {
		libVersion = ws.rootScriptLibVersion
	} else {
		libVersion = currentLibVersion
	}
	if len(actions) == 0 {
		if err := ws.rootActionsCountValidator.ValidateCounts(libVersion, env.rideV6Activated()); err != nil {
			return nil, errors.Wrap(err, "failed to validate total actions count")
		}
	}
	for _, action := range actions {
		if err := localActionsCountValidator.CountAction(action, currentLibVersion, env.rideV6Activated()); err != nil {
			return nil, errors.Wrap(err, "failed to validate local actions count")
		}
		if err := ws.countActionTotal(action, libVersion, env.rideV6Activated()); err != nil {
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
			err = ws.validatePaymentAction(a, env, restrictions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			senderAddress, err := proto.NewAddressFromPublicKey(ws.scheme, *a.Sender)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			recipient, err := ws.NewestRecipientToAddress(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply attached payment")
			}
			// Self-payment causes no changes, can be ignored.
			if senderAddress == recipient {
				continue
			}
			// No balance validation done below
			if a.Asset.Present { // Update asset balance
				if err := ws.diff.assetTransfer(senderAddress.ID(), recipient.ID(), a.Asset.ID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply attached payment")
				}
			} else { // Update Waves balance
				if err := ws.diff.wavesTransfer(senderAddress.ID(), recipient.ID(), a.Amount); err != nil {
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
			recipient, err := ws.NewestRecipientToAddress(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply transfer action")
			}
			// Self-payment causes no changes, can be ignored.
			if senderAddress == recipient {
				continue
			}
			if a.Asset.Present { // Update asset balance
				if err := ws.diff.assetTransfer(senderAddress.ID(), recipient.ID(), a.Asset.ID, a.Amount); err != nil {
					return nil, errors.Wrap(err, "failed to apply transfer action")
				}
			} else { // Update Waves balance
				if err := ws.diff.wavesTransfer(senderAddress.ID(), recipient.ID(), a.Amount); err != nil {
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

			assetID := proto.AssetIDFromDigest(a.AssetID)
			sponsorship := diffSponsorship{
				minFee: a.MinFee,
			}
			ws.diff.setSponsorshipByAssetID(assetID, sponsorship)

		case *proto.IssueScriptAction:
			if err := ws.validateIssueAction(a); err != nil {
				return nil, errors.Wrapf(err, "failed to validate Issue action before application")
			}

			assetInfo := diffNewAssetInfo{
				asset:       a.ID,
				dAppIssuer:  ws.callee(),
				name:        a.Name,
				description: a.Description,
				quantity:    a.Quantity,
				decimals:    a.Decimals,
				reissuable:  a.Reissuable,
				script:      a.Script,
				nonce:       a.Nonce,
			}
			ws.diff.setNewAssetByAssetID(proto.AssetIDFromDigest(a.ID), assetInfo)

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
			assetID := proto.AssetIDFromDigest(a.AssetID)
			if searchNewAsset := ws.diff.findNewAssetByAssetID(assetID); searchNewAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAssetByAssetID(assetID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity += a.Quantity
					ws.diff.setOldAssetByAssetID(assetID, *oldAssetFromDiff)
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity += a.Quantity
				ws.diff.setOldAssetByAssetID(assetID, assetInfo)
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
			assetID := proto.AssetIDFromDigest(a.AssetID)
			if searchAsset := ws.diff.findNewAssetByAssetID(assetID); searchAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAssetByAssetID(assetID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity -= a.Quantity
					ws.diff.setOldAssetByAssetID(assetID, *oldAssetFromDiff)
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity -= a.Quantity
				ws.diff.setOldAssetByAssetID(assetID, assetInfo)
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

			receiver, err := ws.NewestRecipientToAddress(a.Recipient)
			if err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}

			if err := ws.diff.lease(senderAddress, receiver, a.Amount, a.ID); err != nil {
				return nil, errors.Wrap(err, "failed to apply Lease action")
			}

		case *proto.LeaseCancelScriptAction:
			pk, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			a.Sender = &pk

			l, err := ws.diff.loadLease(a.LeaseID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find lease by leaseID '%s'", a.LeaseID.String())
			}
			if !l.active {
				return nil, errors.Errorf("failed to cancel lease with leaseID '%s' because it's not actve", a.LeaseID.String())
			}
			if canceler := ws.callee(); canceler != l.sender {
				return nil, errors.Errorf(
					"attempt to cancel leasing that was created by other account; leaser '%s'; canceller '%s'; leasing: %s",
					l.sender.String(), canceler.String(), a.LeaseID.String(),
				)
			}

			if err := ws.diff.cancelLease(l.sender, l.recipient, l.amount, a.LeaseID); err != nil {
				return nil, errors.Wrap(err, "failed to apply LeaseCancel action")
			}

		default:
			return nil, errors.Errorf("unknown script action type %T", a)
		}
	}

	return actions, nil
}

type EvaluationEnvironment struct {
	sch                                proto.Scheme
	st                                 types.SmartState
	h                                  rideInt
	tx                                 rideType
	id                                 rideType
	th                                 rideType
	time                               uint64
	b                                  rideType
	check                              func(int) bool
	takeStr                            func(s string, n int) rideString
	inv                                rideType
	ver                                ast.LibraryVersion
	validatePaymentsAfter              proto.Height
	paymentsFixAfter                   proto.Height
	isBlockV5Activated                 bool
	isRideV6Activated                  bool
	isConsensusImprovementsActivated   bool // isConsensusImprovementsActivated => nodeVersion >= 1.4.12
	isBlockRewardDistributionActivated bool // isBlockRewardDistributionActivated => nodeVersion >= 1.4.16
	isLightNodeActivated               bool // isLightNodeActivated => nodeVersion >= 1.5.0
	isProtobufTransaction              bool
	mds                                int
	cc                                 complexityCalculator
}

func bytesSizeCheckV1V2(l int) bool {
	return l <= math.MaxInt32
}

func bytesSizeCheckV3V6(l int) bool {
	return l <= proto.MaxDataWithProofsBytes
}

func NewEnvironment(
	scheme proto.Scheme,
	state types.SmartState,
	internalPaymentsValidationHeight, paymentsFixAfterHeight proto.Height,
	blockV5, rideV6, consensusImprovements, blockRewardDistribution, lightNode bool,
) (*EvaluationEnvironment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	return &EvaluationEnvironment{
		sch:                                scheme,
		st:                                 state,
		h:                                  rideInt(height),
		check:                              bytesSizeCheckV1V2, // By default almost unlimited
		takeStr:                            func(s string, n int) rideString { panic("function 'takeStr' was not initialized") },
		validatePaymentsAfter:              internalPaymentsValidationHeight,
		paymentsFixAfter:                   paymentsFixAfterHeight,
		isBlockV5Activated:                 blockV5,
		isRideV6Activated:                  rideV6,
		isBlockRewardDistributionActivated: blockRewardDistribution,
		isLightNodeActivated:               lightNode,
		isConsensusImprovementsActivated:   consensusImprovements,
		cc:                                 newComplexityCalculatorByRideV6Activation(rideV6),
	}, nil
}

func NewEnvironmentWithWrappedState(
	env *EvaluationEnvironment,
	originalState types.EnrichedSmartState,
	payments proto.ScriptPayments,
	sender proto.WavesAddress,
	isProtobufTransaction bool,
	rootScriptLibVersion ast.LibraryVersion,
	checkSenderBalance bool,
) (*EvaluationEnvironment, error) {
	recipient := proto.WavesAddress(env.th.(rideAddress))
	st := newWrappedState(env, originalState, rootScriptLibVersion)
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
		if checkSenderBalance && senderBalance < payment.Amount {
			return nil, errors.Errorf("not enough money for tx attached payment #%d of asset '%s' with amount %d",
				i+1, payment.Asset.String(), payment.Amount)
		}
		if payment.Asset.Present { // Update asset balance
			if err := st.diff.assetTransfer(sender.ID(), recipient.ID(), payment.Asset.ID, int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to apply transfer action")
			}
		} else { // Update Waves balance
			if err := st.diff.wavesTransfer(sender.ID(), recipient.ID(), int64(payment.Amount)); err != nil {
				return nil, errors.Wrap(err, "failed to apply transfer action")
			}
		}
	}

	newEnv := *env
	newEnv.st = st
	newEnv.isProtobufTransaction = isProtobufTransaction
	return &newEnv, nil
}

func (e *EvaluationEnvironment) consensusImprovementsActivated() bool {
	return e.isConsensusImprovementsActivated
}

func (e *EvaluationEnvironment) blockRewardDistributionActivated() bool {
	return e.isBlockRewardDistributionActivated
}

func (e *EvaluationEnvironment) lightNodeActivated() bool {
	return e.isLightNodeActivated
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
	e.setLibVersion(v)
	if v > ast.LibV2 {
		e.check = bytesSizeCheckV3V6
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

func (e *EvaluationEnvironment) SetLastBlockFromBlockInfo(info *proto.BlockInfo) error {
	v, err := e.libVersion()
	if err != nil {
		return err
	}
	block := blockInfoToObject(info, v)
	e.setLastBlock(block)
	return nil
}

func (e *EvaluationEnvironment) setLastBlock(block rideType) {
	e.b = block
}

func (e *EvaluationEnvironment) SetTransactionFromScriptTransfer(transfer *proto.FullScriptTransfer) {
	e.id = rideByteVector(transfer.ID.Bytes())
	e.tx = scriptTransferToTransferTransactionObject(transfer)
}

func (e *EvaluationEnvironment) SetTransactionWithoutProofs(tx proto.Transaction) error {
	if err := e.SetTransaction(tx); err != nil {
		return err
	}
	if err := resetProofs(e.tx); err != nil {
		return err
	}
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
	e.id = rideByteVector(id)

	obj, err := transactionToObject(e, tx)
	if err != nil {
		return err
	}
	e.tx = obj
	return nil
}

func (e *EvaluationEnvironment) SetTransactionFromOrder(order proto.Order, v ast.LibraryVersion) error {
	obj, err := orderToObject(v, e.sch, order)
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

func (e *EvaluationEnvironment) SetLimit(limit uint32) {
	e.cc.setLimit(limit)
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

func (e *EvaluationEnvironment) transaction() rideType {
	return e.tx
}

func (e *EvaluationEnvironment) this() rideType {
	return e.th
}

func (e *EvaluationEnvironment) block() rideType {
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

func (e *EvaluationEnvironment) invocation() rideType {
	return e.inv
}

func (e *EvaluationEnvironment) setInvocation(inv rideType) {
	e.inv = inv
}

func (e *EvaluationEnvironment) setLibVersion(v ast.LibraryVersion) {
	e.ver = v
}

func (e *EvaluationEnvironment) libVersion() (ast.LibraryVersion, error) {
	if e.ver == 0 {
		return 0, errors.New("library version is uninitialized")
	}
	return e.ver, nil
}

func (e *EvaluationEnvironment) validateInternalPayments() bool {
	return int(e.h) > int(e.validatePaymentsAfter)
}

func (e *EvaluationEnvironment) internalPaymentsValidationHeight() proto.Height {
	return e.validatePaymentsAfter
}

func (e *EvaluationEnvironment) paymentsFixActivated() bool {
	// according to the logic of the scala node
	// see commit https://github.com/wavesplatform/Waves/commit/2f9e61c0fe04beeeb5b94b508b124a7ec1a746ff
	return int(e.h) >= int(e.paymentsFixAfter)
}

func (e *EvaluationEnvironment) paymentsFixAfterHeight() proto.Height {
	return e.paymentsFixAfter
}

func (e *EvaluationEnvironment) maxDataEntriesSize() int {
	return e.mds
}

func (e *EvaluationEnvironment) isProtobufTx() bool {
	return e.isProtobufTransaction
}

func (e *EvaluationEnvironment) complexityCalculator() complexityCalculator {
	return e.cc
}

func (e *EvaluationEnvironment) setComplexityCalculator(cc complexityCalculator) {
	e.cc = cc
}
