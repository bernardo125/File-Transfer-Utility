package main

import (
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
)

func main() {
	var DefaultSTUNServers = []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
		"stun.cloudflare.com:3478",
	}
	stunClient := stundiscover.Client{}

	stunClient.SendBindingRequest(DefaultSTUNServers[0])
}
