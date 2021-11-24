package monitor

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/ha"
	"github.com/square/blip/proto"
	"github.com/square/blip/status"
)

var Now func() time.Time = time.Now

// LevelAdjuster changes the plan based on database instance state.
type LevelAdjuster interface {
	Run(stopChan, doneChan chan struct{}) error

	Status() proto.MonitorAdjusterStatus
}

type LevelAdjusterArgs struct {
	MonitorId string
	Config    blip.ConfigPlanAdjuster
	DB        *sql.DB
	LPC       LevelCollector
	HA        ha.Manager
}

var _ LevelAdjuster = &adjuster{}

type state struct {
	state string
	plan  string
	ts    time.Time
}

type change struct {
	after time.Duration
	plan  string
}

// adjuster is the implementation of LevelAdjuster.
type adjuster struct {
	cfg       blip.ConfigPlanAdjuster
	monitorId string
	db        *sql.DB
	lpc       LevelCollector
	ha        ha.Manager
	// --
	*sync.Mutex
	states  map[string]change
	prev    state
	curr    state
	pending state
	first   bool
	event   event.MonitorSink
	retry   *backoff.ExponentialBackOff
	lerr    error
}

func NewLevelAdjuster(args LevelAdjusterArgs) *adjuster {
	states := map[string]change{}
	d, _ := time.ParseDuration(args.Config.Offline.After)
	states[blip.STATE_OFFLINE] = change{
		after: d,
		plan:  args.Config.Offline.Plan,
	}
	d, _ = time.ParseDuration(args.Config.Standby.After)
	states[blip.STATE_STANDBY] = change{
		after: d,
		plan:  args.Config.Standby.Plan,
	}
	d, _ = time.ParseDuration(args.Config.ReadOnly.After)
	states[blip.STATE_READ_ONLY] = change{
		after: d,
		plan:  args.Config.ReadOnly.Plan,
	}
	d, _ = time.ParseDuration(args.Config.Active.After)
	states[blip.STATE_ACTIVE] = change{
		after: d,
		plan:  args.Config.Active.Plan,
	}

	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 0

	return &adjuster{
		monitorId: args.MonitorId,
		cfg:       args.Config,
		db:        args.DB,
		lpc:       args.LPC,
		ha:        args.HA,
		// --
		Mutex:   &sync.Mutex{},
		states:  states,
		prev:    state{},
		curr:    state{state: blip.STATE_OFFLINE},
		pending: state{},
		first:   true,
		event:   event.MonitorSink{MonitorId: args.MonitorId},
		retry:   retry,
	}
}

// setErr sets the last internal error reported by Status.
func (a *adjuster) setErr(err error) {
	a.Lock()
	a.lerr = err
	a.Unlock()
}

// Status returns internal LevelAdjuster status. It's called from Monitor.Status
// in response to GET /status/monitor/internal?id=monitorId.
func (a *adjuster) Status() proto.MonitorAdjusterStatus {
	a.Lock()
	defer a.Unlock()

	status := proto.MonitorAdjusterStatus{
		CurrentState: proto.MonitorState{
			State: a.curr.state,
			Plan:  a.curr.plan,
			Since: a.curr.ts.Format(time.RFC3339),
		},
		PreviousState: proto.MonitorState{
			State: a.prev.state,
			Plan:  a.prev.plan,
			Since: a.prev.ts.Format(time.RFC3339),
		},
		PendingState: proto.MonitorState{
			State: a.pending.state,
			Plan:  a.pending.plan,
			Since: a.pending.ts.Format(time.RFC3339),
		},
	}

	if a.lerr != nil {
		status.Error = a.lerr.Error()
	}

	return status
}

// Run calls CheckState every second; or, if offline, it uses an exponential
// backoff up to 60 seconds until back online (MySQL connection ok). There is
// no logic in this function; it's just a timed loop to call CheckState. It's
// run as a goroutine from Monitor.Run only if config.monitors.plans.adjust
// is enabled (blip.ConfigPlanAdjuster.Enabled returns true).
func (a *adjuster) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	defer status.Monitor(a.monitorId, "lpa", "not running")

	status.Monitor(a.monitorId, "lpa", "running")

	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		a.CheckState()

		if a.curr.state != blip.STATE_OFFLINE {
			time.Sleep(1 * time.Second)
		} else {
			time.Sleep(a.retry.NextBackOff())
		}
	}
}

// CheckState checks the current state and changes state when necessary.
// This is the main logic, called periodically by Run.
func (a *adjuster) CheckState() {
	now := Now()
	obsv := a.state()

	a.Lock()
	defer a.Unlock()

	defer func() {
		if a.pending.state == "" {
			status.Monitor(a.monitorId, "lpa", "%s", a.curr.state)
		} else {
			status.Monitor(a.monitorId, "lpa", "%s -> %s", a.curr.state, a.pending.state)
		}
	}()

	if obsv == a.curr.state {
		if !a.pending.ts.IsZero() {
			// changed back to current state
			a.pending.ts = time.Time{}
			a.pending.state = blip.STATE_NONE
			a.event.Sendf(event.STATE_CHANGE_ABORT, "%s", obsv)
		}
	} else if obsv == a.pending.state {
		// Still in the pending state; is it time to change?
		if now.Sub(a.pending.ts) < a.states[a.pending.state].after {
			return
		}

		// Change state via LPC: current -> pending
		if err := a.lpcChangePlan(a.pending.state, a.pending.plan); err != nil {
			a.setErr(err)
			blip.Debug(err.Error())
			return // ok to ignore error; see comments on lpcChangePlan
		}

		a.prev = a.curr

		a.curr = a.pending

		a.pending.ts = time.Time{}
		a.pending.state = blip.STATE_NONE
		blip.Debug("%s: LPA state changed to %s", a.monitorId, obsv)
		a.event.Sendf(event.STATE_CHANGE_END, "%s", obsv)
	} else if a.first && a.curr.state == blip.STATE_OFFLINE {
		a.first = false

		// Change state via LPC
		if err := a.lpcChangePlan(obsv, a.states[obsv].plan); err != nil {
			a.setErr(err)
			blip.Debug(err.Error())
			return // ok to ignore error; see comments on lpcChangePlan
		}

		a.prev = a.curr
		a.curr = state{
			state: obsv,
			ts:    now,
		}
		blip.Debug("%s: LPA start in state %s", a.monitorId, obsv)
		a.event.Sendf(event.STATE_CHANGE_END, "%s", obsv)
	} else {
		// State change
		a.pending.state = obsv
		a.pending.ts = now
		a.pending.plan = a.states[obsv].plan
		blip.Debug("%s: LPA state changed to %s, waiting %s", a.monitorId, obsv, a.states[obsv].after)
		a.event.Sendf(event.STATE_CHANGE_BEGIN, "%s", obsv)

		a.retry.Reset()
	}
}

// lpcChangePlan calls LevelCollector.ChangePlan to change the metrics collection plan.
// Or, it calls LevelCollector.Pause if there is no plan, which is the usual case when
// offline (can't connect to MySQL. We presume that these calls do not fail; see
// LevelCollector.ChangePlan for details.
func (a *adjuster) lpcChangePlan(state, planName string) error {
	status.Monitor(a.monitorId, "lpa", "calling LPC.ChangePlan: %s %s", state, planName)
	if planName == "" {
		a.lpc.Pause()
		return nil
	}
	return a.lpc.ChangePlan(state, planName)
}

const readOnlyQuery = "SELECT @@read_only, @@super_read_only"

// state queries MySQL to ascertain the HA and read-only state.
func (a *adjuster) state() string {
	status.Monitor(a.monitorId, "lpa", "checking HA standby")
	if a.ha.Standby() {
		return blip.STATE_STANDBY
	}

	// Active, but is MySQL read-only?
	status.Monitor(a.monitorId, "lpa", "checking MySQL read-only")

	var ro, sro int
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := a.db.QueryRowContext(ctx, readOnlyQuery).Scan(&ro, &sro)
	cancel()
	a.setErr(err)
	if err != nil {
		blip.Debug(err.Error())
		return blip.STATE_OFFLINE
	}

	if ro == 1 {
		return blip.STATE_READ_ONLY
	}

	return blip.STATE_ACTIVE
}
