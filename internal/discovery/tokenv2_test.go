package discovery

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func rawV2(t *testing.T, token string) []byte {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, tokenV2Prefix))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func encodeRawV2(b []byte) string { return tokenV2Prefix + base64.RawURLEncoding.EncodeToString(b) }

func withChecksum(b []byte) []byte {
	sum := sha256.Sum256(b)
	return append(append([]byte(nil), b...), sum[:4]...)
}

// Detects: a change in the v2 layout making two builds disagree about addresses, key or expiry.
func TestTokenV2RoundTrip(t *testing.T) {
	pub := newPub(t)
	exp := time.Unix(1_900_000_000, 0)
	for name, v6 := range map[string]*net.UDPAddr{"no ipv6": nil, "ipv6": udp("2001:db8::7", 41000)} {
		t.Run(name, func(t *testing.T) {
			tok := buildTokenV2(udp("192.168.1.20", 51000), v6, udp("203.0.113.7", 62000), pub, exp)
			if !strings.HasPrefix(tok, "ftu2.") {
				t.Fatalf("token %q has no v2 prefix", tok)
			}
			p, err := ParseToken(tok, time.Unix(1_800_000_000, 0))
			if err != nil {
				t.Fatal(err)
			}
			if p.Version != 2 || !bytes.Equal(p.PublicKey, pub) || !p.ExpiresAt.Equal(exp) {
				t.Fatalf("decoded %+v", p)
			}
			if p.Local.Port != 51000 || !p.Stun.IP.Equal(net.ParseIP("203.0.113.7")) {
				t.Fatalf("addresses %v %v", p.Local, p.Stun)
			}
			if (v6 == nil) != (p.IPv6 == nil) || (v6 != nil && (!p.IPv6.IP.Equal(v6.IP) || p.IPv6.Port != v6.Port)) {
				t.Fatalf("ipv6 = %v, want %v", p.IPv6, v6)
			}
			if wantLen := map[bool]int{false: 56, true: 74}[v6 != nil]; len(rawV2(t, tok)) != wantLen {
				t.Fatalf("raw length %d, want %d", len(rawV2(t, tok)), wantLen)
			}
		})
	}
}

// Detects: accidental edits (typo, truncated copy, flipped byte) being accepted. Every single
// byte of the token is covered by the checksum or by structural checks.
func TestTokenV2DetectsAnyCorruptedByte(t *testing.T) {
	tok := buildTokenV2(udp("192.168.1.20", 51000), udp("2001:db8::7", 41000), udp("203.0.113.7", 62000), newPub(t), time.Now().Add(time.Hour))
	raw := rawV2(t, tok)
	for i := range raw {
		mut := append([]byte(nil), raw...)
		mut[i] ^= 0x01
		if _, err := ParseToken(encodeRawV2(mut), time.Now()); err == nil {
			t.Errorf("flipping a bit of byte %d went unnoticed", i)
		}
	}
	for n := 1; n < len(raw); n += 7 {
		if _, err := ParseToken(encodeRawV2(raw[:n]), time.Now()); err == nil {
			t.Errorf("token truncated to %d bytes accepted", n)
		}
	}
}

func TestTokenV2Expiry(t *testing.T) {
	exp := time.Unix(1_900_000_000, 0)
	tok := buildTokenV2(udp("192.168.1.20", 5), nil, udp("203.0.113.7", 6), newPub(t), exp)
	if _, err := ParseToken(tok, exp.Add(-time.Second)); err != nil {
		t.Fatalf("token rejected before it expires: %v", err)
	}
	for _, now := range []time.Time{exp, exp.Add(time.Hour)} {
		if _, err := ParseToken(tok, now); !errors.Is(err, ErrTokenExpired) {
			t.Fatalf("now=%v: err = %v, want ErrTokenExpired", now, err)
		}
	}
}

func TestTokenV2RejectsStructuralProblems(t *testing.T) {
	pub := newPub(t)
	good := rawV2(t, buildTokenV2(udp("192.168.1.20", 5), nil, udp("203.0.113.7", 6), pub, time.Now().Add(time.Hour)))
	body := good[:len(good)-4]
	mk := func(f func(b []byte)) string {
		b := append([]byte(nil), body...)
		f(b)
		return encodeRawV2(withChecksum(b)) // checksum is valid: only the structure is wrong
	}
	cases := map[string]string{
		"bad magic":                   mk(func(b []byte) { b[0] = 'X' }),
		"version 3":                   mk(func(b []byte) { b[2] = 3 }),
		"version 1":                   mk(func(b []byte) { b[2] = 1 }),
		"unknown flag":                mk(func(b []byte) { b[3] = 0x02 }),
		"ipv6 flag but no ipv6 bytes": mk(func(b []byte) { b[3] = 1 }),
		"extra bytes":                 encodeRawV2(withChecksum(append(append([]byte(nil), body...), 0, 0))),
		"empty payload":               tokenV2Prefix,
		"not base64":                  tokenV2Prefix + "!!!",
		"padded base64":               encodeRawV2(good) + "==",
	}
	for name, tok := range cases {
		if _, err := ParseToken(tok, time.Now()); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Documents (F-03): the checksum is NOT authentication. A man in the middle can swap the key and
// recompute it, which is exactly why the pairing fingerprint must be confirmed by both users.
func TestTokenV2ChecksumDoesNotAuthenticate_FingerprintDoes(t *testing.T) {
	real, mitm, me := newPub(t), newPub(t), newPub(t)
	tok := buildTokenV2(udp("192.168.1.20", 5), nil, udp("203.0.113.7", 6), real, time.Now().Add(time.Hour))
	raw := rawV2(t, tok)
	body := append([]byte(nil), raw[:len(raw)-4]...)
	copy(body[4:36], mitm)
	forged, err := ParseToken(encodeRawV2(withChecksum(body)), time.Now())
	if err != nil {
		t.Fatalf("recomputed checksum should pass (that is the point): %v", err)
	}
	if !bytes.Equal(forged.PublicKey, mitm) {
		t.Fatal("forged key not adopted")
	}
	if PairingFingerprint(me, real) == PairingFingerprint(me, forged.PublicKey) {
		t.Fatal("a substituted key must change the fingerprint the users compare")
	}
}

func TestPairingFingerprint(t *testing.T) {
	a, b, c := newPub(t), newPub(t), newPub(t)
	if PairingFingerprint(a, b) != PairingFingerprint(b, a) {
		t.Fatal("both sides must compute the same value")
	}
	if PairingFingerprint(a, b) == PairingFingerprint(a, c) || PairingFingerprint(a, b) == PairingFingerprint(c, b) {
		t.Fatal("different keys must give different fingerprints")
	}
	fp := PairingFingerprint(a, b)
	parts := strings.Split(fp, "-")
	if len(parts) != 5 {
		t.Fatalf("format %q", fp)
	}
	for _, p := range parts {
		if len(p) != 4 || strings.Trim(p, "0123456789ABCDEF") != "" {
			t.Fatalf("format %q", fp)
		}
	}
}

// Legacy tokens keep working (migration) but are reported as version 1.
func TestParseTokenAcceptsLegacyV1(t *testing.T) {
	pub := newPub(t)
	tok := buildTokenV1(udp("192.168.1.20", 51000), nil, udp("203.0.113.7", 62000), ed25519.PublicKey(pub))
	p, err := ParseToken(tok, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 1 || !bytes.Equal(p.PublicKey, pub) || !p.ExpiresAt.IsZero() {
		t.Fatalf("legacy token decoded as %+v", p)
	}
	if _, _, _, k, err := GetPeer(tok); err != nil || !bytes.Equal(*k, pub) {
		t.Fatalf("GetPeer on a legacy token: %v", err)
	}
}

func FuzzParseToken(f *testing.F) {
	pub := ed25519.PublicKey(make([]byte, 32))
	f.Add(buildTokenV2(udp("10.0.0.1", 1), nil, udp("10.0.0.2", 2), pub, time.Now().Add(time.Hour)))
	f.Add(buildTokenV1(udp("10.0.0.1", 1), nil, udp("10.0.0.2", 2), pub))
	f.Add("ftu2.")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		p, err := ParseToken(s, time.Now())
		if err != nil {
			return
		}
		if p.Local == nil || p.Stun == nil || len(p.PublicKey) != ed25519.PublicKeySize || p.Local.Port == 0 || p.Stun.Port == 0 {
			t.Fatalf("accepted unusable token %q -> %+v", s, p)
		}
	})
}
