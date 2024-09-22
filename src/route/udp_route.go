package route

import (
	"fmt"
	"io"
	"net"

	U "github.com/yusing/go-proxy/utils"
	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	UDPRoute struct {
		*StreamRoute

		connMap UDPConnMap

		listeningConn *net.UDPConn
		targetAddr    *net.UDPAddr
	}
	UDPConn struct {
		src *net.UDPConn
		dst *net.UDPConn
		U.BidirectionalPipe
	}
	UDPConnMap = F.Map[string, *UDPConn]
)

var NewUDPConnMap = F.NewMapOf[string, *UDPConn]

func NewUDPRoute(base *StreamRoute) StreamImpl {
	return &UDPRoute{
		StreamRoute: base,
		connMap:     NewUDPConnMap(),
	}
}

func (route *UDPRoute) Setup() error {
	laddr, err := net.ResolveUDPAddr(string(route.Scheme.ListeningScheme), fmt.Sprintf(":%v", route.Port.ListeningPort))
	if err != nil {
		return err
	}
	source, err := net.ListenUDP(string(route.Scheme.ListeningScheme), laddr)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveUDPAddr(string(route.Scheme.ProxyScheme), fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort))
	if err != nil {
		source.Close()
		return err
	}

	route.listeningConn = source
	route.targetAddr = raddr
	return nil
}

func (route *UDPRoute) Accept() (any, error) {
	in := route.listeningConn

	buffer := make([]byte, udpBufferSize)
	nRead, srcAddr, err := in.ReadFromUDP(buffer)

	if err != nil {
		return nil, err
	}

	if nRead == 0 {
		return nil, io.ErrShortBuffer
	}

	key := srcAddr.String()
	conn, ok := route.connMap.Load(key)

	if !ok {
		srcConn, err := net.DialUDP("udp", nil, srcAddr)
		if err != nil {
			return nil, err
		}
		dstConn, err := net.DialUDP("udp", nil, route.targetAddr)
		if err != nil {
			srcConn.Close()
			return nil, err
		}
		conn = &UDPConn{
			srcConn,
			dstConn,
			U.NewBidirectionalPipe(route.ctx, sourceRWCloser{in, dstConn}, sourceRWCloser{in, srcConn}),
		}
		route.connMap.Store(key, conn)
	}

	_, err = conn.dst.Write(buffer[:nRead])
	return conn, err
}

func (route *UDPRoute) Handle(c any) error {
	return c.(*UDPConn).Start()
}

func (route *UDPRoute) CloseListeners() {
	if route.listeningConn != nil {
		route.listeningConn.Close()
		route.listeningConn = nil
	}
	route.connMap.RangeAll(func(_ string, conn *UDPConn) {
		if err := conn.src.Close(); err != nil {
			route.l.Errorf("error closing src conn: %s", err)
		}
		if err := conn.dst.Close(); err != nil {
			route.l.Error("error closing dst conn: %s", err)
		}
	})
	route.connMap.Clear()
}

type sourceRWCloser struct {
	server *net.UDPConn
	*net.UDPConn
}

func (w sourceRWCloser) Write(p []byte) (int, error) {
	return w.server.WriteToUDP(p, w.RemoteAddr().(*net.UDPAddr)) // TODO: support non udp
}
