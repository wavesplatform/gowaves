package ride

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func (wrappedSt *wrappedState) AddingBlockHeight() (uint64, error) {
	return wrappedSt.state.AddingBlockHeight()
}

func (wrappedSt *wrappedState) NewestScriptPKByAddr(addr proto.Address, filter bool) (crypto.PublicKey, error) {
	return wrappedSt.state.NewestScriptPKByAddr(addr, filter)
}
func (wrappedSt *wrappedState) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	return wrappedSt.state.NewestTransactionByID(id)
}
func (wrappedSt *wrappedState) NewestTransactionHeightByID(id []byte) (uint64, error) {
	return wrappedSt.state.NewestTransactionHeightByID(id)
}
func (wrappedSt *wrappedState) GetByteTree(recipient proto.Recipient) (proto.Script, error) {
	return wrappedSt.state.GetByteTree(recipient)
}
func (wrappedSt *wrappedState) NewestRecipientToAddress(recipient proto.Recipient) (*proto.Address, error) {
	return wrappedSt.state.NewestRecipientToAddress(recipient)
}

func (wrappedSt *wrappedState) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	return wrappedSt.state.NewestAddrByAlias(alias)
}

func (wrappedSt *wrappedState) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	balance, err := wrappedSt.state.NewestAccountBalance(account, asset)
	if err != nil {
		return 0, err
	}
	if balanceDiff := wrappedSt.diff.findBalance(account, asset); balanceDiff != nil {
		resBalance := int64(balance) + balanceDiff.amount
		if resBalance >= 0 {
			return uint64(resBalance), nil
		} else {
			return 0, errors.Errorf("The resulting balance is negative")
		}
	}
	return balance, nil
}

func (wrappedSt *wrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	balance, err := wrappedSt.state.NewestFullWavesBalance(account)
	if err != nil {
		return nil, err
	}

	if wavesBalanceDiff := wrappedSt.diff.findWavesBalance(account); wavesBalanceDiff != nil {
		resRegular := wavesBalanceDiff.regular + int64(balance.Regular)
		resGenerating := wavesBalanceDiff.generating + int64(balance.Generating)
		resAvailable := wavesBalanceDiff.available + int64(balance.Available)
		resEffective := wavesBalanceDiff.effective + int64(balance.Effective)

		if resRegular >= 0 && resGenerating >= 0 && resAvailable >= 0 && resEffective >= 0 {
			return &proto.FullWavesBalance{Regular: uint64(resRegular),
				Generating: uint64(resGenerating),
				Available:  uint64(resAvailable),
				Effective:  uint64(resEffective),
				LeaseIn:    balance.LeaseIn,
				LeaseOut:   balance.LeaseOut}, nil
		}

		return nil, errors.Errorf("The resulting balance is negative")

	}
	return balance, nil
}

func (wrappedSt *wrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	if intDataEntry := wrappedSt.diff.findIntFromDataEntryByKey(key); intDataEntry != nil {
		return intDataEntry, nil
	}
	return wrappedSt.state.RetrieveNewestIntegerEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	if boolDataEntry := wrappedSt.diff.findBoolFromDataEntryByKey(key); boolDataEntry != nil {
		return boolDataEntry, nil
	}
	return wrappedSt.state.RetrieveNewestBooleanEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	if stringDataEntry := wrappedSt.diff.findStringFromDataEntryByKey(key); stringDataEntry != nil {
		return stringDataEntry, nil
	}
	return wrappedSt.state.RetrieveNewestStringEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	if binaryDataEntry := wrappedSt.diff.findBinaryFromDataEntryByKey(key); binaryDataEntry != nil {
		return binaryDataEntry, nil
	}
	return wrappedSt.state.RetrieveNewestBinaryEntry(account, key)
}
func (wrappedSt *wrappedState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	if cost := wrappedSt.diff.findSponsorship(assetID); cost != nil {
		if *cost == 0 {
			return false, nil
		}
		return true, nil
	}
	return wrappedSt.state.NewestAssetIsSponsored(assetID)
}
func (wrappedSt *wrappedState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	searchNewAsset := wrappedSt.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.state.NewestAssetInfo(assetID)
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

	addr, err := wrappedSt.NewestRecipientToAddress(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get address from recipient in NewestAssetInfo")
	}

	issuerPK, err := wrappedSt.NewestScriptPKByAddr(*addr, false)
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
		Issuer:          *addr,
		IssuerPublicKey: issuerPK,
		Reissuable:      searchNewAsset.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}
func (wrappedSt *wrappedState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	searchNewAsset := wrappedSt.diff.findNewAsset(assetID)

	if searchNewAsset == nil {

		assetFromStore, err := wrappedSt.state.NewestFullAssetInfo(assetID)
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

	addr, err := wrappedSt.NewestRecipientToAddress(searchNewAsset.dAppIssuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get address from recipient in NewestAssetInfo")
	}

	issuerPK, err := wrappedSt.NewestScriptPKByAddr(*addr, false)
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
		Issuer:          *addr,
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
	return wrappedSt.state.NewestHeaderByHeight(height)
}
func (wrappedSt *wrappedState) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	return wrappedSt.state.BlockVRF(blockHeader, height)
}

func (wrappedSt *wrappedState) EstimatorVersion() (int, error) {
	return wrappedSt.state.EstimatorVersion()
}
func (wrappedSt *wrappedState) IsNotFound(err error) bool {
	return wrappedSt.state.IsNotFound(err)
}

type wrappedState struct {
	state    types.SmartState
	tmpState types.SmartState
	diff     diffState
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
	wrappedSt := wrappedState{state: state, tmpState: state}
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
			if res.Entry.GetValueType() == proto.DataInteger {

				bVal, err := res.Entry.MarshalValue()
				if err != nil {
					return errors.Wrap(err, "failed to marshal IntegerDataEntry bytes from value")
				}
				value := int64(binary.BigEndian.Uint64(bVal[1:]))

				var intEntry proto.IntegerDataEntry
				intEntry.Value = value
				intEntry.Key = res.Entry.GetKey()

				e.st.diff.dataEntries.diffInteger = append(e.st.diff.dataEntries.diffInteger, intEntry)
			}

			if res.Entry.GetValueType() == proto.DataBoolean {
				bVal, err := res.Entry.MarshalValue()
				if err != nil {
					return errors.Wrap(err, "failed to marshal value value from Bool Entry")
				}
				value, err := proto.Bool(bVal[1:])
				if err != nil {
					return errors.Wrap(err, "failed to cast bytes value to bool format")
				}

				var boolEntry proto.BooleanDataEntry
				boolEntry.Value = value
				boolEntry.Key = res.Entry.GetKey()

				e.st.diff.dataEntries.diffBool = append(e.st.diff.dataEntries.diffBool, boolEntry)
			}

			if res.Entry.GetValueType() == proto.DataBinary {
				bVal, err := res.Entry.MarshalValue()
				if err != nil {
					return errors.Wrap(err, "failed to marshal value value from Binary Entry")
				}
				value, err := proto.BytesWithUInt16Len(bVal[1:])
				if err != nil {
					return errors.Wrap(err, "failed to cast binary value to binary format")
				}

				var binaryEntry proto.BinaryDataEntry
				binaryEntry.Value = value
				binaryEntry.Key = res.Entry.GetKey()

				e.st.diff.dataEntries.diffBinary = append(e.st.diff.dataEntries.diffBinary, binaryEntry)
			}

			if res.Entry.GetValueType() == proto.DataString {
				bVal, err := res.Entry.MarshalValue()
				if err != nil {
					return errors.Wrap(err, "failed to marshal value value from String Entry")
				}
				value, err := proto.StringWithUInt16Len(bVal[1:])
				if err != nil {
					return errors.Wrap(err, "failed to cast binary value to binary format")
				}

				var stringEntry proto.StringDataEntry
				stringEntry.Value = value
				stringEntry.Key = res.Entry.GetKey()

				e.st.diff.dataEntries.diffString = append(e.st.diff.dataEntries.diffString, stringEntry)
			}

			if res.Entry.GetValueType() == proto.DataDelete {

				key := res.Entry.GetKey()

				for i, intDataEntry := range e.st.diff.dataEntries.diffInteger {
					if key == intDataEntry.Key {
						length := len(e.st.diff.dataEntries.diffInteger)

						e.st.diff.dataEntries.diffInteger[i] = e.st.diff.dataEntries.diffInteger[length-1] // Copy last element to index i.
						e.st.diff.dataEntries.diffInteger = e.st.diff.dataEntries.diffInteger[:length-1]   // Truncate

						return nil
					}
				}

				for i, stringDataEntry := range e.st.diff.dataEntries.diffString {
					if key == stringDataEntry.Key {
						length := len(e.st.diff.dataEntries.diffString)

						e.st.diff.dataEntries.diffString[i] = e.st.diff.dataEntries.diffString[length-1]
						e.st.diff.dataEntries.diffString = e.st.diff.dataEntries.diffString[:length-1]

						return nil
					}
				}

				for i, boolDataEntry := range e.st.diff.dataEntries.diffBool {
					if key == boolDataEntry.Key {
						length := len(e.st.diff.dataEntries.diffBool)

						e.st.diff.dataEntries.diffBool[i] = e.st.diff.dataEntries.diffBool[length-1]
						e.st.diff.dataEntries.diffBool = e.st.diff.dataEntries.diffBool[:length-1]

						return nil
					}
				}

				for i, binaryDataEntry := range e.st.diff.dataEntries.diffBinary {
					if key == binaryDataEntry.Key {
						length := len(e.st.diff.dataEntries.diffBinary)

						e.st.diff.dataEntries.diffBinary[i] = e.st.diff.dataEntries.diffBinary[length-1]
						e.st.diff.dataEntries.diffBinary = e.st.diff.dataEntries.diffBinary[:length-1]

						return nil
					}
				}

			}
		case proto.TransferScriptAction:

			if searchBalance := e.st.diff.findBalance(res.Recipient, res.Asset.ID.Bytes()); searchBalance != nil {
				searchBalance.amount += res.Amount
			} else {
				var balance diffBalance

				balance.recipient = res.Recipient
				balance.assetID = res.Asset.ID
				balance.amount = res.Amount
				e.st.diff.balances = append(e.st.diff.balances, balance)
			}

			if searchWavesBalance := e.st.diff.findWavesBalance(res.Recipient); searchWavesBalance != nil {
				searchWavesBalance.regular += res.Amount
				searchWavesBalance.available += res.Amount
				searchWavesBalance.generating += res.Amount
				searchWavesBalance.effective += res.Amount
			} else {
				var wavesBalance diffWavesBalance

				wavesBalance.recipient = res.Recipient
				wavesBalance.regular = res.Amount
				wavesBalance.available = res.Amount
				wavesBalance.generating = res.Amount
				wavesBalance.effective = res.Amount

				e.st.diff.wavesBalances = append(e.st.diff.wavesBalances, wavesBalance)
			}

		case proto.SponsorshipScriptAction:

			var sponsorship diffSponsorship
			sponsorship.assetID = res.AssetID
			sponsorship.MinFee = res.MinFee

			e.st.diff.sponsorships = append(e.st.diff.sponsorships, sponsorship)
		case proto.IssueScriptAction:
			var assetInfo diffNewAssetInfo

			issuerAddr := proto.Address(e.th.(rideAddress))

			assetInfo.dAppIssuer = proto.NewRecipientFromAddress(issuerAddr)
			assetInfo.assetID = res.ID
			assetInfo.name = res.Name
			assetInfo.description = res.Description
			assetInfo.quantity = res.Quantity
			assetInfo.decimals = res.Decimals
			assetInfo.reissuable = res.Reissuable
			assetInfo.script = res.Script
			assetInfo.nonce = res.Nonce

			e.st.diff.newAssetsInfo = append(e.st.diff.newAssetsInfo, assetInfo)

		case proto.ReissueScriptAction:
			searchNewAsset := e.st.diff.findNewAsset(res.AssetID)
			if searchNewAsset == nil {
				var assetInfo diffOldAssetInfo

				issuerAddr := proto.Address(e.th.(rideAddress))

				assetInfo.dAppIssuer = proto.NewRecipientFromAddress(issuerAddr)
				assetInfo.assetID = res.AssetID
				assetInfo.diffQuantity = res.Quantity

				e.st.diff.oldAssetsInfo = append(e.st.diff.oldAssetsInfo, assetInfo)
				break
			}
			searchNewAsset.quantity += res.Quantity

		case proto.BurnScriptAction:
			searchAsset := e.st.diff.findNewAsset(res.AssetID)
			if searchAsset == nil {
				var assetInfo diffOldAssetInfo

				issuerAddr := proto.Address(e.th.(rideAddress))

				assetInfo.dAppIssuer = proto.NewRecipientFromAddress(issuerAddr)
				assetInfo.assetID = res.AssetID
				assetInfo.diffQuantity = -res.Quantity

				e.st.diff.oldAssetsInfo = append(e.st.diff.oldAssetsInfo, assetInfo)
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
