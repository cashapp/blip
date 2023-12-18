package ha

import (
	"fmt"
	"github.com/cashapp/blip"
	"sync"
)

type HAManagerFactory interface {
	Make(monitor blip.ConfigMonitor) (Manager, error)
}

type defaultFactory struct {
}

var df = &defaultFactory{}

func Register(f HAManagerFactory) error {
	hf.Lock()
	defer hf.Unlock()
	// Check if it's different from the default factory
	if hf.ha != df {
		return fmt.Errorf("HA already registered")
	}
	hf.ha = f
	blip.Debug("register HA")
	return nil
}

func Make(args blip.ConfigMonitor) (Manager, error) {
	hf.Lock()
	defer hf.Unlock()
	if hf.ha == nil {
		return nil, fmt.Errorf("HA not registered")
	}
	return hf.ha.Make(args)
}

type haf struct {
	*sync.Mutex
	ha HAManagerFactory
}

var hf = &haf{
	Mutex: &sync.Mutex{},
	ha:    df,
}

func (f *defaultFactory) Make(_ blip.ConfigMonitor) (Manager, error) {
	return Disabled, nil
}
