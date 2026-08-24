package quic

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
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
func QuicUpgrade(mySocket *net.UDPConn, destination *net.UDPAddr, mytoken *[]byte, expectedToken *[]byte) (*quic.Conn, error) {

	// decide who will be the listenner or dialer on handshake
	if bytes.Compare(*mytoken, *expectedToken) == 1 {
		return nil, nil
	} else {
		return quicDoDial(mySocket, destination)
	}

}

func quicDoDial(mySocket *net.UDPConn, destination *net.UDPAddr) (*quic.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Configure QUIC with extended idle timeout and keepalive
	quicConfig := &quic.Config{
		MaxIdleTimeout:  5 * time.Minute,  // Extended idle timeout
		KeepAlivePeriod: 30 * time.Second, // Send keepalive pings
	}

	// start tls self certificate to secure data transfer
	quicConn, err := quic.Dial(ctx, mySocket, destination, generateTLSConfig(), quicConfig)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", destination, err)
		return nil, fmt.Errorf("failed to connect to any candidate")
	}

	log.Printf("Successfully connected to %s", destination)
	return quicConn, nil

}

/*
func quicDoListen(mySocket *net.UDPConn, destination *net.UDPAddr) (*quic.Conn, error) {
	listener, err := quic.ListenAddr(destination, generateTLSConfig(), nil)
	if err != nil {
		return err
	}
	defer listener.Close()

	conn, err := listener.Accept(context.Background())
	if err != nil {
		return err
	}

}
*/
