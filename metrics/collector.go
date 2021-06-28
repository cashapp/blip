package metrics

// Collector collects metrics for a single metric domain.
type Collector interface {
	Domain() string
	Collect() error
}
