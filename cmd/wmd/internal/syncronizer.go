package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/client"
	"net/url"
)

type Sycronizer struct {
	u  url.URL
	bc client.Blocks
}

func (s *Sycronizer) Start(ctx context.Context, node url.URL, storage *Storage) error {
	sh, err := storage.GetHeight()
	if err != nil {
		return errors.Wrap(err, "failed to get stored height")
	}
	bh, _, err := s.bc.Height(ctx)
	nh := int(bh.Height)
	if sh > nh {
		return errors.Errorf("impossible state: stored height %d is more than node's height", sh, nh)
	}
	return nil
}

func (s *Sycronizer) Stop() {

}
