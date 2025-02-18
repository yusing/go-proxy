package common

import (
	"flag"
	"fmt"
	"log"
)

type Args struct {
	Command string
}

const (
	CommandStart              = ""
	CommandValidate           = "validate"
	CommandListConfigs        = "ls-config"
	CommandListRoutes         = "ls-routes"
	CommandListIcons          = "ls-icons"
	CommandReload             = "reload"
	CommandDebugListEntries   = "debug-ls-entries"
	CommandDebugListProviders = "debug-ls-providers"
	CommandDebugListMTrace    = "debug-ls-mtrace"
)

var ValidCommands = []string{
	CommandStart,
	CommandValidate,
	CommandListConfigs,
	CommandListRoutes,
	CommandListIcons,
	CommandReload,
	CommandDebugListEntries,
	CommandDebugListProviders,
	CommandDebugListMTrace,
}

func validateArg(arg string) error {
	for _, v := range ValidCommands {
		if arg == v {
			return nil
		}
	}
	return fmt.Errorf("invalid command %q", arg)
}

func GetArgs() Args {
	var args Args
	flag.Parse()
	args.Command = flag.Arg(0)
	if err := validateArg(args.Command); err != nil {
		log.Fatalf("invalid command: %s", err)
	}
	return args
}
