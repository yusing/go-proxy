package main

const (
	CommandStart     = ""
	CommandNewClient = "new-client"
)

type agentCommandValidator struct{}

func (v agentCommandValidator) IsCommandValid(cmd string) bool {
	switch cmd {
	case CommandStart, CommandNewClient:
		return true
	}
	return false
}
