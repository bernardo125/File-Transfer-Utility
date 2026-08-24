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
	public_token, myDynamicToken := token.CreateKey(conn)
	fmt.Printf("Token: %s\n", public_token)

	var other_peer_token string
	fmt.Print("Enter other peer token code: ")
	fmt.Scan(&other_peer_token)

	// convert key to net.udpAdrr
	local, remote, expectedToken := token.GetPeer(other_peer_token)
	fmt.Printf("OTHER PEER:\nLOCAL IP: %s, PORT: %d\nREMOTE IP: %s, PORT: %d\n", local.IP, local.Port, remote.IP, remote.Port)
	resultChn := punch.UDPHolePunch(conn, local, remote, myDynamicToken, expectedToken)
	result := <-resultChn

	if result.Err != nil {
		fmt.Printf("Hole Punching error: %s\n", result.Err)
		return
	} else {
		fmt.Printf("Hole Punching sucessfuly conected to: %s\n", result.Addr.String())
	}

	comunication.QuicUpgrade(conn, result.Addr, myDynamicToken, expectedToken)
	fmt.Print("TEST: ")
	fmt.Scan(&other_peer_token)
}
