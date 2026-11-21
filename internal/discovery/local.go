package discovery

import (
	"log"
	"net"
)

// LocalEndpoint -> local network ip:port of NIC
// port is ephermeral defiened on local connection Dynamic/Private Ports (49152–65535)
// based on: https://gosamples.dev/local-ip-address/

// dialUDP is net.Dial; a variable so tests can simulate a machine without a route.
var dialUDP = net.Dial

// getLocalIPV4 returns the IPv4 address of the interface that would reach the internet. Dialing a
// UDP "connection" sends nothing; it only asks the OS for a route. Without a route (offline,
// airplane mode) it falls back to loopback instead of terminating the whole application: the
// token stays valid and the user sees a connection failure, not a crash.
func getLocalIPV4() net.IP {
	conn, err := dialUDP("udp4", "8.8.8.8:80")
	if err != nil {
		log.Printf("[Discovery] no IPv4 route, using loopback as the local candidate: %v", err)
		return net.IPv4(127, 0, 0, 1)
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
