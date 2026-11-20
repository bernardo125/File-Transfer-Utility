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
	"strings"
	"unicode"

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
	var stun_response *net.UDPAddr
	var err error
	for _, server := range DefaultSTUNServers {
		if stun_response, err = stunClient.SendBindingRequest(conn, server); err == nil {
			break
		}
		log.Printf("[Discovery] STUN server %s failed: %v", server, err)
	}
	if err != nil {
		// No public mapping available (offline / STUN blocked): advertise an unspecified
		// address so the token is still valid; LAN and IPv6 candidates keep working.
		log.Printf("[Discovery] STUN failed, continuing without public IPv4: %v", err)
		stun_response = &net.UDPAddr{IP: net.IPv4zero, Port: local_ipv4.Port}
	}
	public, priv := ed55519TokenGen()
	return BuildToken(local_ipv4, public_ipv6, stun_response, public), priv
}

// BuildToken serializes the connection candidates into the shareable (obfuscated) token.
// It performs no network I/O, which makes the token format testable without STUN.
// ipv6 may be nil; local and stun must be IPv4 addresses.
func BuildToken(local, ipv6, stun *net.UDPAddr, public ed25519.PublicKey) string {
	if ipv6 != nil && (len(ipv6.IP) == 0 || ipv6.IP.IsUnspecified()) {
		ipv6 = nil // no IPv6 connectivity: do not advertise "::"
	}
	candidates := connectionCandidates{LocalAddressIPV4: local, PublicAddressIPV6: ipv6, StunPublicAddressIPV4: stun, Ed25519PublicToken: &public}
	nonce, ciphertext := chacha20Encryct(secret, &candidates)
	return base64Enconding(append(nonce, ciphertext...))
}

// generate ed55519 key to avoid spoofing and mitm attacks
func ed55519TokenGen() (public ed25519.PublicKey, priv ed25519.PrivateKey) {
	public, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return public, priv
}

func GetPeer(key string) (peerLocalIPV4 *net.UDPAddr, peerRemoteIPV4 *net.UDPAddr, peerPublicIPV6 *net.UDPAddr, publickey *ed25519.PublicKey, err error) {
	if len(key) < 50 {
		return nil, nil, nil, nil, fmt.Errorf("That's not a token")
	}
	decodedBytes, err := base64Decoding(key)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("That's not a token: %w", err)
	}
	// nonce (12) + minimal plaintext (44: key + local + stun); IPv6 form is 62
	plainLen := len(decodedBytes) - chacha20.NonceSize
	if plainLen != 44 && plainLen != 62 {
		return nil, nil, nil, nil, fmt.Errorf("That's not a token: unexpected length %d", len(decodedBytes))
	}
	nonce := decodedBytes[:chacha20.NonceSize]
	cipherText := decodedBytes[chacha20.NonceSize:]
	local, remote, v6, pub, err := chacha20Decryct(secret, nonce, cipherText)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if v6, err = validateCandidates(local, remote, v6); err != nil {
		return nil, nil, nil, nil, err
	}
	return local, remote, v6, pub, nil
}

// validateCandidates rejects candidate addresses that can never be a peer, so a crafted token
// cannot make the app send probes to multicast/broadcast/port-0 destinations. It returns the IPv6
// candidate, dropping it when it is the unspecified placeholder older builds put in every token.
//
// The STUN address may be 0.0.0.0: that is the documented "no public mapping" marker (STUN was
// unreachable); it is accepted here and simply never probed.
func validateCandidates(local, stun, v6 *net.UDPAddr) (*net.UDPAddr, error) {
	if local == nil || stun == nil {
		return nil, fmt.Errorf("That's not a token: missing address")
	}
	if local.Port == 0 || stun.Port == 0 {
		return nil, fmt.Errorf("That's not a token: port 0")
	}
	if local.IP.IsUnspecified() || local.IP.IsMulticast() || local.IP.Equal(net.IPv4bcast) {
		return nil, fmt.Errorf("That's not a token: unusable local address %v", local.IP)
	}
	if stun.IP.IsMulticast() || stun.IP.Equal(net.IPv4bcast) {
		return nil, fmt.Errorf("That's not a token: unusable public address %v", stun.IP)
	}
	if v6 == nil || v6.IP.IsUnspecified() {
		return nil, nil
	}
	if v6.Port == 0 || v6.IP.IsMulticast() {
		return nil, fmt.Errorf("That's not a token: unusable IPv6 address")
	}
	return v6, nil
}

// CleanToken removes whitespace and invisible formatting characters that creep in when a token
// is copied from a chat or a web page. Tokens never contain any.
func CleanToken(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
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
		// public address ipv6 (16 bytes: 44..60)
		copy(response[44:60], candidates_response.PublicAddressIPV6.IP.To16())
		// public address ipv6 port (2 bytes: 60..62)
		binary.BigEndian.PutUint16(response[60:62], uint16(candidates_response.PublicAddressIPV6.Port))
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

func chacha20Decryct(secret []byte, nonce []byte, ciphertext []byte) (peerLocalIPV4 *net.UDPAddr, peerRemoteIPV4 *net.UDPAddr, peerPublicIPV6 *net.UDPAddr, publickey *ed25519.PublicKey, err error) {
	cipher, err := chacha20.NewUnauthenticatedCipher(secret, nonce)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	// Decrypt the ciphertext
	cipher.XORKeyStream(plaintext, ciphertext)

	endpoint := decodeFrom16bytes(plaintext)
	peerLocalIPV4 = endpoint.LocalAddressIPV4
	peerRemoteIPV4 = endpoint.StunPublicAddressIPV4
	peerPublicIPV6 = endpoint.PublicAddressIPV6
	publickey = endpoint.Ed25519PublicToken

	return peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, publickey, nil
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
	if len(encodedText) >= 62 {
		// public address ipv6 (16 bytes: 44..60)
		publicIPV6 := make(net.IP, 16)
		copy(publicIPV6, encodedText[44:60])
		publicPortIPV6 := int(binary.BigEndian.Uint16(encodedText[60:62])) // int to net.UDPAddr struct
		return &net.UDPAddr{IP: publicIPV6, Port: publicPortIPV6}
	}
	return nil
}

func base64Decoding(b64token string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(b64token)
}
