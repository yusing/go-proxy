package route

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	U "github.com/yusing/go-proxy/utils"
)

const tcpDialTimeout = 5 * time.Second

type Pipes []*U.BidirectionalPipe

type TCPRoute struct {
	*StreamRoute
	listener net.Listener
	pipe     Pipes
	mu       sync.Mutex
}

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
	route.listener = in
	return nil
}

func (route *TCPRoute) Accept() (any, error) {
	return route.listener.Accept()
}

func (route *TCPRoute) Handle(c any) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()

	ctx, cancel := context.WithTimeout(route.ctx, tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort)
	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, string(route.Scheme.ProxyScheme), serverAddr)
	if err != nil {
		return err
	}

	route.mu.Lock()
	defer route.mu.Unlock()

	pipe := U.NewBidirectionalPipe(route.ctx, clientConn, serverConn)
	route.pipe = append(route.pipe, pipe)
	return pipe.Start()
}

func (route *TCPRoute) CloseListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
	route.listener = nil
	for _, pipe := range route.pipe {
		if err := pipe.Stop(); err != nil {
			switch err {
			// target closing connection
			// TODO: handle this by fixing utils/io.go
			case net.ErrClosed, syscall.EPIPE:
				return
			default:
				route.l.Error(err)
			}
		}
	}
}
