package internal

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	blocksPrefix        byte = iota // Keys to store blocks by its signatures
	blocksCounterPrefix             // Keys to store the instance of total number of blocks
	blockNumbersPrefix              // Keys to store the number of block by its signature
	blockLinksPrefix                // Keys to store the block link by the block number
	heightsPrefix                   // Keys to store the signatures of blocks by its heights
	peerLeashPrefix                 // Keys to store the numbers of blocks last seen from peers
	peerNodePrefix                  // Keys to store peers by its IPs
)

var (
	blocksCounterKey = []byte{blocksCounterPrefix}
	zeroSignature    = crypto.Signature{}
	maxSignature     = crypto.Signature{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

type peerKey struct {
	prefix byte
	ip     net.IP
}

func (k peerKey) bytes() []byte {
	buf := make([]byte, 1+net.IPv6len)
	buf[0] = k.prefix
	copy(buf[1:], k.ip.To16())
	return buf
}

func (k *peerKey) fromBytes(data []byte) {
	if l := len(data); l < 1+net.IPv6len {
		panic(fmt.Sprintf("%d is not enough bytes for peerKey", l))
	}
	k.prefix = data[0]
	k.ip = net.IP(data[1 : 1+net.IPv6len])
}

func newPeerLeashKey(ip net.IP) peerKey {
	return peerKey{prefix: peerLeashPrefix, ip: ip.To16()}
}

type signatureKey struct {
	prefix    byte
	signature crypto.Signature
}

func (k signatureKey) bytes() []byte {
	buf := make([]byte, 1+crypto.SignatureSize)
	buf[0] = k.prefix
	copy(buf[1:], k.signature[:])
	return buf
}

func newBlockKey(signature crypto.Signature) signatureKey {
	return signatureKey{prefix: blocksPrefix, signature: signature}
}

func newBlockNumberKey(signature crypto.Signature) signatureKey {
	return signatureKey{prefix: blockNumbersPrefix, signature: signature}
}

type numberKey struct {
	prefix byte
	number uint32
}

func (k numberKey) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = k.prefix
	binary.BigEndian.PutUint32(buf[1:], k.number)
	return buf
}

func (k *numberKey) fromBytes(data []byte) {
	if l := len(data); l < 5 {
		panic(fmt.Sprintf("%d is not enough bytes for numberKey", l))
	}
	k.prefix = data[0]
	k.number = binary.BigEndian.Uint32(data[1:])
}

func newBlockLinkKey(number uint32) numberKey {
	return numberKey{prefix: blockLinksPrefix, number: number}
}

type blockLink struct {
	parent    uint32
	height    uint32
	signature crypto.Signature
}

func (w blockLink) bytes() []byte {
	buf := make([]byte, 4+4+crypto.SignatureSize)
	binary.BigEndian.PutUint32(buf, w.parent)
	binary.BigEndian.PutUint32(buf[4:], w.height)
	copy(buf[8:], w.signature[:])
	return buf
}

func (w *blockLink) fromBytes(data []byte) {
	if l := len(data); l < 4+4+crypto.SignatureSize {
		panic(fmt.Sprintf("%d is not enough bytes for blockLink", l))
	}
	w.parent = binary.BigEndian.Uint32(data[0:4])
	w.height = binary.BigEndian.Uint32(data[4:8])
	copy(w.signature[:], data[8:8+crypto.SignatureSize])
}

type heightBlockKey struct {
	height uint32
	block  crypto.Signature
}

func (k heightBlockKey) bytes() []byte {
	buf := make([]byte, 1+4+crypto.SignatureSize)
	buf[0] = heightsPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], k.block[:])
	return buf
}

func (k *heightBlockKey) fromBytes(data []byte) {
	if l := len(data); l < 1+4+crypto.SignatureSize {
		panic(fmt.Sprintf("%d is not enough bytes for heightBlockKey", l))
	}
	if data[0] != heightsPrefix {
		panic("invalid heightBlockKey prefix")
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.block[:], data[1+4:1+4+crypto.SignatureSize])
}

type storage struct {
	db      *leveldb.DB
	genesis crypto.Signature
}

func NewStorage(path string, genesis crypto.Signature) (*storage, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to open storage")
	}
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, wrapError(err)
	}
	s := &storage{db: db, genesis: genesis}

	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, wrapError(err)
	}
	defer sn.Release()

	_, err = s.blockNumber(sn, genesis)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, wrapError(err)
		}
		batch := new(leveldb.Batch)

		num, err := s.nextBlockNumber(sn, batch, genesis)
		if err != nil {
			return nil, wrapError(err)
		}
		l := blockLink{parent: 0, height: 1, signature: genesis}
		batch.Put(newBlockLinkKey(num).bytes(), l.bytes())
		k := heightBlockKey{height: 1, block: genesis}
		batch.Put(k.bytes(), nil)

		err = db.Write(batch, nil)
		if err != nil {
			return nil, wrapError(err)
		}
		zap.S().Infof("Genesis block %s appended to storage", genesis)
	}
	return s, nil
}

func (s *storage) Close() error {
	return s.db.Close()
}

func (s *storage) peer(ip net.IP) (PeerNode, error) {
	peer := PeerNode{}
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return peer, err
	}
	defer sn.Release()
	k := peerKey{prefix: peerNodePrefix, ip: ip.To16()}
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return peer, err
	}
	err = peer.UnmarshalBinary(v)
	if err != nil {
		return peer, err
	}
	return peer, nil
}

func (s *storage) putPeer(ip net.IP, peer PeerNode) error {
	batch := new(leveldb.Batch)
	k := peerKey{prefix: peerNodePrefix, ip: ip.To16()}
	v, err := peer.MarshalBinary()
	if err != nil {
		return err
	}
	batch.Put(k.bytes(), v)
	err = s.db.Write(batch, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *storage) peers() ([]PeerNode, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect peers")
	}
	defer sn.Release()
	st := []byte{peerNodePrefix}
	lm := []byte{peerNodePrefix + 1}
	it := sn.NewIterator(&util.Range{Start: st, Limit: lm}, nil)
	r := make([]PeerNode, 0)
	for it.Next() {
		var v PeerNode
		err = v.UnmarshalBinary(it.Value())
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect peers")
		}
		r = append(r, v)
	}
	it.Release()
	return r, nil
}

func (s *storage) hasPeer(ip net.IP) (bool, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return false, err
	}
	defer sn.Release()
	k := peerKey{prefix: peerNodePrefix, ip: ip.To16()}
	_, err = sn.Get(k.bytes(), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *storage) appendBlock(block *proto.Block) (uint32, uint32, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to append new block")
	}
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return 0, 0, wrapError(err)
	}
	defer sn.Release()
	batch := new(leveldb.Batch)

	// Check that block is new (we don't have such signature in storage
	bk := newBlockKey(block.BlockSignature)
	ok, err := sn.Has(bk.bytes(), nil)
	if err != nil {
		return 0, 0, wrapError(err)
	}
	if ok {
		return 0, 0, nil
	}

	// Put the block bytes in storage
	bb, err := block.MarshalBinary()
	if err != nil {
		return 0, 0, err
	}
	batch.Put(bk.bytes(), bb)

	// Get the block's parent link
	parentNumber, err := s.blockNumber(sn, block.Parent)
	if err != nil {
		return 0, 0, wrapError(err)
	}
	parentLink, err := s.blockLink(sn, parentNumber)
	if err != nil {
		return 0, 0, wrapError(err)
	}

	// Acquire next block number for the block and put a new block link in the storage
	num, err := s.nextBlockNumber(sn, batch, block.BlockSignature)
	if err != nil {
		return 0, 0, wrapError(err)
	}
	link := blockLink{parent: parentNumber, height: parentLink.height + 1, signature: block.BlockSignature}
	batch.Put(newBlockLinkKey(num).bytes(), link.bytes())

	// Update blocks at height
	hk := heightBlockKey{height: link.height, block: block.BlockSignature}
	batch.Put(hk.bytes(), nil)

	err = s.db.Write(batch, nil)
	if err != nil {
		return 0, 0, wrapError(err)
	}
	return num, parentNumber, nil
}

func (s *storage) nextBlockNumber(sn *leveldb.Snapshot, batch *leveldb.Batch, signature crypto.Signature) (uint32, error) {
	v, err := sn.Get(blocksCounterKey, nil)
	var n uint32
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		n = 0
	} else {
		n = binary.BigEndian.Uint32(v)
	}
	n++
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n)
	batch.Put(blocksCounterKey, buf)
	batch.Put(newBlockNumberKey(signature).bytes(), buf)
	return n, nil
}

func (s *storage) blockNumber(sn *leveldb.Snapshot, signature crypto.Signature) (uint32, error) {
	k := newBlockNumberKey(signature)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(v), nil
}

func (s *storage) blockLink(sn *leveldb.Snapshot, number uint32) (blockLink, error) {
	k := newBlockLinkKey(number)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return blockLink{}, err
	}
	var l blockLink
	l.fromBytes(v)
	return l, nil
}

func (s *storage) movePeerLeash(peer net.IP, sig crypto.Signature) error {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return errors.Wrap(err, "failed to update peer pointer")
	}
	defer sn.Release()
	batch := new(leveldb.Batch)

	// Check that the block is already exist
	num, err := s.blockNumber(sn, sig)
	if err != nil {
		return err
	}
	// The block is already known, update the peer link
	k := newPeerLeashKey(peer)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, num)
	batch.Put(k.bytes(), buf)

	err = s.db.Write(batch, nil)
	if err != nil {
		return errors.Wrap(err, "failed to update peer link")
	}
	return nil
}

func (s *storage) block(id crypto.Signature) (*proto.Block, bool, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get block")
	}
	defer sn.Release()

	k := newBlockKey(id)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, false, errors.Wrap(err, "failed to get block")
		}
		return nil, false, nil
	}
	var b proto.Block
	err = b.UnmarshalBinary(v)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to get block")
	}
	return &b, true, nil
}

func (s *storage) blocks(height uint32) ([]crypto.Signature, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect blocks at height")
	}
	defer sn.Release()
	st := heightBlockKey{height: height, block: zeroSignature}
	lm := heightBlockKey{height: height, block: maxSignature}
	it := sn.NewIterator(&util.Range{Start: st.bytes(), Limit: lm.bytes()}, nil)
	r := make([]crypto.Signature, 0)
	for it.Next() {
		var k heightBlockKey
		k.fromBytes(it.Key())
		r = append(r, k.block)
	}
	it.Release()
	return r, nil
}

type blockLinkIterator struct {
	sn *leveldb.Snapshot
	it iterator.Iterator
}

func (i *blockLinkIterator) next() bool {
	return i.it.Next()
}

func (i *blockLinkIterator) value() (uint32, uint32, crypto.Signature) {
	var key numberKey
	key.fromBytes(i.it.Key())
	var bl blockLink
	bl.fromBytes(i.it.Value())
	return key.number, bl.parent, bl.signature
}

func (i *blockLinkIterator) close() {
	i.it.Release()
	i.sn.Release()
}

func (s *storage) newBlockLinkIterator() (*blockLinkIterator, error) {
	sn, err := s.db.GetSnapshot() // Snapshot and iterator will be released with `close` function
	if err != nil {
		return nil, err
	}
	it := sn.NewIterator(&util.Range{Start: []byte{blockLinksPrefix}, Limit: []byte{blockLinksPrefix + 1}}, nil)
	return &blockLinkIterator{sn: sn, it: it}, nil
}

func (s *storage) frontBlocks(peer net.IP, n int) ([]crypto.Signature, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to get front blocks signatures")
	}
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, wrapError(err)
	}
	defer sn.Release()
	v, err := sn.Get(newPeerLeashKey(peer).bytes(), nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, wrapError(err)
		}
		// Peer is not linked to any block, starting from the beginning
		return []crypto.Signature{s.genesis}, nil
	}
	number := binary.BigEndian.Uint32(v)
	signatures := make([]crypto.Signature, n)
	for i := 0; i < n; i++ {
		l, err := s.blockLink(sn, number)
		if err != nil {
			return nil, wrapError(err)
		}
		signatures[i] = l.signature
		if l.signature == s.genesis {
			return signatures[:i+1], nil
		}
		number = l.parent
	}
	return signatures, nil
}

func (s *storage) peersLastBlocks(include func(ip net.IP) bool) (map[uint32][]net.IP, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect peers' last blocks")
	}
	defer sn.Release()
	it := sn.NewIterator(&util.Range{Start: []byte{peerLeashPrefix}, Limit: []byte{peerLeashPrefix + 1}}, nil)
	r := make(map[uint32][]net.IP)
	for it.Next() {
		var k peerKey
		k.fromBytes(it.Key())
		ip := make([]byte, net.IPv6len)
		if !include(k.ip) {
			continue
		}
		copy(ip, k.ip)
		n := binary.BigEndian.Uint32(it.Value())
		if ips, ok := r[n]; ok {
			r[n] = append(ips, ip)
		} else {
			r[n] = []net.IP{ip}
		}
	}
	it.Release()
	return r, nil
}

/* TODO: unused code, need to write tests if it is needed or otherwise remove it.
func (s *storage) peerLastBlock(peer net.IP) (uint32, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get peer's last block")
	}
	defer sn.Release()
	k := newPeerLeashKey(peer)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get peer's last block")
	}
	return binary.BigEndian.Uint32(v), nil
}
*/

func (s *storage) link(num uint32) (blockLink, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return blockLink{}, errors.Wrap(err, "failed to locate block signature")
	}
	defer sn.Release()
	l, err := s.blockLink(sn, num)
	if err != nil {
		return blockLink{}, errors.Wrap(err, "failed to locate block signature")
	}
	return l, nil
}
