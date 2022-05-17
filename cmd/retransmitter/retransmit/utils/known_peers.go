package utils

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const defaultInterval = 5 * time.Minute

type KnownPeers struct {
	knownPeers map[string]proto.Version
	mu         sync.Mutex
	storage    Storage
	cancel     context.CancelFunc
}

func NewKnownPeers(storage Storage) (*KnownPeers, error) {
	return NewKnownPeersInterval(storage, defaultInterval)
}

type JsonKnowPeerRow struct {
	Addr    string
	Version proto.Version
}

func NewKnownPeersInterval(storage Storage, saveInterval time.Duration) (*KnownPeers, error) {
	bts, err := storage.Read()
	if err != nil {
		return nil, err
	}

	var rows []JsonKnowPeerRow
	_ = json.Unmarshal(bts, &rows)

	p := make(map[string]proto.Version)
	for _, row := range rows {
		p[row.Addr] = row.Version
	}

	ctx, cancel := context.WithCancel(context.Background())

	a := &KnownPeers{
		knownPeers: p,
		storage:    storage,
		cancel:     cancel,
	}

	go a.periodicallySave(ctx, saveInterval)
	return a, nil
}

func (a *KnownPeers) periodicallySave(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := a.save()
			if err != nil {
				zap.S().Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *KnownPeers) Addresses() []proto.PeerInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []proto.PeerInfo
	for addr := range a.knownPeers {
		rs, err := proto.NewPeerInfoFromString(string(addr))
		if err != nil {
			zap.S().Error(err)
			continue
		}
		out = append(out, rs)
	}
	return out
}

func (a *KnownPeers) Add(declAddr proto.TCPAddr, version proto.Version) {
	if declAddr.Empty() {
		return
	}
	a.mu.Lock()
	a.knownPeers[declAddr.String()] = version
	a.mu.Unlock()
}

func (a *KnownPeers) GetAll() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for k := range a.knownPeers {
		out = append(out, k)
	}
	return out
}

func (a *KnownPeers) save() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var out []JsonKnowPeerRow
	for k, v := range a.knownPeers {
		out = append(out, JsonKnowPeerRow{
			Addr:    k,
			Version: v,
		})
	}

	bts, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	return a.storage.Save(bts)
}

func (a *KnownPeers) Stop() {
	a.cancel()
	err := a.save()
	if err != nil {
		zap.S().Error(err)
	}
	a.storage.Close()
}
