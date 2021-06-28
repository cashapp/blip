package log

import (
	"log"
)

// Sink logs metrics.
type Sink struct {
}

func NewSink() *Sink {
	return &Sink{}
}

func (s *Sink) Send() error {
	log.Println("todo")
	return nil
}

func (s *Sink) Status() error {
	return nil
}
