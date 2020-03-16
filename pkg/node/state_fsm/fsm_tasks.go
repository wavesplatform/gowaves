package state_fsm

import (
	"context"
	"time"
)

type AskPeersTask struct {
	Type int
	d    time.Duration
}

func NewAskPeersTask(d time.Duration) AskPeersTask {
	return AskPeersTask{
		Type: ASK_PEERS,
		d:    d,
	}
}

func (a AskPeersTask) Run(ctx context.Context, output chan AsyncTask) error {
	<-time.After(5 * time.Second)
	output <- AsyncTask{
		TaskType: a.Type,
	}

	in := time.NewTicker(a.d)
	defer in.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-in.C:
			output <- AsyncTask{
				TaskType: a.Type,
			}
		}
	}
}
