package route

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
	T "github.com/yusing/go-proxy/internal/proxy/fields"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

const tcpDialTimeout = 5 * time.Second

type (
	TCPConnMap = F.Map[net.Conn, struct{}]
	TCPRoute   struct {
		*StreamRoute
		listener *net.TCPListener
	}
)

func NewTCPRoute(base *StreamRoute) *TCPRoute {
	return &TCPRoute{StreamRoute: base}
}

func (route *TCPRoute) Setup() error {
	in, err := net.Listen("tcp", fmt.Sprintf(":%v", route.Port.ListeningPort))
	if err != nil {
		return err
	}
	//! this read the allocated port from original ':0'
	route.Port.ListeningPort = T.Port(in.Addr().(*net.TCPAddr).Port)
	route.listener = in.(*net.TCPListener)
	return nil
}

func (route *TCPRoute) Accept() (types.StreamConn, error) {
	route.listener.SetDeadline(time.Now().Add(time.Second))
	return route.listener.Accept()
}

func (route *TCPRoute) Handle(c types.StreamConn) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()
	route.task.OnComplete("close conn", func() { clientConn.Close() })

	ctx, cancel := context.WithTimeout(route.task.Context(), tcpDialTimeout)

	serverAddr := fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort)
	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, string(route.Scheme.ProxyScheme), serverAddr)
	cancel()
	if err != nil {
		return err
	}

	pipe := U.NewBidirectionalPipe(route.task.Context(), clientConn, serverConn)
	return pipe.Start()
}

func (route *TCPRoute) CloseListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
}
