package encryption

import (
	"bytes"
	stundiscover "cmd/ftu/main.go/internal/stun-discover"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"log"
)

// based on https://pkg.go.dev/crypto/cipher#NewGCM  examples
func AesgsmEncryct(key []byte, endpoint *stundiscover.PublicEndpoint) (nonce []byte, ciphertext []byte) {
	// encode Public Endpoint struct to byte buffer
	var endpointInBytes bytes.Buffer
	enc := gob.NewEncoder(&endpointInBytes)

	err := enc.Encode(endpoint)
	if err != nil {
		log.Fatal(err.Error())
	}

	block, err := aes.NewCipher(key)
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

	ciphertext = aesgcm.Seal(nil, nonce, endpointInBytes.Bytes(), nil)
	fmt.Printf("%x\n", ciphertext)
	return nonce, ciphertext
}
