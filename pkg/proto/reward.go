package proto

import (
	"bytes"
	"sort"
)

type Reward struct {
	address WavesAddress
	amount  uint64
}

func NewReward(address WavesAddress, amount uint64) Reward {
	return Reward{address: address, amount: amount}
}

func (r *Reward) Address() WavesAddress {
	return r.address
}

func (r *Reward) Amount() uint64 {
	return r.amount
}

type Rewards []Reward

func (r *Rewards) Sorted() Rewards {
	rewards := *r
	sort.Stable(rewardsByAddress(rewards))
	return rewards
}

type rewardsByAddress []Reward

func (r rewardsByAddress) Len() int      { return len(r) }
func (r rewardsByAddress) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func (r rewardsByAddress) Less(i, j int) bool {
	//TODO: Consider adding second sort order by amount, which is not scala implementation compatible for now.
	// In scala implementation they depend on stability of sort and add miner on the last position.
	// if r[i].address.Equal(r[j].address) {
	//	 return r[i].amount < r[j].amount
	// }
	return bytes.Compare(r[i].address.Bytes(), r[j].address.Bytes()) < 0
}
