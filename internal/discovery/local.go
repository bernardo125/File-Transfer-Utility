package discovery

import (
	"log"
	"net"
)

// LocalEndpoint -> local network ip:port of NIC
// port is ephermeral defiened on local connection Dynamic/Private Ports (49152–65535)
// based on: https://gosamples.dev/local-ip-address/

func getLocalIPV4() net.IP {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP
}

func getPublicIPV6() net.IP {
	conn, err := net.Dial("udp6", "[2001:4860:4860::8888]:53")
	if err != nil {
		return nil
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP

}

func GetMachineAddrs(conn *net.UDPConn) (ipv4Local *net.UDPAddr, ipv6Public *net.UDPAddr) {
	port := conn.LocalAddr().(*net.UDPAddr).Port
	return &net.UDPAddr{IP: getLocalIPV4(), Port: port}, &net.UDPAddr{IP: getPublicIPV6(), Port: port}

}

func GetLocalUDPAddr(conn *net.UDPConn) *net.UDPAddr {
	localAddress := conn.LocalAddr().(*net.UDPAddr)
	return localAddress
}
