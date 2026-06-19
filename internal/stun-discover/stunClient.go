package stundiscover

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
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
type PublicEndpoint struct {
	response string
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
		return PublicEndpoint{}, err
	}

	conn, err := net.DialUDP("udp", nil, addr) // udp =  ipv4 and ipv6

	if err != nil {
		log.Fatal("Connection failed: ", err)
		return PublicEndpoint{}, err
	}
	defer conn.Close() // defers the close of connection until the surrounding function returns

	// build and send stun binding request
	var transaction_ID = make([]byte, stunTransactionIdLength)
	rand.Read(transaction_ID)
	header := c.buildBindingRequest([12]byte(transaction_ID)) // returns header request binding
	_, err = conn.Write(header)

	if err != nil {
		log.Fatal("Error sending message to Stun server", err)
		return PublicEndpoint{}, err
	}

	// Wait for reply of stun server with timeout - 5 sec timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 512)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Printf("Receive error: %v", err)
		return PublicEndpoint{}, err
	}
	fmt.Printf("Server says: %x\n", buffer[:n])
	return parseXORMappedAddress(buffer)
}

// first 8 bytes are message type and message length
// 4 bytes for messsage cookie
func parseXORMappedAddress(buffer []byte) (PublicEndpoint, error) {
	panic("unimplemented")
}
