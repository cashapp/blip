// Copyright 2022 Block, Inc.

package proto

import "time"

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

type MonitorLoaderStatus struct {
	MonitorCount  uint
	LastLoaded    time.Time
	LastError     string
	LastErrorTime time.Time
}

type MonitorStatus struct {
	MonitorId string
	DSN       string
	Started   bool
	Engine    MonitorEngineStatus    `json:",omitempty"`
	Collector MonitorCollectorStatus `json:",omitempty"`
	Adjuster  *MonitorAdjusterStatus `json:",omitempty"`
	Error     string                 `json:",omitempty"`
}

type MonitorCollectorStatus struct {
	State              string
	Plan               string
	Paused             bool
	Error              string `json:",omitempty"`
	LastCollectTs      time.Time
	LastCollectError   string            `json:",omitempty"`
	LastCollectErrorTs *time.Time        `json:",omitempty"`
	SinkErrors         map[string]string `json:",omitempty"`
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
	CollectAll      uint64
	CollectSome     uint64
	CollectFail     uint64
}

type PlanLoaded struct {
	Name   string
	Source string
}
