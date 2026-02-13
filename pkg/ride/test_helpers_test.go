package ride

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func unsetMockCalls(m *mock.Mock, methodName string) {
	var toUnset []*mock.Call
	for _, c := range m.ExpectedCalls {
		if c.Method == methodName {
			toUnset = append(toUnset, c)
		}
	}
	for _, c := range toUnset {
		c.Unset()
	}
}

func countMockCalls(m *mock.Mock, methodName string) int {
	count := 0
	for _, c := range m.Calls {
		if c.Method == methodName {
			count++
		}
	}
	return count
}

func hasMockExpectation(m *mock.Mock, methodName string) bool {
	for _, c := range m.ExpectedCalls {
		if c.Method == methodName {
			return true
		}
	}
	return false
}

type testAccount struct {
	sk  crypto.SecretKey
	pk  crypto.PublicKey
	wa  proto.WavesAddress
	rcp proto.Recipient
}

func newTestAccount(t *testing.T, seed string) *testAccount {
	return newTestAccountWithScheme(t, proto.TestNetScheme, seed)
}

func newTestAccountFromPublicKey(t *testing.T, scheme proto.Scheme, publicKey string) *testAccount {
	pk, err := crypto.NewPublicKeyFromBase58(publicKey)
	require.NoError(t, err, "failed to create test account")
	ad, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err, "failed to create test account")
	rcp := proto.NewRecipientFromAddress(ad)
	return &testAccount{
		sk:  crypto.SecretKey{},
		pk:  pk,
		wa:  ad,
		rcp: rcp,
	}
}

func newTestAccountWithScheme(t *testing.T, scheme proto.Scheme, seed string) *testAccount {
	sk, pk, err := crypto.GenerateKeyPair([]byte(seed))
	require.NoError(t, err, "failed to create test account")
	ad, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err, "failed to create test account")
	rcp := proto.NewRecipientFromAddress(ad)
	return &testAccount{
		sk:  sk,
		pk:  pk,
		wa:  ad,
		rcp: rcp,
	}
}

// Can be used only when secret and public keys aren't required by test
func newTestAccountFromAddress(addr proto.WavesAddress) *testAccount {
	rcp := proto.NewRecipientFromAddress(addr)
	return &testAccount{
		sk:  crypto.SecretKey{},
		pk:  crypto.PublicKey{},
		wa:  addr,
		rcp: rcp,
	}
}

// Can be used only when secret and public keys aren't required by test
func newTestAccountFromAddressString(t *testing.T, addr string) *testAccount {
	ad, err := proto.NewAddressFromString(addr)
	require.NoError(t, err, "failed to create test account")
	return newTestAccountFromAddress(ad)
}

func (a *testAccount) publicKey() crypto.PublicKey {
	return a.pk
}

func (a *testAccount) publicKeyRef() *crypto.PublicKey {
	return &a.pk
}

func (a *testAccount) address() proto.WavesAddress {
	return a.wa
}

func (a *testAccount) recipient() proto.Recipient {
	return a.rcp
}

type testEnv struct {
	t           *testing.T
	sender      *testAccount
	dApp        *testAccount
	this        proto.WavesAddress
	dAppAddr    proto.WavesAddress
	inv         rideType
	me          *MockEnvironment
	ms          *types.MockEnrichedSmartState
	ws          *WrappedState
	recipients  map[string]proto.WavesAddress
	accounts    map[proto.WavesAddress]*testAccount
	entries     map[proto.WavesAddress]map[string]proto.DataEntry
	trees       map[proto.WavesAddress]*ast.Tree
	waves       map[proto.WavesAddress]*types.WavesBalanceProfile
	aliases     map[proto.Alias]proto.WavesAddress
	assets      map[proto.AssetID]*proto.FullAssetInfo
	sponsorship map[proto.AssetID]bool
	tokens      map[proto.WavesAddress]map[crypto.Digest]uint64
	leasings    map[crypto.Digest]*proto.LeaseInfo
	scripts     map[proto.WavesAddress]proto.Script
	notFoundErr error
}

func newTestEnv(t *testing.T) *testEnv {
	me := NewMockEnvironment(t)
	me.EXPECT().scheme().Return(proto.TestNetScheme).Maybe()
	me.EXPECT().blockV5Activated().Return(false).Maybe()
	me.EXPECT().isProtobufTx().Return(false).Maybe()
	me.EXPECT().maxDataEntriesSize().Return(proto.MaxDataEntriesScriptActionsSizeInBytesV1).Maybe()
	me.EXPECT().checkMessageLength(mock.Anything).RunAndReturn(bytesSizeCheckV1V2).Maybe()
	me.EXPECT().validateInternalPayments().Return(false).Maybe()
	me.EXPECT().rideV6Activated().Return(false).Maybe()
	me.EXPECT().consensusImprovementsActivated().Return(false).Maybe()
	me.EXPECT().blockRewardDistributionActivated().Return(false).Maybe()
	me.EXPECT().lightNodeActivated().Return(false).Maybe()
	me.EXPECT().paymentsFixActivated().Return(false).Maybe()
	r := &testEnv{
		t:           t,
		me:          me,
		ms:          types.NewMockEnrichedSmartState(t),
		recipients:  map[string]proto.WavesAddress{},
		accounts:    map[proto.WavesAddress]*testAccount{},
		entries:     map[proto.WavesAddress]map[string]proto.DataEntry{},
		trees:       map[proto.WavesAddress]*ast.Tree{},
		waves:       map[proto.WavesAddress]*types.WavesBalanceProfile{},
		aliases:     map[proto.Alias]proto.WavesAddress{},
		assets:      map[proto.AssetID]*proto.FullAssetInfo{},
		sponsorship: map[proto.AssetID]bool{},
		tokens:      map[proto.WavesAddress]map[crypto.Digest]uint64{},
		leasings:    map[crypto.Digest]*proto.LeaseInfo{},
		scripts:     map[proto.WavesAddress]proto.Script{},
		notFoundErr: errors.New("not found"),
	}
	me.EXPECT().state().RunAndReturn(func() types.SmartState {
		return r.ms
	}).Maybe()
	r.ms.EXPECT().NewestRecipientToAddress(mock.Anything).RunAndReturn(
		func(recipient proto.Recipient) (proto.WavesAddress, error) {
			if a, ok := r.recipients[recipient.String()]; ok {
				return a, nil
			}
			return proto.WavesAddress{}, errors.Errorf("unknown recipient '%s'", recipient.String())
		}).Maybe()
	r.ms.EXPECT().NewestScriptPKByAddr(mock.Anything).RunAndReturn(
		func(addr proto.WavesAddress) (crypto.PublicKey, error) {
			if acc, ok := r.accounts[addr]; ok {
				return acc.publicKey(), nil
			}
			return crypto.PublicKey{}, errors.Errorf("unknown address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().NewestScriptByAccount(mock.Anything).RunAndReturn(
		func(account proto.Recipient) (*ast.Tree, error) {
			addr, err := r.resolveRecipient(account)
			if err != nil {
				return nil, err
			}
			if t, ok := r.trees[addr]; ok {
				return t, nil
			}
			return nil, errors.Errorf("unknown address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().RetrieveNewestBinaryEntry(mock.Anything, mock.Anything).RunAndReturn(
		func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
			e, err := r.retrieveEntry(account, key)
			if err != nil {
				return nil, err
			}
			if be, ok := e.(*proto.BinaryDataEntry); ok {
				return be, nil
			}
			return nil, errors.Wrapf(r.notFoundErr, // consider as not found, because it is not a binary data entry
				"unexpected type '%T' of entry at '%s' by key '%s'", e, account.String(), key,
			)
		}).Maybe()
	r.ms.EXPECT().RetrieveNewestBooleanEntry(mock.Anything, mock.Anything).RunAndReturn(
		func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
			e, err := r.retrieveEntry(account, key)
			if err != nil {
				return nil, err
			}
			if be, ok := e.(*proto.BooleanDataEntry); ok {
				return be, nil
			}
			return nil, errors.Wrapf(r.notFoundErr, // consider as not found, because it is not a boolean data entry
				"unexpected type '%T' of entry at '%s' by key '%s'", e, account.String(), key,
			)
		}).Maybe()
	r.ms.EXPECT().RetrieveNewestIntegerEntry(mock.Anything, mock.Anything).RunAndReturn(
		func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
			e, err := r.retrieveEntry(account, key)
			if err != nil {
				return nil, err
			}
			if be, ok := e.(*proto.IntegerDataEntry); ok {
				return be, nil
			}
			return nil, errors.Wrapf(r.notFoundErr, // Consider as not found, because it is not an integer data entry.
				"unexpected type '%T' of entry at '%s' by key '%s'", e, account.String(), key,
			)
		}).Maybe()
	r.ms.EXPECT().RetrieveNewestStringEntry(mock.Anything, mock.Anything).RunAndReturn(
		func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
			e, err := r.retrieveEntry(account, key)
			if err != nil {
				return nil, err
			}
			if be, ok := e.(*proto.StringDataEntry); ok {
				return be, nil
			}
			return nil, errors.Wrapf(r.notFoundErr, // consider as not found, because it is not a string data entry
				"unexpected type '%T' of entry at '%s' by key '%s'", e, account.String(), key,
			)
		}).Maybe()
	r.ms.EXPECT().NewestWavesBalance(mock.Anything).RunAndReturn(func(account proto.Recipient) (uint64, error) {
		addr, err := r.resolveRecipient(account)
		if err != nil {
			return 0, err
		}
		if profile, ok := r.waves[addr]; ok {
			return profile.Balance, nil
		}
		return 0, errors.Errorf("no balance profile for address '%s'", addr.String())
	}).Maybe()
	r.ms.EXPECT().WavesBalanceProfile(mock.Anything).RunAndReturn(
		func(id proto.AddressID) (*types.WavesBalanceProfile, error) {
			addr, err := id.ToWavesAddress(r.me.scheme())
			require.NoError(r.t, err)
			if profile, ok := r.waves[addr]; ok {
				return profile, nil
			}
			return nil, errors.Errorf("no balance profile for address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().NewestFullWavesBalance(mock.Anything).RunAndReturn(
		func(account proto.Recipient) (*proto.FullWavesBalance, error) {
			addr, err := r.resolveRecipient(account)
			if err != nil {
				return nil, err
			}
			if profile, ok := r.waves[addr]; ok {
				eff := int64(profile.Balance) + profile.LeaseIn - profile.LeaseOut
				if eff < 0 {
					return nil, errors.New("negative effective balance")
				}
				spb := int64(profile.Balance) - profile.LeaseOut
				if spb < 0 {
					return nil, errors.New("negative spendable balance")
				}
				return &proto.FullWavesBalance{
					Regular:    profile.Balance,
					Generating: profile.Generating,
					Available:  uint64(spb),
					Effective:  uint64(eff),
					LeaseIn:    uint64(profile.LeaseIn),
					LeaseOut:   uint64(profile.LeaseOut),
				}, nil
			}
			return nil, errors.Errorf("no balance profile for address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().NewestAddrByAlias(mock.Anything).RunAndReturn(
		func(alias proto.Alias) (proto.WavesAddress, error) {
			if a, ok := r.aliases[alias]; ok {
				return a, nil
			}
			return proto.WavesAddress{}, errors.Errorf("unknown alias '%s'", alias.String())
		}).Maybe()
	r.ms.EXPECT().NewestAssetIsSponsored(mock.Anything).RunAndReturn(
		func(assetID crypto.Digest) (bool, error) {
			aID := proto.AssetIDFromDigest(assetID)
			if s, ok := r.sponsorship[aID]; ok {
				return s, nil
			}
			return false, errors.Errorf("unknown asset '%s'", assetID.String())
		}).Maybe()
	r.ms.EXPECT().NewestAssetConstInfo(mock.Anything).RunAndReturn(
		func(assetID proto.AssetID) (*proto.AssetConstInfo, error) {
			if ai, ok := r.assets[assetID]; ok {
				return &ai.AssetConstInfo, nil
			}
			return nil, errors.Errorf("unknown asset '%s'", assetID.String())
		}).Maybe()
	r.ms.EXPECT().NewestAssetInfo(mock.Anything).RunAndReturn(
		func(assetID crypto.Digest) (*proto.AssetInfo, error) {
			aID := proto.AssetIDFromDigest(assetID)
			if ai, ok := r.assets[aID]; ok {
				return &ai.AssetInfo, nil
			}
			return nil, errors.Errorf("unknown asset '%s'", assetID.String())
		}).Maybe()
	r.ms.EXPECT().NewestFullAssetInfo(mock.Anything).RunAndReturn(
		func(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
			aID := proto.AssetIDFromDigest(assetID)
			if ai, ok := r.assets[aID]; ok {
				return ai, nil
			}
			return nil, errors.Errorf("unknown asset '%s'", assetID.String())
		}).Maybe()
	r.ms.EXPECT().NewestAssetBalance(mock.Anything, mock.Anything).RunAndReturn(
		func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
			addr, err := r.resolveRecipient(account)
			if err != nil {
				return 0, err
			}
			if balances, ok := r.tokens[addr]; ok {
				if b, bOK := balances[assetID]; bOK {
					return b, nil
				}
				return 0, errors.Errorf("unknown asset '%s' for address '%s'", assetID.String(), addr.String())
			}
			return 0, errors.Errorf("no asset balances for address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().NewestAssetBalanceByAddressID(mock.Anything, mock.Anything).RunAndReturn(
		func(id proto.AddressID, a crypto.Digest) (uint64, error) {
			addr, err := id.ToWavesAddress(r.me.scheme())
			require.NoError(r.t, err)
			if t, ok := r.tokens[addr]; ok {
				if b, tOK := t[a]; tOK {
					return b, nil
				}
				return 0, errors.Errorf("unknown asset '%s' for address '%s'", a.String(), addr.String())
			}
			return 0, errors.Errorf("no asset balances for address '%s'", addr.String())
		}).Maybe()
	r.ms.EXPECT().NewestLeasingInfo(mock.Anything).RunAndReturn(
		func(id crypto.Digest) (*proto.LeaseInfo, error) {
			if l, ok := r.leasings[id]; ok {
				return l, nil
			}
			return nil, errors.Errorf("no leasing '%s'", id.String())
		}).Maybe()
	r.ms.EXPECT().NewestScriptBytesByAccount(mock.Anything).RunAndReturn(
		func(recipient proto.Recipient) (proto.Script, error) {
			addr, err := r.resolveRecipient(recipient)
			if err != nil {
				return nil, err
			}
			if s, ok := r.scripts[addr]; ok {
				return s, nil
			}
			return nil, nil
		}).Maybe()
	r.ms.EXPECT().IsNotFound(mock.Anything).RunAndReturn(func(err error) bool {
		return errors.Is(err, r.notFoundErr)
	}).Maybe()
	return r
}

func (e *testEnv) withScheme(scheme byte) *testEnv {
	unsetMockCalls(&e.me.Mock, "scheme")
	e.me.EXPECT().scheme().Return(scheme).Maybe()
	return e
}

func (e *testEnv) withLibVersion(v ast.LibraryVersion) *testEnv {
	unsetMockCalls(&e.me.Mock, "libVersion")
	unsetMockCalls(&e.me.Mock, "setLibVersion")
	e.me.EXPECT().libVersion().RunAndReturn(func() (ast.LibraryVersion, error) {
		return v, nil
	}).Maybe()
	e.me.EXPECT().setLibVersion(mock.Anything).Run(func(newV ast.LibraryVersion) {
		v = newV
	}).Return().Maybe()
	return e
}

func (e *testEnv) withComplexityLimit(limit int) *testEnv {
	require.True(e.t, limit >= 0)
	var cc complexityCalculator
	unsetMockCalls(&e.me.Mock, "complexityCalculator")
	unsetMockCalls(&e.me.Mock, "setComplexityCalculator")
	e.me.EXPECT().complexityCalculator().RunAndReturn(func() complexityCalculator {
		if cc != nil { // already initialized
			return cc
		}
		currentEnv := e.toEnv()
		isRideV6Activated := currentEnv.rideV6Activated() // We have to check if Ride V6 is activated, false by default
		cc = newComplexityCalculatorByRideV6Activation(isRideV6Activated)
		cc.setLimit(uint32(limit))
		return cc
	}).Maybe()
	e.me.EXPECT().setComplexityCalculator(mock.Anything).Run(func(newCC complexityCalculator) {
		cc = newCC
	}).Return().Maybe()
	return e
}

func (e *testEnv) withBlockV5Activated() *testEnv {
	unsetMockCalls(&e.me.Mock, "blockV5Activated")
	e.me.EXPECT().blockV5Activated().Return(true).Maybe()
	return e
}

func (e *testEnv) withBlock(blockInfo *proto.BlockInfo) *testEnv {
	unsetMockCalls(&e.me.Mock, "block")
	e.me.EXPECT().block().RunAndReturn(func() rideType {
		v, err := e.me.libVersion()
		if err != nil {
			panic(err)
		}
		return blockInfoToObject(blockInfo, v)
	}).Maybe()
	e.ms.EXPECT().AddingBlockHeight().RunAndReturn(func() (uint64, error) {
		return blockInfo.Height, nil
	}).Maybe()
	e.ms.EXPECT().NewestBlockInfoByHeight(mock.Anything).RunAndReturn(func(height uint64) (*proto.BlockInfo, error) {
		if height == blockInfo.Height {
			return blockInfo, nil
		}
		return nil, errors.Errorf("unexpected test height %d", height)
	}).Maybe()
	return e
}

func (e *testEnv) withProtobufTx() *testEnv {
	unsetMockCalls(&e.me.Mock, "isProtobufTx")
	e.me.EXPECT().isProtobufTx().Return(true).Maybe()
	return e
}

func (e *testEnv) withDataEntriesSizeV2() *testEnv {
	unsetMockCalls(&e.me.Mock, "maxDataEntriesSize")
	e.me.EXPECT().maxDataEntriesSize().Return(proto.MaxDataEntriesScriptActionsSizeInBytesV2).Maybe()
	return e
}

func (e *testEnv) withMessageLengthV3() *testEnv {
	unsetMockCalls(&e.me.Mock, "checkMessageLength")
	e.me.EXPECT().checkMessageLength(mock.Anything).RunAndReturn(bytesSizeCheckV3V6).Maybe()
	return e
}

func (e *testEnv) withRideV6Activated() *testEnv {
	unsetMockCalls(&e.me.Mock, "rideV6Activated")
	e.me.EXPECT().rideV6Activated().Return(true).Maybe()
	return e
}

func (e *testEnv) withConsensusImprovementsActivatedFunc() *testEnv {
	unsetMockCalls(&e.me.Mock, "consensusImprovementsActivated")
	e.me.EXPECT().consensusImprovementsActivated().Return(true).Maybe()
	return e
}

func (e *testEnv) withBlockRewardDistribution() *testEnv {
	unsetMockCalls(&e.me.Mock, "blockRewardDistributionActivated")
	e.me.EXPECT().blockRewardDistributionActivated().Return(true).Maybe()
	return e
}

func (e *testEnv) withLightNodeActivated() *testEnv {
	unsetMockCalls(&e.me.Mock, "lightNodeActivated")
	e.me.EXPECT().lightNodeActivated().Return(true).Maybe()
	return e
}

func (e *testEnv) withValidateInternalPayments() *testEnv {
	unsetMockCalls(&e.me.Mock, "validateInternalPayments")
	e.me.EXPECT().validateInternalPayments().Return(true).Maybe()
	return e
}

func (e *testEnv) withPaymentsFix() *testEnv {
	unsetMockCalls(&e.me.Mock, "paymentsFixActivated")
	e.me.EXPECT().paymentsFixActivated().Return(true).Maybe()
	return e
}

func (e *testEnv) withThis(acc *testAccount) *testEnv {
	e.this = acc.address()
	unsetMockCalls(&e.me.Mock, "this")
	e.me.EXPECT().this().RunAndReturn(func() rideType {
		return rideAddress(e.this)
	}).Maybe()
	return e
}

func (e *testEnv) withSender(acc *testAccount) *testEnv {
	e.sender = acc
	rcp := acc.recipient()
	e.recipients[rcp.String()] = acc.address()
	e.accounts[acc.address()] = acc
	return e
}

func (e *testEnv) withDApp(acc *testAccount) *testEnv {
	e.dApp = acc
	e.dAppAddr = e.dApp.address()
	unsetMockCalls(&e.me.Mock, "setNewDAppAddress")
	e.me.EXPECT().setNewDAppAddress(mock.Anything).Run(func(address proto.WavesAddress) {
		e.dAppAddr = address
		e.this = address
		if e.ws != nil {
			e.ws.cle = rideAddress(address) // We have to update wrapped state's `cle` if any
		}
	}).Return().Maybe()
	rcp := acc.recipient()
	e.recipients[rcp.String()] = acc.address()
	e.accounts[acc.address()] = acc
	return e
}

func (e *testEnv) withAdditionalDApp(acc *testAccount) *testEnv {
	rcp := acc.recipient()
	e.recipients[rcp.String()] = acc.address()
	e.accounts[acc.address()] = acc
	return e
}

type testInvocationOption func(*proto.InvokeScriptWithProofs)

func withRecipient(recipient proto.Recipient) testInvocationOption {
	return func(inv *proto.InvokeScriptWithProofs) {
		inv.ScriptRecipient = recipient
	}
}

func withTransactionID(id crypto.Digest) testInvocationOption {
	return func(inv *proto.InvokeScriptWithProofs) {
		inv.ID = &id
	}
}

func withPayments(payments ...proto.ScriptPayment) testInvocationOption {
	return func(inv *proto.InvokeScriptWithProofs) {
		inv.Payments = payments
	}
}

func (e *testEnv) withInvocation(fn string, opts ...testInvocationOption) *testEnv {
	call := proto.NewFunctionCall(fn, proto.Arguments{})
	tx := &proto.InvokeScriptWithProofs{
		Type:            proto.InvokeScriptTransaction,
		Version:         1,
		ID:              makeRandomTxID(e.t),
		Proofs:          proto.NewProofs(),
		SenderPK:        e.sender.publicKey(),
		ScriptRecipient: e.dApp.recipient(),
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	for _, opt := range opts {
		opt(tx)
	}
	return e.withInvokeTransaction(tx)
}

func (e *testEnv) withFullScriptTransfer(transfer *proto.FullScriptTransfer) *testEnv {
	unsetMockCalls(&e.me.Mock, "transaction")
	e.me.EXPECT().transaction().RunAndReturn(func() rideType {
		return scriptTransferToTransferTransactionObject(transfer)
	}).Maybe()
	return e
}

func (e *testEnv) withTransaction(tx proto.Transaction) *testEnv {
	unsetMockCalls(&e.me.Mock, "transaction")
	e.me.EXPECT().transaction().RunAndReturn(func() rideType {
		txo, err := transactionToObject(e.me, tx)
		require.NoError(e.t, err, "failed to set transaction")
		return txo
	}).Maybe()
	e.ms.EXPECT().NewestTransactionByID(mock.Anything).RunAndReturn(func(_ []byte) (proto.Transaction, error) {
		return tx, nil
	}).Maybe()
	id, err := tx.GetID(e.me.scheme())
	require.NoError(e.t, err)
	unsetMockCalls(&e.me.Mock, "txID")
	e.me.EXPECT().txID().RunAndReturn(func() rideType {
		return rideByteVector(id)
	}).Maybe()
	return e
}

func (e *testEnv) withTransactionID(id crypto.Digest) *testEnv {
	unsetMockCalls(&e.me.Mock, "txID")
	e.me.EXPECT().txID().RunAndReturn(func() rideType {
		return rideByteVector(id.Bytes())
	}).Maybe()
	return e
}

func (e *testEnv) withHeight(h int) *testEnv {
	unsetMockCalls(&e.me.Mock, "height")
	e.me.EXPECT().height().Return(rideInt(h)).Maybe()
	e.ms.EXPECT().AddingBlockHeight().RunAndReturn(func() (uint64, error) {
		return uint64(h), nil
	}).Maybe()
	return e
}

func (e *testEnv) withDataFromJSON(s string) *testEnv {
	var data []struct {
		Address string `json:"address"`
		Entry   struct {
			Key         string  `json:"key"`
			BoolValue   *bool   `json:"boolValue"`
			StringVale  *string `json:"stringValue"`
			IntValue    *string `json:"intValue"`
			BinaryValue *string `json:"binaryValue"`
		} `json:"entry"`
	}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		panic(err)
	}
	for _, d := range data {
		key := d.Entry.Key
		var entry proto.DataEntry
		switch {
		case d.Entry.BoolValue != nil:
			entry = &proto.BooleanDataEntry{Key: key, Value: *d.Entry.BoolValue}
		case d.Entry.StringVale != nil:
			entry = &proto.StringDataEntry{Key: key, Value: *d.Entry.StringVale}
		case d.Entry.IntValue != nil:
			v, err := strconv.ParseInt(*d.Entry.IntValue, 10, 64)
			require.NoError(e.t, err, "failed to add IntegerDataEntry")
			entry = &proto.IntegerDataEntry{Key: key, Value: v}
		case d.Entry.BinaryValue != nil:
			entry = &proto.BinaryDataEntry{Key: key, Value: guessBytesFromString(e.t, *d.Entry.BinaryValue)}
		}
		ab := guessBytesFromString(e.t, d.Address)
		addr, err := proto.NewAddressFromBytes(ab)
		require.NoError(e.t, err)
		e.addDataEntry(addr, entry)
	}
	return e
}

func (e *testEnv) withWrappedState() *testEnv {
	v, err := e.me.libVersion()
	require.NoError(e.t, err)
	if !hasMockExpectation(&e.me.Mock, "height") { // create stub height expectation
		e.me.EXPECT().height().Return(rideInt(0)).Maybe()
		defer func() { unsetMockCalls(&e.me.Mock, "height") }()
	}
	e.ws = newWrappedState(e.me, e.ms, v)
	unsetMockCalls(&e.me.Mock, "state")
	e.me.EXPECT().state().RunAndReturn(func() types.SmartState {
		return e.ws
	}).Maybe()
	return e
}
func (e *testEnv) withDataEntries(acc *testAccount, entries ...proto.DataEntry) *testEnv {
	for _, entry := range entries {
		e.addDataEntry(acc.address(), entry)
	}
	return e
}

func (e *testEnv) addDataEntry(addr proto.WavesAddress, entry proto.DataEntry) {
	if m, ok := e.entries[addr]; ok {
		m[entry.GetKey()] = entry
		e.entries[addr] = m
	} else {
		e.entries[addr] = map[string]proto.DataEntry{entry.GetKey(): entry}
	}
}

// withWavesBalance adds information about account's Waves balance profile.
// For the sake of brevity `lease in`, `lease out` and `generating` balances can be provided as last arguments in this order.
func (e *testEnv) withWavesBalance(acc *testAccount, balance int, other ...int) *testEnv {
	var (
		leaseIn    int64
		leaseOut   int64
		generating uint64
	)
	switch len(other) {
	case 3:
		leaseIn = int64(other[0])
		leaseOut = int64(other[1])
		generating = uint64(other[2])
	case 2:
		leaseIn = int64(other[0])
		leaseOut = int64(other[1])
	case 1:
		leaseIn = int64(other[0])
	case 0:
	default:
		e.t.Errorf("too many arguments provided as 'other' balances")
	}
	e.waves[acc.address()] = &types.WavesBalanceProfile{
		Balance:    uint64(balance),
		LeaseIn:    leaseIn,
		LeaseOut:   leaseOut,
		Generating: generating,
	}
	return e
}

func (e *testEnv) withTree(acc *testAccount, tree *ast.Tree) *testEnv {
	e.trees[acc.address()] = tree
	return e
}

func (e *testEnv) withAlias(acc *testAccount, alias *proto.Alias) *testEnv {
	e.aliases[*alias] = acc.address()
	e.recipients[acc.rcp.String()] = acc.address()
	return e
}

func (e *testEnv) withAsset(info *proto.FullAssetInfo) *testEnv {
	aID := proto.AssetIDFromDigest(info.ID)
	e.assets[aID] = info
	e.sponsorship[aID] = info.Sponsored
	return e
}

func (e *testEnv) withAssetBalance(acc *testAccount, asset crypto.Digest, balance uint64) *testEnv {
	if t, ok := e.tokens[acc.address()]; ok {
		t[asset] = balance
		e.tokens[acc.address()] = t
	} else {
		e.tokens[acc.address()] = map[crypto.Digest]uint64{asset: balance}
	}
	return e
}

func (e *testEnv) withTakeStringV5() *testEnv {
	unsetMockCalls(&e.me.Mock, "takeString")
	e.me.EXPECT().takeString(mock.Anything, mock.Anything).RunAndReturn(takeRideString).Maybe()
	return e
}

func (e *testEnv) toEnv() *MockEnvironment {
	return e.me
}

func (e *testEnv) retrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := e.resolveRecipient(account)
	if err != nil {
		return nil, err
	}
	if entries, ok := e.entries[addr]; ok {
		if e, ok := entries[key]; ok {
			return e, nil
		}
		return nil, errors.Wrapf(e.notFoundErr, "no entry by key '%s' at '%s'", key, addr.String())
	}
	return nil, errors.Wrapf(e.notFoundErr, "no entries for address '%s'", addr.String())
}

func (e *testEnv) withNoTransactionAtHeight() *testEnv {
	e.ms.EXPECT().NewestTransactionHeightByID(mock.Anything).RunAndReturn(func(_ []byte) (uint64, error) {
		return 0, e.notFoundErr
	}).Maybe()
	return e
}

func (e *testEnv) resolveRecipient(rcp proto.Recipient) (proto.WavesAddress, error) {
	if rcp.Address() != nil {
		return *rcp.Address(), nil
	}
	if a, ok := e.recipients[rcp.String()]; ok {
		return a, nil
	}
	return proto.WavesAddress{}, errors.Errorf("unknown recipient '%s'", rcp.String())
}

func (e *testEnv) withUntouchedState(acc *testAccount) *testEnv {
	e.ms.EXPECT().IsStateUntouched(mock.Anything).RunAndReturn(func(recipient proto.Recipient) (bool, error) {
		addr, err := e.resolveRecipient(recipient)
		if err != nil {
			return false, err
		}
		if addr == acc.address() {
			return true, nil
		}
		return false, errors.Errorf("unexpected recipient '%s'", recipient.String())
	}).Maybe()
	return e
}

func (e *testEnv) withInvokeTransaction(tx *proto.InvokeScriptWithProofs) *testEnv {
	var err error
	v, err := e.me.libVersion()
	if err != nil {
		panic(err)
	}
	e.inv, err = invocationToObject(v, e.me.scheme(), tx)
	require.NoError(e.t, err)
	unsetMockCalls(&e.me.Mock, "invocation")
	e.me.EXPECT().invocation().RunAndReturn(func() rideType {
		return e.inv
	}).Maybe()
	txo, err := transactionToObject(e.me, tx)
	require.NoError(e.t, err)
	unsetMockCalls(&e.me.Mock, "transaction")
	e.me.EXPECT().transaction().RunAndReturn(func() rideType {
		return txo
	}).Maybe()
	unsetMockCalls(&e.me.Mock, "setInvocation")
	e.me.EXPECT().setInvocation(mock.Anything).Run(func(inv rideType) {
		e.inv = inv
	}).Return().Maybe()
	unsetMockCalls(&e.me.Mock, "txID")
	e.me.EXPECT().txID().RunAndReturn(func() rideType {
		return rideByteVector(tx.ID.Bytes())
	}).Maybe()
	return e
}

func (e *testEnv) withLeasing(id crypto.Digest, info *proto.LeaseInfo) *testEnv {
	e.leasings[id] = info
	return e
}

func (e *testEnv) withScriptBytes(acc *testAccount, script proto.Script) *testEnv {
	e.scripts[acc.address()] = script
	return e
}

type blockBuilder struct {
	v         proto.BlockVersion
	ts        uint64
	h         uint64
	bt        uint64
	generator *testAccount
	gs        []byte
	vrf       []byte
	rewards   proto.Rewards
}

func protobufBlockBuilder() *blockBuilder {
	return &blockBuilder{v: proto.ProtobufBlockVersion}
}

func (bb *blockBuilder) toBlockInfo() *proto.BlockInfo {
	ga := proto.WavesAddress{}
	gpk := crypto.PublicKey{}
	if bb.generator != nil {
		ga = bb.generator.address()
		gpk = bb.generator.publicKey()
	}
	return &proto.BlockInfo{
		Version:             bb.v,
		Timestamp:           bb.ts,
		Height:              bb.h,
		BaseTarget:          bb.bt,
		Generator:           ga,
		GeneratorPublicKey:  gpk,
		GenerationSignature: bb.gs,
		VRF:                 bb.vrf,
		Rewards:             bb.rewards,
	}
}

func (bb *blockBuilder) withGenerator(generator *testAccount) *blockBuilder {
	bb.generator = generator
	return bb
}

func (bb *blockBuilder) withRewards(rewards proto.Rewards) *blockBuilder {
	bb.rewards = rewards
	return bb
}

func (bb *blockBuilder) withHeight(h uint64) *blockBuilder {
	bb.h = h
	return bb
}

func (bb *blockBuilder) withBaseTarget(baseTarget uint64) *blockBuilder {
	bb.bt = baseTarget
	return bb
}

func (bb *blockBuilder) withVRF(vrf []byte) *blockBuilder {
	bb.vrf = vrf
	return bb
}

func parseBase64Script(t *testing.T, src string) (proto.Script, *ast.Tree) {
	script, err := base64.StdEncoding.DecodeString(src)
	require.NoError(t, err)
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	require.NotNil(t, tree)
	return script, tree
}

func makeRandomTxID(t *testing.T) *crypto.Digest {
	b := make([]byte, crypto.DigestSize)
	_, err := rand.Read(b)
	require.NoError(t, err)
	d, err := crypto.NewDigestFromBytes(b)
	require.NoError(t, err)
	return &d
}

func guessBytesFromString(t *testing.T, s string) []byte {
	b, err := base58.Decode(s)
	if err != nil {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			b, err := hex.DecodeString(s)
			if err != nil {
				t.FailNow()
			}
			return b
		}
		return b
	}
	return b
}

func testTransferWithProofs(t *testing.T) *proto.TransferWithProofs {
	var scheme = proto.TestNetScheme
	seed, err := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair(seed)
	require.NoError(t, err)
	tm, err := time.Parse(time.RFC3339, "2020-10-01T00:00:00+00:00")
	require.NoError(t, err)
	ts := uint64(tm.UnixMilli())
	addr, err := proto.NewAddressFromPublicKey(scheme, pk)
	require.NoError(t, err)
	rcp := proto.NewRecipientFromAddress(addr)
	att := []byte("some attachment")
	tx := proto.NewUnsignedTransferWithProofs(3, pk, proto.OptionalAsset{}, proto.OptionalAsset{}, ts, 1234500000000, 100000, rcp, att)
	err = tx.GenerateID(scheme)
	require.NoError(t, err)
	err = tx.Sign(scheme, sk)
	require.NoError(t, err)
	return tx
}

func newTestDataTransactionWithEntries(t *testing.T, acc *testAccount, entries ...proto.DataEntry) *proto.DataWithProofs {
	data := proto.NewUnsignedDataWithProofs(1, acc.publicKey(), 10000, 1544715621)
	for i := range entries {
		err := data.AppendEntry(entries[i])
		require.NoError(t, err)
	}
	require.NoError(t, data.Sign(proto.TestNetScheme, acc.sk))
	return data
}

func newTestDataTransaction(t *testing.T, acc *testAccount) *proto.DataWithProofs {
	return newTestDataTransactionWithEntries(
		t,
		acc,
		&proto.IntegerDataEntry{
			Key:   "integer",
			Value: 100500,
		},
		&proto.BooleanDataEntry{
			Key:   "boolean",
			Value: true,
		},
		&proto.BinaryDataEntry{
			Key:   "binary",
			Value: []byte("hello"),
		},
		&proto.StringDataEntry{
			Key:   "string",
			Value: "world",
		},
	)
}
