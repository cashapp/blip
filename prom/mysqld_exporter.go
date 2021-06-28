package prom

import ()

// Exporter is a pseudo-sink that emulates mysqld_exporter.
type Exporter struct {
}

func NewSink() *Exporter {
	return &Exporter{}
}

func (s *Exporter) Send() error {
	return nil
}

func (s *Exporter) Status() error {
	return nil
}
