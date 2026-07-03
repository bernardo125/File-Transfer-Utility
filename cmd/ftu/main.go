package main

import (
	local "cmd/ftu/main.go/internal/local-discover"
	"fmt"
)

func main() {

	ip := local.GetLocalIP()
	fmt.Printf("LOCAL: %s\n", ip.To4())

}
