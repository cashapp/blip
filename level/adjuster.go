package level

import (
	"sync"

	"github.com/square/blip/monitor"
)

// Adjuster changes the plan based on database instance state.
type Adjuster interface {
	Run(stopChan, doneChan chan struct{}) error
	Stop() error
}

var _ Adjuster = &adjuster{}

// adjuster is the implementation of Adjuster.
type adjuster struct {
	monitor   *monitor.Monitor
	metronome *sync.Cond
	lpc       Collector
}

func NewAdjuster(mon *monitor.Monitor, metronome *sync.Cond, lpc Collector) *adjuster {
	return &adjuster{
		monitor:   mon,
		metronome: metronome,
		lpc:       lpc,
	}
}

func (a *adjuster) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	return nil
}

func (a *adjuster) Stop() error {
	return nil
}
