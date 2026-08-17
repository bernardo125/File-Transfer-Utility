package encryption

import (
	localdiscover "cmd/ftu/main.go/internal/local-discover"
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
)

var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun.cloudflare.com:3478",
}
var secret, _ = hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

type connectionCandidates struct {
	LocalAddress      *net.UDPAddr
	StunPublicAddress *net.UDPAddr
	RandToken         []byte
}

// Orchestrate encryptation and base64 codification
func CreateKey(conn *net.UDPConn) (key string, myDynamicToken *[]byte) {
	stunClient := stundiscover.Client{}
	local_response := localdiscover.GetLocalUDPAddr(conn)
	stun_response, _ := stunClient.SendBindingRequest(conn, DefaultSTUNServers[0])
	randToken := dynamicTokenGen()
	connCandidates := connectionCandidates{LocalAddress: local_response, StunPublicAddress: stun_response, RandToken: randToken}
	nonce, ciphertext := aesgsmEncryct(secret, &connCandidates)
	finalPacket := append(nonce, ciphertext...)
	key = base64Enconding(finalPacket)
	return key, &randToken
}

// generate dynamic 4 byte key to avoid spoofing on udp hole punch
func dynamicTokenGen() (randToken []byte) {
	tokenarray := make([]byte, 4)
	_, err := rand.Read(tokenarray)
	if err != nil {
		log.Fatal(err)
	}
	return tokenarray
}

func GetPeer(key string) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, randToken *[]byte) {
	decodedBytes := base64Decoding(key)
	nonce := decodedBytes[:12] // nonce is 12 bytes
	cipherText := decodedBytes[12:]
	return aesgsmDecryct(secret, nonce, cipherText)
}

// ENCRYPTATION BASED ON AES-GCM
// based on https://pkg.go.dev/crypto/cipher#NewGCM  examples
func aesgsmEncryct(secret []byte, candidates_response *connectionCandidates) (nonce []byte, ciphertext []byte) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		log.Fatal(err.Error())
	}
	//create aescgm encryct
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatal(err)
	}

	// nonce (number used ince) is a random number used only 1 time to AES (IV — Initialization Vector)
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat
	nonce = make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic(err.Error())
	}

	endpointInBytes := encodeTo16bytes(candidates_response)

	ciphertext = aesgcm.Seal(nil, nonce, endpointInBytes, nil)
	return nonce, ciphertext
}

func encodeTo16bytes(candidates_response *connectionCandidates) (encodedEndpoint []byte) {
	// encode connectionCandidates struct to byte buffer
	// build 16 byte array with local and stun address to better compression
	// ip = 4 bytes , port = 2 bytes, random token = 4bytes (avoid spoofing)
	response := make([]byte, 16)
	// local address
	copy(response[0:4], candidates_response.LocalAddress.IP.To4())
	binary.BigEndian.PutUint16(response[4:6], uint16(candidates_response.LocalAddress.Port))

	//stun remote
	copy(response[6:10], candidates_response.StunPublicAddress.IP.To4())
	binary.BigEndian.PutUint16(response[10:12], uint16(candidates_response.StunPublicAddress.Port))

	//rand token
	copy(response[12:16], candidates_response.RandToken)

	return response
}

// Base64 codification
func base64Enconding(data []byte) (b64token string) {
	return base64.RawURLEncoding.EncodeToString(data)
}

func aesgsmDecryct(secret []byte, nonce []byte, ciphertext []byte) (peerLocal *net.UDPAddr, peerRemote *net.UDPAddr, randToken *[]byte) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		panic(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}

	endpoint := decodeFrom16bytes(plaintext)
	peerLocal = endpoint.LocalAddress
	peerRemote = endpoint.StunPublicAddress
	randToken = &endpoint.RandToken

	return peerLocal, peerRemote, randToken
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

	//random token
	randToken := encodedText[12:16]

	localResponse := net.UDPAddr{IP: localIP, Port: localPort}
	remoteResponse := net.UDPAddr{IP: remoteIP, Port: remotePort}
	decodedResponse = &connectionCandidates{LocalAddress: &localResponse, StunPublicAddress: &remoteResponse, RandToken: randToken}
	return decodedResponse
}

func base64Decoding(b64token string) (decoded []byte) {
	decode, err := base64.RawURLEncoding.DecodeString(b64token)
	if err != nil {
		log.Fatal(err.Error())
	}
	return decode
}
