package level

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/ha"
	"github.com/square/blip/status"
)

var Now func() time.Time = time.Now

// Adjuster changes the plan based on database instance state.
type Adjuster interface {
	Run(stopChan, doneChan chan struct{}) error
}

type AdjusterArgs struct {
	MonitorId string
	Config    blip.ConfigPlanAdjuster
	DB        *sql.DB
	Metronome *sync.Cond
	LPC       Collector
	HA        ha.Manager
}

var _ Adjuster = &adjuster{}

type state struct {
	state string
	plan  string
	ts    time.Time
}

type change struct {
	after time.Duration
	plan  string
}

// adjuster is the implementation of Adjuster.
type adjuster struct {
	cfg       blip.ConfigPlanAdjuster
	monitorId string
	db        *sql.DB
	metronome *sync.Cond
	lpc       Collector
	ha        ha.Manager
	// --
	states  map[string]change
	prev    state
	curr    state
	pending state
	first   bool
}

func NewAdjuster(args AdjusterArgs) *adjuster {
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

	return &adjuster{
		monitorId: args.MonitorId,
		cfg:       args.Config,
		db:        args.DB,
		metronome: args.Metronome,
		lpc:       args.LPC,
		ha:        args.HA,
		// --
		states:  states,
		prev:    state{},
		curr:    state{state: blip.STATE_OFFLINE},
		pending: state{},
		first:   true,
	}
}

func (a *adjuster) Run(stopChan, doneChan chan struct{}) error {
	blip.Debug("%s: LPA run", a.monitorId)

	defer close(doneChan)
	defer status.Monitor(a.monitorId, "lpa", "not running")

	status.Monitor(a.monitorId, "lpa", "running")

	n := 1 // 1=whole second tick, -1=half second (500ms) tick

	a.metronome.L.Lock()
	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		// Multiple n by -1 to flip-flop between 1 and -1 to determine
		// if this is a half- or whole-second tick
		a.metronome.Wait() // for tick every 500ms
		n = n * -1
		if n == 1 {
			continue // ignore whole-second ticks
		}

		a.CheckState()
	}
}

func (a *adjuster) CheckState() {
	now := Now()
	obsv := a.state()
	if obsv == a.curr.state {
		if !a.pending.ts.IsZero() {
			// changed back to current state
			a.pending.ts = time.Time{}
			a.pending.state = blip.STATE_NONE
		}
	} else if obsv == a.pending.state {
		// Still in the pending state; is it time to change?
		if now.Sub(a.pending.ts) < a.states[a.pending.state].after {
			return
		}

		// Change state
		if err := a.changePlan(a.pending.state, a.pending.plan); err != nil {
			// @todo
			blip.Debug(err.Error())
		}

		a.prev = a.curr

		a.curr = a.pending

		a.pending.ts = time.Time{}
		a.pending.state = blip.STATE_NONE
		blip.Debug("%s: LPA state changed to %s", a.monitorId, obsv)
	} else if a.first && a.curr.state == blip.STATE_OFFLINE {
		a.first = false
		if err := a.changePlan(obsv, a.states[obsv].plan); err != nil {
			// @todo
			blip.Debug(err.Error())
		}
		a.prev = a.curr
		a.curr = state{
			state: obsv,
			ts:    now,
		}
		blip.Debug("%s: LPA start in state %s", a.monitorId, obsv)
	} else {
		// State change
		a.pending.state = obsv
		a.pending.ts = now
		a.pending.plan = a.states[obsv].plan
		blip.Debug("%s: LPA state changed to %s, waiting %s", a.monitorId, obsv, a.states[obsv].after)
	}
}

func (a *adjuster) changePlan(state, planName string) error {
	if planName == "" {
		return a.lpc.Pause()
	}
	return a.lpc.ChangePlan(state, planName)
}

var readOnlyQuery = "SELECT @@read_only, @@super_read_only"

func (a *adjuster) state() string {
	if a.ha.Standby() {
		return blip.STATE_STANDBY
	}

	// Active, but is MySQL read-only?

	var ro, sro int
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	err := a.db.QueryRowContext(ctx, readOnlyQuery).Scan(&ro, &sro)
	cancel()
	if err != nil {
		blip.Debug(err.Error())
		status.Monitor(a.monitorId, "lpa-error", err.Error())
		return blip.STATE_OFFLINE
	}
	status.Monitor(a.monitorId, "lpa-error", "")

	blip.Debug("ro=%d, sro=%d", ro, sro)
	status.Monitor(a.monitorId, "lpa", "ro=%d, sro=%d", ro, sro)

	if ro == 1 {
		return blip.STATE_READ_ONLY
	}

	return blip.STATE_ACTIVE
}
