package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// Based on https://github.com/quic-go/quic-go/blob/master/example/echo/echo.go
func generateTLSConfig(priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) *tls.Config {
	// The certificate only carries the identity key; peers pin the key, not the certificate, so
	// the validity window is informational. It is kept short and bounded anyway.
	now := time.Now()
	template := x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(7 * 24 * time.Hour)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  priv,
		}},
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{"quic-echo-example"}, // ALPN rename is a wire change, tracked separately
		InsecureSkipVerify: true,                          // hostname check disabled; chain verified below
		// Without this the listening side never asks the dialer for a certificate, so
		// VerifyPeerCertificate would only ever run on the dialer and any host able to
		// reach the UDP port could open a session. Mutual authentication is required.
		ClientAuth: tls.RequireAnyClientCert,
		// VerifyConnection (unlike VerifyPeerCertificate) also runs when a session is resumed, so
		// the key pin can never be skipped by a resumption path.
		VerifyConnection: func(cs tls.ConnectionState) error { return verifyPinnedPeer(cs, *otherPeerPublicKey) },
	}
}

// verifyPinnedPeer is the authentication decision of the product: the peer must be speaking
// TLS 1.3 and its certificate must carry exactly the Ed25519 key that came in the token. TLS 1.3's
// CertificateVerify has already proven that the peer holds the matching private key.
func verifyPinnedPeer(cs tls.ConnectionState, pinned ed25519.PublicKey) error {
	if cs.Version != tls.VersionTLS13 {
		return fmt.Errorf("denied access: TLS version 0x%04x is not TLS 1.3", cs.Version)
	}
	if len(cs.PeerCertificates) == 0 {
		return errors.New("peer sent no certificate")
	}
	got, ok := cs.PeerCertificates[0].PublicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key type: expected ed25519, got %T", cs.PeerCertificates[0].PublicKey)
	}
	if subtle.ConstantTimeCompare(got, pinned) != 1 {
		return errors.New("denied access: not the same key")
	}
	return nil
}

// handshakeTimeout bounds how long either side waits for the QUIC handshake.
const handshakeTimeout = 15 * time.Second

// udp upgrade to quic comunication to send data
func QuicUpgrade(mySocket *net.UDPConn, destination *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {
	return QuicUpgradeContext(context.Background(), mySocket, destination, privateKey, otherPeerPublicKey)
}

// dialAttemptTimeout bounds one dial when several candidate addresses are tried in turn.
var dialAttemptTimeout = 6 * time.Second

// QuicUpgradeCandidates is QuicUpgradeContext for several destinations. The listener role ignores
// them (it waits for the peer); the dialer tries each in order, so a spoofed first candidate only
// costs one attempt timeout. Authentication is unchanged: every attempt is a full mTLS handshake
// with the pinned key, so a wrong address can never yield a session with the wrong peer.
func QuicUpgradeCandidates(ctx context.Context, mySocket *net.UDPConn, destinations []*net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {
	myPublicKey := privateKey.Public().(ed25519.PublicKey)
	if bytes.Compare(myPublicKey, *otherPeerPublicKey) > 0 || len(destinations) <= 1 {
		var dest *net.UDPAddr
		if len(destinations) > 0 {
			dest = destinations[0]
		}
		return QuicUpgradeContext(ctx, mySocket, dest, privateKey, otherPeerPublicKey)
	}
	var lastErr error = errors.New("no candidate address")
	for _, dest := range destinations {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		conn, err := quicDial(ctx, mySocket, dest, privateKey, otherPeerPublicKey, dialAttemptTimeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// QuicUpgradeContext is QuicUpgrade that gives up as soon as ctx is cancelled.
func QuicUpgradeContext(ctx context.Context, mySocket *net.UDPConn, destination *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {

	// decide who will be the listenner or dialer on handshake
	// if my public key is higher than other peer i will be listener, else dialer
	myPublicKey := privateKey.Public().(ed25519.PublicKey)
	if bytes.Compare(myPublicKey, *otherPeerPublicKey) > 0 {
		return quicListen(ctx, mySocket, privateKey, otherPeerPublicKey, handshakeTimeout)
	} else {
		return quicDial(ctx, mySocket, destination, privateKey, otherPeerPublicKey, handshakeTimeout)
	}

}

func quicDoDial(mySocket *net.UDPConn, destination *net.UDPAddr, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey, timeout time.Duration) (*quic.Conn, error) {
	return quicDial(context.Background(), mySocket, destination, priv, otherPeerPublicKey, timeout)
}

func quicDial(parent context.Context, mySocket *net.UDPConn, destination *net.UDPAddr, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey, timeout time.Duration) (*quic.Conn, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	// start tls self certificate to secure data transfer
	quicConn, err := quic.Dial(ctx, mySocket, destination, generateTLSConfig(priv, otherPeerPublicKey), defaultQuicConfig())
	if err != nil {
		log.Printf("Failed to connect")
		return nil, fmt.Errorf("failed to connect to any candidate")
	}

	log.Printf("Successfully connected")
	return quicConn, nil

}

func quicDoListen(mySocket *net.UDPConn, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey, timeout time.Duration) (*quic.Conn, error) {
	return quicListen(context.Background(), mySocket, priv, otherPeerPublicKey, timeout)
}

func quicListen(parent context.Context, mySocket *net.UDPConn, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey, timeout time.Duration) (*quic.Conn, error) {
	listener, err := quic.Listen(mySocket, generateTLSConfig(priv, otherPeerPublicKey), defaultQuicConfig())
	if err != nil {
		return nil, err
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	conn, err := listener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("Successfully connected")
	return conn, nil
}

func defaultQuicConfig() *quic.Config {
	// Configure QUIC with extended idle timeout and keepalive
	return &quic.Config{
		MaxIdleTimeout:  30 * time.Second, // idle timeout
		KeepAlivePeriod: 15 * time.Second, // Send keepalive pings
	}
}
