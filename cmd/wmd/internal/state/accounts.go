package state

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"math"
)

type assetBalanceKey struct {
	address proto.Address
	asset   crypto.Digest
}

func (k *assetBalanceKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressSize+crypto.DigestSize)
	buf[0] = AssetBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.AddressSize:], k.asset[:])
	return buf
}

type assetBalanceHistoryKey struct {
	height  uint32
	address proto.Address
	asset   crypto.Digest
}

func (k *assetBalanceHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+proto.AddressSize+crypto.DigestSize)
	buf[0] = AssetBalanceHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], k.address[:])
	copy(buf[1+4+proto.AddressSize:], k.asset[:])
	return buf
}

func (k *assetBalanceHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+proto.AddressSize+crypto.DigestSize {
		return errors.Errorf("%d bytes is not enough for assetBalanceHistoryKey", l)
	}
	data = data[1:]
	k.height = binary.BigEndian.Uint32(data)
	data = data[4:]
	copy(k.address[:], data[:proto.AddressSize])
	data = data[proto.AddressSize:]
	copy(k.asset[:], data[:crypto.DigestSize])
	return nil
}

type balanceChange struct {
	prev uint64
	curr uint64
}

func (c *balanceChange) bytes() []byte {
	buf := make([]byte, 8+8)
	binary.BigEndian.PutUint64(buf, c.prev)
	binary.BigEndian.PutUint64(buf[8:], c.curr)
	return buf
}

func (c *balanceChange) fromBytes(data []byte) error {
	if l := len(data); l < 8+8 {
		return errors.Errorf("%d is not enough bytes for balanceChange", l)
	}
	c.prev = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.curr = binary.BigEndian.Uint64(data)
	return nil
}

type assetIssuerKey struct {
	address proto.Address
	asset   crypto.Digest
}

func (k *assetIssuerKey) bytes() []byte {
	buf := make([]byte, 1+proto.AddressSize+crypto.DigestSize)
	buf[0] = AssetIssuerKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.AddressSize:], k.asset[:])
	return buf
}

type assetInfoKey struct {
	asset crypto.Digest
}

func (k *assetInfoKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = AssetInfoKeyPrefix
	copy(buf[1:], k.asset[:])
	return buf
}

type assetInfoHistoryKey struct {
	height uint32
	asset  crypto.Digest
}

func (k *assetInfoHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+crypto.DigestSize)
	buf[0] = AssetInfoHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[5:], k.asset[:])
	return buf
}

func (k *assetInfoHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for assetInfoHistoryKey", l)
	}
	if data[0] != AssetInfoHistoryKeyPrefix {
		return errors.Errorf("%d invalid prefix for assetInfoHistoryKey", data[0])
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.asset[:], data[5:5+crypto.DigestSize])
	return nil
}

// assetInfo is the structure to store asset's description in the state.
type assetInfo struct {
	name       string
	issuer     crypto.PublicKey
	decimals   uint8
	reissuable bool
	sponsored  bool
	supply     uint64
}

const assetInfoSize = 2 + crypto.PublicKeySize + 1 + 1 + 1 + 8

func newAssetInfoFromIssueChange(ch IssueChange) assetInfo {
	return assetInfo{name: ch.Name, issuer: ch.Issuer, decimals: ch.Decimals, reissuable: ch.Reissuable, sponsored: false, supply: ch.Quantity}
}

func (a *assetInfo) bytes() []byte {
	nl := len(a.name)
	buf := make([]byte, assetInfoSize+nl)
	var p int
	proto.PutStringWithUInt16Len(buf[p:], a.name)
	p += 2 + nl
	copy(buf[p:], a.issuer[:])
	p += crypto.PublicKeySize
	buf[p] = a.decimals
	p++
	proto.PutBool(buf[p:], a.reissuable)
	p++
	proto.PutBool(buf[p:], a.sponsored)
	p++
	binary.BigEndian.PutUint64(buf[p:], a.supply)
	return buf
}

func (a *assetInfo) fromBytes(data []byte) error {
	if l := len(data); l < assetInfoSize {
		return errors.Errorf("%d bytes is not enough for assetInfo", l)
	}
	s, err := proto.StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal assetInfo from bytes")
	}
	a.name = s
	data = data[2+len(s):]
	copy(a.issuer[:], data[:crypto.PublicKeySize])
	data = data[crypto.PublicKeySize:]
	a.decimals = data[0]
	data = data[1:]
	a.reissuable, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal assetInfo from bytes")
	}
	data = data[1:]
	a.sponsored, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal assetInfo from bytes")
	}
	data = data[1:]
	a.supply = binary.BigEndian.Uint64(data)
	return nil
}

type assetInfoHistory struct {
	supply     uint64
	reissuable bool
	sponsored  bool
}

func (v *assetInfoHistory) bytes() []byte {
	buf := make([]byte, 8+1+1)
	binary.BigEndian.PutUint64(buf, v.supply)
	if v.reissuable {
		buf[8] = 1
	}
	if v.sponsored {
		buf[9] = 1
	}
	return buf
}

func (v *assetInfoHistory) fromBytes(data []byte) error {
	if l := len(data); l < 8+1+1 {
		return errors.Errorf("%d is not enough bytes for assetInfoHistory", l)
	}
	v.supply = binary.BigEndian.Uint64(data)
	v.reissuable = data[8] == 1
	v.sponsored = data[9] == 1
	return nil
}

func putIssues(bs *blockState, batch *leveldb.Batch, scheme byte, height uint32, updates []IssueChange) error {
	for _, u := range updates {
		addr, err := proto.NewAddressFromPublicKey(scheme, u.Issuer)
		if err != nil {
			return errors.Wrapf(err, "failed to create an address from the public key")
		}
		ok, err := bs.isIssuer(addr, u.AssetID)
		if err != nil {
			return errors.Wrapf(err, "failed to find that the address '%s' issued asset '%s'", addr.String(), u.AssetID.String())
		}
		if !ok {
			ik := assetIssuerKey{address: addr, asset: u.AssetID}
			batch.Put(ik.bytes(), nil)
			bs.issuers[ik] = struct{}{}
		}
		k := assetInfoKey{asset: u.AssetID}
		hk := assetInfoHistoryKey{asset: u.AssetID, height: height}
		ai := newAssetInfoFromIssueChange(u)
		batch.Put(k.bytes(), ai.bytes())
		batch.Put(hk.bytes(), nil) // put here empty value to show that where was nothing before
		bs.assets[k] = ai
	}
	return nil
}

func putAssetChanges(bs *blockState, batch *leveldb.Batch, height uint32, updates []AssetChange) error {
	historyUpdated := make(map[assetInfoHistoryKey]struct{})
	for _, u := range updates {
		var k assetInfoKey
		var hk assetInfoHistoryKey
		var ai assetInfo
		var aih assetInfoHistory

		k = assetInfoKey{asset: u.AssetID}
		hk = assetInfoHistoryKey{asset: u.AssetID, height: height}
		ai, ok, err := bs.assetInfo(u.AssetID)
		if err != nil {
			return errors.Wrapf(err, "failed to update assets")
		}
		if !ok {
			return errors.Errorf("failed to locate asset to update")
		}
		//Update history only for the first change at the height
		if _, ok := historyUpdated[hk]; !ok {
			aih = assetInfoHistory{supply: ai.supply, reissuable: ai.reissuable, sponsored: ai.sponsored}
			batch.Put(hk.bytes(), aih.bytes())
			historyUpdated[hk] = struct{}{}
		}
		if u.SetReissuable {
			ai.reissuable = u.Reissuable
		}
		if u.SetSponsored {
			ai.sponsored = u.Sponsored
		}
		ai.supply -= u.Burned
		ai.supply += u.Issued
		batch.Put(k.bytes(), ai.bytes())
		bs.assets[k] = ai
	}
	return nil
}

func rollbackAssetInfos(snapshot *leveldb.Snapshot, batch *leveldb.Batch, height uint32) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to rollback AssetInfos")
	}
	s := uint32Key{AssetInfoHistoryKeyPrefix, height}
	l := uint32Key{AssetInfoHistoryKeyPrefix, math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	downgrade := make(map[assetInfoKey]assetInfo)
	remove := make([]assetInfoKey, 0)
	if it.Last() {
		for {
			var hk assetInfoHistoryKey
			var k assetInfoKey
			err := hk.fromBytes(it.Key())
			if err != nil {
				return wrapError(err)
			}
			k = assetInfoKey{asset: hk.asset}
			if len(it.Value()) == 0 {
				remove = append(remove, k)
			} else {
				var aih assetInfoHistory
				err := aih.fromBytes(it.Value())
				if err != nil {
					return wrapError(err)
				}
				b, err := snapshot.Get(k.bytes(), nil)
				if err != nil {
					return wrapError(err)
				}
				var ai assetInfo
				err = ai.fromBytes(b)
				if err != nil {
					return wrapError(err)
				}
				ai.sponsored = aih.sponsored
				ai.reissuable = aih.reissuable
				ai.supply = aih.supply
				downgrade[k] = ai
			}
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	for k, v := range downgrade {
		batch.Put(k.bytes(), v.bytes())
	}
	for _, k := range remove {
		batch.Delete(k.bytes())
	}
	return nil
}

func putBalancesStateUpdate(bs *blockState, batch *leveldb.Batch, height uint32, updates []AccountChange) error {
	for _, u := range updates {
		// get the address bytes from the account or from state
		var addr proto.Address
		if !bytes.Equal(u.Account.Address[:], EmptyAddress[:]) {
			addr = u.Account.Address
		} else {
			a, ok, err := bs.addressByAlias(u.Account.Alias)
			if !ok {
				if err != nil {
					return errors.Wrapf(err, "failed to find address for alias '%s'", u.Account.Alias.String())
				}
				return errors.Errorf("failed to find address for alias '%s'", u.Account.Alias.String())
			}
			addr = a
		}
		// filter issuers only
		ok, err := bs.isIssuer(addr, u.Asset)
		if err != nil {
			return errors.Wrapf(err, "failed to find that the address '%s' issued asset '%s'", addr.String(), u.Asset.String())
		}
		if ok {
			//get current balance
			balance, k, err := bs.balance(addr, u.Asset)
			if err != nil {
				return errors.Wrapf(err, "failed to get the balance")
			}
			// update the balance
			ch := balanceChange{prev: balance}
			balance += u.In
			balance -= u.Out
			ch.curr = balance
			// put new state of the balance
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, balance)
			batch.Put(k.bytes(), buf)
			hk := assetBalanceHistoryKey{height: height, asset: u.Asset, address: addr}
			batch.Put(hk.bytes(), ch.bytes())
			bs.balances[k] = balance
		}
	}
	return nil
}

func rollbackBalances(snapshot *leveldb.Snapshot, batch *leveldb.Batch, height uint32) error {
	s := uint32Key{AssetBalanceHistoryKeyPrefix, height}
	l := uint32Key{AssetBalanceHistoryKeyPrefix, math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	downgrade := make(map[assetBalanceKey]uint64)
	if it.Last() {
		for {
			var hk assetBalanceHistoryKey
			var k assetBalanceKey
			err := hk.fromBytes(it.Key())
			if err != nil {
				return errors.Wrapf(err, "failed to rollback balances")
			}
			k = assetBalanceKey{address: hk.address, asset: hk.asset}
			var c balanceChange
			err = c.fromBytes(it.Value())
			if err != nil {
				return errors.Wrap(err, "failed to rollback balances")
			}
			downgrade[k] = c.prev
			if !it.Prev() {
				break
			}
		}
	}
	it.Release()
	for k, v := range downgrade {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v)
		batch.Put(k.bytes(), buf)
	}
	return it.Error()
}
