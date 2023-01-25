package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const (
	rollbackMaxBlocks     = 2000
	blocksStorDir         = "blocks_storage"
	keyvalueDir           = "key_value"
	maxScriptsRunsInBlock = 101
)

var empty struct{}

func wrapErr(stateErrorType ErrorType, err error) error {
	switch err.(type) {
	case StateError:
		return err
	default:
		return NewStateError(stateErrorType, err)
	}
}

type blockchainEntitiesStorage struct {
	hs                *historyStorage
	aliases           *aliases
	assets            *assets
	leases            *leases
	scores            *scores
	blocksInfo        *blocksInfo
	balances          *balances
	features          featuresState
	monetaryPolicy    *monetaryPolicy
	ordersVolumes     *ordersVolumes
	accountsDataStor  *accountsDataStorage
	sponsoredAssets   *sponsoredAssets
	scriptsStorage    scriptStorageState
	scriptsComplexity *scriptsComplexity
	invokeResults     *invokeResults
	stateHashes       *stateHashes
	hitSources        *hitSources
	calculateHashes   bool
}

func newBlockchainEntitiesStorage(hs *historyStorage, sets *settings.BlockchainSettings, rw *blockReadWriter, calcHashes bool) (*blockchainEntitiesStorage, error) {
	assets := newAssets(hs.db, hs.dbBatch, hs)
	balances, err := newBalances(hs.db, hs, assets, sets.AddressSchemeCharacter, calcHashes)
	if err != nil {
		return nil, err
	}
	scriptsStorage, err := newScriptsStorage(hs, sets.AddressSchemeCharacter, calcHashes)
	if err != nil {
		return nil, err
	}
	features := newFeatures(rw, hs.db, hs, sets, settings.FeaturesInfo)
	return &blockchainEntitiesStorage{
		hs,
		newAliases(hs.db, hs.dbBatch, hs, calcHashes),
		assets,
		newLeases(hs, calcHashes),
		newScores(hs),
		newBlocksInfo(hs),
		balances,
		features,
		newMonetaryPolicy(hs, sets),
		newOrdersVolumes(hs),
		newAccountsDataStorage(hs.db, hs.dbBatch, hs, calcHashes),
		newSponsoredAssets(rw, features, hs, sets, calcHashes),
		scriptsStorage,
		newScriptsComplexity(hs),
		newInvokeResults(hs),
		newStateHashes(hs),
		newHitSources(hs),
		calcHashes,
	}, nil
}

func (s *blockchainEntitiesStorage) putStateHash(prevHash []byte, height uint64, blockID proto.BlockID) (*proto.StateHash, error) {
	sh := &proto.StateHash{
		BlockID: blockID,
		FieldsHashes: proto.FieldsHashes{
			WavesBalanceHash:  s.balances.wavesHashAt(blockID),
			AssetBalanceHash:  s.balances.assetsHashAt(blockID),
			DataEntryHash:     s.accountsDataStor.hasher.stateHashAt(blockID),
			AccountScriptHash: s.scriptsStorage.getAccountScriptsHasher().stateHashAt(blockID),
			AssetScriptHash:   s.scriptsStorage.getAssetScriptsHasher().stateHashAt(blockID),
			LeaseBalanceHash:  s.balances.leaseHashAt(blockID),
			LeaseStatusHash:   s.leases.hasher.stateHashAt(blockID),
			SponsorshipHash:   s.sponsoredAssets.hasher.stateHashAt(blockID),
			AliasesHash:       s.aliases.hasher.stateHashAt(blockID),
		},
	}
	if err := sh.GenerateSumHash(prevHash); err != nil {
		return nil, err
	}
	if err := s.stateHashes.saveStateHash(sh, height); err != nil {
		return nil, err
	}
	return sh, nil
}

func (s *blockchainEntitiesStorage) prepareHashes() error {
	if err := s.accountsDataStor.prepareHashes(); err != nil {
		return err
	}
	if err := s.balances.prepareHashes(); err != nil {
		return err
	}
	if err := s.scriptsStorage.prepareHashes(); err != nil {
		return err
	}
	if err := s.leases.prepareHashes(); err != nil {
		return err
	}
	if err := s.sponsoredAssets.prepareHashes(); err != nil {
		return err
	}
	if err := s.aliases.prepareHashes(); err != nil {
		return err
	}
	return nil
}

func (s *blockchainEntitiesStorage) handleStateHashes(blockchainHeight uint64, blockIds []proto.BlockID) error {
	if !s.calculateHashes {
		return nil
	}
	if blockchainHeight < 1 {
		return errors.New("bad blockchain height, should be greater than 0")
	}
	// Calculate any remaining hashes.
	if err := s.prepareHashes(); err != nil {
		return err
	}
	prevHash, err := s.stateHashes.stateHash(blockchainHeight)
	if err != nil {
		return err
	}
	startHeight := blockchainHeight + 1
	for i, id := range blockIds {
		height := startHeight + uint64(i)
		newPrevHash, err := s.putStateHash(prevHash.SumHash[:], height, id)
		if err != nil {
			return err
		}
		prevHash = newPrevHash
	}
	return nil
}

func (s *blockchainEntitiesStorage) commitUncertain(blockID proto.BlockID) error {
	if err := s.assets.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.accountsDataStor.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.scriptsStorage.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.leases.commitUncertain(blockID); err != nil {
		return err
	}
	if err := s.sponsoredAssets.commitUncertain(blockID); err != nil {
		return err
	}
	return nil
}

func (s *blockchainEntitiesStorage) dropUncertain() {
	s.assets.dropUncertain()
	s.accountsDataStor.dropUncertain()
	s.scriptsStorage.dropUncertain()
	s.leases.dropUncertain()
	s.sponsoredAssets.dropUncertain()
}

func (s *blockchainEntitiesStorage) reset() {
	s.hs.reset()
	s.assets.reset()
	s.accountsDataStor.reset()
	s.balances.reset()
	s.scriptsStorage.reset()
	s.leases.reset()
	s.sponsoredAssets.reset()
	s.aliases.reset()
}

func (s *blockchainEntitiesStorage) flush() error {
	s.aliases.flush()
	if err := s.hs.flush(); err != nil {
		return err
	}
	if err := s.accountsDataStor.flush(); err != nil {
		return err
	}
	return nil
}

func checkCompatibility(stateDB *stateDB, params StateParams) error {
	version, err := stateDB.stateVersion()
	if err != nil {
		return errors.Errorf("stateVersion: %v", err)
	}
	if version != StateVersion {
		return errors.Errorf("incompatible storage version %d; current state supports only %d", version, StateVersion)
	}
	hasDataForExtendedApi, err := stateDB.stateStoresApiData()
	if err != nil {
		return errors.Errorf("stateStoresApiData(): %v", err)
	}
	if params.StoreExtendedApiData != hasDataForExtendedApi {
		return errors.Errorf("extended API incompatibility: state stores: %v; want: %v", hasDataForExtendedApi, params.StoreExtendedApiData)
	}
	hasDataForHashes, err := stateDB.stateStoresHashes()
	if err != nil {
		return errors.Errorf("stateStoresHashes: %v", err)
	}
	if params.BuildStateHashes != hasDataForHashes {
		return errors.Errorf("state hashes incompatibility: state stores: %v; want: %v", hasDataForHashes, params.BuildStateHashes)
	}
	return nil
}

func handleAmendFlag(stateDB *stateDB, amend bool) (bool, error) {
	storedAmend, err := stateDB.amendFlag()
	if err != nil {
		return false, errors.Wrap(err, "failed to get stored amend flag")
	}
	if !storedAmend && amend { // update if storedAmend == false and amend == true
		if err := stateDB.updateAmendFlag(amend); err != nil {
			return false, errors.Wrap(err, "failed to update amend flag")
		}
		storedAmend = amend
	}
	return storedAmend, nil
}

type newBlocks struct {
	binary    bool
	binBlocks [][]byte
	blocks    []*proto.Block
	curPos    int

	rw       *blockReadWriter
	settings *settings.BlockchainSettings
}

func newNewBlocks(rw *blockReadWriter, settings *settings.BlockchainSettings) *newBlocks {
	return &newBlocks{
		rw:       rw,
		settings: settings,
	}
}

func (n *newBlocks) len() int {
	if n.binary {
		if n.curPos > len(n.binBlocks) {
			return 0
		}
		return len(n.binBlocks) - n.curPos
	}
	if n.curPos > len(n.blocks) {
		return 0
	}
	return len(n.blocks) - n.curPos
}

func (n *newBlocks) setNewBinary(blocks [][]byte) {
	n.reset()
	n.binBlocks = blocks
	n.binary = true
}

func (n *newBlocks) setNew(blocks []*proto.Block) {
	n.reset()
	n.blocks = blocks
	n.binary = false
}

func (n *newBlocks) next() bool {
	n.curPos++
	if n.binary {
		return n.curPos <= len(n.binBlocks)
	} else {
		return n.curPos <= len(n.blocks)
	}
}

func (n *newBlocks) unmarshalBlock(block *proto.Block, blockBytes []byte) error {
	if n.rw.protobufActivated {
		if err := block.UnmarshalFromProtobuf(blockBytes); err != nil {
			return err
		}
	} else {
		if err := block.UnmarshalBinary(blockBytes, n.settings.AddressSchemeCharacter); err != nil {
			return err
		}
	}
	return nil
}

func (n *newBlocks) current() (*proto.Block, error) {
	if !n.binary {
		if n.curPos > len(n.blocks) || n.curPos < 1 {
			return nil, errors.New("bad current position")
		}
		return n.blocks[n.curPos-1], nil
	}
	if n.curPos > len(n.binBlocks) || n.curPos < 1 {
		return nil, errors.New("bad current position")
	}
	blockBytes := n.binBlocks[n.curPos-1]
	b := &proto.Block{}
	if err := n.unmarshalBlock(b, blockBytes); err != nil {
		return nil, err
	}
	return b, nil
}

func (n *newBlocks) reset() {
	n.binBlocks = nil
	n.blocks = nil
	n.curPos = 0
}

type stateManager struct {
	mu *sync.RWMutex

	// Last added block.
	lastBlock atomic.Value

	genesis *proto.Block
	stateDB *stateDB

	stor *blockchainEntitiesStorage
	rw   *blockReadWriter

	// BlockchainSettings: general info about the blockchain type, constants etc.
	settings *settings.BlockchainSettings
	// Validator: validator for block headers.
	cv *consensus.Validator
	// Appender implements validation/diff management functionality.
	appender *txAppender
	atx      *addressTransactions

	// Specifies how many goroutines will be run for verification of transactions and blocks signatures.
	verificationGoroutinesNum int

	newBlocks *newBlocks
}

func newStateManager(dataDir string, amend bool, params StateParams, settings *settings.BlockchainSettings) (*stateManager, error) {
	err := validateSettings(settings)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dataDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(dataDir, 0750); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create state directory: %v", err))
		}
	}
	blockStorageDir := filepath.Join(dataDir, blocksStorDir)
	if _, err := os.Stat(blockStorageDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(blockStorageDir, 0750); err != nil {
			return nil, wrapErr(Other, errors.Errorf("failed to create blocks directory: %v", err))
		}
	}
	// Initialize database.
	dbDir := filepath.Join(dataDir, keyvalueDir)
	zap.S().Info("Initializing state database, will take up to few minutes...")
	params.DbParams.BloomFilterParams.Store.WithPath(filepath.Join(blockStorageDir, "bloom"))
	db, err := keyvalue.NewKeyVal(dbDir, params.DbParams)
	if err != nil {
		return nil, wrapErr(Other, errors.Wrap(err, "failed to create db"))
	}
	zap.S().Info("Finished initializing database")
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, wrapErr(Other, errors.Wrap(err, "failed to create db batch"))
	}
	stateDB, err := newStateDB(db, dbBatch, params)
	if err != nil {
		return nil, wrapErr(Other, errors.Wrap(err, "failed to create stateDB"))
	}
	if err := checkCompatibility(stateDB, params); err != nil {
		return nil, wrapErr(IncompatibilityError, err)
	}
	handledAmend, err := handleAmendFlag(stateDB, amend)
	if err != nil {
		return nil, wrapErr(Other, errors.Wrap(err, "failed to handle amend flag"))
	}
	// rw is storage for blocks.
	rw, err := newBlockReadWriter(
		blockStorageDir,
		params.OffsetLen,
		params.HeaderOffsetLen,
		stateDB,
		settings.AddressSchemeCharacter,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create block storage: %v", err))
	}
	stateDB.setRw(rw)
	hs, err := newHistoryStorage(db, dbBatch, stateDB, handledAmend)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create history storage: %v", err))
	}
	stor, err := newBlockchainEntitiesStorage(hs, settings, rw, params.BuildStateHashes)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create blockchain entities storage: %v", err))
	}
	atxParams := &addressTransactionsParams{
		dir:                 blockStorageDir,
		batchedStorMemLimit: AddressTransactionsMemLimit,
		batchedStorMaxKeys:  AddressTransactionsMaxKeys,
		maxFileSize:         MaxAddressTransactionsFileSize,
		providesData:        params.ProvideExtendedApi,
	}
	atx, err := newAddressTransactions(
		db,
		stateDB,
		rw,
		atxParams,
		handledAmend,
	)
	if err != nil {
		return nil, wrapErr(Other, errors.Errorf("failed to create address transactions storage: %v", err))
	}
	state := &stateManager{
		mu:                        &sync.RWMutex{},
		stateDB:                   stateDB,
		stor:                      stor,
		rw:                        rw,
		settings:                  settings,
		atx:                       atx,
		verificationGoroutinesNum: params.VerificationGoroutinesNum,
		newBlocks:                 newNewBlocks(rw, settings),
	}
	// Set fields which depend on state.
	// Consensus validator is needed to check block headers.
	appender, err := newTxAppender(state, rw, stor, settings, stateDB, atx)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.appender = appender
	state.cv = consensus.NewValidator(state, settings, params.Time)

	height, err := state.Height()
	if err != nil {
		return nil, err
	}
	state.setGenesisBlock(&settings.Genesis)
	// 0 state height means that no blocks are found in state, so blockchain history is empty and we have to add genesis
	if height == 0 {
		// Assign unique block number for this block ID, add this number to the list of valid blocks
		if err := state.stateDB.addBlock(settings.Genesis.BlockID()); err != nil {
			return nil, err
		}
		if err := state.addGenesisBlock(); err != nil {
			return nil, errors.Wrap(err, "failed to apply/save genesis")
		}
		// We apply pre-activated features after genesis block, so they aren't active in genesis itself
		if err := state.applyPreActivatedFeatures(settings.PreactivatedFeatures, settings.Genesis.BlockID()); err != nil {
			return nil, errors.Wrap(err, "failed to apply pre-activated features")
		}
	}

	// check the correct blockchain is being loaded
	genesis, err := state.BlockByHeight(1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis block from state")
	}
	err = settings.Genesis.GenerateBlockID(settings.AddressSchemeCharacter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate genesis block id from config")
	}
	if !bytes.Equal(genesis.ID.Bytes(), settings.Genesis.ID.Bytes()) {
		return nil, errors.Errorf("genesis blocks from state and config mismatch")
	}

	if err := state.loadLastBlock(); err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	h, err := state.Height()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	state.checkProtobufActivation(h + 1)
	return state, nil
}

func (s *stateManager) NewestScriptByAccount(account proto.Recipient) (*ast.Tree, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script by account '%s'", account.String())
	}
	tree, err := s.stor.scriptsStorage.newestScriptByAddr(*addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script by account '%s'", account.String())
	}
	return tree, nil
}

func (s *stateManager) NewestScriptBytesByAccount(account proto.Recipient) (proto.Script, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script bytes by account '%s'", account.String())
	}
	script, err := s.stor.scriptsStorage.newestScriptBytesByAddr(*addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get script bytes by account '%s'", account.String())
	}
	return script, nil
}

func (s *stateManager) NewestScriptByAsset(asset crypto.Digest) (*ast.Tree, error) {
	assetID := proto.AssetIDFromDigest(asset)
	return s.stor.scriptsStorage.newestScriptByAsset(assetID)
}

func (s *stateManager) setGenesisBlock(genesisBlock *proto.Block) {
	s.genesis = genesisBlock
}

func (s *stateManager) TxValidation(func(TxValidation) error) error {
	panic("call TxValidation method on non thread safe state")
}

func (s *stateManager) MapR(func(StateInfo) (interface{}, error)) (interface{}, error) {
	panic("call MapR on non thread safe state")
}

func (s *stateManager) Map(func(State) error) error {
	panic("call Map on non thread safe state")
}

func (s *stateManager) addGenesisBlock() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chans := launchVerifier(ctx, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	if err := s.addNewBlock(s.genesis, nil, chans, 0); err != nil {
		return err
	}
	if err := s.stor.hitSources.appendBlockHitSource(s.genesis, 1, s.genesis.GenSignature); err != nil {
		return err
	}

	if err := s.appender.applyAllDiffs(); err != nil {
		return err
	}
	if err := s.stor.prepareHashes(); err != nil {
		return err
	}
	if _, err := s.stor.putStateHash(nil, 1, s.genesis.BlockID()); err != nil {
		return err
	}
	if verifyError := chans.closeAndWait(); verifyError != nil {
		return wrapErr(ValidationError, verifyError)
	}

	if err := s.flush(); err != nil {
		return wrapErr(ModificationError, err)
	}
	s.reset()
	return nil
}

func (s *stateManager) applyPreActivatedFeatures(features []int16, blockID proto.BlockID) error {
	for _, featureID := range features {
		approvalRequest := &approvedFeaturesRecord{1}
		if err := s.stor.features.approveFeature(featureID, approvalRequest, blockID); err != nil {
			return err
		}
		activationRequest := &activatedFeaturesRecord{1}
		if err := s.stor.features.activateFeature(featureID, activationRequest, blockID); err != nil {
			return err
		}
	}
	if err := s.flush(); err != nil {
		return err
	}
	s.reset()
	return nil
}

func (s *stateManager) checkProtobufActivation(height uint64) {
	activated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockV5), height)
	if activated {
		s.rw.setProtobufActivated()
	}
}

func (s *stateManager) loadLastBlock() error {
	height, err := s.Height()
	if err != nil {
		return errors.Errorf("failed to retrieve height: %v", err)
	}
	lastBlock, err := s.BlockByHeight(height)
	if err != nil {
		return errors.Errorf("failed to get block by height: %v", err)
	}
	s.lastBlock.Store(lastBlock)
	return nil
}

func (s *stateManager) TopBlock() *proto.Block {
	return s.lastBlock.Load().(*proto.Block)
}

func (s *stateManager) BlockVRF(blockHeader *proto.BlockHeader, height proto.Height) ([]byte, error) {
	if blockHeader.Version < proto.ProtobufBlockVersion {
		return nil, nil
	}
	pos := consensus.NewFairPosCalculator(s.settings.DelayDelta, s.settings.MinBlockTime)
	p := pos.HeightForHit(height)
	refHitSource, err := s.NewestHitSourceAtHeight(p)
	if err != nil {
		return nil, err
	}
	gsp := consensus.VRFGenerationSignatureProvider
	ok, vrf, err := gsp.VerifyGenerationSignature(blockHeader.GeneratorPublicKey, refHitSource, blockHeader.GenSignature)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("invalid VRF")
	}
	return vrf, nil
}

func (s *stateManager) Header(blockID proto.BlockID) (*proto.BlockHeader, error) {
	header, err := s.rw.readBlockHeader(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return header, nil
}

func (s *stateManager) NewestHeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	header, err := s.rw.readNewestBlockHeaderByHeight(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return header, nil
}

func (s *stateManager) HeaderByHeight(height uint64) (*proto.BlockHeader, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Header(blockID)
}

func (s *stateManager) Block(blockID proto.BlockID) (*proto.Block, error) {
	block, err := s.rw.readBlock(blockID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return block, nil
}

func (s *stateManager) BlockByHeight(height uint64) (*proto.Block, error) {
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return s.Block(blockID)
}

func (s *stateManager) NewestLeasingInfo(id crypto.Digest) (*proto.LeaseInfo, error) {
	leaseFromStore, err := s.stor.leases.newestLeasingInfo(id)
	if err != nil {
		return nil, err
	}
	leaseInfo := proto.LeaseInfo{
		Sender:      leaseFromStore.Sender,
		Recipient:   leaseFromStore.Recipient,
		IsActive:    leaseFromStore.isActive(),
		LeaseAmount: leaseFromStore.Amount,
	}
	return &leaseInfo, nil
}

func (s *stateManager) NewestScriptPKByAddr(addr proto.WavesAddress) (crypto.PublicKey, error) {
	info, err := s.stor.scriptsStorage.newestScriptBasicInfoByAddressID(addr.ID())
	if err != nil {
		return crypto.PublicKey{}, errors.Wrap(err, "failed to get script public key")
	}
	return info.PK, nil
}

func (s *stateManager) NewestAccountHasScript(addr proto.WavesAddress) (bool, error) {
	return s.stor.scriptsStorage.newestAccountHasScript(addr)
}

func (s *stateManager) AddingBlockHeight() (uint64, error) {
	return s.rw.addingBlockHeight(), nil
}

func (s *stateManager) NewestHeight() (uint64, error) {
	return s.rw.recentHeight(), nil
}

func (s *stateManager) Height() (uint64, error) {
	height, err := s.stateDB.getHeight()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) BlockIDToHeight(blockID proto.BlockID) (uint64, error) {
	height, err := s.rw.heightByBlockID(blockID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) HeightToBlockID(height uint64) (proto.BlockID, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return proto.BlockID{}, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return proto.BlockID{}, wrapErr(InvalidInputError, errors.Errorf("HeightToBlockID: height %d out of valid range [1, %d]", height, maxHeight))
	}
	blockID, err := s.rw.blockIDByHeight(height)
	if err != nil {
		return proto.BlockID{}, wrapErr(RetrievalError, err)
	}
	return blockID, nil
}

func (s *stateManager) newestAssetBalance(addr proto.AddressID, asset proto.AssetID) (uint64, error) {
	// Retrieve old balance from historyStorage.
	balance, err := s.stor.balances.newestAssetBalance(addr, asset)
	if err != nil {
		return 0, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := assetBalanceKey{address: addr, asset: asset}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if err == errNotFound {
		// If there is no diff, old balance is the newest.
		return balance, nil
	} else if err != nil {
		// Something weird happened.
		return 0, err
	}
	balance, err = diff.applyToAssetBalance(balance)
	if err != nil {
		return 0, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return balance, nil
}

func (s *stateManager) newestWavesBalanceProfile(addr proto.AddressID) (*balanceProfile, error) {
	// Retrieve the latest balance from historyStorage.
	profile, err := s.stor.balances.newestWavesBalance(addr)
	if err != nil {
		return nil, err
	}
	// Retrieve the latest balance diff as for the moment of this function call.
	key := wavesBalanceKey{address: addr}
	diff, err := s.appender.diffStorInvoke.latestDiffByKey(string(key.bytes()))
	if err == errNotFound {
		// If there is no diff, old balance is the newest.
		return profile, nil
	} else if err != nil {
		// Something weird happened.
		return nil, err
	}
	newProfile, err := diff.applyTo(profile)
	if err != nil {
		return nil, errors.Errorf("given account has negative balance at this point: %v", err)
	}
	return newProfile, nil
}

func (s *stateManager) GeneratingBalance(account proto.Recipient) (uint64, error) {
	height, err := s.Height()
	if err != nil {
		return 0, errs.Extend(err, "failed to get height")
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	return s.EffectiveBalance(account, start, end)
}

func (s *stateManager) NewestGeneratingBalance(account proto.Recipient) (uint64, error) {
	height, err := s.NewestHeight()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	return s.NewestEffectiveBalance(account, start, end)
}

func (s *stateManager) newestGeneratingBalance(id proto.AddressID) (uint64, error) {
	height, err := s.NewestHeight()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	start, end := s.cv.RangeForGeneratingBalanceByHeight(height)
	effectiveBalance, err := s.stor.balances.newestMinEffectiveBalanceInRange(id, start, end)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return effectiveBalance, nil
}

func (s *stateManager) FullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, errs.Extend(err, "failed convert recipient to address")
	}
	profile, err := s.stor.balances.wavesBalance(addr.ID())
	if err != nil {
		return nil, errs.Extend(err, "failed to get waves balance")
	}
	effective, err := profile.effectiveBalance()
	if err != nil {
		return nil, errs.Extend(err, "failed to get effective balance")
	}
	generating, err := s.GeneratingBalance(account)
	if err != nil {
		return nil, errs.Extend(err, "failed to get generating balance")
	}
	return &proto.FullWavesBalance{
		Regular:    profile.balance,
		Generating: generating,
		Available:  profile.spendableBalance(),
		Effective:  effective,
		LeaseIn:    uint64(profile.leaseIn),
		LeaseOut:   uint64(profile.leaseOut),
	}, nil
}

func (s *stateManager) NewestFullWavesBalance(account proto.Recipient) (*proto.FullWavesBalance, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	profile, err := s.newestWavesBalanceProfile(addr.ID())
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	effective, err := profile.effectiveBalance()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	var generating uint64 = 0
	gb, err := s.NewestGeneratingBalance(account)
	if err == nil {
		generating = gb
		//return nil, wrapErr(RetrievalError, err)
	}
	return &proto.FullWavesBalance{
		Regular:    profile.balance,
		Generating: generating,
		Available:  profile.spendableBalance(),
		Effective:  effective,
		LeaseIn:    uint64(profile.leaseIn),
		LeaseOut:   uint64(profile.leaseOut),
	}, nil
}

func (s *stateManager) WavesBalanceProfile(id proto.AddressID) (*types.WavesBalanceProfile, error) {
	profile, err := s.newestWavesBalanceProfile(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	var generating uint64 = 0
	if gb, err := s.newestGeneratingBalance(id); err == nil {
		generating = gb
	}
	return &types.WavesBalanceProfile{
		Balance:    profile.balance,
		LeaseIn:    profile.leaseIn,
		LeaseOut:   profile.leaseOut,
		Generating: generating,
	}, nil
}

func (s *stateManager) NewestWavesBalance(account proto.Recipient) (uint64, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	profile, err := s.newestWavesBalanceProfile(addr.ID())
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return profile.balance, nil
}

func (s *stateManager) NewestAssetBalance(account proto.Recipient, asset crypto.Digest) (uint64, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	balance, err := s.newestAssetBalance(addr.ID(), proto.AssetIDFromDigest(asset))
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) NewestAssetBalanceByAddressID(id proto.AddressID, asset crypto.Digest) (uint64, error) {
	balance, err := s.newestAssetBalance(id, proto.AssetIDFromDigest(asset))
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) WavesBalance(account proto.Recipient) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	profile, err := s.stor.balances.wavesBalance(addr.ID())
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return profile.balance, nil
}

func (s *stateManager) AssetBalance(account proto.Recipient, assetID proto.AssetID) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	balance, err := s.stor.balances.assetBalance(addr.ID(), assetID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return balance, nil
}

func (s *stateManager) WavesAddressesNumber() (uint64, error) {
	res, err := s.stor.balances.wavesAddressesNumber()
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return res, nil
}

func (s *stateManager) topBlock() (*proto.Block, error) {
	height, err := s.Height()
	if err != nil {
		return nil, err
	}
	// Heights start from 1.
	return s.BlockByHeight(height)
}

func (s *stateManager) addFeaturesVotes(block *proto.Block) error {
	// For Block version 2 Features are always empty, so we don't add anything.
	for _, featureID := range block.Features {
		approved, err := s.stor.features.newestIsApproved(featureID)
		if err != nil {
			return err
		}
		if approved {
			continue
		}
		if err := s.stor.features.addVote(featureID, block.BlockID()); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addRewardVote(block *proto.Block, height uint64) error {
	activation, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
	if err != nil {
		return err
	}
	return s.stor.monetaryPolicy.vote(block.RewardVote, height, activation, block.BlockID())
}

func (s *stateManager) addNewBlock(block, parent *proto.Block, chans *verifierChans, height uint64) error {
	blockHeight := height + 1
	// Add score.
	if err := s.stor.scores.appendBlockScore(block, blockHeight); err != nil {
		return err
	}
	// Indicate new block for storage.
	if err := s.rw.startBlock(block.BlockID()); err != nil {
		return err
	}
	// Save block header to block storage.
	if err := s.rw.writeBlockHeader(&block.BlockHeader); err != nil {
		return err
	}
	transactions := block.Transactions
	if block.TransactionCount != transactions.Count() {
		return errors.Errorf("block.TransactionCount != transactions.Count(), %d != %d", block.TransactionCount, transactions.Count())
	}
	var parentHeader *proto.BlockHeader
	if parent != nil {
		parentHeader = &parent.BlockHeader
	}
	params := &appendBlockParams{
		transactions: transactions,
		chans:        chans,
		block:        &block.BlockHeader,
		parent:       parentHeader,
		height:       height,
	}
	// Check and perform block's transactions, create balance diffs, write transactions to storage.
	if err := s.appender.appendBlock(params); err != nil {
		return err
	}
	// Let block storage know that the current block is over.
	if err := s.rw.finishBlock(block.BlockID()); err != nil {
		return err
	}
	// Count features votes.
	if err := s.addFeaturesVotes(block); err != nil {
		return err
	}
	blockRewardActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.BlockReward), height)
	// Count reward vote.
	if blockRewardActivated {
		err := s.addRewardVote(block, height)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) reset() {
	s.rw.reset()
	s.stor.reset()
	s.stateDB.reset()
	s.appender.reset()
	s.atx.reset()
}

func (s *stateManager) flush() error {
	if err := s.rw.flush(); err != nil {
		return err
	}
	if err := s.stor.flush(); err != nil {
		return err
	}
	if err := s.atx.flush(); err != nil {
		return err
	}
	if err := s.stateDB.flush(); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) AddBlock(block []byte) (*proto.Block, error) {
	s.newBlocks.setNewBinary([][]byte{block})
	rs, err := s.addBlocks()
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddDeserializedBlock(block *proto.Block) (*proto.Block, error) {
	s.newBlocks.setNew([]*proto.Block{block})
	rs, err := s.addBlocks()
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return rs, nil
}

func (s *stateManager) AddBlocks(blockBytes [][]byte) error {
	s.newBlocks.setNewBinary(blockBytes)
	if _, err := s.addBlocks(); err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return err
	}
	return nil
}

func (s *stateManager) AddDeserializedBlocks(blocks []*proto.Block) (*proto.Block, error) {
	s.newBlocks.setNew(blocks)
	lastBlock, err := s.addBlocks()
	if err != nil {
		if err := s.rw.syncWithDb(); err != nil {
			zap.S().Fatalf("Failed to add blocks and can not sync block storage with the database after failure: %v", err)
		}
		return nil, err
	}
	return lastBlock, nil
}

func (s *stateManager) needToFinishVotingPeriod(blockchainHeight uint64) bool {
	nextBlockHeight := blockchainHeight + 1
	votingFinishHeight := (nextBlockHeight % s.settings.ActivationWindowSize(nextBlockHeight)) == 0
	return votingFinishHeight
}

func (s *stateManager) isBlockRewardTermOver(height uint64) (bool, error) {
	feature := int16(settings.BlockReward)
	activated := s.stor.features.newestIsActivatedAtHeight(feature, height)
	if activated {
		activation, err := s.stor.features.newestActivationHeight(int16(settings.BlockReward))
		if err != nil {
			return false, err
		}
		_, end := blockRewardTermBoundaries(height, activation, s.settings.FunctionalitySettings)
		return end == height, nil
	}
	return false, nil
}

func (s *stateManager) needToResetStolenAliases(height uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to reset stolen aliases in custom blockchains.
		return false, nil
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), height)
	if dataTxActivated {
		dataTxHeight, err := s.stor.features.newestActivationHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		return height == dataTxHeight, nil
	}
	return false, nil
}

func (s *stateManager) needToCancelLeases(blockchainHeight uint64) (bool, error) {
	if s.settings.Type == settings.Custom {
		// No need to cancel leases in custom blockchains.
		return false, nil
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), blockchainHeight)
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return false, err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(blockchainHeight)
	}
	rideV5Activated := s.stor.features.newestIsActivatedAtHeight(int16(settings.RideV5), blockchainHeight)
	var rideV5Height uint64 = 0
	if rideV5Activated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.RideV5))
		if err != nil {
			return false, err
		}
		rideV5Height = approvalHeight + s.settings.ActivationWindowSize(blockchainHeight)
	}
	switch blockchainHeight {
	case s.settings.ResetEffectiveBalanceAtHeight:
		return true, nil
	case s.settings.BlockVersion3AfterHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	case dataTxHeight:
		// Only needed for MainNet.
		return s.settings.Type == settings.MainNet, nil
	case rideV5Height:
		// Cancellation of leases to stolen aliases only required for MainNet
		return s.settings.Type == settings.MainNet, nil
	default:
		return false, nil
	}
}

func (s *stateManager) blockchainHeightAction(blockchainHeight uint64, lastBlock, nextBlock proto.BlockID) error {
	cancelLeases, err := s.needToCancelLeases(blockchainHeight)
	if err != nil {
		return err
	}
	if cancelLeases {
		if err := s.cancelLeases(blockchainHeight, lastBlock); err != nil {
			return err
		}
	}
	resetStolenAliases, err := s.needToResetStolenAliases(blockchainHeight)
	if err != nil {
		return err
	}
	if resetStolenAliases {
		if err := s.stor.aliases.disableStolenAliases(); err != nil {
			return err
		}
	}
	if s.needToFinishVotingPeriod(blockchainHeight) {
		if err := s.finishVoting(blockchainHeight, lastBlock); err != nil {
			return err
		}
		if err := s.stor.features.resetVotes(nextBlock); err != nil {
			return err
		}
	}
	termIsOver, err := s.isBlockRewardTermOver(blockchainHeight)
	if err != nil {
		return err
	}
	if termIsOver {
		if err := s.updateBlockReward(blockchainHeight, lastBlock); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) finishVoting(height uint64, blockID proto.BlockID) error {
	nextBlockHeight := height + 1
	if err := s.stor.features.finishVoting(nextBlockHeight, blockID); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) updateBlockReward(height uint64, blockID proto.BlockID) error {
	if err := s.stor.monetaryPolicy.updateBlockReward(height, blockID); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) cancelLeases(height uint64, blockID proto.BlockID) error {
	// Move balance diffs from diffStorage to historyStorage.
	// It must be done before lease cancellation, because
	// lease cancellation iterates through historyStorage.
	if err := s.appender.moveChangesToHistoryStorage(); err != nil {
		return err
	}
	dataTxActivated := s.stor.features.newestIsActivatedAtHeight(int16(settings.DataTransaction), height)
	dataTxHeight := uint64(0)
	if dataTxActivated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.DataTransaction))
		if err != nil {
			return err
		}
		dataTxHeight = approvalHeight + s.settings.ActivationWindowSize(height)
	}
	rideV5Activated := s.stor.features.newestIsActivatedAtHeight(int16(settings.RideV5), height)
	var rideV5Height uint64 = 0
	if rideV5Activated {
		approvalHeight, err := s.stor.features.newestApprovalHeight(int16(settings.RideV5))
		if err != nil {
			return err
		}
		rideV5Height = approvalHeight + s.settings.ActivationWindowSize(height)
	}
	if height == s.settings.ResetEffectiveBalanceAtHeight {
		if err := s.stor.leases.cancelLeases(nil, blockID); err != nil {
			return err
		}
		if err := s.stor.balances.cancelAllLeases(blockID); err != nil {
			return err
		}
	} else if height == s.settings.BlockVersion3AfterHeight {
		overflowAddresses, err := s.stor.balances.cancelLeaseOverflows(blockID)
		if err != nil {
			return err
		}
		if err := s.stor.leases.cancelLeases(overflowAddresses, blockID); err != nil {
			return err
		}
	} else if dataTxActivated && height == dataTxHeight {
		leaseIns, err := s.stor.leases.validLeaseIns()
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelInvalidLeaseIns(leaseIns, blockID); err != nil {
			return err
		}
	} else if rideV5Activated && height == rideV5Height {
		disabledAliases, err := s.stor.aliases.disabledAliases()
		if err != nil {
			return err
		}
		changes, err := s.stor.leases.cancelLeasesToAliases(disabledAliases, blockID)
		if err != nil {
			return err
		}
		if err := s.stor.balances.cancelLeases(changes, blockID); err != nil {
			return err
		}
	}
	return nil
}

func (s *stateManager) addBlocks() (*proto.Block, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		// Reset in-memory storages and load last block in defer.
		s.reset()
		if err := s.loadLastBlock(); err != nil {
			zap.S().Fatalf("Failed to load last block: %v", err)
		}
		s.newBlocks.reset()
	}()

	blocksNumber := s.newBlocks.len()
	if blocksNumber == 0 {
		return nil, wrapErr(InvalidInputError, errors.New("no blocks provided"))
	}

	// Read some useful values for later.
	lastAppliedBlock, err := s.topBlock()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	zap.S().Debugf("StateManager: parent (top) block ID: %s, ts: %d", lastAppliedBlock.BlockID().String(), lastAppliedBlock.Timestamp)
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	headers := make([]proto.BlockHeader, blocksNumber)

	// Launch verifier that checks signatures of blocks and transactions.
	chans := launchVerifier(ctx, s.verificationGoroutinesNum, s.settings.AddressSchemeCharacter)

	var ids []proto.BlockID
	pos := 0
	for s.newBlocks.next() {
		blockchainCurHeight := height + uint64(pos)
		block, err := s.newBlocks.current()
		if err != nil {
			return nil, wrapErr(DeserializationError, err)
		}
		if err := s.cv.ValidateHeaderBeforeBlockApplying(&block.BlockHeader, blockchainCurHeight); err != nil {
			return nil, err
		}
		// Assign unique block number for this block ID, add this number to the list of valid blocks.
		if err := s.stateDB.addBlock(block.BlockID()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// At some blockchain heights specific logic is performed.
		// This includes voting for features, block rewards and so on.
		if err := s.blockchainHeightAction(blockchainCurHeight, lastAppliedBlock.BlockID(), block.BlockID()); err != nil {
			return nil, wrapErr(ModificationError, err)
		}
		// Send block for signature verification, which works in separate goroutine.
		task := &verifyTask{
			taskType: verifyBlock,
			parentID: lastAppliedBlock.BlockID(),
			block:    block,
		}
		if err := chans.trySend(task); err != nil {
			return nil, err
		}
		hs, err := s.cv.GenerateHitSource(blockchainCurHeight, block.BlockHeader)
		if err != nil {
			return nil, err
		}
		if err := s.stor.hitSources.appendBlockHitSource(block, blockchainCurHeight+1, hs); err != nil {
			return nil, err
		}
		// Save block to storage, check its transactions, create and save balance diffs for its transactions.
		if err := s.addNewBlock(block, lastAppliedBlock, chans, blockchainCurHeight); err != nil {
			return nil, err
		}
		if s.needToFinishVotingPeriod(blockchainCurHeight + 1) {
			// If we need to finish voting period on the next block (h+1) then
			// we have to check that protobuf will be activated on next block
			s.checkProtobufActivation(blockchainCurHeight + 2)
		}
		headers[pos] = block.BlockHeader
		pos++
		ids = append(ids, block.BlockID())
		lastAppliedBlock = block
	}
	// Tasks chan can now be closed, since all the blocks and transactions have been already sent for verification.
	// wait for all verifier goroutines
	if verifyError := chans.closeAndWait(); err != nil {
		return nil, wrapErr(ValidationError, verifyError)
	}

	// Apply all the balance diffs accumulated from this blocks batch.
	// This also validates diffs for negative balances.
	if err := s.appender.applyAllDiffs(); err != nil {
		return nil, err
	}
	// Retrieve and store state hashes for each of new blocks.
	if err := s.stor.handleStateHashes(height, ids); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	// Validate consensus (i.e. that all the new blocks were mined fairly).
	if err := s.cv.ValidateHeadersBatch(headers[:pos], height); err != nil {
		return nil, wrapErr(ValidationError, err)
	}
	// After everything is validated, save all the changes to DB.
	if err := s.flush(); err != nil {
		return nil, wrapErr(ModificationError, err)
	}
	zap.S().Infof(
		"Height: %d; Block ID: %s, GenSig: %s, ts: %d",
		height+uint64(blocksNumber),
		lastAppliedBlock.BlockID().String(),
		base58.Encode(lastAppliedBlock.GenSignature),
		lastAppliedBlock.Timestamp,
	)
	return lastAppliedBlock, nil
}

func (s *stateManager) checkRollbackHeight(height uint64) error {
	maxHeight, err := s.Height()
	if err != nil {
		return err
	}
	minRollbackHeight, err := s.stateDB.getRollbackMinHeight()
	if err != nil {
		return err
	}
	if height < minRollbackHeight || height > maxHeight {
		return errors.Errorf("invalid height; valid range is: [%d, %d]", minRollbackHeight, maxHeight)
	}
	return nil
}

func (s *stateManager) checkRollbackInput(blockID proto.BlockID) error {
	height, err := s.BlockIDToHeight(blockID)
	if err != nil {
		return err
	}
	return s.checkRollbackHeight(height)
}

func (s *stateManager) RollbackToHeight(height uint64) error {
	if err := s.checkRollbackHeight(height); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	blockID, err := s.HeightToBlockID(height)
	if err != nil {
		return wrapErr(RetrievalError, err)
	}
	return s.rollbackToImpl(blockID)
}

func (s *stateManager) rollbackToImpl(removalEdge proto.BlockID) error {
	// The database part of rollback.
	if err := s.stateDB.rollback(removalEdge); err != nil {
		return wrapErr(RollbackError, err)
	}
	// After this point Fatalf() is called instead of returning errors,
	// because exiting would lead to incorrect state.
	// Remove blocks from block storage by syncing block storage with the database.
	if err := s.rw.syncWithDb(); err != nil {
		zap.S().Fatalf("Failed to sync block storage with db: %v", err)
	}
	// Clear scripts cache after rollback.
	if err := s.stor.scriptsStorage.clearCache(); err != nil {
		zap.S().Fatalf("Failed to clear scripts cache after rollback: %v", err)
	}
	// Clear features cache
	s.stor.features.clearCache()

	if err := s.stor.flush(); err != nil {
		zap.S().Fatalf("Failed to flush history storage cache after rollback: %v", err)
	}

	if err := s.loadLastBlock(); err != nil {
		zap.S().Fatalf("Failed to load last block after rollback: %v", err)
	}
	return nil
}

func (s *stateManager) RollbackTo(removalEdge proto.BlockID) error {
	if err := s.checkRollbackInput(removalEdge); err != nil {
		return wrapErr(InvalidInputError, err)
	}
	return s.rollbackToImpl(removalEdge)
}

func (s *stateManager) ScoreAtHeight(height uint64) (*big.Int, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("ScoreAtHeight: %d height out of valid range [1, %d]", height, maxHeight))
	}
	score, err := s.stor.scores.score(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return score, nil
}

func (s *stateManager) HitSourceAtHeight(height uint64) ([]byte, error) {
	maxHeight, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("HitSourceAtHeight: height %d out of valid range [1, %d]", height, maxHeight))
	}
	return s.stor.hitSources.hitSource(height)
}

func (s *stateManager) NewestHitSourceAtHeight(height uint64) ([]byte, error) {
	maxHeight, err := s.NewestHeight()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if height < 1 || height > maxHeight {
		return nil, wrapErr(InvalidInputError, errors.Errorf("NewestHitSourceAtHeight: height %d out of valid range [1, %d]", height, maxHeight))
	}
	return s.stor.hitSources.newestHitSource(height)
}

func (s *stateManager) CurrentScore() (*big.Int, error) {
	height, err := s.Height()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	score, err := s.stor.scores.score(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return score, nil
}

func (s *stateManager) NewestRecipientToAddress(recipient proto.Recipient) (*proto.WavesAddress, error) {
	if addr := recipient.Address(); addr != nil {
		return addr, nil
	}
	return s.stor.aliases.newestAddrByAlias(recipient.Alias().Alias)
}

func (s *stateManager) recipientToAddress(recipient proto.Recipient) (*proto.WavesAddress, error) {
	if addr := recipient.Address(); addr != nil {
		return addr, nil
	}
	return s.stor.aliases.addrByAlias(recipient.Alias().Alias)
}

func (s *stateManager) EffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return 0, errs.Extend(err, "failed convert recipient to address ")
	}
	effectiveBalance, err := s.stor.balances.minEffectiveBalanceInRange(addr.ID(), startHeight, endHeight)
	if err != nil {
		return 0, errs.Extend(err, fmt.Sprintf("failed get min effective balance: startHeight: %d, endHeight: %d", startHeight, endHeight))
	}
	return effectiveBalance, nil
}

func (s *stateManager) NewestEffectiveBalance(account proto.Recipient, startHeight, endHeight uint64) (uint64, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	effectiveBalance, err := s.stor.balances.newestMinEffectiveBalanceInRange(addr.ID(), startHeight, endHeight)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return effectiveBalance, nil
}

func (s *stateManager) BlockchainSettings() (*settings.BlockchainSettings, error) {
	cp := *s.settings
	return &cp, nil
}

func (s *stateManager) ResetValidationList() {
	s.reset()
	if err := s.stor.scriptsStorage.clearCache(); err != nil {
		zap.S().Fatalf("Failed to clearCache scripts cache after UTX validation: %v", err)
	}
}

// ValidateNextTx function must be used for UTX validation only.
func (s *stateManager) ValidateNextTx(tx proto.Transaction, currentTimestamp, parentTimestamp uint64, v proto.BlockVersion, acceptFailed bool) error {
	if err := s.appender.validateNextTx(tx, currentTimestamp, parentTimestamp, v, acceptFailed); err != nil {
		return err
	}
	return nil
}

func (s *stateManager) NewestAddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	addr, err := s.stor.aliases.newestAddrByAlias(alias.Alias)
	if err != nil {
		return proto.WavesAddress{}, wrapErr(RetrievalError, err)
	}
	return *addr, nil
}

func (s *stateManager) AddrByAlias(alias proto.Alias) (proto.WavesAddress, error) {
	addr, err := s.stor.aliases.addrByAlias(alias.Alias)
	if err != nil {
		return proto.WavesAddress{}, wrapErr(RetrievalError, err)
	}
	return *addr, nil
}

func (s *stateManager) VotesNumAtHeight(featureID int16, height proto.Height) (uint64, error) {
	votesNum, err := s.stor.features.featureVotesAtHeight(featureID, height)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return votesNum, nil
}

func (s *stateManager) VotesNum(featureID int16) (uint64, error) {
	votesNum, err := s.stor.features.featureVotes(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return votesNum, nil
}

func (s *stateManager) IsActivated(featureID int16) (bool, error) {
	activated, err := s.stor.features.newestIsActivated(featureID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return activated, nil
}

func (s *stateManager) NewestIsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	return s.stor.features.newestIsActivatedAtHeight(featureID, height), nil
}

func (s *stateManager) IsActiveAtHeight(featureID int16, height proto.Height) (bool, error) {
	return s.stor.features.isActivatedAtHeight(featureID, height), nil
}

func (s *stateManager) ActivationHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.activationHeight(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) IsApproved(featureID int16) (bool, error) {
	approved, err := s.stor.features.isApproved(featureID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return approved, nil
}

func (s *stateManager) IsApprovedAtHeight(featureID int16, height uint64) (bool, error) {
	return s.stor.features.isApprovedAtHeight(featureID, height), nil
}

func (s *stateManager) ApprovalHeight(featureID int16) (uint64, error) {
	height, err := s.stor.features.approvalHeight(featureID)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return height, nil
}

func (s *stateManager) AllFeatures() ([]int16, error) {
	features, err := s.stor.features.allFeatures()
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return features, nil
}

func (s *stateManager) EstimatorVersion() (int, error) {
	rideV6, err := s.IsActivated(int16(settings.RideV6))
	if err != nil {
		return 0, err
	}
	if rideV6 {
		return 4, nil
	}

	blockV5, err := s.IsActivated(int16(settings.BlockV5))
	if err != nil {
		return 0, err
	}
	if blockV5 {
		return 3, nil
	}

	blockReward, err := s.IsActivated(int16(settings.BlockReward))
	if err != nil {
		return 0, err
	}
	if blockReward {
		return 2, nil
	}

	smartAccounts, err := s.IsActivated(int16(settings.SmartAccounts))
	if err != nil {
		return 0, err
	}
	if smartAccounts {
		return 1, nil
	}
	return 0, errors.New("inactive RIDE")
}

// Accounts data storage.

func (s *stateManager) RetrieveNewestEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveEntries(account proto.Recipient) ([]proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entries, err := s.stor.accountsDataStor.retrieveEntries(addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entries, nil
}

func (s *stateManager) IsStateUntouched(account proto.Recipient) (bool, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	entryExist, err := s.stor.accountsDataStor.newestEntryExists(addr)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return !entryExist, nil
}

func (s *stateManager) RetrieveEntry(account proto.Recipient, key string) (proto.DataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestIntegerEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveIntegerEntry(account proto.Recipient, key string) (*proto.IntegerDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveIntegerEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBooleanEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveBooleanEntry(account proto.Recipient, key string) (*proto.BooleanDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveBooleanEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestStringEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveStringEntry(account proto.Recipient, key string) (*proto.StringDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveStringEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveNewestBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	addr, err := s.NewestRecipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveNewestBinaryEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) RetrieveBinaryEntry(account proto.Recipient, key string) (*proto.BinaryDataEntry, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	entry, err := s.stor.accountsDataStor.retrieveBinaryEntry(addr, key)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return entry, nil
}

func (s *stateManager) NewestTransactionByID(id []byte) (proto.Transaction, error) {
	tx, _, err := s.rw.readNewestTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return tx, nil
}

func (s *stateManager) TransactionByID(id []byte) (proto.Transaction, error) {
	tx, _, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return tx, nil
}

func (s *stateManager) TransactionByIDWithStatus(id []byte) (proto.Transaction, bool, error) {
	tx, failed, err := s.rw.readTransaction(id)
	if err != nil {
		return nil, false, wrapErr(RetrievalError, err)
	}
	return tx, failed, nil
}

func (s *stateManager) NewestTransactionHeightByID(id []byte) (uint64, error) {
	txHeight, err := s.rw.newestTransactionHeightByID(id)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return txHeight, nil
}

func (s *stateManager) TransactionHeightByID(id []byte) (uint64, error) {
	txHeight, err := s.rw.transactionHeightByID(id)
	if err != nil {
		return 0, wrapErr(RetrievalError, err)
	}
	return txHeight, nil
}

func (s *stateManager) NewAddrTransactionsIterator(addr proto.Address) (TransactionIterator, error) {
	providesData, err := s.ProvidesExtendedApi()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !providesData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for transactions by address API"))
	}
	iter, err := s.atx.newTransactionsByAddrIterator(addr)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return iter, nil
}

func (s *stateManager) NewestAssetIsSponsored(asset crypto.Digest) (bool, error) {
	assetID := proto.AssetIDFromDigest(asset)
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) AssetIsSponsored(assetID proto.AssetID) (bool, error) {
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return sponsored, nil
}

func (s *stateManager) NewestAssetInfo(asset crypto.Digest) (*proto.AssetInfo, error) {
	assetID := proto.AssetIDFromDigest(asset)
	info, err := s.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.AssetInfo{
		ID:              proto.ReconstructDigest(assetID, info.tail),
		Quantity:        info.quantity.Uint64(),
		Decimals:        byte(info.decimals),
		Issuer:          issuer,
		IssuerPublicKey: info.issuer,
		Reissuable:      info.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

// NewestFullAssetInfo is used to request full asset info from RIDE,
// because of that we don't try to get issue transaction info.
func (s *stateManager) NewestFullAssetInfo(asset crypto.Digest) (*proto.FullAssetInfo, error) {
	ai, err := s.NewestAssetInfo(asset)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	assetID := proto.AssetIDFromDigest(asset)
	info, err := s.stor.assets.newestAssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: nil, // Always return nil in this function because this field is not used later on
	}
	isSponsored, err := s.stor.sponsoredAssets.newestIsSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.newestAssetCost(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.NewestWavesBalance(proto.NewRecipientFromAddress(ai.Issuer))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted, err := s.stor.scriptsStorage.newestIsSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if isScripted {
		scriptInfo, err := s.NewestScriptInfoByAsset(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.ScriptInfo = *scriptInfo
	}
	return res, nil
}

// AssetInfo returns stable (stored in DB) information about an asset by given ID.
// If there is no asset for the given ID error of type `errs.UnknownAsset` is returned.
// Errors of types `state.RetrievalError` returned in case of broken DB.
func (s *stateManager) AssetInfo(assetID proto.AssetID) (*proto.AssetInfo, error) {
	info, err := s.stor.assets.assetInfo(assetID)
	if err != nil {
		if errors.Is(err, errs.UnknownAsset{}) {
			return nil, err
		}
		return nil, wrapErr(RetrievalError, err)
	}
	if !info.quantity.IsUint64() {
		return nil, wrapErr(Other, errors.New("asset quantity overflows uint64"))
	}
	issuer, err := proto.NewAddressFromPublicKey(s.settings.AddressSchemeCharacter, info.issuer)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	sponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scripted, err := s.stor.scriptsStorage.isSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return &proto.AssetInfo{
		ID:              proto.ReconstructDigest(assetID, info.tail),
		Quantity:        info.quantity.Uint64(),
		Decimals:        byte(info.decimals),
		Issuer:          issuer,
		IssuerPublicKey: info.issuer,
		Reissuable:      info.reissuable,
		Scripted:        scripted,
		Sponsored:       sponsored,
	}, nil
}

func (s *stateManager) FullAssetInfo(assetID proto.AssetID) (*proto.FullAssetInfo, error) {
	ai, err := s.AssetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	info, err := s.stor.assets.assetInfo(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	tx, _ := s.TransactionByID(assetID.Bytes()) // Explicitly ignore error here, in case of error tx is nil as expected
	res := &proto.FullAssetInfo{
		AssetInfo:        *ai,
		Name:             info.name,
		Description:      info.description,
		IssueTransaction: tx,
	}

	isSponsored, err := s.stor.sponsoredAssets.isSponsored(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isSponsored {
		assetCost, err := s.stor.sponsoredAssets.assetCost(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		sponsorBalance, err := s.WavesBalance(proto.NewRecipientFromAddress(ai.Issuer))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.SponsorshipCost = assetCost
		res.SponsorBalance = sponsorBalance
	}
	isScripted, err := s.stor.scriptsStorage.isSmartAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	if isScripted {
		scriptInfo, err := s.ScriptInfoByAsset(assetID)
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		res.ScriptInfo = *scriptInfo
	}
	return res, nil
}

func (s *stateManager) NFTList(account proto.Recipient, limit uint64, afterAssetID *proto.AssetID) ([]*proto.FullAssetInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	nfts, err := s.stor.balances.nftList(addr.ID(), limit, afterAssetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	infos := make([]*proto.FullAssetInfo, len(nfts))
	for i, nft := range nfts {
		info, err := s.FullAssetInfo(proto.AssetIDFromDigest(nft))
		if err != nil {
			return nil, wrapErr(RetrievalError, err)
		}
		infos[i] = info
	}
	return infos, nil
}

func (s *stateManager) ScriptBasicInfoByAccount(account proto.Recipient) (*proto.ScriptBasicInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	hasScript, err := s.stor.scriptsStorage.accountHasScript(*addr)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasScript {
		return nil, proto.ErrNotFound
	}
	info, err := s.stor.scriptsStorage.scriptBasicInfoByAddressID(addr.ID())
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptBasicInfo{
		PK:             info.PK,
		ScriptLen:      info.ScriptLen,
		LibraryVersion: info.LibraryVersion,
		HasVerifier:    info.HasVerifier,
		IsDApp:         info.IsDApp,
	}, nil
}

func (s *stateManager) ScriptInfoByAccount(account proto.Recipient) (*proto.ScriptInfo, error) {
	addr, err := s.recipientToAddress(account)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAddr(*addr)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	ev, err := s.EstimatorVersion()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	est, err := s.stor.scriptsComplexity.scriptComplexityByAddress(*addr, ev)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Base64:     text,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) ScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.scriptBytesByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	est, err := s.stor.scriptsComplexity.scriptComplexityByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Base64:     text,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) NewestScriptInfoByAsset(assetID proto.AssetID) (*proto.ScriptInfo, error) {
	scriptBytes, err := s.stor.scriptsStorage.newestScriptBytesByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	text := base64.StdEncoding.EncodeToString(scriptBytes)
	est, err := s.stor.scriptsComplexity.newestScriptComplexityByAsset(assetID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	version, err := proto.VersionFromScriptBytes(scriptBytes)
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	return &proto.ScriptInfo{
		Version:    version,
		Bytes:      scriptBytes,
		Base64:     text,
		Complexity: uint64(est.Estimation),
	}, nil
}

func (s *stateManager) IsActiveLeasing(leaseID crypto.Digest) (bool, error) {
	isActive, err := s.stor.leases.isActive(leaseID)
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return isActive, nil
}

func (s *stateManager) InvokeResultByID(invokeID crypto.Digest) (*proto.ScriptResult, error) {
	hasData, err := s.storesExtendedApiData()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for invoke results"))
	}
	res, err := s.stor.invokeResults.invokeResult(s.settings.AddressSchemeCharacter, invokeID)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return res, nil
}

func (s *stateManager) storesExtendedApiData() (bool, error) {
	stores, err := s.stateDB.stateStoresApiData()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return stores, nil
}

func (s *stateManager) ProvidesExtendedApi() (bool, error) {
	hasData, err := s.storesExtendedApiData()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	if !hasData {
		// State does not have extended API data.
		return false, nil
	}
	// State has data for extended API, but we need to make sure it is served.
	return s.atx.providesData(), nil
}

func (s *stateManager) ProvidesStateHashes() (bool, error) {
	provides, err := s.stateDB.stateStoresHashes()
	if err != nil {
		return false, wrapErr(RetrievalError, err)
	}
	return provides, nil
}

func (s *stateManager) StateHashAtHeight(height uint64) (*proto.StateHash, error) {
	hasData, err := s.ProvidesStateHashes()
	if err != nil {
		return nil, wrapErr(Other, err)
	}
	if !hasData {
		return nil, wrapErr(IncompatibilityError, errors.New("state does not have data for state hashes"))
	}
	sh, err := s.stor.stateHashes.stateHash(height)
	if err != nil {
		return nil, wrapErr(RetrievalError, err)
	}
	return sh, nil
}

func (s *stateManager) IsNotFound(err error) bool {
	return IsNotFound(err)
}

func (s *stateManager) StartProvidingExtendedApi() error {
	if err := s.atx.startProvidingData(); err != nil {
		return wrapErr(ModificationError, err)
	}
	return nil
}

func (s *stateManager) PersistAddressTransactions() error {
	return s.atx.persist()
}

func (s *stateManager) ShouldPersistAddressTransactions() (bool, error) {
	return s.atx.shouldPersist()
}

func (s *stateManager) Close() error {
	if err := s.atx.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	if err := s.rw.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	if err := s.stateDB.close(); err != nil {
		return wrapErr(ClosureError, err)
	}
	return nil
}

func (s *stateManager) NewestScriptVersionByAddressID(id proto.AddressID) (ast.LibraryVersion, error) {
	info, err := s.stor.scriptsStorage.newestScriptBasicInfoByAddressID(id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get script version")
	}
	return info.LibraryVersion, nil
}
