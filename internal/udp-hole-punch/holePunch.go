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
	MaxAttempts   int
}

// hole puncher struct contains all required information to perform udp hole punch. May be updated to include TCP hole punch (not necessary for this project)
type HolePuncher struct {
	config            HolePunchConfig
	socket            *net.UDPConn
	peerLocalAddress  *net.UDPAddr
	peerRemoteAddress *net.UDPAddr
	resultCh          chan HolePunchResult
}

// struct that will be returned to start send chunks
type HolePunchResult struct {
	Addr *net.UDPAddr
	Err  error
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 10 * time.Second, ProbeInterval: 150 * time.Millisecond, MaxAttempts: 20}
}

func UDPHolePunch(conn *net.UDPConn, peerLocal *net.UDPAddr, peerRemote *net.UDPAddr) <-chan HolePunchResult {
	defaultConfig := getDefaultConfig()
	punch := &HolePuncher{socket: conn, config: defaultConfig,
		peerLocalAddress: peerLocal, peerRemoteAddress: peerRemote}
	go punch.tryPunch()
	go punch.listenPeer()

	return punch.resultCh
}

// First-to-Respond principle: local network is faster than stun
func (punch *HolePuncher) tryPunch() {
	// simple message SYN - ACK for hole punch
	// 1 peer send SYN, 2 peer reply with ACK
	// TODO : EXPLICAR ISTO COMO FUNCIONA
	go punch.probe(punch.peerLocalAddress)
	go punch.probe(punch.peerRemoteAddress)

}

func (punch *HolePuncher) probe(addr *net.UDPAddr) {
	probeMessage := []byte(punchRequest)
	for i := 0; i < punch.config.MaxAttempts; i++ {

		_, err := punch.socket.WriteToUDP(probeMessage, addr)

		if err == nil { // the address that work will go to channel
			result := HolePunchResult{Addr: addr, Err: nil}
			punch.resultCh <- result
		}
		_ = err // some probes will fail, that's expected
		time.Sleep(punch.config.ProbeInterval)
	}

}

func (punch *HolePuncher) listenPeer() {

}
