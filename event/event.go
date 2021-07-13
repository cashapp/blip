// Package event provides a simple event stream in lieu of standard logging.
package event

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Event is something that happened in Blip. Events replace traditional logging.
// All parts of Blip send detailed events about what's happening.
type Event struct {
	Ts        time.Time
	Event     string
	MonitorId string
	Message   string
}

// A Sink sends events to a destination. Use Tee to send events to multiple destinations.
// Implementations must be non-blocking; callers expect this.
type Sink interface {
	// Recv receives one event asynchronously. It must not block.
	// A specific implementation determines what is done with the event: logged,
	// sent to Slack, emitted to a pseudo metric, and so on.
	//
	// eventName should be an EVENT_* const. msg is an optional message, like a
	// log line; and args are optional arguments to interpolate into msg.
	Recv(Event)
}

// Send sends an event with no additional message.
// This is a convenience function for Sendf.
// Non-monitor parts of Blip use this function.
func Send(eventName string) {
	defaultSink.Recv(Event{Ts: time.Now(), Event: eventName})
}

// Sendf sends an event and formatted message.
// Non-monitor parts of Blip use this function.
func Sendf(eventName string, msg string, args ...interface{}) {
	defaultSink.Recv(Event{
		Ts:      time.Now(),
		Event:   eventName,
		Message: fmt.Sprintf(msg, args...),
	})
}

// MonitorSink is an event sink bound to a single monitor. The monitor uses
// this sink as a convenience to ensure that all its events include its monitor ID.
type MonitorSink struct {
	MonitorId string
}

var _ Sink = MonitorSink{}

func (s MonitorSink) Recv(e Event) {
	defaultSink.Recv(e)
}

// Send sends an event with no additional message from the monitor.
// This is a convenience function for Sendf.
func (s MonitorSink) Send(eventName string) {
	defaultSink.Recv(Event{Ts: time.Now(), Event: eventName, MonitorId: s.MonitorId})
}

// Sendf sends an event and formatted message from the monitor.
func (s MonitorSink) Sendf(eventName string, msg string, args ...interface{}) {
	defaultSink.Recv(Event{
		Ts:        time.Now(),
		Event:     eventName,
		Message:   fmt.Sprintf(msg, args...),
		MonitorId: s.MonitorId,
	})
}

// Tee pipes multiple Sink together. Events sent to Tee are sent to Sink then Out.
// To "pipe fit" multiple Tee together, use another Tee for Out.
//
//   event -> Tee -> Stdout
//            |
//            Slack
//
// Tee is like:
//
//   In -+-> Out
//       |
//      Sink
type Tee struct {
	In   Sink
	Out  Sink
	Sink Sink
}

func (t Tee) Recv(e Event) {
	t.Sink.Recv(e)
	t.Out.Recv(e)
}

// --------------------------------------------------------------------------

// SetSink sets the sink used by all Blip code. The default sink prints all
// events to STDOUT (using the Go log package).
//
// Do not call this function directly. To set a user-defined sink, use the
// InitEventSink plugin. The sink is set only once when the server boots.
func SetSink(sink Sink) {
	defaultSink = sink
}

var defaultSink Sink = stdoutSink{}

var stdout = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

// stdoutSink prints events to STDOUT. It's the default event sink.
type stdoutSink struct{}

func (s stdoutSink) Recv(e Event) {
	stdout.Printf("[%s] [%s] %s", e.MonitorId, e.Event, e.Message)
}
