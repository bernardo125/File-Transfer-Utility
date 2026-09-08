package discovery

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/chacha20"
)

var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun.cloudflare.com:3478",
}
var secret, _ = hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

type connectionCandidates struct {
	LocalAddressIPV4      *net.UDPAddr
	PublicAddressIPV6     *net.UDPAddr
	StunPublicAddressIPV4 *net.UDPAddr
	Ed25519PublicToken    *ed25519.PublicKey
}

// Orchestrate encryptation and base64 codification
func CreateKey(conn *net.UDPConn) (key string, private ed25519.PrivateKey) {
	stunClient := Client{}
	local_ipv4, public_ipv6 := GetMachineAddrs(conn)
	stun_response, _ := stunClient.SendBindingRequest(conn, DefaultSTUNServers[0])
	public, priv := ed55519TokenGen()
	connCandidates := connectionCandidates{LocalAddressIPV4: local_ipv4, PublicAddressIPV6: public_ipv6, StunPublicAddressIPV4: stun_response, Ed25519PublicToken: &public}
	nonce, ciphertext := chacha20Encryct(secret, &connCandidates)
	finalPacket := append(nonce, ciphertext...)
	key = base64Enconding(finalPacket)
	return key, priv
}

// generate ed55519 key to avoid spoofing and mitm attacks
func ed55519TokenGen() (public ed25519.PublicKey, priv ed25519.PrivateKey) {
	public, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return public, priv
}

func GetPeer(key string) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, publickey *ed25519.PublicKey, err error) {
	if len(key) < 50 {
		return nil, nil, nil, fmt.Errorf("That's not a token")
	}
	decodedBytes := base64Decoding(key)
	nonce := decodedBytes[:12] // nonce is 12 bytes
	cipherText := decodedBytes[12:]
	return chacha20Decryct(secret, nonce, cipherText)
}

// ENCRYPTATION BASED ON Chacha20
// based on https://mojoauth.com/encryption-decryption/chacha20-256-encryption--go#encrypting-data-with-chacha20-256-in-goM
func chacha20Encryct(secret []byte, candidates_response *connectionCandidates) (nonce []byte, ciphertext []byte) {
	// Generate a nonce
	nonce = make([]byte, chacha20.NonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		panic(err)
	}
	// Create a ChaCha20 cipher
	cipher, err := chacha20.NewUnauthenticatedCipher(secret, nonce)
	if err != nil {
		panic(err)
	}

	plaintext := encodeTo62bytes(candidates_response)
	ciphertext = make([]byte, len(plaintext))
	// Encrypt the plaintext
	cipher.XORKeyStream(ciphertext, plaintext)
	return nonce, ciphertext
}

func encodeTo62bytes(candidates_response *connectionCandidates) (encodedEndpoint []byte) {
	// encode connectionCandidates struct to byte buffer
	// build 62 byte array with local and stun address to better compression
	// ipv4 = 4 bytes , ipv6 = 16 bytes, port = 2 bytes, ed25519 token = 32 bytes (avoid spoofing and mitm)
	var response []byte
	// verify if ipv6 is present
	if candidates_response.PublicAddressIPV6 != nil {
		response = make([]byte, 62)
		// public address ipv6
		copy(response[44:62], candidates_response.PublicAddressIPV6.IP.To16())
		binary.BigEndian.PutUint16(response[22:24], uint16(candidates_response.LocalAddressIPV4.Port))
	} else {
		response = make([]byte, 44)
	}

	//ed25519 token
	copy(response[0:32], *candidates_response.Ed25519PublicToken)

	// local address ipv4
	copy(response[32:36], candidates_response.LocalAddressIPV4.IP.To4())
	binary.BigEndian.PutUint16(response[36:38], uint16(candidates_response.LocalAddressIPV4.Port))

	//stun remote ipv4
	copy(response[38:42], candidates_response.StunPublicAddressIPV4.IP.To4())
	binary.BigEndian.PutUint16(response[42:44], uint16(candidates_response.StunPublicAddressIPV4.Port))

	return response
}

// Base64 codification
func base64Enconding(data []byte) (b64token string) {
	return base64.RawURLEncoding.EncodeToString(data)
}

func chacha20Decryct(secret []byte, nonce []byte, ciphertext []byte) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, publickey *ed25519.PublicKey, err error) {
	cipher, err := chacha20.NewUnauthenticatedCipher(secret, nonce)
	if err != nil {
		return nil, nil, nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	// Decrypt the ciphertext
	cipher.XORKeyStream(plaintext, ciphertext)

	endpoint := decodeFrom16bytes(plaintext)
	peerLocal = endpoint.LocalAddressIPV4
	peerRemote = endpoint.StunPublicAddressIPV4
	publickey = endpoint.Ed25519PublicToken

	return peerLocal, peerRemote, publickey, nil
}

func decodeFrom16bytes(encodedText []byte) (decodedResponse *connectionCandidates) {
	// decode byte buffer to connectionCandidates struct
	// build 12byte array with local and stun address to better compression
	// ip = 4 bytes , port = 2 bytes
	//public ed25519
	publickey := ed25519.PublicKey(encodedText[0:32])

	// local address ipv4
	localIPV4 := make(net.IP, 4)
	raw_localIPV4 := binary.BigEndian.Uint32(encodedText[32:36])
	binary.BigEndian.PutUint32(localIPV4, raw_localIPV4)
	localPortIPV4 := int(binary.BigEndian.Uint16(encodedText[36:38])) // int to net.UDPAddr struct

	// stun remote address ipv4
	remoteIPV4 := make(net.IP, 4)
	raw_remoteIPV4 := binary.BigEndian.Uint32(encodedText[38:42])
	binary.BigEndian.PutUint32(remoteIPV4, raw_remoteIPV4)
	remotePort := int(binary.BigEndian.Uint16(encodedText[42:44]))

	localResponseIPV4 := net.UDPAddr{IP: localIPV4, Port: localPortIPV4}
	remoteResponseIPV4 := net.UDPAddr{IP: remoteIPV4, Port: remotePort}
	decodedResponse = &connectionCandidates{LocalAddressIPV4: &localResponseIPV4, PublicAddressIPV6: decodeIPV6IfExists(encodedText), StunPublicAddressIPV4: &remoteResponseIPV4, Ed25519PublicToken: &publickey}
	return decodedResponse
}

func decodeIPV6IfExists(encodedText []byte) *net.UDPAddr {
	// magic number
	if len(encodedText) > 44 {
		// public address ipv6
		publicIPV6 := make(net.IP, 16)
		raw_publicIPV6 := binary.BigEndian.Uint32(encodedText[44:60])
		binary.BigEndian.PutUint32(publicIPV6, raw_publicIPV6)
		publicPortIPV6 := int(binary.BigEndian.Uint16(encodedText[60:62])) // int to net.UDPAddr struct
		return &net.UDPAddr{IP: publicIPV6, Port: publicPortIPV6}
	}
	return nil
}

func base64Decoding(b64token string) (decoded []byte) {
	decode, err := base64.RawURLEncoding.DecodeString(b64token)
	if err != nil {
		log.Fatal(err.Error())
	}
	return decode
}
