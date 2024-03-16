// Copyright 2024 Block, Inc.

// Package event provides a simple event stream in lieu of standard logging.
package event

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cashapp/blip"
)

// Event is something that happened in Blip. Events replace traditional logging.
// All parts of Blip send detailed events about what's happening.
type Event struct {
	Ts        time.Time
	Event     string
	MonitorId string
	Message   string
	Error     bool
}

// A Receiver sends events to a destination. Use Tee to send events to multiple destinations.
// Implementations must be non-blocking; callers expect this.
type Receiver interface {
	// Recv receives one event asynchronously. It must not block.
	// A specific implementation determines what is done with the event: logged,
	// sent to Slack, emitted to a pseudo metric, and so on.
	Recv(Event)
}

// SetReceiver sets the receiver used by Blip to handle events. The default
// receiver is Log. To override the default, call this function to set a new
// receiver before calling Server.Boot.
func SetReceiver(r Receiver, override bool) {
	// Don't override becuase it's set only once. If user calls before
	// Server.Start, then we keep their receiver. Else, Server.Start calls
	// to set a built-in Log receiver.
	if receiver != nil && !override {
		return
	}
	receiver = r
}

// receiver is the private package Receiver that the public packages below use.
// It defaults to a Log type receiver (set by Server.Start), but users can call
// SetReceiver (before Server.Start) to override.
var receiver Receiver = Log{}

var subscribers = []Receiver{}
var submux = &sync.Mutex{}

func Subscribe(r Receiver) {
	submux.Lock()
	subscribers = append(subscribers, r)
	submux.Unlock()
}

func RemoveSubscribers() {
	submux.Lock()
	subscribers = []Receiver{}
	submux.Unlock()
}

// Send sends an event with no additional message.
// This is a convenience function for Sendf.
// Non-monitor parts of Blip use this function.
func Send(eventName string) {
	send(Event{Ts: time.Now(), Event: eventName})
}

// Sendf sends an event and formatted message.
// Non-monitor parts of Blip use this function.
func Sendf(eventName string, msg string, args ...interface{}) {
	send(Event{
		Ts:      time.Now(),
		Event:   eventName,
		Message: fmt.Sprintf(msg, args...),
	})
}

// Errorf sends an event flagged as an error with a formatted message.
func Errorf(eventName string, msg string, args ...interface{}) {
	send(Event{
		Ts:      time.Now(),
		Event:   eventName,
		Message: fmt.Sprintf(msg, args...),
		Error:   true,
	})
}

func send(e Event) {
	receiver.Recv(e)
	submux.Lock()
	for _, s := range subscribers {
		s.Recv(e)
	}
	submux.Unlock()
}

// --------------------------------------------------------------------------

// MonitorReceiver is a Receiver bound to a single monitor. Monitors use this
// type to send events with the monitor ID.
type MonitorReceiver struct {
	MonitorId string
}

var _ Receiver = MonitorReceiver{}

func (s MonitorReceiver) Recv(e Event) {
	send(e)
}

// Send sends an event with no additional message from the monitor.
// This is a convenience function for Sendf.
func (s MonitorReceiver) Send(eventName string) {
	send(Event{Ts: time.Now(), Event: eventName, MonitorId: s.MonitorId})
}

// Sendf sends an event and formatted message from the monitor.
func (s MonitorReceiver) Sendf(eventName string, msg string, args ...interface{}) {
	send(Event{
		Ts:        time.Now(),
		Event:     eventName,
		Message:   fmt.Sprintf(msg, args...),
		MonitorId: s.MonitorId,
	})
}

func (s MonitorReceiver) Errorf(eventName string, msg string, args ...interface{}) {
	send(Event{
		Ts:        time.Now(),
		Event:     eventName,
		Message:   fmt.Sprintf(msg, args...),
		MonitorId: s.MonitorId,
		Error:     true,
	})
}

// --------------------------------------------------------------------------

var stdout = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
var stderr = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

// Log is the default Receiver that uses the Go built-in log package to print
// certain events to STDOUT and error events to STDERR. Call SetReceiver to
// override this default.
type Log struct {
	All bool
}

func (s Log) Recv(e Event) {
	// Always print error events to STDERR
	if e.Error {
		stderr.Printf("[%-25s] [%s] ERROR: %s", e.Event, e.MonitorId, e.Message)
		return
	}

	// Log all events? If true, then log to stdout.
	if s.All {
		stdout.Printf("[%-25s] [%s] %s", e.Event, e.MonitorId, e.Message)
		return
	}

	// If debugging, print all events
	if blip.Debugging {
		stdout.Printf("[%-25s] [%s] %s", e.Event, e.MonitorId, e.Message)
		return
	}
}

// --------------------------------------------------------------------------

// Tee connects multiple Receiver, like the Unix tee command. It implements
// Receiver. On Tee.Recv, it copies the event to a real receiver: Tee.Receiver.
// Then it copies the event to Tee.Out, if Out is not nil.  To "pipe fit"
// multiple Tee together, use another Tee for Out.
//
//	  event --> Tee.Recv --> Tee.Out.Recv // second
//				   |
//	            +-> Tee.Receiver.Recv // first
type Tee struct {
	Receiver Receiver
	Out      Receiver
}

func (t Tee) Recv(e Event) {
	t.Receiver.Recv(e)
	if t.Out != nil {
		t.Out.Recv(e)
	}
}
