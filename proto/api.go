package proto

type Status struct {
	Started      string            // ISO timestamp (UTC)
	Uptime       int64             // seconds
	MonitorCount uint              // number of monitors loaded
	Internal     map[string]string // Blip components
	Version      string            // Blip version
}

type Registered struct {
	Collectors []string
	Sinks      []string
}

type MonitorStatus struct {
	MonitorId string
	DSN       string
	Started   bool
	Engine    MonitorEngineStatus    `json:",omitempty"`
	Collector MonitorLevelStatus     `json:",omitempty"`
	Adjuster  *MonitorAdjusterStatus `json:",omitempty"`
	Error     string                 `json:",omitempty"`
}

type MonitorLevelStatus struct {
	State  string
	Plan   string
	Paused bool
	Error  string `json:",omitempty"`
}

type MonitorAdjusterStatus struct {
	CurrentState  MonitorState
	PreviousState MonitorState
	PendingState  MonitorState
	Error         string `json:",omitempty"`
}

type MonitorState struct {
	State string
	Plan  string
	Since string
}

type MonitorEngineStatus struct {
	Plan            string
	Connected       bool
	Error           string            `json:",omitempty"`
	CollectorErrors map[string]string `json:",omitempty"`
	CollectOK       uint              // number of collections all OK (no errors)
	CollectError    uint              // number of collections with at least 1 error
}

type PlanLoaded struct {
	Name   string
	Source string
}
