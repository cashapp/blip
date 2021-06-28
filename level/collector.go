package level

// Collector calls a monitor to collect metrics according to a plan.
type Collector interface {
	// ChangePlan changes the plan; it's called an Adjuster.
	ChangePlan() error
}
