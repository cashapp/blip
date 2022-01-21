// Copyright 2022 Block, Inc.

// Package status provides real-time instantaneous status of every Blip component.
// The only caller is server.API via GET /status.
package status

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

type status struct {
	*sync.Mutex
	blip     map[string]string
	monitors map[string]map[string]string
	counters map[string]map[string]*uint64
}

var s = &status{
	Mutex:    &sync.Mutex{},
	blip:     map[string]string{},
	monitors: map[string]map[string]string{},  // monitorId => component
	counters: map[string]map[string]*uint64{}, // monitorId => component
}

func Blip(component, msg string, args ...interface{}) {
	s.Lock()
	s.blip[component] = fmt.Sprintf(msg, args...)
	s.Unlock()
}

func Monitor(monitorId, component string, msg string, args ...interface{}) {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.monitors[monitorId]; !ok {
		s.monitors[monitorId] = map[string]string{}
		s.counters[monitorId] = map[string]*uint64{}
	}
	s.monitors[monitorId][component] = fmt.Sprintf(msg, args...)
}

func MonitorMulti(monitorId, component, msg string, args ...interface{}) string {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.monitors[monitorId]; !ok {
		s.monitors[monitorId] = map[string]string{}
		s.counters[monitorId] = map[string]*uint64{}
	}
	var uniqComponent string
	n, ok := s.counters[monitorId][component]
	if ok {
		// 2nd+ instance of this component; increase the counter
		uniqComponent = component + "(" + strconv.FormatUint(atomic.AddUint64(n, 1), 10) + ")"
	} else {
		// 1st instance of this component; init the counter
		var first uint64 = 1
		s.counters[monitorId][component] = &first
		uniqComponent = component + "(1)"
	}
	s.monitors[monitorId][uniqComponent] = fmt.Sprintf(msg, args...)
	return uniqComponent
}

func RemoveComponent(monitorId, component string) {
	s.Lock()
	m, ok := s.monitors[monitorId]
	if ok {
		delete(m, component)
	}
	s.Unlock()
}

func ReportBlip() map[string]string {
	s.Lock()
	defer s.Unlock()
	status := map[string]string{}
	for k, v := range s.blip {
		status[k] = v
	}
	return status
}

func ReportMonitors(monitorId string) map[string]map[string]string {
	s.Lock()
	defer s.Unlock()
	status := map[string]map[string]string{}
	if monitorId == "*" {
		for k, v := range s.monitors {
			status[k] = v
		}
	} else if monitorId != "" {
		status[monitorId] = s.monitors[monitorId]
	}

	return status
}

func RemoveMonitor(monitorId string) {
	s.Lock()
	delete(s.monitors, monitorId)
	delete(s.counters, monitorId)
	s.Unlock()
}
