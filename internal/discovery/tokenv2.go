package discovery

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Token format v2 (public identifier, NOT a secret)
//
//	"ftu2." + base64url( "FT" | version(1)=2 | flags(1) | ed25519 pubkey(32) |
//	                     local ipv4(4)+port(2) | stun ipv4(4)+port(2) |
//	                     [ipv6(16)+port(2) if flags&1] | expiry unix seconds(4) |
//	                     checksum(4) )
//
// checksum = first 4 bytes of SHA-256 over every preceding byte. It detects typos, truncation and
// accidental edits only; anyone can recompute it, so it does NOT authenticate the token. The peer
// is authenticated by the mutual key pin plus the fingerprint both users must confirm
// (PairingFingerprint) before any data channel is used.
const (
	tokenV2Prefix = "ftu2."
	tokenV2Flags6 = 0x01
)

// TokenValidity is how long a freshly generated token is accepted for.
var TokenValidity = 24 * time.Hour

// ErrTokenExpired is returned for a well-formed token whose expiry has passed.
var ErrTokenExpired = errors.New("token expired")

// PeerToken is a decoded and validated token.
type PeerToken struct {
	Version   int // 1 = legacy (obfuscated, unauthenticated, no expiry), 2 = current
	Local     *net.UDPAddr
	Stun      *net.UDPAddr
	IPv6      *net.UDPAddr // nil when absent
	PublicKey ed25519.PublicKey
	ExpiresAt time.Time // zero for v1
}

func appendAddr4(b []byte, a *net.UDPAddr) []byte {
	ip := make([]byte, 4)
	if a != nil {
		if v4 := a.IP.To4(); v4 != nil {
			copy(ip, v4)
		}
	}
	b = append(b, ip...)
	port := 0
	if a != nil {
		port = a.Port
	}
	return binary.BigEndian.AppendUint16(b, uint16(port))
}

func buildTokenV2(local, ipv6, stun *net.UDPAddr, public ed25519.PublicKey, expires time.Time) string {
	hasV6 := ipv6 != nil && len(ipv6.IP) > 0 && !ipv6.IP.IsUnspecified()
	var flags byte
	if hasV6 {
		flags |= tokenV2Flags6
	}
	b := make([]byte, 0, 80)
	b = append(b, 'F', 'T', 2, flags)
	b = append(b, public...)
	b = appendAddr4(b, local)
	b = appendAddr4(b, stun)
	if hasV6 {
		b = append(b, ipv6.IP.To16()...)
		b = binary.BigEndian.AppendUint16(b, uint16(ipv6.Port))
	}
	b = binary.BigEndian.AppendUint32(b, uint32(expires.Unix()))
	sum := sha256.Sum256(b)
	b = append(b, sum[:4]...)
	return tokenV2Prefix + base64.RawURLEncoding.EncodeToString(b)
}

func parseTokenV2(token string, now time.Time) (*PeerToken, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, tokenV2Prefix))
	if err != nil {
		return nil, fmt.Errorf("That's not a token: %w", err)
	}
	const fixed = 4 + ed25519.PublicKeySize + 6 + 6 + 4 + 4 // header+key+2 addrs+expiry+checksum
	if len(raw) < fixed {
		return nil, fmt.Errorf("That's not a token: too short")
	}
	if raw[0] != 'F' || raw[1] != 'T' {
		return nil, fmt.Errorf("That's not a token: bad magic")
	}
	if raw[2] != 2 {
		return nil, fmt.Errorf("unsupported token version %d: update the app", raw[2])
	}
	flags := raw[3]
	if flags&^byte(tokenV2Flags6) != 0 {
		return nil, fmt.Errorf("That's not a token: unknown flags 0x%02x", flags)
	}
	want := fixed
	if flags&tokenV2Flags6 != 0 {
		want += 18
	}
	if len(raw) != want {
		return nil, fmt.Errorf("That's not a token: unexpected length %d", len(raw))
	}
	sum := sha256.Sum256(raw[:len(raw)-4])
	if subtle.ConstantTimeCompare(sum[:4], raw[len(raw)-4:]) != 1 {
		return nil, fmt.Errorf("That's not a token: checksum mismatch (typo or truncated copy)")
	}

	p := 4
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, raw[p:p+32])
	p += 32
	addr4 := func() *net.UDPAddr {
		ip := make(net.IP, 4)
		copy(ip, raw[p:p+4])
		port := int(binary.BigEndian.Uint16(raw[p+4 : p+6]))
		p += 6
		return &net.UDPAddr{IP: ip, Port: port}
	}
	local, stun := addr4(), addr4()
	var v6 *net.UDPAddr
	if flags&tokenV2Flags6 != 0 {
		ip := make(net.IP, 16)
		copy(ip, raw[p:p+16])
		v6 = &net.UDPAddr{IP: ip, Port: int(binary.BigEndian.Uint16(raw[p+16 : p+18]))}
		p += 18
	}
	expires := time.Unix(int64(binary.BigEndian.Uint32(raw[p:p+4])), 0)
	if !now.Before(expires) {
		return nil, fmt.Errorf("%w: it expired at %s; ask your peer for a fresh one", ErrTokenExpired, expires.Format(time.RFC3339))
	}
	v6, err = validateCandidates(local, stun, v6)
	if err != nil {
		return nil, err
	}
	return &PeerToken{Version: 2, Local: local, Stun: stun, IPv6: v6, PublicKey: pub, ExpiresAt: expires}, nil
}

// ParseToken decodes a token of any supported version, validating format, checksum (v2), expiry
// (v2) and candidate addresses. now is injected for tests.
func ParseToken(token string, now time.Time) (*PeerToken, error) {
	if strings.HasPrefix(token, tokenV2Prefix) {
		return parseTokenV2(token, now)
	}
	local, stun, v6, pub, err := parseTokenV1(token)
	if err != nil {
		return nil, err
	}
	return &PeerToken{Version: 1, Local: local, Stun: stun, IPv6: v6, PublicKey: *pub}, nil
}

// PairingFingerprint is the short code both users compare out of band before any data is
// exchanged. It covers BOTH identity keys in a canonical order, so each side computes the same
// value, and a man in the middle who substituted a key on either leg produces different codes on
// the two screens. 80 bits, shown as 5 groups of 4 hex digits.
func PairingFingerprint(a, b ed25519.PublicKey) string {
	lo, hi := a, b
	if string(lo) > string(hi) {
		lo, hi = hi, lo
	}
	h := sha256.New()
	h.Write([]byte("ftu-pairing-fingerprint-v1"))
	h.Write(lo)
	h.Write(hi)
	d := h.Sum(nil)[:10]
	groups := make([]string, 5)
	for i := range groups {
		groups[i] = fmt.Sprintf("%02X%02X", d[2*i], d[2*i+1])
	}
	return strings.Join(groups, "-")
}
