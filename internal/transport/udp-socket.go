package transport

import (
	"log"
	"net"
)

func InitializeUDPSocket() (conn *net.UDPConn) {
	// initialize one socket for all interaction on utility
	addr := &net.UDPAddr{Port: 0} // force not specified binding

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Connection failed: ", err)
	}

	// 7mb buffer to increase throughput
	conn.SetReadBuffer(7 * 1024 * 1024)
	conn.SetWriteBuffer(7 * 1024 * 1024)
	return conn
}
