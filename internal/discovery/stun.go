package discovery

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Stun protocol header constants
const stunMessageType uint16 = 0x0001     // 1  -> binding  request
const stunMagicCookie uint32 = 0x2112A442 // fixed constant used by STUN so servers can recognize and validate the packet (definied by RFC 538 law)
const stunRequestHeaderLength = 20        // fixed 20 bytes for simples STUN binding request
const stunTransactionIdLength = 12        // 12 random bytes used to match the response with the correct request

// STUN CLIENT -> with all stun binding logic
type StunClient interface {
	buildBindingRequest(txID [12]byte) []byte
	SendBindingRequest(stunAddr string) (*net.UDPAddr, error) // PUBLIC FUNCS START WITH CAPS
	parseXORMappedAddress(val []byte) (*net.UDPAddr, error)
}

// StunClient implementation
type Client struct{}

func (c Client) buildBindingRequest(transaction_ID [12]byte) []byte {
	var header = make([]byte, stunRequestHeaderLength)
	binary.BigEndian.PutUint16(header[0:2], stunMessageType) // message type = 2 bytes
	binary.BigEndian.PutUint16(header[2:4], 0)               // message length = 2 bytes (no adicional data)
	binary.BigEndian.PutUint32(header[4:8], stunMagicCookie) // magic cookie = 4 bytes
	copy(header[8:20], transaction_ID[:])                    // 12 random bytes used to match the response with the correct request
	return header
}

// for now this func returns the response from stun server
func (c Client) SendBindingRequest(conn *net.UDPConn, stunAddr string) (*net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp", stunAddr) // udp = ipv4 and ipv6
	if err != nil {
		return nil, fmt.Errorf("stun: resolving %s: %w", stunAddr, err)
	}

	// build and send stun binding request
	var transaction_ID = make([]byte, stunTransactionIdLength)
	rand.Read(transaction_ID)
	header := c.buildBindingRequest([12]byte(transaction_ID)) // returns header request binding
	_, err = conn.WriteTo(header, addr)

	if err != nil {
		return nil, fmt.Errorf("stun: sending request: %w", err)
	}

	// Wait for reply of stun server with timeout - 5 sec timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 512)
	n, _, err := conn.ReadFromUDP(buffer)
	conn.SetReadDeadline(time.Time{}) // clear deadline after receive STUN
	if err != nil {
		return nil, fmt.Errorf("stun: no response: %w", err)
	}
	return parseXORMappedAddress(buffer[:n])
}

// parseXORMappedAddress decodes the XOR-MAPPED-ADDRESS attribute of a STUN binding response.
// The buffer comes straight from the network, so every length is validated and any problem is
// reported as an error (a stray or hostile datagram must never crash the process).
// Layout after the 20 byte header: attr type (2) | attr len (2) | reserved (1) | family (1) | port (2) | ipv4 (4).
func parseXORMappedAddress(buffer []byte) (*net.UDPAddr, error) {
	const family_ipv4 byte = 0x01
	const STUN_ATTR_XOR_MAPPED_ADDRESS uint16 = 0x0020
	const attrHeaderLen = 4
	const ipv4ValueLen = 8

	if len(buffer) < stunRequestHeaderLength+attrHeaderLen+ipv4ValueLen {
		return nil, fmt.Errorf("stun: response too short (%d bytes)", len(buffer))
	}
	attr_type := binary.BigEndian.Uint16(buffer[stunRequestHeaderLength : stunRequestHeaderLength+2])
	attr_len := int(binary.BigEndian.Uint16(buffer[stunRequestHeaderLength+2 : stunRequestHeaderLength+4]))

	if attr_type != STUN_ATTR_XOR_MAPPED_ADDRESS {
		return nil, fmt.Errorf("stun: unexpected attribute type 0x%04x", attr_type)
	}
	if attr_len != ipv4ValueLen {
		return nil, fmt.Errorf("stun: unsupported attribute length %d", attr_len)
	}
	payload := buffer[stunRequestHeaderLength+attrHeaderLen : stunRequestHeaderLength+attrHeaderLen+attr_len]
	if payload[1] != family_ipv4 {
		return nil, fmt.Errorf("stun: unsupported address family %d", payload[1])
	}

	// port is XORed with the top 16 bits of the magic cookie
	raw_port := binary.BigEndian.Uint16(payload[2:4])
	port := int(raw_port ^ uint16(stunMagicCookie>>16))

	// ip is XORed with the full magic cookie
	ip := make(net.IP, 4)
	raw_ip := binary.BigEndian.Uint32(payload[4:8])
	binary.BigEndian.PutUint32(ip, raw_ip^stunMagicCookie)

	return &net.UDPAddr{IP: ip, Port: port}, nil
}
