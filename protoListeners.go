package proxy

type protoListenerArgs struct {
	protocol string
	addr     string
	fn       listenerBuilderFN
}

func getProtoListenersArgs(p *Proxy) []protoListenerArgs {
	return []protoListenerArgs{
		/**********   listeners for base transport   **********/
		{"https", p.HTTPAddr, p.wrapTLSIfNecessary(p.listenHTTP(p.listenTCP))},
		{
			"https_multiplex",
			p.HTTPMultiplexAddr,
			p.wrapMultiplexing(p.wrapTLSIfNecessary(p.listenHTTP(p.listenTCP))),
		},
		{"tlsmasq", p.TLSMasqAddr, p.wrapMultiplexing(p.listenTLSMasq(p.listenTCP))},
		{"starbridge", p.StarbridgeAddr, p.wrapMultiplexing(p.listenStarbridge(p.listenTCP))},
		{"broflake", p.BroflakeAddr, p.listenBroflake(p.listenTCP)},
		{"algeneva", p.AlgenevaAddr, p.wrapMultiplexing(p.listenAlgeneva(p.listenTCP))},
		/******************************************************/

		{"kcp", p.KCPConf, p.wrapTLSIfNecessary(p.listenKCP)},
		{"quic_ietf", p.QUICIETFAddr, p.listenQUICIETF},
		{"shadowsocks", p.ShadowsocksAddr, p.listenShadowsocks},
		{
			"shadowsocks_multiplex",
			p.ShadowsocksMultiplexAddr,
			p.wrapMultiplexing(p.listenShadowsocks),
		},
		{"water", p.WaterAddr, p.wrapMultiplexing(p.listenWATER)},
	}
}
