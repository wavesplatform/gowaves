package signatures

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
)

// from small to big by height
type NaturalOrdering = Signatures
type ReverseOrdering = Signatures

type Signatures struct {
	signatures []crypto.Signature
	unique     map[crypto.Signature]struct{}
}

func (a *Signatures) Signatures() []crypto.Signature {
	return a.signatures
}

func NewSignatures(signatures ...crypto.Signature) *Signatures {
	unique := make(map[crypto.Signature]struct{})
	for _, v := range signatures {
		unique[v] = struct{}{}
	}

	return &Signatures{
		signatures: signatures,
		unique:     unique,
	}
}

func (a *Signatures) Exists(sig crypto.Signature) bool {
	_, ok := a.unique[sig]
	return ok
}

func (a *Signatures) Revert() *Signatures {
	out := make([]crypto.Signature, len(a.signatures))
	for k, v := range a.signatures {
		out[len(a.signatures)-1-k] = v
	}
	return NewSignatures(out...)
}

func (a *Signatures) Len() int {
	return len(a.signatures)
}

type LastSignatures interface {
	LastSignatures(state storage.State) (*ReverseOrdering, error)
}

type LastSignaturesImpl struct {
}

func (LastSignaturesImpl) LastSignatures(state storage.State) (*ReverseOrdering, error) {
	var signatures []crypto.Signature

	height, err := state.Height()
	if err != nil {
		zap.S().Error(err)
		return nil, err
	}

	for i := 0; i < 100 && height > 0; i++ {
		sig, err := state.HeightToBlockID(height)
		if err != nil {
			zap.S().Error(err)
			return nil, err
		}
		signatures = append(signatures, sig)
		height -= 1
	}
	return NewSignatures(signatures...), nil
}
