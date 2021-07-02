package state

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/wavesplatform/gowaves/cmd/wmd/internal/data"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type assetBalanceKey struct {
	address proto.WavesAddress
	asset   crypto.Digest
}

func (k *assetBalanceKey) bytes() []byte {
	buf := make([]byte, 1+proto.WavesAddressSize+crypto.DigestSize)
	buf[0] = assetBalanceKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.WavesAddressSize:], k.asset[:])
	return buf
}

type assetBalanceHistoryKey struct {
	height  uint32
	address proto.WavesAddress
	asset   crypto.Digest
}

func (k assetBalanceHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+proto.WavesAddressSize+crypto.DigestSize)
	buf[0] = assetBalanceHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[1+4:], k.address[:])
	copy(buf[1+4+proto.WavesAddressSize:], k.asset[:])
	return buf
}

func (k *assetBalanceHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+proto.WavesAddressSize+crypto.DigestSize {
		return errors.Errorf("%d bytes is not enough for assetBalanceHistoryKey", l)
	}
	data = data[1:]
	k.height = binary.BigEndian.Uint32(data)
	data = data[4:]
	copy(k.address[:], data[:proto.WavesAddressSize])
	data = data[proto.WavesAddressSize:]
	copy(k.asset[:], data[:crypto.DigestSize])
	return nil
}

type balanceDiff struct {
	prev uint64
	curr uint64
}

func (c balanceDiff) bytes() []byte {
	buf := make([]byte, 8+8)
	binary.BigEndian.PutUint64(buf, c.prev)
	binary.BigEndian.PutUint64(buf[8:], c.curr)
	return buf
}

func (c *balanceDiff) fromBytes(data []byte) error {
	if l := len(data); l < 8+8 {
		return errors.Errorf("%d is not enough bytes for balanceDiff", l)
	}
	c.prev = binary.BigEndian.Uint64(data)
	data = data[8:]
	c.curr = binary.BigEndian.Uint64(data)
	return nil
}

type assetIssuerKey struct {
	address proto.WavesAddress
	asset   crypto.Digest
}

func (k *assetIssuerKey) bytes() []byte {
	buf := make([]byte, 1+proto.WavesAddressSize+crypto.DigestSize)
	buf[0] = assetIssuerKeyPrefix
	copy(buf[1:], k.address[:])
	copy(buf[1+proto.WavesAddressSize:], k.asset[:])
	return buf
}

type assetKey struct {
	asset crypto.Digest
}

func (k *assetKey) bytes() []byte {
	buf := make([]byte, 1+crypto.DigestSize)
	buf[0] = assetKeyPrefix
	copy(buf[1:], k.asset[:])
	return buf
}

type assetHistoryKey struct {
	height uint32
	asset  crypto.Digest
}

func (k assetHistoryKey) bytes() []byte {
	buf := make([]byte, 1+4+crypto.DigestSize)
	buf[0] = assetInfoHistoryKeyPrefix
	binary.BigEndian.PutUint32(buf[1:], k.height)
	copy(buf[5:], k.asset[:])
	return buf
}

func (k *assetHistoryKey) fromBytes(data []byte) error {
	if l := len(data); l < 1+4+crypto.DigestSize {
		return errors.Errorf("%d is not enough bytes for assetHistoryKey", l)
	}
	if data[0] != assetInfoHistoryKeyPrefix {
		return errors.Errorf("%d invalid prefix for assetHistoryKey", data[0])
	}
	k.height = binary.BigEndian.Uint32(data[1:])
	copy(k.asset[:], data[5:5+crypto.DigestSize])
	return nil
}

type asset struct {
	name       string
	issuer     proto.WavesAddress
	decimals   uint8
	reissuable bool
	sponsored  bool
	supply     uint64
}

const assetInfoSize = 2 + proto.WavesAddressSize + 1 + 1 + 1 + 8

func newAssetInfoFromIssueChange(scheme byte, ch data.IssueChange) (asset, error) {
	a, err := proto.NewAddressFromPublicKey(scheme, ch.Issuer)
	if err != nil {
		return asset{}, err
	}
	return asset{name: ch.Name, issuer: a, decimals: ch.Decimals, reissuable: ch.Reissuable, sponsored: false, supply: ch.Quantity}, nil
}

func (a asset) bytes() []byte {
	nl := len(a.name)
	buf := make([]byte, assetInfoSize+nl)
	var p int
	proto.PutStringWithUInt16Len(buf[p:], a.name)
	p += 2 + nl
	copy(buf[p:], a.issuer[:])
	p += proto.WavesAddressSize
	buf[p] = a.decimals
	p++
	proto.PutBool(buf[p:], a.reissuable)
	p++
	proto.PutBool(buf[p:], a.sponsored)
	p++
	binary.BigEndian.PutUint64(buf[p:], a.supply)
	return buf
}

func (a *asset) fromBytes(data []byte) error {
	if l := len(data); l < assetInfoSize {
		return errors.Errorf("%d bytes is not enough for asset", l)
	}
	s, err := proto.StringWithUInt16Len(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal asset from bytes")
	}
	a.name = s
	data = data[2+len(s):]
	copy(a.issuer[:], data[:proto.WavesAddressSize])
	data = data[proto.WavesAddressSize:]
	a.decimals = data[0]
	data = data[1:]
	a.reissuable, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal asset from bytes")
	}
	data = data[1:]
	a.sponsored, err = proto.Bool(data)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal asset from bytes")
	}
	data = data[1:]
	a.supply = binary.BigEndian.Uint64(data)
	return nil
}

type assetHistory struct {
	supply     uint64
	reissuable bool
	sponsored  bool
}

func (v assetHistory) bytes() []byte {
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

func (v *assetHistory) fromBytes(data []byte) error {
	if l := len(data); l < 8+1+1 {
		return errors.Errorf("%d is not enough bytes for assetHistory", l)
	}
	v.supply = binary.BigEndian.Uint64(data)
	v.reissuable = data[8] == 1
	v.sponsored = data[9] == 1
	return nil
}

func putIssues(bs *blockState, batch *leveldb.Batch, scheme byte, height uint32, issueChanges []data.IssueChange) error {
	for _, u := range issueChanges {
		ai, err := newAssetInfoFromIssueChange(scheme, u)
		if err != nil {
			return errors.Wrapf(err, "failed to put issue")
		}
		ok, err := bs.isIssuer(ai.issuer, u.AssetID)
		if err != nil {
			return errors.Wrapf(err, "failed to find that the address '%s' issued asset '%s'", ai.issuer.String(), u.AssetID.String())
		}
		if !ok {
			ik := assetIssuerKey{address: ai.issuer, asset: u.AssetID}
			batch.Put(ik.bytes(), nil)
			bs.issuers[ik] = struct{}{}
		}
		k := assetKey{asset: u.AssetID}
		hk := assetHistoryKey{asset: u.AssetID, height: height}
		batch.Put(k.bytes(), ai.bytes())
		batch.Put(hk.bytes(), nil) // put empty value to show that there was nothing before
		bs.assets[k] = ai
	}
	return nil
}

func putAssets(bs *blockState, batch *leveldb.Batch, height uint32, assetChanges []data.AssetChange) error {
	historyUpdated := make(map[assetHistoryKey]struct{})
	for _, u := range assetChanges {
		var k assetKey
		var hk assetHistoryKey
		var ai asset
		var aih assetHistory

		k = assetKey{asset: u.AssetID}
		hk = assetHistoryKey{asset: u.AssetID, height: height}
		ai, ok, err := bs.assetInfo(u.AssetID)
		if err != nil {
			return errors.Wrapf(err, "failed to update assets")
		}
		if !ok {
			zap.S().Warnf("Failed to locate asset '%s'", u.AssetID.String())
			continue
		}
		//Update history only for the first change at the height
		if _, ok := historyUpdated[hk]; !ok {
			aih = assetHistory{supply: ai.supply, reissuable: ai.reissuable, sponsored: ai.sponsored}
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

func rollbackAssets(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to rollback AssetInfos")
	}
	s := uint32Key{assetInfoHistoryKeyPrefix, removeHeight}
	l := uint32Key{assetInfoHistoryKeyPrefix, math.MaxInt32}
	it := snapshot.NewIterator(&util.Range{Start: s.bytes(), Limit: l.bytes()}, nil)
	downgrade := make(map[assetKey]asset)
	remove := make([]assetKey, 0)
	if it.Last() {
		for {
			var hk assetHistoryKey
			var k assetKey
			err := hk.fromBytes(it.Key())
			if err != nil {
				return wrapError(err)
			}
			k = assetKey{asset: hk.asset}
			if len(it.Value()) == 0 {
				remove = append(remove, k)
				delete(downgrade, k)
			} else {
				var aih assetHistory
				err := aih.fromBytes(it.Value())
				if err != nil {
					return wrapError(err)
				}
				b, err := snapshot.Get(k.bytes(), nil)
				if err != nil {
					return wrapError(err)
				}
				var ai asset
				err = ai.fromBytes(b)
				if err != nil {
					return wrapError(err)
				}
				ai.sponsored = aih.sponsored
				ai.reissuable = aih.reissuable
				ai.supply = aih.supply
				downgrade[k] = ai
			}
			batch.Delete(hk.bytes())
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

func updateBalanceAndHistory(bs *blockState, batch *leveldb.Batch, height uint32, addr proto.WavesAddress, asset crypto.Digest, in, out uint64) error {
	//get current state of balance
	balance, k, err := bs.balance(addr, asset)
	if err != nil {
		return errors.Wrapf(err, "failed to get the balance")
	}
	// update the balance
	ch := balanceDiff{prev: balance}
	balance += in
	balance -= out
	ch.curr = balance
	// put new state of the balance
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, balance)
	batch.Put(k.bytes(), buf)
	hk := assetBalanceHistoryKey{height: height, asset: asset, address: addr}
	batch.Put(hk.bytes(), ch.bytes())
	bs.balances[k] = balance
	return nil
}

func putAccounts(bs *blockState, batch *leveldb.Batch, height uint32, accountChanges []data.AccountChange) error {
	for _, u := range accountChanges {
		// get the address bytes from the account or from state
		var addr proto.WavesAddress
		if !bytes.Equal(u.Account.Address[:], emptyAddress[:]) {
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
		ok, err := bs.isIssuer(addr, u.Asset)
		if err != nil {
			return errors.Wrapf(err, "failed to find that the address '%s' issued asset '%s'", addr.String(), u.Asset.String())
		}
		if ok {
			//This is an issuer's account
			err := updateBalanceAndHistory(bs, batch, height, addr, u.Asset, u.In, u.Out)
			if err != nil {
				return errors.Wrapf(err, "failed to update balance of address '%s' for asset '%s'", addr.String(), u.Asset.String())
			}
		} else {
			//This is not an issuer's account, but maybe this is a sponsored asset
			if u.MinersReward {
				// in this case if this also a miner's reward
				a, ok, err := bs.assetInfo(u.Asset)
				if err != nil {
					return errors.Wrapf(err, "failed to get the sponsorship for '%s'", u.Asset.String())
				}
				if !ok {
					zap.S().Warnf("Transaction sponsored with asset '%s' issued by Invoke", u.Asset.String())
					return nil //TODO: errors.Errorf("no asset info for an asset '%s'", u.Asset.String())
				}
				if a.sponsored {
					err := updateBalanceAndHistory(bs, batch, height, a.issuer, u.Asset, u.In, u.Out)
					if err != nil {
						return errors.Wrapf(err, "failed to update balance of address '%s' for asset '%s'", a.issuer.String(), u.Asset.String())
					}
				}
			}
		}
	}
	return nil
}

func rollbackAccounts(snapshot *leveldb.Snapshot, batch *leveldb.Batch, removeHeight uint32) error {
	s := uint32Key{assetBalanceHistoryKeyPrefix, removeHeight}
	l := uint32Key{assetBalanceHistoryKeyPrefix, math.MaxInt32}
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
			var c balanceDiff
			err = c.fromBytes(it.Value())
			if err != nil {
				return errors.Wrap(err, "failed to rollback balances")
			}
			downgrade[k] = c.prev
			batch.Delete(hk.bytes())
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
	return nil
}
