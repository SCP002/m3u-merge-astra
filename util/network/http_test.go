package network

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"m3u-merge-astra/util/network/nettest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMux returns handler serving /ok/, /not_found/ and /timeout/ paths.
func newTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	mux.HandleFunc("/not_found/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/timeout/", func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // Hang until the client gives up.
	})
	return mux
}

func TestNewHttpClient(t *testing.T) {
	httpSrv := httptest.NewServer(newTestMux())
	defer httpSrv.Close()
	httpsSrv := httptest.NewTLSServer(newTestMux())
	defer httpsSrv.Close()

	// Swap schemes to send requests with the wrong protocol.
	httpURL := httpSrv.URL
	httpsURL := httpsSrv.URL
	httpToHTTPSURL := strings.Replace(httpsURL, "https://", "http://", 1)
	httpsToHTTPURL := strings.Replace(httpURL, "http://", "https://", 1)

	// Reserve a free port and release it, so connections to it get refused.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	refusedURL := "http://" + listener.Addr().String()
	_ = listener.Close()

	client := NewHttpClient(time.Millisecond * 200)

	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantErr    ErrType
	}{
		{name: "http ok", url: httpURL + "/ok/1", wantStatus: 200, wantErr: Nil},
		{name: "https ok with self-signed cert", url: httpsURL + "/ok/1", wantStatus: 200, wantErr: Nil},
		{name: "http not found", url: httpURL + "/not_found/1", wantStatus: 404, wantErr: Nil},
		{name: "https not found", url: httpsURL + "/not_found/1", wantStatus: 404, wantErr: Nil},
		{name: "http request to https server", url: httpToHTTPSURL + "/ok/1", wantStatus: 400, wantErr: Nil},
		{name: "https request to http server", url: httpsToHTTPURL + "/ok/1", wantErr: HTTPSClientHTTPServer},
		{name: "refused", url: refusedURL + "/ok/1", wantErr: Refused},
		{name: "http timeout", url: httpURL + "/timeout/1", wantErr: Timeout},
		{name: "https timeout", url: httpsURL + "/timeout/1", wantErr: Timeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(tt.url)
			assert.Exactly(t, tt.wantErr, GetErrType(err))
			if tt.wantStatus == 0 {
				assert.Nil(t, resp)
				return
			}
			if assert.NotNil(t, resp) {
				defer func() { _ = resp.Body.Close() }()
				assert.Exactly(t, tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestGetErrTypeNoSuchHost(t *testing.T) {
	srv := nettest.NewServers(t, newTestMux())
	client := srv.Client(time.Second)

	for _, scheme := range []string{"http", "https"} {
		resp, err := client.Get(scheme + "://" + nettest.UnresolvableHost + "/ok/1")
		assert.Nil(t, resp)
		assert.Exactly(t, NoSuchHost, GetErrType(err), scheme)
	}
}
