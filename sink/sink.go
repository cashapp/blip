package sink

// Sink sends metrics to an external destination.
type Sink interface {
	Send() error
	Status() error
}
