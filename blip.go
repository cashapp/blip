package blip

const VERSION = "0.0.0"

var SHA = ""

// Config represents the Blip startup configuration.
type Config struct {
}

func DefaultConfig() Config {
	return Config{
		// Default values
	}
}
