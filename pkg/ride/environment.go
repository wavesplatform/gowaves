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
	balanceDiff, err := wrappedSt.diff.findBalance(account, asset)
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

	wavesBalanceDiff, err := wrappedSt.diff.findWavesBalance(account)
	if err != nil {
		return nil, err
	}
	if wavesBalanceDiff != nil {
		resRegular := wavesBalanceDiff.regular + int64(balance.Regular)
		resGenerating := wavesBalanceDiff.generating + int64(balance.Generating)
		resAvailable := wavesBalanceDiff.available + int64(balance.Available)
		resEffective := wavesBalanceDiff.effective + int64(balance.Effective)

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
		ID:              searchNewAsset.assetID,
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
		ID:              searchNewAsset.assetID,
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
	if sponsorship := wrappedSt.diff.findSponsorship(searchNewAsset.assetID); sponsorship != nil {
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

type wrappedState struct {
	diff diffState
}

type Environment struct {
	sch   proto.Scheme
	st    wrappedState
	h     rideInt
	tx    rideObject
	id    rideType
	th    rideType
	b     rideObject
	check func(int) bool
	inv   rideObject
}

func NewEnvironment(scheme proto.Scheme, state types.SmartState) (*Environment, error) {
	height, err := state.AddingBlockHeight()
	if err != nil {
		return nil, err
	}
	var dataEntries diffDataEntries
	dataEntries.diffInteger = map[string]proto.IntegerDataEntry{}
	dataEntries.diffBool = map[string]proto.BooleanDataEntry{}
	dataEntries.diffString = map[string]proto.StringDataEntry{}
	dataEntries.diffBinary = map[string]proto.BinaryDataEntry{}
	dataEntries.diffDDelete = map[string]proto.DeleteDataEntry{}

	diffSt := diffState{state: state, dataEntries: dataEntries}
	wrappedSt := wrappedState{diff: diffSt}
	return &Environment{
		sch:   scheme,
		st:    wrappedSt,
		h:     rideInt(height),
		tx:    nil,
		id:    nil,
		th:    nil,
		b:     nil,
		check: func(int) bool { return true },
		inv:   nil,
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
	return &e.st
}

func (e *Environment) setNewDAppAddress(address proto.Address) {
	e.SetThisFromAddress(address)
}

func (e *Environment) applyToState(actions []proto.ScriptAction) error {

	for _, action := range actions {
		switch res := action.(type) {

		case proto.DataEntryScriptAction:

			switch dataEntry := res.Entry.(type) {

			case *proto.IntegerDataEntry:
				intEntry := *dataEntry
				addr := proto.Address(e.th.(rideAddress))

				e.st.diff.dataEntries.diffInteger[dataEntry.Key+addr.String()] = intEntry

			case *proto.BooleanDataEntry:
				boolEntry := *dataEntry
				addr := proto.Address(e.th.(rideAddress))

				e.st.diff.dataEntries.diffBool[dataEntry.Key+addr.String()] = boolEntry

			case *proto.BinaryDataEntry:
				binaryEntry := *dataEntry
				addr := proto.Address(e.th.(rideAddress))

				e.st.diff.dataEntries.diffBinary[dataEntry.Key+addr.String()] = binaryEntry

			case *proto.DeleteDataEntry:
				deleteEntry := *dataEntry
				addr := proto.Address(e.th.(rideAddress))

				e.st.diff.dataEntries.diffDDelete[dataEntry.Key+addr.String()] = deleteEntry
			default:

			}

		case proto.TransferScriptAction:
			searchBalance, err := e.st.diff.findBalance(res.Recipient, res.Asset.ID.Bytes())
			if err != nil {
				return err
			}
			if searchBalance != nil {
				searchBalance.amount += res.Amount

				// TODO списать у отправителя

			} else {
				address, err := e.st.NewestRecipientToAddress(res.Recipient)
				if err != nil {
					return err
				}
				var balance diffBalance
				balance.assetID = res.Asset.ID
				balance.amount = res.Amount

				e.st.diff.balances[address.String()+res.Asset.ID.String()] = balance
			}
			searchWavesBalance, err := e.st.diff.findWavesBalance(res.Recipient)
			if err != nil {
				return err
			}
			if searchWavesBalance != nil {
				searchWavesBalance.regular += res.Amount
				searchWavesBalance.available += res.Amount
				searchWavesBalance.generating += res.Amount
				searchWavesBalance.effective += res.Amount
			} else {
				address, err := e.st.NewestRecipientToAddress(res.Recipient)
				if err != nil {
					return err
				}
				var wavesBalance diffWavesBalance
				wavesBalance.regular = res.Amount
				wavesBalance.available = res.Amount
				wavesBalance.generating = res.Amount
				wavesBalance.effective = res.Amount

				e.st.diff.wavesBalances[address.String()+res.Asset.ID.String()] = wavesBalance
			}

		case proto.SponsorshipScriptAction:
			var sponsorship diffSponsorship
			sponsorship.MinFee = res.MinFee

			e.st.diff.sponsorships[res.AssetID.String()] = sponsorship

		case proto.IssueScriptAction:
			var assetInfo diffNewAssetInfo
			assetInfo.dAppIssuer = proto.Address(e.th.(rideAddress))
			assetInfo.assetID = res.ID
			assetInfo.name = res.Name
			assetInfo.description = res.Description
			assetInfo.quantity = res.Quantity
			assetInfo.decimals = res.Decimals
			assetInfo.reissuable = res.Reissuable
			assetInfo.script = res.Script
			assetInfo.nonce = res.Nonce

			e.st.diff.newAssetsInfo[assetInfo.dAppIssuer.String()] = assetInfo

		case proto.ReissueScriptAction:
			searchNewAsset := e.st.diff.findNewAsset(res.AssetID)
			if searchNewAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.assetID = res.AssetID
				assetInfo.diffQuantity = res.Quantity

				e.st.diff.oldAssetsInfo[proto.Address(e.th.(rideAddress)).String()] = assetInfo
				break
			}
			searchNewAsset.quantity += res.Quantity

		case proto.BurnScriptAction:
			searchAsset := e.st.diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				var assetInfo diffOldAssetInfo

				assetInfo.assetID = res.AssetID
				assetInfo.diffQuantity = -res.Quantity

				e.st.diff.oldAssetsInfo[proto.Address(e.th.(rideAddress)).String()] = assetInfo
				break
			}
			searchAsset.quantity -= res.Quantity

		default:
		}

	}
	return nil
}

func (e *Environment) checkMessageLength(l int) bool {
	return e.check(l)
}

func (e *Environment) invocation() rideObject {
	return e.inv
}
