package api

import (
	"net/http/httptest"
	"testing"
)

func TestRequestIP(t *testing.T) {
	for _, test := range []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{name: "network client ignores forwarded header", remoteAddr: "192.0.2.44:1234", forwarded: "203.0.113.7", want: "192.0.2.44"},
		{name: "loopback proxy accepts valid client address", remoteAddr: "127.0.0.1:1234", forwarded: "203.0.113.7", want: "203.0.113.7"},
		{name: "loopback proxy rejects invalid client address", remoteAddr: "[::1]:1234", forwarded: "not-an-address", want: "::1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/v1/identify", nil)
			request.RemoteAddr = test.remoteAddr
			request.Header.Set("CF-Connecting-IP", test.forwarded)
			if got := requestIP(request); got != test.want {
				t.Fatalf("requestIP() = %q, want %q", got, test.want)
			}
		})
	}
}
