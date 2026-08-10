package main

import (
	local "cmd/ftu/main.go/internal/local-discover"
	socket "cmd/ftu/main.go/internal/network"
	token "cmd/ftu/main.go/internal/signaling"
	punch "cmd/ftu/main.go/internal/udp-hole-punch"
	"fmt"
)

func main() {
	localIP := local.GetLocalIP()
	conn := socket.InitializeUDPSocket(localIP)
	defer conn.Close() // defers the close of connection until the surrounding function returns
	public_token := token.CreateKey(conn)
	fmt.Printf("Token: %s\n", public_token)

	var other_peer_token string
	fmt.Print("Enter other peer token code: ")
	fmt.Scan(&other_peer_token)

	// convert key to net.udpAdrr
	local, remote := token.GetPeer(other_peer_token)
	fmt.Printf("OTHER PEER:\nLOCAL IP: %s, PORT: %d\nREMOTE IP: %s, PORT: %d\n", local.IP, local.Port, remote.IP, remote.Port)
	resultChn := punch.UDPHolePunch(conn, local, remote)
	result := <-resultChn
	fmt.Printf("Hole Punching concluído com sucesso para: %s\n", result.Addr.String())
}
