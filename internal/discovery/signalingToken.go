package discovery

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
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
	LocalAddress       *net.UDPAddr
	StunPublicAddress  *net.UDPAddr
	Ed25519PublicToken *ed25519.PublicKey
}

// Orchestrate encryptation and base64 codification
func CreateKey(conn *net.UDPConn) (key string, private ed25519.PrivateKey) {
	stunClient := Client{}
	local_response := GetLocalUDPAddr(conn)
	stun_response, _ := stunClient.SendBindingRequest(conn, DefaultSTUNServers[0])
	public, priv := ed55519TokenGen()
	connCandidates := connectionCandidates{LocalAddress: local_response, StunPublicAddress: stun_response, Ed25519PublicToken: &public}
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

func GetPeer(key string) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, publickey *ed25519.PublicKey) {
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

	plaintext := encodeTo16bytes(candidates_response)
	ciphertext = make([]byte, len(plaintext))
	// Encrypt the plaintext
	cipher.XORKeyStream(ciphertext, plaintext)
	return nonce, ciphertext
}

func encodeTo16bytes(candidates_response *connectionCandidates) (encodedEndpoint []byte) {
	// encode connectionCandidates struct to byte buffer
	// build 44 byte array with local and stun address to better compression
	// ip = 4 bytes , port = 2 bytes, ed25519 token = 32 bytes (avoid spoofing and mitm)
	response := make([]byte, 44)
	// local address
	copy(response[0:4], candidates_response.LocalAddress.IP.To4())
	binary.BigEndian.PutUint16(response[4:6], uint16(candidates_response.LocalAddress.Port))

	//stun remote
	copy(response[6:10], candidates_response.StunPublicAddress.IP.To4())
	binary.BigEndian.PutUint16(response[10:12], uint16(candidates_response.StunPublicAddress.Port))

	//ed25519 token
	copy(response[12:44], *candidates_response.Ed25519PublicToken)

	return response
}

// Base64 codification
func base64Enconding(data []byte) (b64token string) {
	return base64.RawURLEncoding.EncodeToString(data)
}

func chacha20Decryct(secret []byte, nonce []byte, ciphertext []byte) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, publickey *ed25519.PublicKey) {
	cipher, err := chacha20.NewUnauthenticatedCipher(secret, nonce)
	if err != nil {
		panic(err)
	}

	plaintext := make([]byte, len(ciphertext))
	// Decrypt the ciphertext
	cipher.XORKeyStream(plaintext, ciphertext)

	endpoint := decodeFrom16bytes(plaintext)
	peerLocal = endpoint.LocalAddress
	peerRemote = endpoint.StunPublicAddress
	publickey = endpoint.Ed25519PublicToken

	return peerLocal, peerRemote, publickey
}

func decodeFrom16bytes(encodedText []byte) (decodedResponse *connectionCandidates) {
	// decode byte buffer to connectionCandidates struct
	// build 12byte array with local and stun address to better compression
	// ip = 4 bytes , port = 2 bytes
	// local address
	localIP := make(net.IP, 4)
	raw_localIP := binary.BigEndian.Uint32(encodedText[0:4])
	binary.BigEndian.PutUint32(localIP, raw_localIP)
	localPort := int(binary.BigEndian.Uint16(encodedText[4:6])) // int to net.UDPAddr struct
	// stun remote address
	remoteIP := make(net.IP, 4)
	raw_remoteIP := binary.BigEndian.Uint32(encodedText[6:10])
	binary.BigEndian.PutUint32(remoteIP, raw_remoteIP)
	remotePort := int(binary.BigEndian.Uint16(encodedText[10:12]))

	//public ed25519
	publickey := ed25519.PublicKey(encodedText[12:44])

	localResponse := net.UDPAddr{IP: localIP, Port: localPort}
	remoteResponse := net.UDPAddr{IP: remoteIP, Port: remotePort}
	decodedResponse = &connectionCandidates{LocalAddress: &localResponse, StunPublicAddress: &remoteResponse, Ed25519PublicToken: &publickey}
	return decodedResponse
}

func base64Decoding(b64token string) (decoded []byte) {
	decode, err := base64.RawURLEncoding.DecodeString(b64token)
	if err != nil {
		log.Fatal(err.Error())
	}
	return decode
}
