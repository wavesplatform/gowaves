package internal

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/state"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Synchronizer struct {
	interrupt <-chan struct{}
	done      chan struct{}
	client    *client.Client
	log       *zap.SugaredLogger
	storage   *state.Storage
	scheme    byte
	matcher   crypto.PublicKey
	mu        *sync.RWMutex
	active    bool
	ticker    *time.Ticker
}

const (
	heightCheckInterval = 10
)

func NewSynchronizer(interrupt <-chan struct{}, log *zap.SugaredLogger, storage *state.Storage, scheme byte, matcher crypto.PublicKey, node url.URL) (*Synchronizer, error) {
	c, err := client.NewClient(client.Options{BaseUrl: node.String(), Client: &http.Client{}})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new synchronizer")
	}
	t := time.NewTicker(time.Duration(heightCheckInterval) * time.Second)
	done := make(chan struct{})
	s := Synchronizer{interrupt: interrupt, done: done, client: c, log: log, storage: storage, scheme: scheme, matcher: matcher, mu: new(sync.RWMutex), active: false, ticker: t}
	go s.run()
	return &s, nil
}

func (s *Synchronizer) Pause() {
	s.mu.Lock()
	if s.active {
		s.active = false
	}
	s.mu.Unlock()
}

func (s *Synchronizer) Resume() {
	s.mu.Lock()
	if !s.active {
		s.active = true
	}
	s.mu.Unlock()
}

func (s *Synchronizer) Done() <-chan struct{} {
	return s.done
}

func (s *Synchronizer) run() {
	defer close(s.done)
	for {
		select {
		case <-s.interrupt:
			s.log.Info("Shutting down synchronizer...")
			s.ticker.Stop()
			return
		case <-s.ticker.C:
			s.mu.RLock()
			if s.active {
				s.synchronize()
			}
			s.mu.RUnlock()
		}
	}
}

func (s *Synchronizer) synchronize() {
	rh, err := s.nodeHeight()
	if err != nil {
		s.log.Error("Failed to synchronize with node", err)
		return
	}
	lh, err := s.storage.Height()
	if err != nil {
		s.log.Error("Failed to synchronize with node", err)
	}
	if rh > lh {
		s.log.Infof("Local height %d, node height %d, have to sync %d blocks", lh, rh, rh-lh)
	}
}

func (s *Synchronizer) nodeHeight() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	bh, _, err := s.client.Blocks.Height(ctx)
	if err != nil {
		return 0, err
	}
	return int(bh.Height), nil
}
