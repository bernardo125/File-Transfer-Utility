package encryption

import (
	"bytes"
	localdiscover "cmd/ftu/main.go/internal/local-discover"
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
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
}

// Orchestrate encryptation and base64 codification
func CreateKey(conn *net.UDPConn) (key string) {
	stunClient := stundiscover.Client{}
	local_response := localdiscover.GetLocalUDPAddr(conn)
	stun_response, _ := stunClient.SendBindingRequest(conn, DefaultSTUNServers[0])
	connCandidates := connectionCandidates{LocalAddress: local_response, StunPublicAddress: stun_response}
	nonce, ciphertext := aesgsmEncryct(secret, &connCandidates)
	finalPacket := append(nonce, ciphertext...)
	key = base64Enconding(finalPacket)
	return key
}

func GetPeer(key string) (peer *connectionCandidates) {
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

	endpointInBytes := endpointEncode(candidates_response)

	ciphertext = aesgcm.Seal(nil, nonce, endpointInBytes, nil)
	return nonce, ciphertext
}

func endpointEncode(candidates_response *connectionCandidates) (encodedEndpoint []byte) {
	// encode connectionCandidates struct to byte buffer
	var endpointInBytes bytes.Buffer
	enc := gob.NewEncoder(&endpointInBytes)

	err := enc.Encode(candidates_response)
	if err != nil {
		log.Fatal(err.Error())
	}
	return endpointInBytes.Bytes()
}

func aesgsmDecryct(secret []byte, nonce []byte, ciphertext []byte) (candidates_response *connectionCandidates) {
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

	endpoint := endpointDecode(plaintext)

	return endpoint
}

func endpointDecode(encodedText []byte) (decodedResponse *connectionCandidates) {
	// Decode byte buffer to Public Endpoint struct
	bytesToPeerEndpoint := bytes.NewBuffer(encodedText)
	decEndpoint := gob.NewDecoder(bytesToPeerEndpoint)

	var response connectionCandidates
	err := decEndpoint.Decode(&response)
	if err != nil {
		log.Fatal(err.Error())
	}

	return &response
}

// Base64 codification
func base64Enconding(data []byte) (b64token string) {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64Decoding(b64token string) (decoded []byte) {
	decode, err := base64.RawURLEncoding.DecodeString(b64token)
	if err != nil {
		log.Fatal(err.Error())
	}
	return decode
}
