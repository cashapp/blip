package level

// Adjuster changes the plan based on database instance state.
type Adjuster interface {
	Stop() error
}
