package main

import (
	connectivity "cmd/ftu/main.go/internal/connectivity"
	discovery "cmd/ftu/main.go/internal/discovery"
	transport "cmd/ftu/main.go/internal/transport"
	"fmt"
)

func main() {
	localIP := discovery.GetLocalIP()
	conn := transport.InitializeUDPSocket(localIP)
	defer conn.Close() // defers the close of connection until the surrounding function returns
	publicToken, privateKey := discovery.CreateKey(conn)
	fmt.Printf("Token: %s\n", publicToken)

	var other_peer_token string
	fmt.Print("Enter other peer token code: ")
	fmt.Scan(&other_peer_token)

	// convert key to net.udpAdrr
	local, remote, otherPeerPublicKey := discovery.GetPeer(other_peer_token)
	fmt.Printf("OTHER PEER:\nLOCAL IP: %s, PORT: %d\nREMOTE IP: %s, PORT: %d\n", local.IP, local.Port, remote.IP, remote.Port)
	resultChn := connectivity.UDPHolePunch(conn, local, remote, &privateKey, otherPeerPublicKey)
	result := <-resultChn

	if result.Err != nil {
		fmt.Printf("Hole Punching error: %s\n", result.Err)
		return
	} else {
		fmt.Printf("Hole Punching sucessfuly conected to: %s\n", result.Addr.String())
	}

	transport.QuicUpgrade(conn, result.Addr, &privateKey, otherPeerPublicKey)
	fmt.Print("TEST: ")
	fmt.Scan(&other_peer_token)
}
