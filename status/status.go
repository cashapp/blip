package status

import (
	"fmt"
	"sync"
)

type status struct {
	*sync.Mutex
	blip     map[string]string
	monitors map[string]map[string]string
}

var s = &status{
	Mutex:    &sync.Mutex{},
	blip:     map[string]string{},
	monitors: map[string]map[string]string{},
}

func Blip(component, msg string, args ...interface{}) {
	s.Lock()
	s.blip[component] = fmt.Sprintf(msg, args...)
	s.Unlock()
}

func Monitor(monitorId, component, msg string, args ...interface{}) {
	s.Lock()
	if _, ok := s.monitors[monitorId]; !ok {
		s.monitors[monitorId] = map[string]string{}
	}
	s.monitors[monitorId][component] = fmt.Sprintf(msg, args...)
	s.Unlock()
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
	s.Unlock()
}
