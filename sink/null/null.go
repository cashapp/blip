package null

// Sink discards metrics.
type Sink struct {
}

func NewSink() *Sink {
	return &Sink{}
}

func (s *Sink) Send() error {
	return nil
}

func (s *Sink) Status() error {
	return nil
}
