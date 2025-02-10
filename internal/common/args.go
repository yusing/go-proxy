package common

const (
	CommandStart              = ""
	CommandSetup              = "setup"
	CommandAddAgent           = "add-agent"
	CommandValidate           = "validate"
	CommandListConfigs        = "ls-config"
	CommandListRoutes         = "ls-routes"
	CommandListIcons          = "ls-icons"
	CommandReload             = "reload"
	CommandDebugListEntries   = "debug-ls-entries"
	CommandDebugListProviders = "debug-ls-providers"
	CommandDebugListMTrace    = "debug-ls-mtrace"
)

type MainServerCommandValidator struct{}

func (v MainServerCommandValidator) IsCommandValid(cmd string) bool {
	switch cmd {
	case CommandStart,
		CommandSetup,
		CommandAddAgent,
		CommandValidate,
		CommandListConfigs,
		CommandListRoutes,
		CommandListIcons,
		CommandReload,
		CommandDebugListEntries,
		CommandDebugListProviders,
		CommandDebugListMTrace:
		return true
	}
	return false
}
