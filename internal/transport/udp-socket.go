package transport

import (
	"log"
	"net"
)

func InitializeUDPSocket(localIP net.IP) (conn *net.UDPConn) {
	// initialize one socket for all interaction on utility
	addr := &net.UDPAddr{IP: localIP, Port: 0}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Connection failed: ", err)
	}
	return conn
}
