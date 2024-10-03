package itests

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type SimpleChallengingSuite struct {
	fixtures.SingleGoNodeSuite
}

func (s *SimpleChallengingSuite) SetupSuite() {
	s.BaseSetup(
		config.WithNoGoMining(),
		config.WithPreactivatedFeatures([]config.FeatureInfo{{Feature: int16(settings.LightNode), Height: 1}}),
		config.WithAbsencePeriod(1),
	)
}

func (s *SimpleChallengingSuite) TestSimpleChallenging() {
	conn, err := net.NewConnection(
		proto.TCPAddr{},
		config.DefaultIP+":"+s.Docker.GoNode().Ports().BindPort,
		proto.ProtocolVersion(),
		"wavesL",
	)
	require.NoError(s.T(), err, "failed to create connection to go node")
	defer func(conn *net.OutgoingPeer) {
		if clErr := conn.Close(); clErr != nil {
			s.T().Logf("Failed to close connection: %v", clErr)
		}
	}(conn)

	acc := s.Cfg.GetRichestAccount()

	// Calculate state hash for the key-block. Take into account only miner's reward.
	genesisSH := s.Cfg.GenesisSH()
	s.T().Logf("Genesis snapshot hash: %s", genesisSH.String())

	newBalance := acc.Amount + s.Cfg.BlockchainSettings.InitialBlockReward
	sh, err := KeyBlockSH(genesisSH, acc.Address, newBalance)
	require.NoError(s.T(), err, "failed to calculate state hash")
	s.T().Logf("Key-block snapshot hash: %s", sh.String())

	// Generate key-block
	hs := s.Cfg.BlockchainSettings.Genesis.BlockHeader.GenSignature
	gsp := consensus.VRFGenerationSignatureProvider
	pos := consensus.NewFairPosCalculator(s.Cfg.BlockchainSettings.DelayDelta, s.Cfg.BlockchainSettings.MinBlockTime)
	gs, err := gsp.GenerationSignature(acc.SecretKey, hs)
	require.NoError(s.T(), err, "failed to generate generation signature")

	source, err := gsp.HitSource(acc.SecretKey, hs)
	require.NoError(s.T(), err, "failed to generate hit source")

	hit, err := consensus.GenHit(source)
	require.NoError(s.T(), err, "failed to generate hit from source")

	delay, err := pos.CalculateDelay(hit, s.Cfg.BlockchainSettings.Genesis.BaseTarget, acc.Amount)
	require.NoError(s.T(), err, "failed to calculate delay")

	ts := s.Cfg.BlockchainSettings.Genesis.Timestamp + delay
	bt, err := pos.CalculateBaseTarget(
		s.Cfg.BlockchainSettings.AverageBlockDelaySeconds,
		1,
		s.Cfg.BlockchainSettings.Genesis.BaseTarget,
		s.Cfg.BlockchainSettings.Genesis.Timestamp,
		0, // Zero for heights less than 2.
		ts,
	)

	nxt := proto.NxtConsensus{BaseTarget: bt, GenSignature: gs}

	err = s.Cfg.BlockchainSettings.Genesis.GenerateBlockID(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to generate genesis block ID")

	genesisID := s.Cfg.BlockchainSettings.Genesis.BlockID()
	s.T().Logf("Genesis block ID: %s", genesisID.String())
	bl, err := proto.CreateBlock(proto.Transactions(nil), ts, genesisID, acc.PublicKey,
		nxt, proto.ProtobufBlockVersion, nil, int64(s.Cfg.BlockchainSettings.InitialBlockReward),
		s.Cfg.BlockchainSettings.AddressSchemeCharacter, &sh)
	require.NoError(s.T(), err, "failed to create block")

	// Sign the block and generate its ID.
	err = bl.Sign(s.Cfg.BlockchainSettings.AddressSchemeCharacter, acc.SecretKey)
	require.NoError(s.T(), err, "failed to sing the block")

	err = bl.GenerateBlockID(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to generate block ID")

	// Calculate score.
	genesisScore := calculateScore(s.Cfg.BlockchainSettings.Genesis.BaseTarget)
	blockScore := calculateCumulativeScore(genesisScore, bl.BaseTarget)
	scoreMsg := &proto.ScoreMessage{Score: blockScore.Bytes()}
	err = conn.SendMessage(scoreMsg)
	require.NoError(s.T(), err, "failed to send score to node")
	time.Sleep(100 * time.Millisecond)

	// Send block IDs to the node.
	blocksMsg := &proto.BlockIdsMessage{Blocks: []proto.BlockID{bl.BlockID()}}
	err = conn.SendMessage(blocksMsg)
	require.NoError(s.T(), err, "failed to send block IDs to node")
	time.Sleep(100 * time.Millisecond)

	// Marshal the block and send it to the node.
	bb, err := bl.MarshalToProtobuf(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to marshal block")
	blMsg := &proto.PBBlockMessage{PBBlockBytes: bb}
	err = conn.SendMessage(blMsg)
	require.NoError(s.T(), err, "failed to send block to node")
	time.Sleep(10 * time.Second)
}

func TestSimpleChallengingSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SimpleChallengingSuite))
}

func KeyBlockSH(prevSH crypto.Digest, miner proto.WavesAddress, balance uint64) (crypto.Digest, error) {
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

const decimalBase = 10

func calculateScore(baseTarget uint64) *big.Int {
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
