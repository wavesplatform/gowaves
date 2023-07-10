package tasks

import (
	"context"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	Ping = iota + 1
	AskPeers
	MineMicro
	PersistComplete
)

// SendAsyncTask sends task into channel with overflow check.
func SendAsyncTask(output chan AsyncTask, task AsyncTask) {
	select {
	case output <- task:
	default:
		zap.S().Debugf("AsyncTask channel is full on task '%T'", task)
	}
}

type TaskType int

type TaskData interface{ taskDataMarker() }

type AsyncTask struct {
	TaskType int
	Data     TaskData
}

type Task interface {
	Run(ctx context.Context, output chan AsyncTask) error
	Type() int
}

func Tasks(tasks ...Task) []Task {
	return tasks
}

type AskPeersTask struct {
	type_ int
	d     time.Duration
}

func NewAskPeersTask(d time.Duration) AskPeersTask {
	return AskPeersTask{
		type_: AskPeers,
		d:     d,
	}
}

func (a AskPeersTask) Type() int {
	return a.type_
}

func (a AskPeersTask) Run(ctx context.Context, output chan AsyncTask) error {
	<-time.After(5 * time.Second)
	SendAsyncTask(output, AsyncTask{
		TaskType: a.type_,
	})

	in := time.NewTicker(a.d)
	defer in.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-in.C:
			SendAsyncTask(output, AsyncTask{
				TaskType: a.type_,
			})
		}
	}
}

type PingTask struct {
}

func NewPingTask() Task {
	return PingTask{}
}

func (PingTask) Type() int {
	return Ping
}

func (PingTask) Run(ctx context.Context, output chan AsyncTask) error {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			SendAsyncTask(output, AsyncTask{
				TaskType: Ping,
				Data:     nil,
			})
		}
	}
}

type MineMicroTaskData struct {
	Block   *proto.Block
	Limits  proto.MiningLimits
	KeyPair proto.KeyPair
	Vrf     []byte
}

func (MineMicroTaskData) taskDataMarker() {}

type MineMicroTask struct {
	timeout           time.Duration
	MineMicroTaskData MineMicroTaskData
}

func NewMineMicroTask(timeout time.Duration, block *proto.Block, limits proto.MiningLimits, keyPair proto.KeyPair, vrf []byte) MineMicroTask {
	if block == nil {
		panic("NewMineMicroTask block is nil")
	}
	return MineMicroTask{
		timeout: timeout,
		MineMicroTaskData: MineMicroTaskData{
			Block:   block,
			Limits:  limits,
			KeyPair: keyPair,
			Vrf:     vrf,
		},
	}
}

func (MineMicroTask) Type() int {
	return MineMicro
}

func (a MineMicroTask) Run(ctx context.Context, output chan AsyncTask) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(a.timeout):
		SendAsyncTask(output, AsyncTask{
			TaskType: a.Type(),
			Data:     a.MineMicroTaskData,
		})
	}
	return nil
}

type funcTask struct {
	f     func(ctx context.Context, output chan AsyncTask) error
	_type int
}

func (a funcTask) Run(ctx context.Context, output chan AsyncTask) error {
	return a.f(ctx, output)
}

func (a funcTask) Type() int {
	return a._type
}

func NewFuncTask(f func(ctx context.Context, output chan AsyncTask) error, taskType int) Task {
	return funcTask{
		f:     f,
		_type: taskType,
	}
}
