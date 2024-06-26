package analytics

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var portsAndProtos = map[string]string{
	"80":   "http",
	"443":  "https",
	"5002": "other",
	"0":    "unknown",
	"":     "unknown",
}

func TestNormalizeSite(t *testing.T) {
	am := New(&Options{
		TrackingID:       "12345",
		SamplePercentage: 1,
	})
	addrs, err := net.LookupHost("iad30s21-in-x05.1e100.net")
	if assert.NoError(t, err, "Should have been able to resolve yahoo.com") {
		for port, proto := range portsAndProtos {
			normalized := am.(*analyticsMiddleware).normalizeSite(addrs[0], port)
			assert.Len(t, normalized, 4, "Should have gotten two sites and one protocol")
			assert.Equal(t, "iad30s21-in-x05.1e100.net", normalized[1])
			assert.Equal(t, "/generated/1e100.net", normalized[2])
			assert.Equal(t, "/protocol/"+proto, normalized[3])
		}
	}
}

func TestDefaultPort(t *testing.T) {
	am := &analyticsMiddleware{
		Options: &Options{
			SamplePercentage: 1,
		},
		siteAccesses: make(chan *siteAccess, 1000),
	}
	req, _ := http.NewRequest(http.MethodGet, "www.google.com", nil)
	for _, host := range []string{"www.google.com", "www.google.com:0"} {
		req.Host = host
		am.track(req)
		sa := <-am.siteAccesses
		assert.Equal(t, "80", sa.port)
	}
}
