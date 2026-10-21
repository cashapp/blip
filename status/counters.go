// Copyright 2024 Block, Inc.

package status

import "sync/atomic"

// Monitor counters are cumulative counts of collection faults: conditions
// where a monitor collected late, partially, or dropped metrics. Each counter
// only increases, and it persists across monitor stop and start; it's reset
// only when the monitor is unloaded. A host can read counters with
// ReportCounters (or API endpoint /status/monitors/counters) and forward them
// to its own metrics system. Compare successive values to get a rate.
const (
	// Domains not started before the engine max runtime (EMR) expired.
	ENGINE_DOMAINS_NOT_STARTED = "engine-domains-not-started"

	// Domains started but not returned before the EMR expired. Their metrics
	// are missing from the interval; they're reported late if the domain
	// collector returns before its own max runtime (CMR).
	ENGINE_DOMAINS_NOT_RETURNED = "engine-domains-not-returned"

	// Engine collections (one per interval) that hit the EMR before every
	// domain was started and returned.
	ENGINE_EMR_TIMEOUTS = "engine-emr-timeouts"

	// Domain collectors still running at their next interval because they
	// didn't stop at their CMR (event COLLECTOR_FAULT). The previous interval's
	// metrics from the domain are dropped (event DROP_METRICS_FENCE) if the
	// collector ever returns.
	COLLECTOR_FAULTS = "collector-faults"

	// Domain metrics dropped because the engine collection channel was full
	// (event DROP_METRICS_FLUSH).
	COLLECTOR_FLUSH_DROPS = "collector-flush-drops"

	// Level collector ticks dropped because the previous collection or plan
	// change blocked the level collector loop past the next tick. Each dropped
	// tick delays every later level by one tick.
	LEVEL_TICKS_DROPPED = "level-ticks-dropped"

	// Intervals of metrics dropped because the sink channel was full: sinks are
	// slower than collection (event LCO_METRICS_FAULT).
	LEVEL_METRICS_DROPPED = "level-metrics-dropped"

	// Metrics dropped from a sink retry buffer because it was full: the oldest
	// buffered metrics are pushed off to make room for the latest. The counter
	// name is suffixed with ":" and the sink name, like "sink-retry-dropped:datadog".
	SINK_RETRY_DROPPED = "sink-retry-dropped"
)

// Counter returns the named counter for the monitor, creating it at zero if it
// doesn't exist. Callers should get the counter once (for example, when creating
// a component) and increment it with Add(1): increments are lock-free and do not
// allocate. The returned pointer is valid until RemoveMonitor is called.
func Counter(monitorId, name string) *atomic.Uint64 {
	s.Lock()
	defer s.Unlock()
	m, ok := s.counters[monitorId]
	if !ok {
		m = map[string]*atomic.Uint64{}
		s.counters[monitorId] = m
	}
	c, ok := m[name]
	if !ok {
		c = &atomic.Uint64{}
		m[name] = c
	}
	return c
}

// ReportCounters returns the current value of all monitor counters keyed on
// monitor ID, then counter name. If ids are given, only those monitors are
// reported. Counters are created at zero by the components that use them, so
// a counter can be reported as zero before it has ever incremented.
func ReportCounters(ids ...string) map[string]map[string]uint64 {
	s.Lock()
	defer s.Unlock()
	var allow map[string]bool
	if len(ids) > 0 {
		allow = map[string]bool{}
		for _, id := range ids {
			allow[id] = true
		}
	}
	report := map[string]map[string]uint64{}
	for monitorId, counters := range s.counters {
		if len(allow) > 0 && !allow[monitorId] {
			continue
		}
		report[monitorId] = make(map[string]uint64, len(counters))
		for name, c := range counters {
			report[monitorId][name] = c.Load()
		}
	}
	return report
}
