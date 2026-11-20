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
	return newHTTPClient(timeout, &tls.Config{InsecureSkipVerify: true})
}

// NewVerifyingHTTPClient is the same as NewHTTPClient, but verifies TLS certificates.
func NewVerifyingHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, nil)
}

// newHTTPClient returns new HTTP client with `timeout`, proxy from the environment and TLS config `tlsCfg`.
func newHTTPClient(timeout time.Duration, tlsCfg *tls.Config) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsCfg,
			IdleConnTimeout: time.Second * 5,
		},
	}
}
