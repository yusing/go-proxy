package docker

const (
	WildcardAlias = "*"

	NSProxy    = "proxy"
	NSHomePage = "homepage"

	LabelAliases     = NSProxy + ".aliases"
	LabelExclude     = NSProxy + ".exclude"
	LabelIdleTimeout = NSProxy + ".idle_timeout"
	LabelWakeTimeout = NSProxy + ".wake_timeout"
	LabelStopMethod  = NSProxy + ".stop_method"
	LabelStopTimeout = NSProxy + ".stop_timeout"
	LabelStopSignal  = NSProxy + ".stop_signal"
)
