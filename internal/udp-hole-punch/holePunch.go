package udpholepunch

import (
	"net"
	"time"
)

var punchRequest = "SYN"
var punchResponse = "ACK"

// based on https://medium.com/@knightkun/udp-hole-punching-from-scratch-in-golang-building-tsuna-448ca66a733b
type HolePunchConfig struct {
	Timeout       time.Duration
	ProbeInterval time.Duration
	MaxAttempts   int8
}

type HolePunchProtocol struct {
	config            HolePunchConfig
	socket            *net.UDPConn
	peerLocalAddress  *net.UDPAddr
	peerRemoteAddress *net.UDPAddr
	resultCh          chan error
}

func UDPHolePunch(conn *net.UDPConn, peerLocal *net.UDPAddr, peerRemote *net.UDPAddr) (result chan error) {
	defaultConfig := getDefaultConfig()
	proto := HolePunchProtocol{socket: conn, config: defaultConfig,
		peerLocalAddress: peerLocal, peerRemoteAddress: peerRemote}
	go tryPunch(proto)
	go listenPeer(proto)

	return proto.resultCh
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 10 * time.Second, ProbeInterval: 150 * time.Millisecond, MaxAttempts: 20}
}

func tryPunch(proto HolePunchProtocol) {

}

func listenPeer(proto HolePunchProtocol) {

}
