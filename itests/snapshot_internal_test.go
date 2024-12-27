package itests

import (
	"encoding/binary"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type SimpleSnapshotSuite struct {
	fixtures.SingleGoNodeSuite
}

func (s *SimpleSnapshotSuite) SetupSuite() {
	s.BaseSetup(
		config.WithNoGoMining(),
		config.WithPreactivatedFeatures([]config.FeatureInfo{{Feature: int16(settings.LightNode), Height: 1}}),
		config.WithAbsencePeriod(1),
	)
}

func (s *SimpleSnapshotSuite) TestSimpleSnapshot() {
	const messageTimeout = 5 * time.Second

	acc := s.Cfg.GetRichestAccount()

	// Initialize genesis block ID.
	err := s.Cfg.BlockchainSettings.Genesis.GenerateBlockID(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to generate genesis block ID")
	genesisID := s.Cfg.BlockchainSettings.Genesis.BlockID()

	// Calculate state hash for the key-block. Take into account only miner's reward.
	genesisSH := s.Cfg.GenesisSH()

	// Calculate new balance of the richest account (block generator).
	newBalance := acc.Amount + s.Cfg.BlockchainSettings.InitialBlockReward

	// Calculate state hash for the key-block.
	sh, err := keyBlockSH(genesisSH, acc.Address, newBalance)
	require.NoError(s.T(), err, "failed to calculate state hash")

	// Generate key-block
	bl, delay := createKeyBlock(s.T(), s.Cfg.BlockchainSettings.Genesis.GenSignature, s.Cfg.BlockchainSettings,
		acc.SecretKey, acc.PublicKey, newBalance, genesisID, s.Cfg.BlockchainSettings.Genesis.Timestamp,
		s.Cfg.BlockchainSettings.Genesis.BaseTarget, sh)
	if delay > 0 {
		time.Sleep(delay)
	}

	err = s.Client.Connection.SubscribeForMessages(
		reflect.TypeOf(&proto.GetBlockIdsMessage{}),
		reflect.TypeOf(&proto.GetBlockMessage{}),
		reflect.TypeOf(&proto.ScoreMessage{}),
		reflect.TypeOf(&proto.MicroBlockRequestMessage{}),
	)
	require.NoError(s.T(), err, "failed to subscribe for messages")

	// Calculate new score and send score to the node.
	genesisScore := calculateScore(s.Cfg.BlockchainSettings.Genesis.BaseTarget)
	blockScore := calculateCumulativeScore(genesisScore, bl.BaseTarget)
	scoreMsg := &proto.ScoreMessage{Score: blockScore.Bytes()}
	s.Client.Connection.SendMessage(scoreMsg)

	// Wait for the node to request block IDs.
	_, err = s.Client.Connection.AwaitMessage(reflect.TypeOf(&proto.GetBlockIdsMessage{}), messageTimeout)
	require.NoError(s.T(), err, "failed to wait for block IDs request")

	// Send block IDs to the node.
	blocksMsg := &proto.BlockIdsMessage{Blocks: []proto.BlockID{bl.BlockID()}}
	s.Client.Connection.SendMessage(blocksMsg)

	// Wait for the node to request the block.
	blockID, err := s.Client.Connection.AwaitGetBlockMessage(messageTimeout)
	require.NoError(s.T(), err, "failed to wait for block request")
	assert.Equal(s.T(), bl.BlockID(), blockID)

	// Marshal the block and send it to the node.
	bb, err := bl.MarshalToProtobuf(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to marshal block")
	blMsg := &proto.PBBlockMessage{PBBlockBytes: bb}
	s.Client.Connection.SendMessage(blMsg)

	// Wait for updated score message.
	score, err := s.Client.Connection.AwaitScoreMessage(messageTimeout)
	require.NoError(s.T(), err, "failed to wait for score")
	assert.Equal(s.T(), blockScore, score)

	// Wait for 2.5 seconds and send micro-block (imitate real life).
	time.Sleep(2500 * time.Millisecond)

	// Add transactions to block.
	tx := proto.NewUnsignedTransferWithProofs(3, acc.PublicKey,
		proto.NewOptionalAssetWaves(), proto.NewOptionalAssetWaves(), uint64(time.Now().UnixMilli()), 1_0000_0000,
		100_000, proto.NewRecipientFromAddress(acc.Address), nil)
	err = tx.Sign(s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc.SecretKey)
	require.NoError(s.T(), err, "failed to sign tx")

	// Create micro-block with the transaction and unchanged state hash.
	mb, inv := createMicroBlockAndInv(s.T(), *bl, s.Cfg.BlockchainSettings, tx, acc.SecretKey, acc.PublicKey, sh)

	// Send micro-block inv to the node.
	ib, err := inv.MarshalBinary()
	require.NoError(s.T(), err, "failed to marshal inv")
	invMsg := &proto.MicroBlockInvMessage{Body: ib}
	s.Client.Connection.SendMessage(invMsg)

	// Wait for the node to request micro-block.
	mbID, err := s.Client.Connection.AwaitMicroblockRequest(messageTimeout)
	require.NoError(s.T(), err, "failed to wait for micro-block request")
	assert.Equal(s.T(), inv.TotalBlockID, mbID)

	// Marshal the micro-block and send it to the node.
	mbb, err := mb.MarshalToProtobuf(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to marshal micro block")
	mbMsg := &proto.PBMicroBlockMessage{MicroBlockBytes: mbb}
	s.Client.Connection.SendMessage(mbMsg)

	h := s.Client.HTTPClient.GetHeight(s.T())
	header := s.Client.HTTPClient.BlockHeader(s.T(), h.Height)
	assert.Equal(s.T(), bl.BlockID().String(), header.ID.String())
}

func TestSimpleSnapshotSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SimpleSnapshotSuite))
}

func keyBlockSH(prevSH crypto.Digest, miner proto.WavesAddress, balance uint64) (crypto.Digest, error) {
	hash, err := crypto.NewFastHash()
	if err != nil {
		return crypto.Digest{}, errors.Wrap(err, "failed to calculate key block snapshot hash")
	}

	buf := make([]byte, proto.WavesAddressSize+8)
	copy(buf, miner[:])
	binary.BigEndian.PutUint64(buf[proto.WavesAddressSize:], balance)
	hash.Write(buf)

	var txSHD crypto.Digest
	hash.Sum(txSHD[:0])

	hash.Reset()
	hash.Write(prevSH.Bytes())
	hash.Write(txSHD.Bytes())

	var r crypto.Digest
	hash.Sum(r[:0])
	return r, nil
}

func createKeyBlock(t *testing.T, hitSource []byte, cfg *settings.BlockchainSettings,
	generatorSK crypto.SecretKey, generatorPK crypto.PublicKey, generatorBalance uint64,
	parentID proto.BlockID, parentTimestamp uint64, parentBaseTarget uint64,
	sh crypto.Digest,
) (*proto.Block, time.Duration) {
	gsp := consensus.VRFGenerationSignatureProvider
	pos := consensus.NewFairPosCalculator(cfg.DelayDelta, cfg.MinBlockTime)
	gs, err := gsp.GenerationSignature(generatorSK, hitSource)
	require.NoError(t, err, "failed to generate generation signature")

	source, err := gsp.HitSource(generatorSK, hitSource)
	require.NoError(t, err, "failed to generate hit source")

	hit, err := consensus.GenHit(source)
	require.NoError(t, err, "failed to generate hit from source")

	delay, err := pos.CalculateDelay(hit, parentBaseTarget, generatorBalance)
	require.NoError(t, err, "failed to calculate delay")

	ts := parentTimestamp + delay
	bt, err := pos.CalculateBaseTarget(cfg.AverageBlockDelaySeconds, 1, parentBaseTarget, parentTimestamp, 0, ts)
	require.NoError(t, err, "failed to calculate base target")

	nxt := proto.NxtConsensus{BaseTarget: bt, GenSignature: gs}

	bl, err := proto.CreateBlock(proto.Transactions(nil), ts, parentID, generatorPK, nxt, proto.ProtobufBlockVersion,
		nil, int64(cfg.InitialBlockReward), cfg.AddressSchemeCharacter, &sh)
	require.NoError(t, err, "failed to create block")

	// Sign the block and generate its ID.
	err = bl.Sign(cfg.AddressSchemeCharacter, generatorSK)
	require.NoError(t, err, "failed to sing the block")

	err = bl.GenerateBlockID(cfg.AddressSchemeCharacter)
	require.NoError(t, err, "failed to generate block ID")

	return bl, time.Until(time.UnixMilli(int64(ts)))
}

func createMicroBlockAndInv(t *testing.T, b proto.Block, cfg *settings.BlockchainSettings, tx proto.Transaction,
	generatorSK crypto.SecretKey, generatorPK crypto.PublicKey, sh crypto.Digest,
) (*proto.MicroBlock, *proto.MicroBlockInv) {
	b.Transactions = []proto.Transaction{tx}
	b.TransactionCount = len(b.Transactions)
	err := b.SetTransactionsRootIfPossible(cfg.AddressSchemeCharacter)
	require.NoError(t, err, "failed to set transactions root")
	err = b.Sign(cfg.AddressSchemeCharacter, generatorSK)
	require.NoError(t, err, "failed to sign block")
	err = b.GenerateBlockID(cfg.AddressSchemeCharacter)
	require.NoError(t, err, "failed to generate block ID")

	mb := &proto.MicroBlock{
		VersionField:          byte(b.Version),
		SenderPK:              generatorPK,
		Transactions:          b.Transactions,
		TransactionCount:      uint32(b.TransactionCount),
		Reference:             b.ID,
		TotalResBlockSigField: b.BlockSignature,
		TotalBlockID:          b.BlockID(),
		StateHash:             &sh,
	}

	err = mb.Sign(cfg.AddressSchemeCharacter, generatorSK)
	require.NoError(t, err, "failed to sign mb block")

	inv := proto.NewUnsignedMicroblockInv(generatorPK, mb.TotalBlockID, mb.Reference)
	err = inv.Sign(generatorSK, cfg.AddressSchemeCharacter)
	require.NoError(t, err, "failed to sign inv")

	return mb, inv
}

func calculateScore(baseTarget uint64) *big.Int {
	const decimalBase = 10

	res := big.NewInt(0)
	if baseTarget == 0 {
		return res
	}
	if baseTarget > math.MaxInt64 {
		panic("base target is too big")
	}
	bt := big.NewInt(int64(baseTarget))
	maxBlockScore, ok := big.NewInt(0).SetString("18446744073709551616", decimalBase)
	if !ok {
		return res
	}
	res.Div(maxBlockScore, bt)
	return res
}

func calculateCumulativeScore(parentScore *big.Int, baseTarget uint64) *big.Int {
	s := calculateScore(baseTarget)
	if parentScore == nil {
		return s
	}
	return s.Add(s, parentScore)
}
