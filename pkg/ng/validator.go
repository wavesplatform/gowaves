package ng

// work in progress
/*
import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)



type validator struct {
}

func (a validator) validateMicro(row Row) error {
	keyBlock := row.KeyBlock.Clone()

	ok, err := keyBlock.VerifySignature()
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("signature validation failed")
	}

	for _, m := range row.MicroBlocks {
		joined, err := keyBlock.Transactions.Join(m.Transactions)
		if err != nil {
			return err
		}
		newBlock, err := proto.CreateBlock(
			joined,
			keyBlock.Timestamp,
			keyBlock.Parent,
			keyBlock.GenPublicKey,
			keyBlock.NxtConsensus,
			keyBlock.Version,
			keyBlock.Features,
			keyBlock.RewardVote,
		)
		if err != nil {
			return err
		}

		newBlock.BlockSignature = m.TotalResBlockSigField
		ok, err = newBlock.VerifySignature()
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("signature validation failed")
		}
		keyBlock = newBlock
	}

}
*/
