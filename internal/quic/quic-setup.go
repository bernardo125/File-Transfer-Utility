package quic

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/quic/quic"
)

// Based on https://github.com/quic-go/quic-go/blob/master/example/echo/echo.go
func generateTLSConfig() *tls.Config {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
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
		NextProtos: []string{"quic-echo-example"},
	}
}

// udp upgrade to quic comunication to send data
func QuicUpgrade(mySocket *net.UDPConn, destination *net.UDPAddr) {
	// start tls self certificate to secure data transfer
	peerConfig := generateTLSConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Configure QUIC with extended idle timeout and keepalive
	quicConfig := &quic.Config{
		MaxIdleTimeout:  5 * time.Minute,  // Extended idle timeout
		KeepAlivePeriod: 30 * time.Second, // Send keepalive pings
	}

	quicConn, err := quic.Dial(ctx, mySocket, destination, peerConfig, quicConfig)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", destination, err)
	}

	log.Printf("Successfully connected to %s", destination)
	return quicConn, nil

}
