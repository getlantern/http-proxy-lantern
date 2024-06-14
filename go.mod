module github.com/getlantern/http-proxy-lantern/v2

go 1.21

require (
	cloud.google.com/go/errorreporting v0.3.0
	git.torproject.org/pluggable-transports/goptlib.git v1.2.0
	github.com/Jigsaw-Code/outline-sdk v0.0.16
	github.com/Jigsaw-Code/outline-ss-server v1.5.0
	github.com/OperatorFoundation/Replicant-go/Replicant/v3 v3.0.23
	github.com/OperatorFoundation/Starbridge-go/Starbridge/v3 v3.0.17
	github.com/dustin/go-humanize v1.0.1
	github.com/getlantern/broflake v0.0.0-20231117182649-7d46643a6f87
	github.com/getlantern/cmux/v2 v2.0.0-20230301223233-dac79088a4c0
	github.com/getlantern/cmuxprivate v0.0.0-20211216020409-d29d0d38be54
	github.com/getlantern/enhttp v0.0.0-20210901195634-6f89d45ee033
	github.com/getlantern/errors v1.0.4
	github.com/getlantern/fdcount v0.0.0-20210503151800-5decd65b3731
	github.com/getlantern/geo v0.0.0-20230612145351-d1374c8f8dec
	github.com/getlantern/golog v0.0.0-20230503153817-8e72de7e0a65
	github.com/getlantern/gonat v0.0.0-20201001145726-634575ba87fb
	github.com/getlantern/grtrack v0.0.0-20231025115619-bfbfadb228f3
	github.com/getlantern/idletiming v0.0.0-20200228204104-10036786eac5
	github.com/getlantern/iptool v0.0.0-20230112135223-c00e863b2696
	github.com/getlantern/kcpwrapper v0.0.0-20230327091313-c12d7c17c6de
	github.com/getlantern/keyman v0.0.0-20230503155501-4e864ca2175b
	github.com/getlantern/lampshade v0.0.0-20200303040944-fe53f13203e9
	github.com/getlantern/lantern-algeneva v0.0.0-20240418193310-610690afddbc
	github.com/getlantern/measured v0.0.0-20230919230611-3d9e3776a6cd
	github.com/getlantern/memhelper v0.0.0-20220104170102-df557102babd
	github.com/getlantern/mockconn v0.0.0-20200818071412-cb30d065a848
	github.com/getlantern/multipath v0.0.0-20230510135141-717ed305ef50
	github.com/getlantern/netx v0.0.0-20211206143627-7ccfeb739cbd
	github.com/getlantern/ops v0.0.0-20230424193308-26325dfed3cf
	github.com/getlantern/packetforward v0.0.0-20201001150407-c68a447b0360
	github.com/getlantern/proxy/v3 v3.0.0-20240328103708-9185589b6a99
	github.com/getlantern/psmux v1.5.15
	github.com/getlantern/quicwrapper v0.0.0-20231117185542-d951689c4970
	github.com/getlantern/ratelimit v0.0.0-20220926192648-933ab81a6fc7
	github.com/getlantern/tinywss v0.0.0-20211216020538-c10008a7d461
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsmasq v0.4.7-0.20230302000139-6e479a593298
	github.com/getlantern/tlsutil v0.5.3
	github.com/getlantern/waitforserver v1.0.1
	github.com/getlantern/withtimeout v0.0.0-20160829163843-511f017cd913
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/hashicorp/golang-lru v0.5.4
	github.com/mitchellh/panicwrap v1.0.0
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/refraction-networking/utls v1.3.3
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726
	github.com/spaolacci/murmur3 v1.1.0
	github.com/stretchr/testify v1.8.4
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	github.com/xtaci/smux v1.5.24
	gitlab.com/yawning/obfs4.git v0.0.0-20220204003609-77af0cba934d
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.42.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.19.0
	go.opentelemetry.io/otel/metric v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
	go.opentelemetry.io/otel/sdk/metric v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
	golang.org/x/net v0.20.0
	google.golang.org/api v0.148.0
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/OperatorFoundation/ghostwriter-go v1.0.6 // indirect
	github.com/OperatorFoundation/go-bloom v1.0.1 // indirect
	github.com/OperatorFoundation/go-shadowsocks2 v1.2.1 // indirect
	github.com/Yawning/chacha20 v0.0.0-20170904085104-e3b1f968fc63 // indirect
	github.com/aead/ecdh v0.2.0 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/dvyukov/go-fuzz v0.0.0-20210429054444-fca39067bc72 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/gaukas/godicttls v0.0.4 // indirect
	github.com/getlantern/algeneva v0.0.0-20240222191137-2b4e88234f59 // indirect
	github.com/getlantern/bufconn v0.0.0-20190625204133-a08544339f8d // indirect
	github.com/getlantern/byteexec v0.0.0-20220903142956-e6ed20032cfd // indirect
	github.com/getlantern/cmux v0.0.0-20230301223233-dac79088a4c0 // indirect
	github.com/getlantern/context v0.0.0-20220418194847-3d5e7a086201 // indirect
	github.com/getlantern/elevate v0.0.0-20220903142053-479ab992b264 // indirect
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4 // indirect
	github.com/getlantern/eventual v0.0.0-20180125201821-84b02499361b // indirect
	github.com/getlantern/filepersist v0.0.0-20210901195658-ed29a1cb0b7c // indirect
	github.com/getlantern/framed v0.0.0-20190601192238-ceb6431eeede // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20201229170000-e66e7f878730 // indirect
	github.com/getlantern/kcp-go/v5 v5.0.0-20220503142114-f0c1cd6e1b54 // indirect
	github.com/getlantern/keepcurrent v0.0.0-20221014183517-fcee77376b89 // indirect
	github.com/getlantern/mtime v0.0.0-20200417132445-23682092d1f7 // indirect
	github.com/getlantern/preconn v1.0.0 // indirect
	github.com/getlantern/telemetry v0.0.0-20230523155019-be7c1d8cd8cb // indirect
	github.com/getlantern/uuid v1.2.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20230821062121-407c9e7a662f // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.1 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/klauspost/reedsolomon v1.9.9 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/mdlayher/netlink v1.1.0 // indirect
	github.com/mholt/archiver/v3 v3.5.1 // indirect
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/nwaples/rardecode v1.1.2 // indirect
	github.com/onsi/ginkgo/v2 v2.12.0 // indirect
	github.com/oschwald/geoip2-golang v1.8.0 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pierrec/lz4/v4 v4.1.12 // indirect
	github.com/pion/datachannel v1.5.5 // indirect
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/ice/v2 v2.3.5 // indirect
	github.com/pion/interceptor v0.1.17 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.7 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.10 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sctp v1.8.7 // indirect
	github.com/pion/sdp/v3 v3.0.6 // indirect
	github.com/pion/srtp/v2 v2.0.15 // indirect
	github.com/pion/stun v0.6.0 // indirect
	github.com/pion/transport/v2 v2.2.1 // indirect
	github.com/pion/turn/v2 v2.1.0 // indirect
	github.com/pion/webrtc/v3 v3.2.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/quic-go/quic-go v0.42.0 // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.5 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	github.com/templexxx/cpu v0.0.8 // indirect
	github.com/templexxx/xorsimd v0.4.1 // indirect
	github.com/ti-mo/conntrack v0.3.0 // indirect
	github.com/ti-mo/netfilter v0.3.1 // indirect
	github.com/tjfoc/gmsm v1.3.2 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	gitlab.com/yawning/edwards25519-extra.git v0.0.0-20211229043746-2f91fcc9fbdb // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/exp v0.0.0-20240110193028-0dcbfd608b1e // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231012201019-e917dd12ba7a // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.10 // indirect
)

// Waiting on https://github.com/mitchellh/panicwrap/pull/27 to be merged upstream
replace github.com/mitchellh/panicwrap v1.0.0 => github.com/getlantern/panicwrap v0.0.0-20200707191944-9ba45baf8e51
