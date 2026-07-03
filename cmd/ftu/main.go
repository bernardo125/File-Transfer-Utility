package main

import (
	token "cmd/ftu/main.go/internal/signaling"
	"fmt"
	"log"
	"net"
)

func main() {
	// initialize one socket for all interaction on utility
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Connection failed: ", err)
	}
	defer conn.Close() // defers the close of connection until the surrounding function returns
	public_token := token.CreateKey(conn)
	fmt.Printf("Token: %s\n", public_token)

	// convert key to net.udpAdrr
	remote_peer_addr := token.GetPeer(public_token)
	fmt.Printf("PEER IP: %s, PORT: %d\n", remote_peer_addr.IP, remote_peer_addr.Port)
}
