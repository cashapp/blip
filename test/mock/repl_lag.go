// Copyright 2022 Block, Inc.

package mock

import (
	"time"
)

type LagWaiter struct {
	WaitFunc func(now, then time.Time, f int, srcId string) (int64, time.Duration)
}

func (w LagWaiter) Wait(now, then time.Time, f int, srcId string) (int64, time.Duration) {
	if w.WaitFunc != nil {
		return w.WaitFunc(now, then, f, srcId)
	}
	return 0, 0
}
