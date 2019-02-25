package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
)

const (
	peersPrefix uint8 = iota // All seen peers
	nodesPrefix              // Only public nodes
	blockKeyPrefix
	heightKeyPrefix
)

type peerKey struct {
	prefix byte
	ip     net.IP
	nonce  uint64
}

func (k peerKey) bytes() []byte {
	buf := make([]byte, 1+net.IPv4len+8)
	buf[0] = k.prefix
	copy(buf[1:], k.ip.To4())
	binary.BigEndian.PutUint64(buf[1+net.IPv4len:], k.nonce)
	return buf
}

func (k *peerKey) fromByte(data []byte) error {
	if l := len(data); l < 1+net.IPv4len+8 {
		return errors.Errorf("%d is not enough bytes for peerKey", l)
	}
	k.prefix = data[0]
	k.ip = net.IP(data[1 : 1+net.IPv4len])
	k.nonce = binary.BigEndian.Uint64(data[1+net.IPv4len:])
	return nil
}

type peerInfo struct {
	port    uint16
	name    string
	version proto.Version
	block   crypto.Signature
	last    uint64
}

func (i peerInfo) bytes() []byte {
	nameLen := len(i.name)
	buf := make([]byte, 2+1+nameLen+3*4+crypto.SignatureSize+8)
	binary.BigEndian.PutUint16(buf, i.port)
	proto.PutStringWithUInt8Len(buf[2:], i.name)
	putVersion(buf[2+1+nameLen:], i.version)
	copy(buf[2+1+nameLen+3*4:], i.block[:])
	binary.BigEndian.PutUint64(buf[2+1+nameLen+3*4+crypto.SignatureSize:], i.last)
	return buf
}

func (i *peerInfo) fromBytes(data []byte) error {
	if l := len(data); l < 2+1+3*4+crypto.SignatureSize+8 {
		return errors.Errorf("%d is not enough bytes for peerInfo", l)
	}
	i.port = binary.BigEndian.Uint16(data)
	data = data[2:]
	var err error
	i.name, err = proto.StringWithUInt8Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal peerInfo")
	}
	data = data[1+len(i.name):]
	i.version, err = readVersion(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal peerInfo")
	}
	data = data[3*4:]
	copy(i.block[:], data[:crypto.SignatureSize])
	data = data[crypto.SignatureSize:]
	i.last = binary.BigEndian.Uint64(data)
	return nil
}

type nodeInfo struct {
	peerInfo
	failures uint8
	error    string
}

type heightBlockKey struct {
	height uint32
	block  crypto.Signature
}

type storage struct {
	db      *leveldb.DB
	log     *zap.SugaredLogger
	genesis crypto.Signature
}

func NewStorage(path string, log *zap.SugaredLogger, genesis crypto.Signature) (*storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open storage")
	}
	return &storage{db: db, log: log, genesis: genesis}, nil
}

func (s *storage) Close() error {
	return s.db.Close()
}

func (s *storage) PutPeer(ip net.IP, nonce uint64, info peerInfo) error {
	k := peerKey{prefix: peersPrefix, ip: ip, nonce: nonce}
	err := s.db.Put(k.bytes(), info.bytes(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to put peer")
	}
	return nil
}

func (s *storage) PutPeers(peers []peer) error {
	return nil
}

func (s *storage) GetPeers() ([]peer, error) {
	return nil, nil
}

func (s *storage) PutBlock(block proto.Block) error {
	/*	sn, err := s.db.GetSnapshot()
		if err != nil {
			return errors.Wrap(err, "failed to put block in storage")
		}
		var height uint32
		switch {
		case block.BlockSignature == s.genesis:
			height = 1
		default:
			height, err := s.getHeight(block.Parent)
			if err != nil {
				return errors.Wrap(err, "failed to find parent's height")
			}
			height++
		}
		batch := new(leveldb.Batch)
		k := heightBlockKey{height: height, block: block.BlockSignature}
		v := block
	*///batch.Put(k.bytes(), )
	return nil
}

func (s *storage) GetBlock(id crypto.Signature) (*proto.Block, error) {
	return nil, nil
}

func (s *storage) GetBlocks() ([]proto.Block, error) {
	return nil, nil
}

func (s *storage) getHeight(id crypto.Signature) (uint32, error) {
	return 0, nil
}

func putVersion(buf []byte, v proto.Version) {
	binary.BigEndian.PutUint32(buf, v.Major)
	binary.BigEndian.PutUint32(buf[4:], v.Minor)
	binary.BigEndian.PutUint32(buf[8:], v.Patch)
}

func readVersion(data []byte) (proto.Version, error) {
	if l := len(data); l < 3*4 {
		return proto.Version{}, errors.Errorf("%d is not enough bytes for Version", l)
	}
	var v proto.Version
	v.Major = binary.BigEndian.Uint32(data)
	v.Minor = binary.BigEndian.Uint32(data[4:])
	v.Patch = binary.BigEndian.Uint32(data[8:])
	return v, nil
}
