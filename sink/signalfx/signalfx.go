package signalfx

// Sink sends metrics to SignalFx.
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
