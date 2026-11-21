package cli

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"

	"ftu/internal/connectivity"
	"ftu/internal/discovery"
	"ftu/internal/protocol"
	"ftu/internal/transport"
	clierr "ftu/pkg/cli/errors"
	"ftu/pkg/cli/info"
	"ftu/pkg/cli/input"
)

// Run coordinates and executes the File Transfer Utility CLI workflow.
func Run() {
	reader := input.NewReader()

	option, _ := input.AskMenuOption(reader)

	if option == "3" {
		os.Exit(0)
	}
	conn := transport.InitializeUDPSocket()
	defer conn.Close()

	myToken, myPrivKey := discovery.CreateKey(conn)

	info.PrintMyToken(myToken)

	peerToken, err := input.AskPeerToken(reader)

	// A função GetPeer devolve peerLocal além do peerRemote e a chave pública
	peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, peerPubKey, err := discovery.GetPeer(peerToken)

	if err != nil {
		clierr.PrintConnectionError(err)
		os.Exit(1)
	}

	info.PrintExecutingHolePunch()

	// Executa o Hole Punch usando o módulo connectivity
	punchCh := connectivity.UDPHolePunch(conn, peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, &myPrivKey, peerPubKey)
	punchResult := <-punchCh // Aguarda a conclusão via channel

	if punchResult.Err != nil {
		clierr.PrintHolePunchFailure(punchResult.Err)
		os.Exit(1)
	}

	info.PrintHolePunchSuccess()

	info.PrintConnectingQUIC()
	// Conecta ao endereço validado pelo Hole Punch (punchResult.Addr)
	quicConn, err := transport.QuicUpgradeCandidates(context.Background(), conn, punchResult.Candidates, &myPrivKey, peerPubKey)
	if err != nil {
		clierr.PrintConnectionError(err)
		os.Exit(1)
	}
	defer quicConn.CloseWithError(0, "Conexão terminada")

	// The session is encrypted and key-pinned, but the pinned key came from a token that travelled
	// over a channel nobody authenticated: the users must confirm they see the same fingerprint.
	fingerprint := discovery.PairingFingerprint(myPrivKey.Public().(ed25519.PublicKey), *peerPubKey)
	confirmed, err := input.AskConfirmFingerprint(reader, fingerprint)
	if err != nil || !confirmed {
		quicConn.CloseWithError(0, "pairing refused")
		fmt.Println("Pairing not confirmed: connection closed. Do not share files with this peer; create new tokens and try again.")
		os.Exit(1)
	}

	info.PrintConnectionSuccess()

	switch option {
	case "1":
		err = protocol.SenderLoop(quicConn, reader)
	case "2":
		err = protocol.ReceiverLoop(quicConn, reader)
	default:
		clierr.PrintInvalidOption()
	}

	if err != nil {
		clierr.PrintExecutionError(err)
	}
}
