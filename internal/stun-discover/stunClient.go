package stundiscover

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"net"
	"time"
)

// Stun protocol header constants
const stunMessageType uint16 = 0x0001     // 1  -> binding  request
const stunMagicCookie uint32 = 0x2112A442 // fixed constant used by STUN so servers can recognize and validate the packet (definied by RFC 538 law)
const stunRequestHeaderLength int8 = 20   // fixed 20 bytes for simples STUN binding request
const stunTransactionIdLength int8 = 12   // 12 random bytes used to match the response with the correct request

// STUN CLIENT -> with all stun binding logic
type StunClient interface {
	buildBindingRequest(txID [12]byte) []byte
	SendBindingRequest(stunAddr string) (PublicEndpoint, error) // PUBLIC FUNCS START WITH CAPS
	parseXORMappedAddress(val []byte) (PublicEndpoint, error)
}

// PublicEndpoint -> reponse by stun server
// gob only serialize in uppercase 1 character
type PublicEndpoint struct {
	Ip   uint32
	Port uint16
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

func (c Client) SendBindingRequest(stunAddr string) (PublicEndpoint, error) {
	addr, err := net.ResolveUDPAddr("udp", stunAddr) // udp = ipv4 and ipv6
	if err != nil {
		log.Fatalf("Error resolving: %s", stunAddr)
	}

	conn, err := net.DialUDP("udp", nil, addr) // udp =  ipv4 and ipv6

	if err != nil {
		log.Fatal("Connection failed: ", err)
	}
	defer conn.Close() // defers the close of connection until the surrounding function returns

	// build and send stun binding request
	var transaction_ID = make([]byte, stunTransactionIdLength)
	rand.Read(transaction_ID)
	header := c.buildBindingRequest([12]byte(transaction_ID)) // returns header request binding
	_, err = conn.Write(header)

	if err != nil {
		log.Fatal("Error sending message to Stun server", err)
	}

	// Wait for reply of stun server with timeout - 5 sec timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 512)
	_, _, err = conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal("Receive error: ", err)
	}
	//TODO : TO REMOVE===========================================================================
	return parseXORMappedAddress(buffer)
}

// first 8 bytes are message type and message length
// 4 bytes for messsage cookie
// last 6 bytes are port and ip (2 bytes = port, 4 bytes = ip)
func parseXORMappedAddress(buffer []byte) (PublicEndpoint, error) {
	// plus 2 because of type on message atribute
	const family_ipv4 byte = 0x01
	const STUN_ATTR_XOR_MAPPED_ADDRESS uint16 = 0x0020

	attr_type := binary.BigEndian.Uint16(buffer[stunRequestHeaderLength : stunRequestHeaderLength+2])
	attr_len := int8(binary.BigEndian.Uint16(buffer[stunRequestHeaderLength+2 : stunRequestHeaderLength+4]))

	if attr_type != STUN_ATTR_XOR_MAPPED_ADDRESS {
		log.Fatal("This is not RESPONSE-ADDRESS message type")
	}
	// plus 4 to jump to message atribute value
	payload := buffer[stunRequestHeaderLength+4 : stunRequestHeaderLength+attr_len+4]
	family := payload[1]

	if family != family_ipv4 {
		log.Fatalf("unsupported family: %d", family)
	}

	// port is XORed with the top 16 bits of the magic cookie
	raw_port := binary.BigEndian.Uint16(payload[2:4])
	port := raw_port ^ uint16(stunMagicCookie>>16)

	// ip is XORed with the full magic cookie
	raw_ip := binary.BigEndian.Uint32(payload[4:8])
	ip := raw_ip ^ stunMagicCookie // uint32
	//TODO : TO REMOVE===========================================================================
	return PublicEndpoint{Ip: ip, Port: port}, nil
}
