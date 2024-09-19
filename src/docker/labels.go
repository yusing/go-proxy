package docker

const (
	WildcardAlias = "*"

	LableAliases     = NSProxy + ".aliases"
	LableExclude     = NSProxy + ".exclude"
	LabelIdleTimeout = NSProxy + ".idle_timeout"
	LabelWakeTimeout = NSProxy + ".wake_timeout"
	LabelStopMethod  = NSProxy + ".stop_method"
	LabelStopTimeout = NSProxy + ".stop_timeout"
	LabelStopSignal  = NSProxy + ".stop_signal"
)
