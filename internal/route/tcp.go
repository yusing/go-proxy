package route

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	T "github.com/yusing/go-proxy/internal/proxy/fields"
	U "github.com/yusing/go-proxy/internal/utils"
)

const tcpDialTimeout = 5 * time.Second

type (
	Pipes []U.BidirectionalPipe

	TCPRoute struct {
		*StreamRoute
		listener net.Listener
		pipe     Pipes
		mu       sync.Mutex
	}
)

func NewTCPRoute(base *StreamRoute) StreamImpl {
	return &TCPRoute{
		StreamRoute: base,
		pipe:        make(Pipes, 0),
	}
}

func (route *TCPRoute) Setup() error {
	in, err := net.Listen("tcp", fmt.Sprintf(":%v", route.Port.ListeningPort))
	if err != nil {
		return err
	}
	//! this read the allocated port from original ':0'
	route.Port.ListeningPort = T.Port(in.Addr().(*net.TCPAddr).Port)
	route.listener = in
	return nil
}

func (route *TCPRoute) Accept() (any, error) {
	return route.listener.Accept()
}

func (route *TCPRoute) Handle(c any) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()

	ctx, cancel := context.WithTimeout(route.task.Context(), tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort)
	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, string(route.Scheme.ProxyScheme), serverAddr)
	if err != nil {
		return err
	}

	route.mu.Lock()

	pipe := U.NewBidirectionalPipe(route.task.Context(), clientConn, serverConn)
	route.pipe = append(route.pipe, pipe)

	route.mu.Unlock()
	return pipe.Start()
}

func (route *TCPRoute) CloseListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
	route.listener = nil
}
