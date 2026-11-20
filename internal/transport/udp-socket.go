package transport

import (
	"fmt"
	"log"
	"net"
)

// ListenUDPSocket opens the single UDP socket shared by STUN, hole punching and QUIC. It returns
// an error instead of exiting the process, so a GUI can report it.
func ListenUDPSocket() (*net.UDPConn, error) {
	addr := &net.UDPAddr{Port: 0} // force not specified binding

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("opening UDP socket: %w", err)
	}

	// 7mb buffer to increase throughput
	conn.SetReadBuffer(7 * 1024 * 1024)
	conn.SetWriteBuffer(7 * 1024 * 1024)
	return conn, nil
}

// InitializeUDPSocket is ListenUDPSocket for the CLI and tests: a failure is fatal there.
func InitializeUDPSocket() (conn *net.UDPConn) {
	conn, err := ListenUDPSocket()
	if err != nil {
		log.Fatal("Connection failed: ", err)
	}
	return conn
}
