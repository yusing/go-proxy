package common

import (
	"flag"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
)

type Args struct {
	Command string
}

const (
	CommandStart       = ""
	CommandValidate    = "validate"
	CommandListConfigs = "ls-config"
	CommandReload      = "reload"
)

var ValidCommands = []string{CommandStart, CommandValidate, CommandListConfigs, CommandReload}

func GetArgs() Args {
	var args Args
	flag.Parse()
	args.Command = flag.Arg(0)
	if err := validateArgs(args.Command, ValidCommands); err.HasError() {
		logrus.Fatal(err)
	}
	return args
}

func validateArgs[T comparable](arg T, validArgs []T) E.NestedError {
	for _, v := range validArgs {
		if arg == v {
			return E.Nil()
		}
	}
	return E.Invalid("argument", arg)
}
