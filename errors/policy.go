package errors

import (
	"strings"
)

const (
	POLICY_REPORT_IGNORE = "ignore"
	POLICY_REPORT_EVERY  = "report"
	POLICY_REPORT_ONCE   = "report-once"

	POLICY_METRIC_DROP = "drop"
	POLICY_METRIC_ZERO = "zero"

	POLICY_RETRY_YES = "retry"
	POLICY_RETRY_NO  = "stop"
)

type Policy struct {
	Report string
	Metric string
	Retry  string
	n      uint
}

func NewPolicy(s string) *Policy {
	p := &Policy{ // default policy:
		Report: POLICY_REPORT_EVERY, // report the error,
		Metric: POLICY_METRIC_DROP,  // drop the metric,
		Retry:  POLICY_RETRY_YES,    // and keep trying
	}
	for _, ap := range strings.Split(s, ",") {
		switch ap {
		case POLICY_REPORT_IGNORE, POLICY_REPORT_EVERY, POLICY_REPORT_ONCE:
			p.Report = ap
		case POLICY_RETRY_YES, POLICY_RETRY_NO:
			p.Retry = ap
		case POLICY_METRIC_DROP, POLICY_METRIC_ZERO:
			p.Metric = ap
		}
	}
	return p
}

func (p Policy) String() string {
	return p.Report + "," + p.Metric + "," + p.Retry
}

func (p *Policy) ReportError() bool {
	p.n++
	return p.Report == POLICY_REPORT_EVERY || (p.Report == POLICY_REPORT_ONCE && p.n == 1)
}
