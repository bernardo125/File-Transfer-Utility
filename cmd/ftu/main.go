package main

import (
	local "cmd/ftu/main.go/internal/local-discover"
	socket "cmd/ftu/main.go/internal/network"
	comunication "cmd/ftu/main.go/internal/quic"
	token "cmd/ftu/main.go/internal/signaling"
	punch "cmd/ftu/main.go/internal/udp-hole-punch"
	"fmt"
)

func main() {
	localIP := local.GetLocalIP()
	conn := socket.InitializeUDPSocket(localIP)
	defer conn.Close() // defers the close of connection until the surrounding function returns
	publicToken, privateKey := token.CreateKey(conn)
	fmt.Printf("Token: %s\n", publicToken)

	var other_peer_token string
	fmt.Print("Enter other peer token code: ")
	fmt.Scan(&other_peer_token)

	// convert key to net.udpAdrr
	local, remote, otherPeerPublicKey := token.GetPeer(other_peer_token)
	fmt.Printf("OTHER PEER:\nLOCAL IP: %s, PORT: %d\nREMOTE IP: %s, PORT: %d\n", local.IP, local.Port, remote.IP, remote.Port)
	resultChn := punch.UDPHolePunch(conn, local, remote, &privateKey, otherPeerPublicKey)
	result := <-resultChn

	if result.Err != nil {
		fmt.Printf("Hole Punching error: %s\n", result.Err)
		return
	} else {
		fmt.Printf("Hole Punching sucessfuly conected to: %s\n", result.Addr.String())
	}

	comunication.QuicUpgrade(conn, result.Addr, &privateKey, otherPeerPublicKey)
	fmt.Print("TEST: ")
	fmt.Scan(&other_peer_token)
}
