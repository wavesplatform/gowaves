package clients

import (
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type NodesWavesBalance struct {
	GoBalance    int64
	ScalaBalance int64
}

func (b *NodesWavesBalance) Add(other NodesWavesBalance) NodesWavesBalance {
	return NodesWavesBalance{
		GoBalance:    b.GoBalance + other.GoBalance,
		ScalaBalance: b.ScalaBalance + other.ScalaBalance,
	}
}

type NodesWavesBalanceAtHeight struct {
	Balance NodesWavesBalance
	Height  proto.Height
}

// SynchronisedBalances is a struct that contains BalanceInWaves of addresses and the height
// at which they were received.
type SynchronisedBalances struct {
	balances map[proto.WavesAddress]NodesWavesBalance
	Height   proto.Height
}

func NewSynchronisedBalances() SynchronisedBalances {
	return SynchronisedBalances{balances: make(map[proto.WavesAddress]NodesWavesBalance)}
}

func (b *SynchronisedBalances) Put(address proto.WavesAddress, balance NodesWavesBalance) {
	b.balances[address] = balance
}

func (b *SynchronisedBalances) GetByAccountInfo(accountInfo *config.AccountInfo) NodesWavesBalance {
	if accountInfo == nil {
		return NodesWavesBalance{}
	}
	return b.balances[accountInfo.Address]
}
