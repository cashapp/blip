package heartbeat

import (
	"time"

	"github.com/square/blip"
)

type LagWaiter interface {
	Wait(now, then time.Time, f int) (int64, time.Duration)
}

type SlowFastWaiter struct {
	waits int
}

var _ LagWaiter = &SlowFastWaiter{}

var offset = time.Duration(50 * time.Millisecond)

func NewSlowFastWaiter() *SlowFastWaiter {
	return &SlowFastWaiter{
		waits: 0,
	}
}

func (w *SlowFastWaiter) Wait(now, then time.Time, freq int) (int64, time.Duration) {
	next := then.Add(time.Duration(freq) * time.Millisecond)
	blip.Debug("then=%s  now=%s  next=%s", then, now, next)

	if now.Before(next) {
		w.waits = 0

		// Wait until next hb
		d := next.Sub(now) + offset
		if d < 0 {
			d = offset
		}
		blip.Debug("CURRENT: %s after, - wait %s", now.Sub(then), d)
		return 0, d
	}

	var waitTime time.Duration
	w.waits += 1
	switch {
	case w.waits <= 3:
		waitTime = time.Duration(50 * time.Millisecond)
		break
	case w.waits <= 6:
		waitTime = time.Duration(100 * time.Millisecond)
		break
	case w.waits <= 9:
		waitTime = time.Duration(200 * time.Millisecond)
		break
	default:
		waitTime = time.Duration(500 * time.Millisecond)
	}

	// Next hb is late (lagging)
	blip.Debug("lagging: %s past ETA, wait %s (%d)", now.Sub(next), waitTime, w.waits)
	return now.Sub(next).Milliseconds(), waitTime
}
