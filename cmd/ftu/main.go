package main

import (
	local "cmd/ftu/main.go/internal/local-discover"
	socket "cmd/ftu/main.go/internal/network"
	token "cmd/ftu/main.go/internal/signaling"
	"fmt"
)

func main() {
	localIP := local.GetLocalIP()
	conn := socket.InitializeUDPSocket(localIP)
	defer conn.Close() // defers the close of connection until the surrounding function returns
	public_token := token.CreateKey(conn)
	fmt.Printf("Token: %s\n", public_token)

	// convert key to net.udpAdrr
	remote := token.GetPeer(public_token)
	fmt.Printf("LOCAL PEER IP: %s, PORT: %d\nSTUN REMOTE IP: %s, PORT: %d\n", remote.LocalAddress.IP, remote.LocalAddress.Port, remote.StunPublicAddress.IP, remote.StunPublicAddress.Port)
}
