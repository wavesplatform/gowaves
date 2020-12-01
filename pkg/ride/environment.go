package ride

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

//type interlayerState struct {
//
//}

func (wrappedSt *wrappedState) AddingBlockHeight() (uint64, error) {
	return wrappedSt.state.AddingBlockHeight()
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

//-----//
//const wavesBalanceKeySize = 1 + proto.AddressSize
//const wavesBalanceKeyPrefix byte = iota

//type wavesBalanceKey struct {
//	address proto.Address
//}

//func (k *wavesBalanceKey) bytes() []byte {
//	buf := make([]byte, wavesBalanceKeySize)
//	buf[0] = wavesBalanceKeyPrefix
//	copy(buf[1:], k.address[:])
//	return buf
//}

func (wrappedSt *wrappedState) NewestAccountBalance(account proto.Recipient, asset []byte) (uint64, error) {
	//addr, err := wrappedSt.state.NewestRecipientToAddress(account)
	//if err != nil {
	//	return 0, errors.Wrap(err, "Failed to get script by recipient")
	//}
	////key := wavesBalanceKey{address: *addr}
	//
	//newProfile := &balanceProfile{}
	//newProfile.balance = uint64(newBalance)
	//newProfile.leaseIn = newLeaseIn
	//newProfile.leaseOut = newLeaseOut
	//
	//
	//
	//
	//if asset == nil {
	//	profile, err := s.newestWavesBalanceProfile(*addr)
	//	if err != nil {
	//		return 0, wrapErr(RetrievalError, err)
	//	}
	//	return profile.balance, nil
	//}
	//balance, err := s.newestAssetBalance(*addr, asset)
	//if err != nil {
	//	return 0, wrapErr(RetrievalError, err)
	//}
	//return balance, nil
	//TODO
	return wrappedSt.state.NewestAccountBalance(account, asset)
}
func (wrappedSt *wrappedState) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	//TODO
	return wrappedSt.state.NewestFullWavesBalance(account)
}
func (wrappedSt *wrappedState) NewestAddrByAlias(alias proto.Alias) (proto.Address, error) {
	//TODO
	return wrappedSt.state.NewestAddrByAlias(alias)
}
func (wrappedSt *wrappedState) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	//TODO
	return wrappedSt.state.RetrieveNewestIntegerEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	//TODO
	return wrappedSt.state.RetrieveNewestBooleanEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	//TODO
	return wrappedSt.state.RetrieveNewestStringEntry(account, key)
}
func (wrappedSt *wrappedState) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	//TODO
	return wrappedSt.state.RetrieveNewestBinaryEntry(account, key)
}
func (wrappedSt *wrappedState) NewestAssetIsSponsored(assetID crypto.Digest) (bool, error) {
	//TODO
	return wrappedSt.state.NewestAssetIsSponsored(assetID)
}
func (wrappedSt *wrappedState) NewestAssetInfo(assetID crypto.Digest) (*proto.AssetInfo, error) {
	//TODO
	return wrappedSt.state.NewestAssetInfo(assetID)
}
func (wrappedSt *wrappedState) NewestFullAssetInfo(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
	//TODO
	return wrappedSt.state.NewestFullAssetInfo(assetID)
}

//---------//

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
	actions  []proto.ScriptAction
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

func (e *Environment) applyToState(actions []proto.ScriptAction) {
	e.st.actions = append(e.st.actions, actions...)
}

func (e *Environment) checkMessageLength(l int) bool {
	return e.check(l)
}

func (e *Environment) invocation() rideObject {
	return e.inv
}
