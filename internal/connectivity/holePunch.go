package udpholepunch

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
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
	config             HolePunchConfig
	socket             *net.UDPConn
	peerLocalAddress   *net.UDPAddr
	peerRemoteAddress  *net.UDPAddr
	myPublicKey        []byte
	otherPeerPublicKey []byte
	resultCh           chan HolePunchResult
}

// struct that will be returned to start send chunks
type HolePunchResult struct {
	Addr *net.UDPAddr
	Rtt  time.Duration
	Err  error
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 60 * time.Second, ProbeInterval: 200 * time.Millisecond, ListenInterval: 500 * time.Millisecond, MaxAttempts: 10}
}

func UDPHolePunch(conn *net.UDPConn, peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) <-chan HolePunchResult {
	defaultConfig := getDefaultConfig()
	publicKey := privateKey.Public().(ed25519.PublicKey)
	punch := &HolePuncher{socket: conn, config: defaultConfig,
		peerLocalAddress: peerLocal, peerRemoteAddress: peerRemote,
		otherPeerPublicKey: *otherPeerPublicKey, myPublicKey: publicKey, resultCh: make(chan HolePunchResult)}
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
	go punch.probe(punch.peerLocalAddress, punch.myPublicKey)
	go punch.probe(punch.peerRemoteAddress, punch.myPublicKey)

}

func (punch *HolePuncher) probe(addr *net.UDPAddr, myPublicKey []byte) {
	// binary protocol : 1 byte (PUNCH), 32 bytes (public key)
	probeMessage := make([]byte, 33)
	probeMessage[0] = PUNCH
	copy(probeMessage[1:33], myPublicKey)
	for i := 0; i < punch.config.MaxAttempts; i++ {

		_, err := punch.socket.WriteToUDP(probeMessage, addr)

		if err != nil {
			_ = err // some probes will fail, that's expected
			continue
		}
		time.Sleep(punch.config.ProbeInterval)
		// the address that work will go to channel
	}

}

func (punch *HolePuncher) listenPeer() {
	buffer := make([]byte, 256)
	for {
		punch.socket.SetReadDeadline(time.Now().Add(punch.config.Timeout))
		n, addr, err := punch.socket.ReadFromUDP(buffer)

		// if error type is timeout, a null address will be returned to the channel
		if err != nil && os.IsTimeout(err) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: err}
			return
		}

		// test if is the same address
		if isMyOwnPC(&punch.myPublicKey, &punch.otherPeerPublicKey) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: ErrSameAdress}
			return
		}

		msg := buffer[:n]

		//otherwise continue looping
		if n < 5 || msg[0] != PUNCH {
			continue
		}
		isExpectedToken := bytes.Equal(msg[1:33], punch.otherPeerPublicKey)

		if isExpectedToken {
			fmt.Printf("PUNCH RECEIVED BY: %s\n", addr.IP)
			punch.resultCh <- HolePunchResult{Addr: addr, Err: nil}
			return

		}
	}

}

func isMyOwnPC(myToken *[]byte, expectedToken *[]byte) bool {
	// verify if the target token is the same as pc token
	return bytes.Equal(*myToken, *expectedToken)
}
