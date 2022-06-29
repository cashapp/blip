// Copyright 2022 Block, Inc.

package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/cashapp/blip"
)

// Sink logs metrics.
type logSink struct {
	monitorId string
}

func NewLogSink(monitorId string) (logSink, error) {
	return logSink{monitorId: monitorId}, nil
}

func (s logSink) Send(ctx context.Context, m *blip.Metrics) error {
	fmt.Printf("# monitor:  %s\n", m.MonitorId)
	fmt.Printf("# plan:     %s\n", m.Plan)
	fmt.Printf("# level:    %s\n", m.Level)
	fmt.Printf("# ts:       %s\n", m.Begin.Format(time.RFC3339Nano))
	fmt.Printf("# duration: %d ms\n", m.End.Sub(m.Begin).Milliseconds())
	for domain, values := range m.Values {
		for i := range values {
			fmt.Printf("%s.%s = %d\n", domain, values[i].Name, int64(values[i].Value))
		}
	}
	fmt.Println()
	return nil
}

func (s logSink) Status() string {
	return "swimmingly"
}

func (s logSink) Name() string {
	return "log"
}
