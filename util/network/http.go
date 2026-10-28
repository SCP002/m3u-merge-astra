package network

import (
	"crypto/tls"
	"net/http"
	"time"
)

// NewHttpClient returns new HTTP client.
//
// `timeout` is a time limit for requests made by returned client.
func NewHttpClient(timeout time.Duration) *http.Client {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	return client
}
