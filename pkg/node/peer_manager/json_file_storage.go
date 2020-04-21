package peer_manager

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type JsonFileStorage struct {
	fullpath string
	peers    []proto.TCPAddr
	sync.Mutex
}

func (a *JsonFileStorage) SavePeers(addrs []proto.TCPAddr) error {
	a.Lock()
	defer a.Unlock()
	a.peers = addrs
	bts, err := json.Marshal(addrs)
	if err != nil {
		return err
	}
	zap.S().Debugf("*JsonFileStorage SavePeers %s", string(bts))
	return ioutil.WriteFile(a.fullpath, bts, 0644)
}

func (a *JsonFileStorage) Peers() ([]proto.TCPAddr, error) {
	a.Lock()
	defer a.Unlock()
	zap.S().Debugf("*JsonFileStorage Peers %+v", a.peers)
	return a.peers, nil
}

func NewJsonFileStorage(p string) (*JsonFileStorage, error) {
	// if directory not writable or other problems, state fill fail before this
	fullpath := path.Join(p, "blocks_storage", "peers.dat")
	bts, err := ioutil.ReadFile(fullpath)
	if err != nil {
		if os.IsNotExist(err) {
			return &JsonFileStorage{
				fullpath: fullpath,
				peers:    nil,
				Mutex:    sync.Mutex{},
			}, nil
		}
		return nil, err
	}
	var peers []proto.TCPAddr
	err = json.Unmarshal(bts, &peers)
	if err != nil {
		return nil, err
	}
	return &JsonFileStorage{
		fullpath: fullpath,
		peers:    peers,
		Mutex:    sync.Mutex{},
	}, nil
}
