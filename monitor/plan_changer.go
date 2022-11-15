// Copyright 2022 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/ha"
	"github.com/cashapp/blip/proto"
	"github.com/cashapp/blip/status"
)

var Now func() time.Time = time.Now

// PlanChanger changes the plan based on database instance state.
type PlanChanger interface {
	Run(stopChan, doneChan chan struct{}) error

	Status() proto.MonitorAdjusterStatus
}

type PlanChangerArgs struct {
	MonitorId string
	Config    blip.ConfigPlanChange
	DB        *sql.DB
	LCO       LevelCollector
	HA        ha.Manager
}

var _ PlanChanger = &planChanger{}

type state struct {
	state string
	plan  string
	ts    time.Time
}

type change struct {
	after time.Duration
	plan  string
}

// planChanger is the implementation of PlanChanger.
type planChanger struct {
	cfg       blip.ConfigPlanChange
	monitorId string
	db        *sql.DB
	lco       LevelCollector
	ha        ha.Manager
	// --
	*sync.Mutex
	states  map[string]change
	prev    state
	curr    state
	pending state
	first   bool
	event   event.MonitorReceiver
	retry   *backoff.ExponentialBackOff
	lerr    error
}

func NewPlanChanger(args PlanChangerArgs) *planChanger {
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

	return &planChanger{
		monitorId: args.MonitorId,
		cfg:       args.Config,
		db:        args.DB,
		lco:       args.LCO,
		ha:        args.HA,
		// --
		Mutex:   &sync.Mutex{},
		states:  states,
		prev:    state{},
		curr:    state{state: blip.STATE_OFFLINE},
		pending: state{},
		first:   true,
		event:   event.MonitorReceiver{MonitorId: args.MonitorId},
		retry:   retry,
	}
}

// setErr sets the last internal error reported by Status.
func (pch *planChanger) setErr(err error) {
	pch.Lock()
	pch.lerr = err
	pch.Unlock()
}

// Status returns internal PlanChanger status. It's called from Monitor.Status
// in response to GET /status/monitor/internal?id=monitorId.
func (pch *planChanger) Status() proto.MonitorAdjusterStatus {
	pch.Lock()
	defer pch.Unlock()

	status := proto.MonitorAdjusterStatus{
		CurrentState: proto.MonitorState{
			State: pch.curr.state,
			Plan:  pch.curr.plan,
			Since: pch.curr.ts.Format(time.RFC3339),
		},
		PreviousState: proto.MonitorState{
			State: pch.prev.state,
			Plan:  pch.prev.plan,
			Since: pch.prev.ts.Format(time.RFC3339),
		},
		PendingState: proto.MonitorState{
			State: pch.pending.state,
			Plan:  pch.pending.plan,
			Since: pch.pending.ts.Format(time.RFC3339),
		},
	}

	if pch.lerr != nil {
		status.Error = pch.lerr.Error()
	}

	return status
}

// Run calls CheckState every second; or, if offline, it uses an exponential
// backoff up to 60 seconds until back online (MySQL connection ok). There is
// no logic in this function; it's just a timed loop to call CheckState. It's
// run as a goroutine from Monitor.Run only if config.monitors.plans.adjust
// is enabled (blip.ConfigPlanChange.Enabled returns true).
func (pch *planChanger) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	defer status.Monitor(pch.monitorId, "lpa", "not running")

	status.Monitor(pch.monitorId, "lpa", "running")

	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		pch.CheckState()

		if pch.curr.state != blip.STATE_OFFLINE {
			time.Sleep(1 * time.Second)
		} else {
			time.Sleep(pch.retry.NextBackOff())
		}
	}
}

// CheckState checks the current state and changes state when necessary.
// This is the main logic, called periodically by Run.
func (pch *planChanger) CheckState() {
	now := Now()
	obsv := pch.state()

	pch.Lock()
	defer pch.Unlock()

	defer func() {
		if pch.pending.state == "" {
			status.Monitor(pch.monitorId, "lpa", "%s", pch.curr.state)
		} else {
			status.Monitor(pch.monitorId, "lpa", "%s -> %s", pch.curr.state, pch.pending.state)
		}
	}()

	if obsv == pch.curr.state {
		if !pch.pending.ts.IsZero() {
			// changed back to current state
			pch.pending.ts = time.Time{}
			pch.pending.state = blip.STATE_NONE
			pch.event.Sendf(event.STATE_CHANGE_ABORT, "%s", obsv)
		}
	} else if obsv == pch.pending.state {
		// Still in the pending state; is it time to change?
		if now.Sub(pch.pending.ts) < pch.states[pch.pending.state].after {
			return
		}

		// Change state via LPC: current -> pending
		if err := pch.lcoChangePlan(pch.pending.state, pch.pending.plan); err != nil {
			pch.setErr(err)
			blip.Debug(err.Error())
			return // ok to ignore error; see comments on lcoChangePlan
		}

		pch.prev = pch.curr

		pch.curr = pch.pending

		pch.pending.ts = time.Time{}
		pch.pending.state = blip.STATE_NONE
		blip.Debug("%s: LPA state changed to %s", pch.monitorId, obsv)
		pch.event.Sendf(event.STATE_CHANGE_END, "%s", obsv)
	} else if pch.first && pch.curr.state == blip.STATE_OFFLINE {
		pch.first = false

		// Change state via LPC
		if err := pch.lcoChangePlan(obsv, pch.states[obsv].plan); err != nil {
			pch.setErr(err)
			blip.Debug(err.Error())
			return // ok to ignore error; see comments on lcoChangePlan
		}

		pch.prev = pch.curr
		pch.curr = state{
			state: obsv,
			ts:    now,
		}
		blip.Debug("%s: LPA start in state %s", pch.monitorId, obsv)
		pch.event.Sendf(event.STATE_CHANGE_END, "%s", obsv)
	} else {
		// State change
		pch.pending.state = obsv
		pch.pending.ts = now
		pch.pending.plan = pch.states[obsv].plan
		blip.Debug("%s: LPA state changed to %s, waiting %s", pch.monitorId, obsv, pch.states[obsv].after)
		pch.event.Sendf(event.STATE_CHANGE_BEGIN, "%s", obsv)

		pch.retry.Reset()
	}
}

// lcoChangePlan calls LevelCollector.ChangePlan to change the metrics collection plan.
// Or, it calls LevelCollector.Pause if there is no plan, which is the usual case when
// offline (can't connect to MySQL. We presume that these calls do not fail; see
// LevelCollector.ChangePlan for details.
func (pch *planChanger) lcoChangePlan(state, planName string) error {
	status.Monitor(pch.monitorId, "lpa", "calling LPC.ChangePlan: %s %s", state, planName)
	if planName == "" {
		pch.lco.Pause()
		return nil
	}
	return pch.lco.ChangePlan(state, planName)
}

const readOnlyQuery = "SELECT @@read_only, @@super_read_only"

// state queries MySQL to ascertain the HA and read-only state.
func (pch *planChanger) state() string {
	status.Monitor(pch.monitorId, "lpa", "checking HA standby")
	if pch.ha.Standby() {
		return blip.STATE_STANDBY
	}

	// Active, but is MySQL read-only?
	status.Monitor(pch.monitorId, "lpa", "checking MySQL read-only")

	var ro, sro int
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := pch.db.QueryRowContext(ctx, readOnlyQuery).Scan(&ro, &sro)
	cancel()
	pch.setErr(err)
	if err != nil {
		blip.Debug(err.Error())
		return blip.STATE_OFFLINE
	}

	if ro == 1 {
		return blip.STATE_READ_ONLY
	}

	return blip.STATE_ACTIVE
}
