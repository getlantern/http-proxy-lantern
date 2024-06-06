package shadowsocks

import (
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/netx"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	onet "github.com/Jigsaw-Code/outline-ss-server/net"
	"github.com/Jigsaw-Code/outline-ss-server/service"
)

// shadowsocks/local.go houses adapters for use with Lantern. This mostly is in
// place to allow the Lantern backend to handle upstream connections itself and
// have shadowsocks behave like other transports we use in Lantern.

var (
	log = golog.LoggerFor("shadowsocks")

	ErrListenerClosed = errors.New("listener closed")
)

// This value is lifted from the the main server.go to match behavior
// 59 seconds is most common timeout for servers that do not respond to invalid requests
const tcpReadTimeout time.Duration = 59 * time.Second

// HandleLocalPredicate is a type of function that determines whether to handle an
// upstream address locally or not.  If the function returns true, the address is
// handled locally.  If the funtion returns false, the address is handled by the
// default upstream dial.
type HandleLocalPredicate func(addr string) bool

// AlwaysLocal is a HandleLocalPredicate that requests local handling for all addresses
func AlwaysLocal(addr string) bool { return true }

// ListenLocalTCP creates a net.Listener that returns all inbound shadowsocks connections to the
// returned listener rather than dialing upstream. Any upstream or local handling should be handled by the
// caller of Accept().
func ListenLocalTCP(
	l net.Listener,
	ciphers service.CipherList,
	replayHistory int,
) (net.Listener, error) {
	replayCache := service.NewReplayCache(replayHistory)

	options := &ListenerOptions{
		Listener:           &tcpListenerAdapter{l},
		Ciphers:            ciphers,
		ReplayCache:        &replayCache,
		ShadowsocksMetrics: &service.NoOpTCPMetrics{},
	}

	return ListenLocalTCPOptions(options), nil
}

// ListenLocalTCPOptions creates a net.Listener that returns some inbound shadowsocks connections
// to the returned listener.  Which connnections are returned to the listener are determined
// by the ShouldHandleLocally predicate, which defaults to all connections.
// Any upstream handling should be handled by the caller of Accept() for any connection returned.
func ListenLocalTCPOptions(options *ListenerOptions) net.Listener {
	maxPending := options.MaxPendingConnections
	if maxPending == 0 {
		maxPending = DefaultMaxPending
	}

	l := &llistener{
		wrapped:      options.Listener,
		connections:  make(chan net.Conn, maxPending),
		closedSignal: make(chan struct{}),
	}

	timeout := options.Timeout
	if timeout == 0 {
		timeout = tcpReadTimeout
	}

	validator := options.TargetIPValidator
	if validator == nil {
		validator = onet.RequirePublicIP
	}

	isLocal := options.ShouldHandleLocally
	if isLocal == nil {
		isLocal = AlwaysLocal
	}

	authFunc := service.NewShadowsocksStreamAuthenticator(options.Ciphers, options.ReplayCache, options.ShadowsocksMetrics)
	tcpHandler := service.NewTCPHandler(options.Listener.Addr().(*net.TCPAddr).Port, authFunc, options.ShadowsocksMetrics, timeout)
	tcpHandler.SetTargetDialer(&transport.TCPDialer{Dialer: net.Dialer{Control: func(network, address string, c syscall.RawConn) error {
		ip, _, _ := net.SplitHostPort(address)
		return validator(net.ParseIP(ip))
	}}})
	accept := func() (transport.StreamConn, error) {
		switch l.wrapped.(type) {
		case *tcpListenerAdapter:
			// This is a local listener, we can handle the connection locally
			conn, err := l.wrapped.(*tcpListenerAdapter).AcceptTCP()
			if err == nil {
				conn.SetKeepAlive(true)
				if options.Accept != nil {
					err = options.Accept(conn)
				}
			}
			return conn, err
		case *net.TCPListener:
			conn, err := l.wrapped.(*net.TCPListener).AcceptTCP()
			if err == nil {
				conn.SetKeepAlive(true)
			}
			return conn, err
		default:
			return nil, errors.New("unsupported listener type")
		}
	}

	go service.StreamServe(accept, tcpHandler.Handle)
	return l.wrapped
}

// this is an adapter that fulfills the expectation
// of the shadowsocks handler that it can independently
// close the read and write on it's upstream side.
type tcpConnAdapter struct {
	transport.StreamConn
}

func (c *tcpConnAdapter) Wrapped() net.Conn {
	return c.StreamConn
}

// this is triggered when the remote end is finished.
// This triggers a close of both ends.
func (c *tcpConnAdapter) CloseRead() error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.CloseRead()
	}
	return c.StreamConn.Close()
}

// this is triggered when a client finishes writing,
// it is handled as a no-op, we just ignore it since
// we don't depend on half closing the connection to
// signal anything.
func (c *tcpConnAdapter) CloseWrite() error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.CloseWrite()
	}
	return nil
}

func (c *tcpConnAdapter) SetKeepAlive(keepAlive bool) error {
	tcpConn, ok := c.asTCPConn()
	if ok {
		return tcpConn.SetKeepAlive(keepAlive)
	}
	return nil
}

func (c *tcpConnAdapter) asTCPConn() (*net.TCPConn, bool) {
	var tcpConn *net.TCPConn
	netx.WalkWrapped(c.StreamConn, func(conn net.Conn) bool {
		switch t := conn.(type) {
		case *net.TCPConn:
			tcpConn = t
			return false
		}

		// Keep looking
		return true
	})
	return tcpConn, tcpConn != nil
}

type tcpListenerAdapter struct {
	net.Listener
}

func (l *tcpListenerAdapter) AcceptTCP() (TCPConn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &tcpConnAdapter{conn.(transport.StreamConn)}, nil
}

// this is an adapter that forwards the remote address
// on the "real" client connection to the consumer of
// the listener.  The real requested upstream address
// is also available if needed.
type lfwd struct {
	net.Conn
	clientTCPConn  net.Conn
	remoteAddr     net.Addr
	upstreamTarget string
}

func (l *lfwd) RemoteAddr() net.Addr {
	return l.remoteAddr
}

func (l *lfwd) UpstreamTarget() string {
	return l.upstreamTarget
}

func (l *lfwd) Wrapped() net.Conn {
	return l.clientTCPConn.(*tcpConnAdapter).Wrapped()
}
