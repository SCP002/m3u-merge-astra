// Package nettest provides hermetic HTTP(S) servers and clients for tests.
package nettest

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// UnresolvableHost is a host name for which clients returned by Servers.Client fail with DNS "no such host" error.
const UnresolvableHost = "no-such-host.test"

// RefusedHost is a host name for which clients returned by Servers.Client fail with "connection refused" error.
const RefusedHost = "refused.test"

// Servers holds HTTP and HTTPS test servers sharing the same handler.
type Servers struct {
	HTTP        *httptest.Server
	HTTPS       *httptest.Server
	refusedAddr string // refusedAddr is a local address nothing is listening on.
}

// NewServers starts HTTP and HTTPS servers on random local ports serving `handler`.
//
// Servers are closed on `t` cleanup. Handlers which block should return on request context cancellation, otherwise
// the cleanup waits for them.
func NewServers(t testing.TB, handler http.Handler) Servers {
	t.Helper()

	httpSrv := httptest.NewServer(handler)
	t.Cleanup(httpSrv.Close)
	httpsSrv := httptest.NewTLSServer(handler)
	t.Cleanup(httpsSrv.Close)

	// Reserve a free port and release it, so connections to it get refused.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Reserve a free port: %v", err)
	}
	refusedAddr := listener.Addr().String()
	_ = listener.Close()

	return Servers{HTTP: httpSrv, HTTPS: httpsSrv, refusedAddr: refusedAddr}
}

// Client returns new HTTP client with request `timeout`, which connects to the test servers.
func (s Servers) Client(timeout time.Duration) *http.Client {
	var dialer net.Dialer

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				switch {
				case host == UnresolvableHost:
					return nil, &net.OpError{Op: "dial", Net: network, Err: &net.DNSError{
						Err: "no such host", Name: host, IsNotFound: true,
					}}
				case host == RefusedHost:
					return dialer.DialContext(ctx, network, s.refusedAddr)
				case port == "443":
					return dialer.DialContext(ctx, network, s.HTTPS.Listener.Addr().String())
				default:
					return dialer.DialContext(ctx, network, s.HTTP.Listener.Addr().String())
				}
			},
		},
	}
}
