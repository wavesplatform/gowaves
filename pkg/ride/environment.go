package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func (wrappedSt *wrappedState) AddingBlockHeight() (uint64, error) {
	return wrappedSt.diff.state.AddingBlockHeight()
}

func (wrappedSt *wrappedState) NewestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error) {
	return wrappedSt.diff.state.NewestScriptPKByAddr(addr, filter)
}
func (wrappedSt *wrappedState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	return wrappedSt.diff.state.NewestTransactionByID(id)
}
func (wrappedSt *wrappedState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	return wrappedSt.diff.state.NewestTransactionHeightByID(id)
}
func (wrappedSt *wrappedState) GetByteTree(recipient proto.Recipient) (proto.Script, error) {
	return wrappedSt.diff.state.GetByteTree(recipient)
}
func (wrappedSt *wrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	return wrappedSt.diff.state.NewestRecipientToAddress(recipient)
}

func (wrappedSt *wrappedState) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	return wrappedSt.diff.state.NewestAddrByAlias(alias)
}

func (wrappedSt *wrappedState) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	balance, err := wrappedSt.diff.state.NewestAccountBalance(account, asset)
	if err != nil {
		return 0, err
	}
	balanceDiff, _, err := wrappedSt.diff.findBalance(account, asset)
	if err != nil {
		return 0, err
	}
	if balanceDiff != nil {
		resBalance := int64(balance) + balanceDiff.amount
		return uint64(resBalance), nil

	}
	return balance, nil
}

func (wrappedSt *wrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	balance, err := wrappedSt.diff.state.NewestFullWavesBalance(account)
	if err != nil {
		return nil, err
	}

	wavesBalanceDiff, _, err := wrappedSt.diff.findBalance(account, nil)
	if err != nil {
		return nil, err
	}
	if wavesBalanceDiff != nil {
		resRegular := wavesBalanceDiff.amount + int64(balance.Regular)
		resGenerating := wavesBalanceDiff.amount + int64(balance.Generating)
		resAvailable := wavesBalanceDiff.amount + int64(balance.Available)
		resEffective := wavesBalanceDiff.amount + int64(balance.Effective)

		return &proto.FullWavesBalance{Regular: uint64(resRegular),
			Generating: uint64(resGenerating),
			Available:  uint64(resAvailable),
			Effective:  uint64(resEffective),
			LeaseIn:    balance.LeaseIn,
			LeaseOut:   balance.LeaseOut}, nil

	}
	return balance, nil
}

func (wrappedSt *wrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	address, err := wrappedSt.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if intDataEntry := wrappedSt.diff.findIntFromDataEntryByKey(key, address.String()); intDataEntry != nil {
		return intDataEntry, nil
	}

	return wrappedSt.diff.state.RetrieveNewestIntegerEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	address, err := wrappedSt.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if boolDataEntry := wrappedSt.diff.findBoolFromDataEntryByKey(key, address.String()); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return wrappedSt.diff.state.RetrieveNewestBooleanEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	address, err := wrappedSt.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if stringDataEntry := wrappedSt.diff.findStringFromDataEntryByKey(key, address.String()); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return wrappedSt.diff.state.RetrieveNewestStringEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	address, err := wrappedSt.diff.state.NewestRecipientToAddress(account)
	if err != nil {
		return nil, err
	}

	if deletedDataEntry := wrappedSt.diff.findDeleteFromDataEntryByKey(key, address.String()); deletedDataEntry != nil {
		return nil, nil
	}

	if binaryDataEntry := wrappedSt.diff.findBinaryFromDataEntryByKey(key, address.String()); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return wrappedSt.diff.state.RetrieveNewestBinaryEntry(account, key)
}
func (wrappedSt *wrappedState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	if cost := wrappedSt.diff.findSponsorship(assetID); cost != nil {
		if *cost == 0 {
			return false, nil
		}
		return true, nil
	}
	return wrappedSt.diff.state.NewestAssetIsSponsored(assetID)
}
func (wrappedSt *wrappedState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	searchNewAsset := wrappedSt.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.diff.state.NewestAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := wrappedSt.diff.findOldAsset(assetID); oldAssetFromDiff != nil {
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
func (wrappedSt *wrappedState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	searchNewAsset := wrappedSt.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.diff.state.NewestFullAssetInfo(assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset's info from store")
		}

		if oldAssetFromDiff := wrappedSt.diff.findOldAsset(assetID); oldAssetFromDiff != nil {
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
	if sponsorship := wrappedSt.diff.findSponsorship(assetID); sponsorship != nil {
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

func (wrappedSt *wrappedState) NewestHeaderByHeight(height proto.Height) (*proto.BlockHeader, error) {
	return wrappedSt.diff.state.NewestHeaderByHeight(height)
}
func (wrappedSt *wrappedState) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	return wrappedSt.diff.state.BlockVRF(blockHeader, height)
}

func (wrappedSt *wrappedState) EstimatorVersion() (int, error) {
	return wrappedSt.diff.state.EstimatorVersion()
}
func (wrappedSt *wrappedState) IsNotFound(err error) bool {
	return wrappedSt.diff.state.IsNotFound(err)
}

func (wrappedSt *wrappedState) ApplyToState(actions []proto.ScriptAction) error {

	for _, action := range actions {
		switch res := action.(type) {

		case *proto.DataEntryScriptAction:

			switch dataEntry := res.Entry.(type) {

			case *proto.IntegerDataEntry:
				intEntry := *dataEntry
				addr := proto.Address(wrappedSt.envThis)

				wrappedSt.diff.dataEntries.diffInteger[dataEntry.Key+addr.String()] = intEntry
			case *proto.StringDataEntry:
				stringEntry := *dataEntry
				addr := proto.Address(wrappedSt.envThis)

				wrappedSt.diff.dataEntries.diffString[dataEntry.Key+addr.String()] = stringEntry

			case *proto.BooleanDataEntry:
				boolEntry := *dataEntry
				addr := proto.Address(wrappedSt.envThis)

				wrappedSt.diff.dataEntries.diffBool[dataEntry.Key+addr.String()] = boolEntry

			case *proto.BinaryDataEntry:
				binaryEntry := *dataEntry
				addr := proto.Address(wrappedSt.envThis)

				wrappedSt.diff.dataEntries.diffBinary[dataEntry.Key+addr.String()] = binaryEntry

			case *proto.DeleteDataEntry:
				deleteEntry := *dataEntry
				addr := proto.Address(wrappedSt.envThis)

				wrappedSt.diff.dataEntries.diffDDelete[dataEntry.Key+addr.String()] = deleteEntry
			default:

			}

		case *proto.TransferScriptAction:
			searchBalance, searchAddr, err := wrappedSt.diff.findBalance(res.Recipient, res.Asset.ID.Bytes())
			if err != nil {
				return err
			}
			err = wrappedSt.diff.changeBalance(searchBalance, searchAddr, res.Amount, res.Asset.ID, res.Recipient)
			if err != nil {
				return err
			}

			senderAddr := proto.Address(wrappedSt.envThis)
			senderRecip := proto.Recipient{Address: &senderAddr}
			senderSearchBalance, senderSearchAddr, err := wrappedSt.diff.findBalance(senderRecip, res.Asset.ID.Bytes())
			if err != nil {
				return err
			}

			err = wrappedSt.diff.changeBalance(senderSearchBalance, senderSearchAddr, -res.Amount, res.Asset.ID, senderRecip)
			if err != nil {
				return err
			}

		case *proto.SponsorshipScriptAction:
			var sponsorship diffSponsorship
			sponsorship.MinFee = res.MinFee

			wrappedSt.diff.sponsorships[res.AssetID.String()] = sponsorship

		case *proto.IssueScriptAction:
			var assetInfo diffNewAssetInfo
			assetInfo.dAppIssuer = proto.Address(wrappedSt.envThis)
			assetInfo.name = res.Name
			assetInfo.description = res.Description
			assetInfo.quantity = res.Quantity
			assetInfo.decimals = res.Decimals
			assetInfo.reissuable = res.Reissuable
			assetInfo.script = res.Script
			assetInfo.nonce = res.Nonce

			wrappedSt.diff.newAssetsInfo[res.ID.String()] = assetInfo

		case *proto.ReissueScriptAction:
			searchNewAsset := wrappedSt.diff.findNewAsset(res.AssetID)
			if searchNewAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += res.Quantity

				wrappedSt.diff.oldAssetsInfo[res.AssetID.String()] = assetInfo
				break
			}
			wrappedSt.diff.reissueNewAsset(res.AssetID, res.Quantity, res.Reissuable)

		case *proto.BurnScriptAction:
			searchAsset := wrappedSt.diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.diffQuantity += -res.Quantity

				wrappedSt.diff.oldAssetsInfo[res.AssetID.String()] = assetInfo

				break
			}
			wrappedSt.diff.burnNewAsset(res.AssetID, res.Quantity)

		default:
		}

	}
	return nil
}

type wrappedState struct {
	diff    diffState
	envThis rideAddress
}

type Environment struct {
	sch         proto.Scheme
	st          types.SmartState
	act         []proto.ScriptAction
	h           rideInt
	tx          rideObject
	id          rideType
	th          rideType
	b           rideObject
	check       func(int) bool
	inv         rideObject
	invokeCount uint64
}

func newWrappedState(state types.SmartState, envThis rideType) types.SmartState {
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

	diffSt := &diffState{state: state, dataEntries: dataEntries, balances: balances, sponsorships: sponsorships, newAssetsInfo: newAssetInfo, oldAssetsInfo: oldAssetInfo}
	wrappedSt := wrappedState{diff: *diffSt, envThis: envThis.(rideAddress)}
	return &wrappedSt
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState) (*Environment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}

	return &Environment{
		sch:         scheme,
		st:          state,
		act:         nil,
		h:           rideInt(height),
		tx:          nil,
		id:          nil,
		th:          nil,
		b:           nil,
		check:       func(int) bool { return true },
		inv:         nil,
		invokeCount: 0,
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

func (e *Environment) actions() []proto.ScriptAction {
	return e.act
}

func (e *Environment) setNewDAppAddress(address proto.Address) {
	e.SetThisFromAddress(address)
}

func (e *Environment) applyToState(actions []proto.ScriptAction) error {
	return e.st.ApplyToState(actions)
}

func (e *Environment) appendActions(actions []proto.ScriptAction) {
	e.act = append(e.act, actions...)
}

func (e *Environment) smartAppendActions(actions []proto.ScriptAction) error {
	_, ok := e.st.(*wrappedState)
	if !ok {
		wrappedSt := newWrappedState(e.state(), e.this())
		e.st = wrappedSt
	}
	e.appendActions(actions)

	return e.applyToState(actions)
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
