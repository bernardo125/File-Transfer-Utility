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
	local, remote := token.GetPeer(public_token)
	fmt.Printf("OTHER PEER:\nLOCAL IP: %s, PORT: %d\nREMOTE IP: %s, PORT: %d\n", local.IP, local.Port, remote.IP, remote.Port)
}
