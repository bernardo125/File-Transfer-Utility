package main

import (
	token "cmd/ftu/main.go/internal/signaling"
	"fmt"
)

func main() {

	key := token.CreateKey()
	fmt.Printf("KEY: %s\n", key)

	result := token.GetPeer(key)
	fmt.Printf("RESULT: %x\n", *result)

}
