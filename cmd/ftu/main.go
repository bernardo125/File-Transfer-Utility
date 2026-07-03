package main

import (
	local "cmd/ftu/main.go/internal/local-discover"
	token "cmd/ftu/main.go/internal/signaling"
	"fmt"
)

func main() {

	ip := local.GetLocalIP()
	fmt.Printf("LOCAL: %s\n", ip.To4())
	public_token := token.CreateKey()
	fmt.Printf("Token: %s\n", public_token)

	// convert key to net.udpAdrr
	remote_peer_addr := token.GetPeer(public_token)
	fmt.Printf("PEER IP: %s, PORT: %d\n", remote_peer_addr.IP, remote_peer_addr.Port)
}
