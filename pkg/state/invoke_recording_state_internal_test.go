package state

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// recordingStateTestCase describes a single call through recordingSmartState.
type recordingStateTestCase struct {
	name   string // Name of the case.
	method string // Name of the tested types.EnrichedSmartState method.
	// mock registers expectations on the underlying state.
	mock func(m *types.MockEnrichedSmartState)
	// call invokes the tested method on the view.
	call func(s types.EnrichedSmartState) (any, error)
	res  any         // Expected result.
	err  error       // Expected error, the very same value that is returned by the underlying state.
	keys []accessKey // Expected read set.
}

func recordingStateTestCases(t *testing.T) []recordingStateTestCase {
	_, addr := accessTestAccount(t, "recording-state-account")
	issuerPK, issuer := accessTestAccount(t, "recording-state-issuer")
	id := addr.ID()
	addrRcp := proto.NewRecipientFromAddress(addr)
	alias := *proto.NewAlias(accessTestScheme, "alice")
	aliasRcp := proto.NewRecipientFromAlias(alias)
	aliasKey := aliasAccessKey(alias.Alias)
	assetID := accessTestDigest(0x31)
	shortAssetID := proto.AssetIDFromDigest(assetID)
	leaseID := accessTestDigest(0x32)
	txID := accessTestDigest(0x33).Bytes()
	longTxID := make([]byte, crypto.SignatureSize)
	for i := range longTxID {
		longTxID[i] = 0x34
	}
	errMissing := errors.New("not found")
	errFailure := errors.New("failure")

	tree := &ast.Tree{LibVersion: ast.LibV5}
	script := proto.Script{0x01, 0x02, 0x03}
	tx := &proto.TransferWithProofs{Version: 2}
	leaseInfo := &proto.LeaseInfo{IsActive: true, LeaseAmount: 100, Recipient: addr, Sender: issuer}
	fullWavesBalance := &proto.FullWavesBalance{Regular: 10, Generating: 5, Available: 8, Effective: 9}
	intEntry := &proto.IntegerDataEntry{Key: "int", Value: 42}
	boolEntry := &proto.BooleanDataEntry{Key: "bool", Value: true}
	stringEntry := &proto.StringDataEntry{Key: "string", Value: "value"}
	binaryEntry := &proto.BinaryDataEntry{Key: "binary", Value: []byte{0xca, 0xfe}}
	entries := []proto.DataEntry{intEntry, boolEntry}
	constInfo := &proto.AssetConstInfo{ID: assetID, IssueHeight: 7, Issuer: issuer, Decimals: 8}
	assetInfo := &proto.AssetInfo{AssetConstInfo: *constInfo, Quantity: 1000, IssuerPublicKey: issuerPK}
	fullAssetInfo := &proto.FullAssetInfo{AssetInfo: *assetInfo, Name: "asset"}
	sponsoredAssetInfo := *assetInfo
	sponsoredAssetInfo.Sponsored = true
	fullSponsoredAssetInfo := &proto.FullAssetInfo{
		AssetInfo: sponsoredAssetInfo, Name: "sponsored", SponsorshipCost: 3, SponsorBalance: 500,
	}
	blockInfo := &proto.BlockInfo{Height: 100, Generator: addr}
	profile := &types.WavesBalanceProfile{Balance: 10, LeaseIn: 2, LeaseOut: 1, Generating: 5}

	return []recordingStateTestCase{
		// Account script.
		{
			name: "NewestScriptPKByAddr", method: "NewestScriptPKByAddr",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptPKByAddr(addr).Return(issuerPK, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptPKByAddr(addr) },
			res:  issuerPK,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptPKByAddr/not found", method: "NewestScriptPKByAddr",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptPKByAddr(addr).Return(crypto.PublicKey{}, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptPKByAddr(addr) },
			res:  crypto.PublicKey{}, err: errMissing,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptByAccount/address", method: "NewestScriptByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptByAccount(addrRcp).Return(tree, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptByAccount(addrRcp) },
			res:  tree,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptByAccount/alias", method: "NewestScriptByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().NewestScriptByAccount(aliasRcp).Return(tree, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptByAccount(aliasRcp) },
			res:  tree,
			keys: []accessKey{aliasKey, accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptByAccount/not found", method: "NewestScriptByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptByAccount(addrRcp).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptByAccount(addrRcp) },
			res:  (*ast.Tree)(nil), err: errMissing,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptBytesByAccount/address", method: "NewestScriptBytesByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptBytesByAccount(addrRcp).Return(script, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptBytesByAccount(addrRcp) },
			res:  script,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptBytesByAccount/alias", method: "NewestScriptBytesByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().NewestScriptBytesByAccount(aliasRcp).Return(script, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptBytesByAccount(aliasRcp) },
			res:  script,
			keys: []accessKey{aliasKey, accountScriptAccessKey(id)},
		},
		{
			name: "NewestScriptBytesByAccount/not found", method: "NewestScriptBytesByAccount",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptBytesByAccount(addrRcp).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptBytesByAccount(addrRcp) },
			res:  proto.Script(nil), err: errMissing,
			keys: []accessKey{accountScriptAccessKey(id)},
		},
		// Transactions.
		{
			name: "NewestTransactionByID", method: "NewestTransactionByID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestTransactionByID(txID).Return(tx, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestTransactionByID(txID) },
			res:  tx,
			keys: []accessKey{txAccessKey(txID)},
		},
		{
			name: "NewestTransactionByID/not found", method: "NewestTransactionByID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestTransactionByID(longTxID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestTransactionByID(longTxID) },
			res:  proto.Transaction(nil), err: errMissing,
			keys: []accessKey{txAccessKey(longTxID)},
		},
		{
			name: "NewestTransactionHeightByID", method: "NewestTransactionHeightByID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestTransactionHeightByID(txID).Return(uint64(77), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestTransactionHeightByID(txID) },
			res:  uint64(77),
			keys: []accessKey{txAccessKey(txID)},
		},
		{
			name: "NewestTransactionHeightByID/not found", method: "NewestTransactionHeightByID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestTransactionHeightByID(txID).Return(uint64(0), errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestTransactionHeightByID(txID) },
			res:  uint64(0), err: errMissing,
			keys: []accessKey{txAccessKey(txID)},
		},
		// Aliases.
		{
			name: "NewestRecipientToAddress/address", method: "NewestRecipientToAddress",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestRecipientToAddress(addrRcp).Return(addr, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestRecipientToAddress(addrRcp) },
			res:  addr,
			keys: nil, // Nothing is read from the state to convert an address recipient.
		},
		{
			name: "NewestRecipientToAddress/alias", method: "NewestRecipientToAddress",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestRecipientToAddress(aliasRcp).Return(addr, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestRecipientToAddress(aliasRcp) },
			res:  addr,
			keys: []accessKey{aliasKey},
		},
		{
			name: "NewestRecipientToAddress/alias not found", method: "NewestRecipientToAddress",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestRecipientToAddress(aliasRcp).Return(proto.WavesAddress{}, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestRecipientToAddress(aliasRcp) },
			res:  proto.WavesAddress{}, err: errMissing,
			keys: []accessKey{aliasKey},
		},
		{
			name: "NewestAddrByAlias", method: "NewestAddrByAlias",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAddrByAlias(alias) },
			res:  addr,
			keys: []accessKey{aliasKey},
		},
		{
			name: "NewestAddrByAlias/not found", method: "NewestAddrByAlias",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(proto.WavesAddress{}, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAddrByAlias(alias) },
			res:  proto.WavesAddress{}, err: errMissing,
			keys: []accessKey{aliasKey},
		},
		// Leases.
		{
			name: "NewestLeasingInfo", method: "NewestLeasingInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestLeasingInfo(leaseID).Return(leaseInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestLeasingInfo(leaseID) },
			res:  leaseInfo,
			keys: []accessKey{leaseAccessKey(leaseID)},
		},
		{
			name: "NewestLeasingInfo/not found", method: "NewestLeasingInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestLeasingInfo(leaseID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestLeasingInfo(leaseID) },
			res:  (*proto.LeaseInfo)(nil), err: errMissing,
			keys: []accessKey{leaseAccessKey(leaseID)},
		},
		// Account data.
		{
			name: "IsStateUntouched/address", method: "IsStateUntouched",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().IsStateUntouched(addrRcp).Return(true, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.IsStateUntouched(addrRcp) },
			res:  true,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "IsStateUntouched/alias", method: "IsStateUntouched",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().IsStateUntouched(aliasRcp).Return(false, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.IsStateUntouched(aliasRcp) },
			res:  false,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "IsStateUntouched/failure", method: "IsStateUntouched",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().IsStateUntouched(addrRcp).Return(false, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.IsStateUntouched(addrRcp) },
			res:  false, err: errFailure,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestIntegerEntry/address", method: "RetrieveNewestIntegerEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestIntegerEntry(addrRcp, "int").Return(intEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestIntegerEntry(addrRcp, "int") },
			res:  intEntry,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestIntegerEntry/alias", method: "RetrieveNewestIntegerEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().RetrieveNewestIntegerEntry(aliasRcp, "int").Return(intEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestIntegerEntry(aliasRcp, "int") },
			res:  intEntry,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestIntegerEntry/not found", method: "RetrieveNewestIntegerEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestIntegerEntry(addrRcp, "int").Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestIntegerEntry(addrRcp, "int") },
			res:  (*proto.IntegerDataEntry)(nil), err: errMissing,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBooleanEntry/address", method: "RetrieveNewestBooleanEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestBooleanEntry(addrRcp, "bool").Return(boolEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBooleanEntry(addrRcp, "bool") },
			res:  boolEntry,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBooleanEntry/alias", method: "RetrieveNewestBooleanEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().RetrieveNewestBooleanEntry(aliasRcp, "bool").Return(boolEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBooleanEntry(aliasRcp, "bool") },
			res:  boolEntry,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBooleanEntry/not found", method: "RetrieveNewestBooleanEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestBooleanEntry(addrRcp, "bool").Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBooleanEntry(addrRcp, "bool") },
			res:  (*proto.BooleanDataEntry)(nil), err: errMissing,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestStringEntry/address", method: "RetrieveNewestStringEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestStringEntry(addrRcp, "string").Return(stringEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestStringEntry(addrRcp, "string") },
			res:  stringEntry,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestStringEntry/alias", method: "RetrieveNewestStringEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().RetrieveNewestStringEntry(aliasRcp, "string").Return(stringEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestStringEntry(aliasRcp, "string") },
			res:  stringEntry,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestStringEntry/not found", method: "RetrieveNewestStringEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestStringEntry(addrRcp, "string").Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestStringEntry(addrRcp, "string") },
			res:  (*proto.StringDataEntry)(nil), err: errMissing,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBinaryEntry/address", method: "RetrieveNewestBinaryEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestBinaryEntry(addrRcp, "binary").Return(binaryEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBinaryEntry(addrRcp, "binary") },
			res:  binaryEntry,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBinaryEntry/alias", method: "RetrieveNewestBinaryEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().RetrieveNewestBinaryEntry(aliasRcp, "binary").Return(binaryEntry, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBinaryEntry(aliasRcp, "binary") },
			res:  binaryEntry,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "RetrieveNewestBinaryEntry/not found", method: "RetrieveNewestBinaryEntry",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveNewestBinaryEntry(addrRcp, "binary").Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveNewestBinaryEntry(addrRcp, "binary") },
			res:  (*proto.BinaryDataEntry)(nil), err: errMissing,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveEntries/address", method: "RetrieveEntries",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveEntries(addrRcp).Return(entries, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveEntries(addrRcp) },
			res:  entries,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		{
			name: "RetrieveEntries/alias", method: "RetrieveEntries",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().RetrieveEntries(aliasRcp).Return(entries, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveEntries(aliasRcp) },
			res:  entries,
			keys: []accessKey{aliasKey, accountDataAccessKey(id)},
		},
		{
			name: "RetrieveEntries/failure", method: "RetrieveEntries",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().RetrieveEntries(addrRcp).Return(nil, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.RetrieveEntries(addrRcp) },
			res:  []proto.DataEntry(nil), err: errFailure,
			keys: []accessKey{accountDataAccessKey(id)},
		},
		// Account balances.
		{
			name: "NewestWavesBalance/address", method: "NewestWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestWavesBalance(addrRcp).Return(uint64(1000), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestWavesBalance(addrRcp) },
			res:  uint64(1000),
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestWavesBalance/alias", method: "NewestWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().NewestWavesBalance(aliasRcp).Return(uint64(1000), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestWavesBalance(aliasRcp) },
			res:  uint64(1000),
			keys: []accessKey{aliasKey, accountBalanceAccessKey(id)},
		},
		{
			name: "NewestWavesBalance/alias not found", method: "NewestWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(proto.WavesAddress{}, errMissing).Once()
				m.EXPECT().NewestWavesBalance(aliasRcp).Return(uint64(0), errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestWavesBalance(aliasRcp) },
			res:  uint64(0), err: errFailure, // The error of the delegated call, not of the alias resolution.
			keys: []accessKey{aliasKey},
		},
		{
			name: "NewestWavesBalance/invalid recipient", method: "NewestWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestWavesBalance(proto.Recipient{}).Return(uint64(0), errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestWavesBalance(proto.Recipient{}) },
			res:  uint64(0), err: errFailure,
			keys: nil,
		},
		{
			name: "NewestFullWavesBalance/address", method: "NewestFullWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestFullWavesBalance(addrRcp).Return(fullWavesBalance, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullWavesBalance(addrRcp) },
			res:  fullWavesBalance,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestFullWavesBalance/alias", method: "NewestFullWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().NewestFullWavesBalance(aliasRcp).Return(fullWavesBalance, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullWavesBalance(aliasRcp) },
			res:  fullWavesBalance,
			keys: []accessKey{aliasKey, accountBalanceAccessKey(id)},
		},
		{
			name: "NewestFullWavesBalance/failure", method: "NewestFullWavesBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestFullWavesBalance(addrRcp).Return(nil, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullWavesBalance(addrRcp) },
			res:  (*proto.FullWavesBalance)(nil), err: errFailure,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestAssetBalance/address", method: "NewestAssetBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetBalance(addrRcp, assetID).Return(uint64(55), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetBalance(addrRcp, assetID) },
			res:  uint64(55),
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestAssetBalance/alias", method: "NewestAssetBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAddrByAlias(alias).Return(addr, nil).Once()
				m.EXPECT().NewestAssetBalance(aliasRcp, assetID).Return(uint64(55), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetBalance(aliasRcp, assetID) },
			res:  uint64(55),
			keys: []accessKey{aliasKey, accountBalanceAccessKey(id)},
		},
		{
			name: "NewestAssetBalance/failure", method: "NewestAssetBalance",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetBalance(addrRcp, assetID).Return(uint64(0), errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetBalance(addrRcp, assetID) },
			res:  uint64(0), err: errFailure,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "WavesBalanceProfile", method: "WavesBalanceProfile",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().WavesBalanceProfile(id).Return(profile, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.WavesBalanceProfile(id) },
			res:  profile,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "WavesBalanceProfile/failure", method: "WavesBalanceProfile",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().WavesBalanceProfile(id).Return(nil, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.WavesBalanceProfile(id) },
			res:  (*types.WavesBalanceProfile)(nil), err: errFailure,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestAssetBalanceByAddressID", method: "NewestAssetBalanceByAddressID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetBalanceByAddressID(id, assetID).Return(uint64(66), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetBalanceByAddressID(id, assetID) },
			res:  uint64(66),
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		{
			name: "NewestAssetBalanceByAddressID/failure", method: "NewestAssetBalanceByAddressID",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetBalanceByAddressID(id, assetID).Return(uint64(0), errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetBalanceByAddressID(id, assetID) },
			res:  uint64(0), err: errFailure,
			keys: []accessKey{accountBalanceAccessKey(id)},
		},
		// Assets.
		{
			name: "NewestAssetIsSponsored", method: "NewestAssetIsSponsored",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetIsSponsored(assetID).Return(true, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetIsSponsored(assetID) },
			res:  true,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestAssetIsSponsored/failure", method: "NewestAssetIsSponsored",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetIsSponsored(assetID).Return(false, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetIsSponsored(assetID) },
			res:  false, err: errFailure,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestAssetConstInfo", method: "NewestAssetConstInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetConstInfo(shortAssetID).Return(constInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetConstInfo(shortAssetID) },
			res:  constInfo,
			keys: []accessKey{assetAccessKey(assetID)}, // The same key as for the full asset ID.
		},
		{
			name: "NewestAssetConstInfo/not found", method: "NewestAssetConstInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetConstInfo(shortAssetID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetConstInfo(shortAssetID) },
			res:  (*proto.AssetConstInfo)(nil), err: errMissing,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestAssetInfo", method: "NewestAssetInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetInfo(assetID).Return(assetInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetInfo(assetID) },
			res:  assetInfo,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestAssetInfo/not found", method: "NewestAssetInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestAssetInfo(assetID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestAssetInfo(assetID) },
			res:  (*proto.AssetInfo)(nil), err: errMissing,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestFullAssetInfo", method: "NewestFullAssetInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestFullAssetInfo(assetID).Return(fullAssetInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullAssetInfo(assetID) },
			res:  fullAssetInfo,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestFullAssetInfo/sponsored", method: "NewestFullAssetInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestFullAssetInfo(assetID).Return(fullSponsoredAssetInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullAssetInfo(assetID) },
			res:  fullSponsoredAssetInfo,
			// The balance of the sponsor is a part of the full info of a sponsored asset.
			keys: []accessKey{assetAccessKey(assetID), accountBalanceAccessKey(issuer.ID())},
		},
		{
			name: "NewestFullAssetInfo/not found", method: "NewestFullAssetInfo",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestFullAssetInfo(assetID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestFullAssetInfo(assetID) },
			res:  (*proto.FullAssetInfo)(nil), err: errMissing,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestScriptByAsset", method: "NewestScriptByAsset",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptByAsset(assetID).Return(tree, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptByAsset(assetID) },
			res:  tree,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		{
			name: "NewestScriptByAsset/not found", method: "NewestScriptByAsset",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestScriptByAsset(assetID).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestScriptByAsset(assetID) },
			res:  (*ast.Tree)(nil), err: errMissing,
			keys: []accessKey{assetAccessKey(assetID)},
		},
		// Constant within a block, nothing is recorded.
		{
			name: "AddingBlockHeight", method: "AddingBlockHeight",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().AddingBlockHeight().Return(uint64(101), nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.AddingBlockHeight() },
			res:  uint64(101),
		},
		{
			name: "AddingBlockHeight/failure", method: "AddingBlockHeight",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().AddingBlockHeight().Return(uint64(0), errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.AddingBlockHeight() },
			res:  uint64(0), err: errFailure,
		},
		{
			name: "NewestBlockInfoByHeight", method: "NewestBlockInfoByHeight",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestBlockInfoByHeight(proto.Height(100)).Return(blockInfo, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestBlockInfoByHeight(100) },
			res:  blockInfo,
		},
		{
			name: "NewestBlockInfoByHeight/not found", method: "NewestBlockInfoByHeight",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().NewestBlockInfoByHeight(proto.Height(100)).Return(nil, errMissing).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.NewestBlockInfoByHeight(100) },
			res:  (*proto.BlockInfo)(nil), err: errMissing,
		},
		{
			name: "EstimatorVersion", method: "EstimatorVersion",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().EstimatorVersion().Return(4, nil).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.EstimatorVersion() },
			res:  4,
		},
		{
			name: "EstimatorVersion/failure", method: "EstimatorVersion",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().EstimatorVersion().Return(0, errFailure).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.EstimatorVersion() },
			res:  0, err: errFailure,
		},
		{
			name: "IsNotFound/true", method: "IsNotFound",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().IsNotFound(errMissing).Return(true).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.IsNotFound(errMissing), nil },
			res:  true,
		},
		{
			name: "IsNotFound/false", method: "IsNotFound",
			mock: func(m *types.MockEnrichedSmartState) {
				m.EXPECT().IsNotFound(errFailure).Return(false).Once()
			},
			call: func(s types.EnrichedSmartState) (any, error) { return s.IsNotFound(errFailure), nil },
			res:  false,
		},
	}
}

func TestRecordingSmartState(t *testing.T) {
	for _, test := range recordingStateTestCases(t) {
		t.Run(test.name, func(t *testing.T) {
			m := types.NewMockEnrichedSmartState(t)
			test.mock(m)
			rs := newRecordingSmartState(m)
			require.Zero(t, rs.readSet().len())

			res, err := test.call(rs)

			if test.err != nil {
				assert.Same(t, test.err, err, "error must be passed through unchanged")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.res, res)
			assert.Equal(t, newAccessSet(test.keys...), rs.readSet())
		})
	}
}

// TestRecordingSmartStateCoversInterface checks that every method of types.EnrichedSmartState has a test case,
// so a new method of the interface can not be left without a decision on what it reads.
func TestRecordingSmartStateCoversInterface(t *testing.T) {
	covered := make(map[string]struct{})
	for _, test := range recordingStateTestCases(t) {
		covered[test.method] = struct{}{}
	}
	iface := reflect.TypeFor[types.EnrichedSmartState]()
	for method := range iface.Methods() {
		name := method.Name
		assert.Contains(t, covered, name, "no test case for method %s", name)
		delete(covered, name)
	}
	assert.Empty(t, covered, "test cases for methods that are not in the interface")
}

// TestRecordingSmartStateRecordsBeforeDelegation checks that the key is already in the read set when the wrapped
// state is called, so the read is recorded even if the wrapped state never returns normally.
func TestRecordingSmartStateRecordsBeforeDelegation(t *testing.T) {
	_, addr := accessTestAccount(t, "recording-state-order")
	rcp := proto.NewRecipientFromAddress(addr)
	m := types.NewMockEnrichedSmartState(t)
	rs := newRecordingSmartState(m)
	m.EXPECT().RetrieveNewestIntegerEntry(rcp, "key").
		RunAndReturn(func(proto.Recipient, string) (*proto.IntegerDataEntry, error) {
			assert.True(t, rs.readSet().has(accountDataAccessKey(addr.ID())))
			panic("underlying state failure")
		}).Once()

	assert.PanicsWithValue(t, "underlying state failure", func() {
		_, _ = rs.RetrieveNewestIntegerEntry(rcp, "key")
	})
	assert.Equal(t, newAccessSet(accountDataAccessKey(addr.ID())), rs.readSet())
}

func TestRecordingSmartStateAccumulatesReads(t *testing.T) {
	_, addr1 := accessTestAccount(t, "recording-state-accumulate-1")
	_, addr2 := accessTestAccount(t, "recording-state-accumulate-2")
	rcp1 := proto.NewRecipientFromAddress(addr1)
	rcp2 := proto.NewRecipientFromAddress(addr2)
	assetID := accessTestDigest(0x41)
	errMissing := errors.New("not found")

	m := types.NewMockEnrichedSmartState(t)
	m.EXPECT().NewestWavesBalance(rcp1).Return(uint64(1), nil).Twice()
	m.EXPECT().NewestAssetBalance(rcp1, assetID).Return(uint64(2), nil).Once()
	m.EXPECT().RetrieveNewestStringEntry(rcp2, "key").Return(nil, errMissing).Once()
	m.EXPECT().NewestAssetInfo(assetID).Return(nil, errMissing).Once()
	m.EXPECT().AddingBlockHeight().Return(uint64(10), nil).Once()

	rs := newRecordingSmartState(m)
	_, err := rs.NewestWavesBalance(rcp1)
	require.NoError(t, err)
	_, err = rs.NewestWavesBalance(rcp1)
	require.NoError(t, err)
	_, err = rs.NewestAssetBalance(rcp1, assetID)
	require.NoError(t, err)
	_, err = rs.RetrieveNewestStringEntry(rcp2, "key")
	require.ErrorIs(t, err, errMissing)
	_, err = rs.NewestAssetInfo(assetID)
	require.ErrorIs(t, err, errMissing)
	_, err = rs.AddingBlockHeight()
	require.NoError(t, err)

	assert.Equal(t, newAccessSet(
		accountBalanceAccessKey(addr1.ID()),
		accountDataAccessKey(addr2.ID()),
		assetAccessKey(assetID),
	), rs.readSet())

	// Views are independent of each other.
	other := newRecordingSmartState(m)
	assert.Zero(t, other.readSet().len())
}
