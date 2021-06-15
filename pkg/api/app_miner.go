package api

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

type Scheduler struct {
	TimeNow time.Time `json:"time_now"`
	Next    []Next    `json:"next"`
}
type Next struct {
	PublicKey crypto.PublicKey `json:"public_key"`
	Time      time.Time        `json:"time"`
}

type MinerInfo struct {
	Scheduler Scheduler
}

func (a *App) Miner() MinerInfo {
	e := a.scheduler.Emits()

	next := make([]Next, 0, len(e))
	for _, row := range e {
		next = append(next, Next{
			PublicKey: row.KeyPair.Public,
			Time:      time.Unix(int64(row.Timestamp/1000), 0).Add(time.Duration(row.Timestamp%1000) * time.Millisecond),
		})
	}

	return MinerInfo{
		Scheduler: Scheduler{
			TimeNow: time.Now(),
			Next:    next,
		},
	}
}
