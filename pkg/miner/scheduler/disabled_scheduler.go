package scheduler

import "go.uber.org/zap"

type DisabledScheduler struct {
}

func (d DisabledScheduler) Mine() chan Emit {
	zap.S().Debugf("Calling Mine on disabled Scheduler")
	return nil
}

func (d DisabledScheduler) Emits() []Emit {
	zap.S().Debugf("Calling Emits on disabled Scheduler")
	return nil
}

func (d DisabledScheduler) Reschedule() {
	zap.S().Debugf("Calling Reschedule on disabled Scheduler")
}
