package route

import (
	"context"
	"io"
	"net"
	"sync"

	F "github.com/yusing/go-proxy/internal/utils/functional"
)

type (
	UDPListener struct {
		ctx      context.Context
		listener net.PacketConn
		connMap  UDPConnMap
		mu       sync.Mutex
	}
	UDPConnMap = F.Map[string, net.Conn]
)

var NewUDPConnMap = F.NewMap[UDPConnMap]

func newUDPListenerAdaptor(ctx context.Context, listener net.PacketConn) net.Listener {
	return &UDPListener{
		ctx:      ctx,
		listener: listener,
		connMap:  NewUDPConnMap(),
	}
}

// Addr implements net.Listener.
func (route *UDPListener) Addr() net.Addr {
	return route.listener.LocalAddr()
}

func (udpl *UDPListener) Accept() (net.Conn, error) {
	in := udpl.listener

	buffer := make([]byte, streamBufferSize)
	nRead, srcAddr, err := in.ReadFrom(buffer)
	if err != nil {
		return nil, err
	}

	if nRead == 0 {
		return nil, io.ErrShortBuffer
	}

	udpl.mu.Lock()
	defer udpl.mu.Unlock()

	key := srcAddr.String()
	conn, ok := udpl.connMap.Load(key)
	if !ok {
		dialer := &net.Dialer{Timeout: streamDialTimeout}
		srcConn, err := dialer.DialContext(udpl.ctx, srcAddr.Network(), srcAddr.String())
		if err != nil {
			return nil, err
		}
		udpl.connMap.Store(key, srcConn)
	}
	return conn, nil
}

// Close implements net.Listener.
func (route *UDPListener) Close() error {
	route.connMap.RangeAllParallel(func(key string, conn net.Conn) {
		conn.Close()
	})
	route.connMap.Clear()
	return route.listener.Close()
}
