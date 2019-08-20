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

func (a *App) Miner() (*MinerInfo, error) {
	e := a.scheduler.Emits()

	next := make([]Next, 0)
	for _, row := range e {
		pk, err := row.KeyPair.Public()
		if err != nil {
			return nil, err
		}
		next = append(next, Next{
			PublicKey: pk,
			Time:      time.Unix(int64(row.Timestamp/1000), 0).Add(time.Duration(row.Timestamp%1000) * time.Millisecond),
		})
	}

	return &MinerInfo{
		Scheduler: Scheduler{
			TimeNow: time.Now(),
			Next:    next,
		},
	}, nil
}
