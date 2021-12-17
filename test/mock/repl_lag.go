package mock

import (
	"time"
)

type LagWaiter struct {
	WaitFunc func(now, then time.Time, f int) (int64, time.Duration)
}

func (w LagWaiter) Wait(now, then time.Time, f int) (int64, time.Duration) {
	if w.WaitFunc != nil {
		return w.WaitFunc(now, then, f)
	}
	return 0, 0
}
