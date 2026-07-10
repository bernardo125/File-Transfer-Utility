package localdiscover

import (
	"log"
	"net"
)

// LocalEndpoint -> local network ip:port of NIC
// port is ephermeral defiened on local connection Dynamic/Private Ports (49152–65535)
// based on: https://gosamples.dev/local-ip-address/

func GetLocalIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddress := conn.LocalAddr().(*net.UDPAddr)

	return localAddress.IP
}

func GetLocalUDPAddr(conn *net.UDPConn) *net.UDPAddr {
	localAddress := conn.LocalAddr().(*net.UDPAddr)
	return localAddress
}
