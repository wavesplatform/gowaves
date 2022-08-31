package ride

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

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
	t          *testing.T
	sender     *testAccount
	dApp       *testAccount
	this       proto.WavesAddress
	dAppAddr   proto.WavesAddress
	inv        rideType
	me         *mockRideEnvironment
	ms         *MockSmartState
	ws         *WrappedState
	recipients map[string]proto.WavesAddress
	accounts   map[proto.WavesAddress]*testAccount
	entries    map[proto.WavesAddress]map[string]proto.DataEntry
	trees      map[proto.WavesAddress]*ast.Tree
	waves      map[proto.WavesAddress]*types.WavesBalanceProfile
}

func newTestEnv(t *testing.T) *testEnv {
	ms := &MockSmartState{}
	me := &mockRideEnvironment{
		schemeFunc: func() byte {
			return proto.TestNetScheme
		},
		stateFunc: func() types.SmartState {
			return ms
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
		checkMessageLengthFunc: func(n int) bool {
			return true // V2 by default
		},
		validateInternalPaymentsFunc: func() bool {
			return false
		},
		rideV6ActivatedFunc: func() bool {
			return false
		},
	}
	r := &testEnv{
		t:          t,
		me:         me,
		ms:         ms,
		recipients: map[string]proto.WavesAddress{},
		accounts:   map[proto.WavesAddress]*testAccount{},
		entries:    map[proto.WavesAddress]map[string]proto.DataEntry{},
		trees:      map[proto.WavesAddress]*ast.Tree{},
		waves:      map[proto.WavesAddress]*types.WavesBalanceProfile{},
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
		if a, ok := r.recipients[account.String()]; ok {
			if t, ok := r.trees[a]; ok {
				return t, nil
			}
			return nil, errors.Errorf("unknow address '%s'", a.String())
		}
		return nil, errors.Errorf("unknown recipient '%s'", account.String())
	}
	r.ms.NewestScriptVersionByAddressIDFunc = func(id proto.AddressID) (ast.LibraryVersion, error) {
		a, err := id.ToWavesAddress(r.me.scheme())
		require.NoError(r.t, err, "failed to recreate waves address")
		if t, ok := r.trees[a]; ok {
			return t.LibVersion, nil
		}
		return 0, errors.Errorf("unknown address '%s'", a.String())
	}
	r.ms.RetrieveNewestIntegerEntryFunc = func(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
		if addr, ok := r.recipients[account.String()]; ok {
			if entries, ok := r.entries[addr]; ok {
				if e, ok := entries[key]; ok {
					if ie, ok := e.(*proto.IntegerDataEntry); ok {
						return ie, nil
					}
					return nil, errors.Errorf("unxepected type '%T' of entry at '%s' by key '%s'", e, addr.String(), key)
				}
				return nil, errors.Errorf("no entry by key '%s' at '%s'", key, addr.String())
			}
			return nil, errors.Errorf("no entries for address '%s'", addr.String())
		}
		return nil, errors.Errorf("unknown recipient '%s'", account.String())
	}
	r.ms.NewestWavesBalanceFunc = func(account proto.Recipient) (uint64, error) {
		if addr, ok := r.recipients[account.String()]; ok {
			if profile, ok := r.waves[addr]; ok {
				return profile.Balance, nil
			}
			return 0, errors.Errorf("no balance profile for address '%s'", addr.String())
		}
		return 0, errors.Errorf("unknown recipient '%s'", account.String())
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
		if addr, ok := r.recipients[account.String()]; ok {
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
		return nil, errors.Errorf("unknown recipient '%s'", account.String())
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
	e.me.libVersionFunc = func() ast.LibraryVersion {
		return v
	}
	return e
}

func (e *testEnv) withBlockV5Activated() *testEnv {
	e.me.blockV5ActivatedFunc = func() bool {
		return true
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
	e.me.checkMessageLengthFunc = func(n int) bool {
		return n <= maxMessageLength
	}
	return e
}

func (e *testEnv) withRideV6Activated() *testEnv {
	e.me.rideV6ActivatedFunc = func() bool {
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

func (e *testEnv) withThis(addr proto.WavesAddress) *testEnv {
	e.this = addr
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

func (e *testEnv) withInvocation(fn string) *testEnv {
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
		ChainID:         proto.TestNetScheme,
		SenderPK:        e.sender.publicKey(),
		ScriptRecipient: e.dApp.recipient(),
		FunctionCall:    call,
		Payments:        proto.ScriptPayments{},
		FeeAsset:        proto.OptionalAsset{},
		Fee:             500000,
		Timestamp:       1624967106278,
	}
	var err error
	e.inv, err = invocationToObject(e.me.libVersion(), e.me.scheme(), tx)
	require.NoError(e.t, err)
	e.me.invocationFunc = func() rideType {
		return e.inv
	}
	txo, err := transactionToObject(e.me.scheme(), tx)
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

func (e *testEnv) withWrappedState() *testEnv {
	e.ws = &WrappedState{
		diff:                      newDiffState(e.ms),
		cle:                       e.me.this().(rideAddress),
		scheme:                    e.me.scheme(),
		rootScriptLibVersion:      e.me.libVersion(),
		rootActionsCountValidator: proto.NewScriptActionsCountValidator(),
	}
	e.me.stateFunc = func() types.SmartState {
		return e.ws
	}
	return e
}

func (e *testEnv) withIntegerEntries(addr proto.WavesAddress, entry *proto.IntegerDataEntry) *testEnv {
	if m, ok := e.entries[addr]; ok {
		m[entry.Key] = entry
		e.entries[addr] = m
	} else {
		e.entries[addr] = map[string]proto.DataEntry{entry.Key: entry}
	}
	return e
}

// withWavesBalance adds information about account's Waves balance profile.
// For the sake of brevity `lease in`, `lease out` and `generating` balances can be provided as last arguments in this order.
func (e *testEnv) withWavesBalance(acc *testAccount, balance int, other ...int) *testEnv {
	var leaseIn int64 = 0
	var leaseOut int64 = 0
	var generating uint64 = 0
	switch {
	case len(other) >= 3:
		leaseIn = int64(other[0])
		leaseOut = int64(other[1])
		generating = uint64(other[2])
	case len(other) == 2:
		leaseIn = int64(other[0])
		leaseOut = int64(other[1])
	case len(other) == 1:
		leaseIn = int64(other[0])
	}
	e.waves[acc.address()] = &types.WavesBalanceProfile{
		Balance:    uint64(balance),
		LeaseIn:    leaseIn,
		LeaseOut:   leaseOut,
		Generating: generating,
	}
	return e
}

func (e *testEnv) withTree(addr proto.WavesAddress, tree *ast.Tree) *testEnv {
	e.trees[addr] = tree
	return e
}

func (e *testEnv) toEnv() *mockRideEnvironment {
	return e.me
}

func parseBase64Script(t *testing.T, src string) (proto.Script, *ast.Tree) {
	script, err := base64.StdEncoding.DecodeString(src)
	require.NoError(t, err)
	tree, err := serialization.Parse(script)
	require.NoError(t, err)
	require.NotNil(t, tree)
	return script, tree
}

// makeAddressAndPK creates keys and an address on TestNet from given string as seed
// DEPRECATED
func makeAddressAndPK(t *testing.T, s string) (crypto.SecretKey, crypto.PublicKey, proto.WavesAddress) {
	sk, pk, err := crypto.GenerateKeyPair([]byte(s))
	require.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey(proto.TestNetScheme, pk)
	require.NoError(t, err)
	return sk, pk, addr
}

func makeRandomTxID(t *testing.T) *crypto.Digest {
	b := make([]byte, crypto.DigestSize)
	_, err := rand.Read(b)
	require.NoError(t, err)
	d, err := crypto.NewDigestFromBytes(b)
	require.NoError(t, err)
	return &d
}
