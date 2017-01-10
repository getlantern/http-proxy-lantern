package opsfilter

import (
	"net"
	"net/http"

	"github.com/getlantern/golog"
	"github.com/getlantern/ops"
	"github.com/gorilla/context"

	"github.com/getlantern/http-proxy-lantern/common"
	"github.com/getlantern/http-proxy/filters"
	"github.com/getlantern/http-proxy/listeners"
)

var (
	log = golog.LoggerFor("logging")
)

type opsfilter struct{}

// New constructs a new filter that adds ops context.
func New() filters.Filter {
	return &opsfilter{}
}

func (f *opsfilter) Apply(resp http.ResponseWriter, req *http.Request, next filters.Next) error {
	deviceID := req.Header.Get(common.DeviceIdHeader)
	originHost, originPort, _ := net.SplitHostPort(req.Host)
	if (originPort == "0" || originPort == "") && req.Method != http.MethodConnect {
		// Default port for HTTP
		originPort = "80"
	}

	op := ops.
		Begin("proxy").
		Set("device_id", deviceID).
		Set("origin", req.Host).
		Set("origin_host", originHost).
		Set("origin_port", originPort)
	defer op.End()

	ctx := map[string]interface{}{
		"deviceid":    deviceID,
		"origin":      req.Host,
		"origin_host": originHost,
		"origin_port": originPort,
	}

	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		op.Set("client_ip", clientIP)
		ctx["client_ip"] = clientIP
	}

	// Send the same context data to measured as well
	wc := context.Get(req, "conn").(listeners.WrapConn)
	wc.ControlMessage("measured", ctx)

	return next()
}
