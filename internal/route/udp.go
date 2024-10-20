package route

// import (
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net"

// 	"github.com/yusing/go-proxy/internal/net/types"
// 	T "github.com/yusing/go-proxy/internal/proxy/fields"
// 	U "github.com/yusing/go-proxy/internal/utils"
// 	F "github.com/yusing/go-proxy/internal/utils/functional"
// )

// type (
// 	UDPRoute struct {
// 		*StreamRoute

// 		connMap UDPConnMap

// 		listeningConn net.PacketConn
// 		targetAddr    *net.UDPAddr
// 	}
// 	UDPConn struct {
// 		key string
// 		src net.Conn
// 		dst net.Conn
// 		U.BidirectionalPipe
// 	}
// 	UDPConnMap = F.Map[string, *UDPConn]
// )

// var NewUDPConnMap = F.NewMap[UDPConnMap]

// const udpBufferSize = 8192

// func NewUDPRoute(base *StreamRoute) *UDPRoute {
// 	return &UDPRoute{
// 		StreamRoute: base,
// 		connMap:     NewUDPConnMap(),
// 	}
// }

// func (route *UDPRoute) Setup() error {
// 	var cfg net.ListenConfig
// 	source, err := cfg.ListenPacket(route.task.Context(), string(route.Scheme.ListeningScheme), fmt.Sprintf(":%v", route.Port.ListeningPort))
// 	if err != nil {
// 		return err
// 	}
// 	raddr, err := net.ResolveUDPAddr(string(route.Scheme.ProxyScheme), fmt.Sprintf("%s:%v", route.Host, route.Port.ProxyPort))
// 	if err != nil {
// 		source.Close()
// 		return err
// 	}

// 	//! this read the allocated listeningPort from original ':0'
// 	route.Port.ListeningPort = T.Port(source.LocalAddr().(*net.UDPAddr).Port)

// 	route.listeningConn = source
// 	route.targetAddr = raddr

// 	return nil
// }

// func (route *UDPRoute) Accept() (types.StreamConn, error) {
// 	in := route.listeningConn

// 	buffer := make([]byte, udpBufferSize)
// 	nRead, srcAddr, err := in.ReadFrom(buffer)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if nRead == 0 {
// 		return nil, io.ErrShortBuffer
// 	}

// 	key := srcAddr.String()
// 	conn, ok := route.connMap.Load(key)

// 	if !ok {
// 		srcConn, err := net.Dial(srcAddr.Network(), srcAddr.String())
// 		if err != nil {
// 			return nil, err
// 		}
// 		dstConn, err := net.Dial(route.targetAddr.Network(), route.targetAddr.String())
// 		if err != nil {
// 			srcConn.Close()
// 			return nil, err
// 		}
// 		conn = &UDPConn{
// 			key,
// 			srcConn,
// 			dstConn,
// 			U.NewBidirectionalPipe(route.task.Context(), sourceRWCloser{in, dstConn}, sourceRWCloser{in, srcConn}),
// 		}
// 		route.connMap.Store(key, conn)
// 	}

// 	_, err = conn.dst.Write(buffer[:nRead])
// 	return conn, err
// }

// func (route *UDPRoute) Handle(c types.StreamConn) error {
// 	switch c := c.(type) {
// 	case *UDPConn:
// 		err := c.Start()
// 		route.connMap.Delete(c.key)
// 		c.Close()
// 		return err
// 	case *net.TCPConn:
// 		in := route.listeningConn
// 		srcConn, err := net.DialTCP("tcp", nil, c.RemoteAddr().(*net.TCPAddr))
// 		if err != nil {
// 			return err
// 		}
// 		err = U.NewBidirectionalPipe(route.task.Context(), sourceRWCloser{in, c}, sourceRWCloser{in, srcConn}).Start()
// 		c.Close()
// 		return err
// 	}
// 	return fmt.Errorf("unknown conn type: %T", c)
// }

// func (route *UDPRoute) Close() error {
// 	route.connMap.RangeAllParallel(func(k string, v *UDPConn) {
// 		v.Close()
// 	})
// 	route.connMap.Clear()
// 	return route.listeningConn.Close()
// }

// // Close implements types.StreamConn
// func (conn *UDPConn) Close() error {
// 	return errors.Join(conn.src.Close(), conn.dst.Close())
// }

// // RemoteAddr implements types.StreamConn
// func (conn *UDPConn) RemoteAddr() net.Addr {
// 	return conn.src.RemoteAddr()
// }

// type sourceRWCloser struct {
// 	server net.PacketConn
// 	net.Conn
// }

// func (w sourceRWCloser) Write(p []byte) (int, error) {
// 	return w.server.WriteTo(p, w.RemoteAddr().(*net.UDPAddr))
// }
