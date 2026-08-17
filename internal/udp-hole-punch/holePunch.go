package udpholepunch

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

const (
	PunchSYN byte = 0x01
	PunchACK byte = 0x02
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
	config            HolePunchConfig
	socket            *net.UDPConn
	peerLocalAddress  *net.UDPAddr
	peerRemoteAddress *net.UDPAddr
	myToken           []byte
	expectedToken     []byte
	resultCh          chan HolePunchResult
}

// struct that will be returned to start send chunks
type HolePunchResult struct {
	Addr *net.UDPAddr
	Rtt  time.Duration
	Err  error
}

func getDefaultConfig() (config HolePunchConfig) {
	return HolePunchConfig{Timeout: 10 * time.Second, ProbeInterval: 200 * time.Millisecond, ListenInterval: 500 * time.Millisecond, MaxAttempts: 200}
}

func UDPHolePunch(conn *net.UDPConn, peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, mytoken *[]byte, expectedToken *[]byte) <-chan HolePunchResult {
	defaultConfig := getDefaultConfig()
	punch := &HolePuncher{socket: conn, config: defaultConfig,
		peerLocalAddress: peerLocal, peerRemoteAddress: peerRemote,
		expectedToken: *expectedToken, myToken: *mytoken, resultCh: make(chan HolePunchResult)}
	//goroutine - try to punch and listen for udp packets (async)
	go punch.tryPunch(mytoken)
	//when resultCh channel receive data or timeout this func will return the result to main app
	go punch.listenPeer(mytoken, expectedToken)

	return punch.resultCh
}

// First-to-Respond principle: local network is faster than stun
func (punch *HolePuncher) tryPunch(mytoken *[]byte) {
	// TODO : dynamic 4 byte key to avoid spoofing
	// simple message SYN - ACK for hole punch
	// 1 peer send SYN, 2 peer reply with ACK
	// this routine separates on 2 routines that will try to probe locally and stun
	go punch.probe(punch.peerLocalAddress, mytoken)
	go punch.probe(punch.peerRemoteAddress, mytoken)

}

func (punch *HolePuncher) probe(addr *net.UDPAddr, mytoken *[]byte) {
	// binary protocol : 1 byte (SYN or ACK), 4 bytes (token)
	probeMessage := make([]byte, 5)
	probeMessage[0] = PunchSYN
	copy(probeMessage[1:5], *mytoken)
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

func (punch *HolePuncher) listenPeer(myToken *[]byte, expectedToken *[]byte) {
	buffer := make([]byte, 256)
	for {
		punch.socket.SetReadDeadline(time.Now().Add(punch.config.ListenInterval * time.Millisecond))
		n, addr, err := punch.socket.ReadFromUDP(buffer)

		// if error type is timeout, a null address will be returned to the channel
		if err != nil && os.IsTimeout(err) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: err}
			return
		}

		// test if is the same address
		if isMyOwnPC(myToken, expectedToken) {
			punch.resultCh <- HolePunchResult{Addr: nil, Err: ErrSameAdress}
			return
		}

		//otherwise continue looping
		if err != nil {
			continue
		}

		msg := buffer[:n]
		isExpectedToken := bytes.Equal(msg[1:5], *expectedToken)

		if msg[0] == PunchSYN && isExpectedToken {
			fmt.Printf("SYN RECEIVED BY: %s\n", addr.IP)
			ackMessage := make([]byte, 5)
			ackMessage[0] = PunchACK
			copy(ackMessage[1:5], *myToken)
			punch.socket.WriteToUDP(ackMessage, addr)

		} else if msg[0] == PunchACK && isExpectedToken {
			// the address that work will go to channel
			fmt.Printf("ACK RECEIVED BY: %s\n", addr.IP)
			punch.resultCh <- HolePunchResult{Addr: addr, Err: nil}
			return
		}

	}
}

func isMyOwnPC(myToken *[]byte, expectedToken *[]byte) bool {
	// verify if the target token is the same as pc token
	return bytes.Equal(*myToken, *expectedToken)
}
