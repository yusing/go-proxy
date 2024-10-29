package route

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/yusing/go-proxy/internal/net/types"
	T "github.com/yusing/go-proxy/internal/proxy/fields"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	Stream struct {
		*StreamRoute

		listener   types.StreamListener
		targetAddr net.Addr
	}
)

const (
	streamFirstConnBufferSize = 128
	streamDialTimeout         = 5 * time.Second
)

func NewStream(base *StreamRoute) *Stream {
	return &Stream{
		StreamRoute: base,
	}
}

func (stream *Stream) Addr() net.Addr {
	if stream.listener == nil {
		panic("listener is nil")
	}
	return stream.listener.Addr()
}

func (stream *Stream) Setup() error {
	var lcfg net.ListenConfig
	var err error

	switch stream.Scheme.ListeningScheme {
	case "tcp":
		stream.targetAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%v", stream.Host, stream.Port.ProxyPort))
		if err != nil {
			return err
		}
		tcpListener, err := lcfg.Listen(stream.task.Context(), "tcp", fmt.Sprintf(":%v", stream.Port.ListeningPort))
		if err != nil {
			return err
		}
		stream.Port.ListeningPort = T.Port(tcpListener.Addr().(*net.TCPAddr).Port)
		stream.listener = types.NetListener(tcpListener)
	case "udp":
		stream.targetAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", stream.Host, stream.Port.ProxyPort))
		if err != nil {
			return err
		}
		udpListener, err := lcfg.ListenPacket(stream.task.Context(), "udp", fmt.Sprintf(":%v", stream.Port.ListeningPort))
		if err != nil {
			return err
		}
		udpConn, ok := udpListener.(*net.UDPConn)
		if !ok {
			udpListener.Close()
			return errors.New("udp listener is not *net.UDPConn")
		}
		stream.Port.ListeningPort = T.Port(udpConn.LocalAddr().(*net.UDPAddr).Port)
		stream.listener = NewUDPForwarder(stream.task.Context(), udpConn, stream.targetAddr)
	default:
		panic("should not reach here")
	}

	return nil
}

func (stream *Stream) Accept() (types.StreamConn, error) {
	if stream.listener == nil {
		return nil, errors.New("listener is nil")
	}
	return stream.listener.Accept()
}

func (stream *Stream) Handle(conn types.StreamConn) error {
	switch conn := conn.(type) {
	case *UDPConn:
		switch stream := stream.listener.(type) {
		case *UDPForwarder:
			return stream.Handle(conn)
		default:
			return fmt.Errorf("unexpected listener type: %T", stream)
		}
	case io.ReadWriteCloser:
		stream.task.OnCancel("close conn", func() { conn.Close() })

		dialer := &net.Dialer{Timeout: streamDialTimeout}
		dstConn, err := dialer.DialContext(stream.task.Context(), stream.targetAddr.Network(), stream.targetAddr.String())
		if err != nil {
			return err
		}
		defer dstConn.Close()
		pipe := U.NewBidirectionalPipe(stream.task.Context(), conn, dstConn)
		return pipe.Start()
	default:
		return fmt.Errorf("unexpected conn type: %T", conn)
	}
}

func (stream *Stream) Close() error {
	return stream.listener.Close()
}
