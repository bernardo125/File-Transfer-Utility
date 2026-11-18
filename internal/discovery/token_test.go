package discovery

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

func newPub(t testing.TB) ed25519.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func udp(ip string, port int) *net.UDPAddr { return &net.UDPAddr{IP: net.ParseIP(ip), Port: port} }

// Detects: any change in the token layout that makes two peers disagree about addresses/keys.
func TestTokenRoundTrip(t *testing.T) {
	pub := newPub(t)
	tests := []struct {
		name string
		v6   *net.UDPAddr
	}{
		{"without ipv6", nil},
		{"with public ipv6", udp("2001:db8::1", 40000)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := BuildToken(udp("192.168.1.20", 51000), tc.v6, udp("203.0.113.7", 62000), pub)
			local, remote, v6, key, err := GetPeer(token)
			if err != nil {
				t.Fatalf("GetPeer: %v", err)
			}
			if !local.IP.Equal(net.ParseIP("192.168.1.20")) || local.Port != 51000 {
				t.Errorf("local = %v", local)
			}
			if !remote.IP.Equal(net.ParseIP("203.0.113.7")) || remote.Port != 62000 {
				t.Errorf("remote = %v", remote)
			}
			if !bytes.Equal(*key, pub) {
				t.Error("public key not preserved")
			}
			if tc.v6 != nil && (v6 == nil || !v6.IP.Equal(tc.v6.IP) || v6.Port != tc.v6.Port) {
				t.Errorf("ipv6 = %v, want %v", v6, tc.v6)
			}
			if tc.v6 == nil && v6 != nil {
				t.Errorf("unexpected ipv6 %v", v6)
			}
		})
	}
}

// Detects: user-typed garbage crashing the process (log.Fatal / index out of range).
func TestGetPeerRejectsMalformedTokens(t *testing.T) {
	good := BuildToken(udp("10.0.0.1", 1), nil, udp("10.0.0.2", 2), newPub(t))
	tests := []struct{ name, token string }{
		{"empty", ""},
		{"too short", "abc"},
		{"not base64", strings.Repeat("!", 80)},
		{"standard base64 padding", good + "=="},
		{"truncated valid token", good[:len(good)-8]},
		{"extended valid token", good + "AAAA"},
		{"only nonce", base64Enc(make([]byte, 12))},
		{"leading space", " " + good},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, _, err := GetPeer(tc.token); err == nil {
				t.Fatalf("malformed token accepted: %q", tc.token)
			}
		})
	}
}

func base64Enc(b []byte) string { return base64Enconding(b) }

// Invariant: GetPeer never panics or exits, and a nil error always comes with usable values.
func FuzzGetPeer(f *testing.F) {
	f.Add(BuildToken(udp("10.0.0.1", 1), nil, udp("10.0.0.2", 2), make([]byte, 32)))
	f.Add(BuildToken(udp("10.0.0.1", 1), udp("2001:db8::1", 9), udp("10.0.0.2", 2), make([]byte, 32)))
	f.Add("")
	f.Add(strings.Repeat("A", 50))
	f.Fuzz(func(t *testing.T, token string) {
		local, remote, _, key, err := GetPeer(token)
		if err != nil {
			return
		}
		if local == nil || remote == nil || key == nil || len(*key) != ed25519.PublicKeySize {
			t.Fatalf("nil error but unusable result for %q", token)
		}
	})
}

func stunResponse(attrType uint16, attrLen uint16, family byte, port uint16, ip [4]byte) []byte {
	b := make([]byte, 20, 32)
	b = binary.BigEndian.AppendUint16(b, attrType)
	b = binary.BigEndian.AppendUint16(b, attrLen)
	b = append(b, 0, family)
	b = binary.BigEndian.AppendUint16(b, port^uint16(stunMagicCookie>>16))
	cookie := binary.BigEndian.AppendUint32(nil, stunMagicCookie)
	for i := range ip {
		b = append(b, ip[i]^cookie[i])
	}
	return b
}

// Detects: wrong XOR decoding of the mapped address (peers would punch the wrong endpoint).
func TestParseXORMappedAddress(t *testing.T) {
	got, err := parseXORMappedAddress(stunResponse(0x0020, 8, 0x01, 54321, [4]byte{203, 0, 113, 9}))
	if err != nil {
		t.Fatal(err)
	}
	if !got.IP.Equal(net.IPv4(203, 0, 113, 9)) || got.Port != 54321 {
		t.Fatalf("got %v, want 203.0.113.9:54321", got)
	}
}

// Detects: hostile / stray datagrams on the STUN socket crashing the app (log.Fatal, slice panic).
func TestParseXORMappedAddressRejectsBadResponses(t *testing.T) {
	tests := []struct {
		name string
		buf  []byte
	}{
		{"nil", nil},
		{"header only", make([]byte, 20)},
		{"truncated attribute", stunResponse(0x0020, 8, 1, 1, [4]byte{})[:27]},
		{"wrong attribute type", stunResponse(0x0001, 8, 1, 1, [4]byte{})},
		{"ipv6 family", stunResponse(0x0020, 8, 0x02, 1, [4]byte{})},
		{"attribute length lies", stunResponse(0x0020, 200, 1, 1, [4]byte{})},
		{"negative-looking length", stunResponse(0x0020, 0xFF80, 1, 1, [4]byte{})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := parseXORMappedAddress(tc.buf); err == nil {
				t.Fatalf("accepted bad response, got %v", got)
			}
		})
	}
}

func FuzzParseXORMappedAddress(f *testing.F) {
	f.Add(stunResponse(0x0020, 8, 1, 3478, [4]byte{1, 2, 3, 4}))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		if a, err := parseXORMappedAddress(b); err == nil && (a == nil || len(a.IP) != 4) {
			t.Fatalf("nil error with unusable address %v", a)
		}
	})
}
