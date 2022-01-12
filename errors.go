package blip

import (
	"fmt"
)

type ErrInvalidDomain struct {
	Domain string
}

func (e ErrInvalidDomain) Error() string {
	return fmt.Sprintf("invalid domain: %s (run `blip --print-domains` to list valid doamins)", e.Domain)
}
