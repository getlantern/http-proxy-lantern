package main

import (
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/golog"
	"github.com/getlantern/measured"

	"github.com/getlantern/http-proxy/commonfilter"
	"github.com/getlantern/http-proxy/forward"
	"github.com/getlantern/http-proxy/httpconnect"
	"github.com/getlantern/http-proxy/listeners"
	"github.com/getlantern/http-proxy/logging"
	"github.com/getlantern/http-proxy/server"

	"github.com/getlantern/http-proxy-lantern/analytics"
	"github.com/getlantern/http-proxy-lantern/configserverfilter"
	"github.com/getlantern/http-proxy-lantern/devicefilter"
	lanternlisteners "github.com/getlantern/http-proxy-lantern/listeners"
	"github.com/getlantern/http-proxy-lantern/mimic"
	"github.com/getlantern/http-proxy-lantern/ping"
	"github.com/getlantern/http-proxy-lantern/profilter"
	"github.com/getlantern/http-proxy-lantern/redis"
	"github.com/getlantern/http-proxy-lantern/tokenfilter"
)

var (
	testingLocal = false
	log          = golog.LoggerFor("lantern-proxy")

	help                         = flag.Bool("help", false, "Get usage help")
	keyfile                      = flag.String("key", "", "Private key file name")
	certfile                     = flag.String("cert", "", "Certificate file name")
	https                        = flag.Bool("https", false, "Use TLS for client to proxy communication")
	addr                         = flag.String("addr", ":8080", "Address to listen")
	maxConns                     = flag.Uint64("maxconns", 0, "Max number of simultaneous connections allowed connections")
	idleClose                    = flag.Uint64("idleclose", 30, "Time in seconds that an idle connection will be allowed before closing it")
	token                        = flag.String("token", "", "Lantern token")
	enableReports                = flag.Bool("enablereports", false, "Enable stats reporting")
	enablePro                    = flag.Bool("enablepro", false, "Enable Lantern Pro support")
	serverId                     = flag.String("serverid", "", "Server Id required for Pro-supporting servers")
	logglyToken                  = flag.String("logglytoken", "", "Token used to report to loggly.com, not reporting if empty")
	cfgSvrAuthToken              = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	cfgSvrDomains                = flag.String("cfgsvrdomains", "", "Config-server domains on which to attach auth token, separated by comma")
	pprofAddr                    = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")
	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0.01, "The percentage of requests to sample (0.01 = 1%)")
)

func main() {
	var err error

	iniflags.Parse()
	if *help {
		flag.Usage()
		return
	}

	// Logging
	// TODO: use real parameters
	err = logging.Init("instanceid", "version", "releasedate", *logglyToken)
	if err != nil {
		log.Fatal(err)
	}

	// Redis configuration
	redisAddr := os.Getenv("REDIS_PRODUCTION_URL")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	// Reporting
	if *enableReports {
		rp, err := redis.NewMeasuredReporter(redisAddr)
		if err != nil {
			log.Errorf("Error connecting to redis: %v", err)
		} else {
			measured.Start(20*time.Second, rp)
			defer measured.Stop()
		}
	}

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	// Middleware
	forwarder, err := forward.New(nil, forward.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	httpConnect, err := httpconnect.New(forwarder, httpconnect.IdleTimeoutSetter(time.Duration(*idleClose)*time.Second))
	if err != nil {
		log.Fatal(err)
	}

	var nextFilter http.Handler = httpConnect
	if *cfgSvrAuthToken != "" || *cfgSvrDomains != "" {
		domains := strings.Split(*cfgSvrDomains, ",")
		nextFilter, err = configserverfilter.New(httpConnect, configserverfilter.AuthToken(*cfgSvrAuthToken), configserverfilter.Domains(domains))
		if err != nil {
			log.Fatal(err)
		}
	}

	pingFilter := ping.New(nextFilter)

	commonFilter, err := commonfilter.New(pingFilter,
		testingLocal,
		commonfilter.SetException("127.0.0.1:7300"),
	)
	if err != nil {
		log.Fatal(err)
	}

	deviceFilterPost := devicefilter.NewPost(commonFilter)

	analyticsFilter := analytics.New(*proxiedSitesTrackingId, *proxiedSitesSamplePercentage, deviceFilterPost)

	deviceFilterPre, err := devicefilter.NewPre(analyticsFilter)
	if err != nil {
		log.Fatal(err)
	}

	tokenFilter, err := tokenfilter.New(deviceFilterPre, tokenfilter.TokenSetter(*token))
	if err != nil {
		log.Fatal(err)
	}

	var srv *server.Server

	// Pro support
	if *enablePro {
		if *serverId == "" {
			log.Fatal("Enabling Pro requires setting the \"serverid\" flag")
		}
		log.Debug("This proxy is configured to support Lantern Pro")
		proFilter, err := profilter.New(tokenFilter,
			profilter.RedisConfigSetter(redisAddr, *serverId),
		)
		if err != nil {
			log.Fatal(err)
		}

		srv = server.NewServer(proFilter)
	} else {
		srv = server.NewServer(tokenFilter)
	}

	// Add net.Listener wrappers for inbound connections
	if *enableReports {
		srv.AddListenerWrappers(
			// Measure connections
			func(ls net.Listener) net.Listener {
				return listeners.NewMeasuredListener(ls, 100*time.Millisecond)
			},
		)
	}
	srv.AddListenerWrappers(
		// Close connections after 30 seconds of no activity
		func(ls net.Listener) net.Listener {
			return listeners.NewIdleConnListener(ls, time.Duration(*idleClose)*time.Second)
		},
		// Preprocess connection to issue custom errors before they are passed to the server
		func(ls net.Listener) net.Listener {
			return lanternlisteners.NewPreprocessorListener(ls)
		},
	)

	initMimic := func(addr string) {
		mimic.SetServerAddr(addr)
	}

	if *https {
		err = srv.ServeHTTPS(*addr, *keyfile, *certfile, initMimic)
	} else {
		err = srv.ServeHTTP(*addr, initMimic)
	}
	if err != nil {
		log.Errorf("Error serving: %v", err)
	}
}
