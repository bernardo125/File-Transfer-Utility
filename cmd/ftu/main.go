package main

import (
	encryption "cmd/ftu/main.go/internal/encryption"
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
	"encoding/hex"
)

func main() {
	var DefaultSTUNServers = []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
		"stun.cloudflare.com:3478",
	}
	stunClient := stundiscover.Client{}

	response, _ := stunClient.SendBindingRequest(DefaultSTUNServers[0])
	key, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")
	encryption.AesgsmEncryct(key, &response)

}
