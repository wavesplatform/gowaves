package scheduler

import "log/slog"

type DisabledScheduler struct {
}

func (d DisabledScheduler) Mine() chan Emit {
	slog.Debug("Calling Mine on disabled Scheduler")
	return nil
}

func (d DisabledScheduler) Emits() []Emit {
	slog.Debug("Calling Emits on disabled Scheduler")
	return nil
}

func (d DisabledScheduler) Reschedule() {
	slog.Debug("Calling Reschedule on disabled Scheduler")
}
