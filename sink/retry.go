package sink

import (
	"context"
	"sync"
	"time"

	"github.com/square/blip"
)

type RetryBuffer struct {
	sink        blip.Sink
	sendTimeout time.Duration

	sendMux *sync.Mutex
	sending bool

	stackMux *sync.Mutex
	stack    []*blip.Metrics
	max      int
	top      int
}

func NewRetryBuffer(sink blip.Sink, sendTimeout time.Duration, bufferSize int) *RetryBuffer {
	if bufferSize < 0 {
		bufferSize = 5
	}
	rb := &RetryBuffer{
		sink: sink,
		// --
		sendMux:     &sync.Mutex{},
		sending:     false,
		sendTimeout: sendTimeout,

		stackMux: &sync.Mutex{},
		stack:    make([]*blip.Metrics, bufferSize),
		max:      bufferSize - 1,
		top:      -1,
	}
	blip.Debug("%s: buff %d, send timeout %s", rb.sink.MonitorId(), rb.max+1, rb.sendTimeout)
	return rb
}

func (rb *RetryBuffer) Name() string {
	return rb.sink.Name()
}

func (rb *RetryBuffer) Status() error {
	return nil
}

func (rb *RetryBuffer) MonitorId() string {
	return rb.sink.MonitorId()
}

func (rb *RetryBuffer) Send(ctx context.Context, m *blip.Metrics) error {
	ctx2, cancel := context.WithTimeout(ctx, rb.sendTimeout)
	defer cancel()

	rb.sendMux.Lock()
	if rb.sending {
		rb.push(m) // top of stack
		rb.sendMux.Unlock()
		return nil
	}
	rb.sending = true

	defer func() {
		rb.sending = false
		rb.sendMux.Unlock()
	}()

	if err := rb.sink.Send(ctx2, m); err != nil {
		rb.push(m) // buffer and retry on next call
		blip.Debug("error sending: %s", err)
		return nil
	}

	// Process stack from newest to oldest, while we have time
	rb.retry(ctx2)

	return nil
}

func (rb *RetryBuffer) push(m *blip.Metrics) {
	rb.stackMux.Lock()
	defer rb.stackMux.Unlock()
	if rb.top < rb.max {
		rb.top++
	} else {
		// Push down stack (push off oldest metrics)
		copy(rb.stack, rb.stack[1:])
	}
	rb.stack[rb.top] = m
}

func (rb *RetryBuffer) retry(ctx context.Context) {
STACK:
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		rb.stackMux.Lock()
		if rb.top < 0 {
			rb.stackMux.Unlock()
			return
		}
		m := rb.stack[rb.top]
		rb.stackMux.Unlock()

		if err := rb.sink.Send(ctx, m); err != nil {
			// Leave in stack and retry later, maybe
			// @todo report error?
			continue STACK
		}

		rb.stackMux.Lock()
		if rb.stack[rb.top] != m {
			k := -1
			for i := range rb.stack {
				if rb.stack[i] != m {
					continue
				}
				k = i
				break
			}
			if k == -1 {
				rb.stackMux.Unlock()
				continue STACK
			}
			copy(rb.stack[k:], rb.stack[k+1:])
		}
		rb.stack[rb.top] = nil
		rb.top--
		rb.stackMux.Unlock()
	}
}
