package types

import (
	"fmt"
	"net"
)

type Stream interface {
	fmt.Stringer
	Setup() error
	Accept() (conn StreamConn, err error)
	Handle(conn StreamConn) error
	CloseListeners()
}

type StreamConn interface {
	RemoteAddr() net.Addr
	Close() error
}
