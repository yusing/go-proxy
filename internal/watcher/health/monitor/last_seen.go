package monitor

import (
	"time"

	F "github.com/yusing/go-proxy/internal/utils/functional"
)

var lastSeenMap = F.NewMapOf[string, time.Time]()

func SetLastSeen(service string, lastSeen time.Time) {
	lastSeenMap.Store(service, lastSeen)
}

func UpdateLastSeen(service string) {
	SetLastSeen(service, time.Now())
}

func GetLastSeen(service string) time.Time {
	lastSeen, _ := lastSeenMap.Load(service)
	return lastSeen
}
