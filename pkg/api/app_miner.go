package api

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"time"
)

type Scheduler struct {
	TimeNow time.Time `json:"time_now"`
	Next    []struct {
		PublicKey crypto.PublicKey `json:"public_key"`
		Time      time.Time        `json:"time"`
	} `json:"next"`
}

type MinerInfo struct {
	Scheduler Scheduler
}

func (a *App) Miner() (*MinerInfo, error) {
	panic("not implemented")
}
