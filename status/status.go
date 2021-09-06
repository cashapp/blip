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

func Report(blip bool, monitorId string) map[string]map[string]string {
	s.Lock()
	defer s.Unlock()
	status := map[string]map[string]string{}
	if blip {
		status["blip"] = map[string]string{}
		for k, v := range s.blip {
			status["blip"][k] = v
		}
	}
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
