package route

import (
	"errors"
	"fmt"
	"net"
	"time"

	T "github.com/yusing/go-proxy/internal/proxy/fields"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	RawStream struct {
		*StreamRoute

		listener   net.Listener
		targetAddr net.Addr
	}
)

const (
	streamBufferSize  = 8192
	streamDialTimeout = 5 * time.Second
)

func NewRawStreamRoute(base *StreamRoute) *RawStream {
	return &RawStream{
		StreamRoute: base,
	}
}

func (route *RawStream) Setup() error {
	var lcfg net.ListenConfig
	var err error

	switch route.Scheme.ListeningScheme {
	case "tcp":
		route.targetAddr, err = net.ResolveTCPAddr(string(route.Scheme.ProxyScheme), fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort))
		if err != nil {
			return err
		}
		tcpListener, err := lcfg.Listen(route.task.Context(), "tcp", fmt.Sprintf(":%v", route.Port.ListeningPort))
		if err != nil {
			return err
		}
		route.Port.ListeningPort = T.Port(tcpListener.Addr().(*net.TCPAddr).Port)
		route.listener = tcpListener
	case "udp":
		route.targetAddr, err = net.ResolveUDPAddr(string(route.Scheme.ProxyScheme), fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort))
		if err != nil {
			return err
		}
		udpListener, err := lcfg.ListenPacket(route.task.Context(), "udp", fmt.Sprintf(":%v", route.Port.ListeningPort))
		if err != nil {
			return err
		}
		route.Port.ListeningPort = T.Port(udpListener.LocalAddr().(*net.UDPAddr).Port)
		route.listener = newUDPListenerAdaptor(route.task.Context(), udpListener)
	default:
		return errors.New("invalid listening scheme: " + string(route.Scheme.ListeningScheme))
	}

	return nil
}

func (route *RawStream) Accept() (net.Conn, error) {
	if route.listener == nil {
		return nil, errors.New("listener not yet set up")
	}
	return route.listener.Accept()
}

func (route *RawStream) Handle(c net.Conn) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()
	route.task.OnCancel("close conn", func() { clientConn.Close() })

	dialer := &net.Dialer{Timeout: streamDialTimeout}

	serverAddr := fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort)
	serverConn, err := dialer.DialContext(route.task.Context(), string(route.Scheme.ProxyScheme), serverAddr)
	if err != nil {
		return err
	}

	pipe := U.NewBidirectionalPipe(route.task.Context(), clientConn, serverConn)
	return pipe.Start()
}

func (route *RawStream) Close() error {
	return route.listener.Close()
}
