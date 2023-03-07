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
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/types"
)

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
func newTestAccountFromAddresString(t *testing.T, addr string) *testAccount {
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
	me          *mockRideEnvironment
	ms          *MockSmartState
	ws          *WrappedState
	recipients  map[string]proto.WavesAddress
	accounts    map[proto.WavesAddress]*testAccount
	entries     map[proto.WavesAddress]map[string]proto.DataEntry
	trees       map[proto.WavesAddress]*ast.Tree
	waves       map[proto.WavesAddress]*types.WavesBalanceProfile
	aliases     map[proto.Alias]proto.WavesAddress
	assets      map[crypto.Digest]*proto.FullAssetInfo
	sponsorship map[crypto.Digest]bool
	tokens      map[proto.WavesAddress]map[crypto.Digest]uint64
	leasings    map[crypto.Digest]*proto.LeaseInfo
	scripts     map[proto.WavesAddress]proto.Script
}

func newTestEnv(t *testing.T) *testEnv {
	me := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		blockV5ActivatedFunc: func() bool {
			return false
		},
		isProtobufTxFunc: func() bool {
			return false
		},
		maxDataEntriesSizeFunc: func() int {
			return proto.MaxDataEntriesScriptActionsSizeInBytesV1 // V1 by default
		},
		checkMessageLengthFunc: bytesSizeCheckV1V2,
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		rideV6ActivatedFunc: func() bool {
			return false
		},
		consensusImprovementsActivatedFunc: func() bool {
			return false
		},
		invokeExpressionActivatedFunc: func() bool {
			return false
		},
	}
	r := &testEnv{
		t:           t,
		me:          me,
		ms:          &MockSmartState{},
		recipients:  map[string]proto.WavesAddress{},
		accounts:    map[proto.WavesAddress]*testAccount{},
		entries:     map[proto.WavesAddress]map[string]proto.DataEntry{},
		trees:       map[proto.WavesAddress]*ast.Tree{},
		waves:       map[proto.WavesAddress]*types.WavesBalanceProfile{},
		aliases:     map[proto.Alias]proto.WavesAddress{},
		assets:      map[crypto.Digest]*proto.FullAssetInfo{},
		sponsorship: map[crypto.Digest]bool{},
		tokens:      map[proto.WavesAddress]map[crypto.Digest]uint64{},
		leasings:    map[crypto.Digest]*proto.LeaseInfo{},
		scripts:     map[proto.WavesAddress]proto.Script{},
	}
	r.me.stateFunc = func() types.SmartState {
		return r.ms
	}
	r.ms.NewestRecipientToAddressFunc = func(recipient proto.Recipient) (*proto.WavesAddress, error) {
		if a, ok := r.recipients[recipient.String()]; ok {
			return &a, nil
		}
		return nil, errors.Errorf("unknown recipient '%s'", recipient.String())
	}
	r.ms.NewestScriptPKByAddrFunc = func(addr proto.WavesAddress) (crypto.PublicKey, error) {
		if acc, ok := r.accounts[addr]; ok {
			return acc.publicKey(), nil
		}
		return crypto.PublicKey{}, errors.Errorf("unknown address '%s'", addr.String())
	}
	r.ms.NewestScriptByAccountFunc = func(account proto.Recipient) (*ast.Tree, error) {
		addr, err := r.resolveRecipient(account)
		if err != nil {
			return nil, err
		}
		if t, ok := r.trees[addr]; ok {
			return t, nil
		}
		return nil, errors.Errorf("unknow address '%s'", addr.String())
	}
	r.ms.NewestScriptVersionByAddressIDFunc = func(id proto.AddressID) (ast.LibraryVersion, error) {
		a, err := id.ToWavesAddress(r.me.scheme())
		require.NoError(r.t, err, "failed to recreate waves address")
		if t, ok := r.trees[a]; ok {
			return t.LibVersion, nil
		}
		return 0, errors.Errorf("unknown address '%s'", a.String())
	}
	r.ms.RetrieveNewestBinaryEntryFunc = func(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
		e, err := r.retrieveEntry(account, key)
		if err != nil {
			return nil, err
		}
		if be, ok := e.(*proto.BinaryDataEntry); ok {
			return be, nil
		}
		return nil, errors.Errorf("unxepected type '%T' of entry at '%s' by key '%s'", e, account.String(), key)
	}
	r.ms.RetrieveNewestBooleanEntryFunc = func(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
		e, err := r.retrieveEntry(account, key)
		if err != nil {
			return nil, err
		}
		if be, ok := e.(*proto.BooleanDataEntry); ok {
			return be, nil
		}
		return nil, errors.Errorf("unxepected type '%T' of entry at '%s' by key '%s'", e, account.String(), key)
	}
	r.ms.RetrieveNewestIntegerEntryFunc = func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
		e, err := r.retrieveEntry(account, key)
		if err != nil {
			return nil, err
		}
		if be, ok := e.(*proto.IntegerDataEntry); ok {
			return be, nil
		}
		return nil, errors.Errorf("unxepected type '%T' of entry at '%s' by key '%s'", e, account.String(), key)
	}
	r.ms.RetrieveNewestStringEntryFunc = func(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
		e, err := r.retrieveEntry(account, key)
		if err != nil {
			return nil, err
		}
		if be, ok := e.(*proto.StringDataEntry); ok {
			return be, nil
		}
		return nil, errors.Errorf("unxepected type '%T' of entry at '%s' by key '%s'", e, account.String(), key)
	}
	r.ms.NewestWavesBalanceFunc = func(account proto.Recipient) (uint64, error) {
		addr, err := r.resolveRecipient(account)
		if err != nil {
			return 0, err
		}
		if profile, ok := r.waves[addr]; ok {
			return profile.Balance, nil
		}
		return 0, errors.Errorf("no balance profile for address '%s'", addr.String())
	}
	r.ms.WavesBalanceProfileFunc = func(id proto.AddressID) (*types.WavesBalanceProfile, error) {
		addr, err := id.ToWavesAddress(r.me.scheme())
		require.NoError(r.t, err)
		if profile, ok := r.waves[addr]; ok {
			return profile, nil
		}
		return nil, errors.Errorf("no balance profile for address '%s'", addr.String())
	}
	r.ms.NewestFullWavesBalanceFunc = func(account proto.Recipient) (*proto.FullWavesBalance, error) {
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
	}
	r.ms.NewestAddrByAliasFunc = func(alias proto.Alias) (proto.WavesAddress, error) {
		if a, ok := r.aliases[alias]; ok {
			return a, nil
		}
		return proto.WavesAddress{}, errors.Errorf("unknown alias '%s'", alias.String())
	}
	r.ms.NewestAssetIsSponsoredFunc = func(assetID crypto.Digest) (bool, error) {
		if s, ok := r.sponsorship[assetID]; ok {
			return s, nil
		}
		return false, errors.Errorf("unknown asset '%s'", assetID.String())
	}
	r.ms.NewestAssetInfoFunc = func(assetID crypto.Digest) (*proto.AssetInfo, error) {
		if ai, ok := r.assets[assetID]; ok {
			return &ai.AssetInfo, nil
		}
		return nil, errors.Errorf("unknown asset '%s'", assetID.String())
	}
	r.ms.NewestFullAssetInfoFunc = func(assetID crypto.Digest) (*proto.FullAssetInfo, error) {
		if ai, ok := r.assets[assetID]; ok {
			return ai, nil
		}
		return nil, errors.Errorf("unknown asset '%s'", assetID.String())
	}
	r.ms.NewestAssetBalanceFunc = func(account proto.Recipient, assetID crypto.Digest) (uint64, error) {
		addr, err := r.resolveRecipient(account)
		if err != nil {
			return 0, err
		}
		if balances, ok := r.tokens[addr]; ok {
			if b, ok := balances[assetID]; ok {
				return b, nil
			}
			return 0, errors.Errorf("unknown asset '%s' for address '%s'", assetID.String(), addr.String())
		}
		return 0, errors.Errorf("no asset balances for address '%s'", addr.String())
	}
	r.ms.NewestAssetBalanceByAddressIDFunc = func(id proto.AddressID, a crypto.Digest) (uint64, error) {
		addr, err := id.ToWavesAddress(r.me.scheme())
		require.NoError(r.t, err)
		if t, ok := r.tokens[addr]; ok {
			if b, ok := t[a]; ok {
				return b, nil
			}
			return 0, errors.Errorf("unknown asset '%s' for address '%s'", a.String(), addr.String())
		}
		return 0, errors.Errorf("no asset balances for address '%s'", addr.String())
	}
	r.ms.NewestLeasingInfoFunc = func(id crypto.Digest) (*proto.LeaseInfo, error) {
		if l, ok := r.leasings[id]; ok {
			return l, nil
		}
		return nil, errors.Errorf("no leasing '%s'", id.String())
	}
	r.ms.NewestScriptBytesByAccountFunc = func(recipient proto.Recipient) (proto.Script, error) {
		addr, err := r.resolveRecipient(recipient)
		if err != nil {
			return nil, err
		}
		if s, ok := r.scripts[addr]; ok {
			return s, nil
		}
		return nil, nil
	}
	return r
}

func (e *testEnv) withScheme(scheme byte) *testEnv {
	e.me.schemeFunc = func() byte {
		return scheme
	}
	return e
}

func (e *testEnv) withLibVersion(v ast.LibraryVersion) *testEnv {
	e.me.libVersionFunc = func() (ast.LibraryVersion, error) {
		return v, nil
	}
	e.me.setLibVersionFunc = func(newV ast.LibraryVersion) {
		v = newV
	}
	return e
}

func (e *testEnv) withComplexityLimit(v ast.LibraryVersion, limit int) *testEnv {
	require.True(e.t, limit >= 0)
	cc := newComplexityCalculator(v, uint32(limit))
	e.me.complexityCalculatorFunc = func() complexityCalculator {
		return cc
	}
	return e
}

func (e *testEnv) withBlockV5Activated() *testEnv {
	e.me.blockV5ActivatedFunc = func() bool {
		return true
	}
	return e
}

func (e *testEnv) withBlock(blockInfo *proto.BlockInfo) *testEnv {
	e.me.blockFunc = func() rideType {
		return blockInfoToObject(blockInfo)
	}
	e.ms.AddingBlockHeightFunc = func() (uint64, error) {
		return blockInfo.Height, nil
	}
	return e
}

func (e *testEnv) withProtobufTx() *testEnv {
	e.me.isProtobufTxFunc = func() bool {
		return true
	}
	return e
}

func (e *testEnv) withDataEntriesSizeV2() *testEnv {
	e.me.maxDataEntriesSizeFunc = func() int {
		return proto.MaxDataEntriesScriptActionsSizeInBytesV2
	}
	return e
}

func (e *testEnv) withMessageLengthV3() *testEnv {
	e.me.checkMessageLengthFunc = bytesSizeCheckV3V6
	return e
}

func (e *testEnv) withRideV6Activated() *testEnv {
	e.me.rideV6ActivatedFunc = func() bool {
		return true
	}
	return e
}

func (e *testEnv) withInvokeExpressionActivated() *testEnv {
	e.me.invokeExpressionActivatedFunc = func() bool {
		return true
	}
	return e
}

func (e *testEnv) withValidateInternalPayments() *testEnv {
	e.me.validateInternalPaymentsFunc = func() bool {
		return true
	}
	return e
}

func (e *testEnv) withThis(acc *testAccount) *testEnv {
	e.this = acc.address()
	e.me.thisFunc = func() rideType {
		return rideAddress(e.this)
	}
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
	e.me.setNewDAppAddressFunc = func(address proto.WavesAddress) {
		e.dAppAddr = address
		e.this = address
		if e.ws != nil {
			e.ws.cle = rideAddress(address) // We have to update wrapped state's `cle` if any
		}
	}
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
	call := proto.FunctionCall{
		Default:   false,
		Name:      fn,
		Arguments: proto.Arguments{},
	}
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

func (e *testEnv) withTransactionObject(txo rideType) *testEnv {
	e.me.transactionFunc = func() rideType {
		return txo
	}
	return e
}

func (e *testEnv) withTransaction(tx proto.Transaction) *testEnv {
	// TODO: hardcoded scheme
	e.me.transactionFunc = func() rideType {
		v, err := e.me.libVersion()
		if err != nil {
			panic(err)
		}
		txo, err := transactionToObject(v, proto.TestNetScheme, e.me.invokeExpressionActivated(), tx)
		require.NoError(e.t, err, "failed to set transaction")
		return txo
	}
	e.ms.NewestTransactionByIDFunc = func(id []byte) (proto.Transaction, error) {
		return tx, nil
	}
	id, err := tx.GetID(e.me.scheme())
	require.NoError(e.t, err)
	e.me.txIDFunc = func() rideType {
		return rideBytes(id)
	}
	return e
}

func (e *testEnv) withTransactionID(id crypto.Digest) *testEnv {
	e.me.txIDFunc = func() rideType {
		return rideBytes(id.Bytes())
	}
	return e
}

func (e *testEnv) withHeight(h int) *testEnv {
	e.me.heightFunc = func() rideInt {
		return rideInt(h)
	}
	e.ms.AddingBlockHeightFunc = func() (uint64, error) {
		return uint64(h), nil
	}
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
	if err != nil {
		panic(err)
	}
	e.ws = &WrappedState{
		diff:                      newDiffState(e.ms),
		cle:                       e.me.this().(rideAddress),
		scheme:                    e.me.scheme(),
		rootScriptLibVersion:      v,
		rootActionsCountValidator: proto.NewScriptActionsCountValidator(),
	}
	e.me.stateFunc = func() types.SmartState {
		return e.ws
	}
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
		e.t.Errorf("too many arguments provided as 'other' balaces")
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
	e.assets[info.ID] = info
	e.sponsorship[info.ID] = info.Sponsored
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
	e.me.takeStringFunc = takeRideString
	return e
}

func (e *testEnv) toEnv() *mockRideEnvironment {
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
		return nil, errors.Errorf("no entry by key '%s' at '%s'", key, addr.String())
	}
	return nil, errors.Errorf("no entries for address '%s'", addr.String())
}

func (e *testEnv) withNoTransactionAtHeight() *testEnv {
	e.ms.NewestTransactionHeightByIDFunc = func(_ []byte) (uint64, error) {
		return 0, proto.ErrNotFound
	}
	e.ms.IsNotFoundFunc = func(err error) bool {
		return true
	}
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
	e.ms.IsStateUntouchedFunc = func(recipient proto.Recipient) (bool, error) {
		addr, err := e.resolveRecipient(recipient)
		if err != nil {
			return false, err
		}
		if addr == acc.address() {
			return true, nil
		}
		return false, errors.Errorf("unexpected recipient '%s'", recipient.String())
	}
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
	e.me.invocationFunc = func() rideType {
		return e.inv
	}
	txo, err := transactionToObject(v, e.me.scheme(), e.me.invokeExpressionActivated(), tx)
	require.NoError(e.t, err)
	e.me.transactionFunc = func() rideType {
		return txo
	}
	e.me.setInvocationFunc = func(inv rideType) {
		e.inv = inv
	}
	e.me.txIDFunc = func() rideType {
		return rideBytes(tx.ID.Bytes())
	}
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
