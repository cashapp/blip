// Copyright 2022 Block, Inc.

package sink

import (
	"context"
	"sync"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
)

// Retry is a pseudo-sink that provides buffering, serialization, and retry for
// a real sink. The built-in sinks, except "log", use Retry to handle those three
// complexities.
//
// Retry uses a LIFO queue (a stack) to prioritize sending the latest metrics.
// This means that, during a long outage of the real sink, Retry drops the oldest
// metrics and keeps the latest metrics, up to its buffer size, which is configurable.
//
// Retry sends SINK_SEND_ERROR events on Send error; the real sink should not.
type BufferedRetry struct {
	sink blip.Sink

	sending     bool
	sendTimeout time.Duration
	retryWait   time.Duration

	event event.MonitorReceiver

	stackMux *sync.Mutex
	stack    []*blip.Metrics // LIFO
	max      int
	top      int

	signalChan chan struct{}
}

func NewBufferedRetry(args RetryArgs) *BufferedRetry {
	// Panic if caller doesn't provide required args
	if args.MonitorId == "" {
		panic("RetryArgs.MonitorId is empty string; value required")
	}
	if args.Sink == nil {
		panic("RetryArgs.Sink is nil; value required")
	}

	if _, ok := args.Sink.(*Delta); ok {
		panic("RetryArgs.Sink cannot be a Delta sink.")
	}

	// Set defaults
	if args.BufferSize == 0 {
		args.BufferSize = DEFAULT_RETRY_BUFFER_SIZE
	}
	if args.SendTimeout == 0 {
		args.SendTimeout, _ = time.ParseDuration(DEFAULT_RETRY_SEND_TIMEOUT)
	}
	if args.SendRetryWait == 0 {
		args.SendRetryWait, _ = time.ParseDuration(DEFAULT_RETRY_SEND_RETRY_WAIT)
	}

	rb := &BufferedRetry{
		sink:  args.Sink,
		event: event.MonitorReceiver{MonitorId: args.MonitorId},

		sending:     false,
		sendTimeout: args.SendTimeout,
		retryWait:   args.SendRetryWait,

		stackMux:   &sync.Mutex{},
		stack:      make([]*blip.Metrics, args.BufferSize),
		max:        int(args.BufferSize) - 1,
		top:        -1,
		signalChan: make(chan struct{}),
	}
	go rb.sendBackground()
	blip.Debug("buff %d, send timeout %s", rb.max+1, rb.sendTimeout)
	return rb
}

// Name returns the name of the real sink, not "retry".
func (rb *BufferedRetry) Name() string {
	return rb.sink.Name()
}

// Send buffers, sends, and retries sending metrics on failure. It is safe to call
// from multiple goroutines.
func (rb *BufferedRetry) Send(ctx context.Context, m *blip.Metrics) error {
	// Push metrics to the stop of the stack
	rb.push(m)
	return nil
}

func (rb *BufferedRetry) sendBackground() {
	for {
		<-rb.signalChan
		// Process stack from newest to oldest and continue until
		// the stack is empty
		n := 0
		for next := rb.pop(nil); next != nil; next = rb.pop(next) {

			// Throttle between send, except on first send
			if n > 0 {
				time.Sleep(rb.retryWait)
			}
			n += 1

			// Send next oldest metrics
			if err := rb.sink.Send(context.Background(), next); err != nil {
				rb.event.Errorf(event.SINK_SEND_ERROR, err.Error())
				next = nil // don't pop metrics; retry stack from top down
			}
		}
	}
}

func (rb *BufferedRetry) push(m *blip.Metrics) {
	rb.stackMux.Lock()
	defer rb.stackMux.Unlock()
	if rb.top < rb.max {
		rb.top++
	} else {
		// Push down stack (push off oldest metrics)
		copy(rb.stack, rb.stack[1:])
	}
	rb.stack[rb.top] = m

	if rb.top == 0 {
		rb.signalChan <- struct{}{}
	}
}

func (rb *BufferedRetry) pop(sent *blip.Metrics) *blip.Metrics {
	rb.stackMux.Lock()
	defer rb.stackMux.Unlock()

	// Remove sent metrics from the stack
	if sent != nil {
		if rb.stack[rb.top] == sent {
			// Easy case: sent is still on top, so just dereference to free memory
			rb.stack[rb.top] = nil
			rb.top--
		} else {
			// Metrics were on top but got pushed down, so remove metrics from
			// middle of stack
			k := -1 // index of sent in stack
			for i := range rb.stack {
				if rb.stack[i] != sent {
					continue
				}
				k = i // found sent in stack
				break
			}
			if k > -1 {
				copy(rb.stack[k:], rb.stack[k+1:]) // remove sent for stack
				rb.top--
			}
			// If k still equals -1, then sent was push off the stack,
			// so we can ignore
		}
	}

	// Stack empty? Nothing to pop.
	if rb.top == -1 {
		return nil
	}

	// Return next oldest metrics
	return rb.stack[rb.top]
}
