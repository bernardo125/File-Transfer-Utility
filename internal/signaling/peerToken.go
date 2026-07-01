package encryption

import (
	"bytes"
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"log"
)

var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun.cloudflare.com:3478",
}
var secret, _ = hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

// Orchestrate encryptation and base64 codification
func CreateKey() (key string) {
	stunClient := stundiscover.Client{}
	response, _ := stunClient.SendBindingRequest(DefaultSTUNServers[0])
	nonce, ciphertext := aesgsmEncryct(secret, &response)
	finalPacket := append(nonce, ciphertext...)
	key = base64Enconding(finalPacket)
	return key
}

func GetPeer(key string) (peer *stundiscover.PublicEndpoint) {
	decodedBytes := base64Decoding(key)
	nonce := decodedBytes[:12] // nonce is 12 bytes
	cipherText := decodedBytes[12:]
	return aesgsmDecryct(secret, nonce, cipherText)
}

// ENCRYPTATION BASED ON AES-GCM
// based on https://pkg.go.dev/crypto/cipher#NewGCM  examples
func aesgsmEncryct(secret []byte, endpoint *stundiscover.PublicEndpoint) (nonce []byte, ciphertext []byte) {
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

	endpointInBytes := endpointEncode(endpoint)

	ciphertext = aesgcm.Seal(nil, nonce, endpointInBytes, nil)
	return nonce, ciphertext
}

func endpointEncode(endpoint *stundiscover.PublicEndpoint) (encodedEndpoint []byte) {
	// encode Public Endpoint struct to byte buffer
	var endpointInBytes bytes.Buffer
	enc := gob.NewEncoder(&endpointInBytes)

	err := enc.Encode(endpoint)
	if err != nil {
		log.Fatal(err.Error())
	}
	return endpointInBytes.Bytes()
}

func aesgsmDecryct(secret []byte, nonce []byte, ciphertext []byte) (otherPeerEndpoint *stundiscover.PublicEndpoint) {
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

func endpointDecode(encodedText []byte) (decodedEndpoint *stundiscover.PublicEndpoint) {
	// Decode byte buffer to Public Endpoint struct
	bytesToPeerEndpoint := bytes.NewBuffer(encodedText)
	decEndpoint := gob.NewDecoder(bytesToPeerEndpoint)

	var endpoint stundiscover.PublicEndpoint
	err := decEndpoint.Decode(&endpoint)
	if err != nil {
		log.Fatal(err.Error())
	}

	return &endpoint
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
