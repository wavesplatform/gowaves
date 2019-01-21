package state

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type AccountImpl struct {
	balances map[string]uint64
	address  proto.Address
}

func (a *AccountImpl) SetAssetBalance(p *proto.OptionalAsset, balance uint64) {
	a.balances[p.String()] = balance
}

func (a *AccountImpl) AssetBalance(p *proto.OptionalAsset) uint64 {
	return a.balances[p.String()]
}

func (a *AccountImpl) Address() proto.Address {
	return a.address
}
