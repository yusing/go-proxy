package route

import (
	"context"
	"fmt"
	"net"
	"sync"
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
		listener:    nil,
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

	ctx, cancel := context.WithTimeout(context.Background(), tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort)
	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, string(route.Scheme.ProxyScheme), serverAddr)
	if err != nil {
		return err
	}

	pipeCtx, pipeCancel := context.WithCancel(context.Background())
	go func() {
		<-route.stopCh
		pipeCancel()
	}()

	route.mu.Lock()
	pipe := U.NewBidirectionalPipe(pipeCtx, clientConn, serverConn)
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
	for _, pipe := range route.pipe {
		if err := pipe.Stop(); err.HasError() {
			route.l.Error(err)
		}
	}
}
