package health

import "encoding/json"

type Status int

const (
	StatusUnknown Status = (iota << 1)

	StatusHealthy
	StatusNapping
	StatusStarting
	StatusUnhealthy
	StatusError

	NumStatuses int = iota - 1

	HealthyMask = StatusHealthy | StatusNapping | StatusStarting
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusNapping:
		return "napping"
	case StatusStarting:
		return "starting"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s Status) Good() bool {
	return s&HealthyMask != 0
}

func (s Status) Bad() bool {
	return s&HealthyMask == 0
}
