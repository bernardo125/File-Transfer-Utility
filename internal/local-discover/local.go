package localdiscover

import (
	"net"
)

// LocalEndpoint -> local network ip:port of NIC
// port is ephermeral defiened on local connection Dynamic/Private Ports (49152–65535)
// based on: https://gosamples.dev/local-ip-address/

func GetLocalIP(conn *net.UDPConn) *net.Addr {

}
