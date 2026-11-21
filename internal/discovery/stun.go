package discovery

import (
	"bytes"
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

func (c Client) buildBindingRequest(transaction_ID [stunTransactionIdLength]byte) []byte {
	var header = make([]byte, stunRequestHeaderLength)
	binary.BigEndian.PutUint16(header[0:2], stunMessageType) // message type = 2 bytes
	binary.BigEndian.PutUint16(header[2:4], 0)               // message length = 2 bytes (no adicional data)
	binary.BigEndian.PutUint32(header[4:8], stunMagicCookie) // magic cookie = 4 bytes
	copy(header[8:20], transaction_ID[:])                    // 12 random bytes used to match the response with the correct request
	return header
}

// stunReadTimeout bounds the wait for one server's reply.
const stunReadTimeout = 3 * time.Second

// stunResponseType is "Binding Success Response" (RFC 8489 §5).
const stunResponseType uint16 = 0x0101

// SendBindingRequest asks one STUN server for our public mapping. Only a reply that really is the
// answer to THIS request is accepted: the datagram must come from the queried server address and
// carry the Binding Success type, the magic cookie, a consistent length and our random 96-bit
// transaction ID. Anything else (stray packets, forged replies from other hosts) is ignored until
// the deadline, so it cannot poison the address that ends up in the token.
func (c Client) SendBindingRequest(conn *net.UDPConn, stunAddr string) (*net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp", stunAddr) // udp = ipv4 and ipv6
	if err != nil {
		return nil, fmt.Errorf("stun: resolving %s: %w", stunAddr, err)
	}

	// build and send stun binding request
	var txID [stunTransactionIdLength]byte
	if _, err := rand.Read(txID[:]); err != nil {
		return nil, fmt.Errorf("stun: generating transaction id: %w", err)
	}
	if _, err = conn.WriteTo(c.buildBindingRequest(txID), addr); err != nil {
		return nil, fmt.Errorf("stun: sending request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(stunReadTimeout))
	defer conn.SetReadDeadline(time.Time{}) // clear deadline after receive STUN
	buffer := make([]byte, 1024)
	for {
		n, from, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return nil, fmt.Errorf("stun: no valid response: %w", err)
		}
		if !from.IP.Equal(addr.IP) || from.Port != addr.Port {
			continue // not from the server we asked
		}
		mapped, err := parseBindingResponse(buffer[:n], txID)
		if err != nil {
			continue // malformed or not ours: keep waiting for the real answer
		}
		return mapped, nil
	}
}

// parseBindingResponse validates the 20-byte header of a STUN Binding Success Response against
// the request's transaction ID and then walks the attributes looking for XOR-MAPPED-ADDRESS
// (IPv4). Attributes are 4-byte aligned; every length is checked against the buffer.
func parseBindingResponse(buf []byte, txID [stunTransactionIdLength]byte) (*net.UDPAddr, error) {
	if len(buf) < stunRequestHeaderLength {
		return nil, fmt.Errorf("stun: response too short (%d bytes)", len(buf))
	}
	if binary.BigEndian.Uint16(buf[0:2]) != stunResponseType {
		return nil, fmt.Errorf("stun: unexpected message type 0x%04x", binary.BigEndian.Uint16(buf[0:2]))
	}
	msgLen := int(binary.BigEndian.Uint16(buf[2:4]))
	if msgLen%4 != 0 || stunRequestHeaderLength+msgLen != len(buf) {
		return nil, fmt.Errorf("stun: inconsistent message length %d for %d bytes", msgLen, len(buf))
	}
	if binary.BigEndian.Uint32(buf[4:8]) != stunMagicCookie {
		return nil, fmt.Errorf("stun: bad magic cookie")
	}
	if !bytes.Equal(buf[8:20], txID[:]) {
		return nil, fmt.Errorf("stun: transaction id mismatch")
	}
	attrs := buf[stunRequestHeaderLength:]
	for len(attrs) >= 4 {
		typ := binary.BigEndian.Uint16(attrs[0:2])
		l := int(binary.BigEndian.Uint16(attrs[2:4]))
		padded := (l + 3) &^ 3
		if 4+padded > len(attrs) {
			return nil, fmt.Errorf("stun: attribute 0x%04x overruns the message", typ)
		}
		if typ == stunAttrXORMappedAddress {
			return decodeXORMappedValue(attrs[4 : 4+l])
		}
		attrs = attrs[4+padded:]
	}
	return nil, fmt.Errorf("stun: no XOR-MAPPED-ADDRESS in response")
}

const stunAttrXORMappedAddress uint16 = 0x0020

// decodeXORMappedValue decodes the value of an IPv4 XOR-MAPPED-ADDRESS attribute:
// reserved (1) | family (1) | port (2) | ipv4 (4).
func decodeXORMappedValue(value []byte) (*net.UDPAddr, error) {
	const familyIPv4 byte = 0x01
	if len(value) != 8 {
		return nil, fmt.Errorf("stun: unsupported attribute length %d", len(value))
	}
	if value[1] != familyIPv4 {
		return nil, fmt.Errorf("stun: unsupported address family %d", value[1])
	}
	// port is XORed with the top 16 bits of the magic cookie, ip with the full cookie
	port := int(binary.BigEndian.Uint16(value[2:4]) ^ uint16(stunMagicCookie>>16))
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(value[4:8])^stunMagicCookie)
	return &net.UDPAddr{IP: ip, Port: port}, nil
}

// parseXORMappedAddress decodes the XOR-MAPPED-ADDRESS attribute of a STUN binding response.
// The buffer comes straight from the network, so every length is validated and any problem is
// reported as an error (a stray or hostile datagram must never crash the process).
// Layout after the 20 byte header: attr type (2) | attr len (2) | reserved (1) | family (1) | port (2) | ipv4 (4).
func parseXORMappedAddress(buffer []byte) (*net.UDPAddr, error) {
	const attrHeaderLen = 4
	if len(buffer) < stunRequestHeaderLength+attrHeaderLen+8 {
		return nil, fmt.Errorf("stun: response too short (%d bytes)", len(buffer))
	}
	attrType := binary.BigEndian.Uint16(buffer[stunRequestHeaderLength : stunRequestHeaderLength+2])
	attrLen := int(binary.BigEndian.Uint16(buffer[stunRequestHeaderLength+2 : stunRequestHeaderLength+4]))
	if attrType != stunAttrXORMappedAddress {
		return nil, fmt.Errorf("stun: unexpected attribute type 0x%04x", attrType)
	}
	start := stunRequestHeaderLength + attrHeaderLen
	if attrLen > len(buffer)-start {
		return nil, fmt.Errorf("stun: attribute overruns the message")
	}
	return decodeXORMappedValue(buffer[start : start+attrLen])
}
