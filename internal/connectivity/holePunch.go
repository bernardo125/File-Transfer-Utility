package connectivity

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"net"
	"os"
	"time"
)

const (
	PUNCH byte = 0x01
)

var ErrSameAdress = errors.New("That's your PC token")

// based on https://medium.com/@knightkun/udp-hole-punching-from-scratch-in-golang-building-tsuna-448ca66a733b
// configuration of punch to fail
type HolePunchConfig struct {
	Timeout        time.Duration
	ProbeInterval  time.Duration
	ListenInterval time.Duration
	MaxAttempts    int
}

// hole puncher struct contains all required information to perform udp hole punch. May be updated to include TCP hole punch (not necessary for this project)
type HolePuncher struct {
	config                HolePunchConfig
	socket                *net.UDPConn
	peerLocalAddressIPV4  *net.UDPAddr
	peerRemoteAddressIPV4 *net.UDPAddr
	peerPublicAddressIPV6 *net.UDPAddr
	myPublicKey           []byte
	otherPeerPublicKey    []byte
	resultCh              chan HolePunchResult
	done                  chan struct{} // closed when the punch finished; stops all probes
}

// struct that will be returned to start send chunks
type HolePunchResult struct {
	Addr *net.UDPAddr
	Rtt  time.Duration
	Err  error
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 60 * time.Second, ProbeInterval: 300 * time.Millisecond, ListenInterval: 500 * time.Millisecond, MaxAttempts: 100}
}

func UDPHolePunch(conn *net.UDPConn, peerLocalIPV4 *net.UDPAddr, peerRemoteIPV4 *net.UDPAddr, peerPublicIPV6 *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) <-chan HolePunchResult {
	return udpHolePunch(getDefaultConfig(), conn, peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, privateKey, otherPeerPublicKey)
}

// udpHolePunch is UDPHolePunch with an injectable configuration (lets tests use short timeouts).
func udpHolePunch(defaultConfig HolePunchConfig, conn *net.UDPConn, peerLocalIPV4 *net.UDPAddr, peerRemoteIPV4 *net.UDPAddr, peerPublicIPV6 *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) <-chan HolePunchResult {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	punch := &HolePuncher{socket: conn, config: defaultConfig,
		peerLocalAddressIPV4: peerLocalIPV4, peerRemoteAddressIPV4: peerRemoteIPV4, peerPublicAddressIPV6: peerPublicIPV6,
		otherPeerPublicKey: *otherPeerPublicKey, myPublicKey: publicKey, resultCh: make(chan HolePunchResult, 1), done: make(chan struct{})}

	// test if is the same address
	if isMyOwnPC(&punch.myPublicKey, &punch.otherPeerPublicKey) {
		punch.resultCh <- HolePunchResult{Addr: nil, Err: ErrSameAdress}
		return punch.resultCh
	}
	//goroutine - try to punch and listen for udp packets (async)
	go punch.tryPunch()
	//when resultCh channel receive data or timeout this func will return the result to main app
	go punch.listenPeer()

	return punch.resultCh
}

// First-to-Respond principle: local network is faster than stun
func (punch *HolePuncher) tryPunch() {
	// simple message PUNCH to enable NAT mapping
	// this routine separates on 2 routines that will try to probe locally and stun
	go punch.probe(punch.peerLocalAddressIPV4, punch.myPublicKey)
	go punch.probe(punch.peerRemoteAddressIPV4, punch.myPublicKey)
	if punch.peerPublicAddressIPV6 != nil {
		go punch.probe(punch.peerPublicAddressIPV6, punch.myPublicKey)
	}

}

func (punch *HolePuncher) probe(addr *net.UDPAddr, myPublicKey []byte) {
	// binary protocol : 1 byte (PUNCH), 32 bytes (public key)
	probeMessage := make([]byte, 33)
	probeMessage[0] = PUNCH
	copy(probeMessage[1:33], myPublicKey)
	if addr == nil {
		return
	}
	ticker := time.NewTicker(punch.config.ProbeInterval)
	defer ticker.Stop()
	for i := 0; i < punch.config.MaxAttempts; i++ {
		// some probes will fail, that's expected; keep pacing so a failing route is not hammered
		_, _ = punch.socket.WriteToUDP(probeMessage, addr)
		select {
		case <-punch.done:
			return
		case <-ticker.C:
		}
	}

}

func (punch *HolePuncher) listenPeer() {
	defer close(punch.done)                         // stop probing as soon as we have a verdict
	defer punch.socket.SetReadDeadline(time.Time{}) // socket without time out to QUIC
	buffer := make([]byte, 256)

	for {
		punch.socket.SetReadDeadline(time.Now().Add(punch.config.Timeout))
		n, addr, err := punch.socket.ReadFromUDP(buffer)

		// if error type is timeout, a null address will be returned to the channel
		if err != nil && os.IsTimeout(err) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: err}
			return
		}

		// a closed socket never recovers: report it instead of spinning forever
		if err != nil && errors.Is(err, net.ErrClosed) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: err}
			return
		}
		msg := buffer[:n]

		//otherwise continue looping
		if n < 33 || msg[0] != PUNCH {
			continue
		}
		isExpectedToken := bytes.Equal(msg[1:33], punch.otherPeerPublicKey)

		if isExpectedToken {
			punch.socket.SetDeadline(time.Time{}) //clean deadline
			punch.resultCh <- HolePunchResult{Addr: addr, Err: nil}
			return

		}
	}

}

func isMyOwnPC(myToken *[]byte, expectedToken *[]byte) bool {
	// verify if the target token is the same as pc token
	return bytes.Equal(*myToken, *expectedToken)
}
