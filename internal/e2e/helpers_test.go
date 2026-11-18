package e2e

import (
	"crypto/ed25519"
	"net"

	"ftu/internal/testutil"
	"ftu/internal/transport"

	"github.com/quic-go/quic-go"
)

func quicUpgrade(me *testutil.Peer, dest *net.UDPAddr, peer *ed25519.PublicKey) (*quic.Conn, error) {
	return transport.QuicUpgrade(me.Conn, dest, &me.Priv, peer)
}
