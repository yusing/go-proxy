package main

import (
	"flag"

	"github.com/sirupsen/logrus"
)

type Args struct {
	Command string
}

const (
	CommandStart  = ""
	CommandVerify = "verify"
	CommandReload = "reload"
)

var ValidCommands = []string{CommandStart, CommandVerify, CommandReload}

func getArgs() Args {
	var args Args
	flag.Parse()
	args.Command = flag.Arg(0)
	if err := validateArgs(args.Command, ValidCommands); err != nil {
		logrus.Fatal(err)
	}
	return args
}

func validateArgs[T comparable](arg T, validArgs []T) error {
	for _, v := range validArgs {
		if arg == v {
			return nil
		}
	}
	return NewNestedError("invalid argument").Subjectf("%v", arg)
}
