package network

import (
	"crypto/tls"
	"net/http"
	"time"
)

// NewHTTPClient returns new HTTP client with a time limit for requests set to `timeout`.
// The proxy is taken from the environment variables HTTP_PROXY, HTTPS_PROXY and NO_PROXY.
// TLS certificates are not verified, as astra plays inputs with invalid certificates, so they should not be
// considered dead.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			IdleConnTimeout: time.Second * 5,
		},
	}
}
