package collect

// Plan represents different levels of metrics collection.
type Plan struct {
	Name   string
	Levels map[string]Level
}

// Level is one collection frequency in a plan.
type Level struct {
	Name    string
	Freq    string
	Collect map[string]Domain
}

// Domain is one metric domain for collecting related metrics.
type Domain struct {
	Name    string
	Options map[string]string
	Metrics []string
}

// Metrics are raw metrics from one collector.
type Metrics struct {
	Values map[string]float64
}

// Help represents information about a collector.
type Help struct {
	Domain      string
	Description string
	Options     [][]string // { {"key", "Description of key", "default;val1;val2}, ... }
}
