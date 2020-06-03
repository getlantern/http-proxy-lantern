package packet_counter

import (
	"errors"
	"fmt"
	"net"

	"github.com/getlantern/golog"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	snaplen int32 = 18 + 60 + 60 // The total of maximum length of Ethernet, IP and TCP headers
)

var (
	log = golog.LoggerFor("packet_counter")
)

// ReportFN is a callback to report how many packets have been sent over a TCP
// connection made from the clientAddr and of which how many are
// retransmissions. It gets called when the connection terminates.
type ReportFN func(clientAddr string, packets, retransmissions int)

// Track keeps capturing all TCP replies from the listening port on the
// interface, and reports when the connection terminates.
func Track(interfaceName, listenPort string, report ReportFN) {
	addrs, err := interfaceAddrs(interfaceName)
	if err != nil {
		log.Errorf("Unable to open %v for packet capture: %v", interfaceName, err)
		return
	}
	filter, err := composeBPF(addrs, listenPort)
	if err != nil {
		log.Errorf("Unable to compose BPF %v for packet capture: %v", interfaceName, err)
		return
	}
	handle, err := pcap.OpenLive(interfaceName, snaplen, false /*promisc*/, pcap.BlockForever)
	if err != nil {
		log.Errorf("Unable to open %v for packet capture: %v", interfaceName, err)
		return
	}
	if err := handle.SetBPFFilter(filter); err != nil {
		log.Errorf("Unable to set BPF filter '%v': %v", filter, err)
		return
	}

	// Map of the string form of the TCPAddr to the counters
	flows := map[string]struct {
		lastSeq         uint32
		packets         int
		retransmissions int
	}{}
	var ether layers.Ethernet
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var tcp layers.TCP
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet,
		&ether, &ip4, &ip6, &tcp)
	decoded := make([]gopacket.LayerType, 0, 4)

	for {
		data, _, err := handle.ZeroCopyReadPacketData()
		if err != nil {
			log.Debugf("error getting packet: %v", err)
			continue
		}
		// error is expected because we don't decode TLS. Ranging over decoded
		// would get correct result.
		_ = parser.DecodeLayers(data, &decoded)
		var dst net.TCPAddr
		var payloadSize uint16
		var tcpDecoded bool
		for _, typ := range decoded {
			switch typ {
			case layers.LayerTypeIPv4:
				dst.IP = ip4.DstIP
				payloadSize = ip4.Length - uint16(ip4.IHL<<2)
			case layers.LayerTypeIPv6:
				dst.IP = ip6.DstIP
				payloadSize = ip6.Length
			case layers.LayerTypeTCP:
				tcpDecoded = true
			}
		}
		if !tcpDecoded {
			log.Error("TCP packet is expected but not seen")
			continue
		}
		length := payloadSize - uint16(tcp.DataOffset<<2)
		// skip pure ACKs
		if length == 0 && !tcp.SYN && !tcp.RST && !tcp.FIN {
			continue
		}
		dst.Port = int(tcp.DstPort)
		key := dst.String()
		if tcp.FIN || tcp.RST {
			flow := flows[key]
			if flow.packets > 0 {
				report(key, flow.packets, flow.retransmissions)
			}
			delete(flows, key)
			continue
		}
		flow := flows[key]
		flow.packets++
		if tcp.Seq > flow.lastSeq {
			flow.lastSeq = tcp.Seq
		} else {
			// Note that ACKs to SYNs and FINs are miscounted as
			// retransmissions but is acceptable in this case.
			flow.retransmissions++
		}
		flows[key] = flow
	}
}

func interfaceAddrs(interfaceName string) ([]net.Addr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		if i.Name == interfaceName {
			return i.Addrs()
		}
	}
	return nil, errors.New("interface not found")
}

func composeBPF(addrs []net.Addr, listenPort string) (string, error) {
	switch len(addrs) {
	case 0:
		return "", errors.New("no address is configured on interface")
	case 1:
		return fmt.Sprintf("tcp and src port %s and src host %s", listenPort, addrs[0].String()), nil
	default:
		str := fmt.Sprintf("tcp and src port %s and (src host %s", listenPort, addrs[0].String())
		for _, addr := range addrs[1:] {
			str += " or src host " + addr.String()
		}
		str += ")"
		return str, nil
	}
}
