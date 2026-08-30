package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
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
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  priv,
		}},
		NextProtos:         []string{"quic-echo-example"},
		InsecureSkipVerify: true, // hostname check disabled; chain verified below
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("peer sent no certificate")
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("parse peer cert: %w", err)
			}
			// Verify received public key with token's public key
			receivedPublicKey, ok := cert.PublicKey.(ed25519.PublicKey)
			if !ok {
				return fmt.Errorf("invalid public key type: expected ed25519, got %T", cert.PublicKey)
			}
			if !bytes.Equal(receivedPublicKey, *otherPeerPublicKey) {
				return fmt.Errorf("denied access: not the same key")
			}
			return nil
		},
	}
}

// udp upgrade to quic comunication to send data
func QuicUpgrade(mySocket *net.UDPConn, destination *net.UDPAddr, privateKey *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {

	// decide who will be the listenner or dialer on handshake
	// if my public key is higher than other peer i will be listener, else dialer
	myPublicKey := privateKey.Public().(ed25519.PublicKey)
	if bytes.Compare(myPublicKey, *otherPeerPublicKey) > 0 {
		return quicDoListen(mySocket, privateKey, otherPeerPublicKey)
	} else {
		return quicDoDial(mySocket, destination, privateKey, otherPeerPublicKey)
	}

}

func quicDoDial(mySocket *net.UDPConn, destination *net.UDPAddr, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Configure QUIC with extended idle timeout and keepalive
	quicConfig := &quic.Config{
		MaxIdleTimeout:  5 * time.Minute,  // Extended idle timeout
		KeepAlivePeriod: 30 * time.Second, // Send keepalive pings
	}

	// start tls self certificate to secure data transfer
	quicConn, err := quic.Dial(ctx, mySocket, destination, generateTLSConfig(priv, otherPeerPublicKey), quicConfig)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", destination, err)
		return nil, fmt.Errorf("failed to connect to any candidate")
	}

	log.Printf("Successfully connected to %s", destination)
	return quicConn, nil

}

func quicDoListen(mySocket *net.UDPConn, priv *ed25519.PrivateKey, otherPeerPublicKey *ed25519.PublicKey) (*quic.Conn, error) {
	listener, err := quic.Listen(mySocket, generateTLSConfig(priv, otherPeerPublicKey), nil)
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	conn, err := listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}
	log.Printf("Successfully connected to %s", conn.RemoteAddr())
	return conn, nil
}
