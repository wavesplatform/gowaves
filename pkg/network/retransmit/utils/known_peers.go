package utils

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/spf13/afero"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const defaultInterval = 5 * time.Minute

type Storage interface {
	Save([]byte) error
	Read() ([]byte, error)
	Close()
}

type FileBasedStorage struct {
	f afero.File
}

func NewFileBasedStorage(fs afero.Fs, pathToFile string) (*FileBasedStorage, error) {
	f, err := fs.OpenFile(pathToFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &FileBasedStorage{
		f: f,
	}, nil
}

func (a *FileBasedStorage) Save(b []byte) error {
	err := a.f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = a.f.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = a.f.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (a *FileBasedStorage) Read() ([]byte, error) {
	bts, err := ioutil.ReadAll(a.f)
	if err != nil {
		return nil, err
	}
	return bts, nil
}

func (a *FileBasedStorage) Close() {
	_ = a.f.Close()
}

type NoOnStorage struct{}

func (a NoOnStorage) Read() ([]byte, error) {
	return []byte{}, nil
}

func (a NoOnStorage) Save(b []byte) error {
	return nil
}

func (a NoOnStorage) Close() {}

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
	for {
		select {
		case <-time.After(interval):
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

func (a *KnownPeers) Add(declAddr proto.PeerInfo, version proto.Version) {
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

	err = a.storage.Save(bts)
	if err != nil {
		return err
	}

	return nil
}

func (a *KnownPeers) Stop() {
	a.cancel()
	err := a.save()
	if err != nil {
		zap.S().Error(err)
	}
	_ = a.storage.Close
}

func (a *KnownPeers) exitWithoutSave() {
	a.cancel()
	_ = a.storage.Close
}
