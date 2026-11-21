package discovery

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// bindingResponse builds a complete Binding Success Response with the given attributes.
func bindingResponse(txID [12]byte, attrs ...[]byte) []byte {
	var body []byte
	for _, a := range attrs {
		body = append(body, a...)
	}
	b := make([]byte, 20, 20+len(body))
	binary.BigEndian.PutUint16(b[0:2], stunResponseType)
	binary.BigEndian.PutUint16(b[2:4], uint16(len(body)))
	binary.BigEndian.PutUint32(b[4:8], stunMagicCookie)
	copy(b[8:20], txID[:])
	return append(b, body...)
}

func xorMappedAttr(port uint16, ip [4]byte) []byte {
	a := binary.BigEndian.AppendUint16(nil, stunAttrXORMappedAddress)
	a = binary.BigEndian.AppendUint16(a, 8)
	a = append(a, 0, 1)
	a = binary.BigEndian.AppendUint16(a, port^uint16(stunMagicCookie>>16))
	cookie := binary.BigEndian.AppendUint32(nil, stunMagicCookie)
	for i := range ip {
		a = append(a, ip[i]^cookie[i])
	}
	return a
}

func softwareAttr(s string) []byte { // attribute 0x8022, padded to 4 bytes
	a := binary.BigEndian.AppendUint16(nil, 0x8022)
	a = binary.BigEndian.AppendUint16(a, uint16(len(s)))
	a = append(a, s...)
	for len(a)%4 != 0 {
		a = append(a, 0)
	}
	return a
}

var testTx = [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

// Detects (F-08): the parser accepting anything that is not the answer to our request.
func TestParseBindingResponseValidatesHeader(t *testing.T) {
	good := bindingResponse(testTx, xorMappedAttr(54321, [4]byte{203, 0, 113, 9}))
	got, err := parseBindingResponse(good, testTx)
	if err != nil || !got.IP.Equal(net.IPv4(203, 0, 113, 9)) || got.Port != 54321 {
		t.Fatalf("good response: %v %v", got, err)
	}
	// real servers often put other attributes first
	withSoftware := bindingResponse(testTx, softwareAttr("coturn-4.6"), xorMappedAttr(1000, [4]byte{198, 51, 100, 7}))
	if got, err := parseBindingResponse(withSoftware, testTx); err != nil || got.Port != 1000 {
		t.Fatalf("attribute after SOFTWARE not found: %v %v", got, err)
	}

	mutate := func(f func(b []byte) []byte) []byte { return f(append([]byte(nil), good...)) }
	bad := map[string][]byte{
		"wrong transaction id": bindingResponse([12]byte{9}, xorMappedAttr(1, [4]byte{1, 2, 3, 4})),
		"binding request type": mutate(func(b []byte) []byte { binary.BigEndian.PutUint16(b[0:2], 0x0001); return b }),
		"error response type":  mutate(func(b []byte) []byte { binary.BigEndian.PutUint16(b[0:2], 0x0111); return b }),
		"bad cookie":           mutate(func(b []byte) []byte { b[4] ^= 0xff; return b }),
		"length too big":       mutate(func(b []byte) []byte { binary.BigEndian.PutUint16(b[2:4], 400); return b }),
		"length not aligned":   mutate(func(b []byte) []byte { binary.BigEndian.PutUint16(b[2:4], 7); return b }),
		"trailing bytes":       append(append([]byte(nil), good...), 0, 0, 0, 0),
		"truncated":            good[:len(good)-3],
		"header only":          bindingResponse(testTx),
		"too short":            good[:10],
		"attribute overruns":   bindingResponse(testTx, []byte{0x80, 0x22, 0x00, 0x40, 0, 0, 0, 0}),
	}
	for name, b := range bad {
		if got, err := parseBindingResponse(b, testTx); err == nil {
			t.Errorf("%s: accepted, got %v", name, got)
		}
	}
}

func FuzzParseBindingResponse(f *testing.F) {
	f.Add(bindingResponse(testTx, xorMappedAttr(3478, [4]byte{1, 2, 3, 4})))
	f.Add(bindingResponse(testTx, softwareAttr("x"), xorMappedAttr(1, [4]byte{9, 9, 9, 9})))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		if a, err := parseBindingResponse(b, testTx); err == nil && (a == nil || len(a.IP) != 4) {
			t.Fatalf("nil error with unusable address %v", a)
		}
	})
}

// Detects (F-08): forged replies (wrong txid, or right txid from another host) poisoning the
// address that is advertised in the token.
func TestSendBindingRequestIgnoresForgedAndStrayReplies(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	stranger, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer stranger.Close()
	client, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	go func() {
		buf := make([]byte, 64)
		n, from, err := server.ReadFromUDP(buf)
		if err != nil || n < 20 {
			return
		}
		var tx [12]byte
		copy(tx[:], buf[8:20])
		forged := xorMappedAttr(6666, [4]byte{6, 6, 6, 6})
		// 1) right source, wrong transaction id
		server.WriteToUDP(bindingResponse([12]byte{0xEE}, forged), from)
		// 2) right transaction id, but from a host that is not the server
		stranger.WriteToUDP(bindingResponse(tx, forged), from)
		// 3) garbage from the server
		server.WriteToUDP([]byte("garbage"), from)
		// 4) the genuine answer
		server.WriteToUDP(bindingResponse(tx, xorMappedAttr(40022, [4]byte{203, 0, 113, 45})), from)
	}()

	got, err := Client{}.SendBindingRequest(client, server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	if !got.IP.Equal(net.IPv4(203, 0, 113, 45)) || got.Port != 40022 {
		t.Fatalf("client adopted %v (a forged reply?)", got)
	}
}

func TestSendBindingRequestTimesOutWithoutValidReply(t *testing.T) {
	server, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer server.Close()
	client, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	defer client.Close()
	go func() { // only ever answers with a wrong transaction id
		buf := make([]byte, 64)
		if _, from, err := server.ReadFromUDP(buf); err == nil {
			server.WriteToUDP(bindingResponse([12]byte{0xAB}, xorMappedAttr(1, [4]byte{1, 1, 1, 1})), from)
		}
	}()
	start := time.Now()
	if got, err := (Client{}).SendBindingRequest(client, server.LocalAddr().String()); err == nil {
		t.Fatalf("accepted %v", got)
	}
	if time.Since(start) > stunReadTimeout+2*time.Second {
		t.Fatalf("did not honour the deadline: %v", time.Since(start))
	}
}
