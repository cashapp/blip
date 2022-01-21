// Copyright 2022 Block, Inc.

package ha

type Manager interface {
	Standby() bool
}

type disabled struct{}

var _ Manager = disabled{}

func (d disabled) Standby() bool {
	return false
}

var Disabled = disabled{}
