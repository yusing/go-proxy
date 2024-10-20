package types

import (
	"fmt"
	"net"
)

type Stream interface {
	fmt.Stringer
	net.Listener
	Setup() error
	Handle(conn net.Conn) error
}
