package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/client"
	"net/url"
)

type Synchronizer struct {
	u  url.URL
	bc client.Blocks
}

func (s *Synchronizer) Start(ctx context.Context, node url.URL, storage *state.Storage) error {
	sh, err := storage.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get stored height")
	}
	bh, _, err := s.bc.Height(ctx)
	nh := int(bh.Height)
	if sh > nh {
		return errors.Errorf("impossible state: stored height %d is more than node's height %d", sh, nh)
	}
	return nil
}

func (s *Synchronizer) Stop() {

}
