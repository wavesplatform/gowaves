package internal

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sort"
)

const (
	forkCounterPrefix byte = iota
	forkHeadersPrefix
	blockWrappersPrefix
	blocksPrefix
	heightsPrefix
	linksPrefix
	publicAddressesPrefix
)

var (
	forkCounterKey = []byte{forkCounterPrefix}
	zeroSignature  = crypto.Signature{}
	maxSignature   = crypto.Signature{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
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

func newPeerLinkKey(d PeerDesignation) peerKey {
	return peerKey{prefix: linksPrefix, ip: d.Address.To4(), nonce: d.Nonce}
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

func newBlockWrapperKey(block crypto.Signature) signatureKey {
	return signatureKey{prefix: blockWrappersPrefix, signature: block}
}

func newBlockKey(block crypto.Signature) signatureKey {
	return signatureKey{prefix: blocksPrefix, signature: block}
}

type forkHeaderKey uint32

func (k forkHeaderKey) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = forkHeadersPrefix
	binary.BigEndian.PutUint32(buf[1:], uint32(k))
	return buf
}

type forkHeader struct {
	height uint32
	length uint32
	last   crypto.Signature
	common crypto.Signature
}

func (h forkHeader) bytes() []byte {
	buf := make([]byte, 4+4+2*crypto.SignatureSize)
	binary.BigEndian.PutUint32(buf, h.height)
	binary.BigEndian.PutUint32(buf[4:], h.length)
	copy(buf[4+4:], h.last[:])
	copy(buf[4+4+crypto.SignatureSize:], h.common[:])
	return buf
}

func (h *forkHeader) fromBytes(data []byte) error {
	if l := len(data); l < 4+4+2*crypto.SignatureSize {
		return errors.Errorf("%d is not enough bytes for forkHeader", l)
	}
	h.height = binary.BigEndian.Uint32(data[:4])
	h.length = binary.BigEndian.Uint32(data[4:8])
	copy(h.last[:], data[8:8+crypto.SignatureSize])
	copy(h.common[:], data[8+crypto.SignatureSize:8+2*crypto.SignatureSize])
	return nil
}

type blockWrapper struct {
	height uint32
	fork   uint32
	parent crypto.Signature
}

func (w blockWrapper) bytes() []byte {
	buf := make([]byte, 4+4+crypto.SignatureSize)
	binary.BigEndian.PutUint32(buf, w.height)
	binary.BigEndian.PutUint32(buf[4:], w.fork)
	copy(buf[8:], w.parent[:])
	return buf
}

func (w *blockWrapper) fromBytes(data []byte) error {
	if l := len(data); l < 4+4+crypto.SignatureSize {
		return errors.Errorf("%d is not enough bytes for blockWrapper", l)
	}
	w.height = binary.BigEndian.Uint32(data[0:4])
	w.fork = binary.BigEndian.Uint32(data[4:8])
	copy(w.parent[:], data[8:8+crypto.SignatureSize])
	return nil
}

type peerLink struct {
	fork   uint32
	height uint32
	block  crypto.Signature
}

func (l peerLink) bytes() []byte {
	buf := make([]byte, 4+4+crypto.SignatureSize)
	binary.BigEndian.PutUint32(buf, l.fork)
	binary.BigEndian.PutUint32(buf[4:], l.height)
	copy(buf[4+4:], l.block[:])
	return buf
}

func (l *peerLink) fromBytes(data []byte) error {
	if l := len(data); l < 4+4+crypto.SignatureSize {
		return errors.Errorf("%d is not enough bytes for peerLink", l)
	}
	l.fork = binary.BigEndian.Uint32(data[0:4])
	l.height = binary.BigEndian.Uint32(data[4:8])
	copy(l.block[:], data[8:8+crypto.SignatureSize])
	return nil
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

func (k *heightBlockKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+crypto.SignatureSize {
		return errors.Errorf("%d is not enough bytes for heightBlockKey", l)
	}
	if data[0] != heightsPrefix {
		return errors.New("invalid heightBlockKey prefix")
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.block[:], data[1+4:1+4+crypto.SignatureSize])
	return nil
}

type publicAddressKey struct {
	addr PeerAddr
}

func (k *publicAddressKey) bytes() []byte {
	buf := make([]byte, 1+PeerAddrLen)
	buf[0] = publicAddressesPrefix
	b, err := k.addr.MarshalBinary()
	if err != nil {
		panic("no error expected")
	}
	copy(buf[1:], b)
	return buf
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
	_, ok, err := wrapper(sn, genesis)
	if err != nil {
		return nil, wrapError(err)
	}
	if !ok {
		batch := new(leveldb.Batch)
		_, _, err := putGenesisBlockWrapper(batch, genesis)
		if err != nil {
			return nil, wrapError(err)
		}
		err = db.Write(batch, nil)
		if err != nil {
			return nil, wrapError(err)
		}
		zap.S().Infof("Genesis block %s appended", genesis)
	}
	return s, nil
}

func (s *storage) Close() error {
	return s.db.Close()
}

func (s *storage) handleBlock(block proto.Block, peer PeerDesignation) error {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to append new block")
	}
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return wrapError(err)
	}
	defer sn.Release()
	batch := new(leveldb.Batch)

	// Check that the block is actually new
	w, ok, err := wrapper(sn, block.BlockSignature)
	if err != nil {
		return wrapError(err)
	}
	if ok {
		// The block is already known, just update link
		link := peerLink{fork: w.fork, height: w.height, block: block.BlockSignature}
		putLink(batch, peer, link)
		err = s.db.Write(batch, nil)
		if err != nil {
			return wrapError(err)
		}
		return nil
	}
	fid, h, err := putNewBlock(sn, batch, block, s.genesis)
	if err != nil {
		return wrapError(err)
	}
	link := peerLink{fork: fid, height: h, block: block.BlockSignature}
	putLink(batch, peer, link)
	err = s.db.Write(batch, nil)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

func (s *storage) appendBlockSignature(sig crypto.Signature, peer PeerDesignation) (bool, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to append new block signature")
	}
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return false, wrapError(err)
	}
	defer sn.Release()
	batch := new(leveldb.Batch)

	// Check that the block is already exist
	w, ok, err := wrapper(sn, sig)
	if err != nil {
		return false, wrapError(err)
	}
	if ok {
		// The block is already known, update the peer link
		link := peerLink{fork: w.fork, height: w.height, block: sig}
		putLink(batch, peer, link)
		err = s.db.Write(batch, nil)
		if err != nil {
			return false, wrapError(err)
		}
		return true, nil
	}
	return false, nil
}

func (s *storage) parentedForks() ([]Fork, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to collect parented forks")
	}

	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, wrapError(err)
	}
	defer sn.Release()

	it := sn.NewIterator(&util.Range{Start: []byte{linksPrefix}, Limit: []byte{linksPrefix + 1}}, nil)
	defer it.Release()

	m := make(map[uint32]Fork, 0)
	for it.Next() {
		var k peerKey
		err = k.fromByte(it.Key())
		if err != nil {
			return nil, wrapError(err)
		}
		pd := NewPeerDesignation(k.ip.To4(), k.nonce)

		var link peerLink
		err = link.fromBytes(it.Value())
		if err != nil {
			return nil, wrapError(err)
		}

		f, ok := m[link.fork]
		if !ok {
			f = Fork{}
			fh, err := header(sn, link.fork)
			if err != nil {
				return nil, wrapError(err)
			}
			f.HeadBlock = fh.last
			f.CommonBlock = fh.common
			f.Height = int(fh.height)
			f.Length = int(fh.length)
		}
		lag := f.Height - int(link.height)
		f.Peers = append(f.Peers, NewPeerForkInfo(pd, lag))
		m[link.fork] = f
	}
	r := make([]Fork, len(m))
	i := 0
	for _, f := range m {
		r[i] = f
		i++
	}
	sort.Sort(ForkByHeightLengthAndPeersCount(r))
	r[0].Longest = true
	return r, nil
}

func (s *storage) publicAddresses() ([]PublicAddress, error) {
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect public addresses")
	}
	st := []byte{publicAddressesPrefix}
	lm := []byte{publicAddressesPrefix + 1}
	it := sn.NewIterator(&util.Range{Start: st, Limit: lm}, nil)
	r := make([]PublicAddress, 0)
	for it.Next() {
		var v PublicAddress
		err = v.UnmarshalBinary(it.Value())
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect public addresses")
		}
		r = append(r, v)
	}
	return r, nil
}

func (s *storage) hasPublicAddress(a PeerAddr) (bool, error) {
	k := publicAddressKey{addr: a}
	ok, err := s.db.Has(k.bytes(), nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to check public address presence")
	}
	return ok, nil
}

func (s *storage) putPublicAddress(pa PublicAddress) error {
	k := publicAddressKey{addr: pa.address}
	v, err := pa.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "failed to store public address")
	}
	err = s.db.Put(k.bytes(), v, nil)
	if err != nil {
		return errors.Wrap(err, "failed to store public address")
	}
	return nil
}

func putGenesisBlockWrapper(batch *leveldb.Batch, genesis crypto.Signature) (fork uint32, height uint32, err error) {
	// update wrapper
	w := blockWrapper{height: 1, fork: 0, parent: zeroSignature}
	batch.Put(newBlockWrapperKey(genesis).bytes(), w.bytes())
	// update last fork id
	updateLastForkID(batch, 0)
	// put fork header
	fh := forkHeader{height: 1, length: 1, last: genesis, common: genesis}
	batch.Put(forkHeaderKey(0).bytes(), fh.bytes())
	// update blocks at height
	k := heightBlockKey{height: 1, block: genesis}
	batch.Put(k.bytes(), nil)
	return 0, 1, nil
}

func putNewBlock(sn *leveldb.Snapshot, batch *leveldb.Batch, block proto.Block, genesis crypto.Signature) (fork uint32, height uint32, err error) {
	bb, err := block.MarshalBinary()
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to marshal new block")
	}
	batch.Put(newBlockKey(block.BlockSignature).bytes(), bb)

	if block.BlockSignature == genesis {
		return putGenesisBlockWrapper(batch, block.BlockSignature)
	}

	var nw blockWrapper
	pw, ok, err := wrapper(sn, block.Parent)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to put new block")
	}
	if !ok {
		return 0, 0, errors.Errorf("no wrapper for parent block '%s'", block.Parent)
	}
	pc := forksCountAtHeight(sn, pw.height)
	height = pw.height + 1
	c := forksCountAtHeight(sn, height)
	var fh forkHeader
	if c < pc { // continue fork
		nw = blockWrapper{height: height, fork: pw.fork, parent: block.Parent}
		fork = pw.fork
		fh, err = header(sn, fork)
		if err != nil {
			return 0, 0, errors.Wrap(err, "failed to put new block")
		}
		fh.height = height
		fh.last = block.BlockSignature
		fh.length = fh.length + 1
	} else { // new fork
		fork, err = numberOfForks(sn)
		if err != nil {
			return 0, 0, errors.Wrap(err, "failed to put new block")
		}
		fork++
		updateLastForkID(batch, fork)
		nw = blockWrapper{height: height, fork: fork, parent: block.Parent}
		fh = forkHeader{height: height, length: 1, last: block.BlockSignature, common: nw.parent}
	}
	// Store fork header
	batch.Put(forkHeaderKey(fork).bytes(), fh.bytes())
	// Store the blockWrapper
	batch.Put(newBlockWrapperKey(block.BlockSignature).bytes(), nw.bytes())
	// update blocks at height
	k2 := heightBlockKey{height: height, block: block.BlockSignature}
	batch.Put(k2.bytes(), nil)
	return fork, height, nil
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
	//TODO: implement with existing method for counting blocks at height
	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect blocks")
	}
	st := heightBlockKey{height: height, block: zeroSignature}
	lm := heightBlockKey{height: height, block: maxSignature}
	it := sn.NewIterator(&util.Range{Start: st.bytes(), Limit: lm.bytes()}, nil)
	r := make([]crypto.Signature, 0)
	for it.Next() {
		var k heightBlockKey
		err = k.fromBytes(it.Key())
		if err != nil {
			return nil, errors.Wrap(err, "failed to collect blocks")
		}
		r = append(r, k.block)
	}
	return r, nil
}

func (s *storage) fork(ip net.IP) ([]NodeForkInfo, error) {
	//TODO: implement
	return nil, nil
}

func (s *storage) frontBlocks(peer PeerDesignation, n int) ([]crypto.Signature, error) {
	wrapError := func(err error) error {
		return errors.Wrap(err, "failed to get front blocks signatures")
	}

	sn, err := s.db.GetSnapshot()
	if err != nil {
		return nil, wrapError(err)
	}
	defer sn.Release()

	k := newPeerLinkKey(peer)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			// No link between peer and fork head, starting to request blocks from new peer
			return []crypto.Signature{s.genesis}, nil
		}
		return nil, wrapError(err)
	}
	var link peerLink
	err = link.fromBytes(v)
	if err != nil {
		return nil, wrapError(err)
	}

	signatures := make([]crypto.Signature, n)
	signatures[0] = link.block
	for i := 1; i < n; i++ {
		ps, err := parent(sn, signatures[i-1])
		signatures[i] = ps
		if err != nil {
			return nil, wrapError(err)
		}
		if ps == s.genesis {
			return signatures[:i+1], nil
		}
	}
	return signatures, nil
}

func parent(sn *leveldb.Snapshot, sig crypto.Signature) (crypto.Signature, error) {
	k := newBlockKey(sig)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return crypto.Signature{}, err
	}
	var b proto.Block
	err = b.UnmarshalBinary(v)
	if err != nil {
		return crypto.Signature{}, err
	}
	return b.Parent, nil
}

func numberOfForks(sn *leveldb.Snapshot) (uint32, error) {
	v, err := sn.Get(forkCounterKey, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, errors.Wrap(err, "failed to get the number of parentedForks")
	}
	return binary.BigEndian.Uint32(v), nil
}

func updateLastForkID(batch *leveldb.Batch, n uint32) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n)
	batch.Put(forkCounterKey, buf)
}

func wrapper(sn *leveldb.Snapshot, block crypto.Signature) (blockWrapper, bool, error) {
	k := newBlockWrapperKey(block)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return blockWrapper{}, false, nil
		}
		return blockWrapper{}, false, errors.Wrap(err, "failed to locate the blockWrapper")
	}
	var w blockWrapper
	err = w.fromBytes(v)
	if err != nil {
		return blockWrapper{}, false, errors.Wrap(err, "failed to unmarshal the blockWrapper")
	}
	return w, true, nil
}

func putLink(batch *leveldb.Batch, peer PeerDesignation, link peerLink) {
	k := newPeerLinkKey(peer)
	batch.Put(k.bytes(), link.bytes())
}

func forksCountAtHeight(sn *leveldb.Snapshot, height uint32) int {
	st := heightBlockKey{height: height, block: zeroSignature}
	lm := heightBlockKey{height: height, block: maxSignature}
	it := sn.NewIterator(&util.Range{Start: st.bytes(), Limit: lm.bytes()}, nil)
	defer it.Release()
	n := 0
	for it.Next() {
		n++
	}
	return n
}

func header(sn *leveldb.Snapshot, fork uint32) (forkHeader, error) {
	k := forkHeaderKey(fork)
	v, err := sn.Get(k.bytes(), nil)
	if err != nil {
		return forkHeader{}, errors.Wrap(err, "failed to get forkHeader")
	}
	var fh forkHeader
	err = fh.fromBytes(v)
	if err != nil {
		return forkHeader{}, errors.Wrap(err, "failed to unmarshal forkHeader")
	}
	return fh, nil
}
