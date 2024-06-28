package water

import (
	"context"
	"encoding/base64"
	"net"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v1"
)

var log = golog.LoggerFor("water")

// NewWATERListener creates a WATER listener
// Currently water doesn't support customized TCP connections and we need to listen and receive requests directly from the WATER listener
func NewWATERListener(ctx context.Context, address, wasm string) (net.Listener, error) {
	decodedWASM, err := base64.StdEncoding.DecodeString(wasm)
	if err != nil {
		log.Errorf("failed to decode WASM base64: %v", err)
		return nil, err
	}

	cfg := &water.Config{
		TransportModuleBin: decodedWASM,
		//NetworkListener:     baseListener,
		ModuleConfigFactory: water.NewWazeroModuleConfigFactory(),
	}

	waterListener, err := cfg.ListenContext(ctx, "tcp", address)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	return waterListener, nil
}
