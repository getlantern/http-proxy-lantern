package main

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/panicwrap"
	"github.com/vharitonsky/iniflags"

	"github.com/getlantern/geo"
	"github.com/getlantern/golog"
	"github.com/getlantern/memhelper"

	proxy "github.com/getlantern/http-proxy-lantern/v2"
	"github.com/getlantern/http-proxy-lantern/v2/blacklist"
	"github.com/getlantern/http-proxy-lantern/v2/googlefilter"
	"github.com/getlantern/http-proxy-lantern/v2/obfs4listener"
	lanternredis "github.com/getlantern/http-proxy-lantern/v2/redis"
	"github.com/getlantern/http-proxy-lantern/v2/shadowsocks"
	"github.com/getlantern/http-proxy-lantern/v2/stackdrivererror"
	"github.com/getlantern/http-proxy-lantern/v2/throttle"
	"github.com/getlantern/http-proxy-lantern/v2/tlslistener"
)

var (
	log        = golog.LoggerFor("lantern-proxy")
	revision   = "unknown" // overridden by Makefile
	build_type = "unknown" // overriden by Makefile

	// Use our own S3 bucket  distribution which fetches the origin at most once per
	// day to avoid hitting the 2000 downloads/day limit imposed by MaxMind.
	geolite2CityURL = "https://storage.googleapis.com/lanterngeo/GeoLite2-City.mmdb.tar.gz"
	geoip2ISPURL    = "https://storage.googleapis.com/lanterngeo/GeoIP2-ISP.mmdb.tar.gz"

	hostname, _ = os.Hostname()

	addr          = flag.String("addr", "", "Address to listen with HTTP(S)")
	multiplexAddr = flag.String("multiplexaddr", "", "Multiplexed address at which to listen with HTTP(S)")
	lampshadeAddr = flag.String("lampshade-addr", "", "Address at which to listen for lampshade connections with tcp. Requires https to be true.")
	quicIETFAddr  = flag.String("quic-ietf-addr", "", "Address at which to listen for IETF QUIC connections.")
	quicBBR       = flag.Bool("quic-bbr", false, "Should quic-go use BBR instead of CUBIC")
	wssAddr       = flag.String("wss-addr", "", "Address at which to listen for WSS connections.")
	kcpConf       = flag.String("kcpconf", "", "Path to file configuring kcp")

	obfs4Addr                          = flag.String("obfs4-addr", "", "Provide an address here in order to listen with obfs4")
	obfs4MultiplexAddr                 = flag.String("obfs4-multiplexaddr", "", "Provide an address here in order to listen with multiplexed obfs4")
	obfs4Dir                           = flag.String("obfs4-dir", ".", "Directory where obfs4 can store its files")
	obfs4HandshakeConcurrency          = flag.Int("obfs4-handshake-concurrency", obfs4listener.DefaultHandshakeConcurrency, "How many concurrent OBFS4 handshakes to process")
	obfs4MaxPendingHandshakesPerClient = flag.Int("obfs4-max-pending-handshakes-per-client", obfs4listener.DefaultMaxPendingHandshakesPerClient, "How many pending OBFS4 handshakes to allow per client")
	obfs4HandshakeTimeout              = flag.Duration("obfs4-handshake-timeout", obfs4listener.DefaultHandshakeTimeout, "How long to wait before timing out an OBFS4 handshake")

	enhttpAddr         = flag.String("enhttp-addr", "", "Address at which to accept encapsulated HTTP requests")
	enhttpServerURL    = flag.String("enhttp-server-url", "", "specify a full URL for domain-fronting to this server with enhttp, required for sticky routing with CloudFront")
	enhttpReapIdleTime = flag.Duration("enhttp-reapidletime", time.Duration(*idleClose)*time.Second, "configure how long enhttp connections are allowed to remain idle before being forcibly closed")

	packetForwardAddr = flag.String("pforward-addr", "", "Address at which to listen for packet forwarding connections")
	packetForwardIntf = flag.String("pforward-intf", "", "The name of the interface to use for upstream packet forwarding connections. Deprecated by external-intf")
	externalIntf      = flag.String("external-intf", "eth0", "The name of the external interface on the host")

	keyfile              = flag.String("key", "", "Private key file name")
	certfile             = flag.String("cert", "", "Certificate file name")
	token                = flag.String("token", "", "Lantern token")
	sessionTicketKeyFile = flag.String("sessionticketkey", "", "File name for storing rotating session ticket keys (deprecated, use -sessionticketkeys instead)")
	sessionTicketKeys    = flag.String("sessionticketkeys", "", "One or more 32 byte session ticket keys, base64 encoded. We will rotate through these every 24 hours. Replaces -sessionticketkey")

	// This flag was added after sessionticketkey (above) to allow the deploying server to configure
	// a key for the proxy without the need to touch its local files. In the interest of backwards
	// compatibility, sessionticketkey was retained and firstSessionTicketKey was implemented to be
	// compatible with sessionticketkey.
	firstSessionTicketKey = flag.String("first-session-ticket-key", "", "initial session ticket key; never expires; 32-byte string, base64-encoded  (deprecated, use -sessionticketkeys instead)")

	lampshadeKeyCacheSize     = flag.Int("lampshade-keycache-size", 0, "set this to a positive value to cache client keys and reject duplicates to thwart replay attacks")
	lampshadeMaxClientInitAge = flag.Duration("lampshade-max-clientinit-age", 0, "set this to a positive value to limit the age of client init messages to thwart replay attacks")

	cfgSvrAuthToken           = flag.String("cfgsvrauthtoken", "", "Token attached to config-server requests, not attaching if empty")
	connectOKWaitsForUpstream = flag.Bool("connect-ok-waits-for-upstream", false, "Set to true to wait for upstream connection before responding OK to CONNECT requests")

	throttleRefreshInterval = flag.Duration("throttlerefresh", throttle.DefaultRefreshInterval, "Specifies how frequently to refresh throttling configuration from redis. Defaults to 5 minutes.")

	enableMultipath = flag.Bool("enablemultipath", false, "Enable multipath. Only clients support multipath can communicate with it.")

	externalIP = flag.String("externalip", "", "The external IP of this proxy, used for reporting")
	https      = flag.Bool("https", false, "Use TLS for client to proxy communication")
	idleClose  = flag.Uint64("idleclose", 70, "Time in seconds that an idle connection will be allowed before closing it")
	_          = flag.Uint64("maxconns", 0, "Max number of simultaneous allowed connections, unused")

	pprofAddr         = flag.String("pprofaddr", "", "pprof address to listen on, not activate pprof if empty")
	maxmindLicenseKey = flag.String("maxmindlicensekey", "", "MaxMind license key to load the GeoLite2 City database")
	geoip2ISPDBFile   = flag.String("geoip2ispdbfile", "", "The local copy of the GeoIP2 ISP database")

	pro = flag.Bool("pro", false, "Set to true to make this a pro proxy (no bandwidth limiting unless forced throttling)")

	proxiedSitesSamplePercentage = flag.Float64("proxied-sites-sample-percentage", 0, "The percentage of requests to sample (0.01 = 1%)")
	proxiedSitesTrackingId       = flag.String("proxied-sites-tracking-id", "UA-21815217-16", "The Google Analytics property id for tracking proxied sites")

	reportingRedisAddr = flag.String("reportingredis", "", "The address of the reporting Redis instance in \"redis[s]://host:port\" format")

	// default value of tunnelPorts matches ports in flashlight/client/client.go
	tunnelPorts         = flag.String("tunnelports", "80,443,22,110,995,143,993,8080,8443,5222,5223,5224,5228,5229,7300,19302,19303,19304,19305,19306,19307,19308,19309", "Comma seperated list of ports allowed for HTTP CONNECT tunnel. Allow all ports if empty.")
	tos                 = flag.Int("tos", 0, "Specify a diffserv TOS to prioritize traffic. Defaults to 0 (off)")
	proxyName           = flag.String("proxyname", hostname, "The name of this proxy (defaults to hostname)")
	proxyProtocol       = flag.String("proxyprotocol", "", "The protocol of this proxy, for information only")
	bbrUpstreamProbeURL = flag.String("bbrprobeurl", "", "optional URL to probe for upstream BBR bandwidth estimates")
	provider            = flag.String("provider", "", "The name of the proxy's provider")
	dc                  = flag.String("dc", "", "The name of the proxy's datacenter")
	frontendProvider    = flag.String("frontendprovider", "", "The name of the provider for the PFE frontend associated with this proxy")
	frontendDC          = flag.String("frontenddc", "", "The name of the datacenter for the PFE frontend associated with this proxy")

	bench   = flag.Bool("bench", false, "Set this flag to set up proxy as a benchmarking proxy. This automatically puts the proxy into tls mode and disables auth token authentication.")
	version = flag.Bool("version", false, "shows the version of the binary")
	help    = flag.Bool("help", false, "Get usage help")

	versionCheck                   = flag.String("versioncheck", "< 7.0.6", "Check if Lantern client matches the semantic version range, like \"< 3.1.1\" or \"<= 3.x\". No check by default.")
	versionCheckRedirectPercentage = flag.Float64("versioncheck-redirect-percentage", .1, "The percentage of requests to be redirected in version check. Defaults to 1 (100%)")

	googleSearchRegex  = flag.String("google-search-regex", googlefilter.DefaultSearchRegex, "Regex for detecting access to Google Search")
	googleCaptchaRegex = flag.String("google-captcha-regex", googlefilter.DefaultCaptchaRegex, "Regex for detecting access to Google captcha page")

	blacklistMaxIdleTime        = flag.Duration("blacklist-max-idle-time", blacklist.DefaultMaxIdleTime, "How long to wait for an HTTP request before considering a connection failed for blacklisting")
	blacklistMaxConnectInterval = flag.Duration("blacklist-max-connect-interval", blacklist.DefaultMaxConnectInterval, "Successive connection attempts within this interval will be treated as a single attempt for blacklisting")
	blacklistAllowedFailures    = flag.Int("blacklist-allowed-failures", blacklist.DefaultAllowedFailures, "The number of failed connection attempts we tolerate before blacklisting an IP address")
	blacklistExpiration         = flag.Duration("blacklist-expiration", blacklist.DefaultExpiration, "How long to wait before removing an ip from the blacklist")

	stackdriverProjectID        = flag.String("stackdriver-project-id", "lantern-http-proxy", "Optional project ID for stackdriver error reporting as in http-proxy-lantern")
	stackdriverCreds            = flag.String("stackdriver-creds", "/home/lantern/lantern-stackdriver.json", "Optional full json file path containing stackdriver credentials")
	stackdriverSamplePercentage = flag.Float64("stackdriver-sample-percentage", 0.0006, "The percentage of devices to report to Stackdriver (0.01 = 1%)")

	requireSessionTickets      = flag.Bool("require-session-tickets", true, "Specifies whether or not to require TLS session tickets in ClientHellos")
	missingTicketReaction      = flag.String("missing-session-ticket-reaction", "None", "Specifies the reaction when seeing ClientHellos without TLS session tickets. Apply only if require-session-tickets is set")
	missingTicketReactionDelay = flag.Duration("missing-session-ticket-reaction-delay", 0, "Specifies the delay before reaction to ClientHellos without TLS session tickets. Apply only if require-session-tickets is set.")
	missingTicketReflectSite   = flag.String("missing-session-ticket-reflect-site", "", "Specifies the site to mirror when seeing no TLS session ticket in ClientHellos. Useful only if missing-session-ticket-reaction is ReflectToSite.")

	tlsListenerAllowTLS13 = flag.Bool("tlslistener-allow-tls13", false, "Allow tlslistener to offer tls13. Because of session ticket issues, this is likely experimental until they can be worked out")

	tlsmasqAddr          = flag.String("tlsmasq-addr", "", "Address at which to listen for tlsmasq connections.")
	tlsmasqOriginAddr    = flag.String("tlsmasq-origin-addr", "", "Address of tlsmasq origin with port.")
	tlsmasqSecret        = flag.String("tlsmasq-secret", "", "Hex encoded 52 byte tlsmasq shared secret.")
	tlsmasqMinVersionStr = flag.String("tlsmasq-tls-min-version", "0x0303", "hex-encoded TLS version")
	tlsmasqSuitesStr     = flag.String("tlsmasq-tls-cipher-suites", "0x1301,0x1302,0x1303,0xcca8,0xcca9,0xc02b,0xc030,0xc02c", "hex-encoded TLS cipher suites")

	starbridgeAddr       = flag.String("starbridge-addr", "", "address at which to listen for starbridge connections")
	starbridgePrivateKey = flag.String("starbridge-private-key", "", "private key for the starbridge server")

	multiplexProtocol    = flag.String("multiplexprotocol", "smux", "multiplexing protocol to use")
	smuxVersion          = flag.Int("smux-version", 0, "smux protocol version")
	smuxMaxFrameSize     = flag.Int("smux-max-frame-size", 0, "smux maximum frame size")
	smuxMaxReceiveBuffer = flag.Int("smux-max-receive-buffer", 0, "smux max receive buffer")
	smuxMaxStreamBuffer  = flag.Int("smux-max-stream-buffer", 0, "smux max stream buffer")

	psmuxVersion                  = flag.Int("psmux-version", 0, "psmux protocol version")
	psmuxMaxFrameSize             = flag.Int("psmux-max-frame-size", 0, "psmux maximum frame size")
	psmuxMaxReceiveBuffer         = flag.Int("psmux-max-receive-buffer", 0, "psmux max receive buffer")
	psmuxMaxStreamBuffer          = flag.Int("psmux-max-stream-buffer", 0, "psmux max stream buffer")
	psmuxMaxPaddingRatio          = flag.Float64("psmux-max-padding-ratio", 0.0, "psmux max padding ratio")
	psmuxMaxPaddedSize            = flag.Int("psmux-max-padded-size", 0, "psmux max padded size")
	psmuxAggressivePadding        = flag.Int("psmux-aggressive-padding", 0, "psmux aggressive padding count")
	psmuxAggressivePaddingRatio   = flag.Float64("psmux-aggressive-padding-ratio", 0, "psmux aggressive padding ratio")
	psmuxDisablePadding           = flag.Bool("psmux-disable-padding", false, "disable all padding")
	psmuxDisableAggressivePadding = flag.Bool("psmux-disable-aggressive-padding", false, "disable aggressive padding only")

	shadowsocksAddr          = flag.String("shadowsocks-addr", "", "Address at which to listen for shadowsocks connections.")
	shadowsocksMultiplexAddr = flag.String("shadowsocks-multiplexaddr", "", "Address at which to listen for multiplexed shadowsocks connections.")
	shadowsocksReplayHistory = flag.Int("shadowsocks-replay-history", shadowsocks.DefaultReplayHistory, "Replay buffer size (# of handshakes)")
	shadowsocksSecret        = flag.String("shadowsocks-secret", "", "shadowsocks secret")
	shadowsocksCipher        = flag.String("shadowsocks-cipher", shadowsocks.DefaultCipher, "shadowsocks cipher")
	shadowsocksWithTLS       = flag.Bool("shadowsocks-with-tls", false, "shadowsocks with tls option")

	tracesSampleRate   = flag.Int("traces-sample-rate", 1000, "rate at which to sample trace data")
	teleportSampleRate = flag.Int("teleport-sample-rate", 1, "rate at which to sample data for Teleport")

	broflakeAddr = flag.String("broflake-addr", "", "Address at which to listen for broflake connections.")

	algenevaAddr = flag.String("algeneva-addr", "", "Address at which to listen for algenAddr connections.")

	waterAddr = flag.String("water-addr", "", "Address at which to listen for WATER connections.")
	waterWASM = flag.String("water-wasm", "", "Base64 encoded WASM for WATER")

	track = flag.String("track", "", "The track this proxy is running on")
)

const (
	cityDBFile = "GeoLite2-City.mmdb"
)

func main() {
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "%s: commit %s built with %s (%s)\n", os.Args[0], revision, runtime.Version(), build_type)
		return
	}
	if *help {
		flag.Usage()
		return
	}

	var reporter *stackdrivererror.Reporter
	if *stackdriverProjectID != "" && *stackdriverCreds != "" {
		reporter = stackdrivererror.Enable(context.Background(), *stackdriverProjectID, *stackdriverCreds, *stackdriverSamplePercentage, *proxyName, *externalIP, *proxyProtocol, *track)
		if reporter != nil {
			defer reporter.Close()
		}
	}

	// panicwrap works by re-executing the running program (retaining arguments,
	// environmental variables, etc.) and monitoring the stderr of the program.
	exitStatus, panicWrapErr := panicwrap.Wrap(
		&panicwrap.WrapConfig{
			DetectDuration: time.Second,
			Handler: func(msg string) {
				if reporter != nil {
					// heuristically separate the error message from the stack trace
					separator := "\ngoroutine "
					splitted := strings.SplitN(msg, separator, 2)
					err := errors.New(splitted[0])
					var maybeStack []byte
					if len(splitted) > 1 {
						maybeStack = []byte(separator + splitted[1])
					}
					reporter.Report(golog.FATAL, err, maybeStack)
				}
				if strings.Contains(msg, "maxmind") {
					log.Debugf("Panic possibly related to maxmind, delete maxmind database files")
					if err := os.Remove(cityDBFile); err != nil {
						log.Errorf("Unable to delete city DB file %v: %v", cityDBFile, err)
					}
					if err := os.Remove(*geoip2ISPDBFile); err != nil {
						log.Errorf("Unable to delete ISP DB file %v: %v", *geoip2ISPDBFile, err)
					}
				}
				os.Exit(1)
			},
			// Just forward signals to the child process
			ForwardSignals: []os.Signal{
				syscall.SIGHUP,
				syscall.SIGTERM,
				syscall.SIGQUIT,
				syscall.SIGINT,
				syscall.SIGUSR1,
			},
		})
	if panicWrapErr != nil {
		log.Fatalf("Error setting up panic wrapper: %v", panicWrapErr)
	} else {
		// If exitStatus >= 0, then we're the parent process.
		if exitStatus >= 0 {
			os.Exit(exitStatus)
		}
	}

	// We're in the child (wrapped) process now

	// Capture signals and exit normally because when relying on the default
	// behavior, exit status -1 would confuse the parent process into thinking
	// it's the child process and keeps running.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for range c {
			log.Debug("Stopping server")
			cancel()
		}
	}()

	if *cfgSvrAuthToken == "" {
		log.Fatal("Config server auth token is required")
	}

	if *lampshadeAddr != "" && !*https {
		log.Fatal("Use of lampshade requires https flag to be true")
	}

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}

	var reaction tlslistener.HandshakeReaction
	switch *missingTicketReaction {
	case "AlertHandshakeFailure":
		reaction = tlslistener.AlertHandshakeFailure
	case "AlertProtocolVersion":
		reaction = tlslistener.AlertProtocolVersion
	case "AlertInternalError":
		reaction = tlslistener.AlertInternalError
	case "CloseConnection":
		reaction = tlslistener.CloseConnection
	case "ReflectToSite":
		if *missingTicketReflectSite == "" {
			log.Fatal("missing-session-ticket-reflect-site should not be empty")
		}
		reaction = tlslistener.ReflectToSite(*missingTicketReflectSite)
		log.Debugf("Reflecting missing session tickets to site %v", *missingTicketReflectSite)
	case "None":
		log.Debug("Not reacting to missing session tickets")
		reaction = tlslistener.None
	default:
		log.Errorf("bad missing-session-ticket-reaction for '%s': '%s', fallback to %s", *proxyProtocol, *missingTicketReaction, reaction.Action())
		reaction = tlslistener.AlertInternalError
	}
	if *missingTicketReactionDelay != 0 {
		reaction = tlslistener.Delayed(*missingTicketReactionDelay, reaction)
	}

	if reaction.Action() == "" {
		log.Debug("Not using missing-session-ticket-reaction")
	} else {
		log.Debugf("Using missing-session-ticket-reaction %v", reaction.Action())
	}

	var (
		tlsmasqTLSMinVersion uint16
		tlsmasqTLSSuites     []uint16
		err                  error
	)
	if *tlsmasqAddr != "" && (*tlsmasqSecret == "" || *tlsmasqOriginAddr == "") {
		log.Fatalf("tlsmasq requires tlsmasq-secret and tlsmasq-origin-addr")
	}
	if *tlsmasqMinVersionStr != "" {
		tlsmasqTLSMinVersion, err = decodeUint16(*tlsmasqMinVersionStr)
		if err != nil {
			log.Fatal(fmt.Sprintln("failed to decode tlsmasq-tls-min-version:", err))
		}
	}
	if *tlsmasqSuitesStr != "" {
		tlsmasqTLSSuites = []uint16{}
		for _, s := range strings.Split(*tlsmasqSuitesStr, ",") {
			suite, err := decodeUint16(s)
			if err != nil {
				log.Fatal(fmt.Sprintln("failed to decode tlsmasq-tls-cipher-suites:", err))
			}
			tlsmasqTLSSuites = append(tlsmasqTLSSuites, suite)
		}
	}

	if *packetForwardIntf != "" {
		*externalIntf = *packetForwardIntf
	}
	mux := *multiplexProtocol
	if mux != "smux" && mux != "psmux" {
		log.Fatalf("unsupported multiplex protocol %v", mux)
	}

	memhelper.Track(15*time.Second, 15*time.Second, func(err error) {
		log.Errorf("error logging memory usage: %v", err)
	})
	go periodicallyForceGC()

	var reportingRedisClient *redis.Client
	if *reportingRedisAddr != "" {
		reportingRedisClient, err = lanternredis.NewClient(*reportingRedisAddr)
		if err != nil {
			log.Errorf("failed to initialize redis client, will not be able to perform bandwidth limiting: %v", err)
		}
	} else {
		log.Debug("no redis address configured for bandwidth reporting")
	}

	p := &proxy.Proxy{
		HTTPAddr:                           *addr,
		HTTPMultiplexAddr:                  *multiplexAddr,
		CertFile:                           *certfile,
		CfgSvrAuthToken:                    *cfgSvrAuthToken,
		ConnectOKWaitsForUpstream:          *connectOKWaitsForUpstream,
		EnableMultipath:                    *enableMultipath,
		ThrottleRefreshInterval:            *throttleRefreshInterval,
		TracesSampleRate:                   *tracesSampleRate,
		TeleportSampleRate:                 *teleportSampleRate,
		ExternalIP:                         *externalIP,
		HTTPS:                              *https,
		IdleTimeout:                        time.Duration(*idleClose) * time.Second,
		KeyFile:                            *keyfile,
		SessionTicketKeys:                  *sessionTicketKeys,
		SessionTicketKeyFile:               *sessionTicketKeyFile,
		FirstSessionTicketKey:              *firstSessionTicketKey,
		Track:                              *track,
		Pro:                                *pro,
		ProxiedSitesSamplePercentage:       *proxiedSitesSamplePercentage,
		ProxiedSitesTrackingID:             *proxiedSitesTrackingId,
		ReportingRedisClient:               reportingRedisClient,
		Token:                              *token,
		TunnelPorts:                        *tunnelPorts,
		Obfs4Addr:                          *obfs4Addr,
		Obfs4MultiplexAddr:                 *obfs4MultiplexAddr,
		Obfs4Dir:                           *obfs4Dir,
		Obfs4HandshakeConcurrency:          *obfs4HandshakeConcurrency,
		Obfs4MaxPendingHandshakesPerClient: *obfs4MaxPendingHandshakesPerClient,
		Obfs4HandshakeTimeout:              *obfs4HandshakeTimeout,
		KCPConf:                            *kcpConf,
		ENHTTPAddr:                         *enhttpAddr,
		ENHTTPServerURL:                    *enhttpServerURL,
		ENHTTPReapIdleTime:                 *enhttpReapIdleTime,
		Benchmark:                          *bench,
		DiffServTOS:                        *tos,
		LampshadeAddr:                      *lampshadeAddr,
		LampshadeKeyCacheSize:              *lampshadeKeyCacheSize,
		LampshadeMaxClientInitAge:          *lampshadeMaxClientInitAge,
		GoogleSearchRegex:                  *googleSearchRegex,
		GoogleCaptchaRegex:                 *googleCaptchaRegex,
		BlacklistMaxIdleTime:               *blacklistMaxIdleTime,
		BlacklistMaxConnectInterval:        *blacklistMaxConnectInterval,
		BlacklistAllowedFailures:           *blacklistAllowedFailures,
		BlacklistExpiration:                *blacklistExpiration,
		ProxyName:                          *proxyName,
		ProxyProtocol:                      *proxyProtocol,
		Provider:                           *provider,
		DC:                                 *dc,
		FrontendProvider:                   *frontendProvider,
		FrontendDC:                         *frontendDC,
		BuildType:                          build_type,
		BBRUpstreamProbeURL:                *bbrUpstreamProbeURL,
		QUICIETFAddr:                       *quicIETFAddr,
		QUICUseBBR:                         *quicBBR,
		WSSAddr:                            *wssAddr,
		PacketForwardAddr:                  *packetForwardAddr,
		ExternalIntf:                       *externalIntf,
		RequireSessionTickets:              *requireSessionTickets,
		MissingTicketReaction:              reaction,
		TLSListenerAllowTLS13:              *tlsListenerAllowTLS13,
		TLSMasqAddr:                        *tlsmasqAddr,
		TLSMasqOriginAddr:                  *tlsmasqOriginAddr,
		TLSMasqSecret:                      *tlsmasqSecret,
		TLSMasqTLSMinVersion:               tlsmasqTLSMinVersion,
		TLSMasqTLSCipherSuites:             tlsmasqTLSSuites,
		ShadowsocksAddr:                    *shadowsocksAddr,
		ShadowsocksMultiplexAddr:           *shadowsocksMultiplexAddr,
		ShadowsocksSecret:                  *shadowsocksSecret,
		ShadowsocksCipher:                  *shadowsocksCipher,
		ShadowsocksReplayHistory:           *shadowsocksReplayHistory,
		ShadowsocksWithTLS:                 *shadowsocksWithTLS,
		StarbridgeAddr:                     *starbridgeAddr,
		StarbridgePrivateKey:               *starbridgePrivateKey,
		MultiplexProtocol:                  *multiplexProtocol,
		SmuxVersion:                        *smuxVersion,
		SmuxMaxFrameSize:                   *smuxMaxFrameSize,
		SmuxMaxReceiveBuffer:               *smuxMaxReceiveBuffer,
		SmuxMaxStreamBuffer:                *smuxMaxStreamBuffer,
		PsmuxVersion:                       *psmuxVersion,
		PsmuxMaxFrameSize:                  *psmuxMaxFrameSize,
		PsmuxMaxReceiveBuffer:              *psmuxMaxReceiveBuffer,
		PsmuxMaxStreamBuffer:               *psmuxMaxStreamBuffer,
		PsmuxDisablePadding:                *psmuxDisablePadding,
		PsmuxMaxPaddingRatio:               *psmuxMaxPaddingRatio,
		PsmuxMaxPaddedSize:                 *psmuxMaxPaddedSize,
		PsmuxDisableAggressivePadding:      *psmuxDisableAggressivePadding,
		PsmuxAggressivePadding:             *psmuxAggressivePadding,
		PsmuxAggressivePaddingRatio:        *psmuxAggressivePaddingRatio,
		BroflakeAddr:                       *broflakeAddr,
		BroflakeCert:                       os.Getenv("BROFLAKE_CERT"),
		BroflakeKey:                        os.Getenv("BROFLAKE_KEY"),
		AlgenevaAddr:                       *algenevaAddr,
		WaterAddr:                          *waterAddr,
		WaterWASM:                          *waterWASM,
	}
	if *maxmindLicenseKey != "" {
		log.Debug("Will use Maxmind for geolocating clients")
		if err := deleteStaleISPDB(); err != nil {
			log.Errorf("Error deleting stale ISP DB, ignore: %v", err)
		}
		p.CountryLookup = geo.FromWeb(geolite2CityURL, "GeoLite2-City.mmdb", 24*time.Hour, cityDBFile, geo.CountryCode)
		p.ISPLookup = geo.FromWeb(geoip2ISPURL, "GeoIP2-ISP.mmdb", 24*time.Hour, *geoip2ISPDBFile, geo.ISP)
	}

	err = p.ListenAndServe(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func periodicallyForceGC() {
	for {
		time.Sleep(1 * time.Minute)
		debug.FreeOSMemory()
	}
}

func decodeUint16(s string) (uint16, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

// Salt has been distributing an out-of-date ISP database. Unfortunately, that database gets a recent timestamp,
// so the logic that checks to see if there's a newer version available online things there isn't, and so
// the proxy keeps using a stale database.
// Deleting the stale database from our thousands of proxies via Salt is time consuming and error prone,
// so instead we just check for it here and delete it if necessary.
func deleteStaleISPDB() error {
	shasum := sha1.New()
	file, err := os.Open(*geoip2ISPDBFile)
	if err != nil {
		return err
	}
	defer file.Close()
	b := make([]byte, 16000)
	for {
		n, err := file.Read(b)
		if err != nil {
			break
		}
		shasum.Write(b[:n])
	}
	file.Close()
	if hex.EncodeToString(shasum.Sum(nil)) == "bc63a41e2418763fbe03b79a7896cea7f4d9b06d" {
		log.Debug("Deleting stale ISP DB")
		return os.Remove(file.Name())
	}
	return nil
}
