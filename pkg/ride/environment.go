package ride

import (
	"unicode/utf16"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type WrappedState struct {
	diff             diffState
	cle              rideAddress
	scheme           proto.Scheme
	act              []proto.ScriptAction
	blocklist        []proto.WavesAddress
	invocationCount  int
	totalComplexity  int
	dataEntriesCount int
	dataEntriesSize  int
	actionsCount     int
}

func newWrappedState(env *EvaluationEnvironment) *WrappedState {
	return &WrappedState{
		diff:   newDiffState(env.st),
		cle:    env.th.(rideAddress),
		scheme: env.sch,
	}
}

func (ws *WrappedState) appendActions(actions []proto.ScriptAction) {
	ws.act = append(ws.act, actions...)
}

func (ws *WrappedState) callee() proto.WavesAddress {
	return proto.WavesAddress(ws.cle)
}

func (ws *WrappedState) smartAppendActions(actions []proto.ScriptAction, env environment) error {
	modifiedActions, err := ws.ApplyToState(actions, env)
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

func (ws *WrappedState) GetByteTree(recipient proto.Recipient) (proto.Script, error) {
	return ws.diff.state.GetByteTree(recipient)
}

func (ws *WrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.WavesAddress, error) {
	return ws.diff.state.NewestRecipientToAddress(recipient)
}

func (ws *WrappedState) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	return ws.diff.state.NewestAddrByAlias(alias)
}

func (ws *WrappedState) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	balance, err := ws.diff.state.NewestWavesBalance(account)
	if err != nil {
		return 0, err
	}
	balanceDiff, _, err := ws.diff.findBalance(account, proto.NewOptionalAssetWaves())
	if err != nil {
		return 0, err
	}
	if balanceDiff != nil {
		resBalance := int64(balance) + balanceDiff.regular
		return uint64(resBalance), nil

	}
	return balance, nil
}

func (ws *WrappedState) NewestAssetBalance(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
	balance, err := ws.diff.state.NewestAssetBalance(account, assetID)
	if err != nil {
		return 0, err
	}
	balanceDiff, _, err := ws.diff.findBalance(account, *proto.NewOptionalAssetFromDigest(assetID))
	if err != nil {
		return 0, err
	}
	if balanceDiff != nil {
		resBalance, err := common.AddInt64(int64(balance), balanceDiff.regular)
		if err != nil {
			return 0, err
		}
		return uint64(resBalance), nil

	}
	return balance, nil
}

// diff.regular - diff.leaseOut + available
func availableBalance(diffRegular int64, diffLeaseOut int64, available int64) (int64, error) {
	tmp, err := common.AddInt64(diffRegular, -diffLeaseOut)
	if err != nil {
		return 0, err
	}

	return common.AddInt64(tmp, available)
}

// diff.regular - diff.leaseOut + diff.leaseIn + effective
func effectiveBalance(diffRegular int64, diffLeaseOut int64, diffLeaseIn int64, available int64) (int64, error) {
	tmp, err := common.AddInt64(diffRegular, -diffLeaseOut)
	if err != nil {
		return 0, err
	}

	res, err := common.AddInt64(tmp, diffLeaseIn)
	if err != nil {
		return 0, err
	}

	return common.AddInt64(res, available)
}

func (ws *WrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	balance, err := ws.diff.state.NewestFullWavesBalance(account)
	if err != nil {
		return nil, err
	}
	wavesBalanceDiff, searchAddress, err := ws.diff.findBalance(account, proto.NewOptionalAssetWaves())
	if err != nil {
		return nil, err
	}
	if wavesBalanceDiff != nil {
		resRegular, err := common.AddInt64(wavesBalanceDiff.regular, int64(balance.Regular))
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate regular balance")
		}
		resAvailable, err := availableBalance(wavesBalanceDiff.regular, wavesBalanceDiff.leaseOut, int64(balance.Available))
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate available balance")
		}
		resEffective, err := effectiveBalance(wavesBalanceDiff.regular, wavesBalanceDiff.leaseOut, wavesBalanceDiff.leaseIn, int64(balance.Effective))
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate effective balance")
		}
		resLeaseIn, err := common.AddInt64(wavesBalanceDiff.leaseIn, int64(balance.LeaseIn))
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate lease in balance")
		}
		resLeaseOut, err := common.AddInt64(wavesBalanceDiff.leaseOut, int64(balance.LeaseOut))
		if err != nil {
			return nil, errors.Wrap(err, "failed to calculate lease out balance")
		}

		err = ws.diff.addEffectiveToHistory(searchAddress, resEffective)
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

func (ws *WrappedState) IsStateUntouched(account proto.Recipient) (bool, error) {
	return ws.diff.state.IsStateUntouched(account)
}

func (ws *WrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := ws.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}
	if ws.isNewestDataEntryDeleted(key, *address) {
		return nil, nil
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
		return nil, nil
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
		return nil, nil
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
		return nil, nil
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

func (ws *WrappedState) NewestScriptByAsset(asset crypto.Digest) (proto.Script, error) {
	return ws.diff.state.NewestScriptByAsset(asset)
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

	localEnv, err := NewEnvironment(env.scheme(), env.state(), env.internalPaymentsValidationHeight())
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

	script, err := ws.NewestScriptByAsset(asset.ID)
	if err != nil {
		return false, err
	}

	tree, err := Parse(script)
	if err != nil {
		return false, errors.Wrap(err, "failed to get tree by script")
	}

	localEnv.ChooseSizeCheck(tree.LibVersion)
	switch tree.LibVersion {
	case 4, 5:
		assetInfo, err := ws.NewestFullAssetInfo(asset.ID)
		if err != nil {
			return false, err
		}
		localEnv.SetThisFromFullAssetInfo(assetInfo)
	default:
		assetInfo, err := ws.NewestAssetInfo(asset.ID)
		if err != nil {
			return false, err
		}
		localEnv.SetThisFromAssetInfo(assetInfo)
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
	if res.Amount < 0 {
		return errors.New("negative transfer amount")
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
		if res.Recipient.Address.Equal(senderAddress) {
			return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
		}
	}
	senderRcp := proto.NewRecipientFromAddress(sender)
	var balance uint64
	if res.Asset.Present {
		balance, err = ws.NewestAssetBalance(senderRcp, res.Asset.ID)
	} else {
		balance, err = ws.NewestWavesBalance(senderRcp)
	}
	if err != nil {
		return err
	}
	if env.validateInternalPayments() && balance < uint64(res.Amount) {
		return errors.Errorf("not enough money in the DApp, balance of DApp with address %s is %d and it tried to transfer asset %s to %s, amount of %d",
			sender.String(), balance, res.Asset.String(), res.Recipient.Address.String(), res.Amount)
	}
	return nil
}

func (ws *WrappedState) validateTransferAction(res *proto.TransferScriptAction, restrictions proto.ActionsValidationRestrictions, sender proto.WavesAddress, env environment) error {
	ws.actionsCount++
	assetResult, err := ws.validateAsset(res, res.Asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
	}
	if res.Amount < 0 {
		return errors.New("negative transfer amount")
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
		if res.Recipient.Address.Equal(senderAddress) {
			return errors.New("transfers to DApp itself are forbidden since activation of RIDE V4")
		}
	}
	var (
		balance   uint64
		senderRcp = proto.NewRecipientFromAddress(sender)
	)
	if res.Asset.Present {
		balance, err = ws.NewestAssetBalance(senderRcp, res.Asset.ID)
	} else {
		balance, err = ws.NewestWavesBalance(senderRcp)
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

func (ws *WrappedState) validateDataEntryAction(res *proto.DataEntryScriptAction, restrictions proto.ActionsValidationRestrictions) error {
	ws.dataEntriesCount++
	if ws.dataEntriesCount > proto.MaxDataEntryScriptActions {
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

	ws.dataEntriesSize += res.Entry.BinarySize()
	if ws.dataEntriesSize > restrictions.MaxDataEntriesSize {
		return errors.Errorf("total size of data entries produced by script is more than %d bytes", restrictions.MaxDataEntriesSize)
	}
	return nil
}

func (ws *WrappedState) validateIssueAction(res *proto.IssueScriptAction) error {
	ws.actionsCount++
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
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

func (ws *WrappedState) validateReissueAction(res *proto.ReissueScriptAction, env environment) error {
	ws.actionsCount++
	asset := proto.NewOptionalAssetFromDigest(res.AssetID)
	assetResult, err := ws.validateAsset(res, *asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
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

func (ws *WrappedState) validateBurnAction(res *proto.BurnScriptAction, env environment) error {
	ws.actionsCount++
	asset := proto.NewOptionalAssetFromDigest(res.AssetID)
	assetResult, err := ws.validateAsset(res, *asset, env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate asset")
	}
	if !assetResult {
		return errors.New("action is forbidden by smart asset script")
	}
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
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

func (ws *WrappedState) validateSponsorshipAction(res *proto.SponsorshipScriptAction) error {
	ws.actionsCount++
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
	}
	if res.MinFee < 0 {
		return errors.New("negative minimal fee")
	}
	return nil
}

func (ws *WrappedState) validateLeaseAction(res *proto.LeaseScriptAction, restrictions proto.ActionsValidationRestrictions) error {
	ws.actionsCount++
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
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
	if res.Recipient.Address.Equal(senderAddress) {
		return errors.New("leasing to DApp itself is forbidden")
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

func (ws *WrappedState) validateLeaseCancelAction() error {
	ws.actionsCount++
	scriptVersion, err := ws.getLibVersion()
	if err != nil {
		return err
	}
	maxScriptActions := proto.GetMaxScriptActions(scriptVersion)
	if ws.actionsCount > maxScriptActions {
		return errors.Errorf("number of actions produced by script is more than allowed %d", maxScriptActions)
	}
	return nil
}

func (ws *WrappedState) getLibVersion() (int, error) {
	script, err := ws.GetByteTree(proto.NewRecipientFromAddress(ws.callee()))
	if err != nil {
		return 0, errors.Wrap(err, "failed to get script by recipient")
	}
	tree, err := Parse(script)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get tree by script")
	}
	return tree.LibVersion, nil
}

func (ws *WrappedState) invCount() int {
	return ws.invocationCount
}

func (ws *WrappedState) incrementInvCount() {
	ws.invocationCount++
}

func (ws *WrappedState) validateBalances() error {
	for key, balanceDiff := range ws.diff.balances {
		address := proto.NewRecipientFromAddress(key.address)
		var (
			balance uint64
			err     error
		)
		if key.asset.Present {
			balance, err = ws.diff.state.NewestAssetBalance(address, key.asset.ID)
		} else {
			balance, err = ws.diff.state.NewestWavesBalance(address)
		}
		if err != nil {
			return err
		}
		res, err := common.AddInt64(int64(balance), balanceDiff.regular)
		if err != nil {
			return err
		}
		if res < 0 {
			return errors.Errorf("the balance of address %s is %d which is negative", address.String(), int64(balance)+balanceDiff.regular)
		}
	}
	return nil
}

func (ws *WrappedState) ApplyToState(actions []proto.ScriptAction, env environment) ([]proto.ScriptAction, error) {
	libVersion, err := ws.getLibVersion()
	if err != nil {
		return nil, err
	}

	disableSelfTransfers := libVersion >= 4 // it's OK, this flag depends on library version, not feature
	var keySizeValidationVersion byte = 1
	if env.blockV5Activated() { // if RideV4 is activated
		keySizeValidationVersion = 2
	}
	restrictions := proto.ActionsValidationRestrictions{
		DisableSelfTransfers:     disableSelfTransfers,
		KeySizeValidationVersion: keySizeValidationVersion,
		MaxDataEntriesSize:       env.maxDataEntriesSize(),
	}

	for _, action := range actions {
		switch res := action.(type) {

		case *proto.DataEntryScriptAction:
			err := ws.validateDataEntryAction(res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of data entry action")
			}
			addr := ws.callee()
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(addr)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			if err := ws.diff.putDataEntry(res.Entry, addr); err != nil {
				return nil, err
			}

		case *proto.AttachedPaymentScriptAction:
			var senderAddress proto.WavesAddress
			var senderPK crypto.PublicKey
			senderPK = *res.Sender
			var err error
			senderAddress, err = proto.NewAddressFromPublicKey(ws.scheme, senderPK)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get address  by public key")
			}
			err = ws.validatePaymentAction(res, senderAddress, env, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of attached payments")
			}
			searchBalance, searchAddr, err := ws.diff.findBalance(res.Recipient, res.Asset)
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeBalance(searchBalance, searchAddr, res.Amount, res.Asset, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderRecipient := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := ws.diff.findBalance(senderRecipient, res.Asset)
			if err != nil {
				return nil, err
			}

			err = ws.diff.changeBalance(senderSearchBalance, senderSearchAddr, -res.Amount, res.Asset, senderRecipient)
			if err != nil {
				return nil, err
			}

		case *proto.TransferScriptAction:
			var senderAddress proto.WavesAddress
			var senderPK crypto.PublicKey

			pk, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			senderPK = pk
			senderAddress = ws.callee()

			res.Sender = &senderPK

			err = ws.validateTransferAction(res, restrictions, senderAddress, env)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of transfer action")
			}

			searchBalance, searchAddr, err := ws.diff.findBalance(res.Recipient, res.Asset)
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeBalance(searchBalance, searchAddr, res.Amount, res.Asset, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderRecipient := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := ws.diff.findBalance(senderRecipient, res.Asset)
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeBalance(senderSearchBalance, senderSearchAddr, -res.Amount, res.Asset, senderRecipient)
			if err != nil {
				return nil, err
			}

		case *proto.SponsorshipScriptAction:
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			err = ws.validateSponsorshipAction(res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			sponsorship := diffSponsorship{
				minFee: res.MinFee,
			}
			ws.diff.sponsorships[res.AssetID] = sponsorship

		case *proto.IssueScriptAction:
			err := ws.validateIssueAction(res)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of issue action")
			}

			assetInfo := diffNewAssetInfo{
				dAppIssuer:  ws.callee(),
				name:        res.Name,
				description: res.Description,
				quantity:    res.Quantity,
				decimals:    res.Decimals,
				reissuable:  res.Reissuable,
				script:      res.Script,
				nonce:       res.Nonce,
			}
			ws.diff.newAssetsInfo[res.ID] = assetInfo

			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			senderRcp := proto.NewRecipientFromAddress(ws.callee())
			asset := *proto.NewOptionalAssetFromDigest(res.ID)
			searchBalance, searchAddr, err := ws.diff.findBalance(senderRcp, asset)
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeBalance(searchBalance, searchAddr, res.Quantity, asset, senderRcp)
			if err != nil {
				return nil, err
			}

		case *proto.ReissueScriptAction:
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			err = ws.validateReissueAction(res, env)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of reissue action")
			}

			senderRcp := proto.NewRecipientFromAddress(ws.callee())
			asset := *proto.NewOptionalAssetFromDigest(res.AssetID)
			searchBalance, searchAddr, err := ws.diff.findBalance(senderRcp, asset)
			if err != nil {
				return nil, err
			}

			searchNewAsset := ws.diff.findNewAsset(asset.ID)
			if searchNewAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAsset(asset.ID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity += res.Quantity

					ws.diff.oldAssetsInfo[asset.ID] = *oldAssetFromDiff
					err = ws.diff.changeBalance(searchBalance, searchAddr, res.Quantity, asset, senderRcp)
					if err != nil {
						return nil, err
					}
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity += res.Quantity
				ws.diff.oldAssetsInfo[asset.ID] = assetInfo
				err = ws.diff.changeBalance(searchBalance, searchAddr, res.Quantity, asset, senderRcp)
				if err != nil {
					return nil, err
				}
				break
			}
			ws.diff.reissueNewAsset(asset.ID, res.Quantity, res.Reissuable)

			err = ws.diff.changeBalance(searchBalance, searchAddr, res.Quantity, asset, senderRcp)
			if err != nil {
				return nil, err
			}
		case *proto.BurnScriptAction:
			senderPK, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}
			res.Sender = &senderPK

			err = ws.validateBurnAction(res, env)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of burn action")
			}

			senderRcp := proto.NewRecipientFromAddress(ws.callee())
			asset := *proto.NewOptionalAssetFromDigest(res.AssetID)
			searchBalance, searchAddr, err := ws.diff.findBalance(senderRcp, asset)
			if err != nil {
				return nil, err
			}

			searchAsset := ws.diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				if oldAssetFromDiff := ws.diff.findOldAsset(res.AssetID); oldAssetFromDiff != nil {
					oldAssetFromDiff.diffQuantity -= res.Quantity

					ws.diff.oldAssetsInfo[asset.ID] = *oldAssetFromDiff
					err = ws.diff.changeBalance(searchBalance, searchAddr, -res.Quantity, asset, senderRcp)
					if err != nil {
						return nil, err
					}
					break
				}
				var assetInfo diffOldAssetInfo
				assetInfo.diffQuantity -= res.Quantity
				ws.diff.oldAssetsInfo[asset.ID] = assetInfo
				err = ws.diff.changeBalance(searchBalance, searchAddr, -res.Quantity, asset, senderRcp)
				if err != nil {
					return nil, err
				}
				break
			}
			ws.diff.burnNewAsset(res.AssetID, res.Quantity)

			err = ws.diff.changeBalance(searchBalance, searchAddr, -res.Quantity, asset, senderRcp)
			if err != nil {
				return nil, err
			}

		case *proto.LeaseScriptAction:
			senderAddress := ws.callee()
			pk, err := ws.diff.state.NewestScriptPKByAddr(senderAddress)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			res.Sender = &pk

			err = ws.validateLeaseAction(res, restrictions)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of lease action")
			}

			waves := proto.NewOptionalAssetWaves()

			recipientSearchBalance, recipientSearchAddress, err := ws.diff.findBalance(res.Recipient, waves)
			if err != nil {
				return nil, err
			}
			err = ws.diff.changeLeaseIn(recipientSearchBalance, recipientSearchAddress, res.Amount, res.Recipient)
			if err != nil {
				return nil, err
			}

			senderAccount := proto.NewRecipientFromAddress(senderAddress)
			senderSearchBalance, senderSearchAddr, err := ws.diff.findBalance(senderAccount, waves)
			if err != nil {
				return nil, err
			}

			err = ws.diff.changeLeaseOut(senderSearchBalance, senderSearchAddr, res.Amount, senderAccount)
			if err != nil {
				return nil, err
			}

			ws.diff.addNewLease(res.Recipient, senderAccount, res.Amount, res.ID)

		case *proto.LeaseCancelScriptAction:
			pk, err := ws.diff.state.NewestScriptPKByAddr(ws.callee())
			if err != nil {
				return nil, errors.Wrap(err, "failed to get public key by address")
			}

			res.Sender = &pk
			err = ws.validateLeaseCancelAction()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pass validation of lease cancel action")
			}

			searchLease, err := ws.diff.findLeaseByIDForCancel(res.LeaseID)
			if err != nil {
				return nil, errors.Errorf("failed to find lease by leaseID")
			}
			if searchLease == nil {
				return nil, errors.Errorf("there is no lease to cancel")
			}

			waves := proto.NewOptionalAssetWaves()

			recipientBalance, recipientSearchAddress, err := ws.diff.findBalance(searchLease.Recipient, waves)
			if err != nil {
				return nil, err
			}
			if recipientBalance == nil {
				_, recipientSearchAddress = ws.diff.createNewWavesBalance(searchLease.Recipient)
			}

			senderBalance, senderSearchAddress, err := ws.diff.findBalance(searchLease.Sender, waves)
			if err != nil {
				return nil, err
			}
			if senderBalance == nil {
				_, senderSearchAddress = ws.diff.createNewWavesBalance(searchLease.Sender)
			}

			ws.diff.cancelLease(*searchLease, senderSearchAddress, recipientSearchAddress)

		default:
		}
	}

	return actions, nil
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
	ver                   int
	validatePaymentsAfter uint64
	isBlockV5Activated    bool
	isRiveV6Activated     bool
	mds                   int
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState, internalPaymentsValidationHeight uint64) (*EvaluationEnvironment, error) {
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
	}, nil
}

func NewEnvironmentWithWrappedState(
	env *EvaluationEnvironment,
	payments proto.ScriptPayments,
	sender proto.WavesAddress,
	isBlockV5Activated bool,
	isRideV6Activated bool,
) (*EvaluationEnvironment, error) {
	recipient := proto.NewRecipientFromAddress(proto.WavesAddress(env.th.(rideAddress)))

	st := newWrappedState(env)
	for _, payment := range payments {
		var (
			senderBalance uint64
			err           error
			callerRcp     = proto.NewRecipientFromAddress(sender)
		)
		if payment.Asset.Present {
			senderBalance, err = st.NewestAssetBalance(callerRcp, payment.Asset.ID)
		} else {
			senderBalance, err = st.NewestWavesBalance(callerRcp)
		}
		if err != nil {
			return nil, err
		}
		if senderBalance < payment.Amount {
			return nil, errors.New("not enough money for tx attached payments")
		}

		searchBalance, searchAddr, err := st.diff.findBalance(recipient, payment.Asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}
		err = st.diff.changeBalance(searchBalance, searchAddr, int64(payment.Amount), payment.Asset, recipient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}

		senderSearchBalance, senderSearchAddr, err := st.diff.findBalance(callerRcp, payment.Asset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
		}

		err = st.diff.changeBalance(senderSearchBalance, senderSearchAddr, -int64(payment.Amount), payment.Asset, callerRcp)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create RIDE environment with wrapped state")
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
		isBlockV5Activated:    isBlockV5Activated,
		isRiveV6Activated:     isRideV6Activated,
	}, nil
}

func (e *EvaluationEnvironment) rideV6Activated() bool {
	return e.isRiveV6Activated
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

func (e *EvaluationEnvironment) ChooseSizeCheck(v int) {
	e.ver = v
	if v > 2 {
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
	e.tx = scriptTransferToObject(transfer)
}

func (e *EvaluationEnvironment) SetTransactionWithoutProofs(tx proto.Transaction) error {
	err := e.SetTransaction(tx)
	if err != nil {
		return err
	}
	e.tx["proofs"] = rideUnit{}
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

func (e *EvaluationEnvironment) SetInvoke(tx *proto.InvokeScriptWithProofs, v int) error {
	obj, err := invocationToObject(v, e.sch, tx)
	if err != nil {
		return err
	}
	e.inv = obj

	return nil
}

func (e *EvaluationEnvironment) SetEthereumInvoke(tx *proto.EthereumTransaction, v int, payments []proto.ScriptPayment) error {
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

func (e *EvaluationEnvironment) libVersion() int {
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
