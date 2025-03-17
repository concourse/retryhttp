package retryhttp

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:generate counterfeiter . Conn

type Conn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

//go:generate counterfeiter . DoHijackCloser

type DoHijackCloser interface {
	Do(req *http.Request) (resp *http.Response, err error)
	Hijack() (c net.Conn, r *bufio.Reader)
	Close() error
}

//go:generate counterfeiter . DoHijackCloserFactory

type DoHijackCloserFactory interface {
	NewDoHijackCloser(c net.Conn, r *bufio.Reader) DoHijackCloser
}

type defaultDoHijackCloserFactory struct{}

var DefaultDoHijackCloserFactory DoHijackCloserFactory = defaultDoHijackCloserFactory{}

type CustomClientConn struct {
	conn   net.Conn
	reader *bufio.Reader
	client *http.Client
}

// Do performs the HTTP request and returns the response
func (c *CustomClientConn) Do(req *http.Request) (*http.Response, error) {
	// Create a simple transport that uses our connection
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return c.conn, nil
		},
		DisableKeepAlives: true,
	}

	c.client = &http.Client{
		Transport: transport,
	}

	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	return c.client.Do(reqCopy)
}

// Hijack returns the underlying connection and reader
func (c *CustomClientConn) Hijack() (net.Conn, *bufio.Reader) {
	return c.conn, c.reader
}

// Close closes the connection
func (c *CustomClientConn) Close() error {
	return c.conn.Close()
}

// NewDoHijackCloser creates a new DoHijackCloser that wraps the given connection and reader
func (f defaultDoHijackCloserFactory) NewDoHijackCloser(c net.Conn, r *bufio.Reader) DoHijackCloser {
	if r == nil {
		r = bufio.NewReader(c)
	}
	return &CustomClientConn{
		conn:   c,
		reader: r,
	}
}

//go:generate counterfeiter . HijackCloser

type HijackCloser interface {
	Hijack() (c net.Conn, r *bufio.Reader)
	Close() error
}

//go:generate counterfeiter . HijackableClient

type HijackableClient interface {
	Do(req *http.Request) (*http.Response, HijackCloser, error)
}

type BasicHijackableClient struct {
	Dial                  func(network, addr string) (net.Conn, error)
	DoHijackCloserFactory DoHijackCloserFactory
}

var DefaultHijackableClient HijackableClient = &BasicHijackableClient{
	Dial: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial,
	DoHijackCloserFactory: DefaultDoHijackCloserFactory,
}

func (c *BasicHijackableClient) Do(req *http.Request) (*http.Response, HijackCloser, error) {
	conn, err := c.Dial("tcp", canonicalAddr(req.URL))
	if err != nil {
		return nil, nil, err
	}

	client := c.DoHijackCloserFactory.NewDoHijackCloser(conn, nil)

	httpResp, err := client.Do(req)
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return httpResp, client, nil
}

var portMap = map[string]string{
	"http": "80",
}

// canonicalAddr returns url.Host but always with a ":port" suffix
func canonicalAddr(url *url.URL) string {
	addr := url.Host
	if !hasPort(addr) {
		port, ok := portMap[url.Scheme]
		if !ok {
			port = "80" // Default to HTTP port if scheme not recognized
		}
		return addr + ":" + port
	}
	return addr
}

// Given a string of the form "host", "host:port", or "[ipv6::address]:port",
// return true if the string includes a port.
func hasPort(s string) bool {
	return strings.LastIndex(s, ":") > strings.LastIndex(s, "]")
}
