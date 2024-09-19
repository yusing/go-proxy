package fields

import (
	"time"

	E "github.com/yusing/go-proxy/error"
)

func ValidateDurationPostitive(value string) (time.Duration, E.NestedError) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, E.Invalid("duration", value)
	}
	if d < 0 {
		return 0, E.Invalid("duration", "negative value")
	}
	return d, nil
}
