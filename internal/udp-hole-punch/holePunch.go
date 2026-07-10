package udpholepunch

import (
	"net"
	"time"
)

var punchRequest = "SYN"
var punchResponse = "ACK"

// based on https://medium.com/@knightkun/udp-hole-punching-from-scratch-in-golang-building-tsuna-448ca66a733b
// configuration of punch to fail
type HolePunchConfig struct {
	Timeout       time.Duration
	ProbeInterval time.Duration
	MaxAttempts   int8
}

// hole puncher struct contains all required information to perform udp hole punch. May be updated to include TCP hole punch (not necessary for this project)
type HolePuncher struct {
	config            HolePunchConfig
	socket            *net.UDPConn
	peerLocalAddress  *net.UDPAddr
	peerRemoteAddress *net.UDPAddr
	resultCh          chan error
}

func UDPHolePunch(conn *net.UDPConn, peerLocal *net.UDPAddr, peerRemote *net.UDPAddr) (result chan error) {
	defaultConfig := getDefaultConfig()
	proto := &HolePuncher{socket: conn, config: defaultConfig,
		peerLocalAddress: peerLocal, peerRemoteAddress: peerRemote}
	go tryPunch(proto)
	go listenPeer(proto)

	return proto.resultCh
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 10 * time.Second, ProbeInterval: 150 * time.Millisecond, MaxAttempts: 20}
}

func tryPunch(proto *HolePuncher) {
	// simple message SYN - ACK for hole punch
	// 1 peer send SYN, 2 peer reply with ACK
	// TODO : EXPLICAR ISTO COMO FUNCIONA
	go proto.probe(proto.peerLocalAddress)
	go proto.probe(proto.peerRemoteAddress)

}

func (proto *HolePuncher) probe(addr *net.UDPAddr) {
	probeMessage := []byte("SYN")
	proto.socket.WriteToUDP(probeMessage, addr)
}

func listenPeer(proto *HolePuncher) {

}
