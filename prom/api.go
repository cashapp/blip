package prom

// API listens on a unique port and responds to GET /metrics for one exporter.
type API struct {
	Port     uint
	Exporter *Exporter
}
