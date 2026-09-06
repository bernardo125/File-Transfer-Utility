package cli

import (
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
	localIP := discovery.GetLocalIP()
	conn := transport.InitializeUDPSocket(localIP)
	defer conn.Close()

	info.PrintContactingSTUN()
	myToken, myPrivKey := discovery.CreateKey(conn)

	info.PrintMyToken(myToken)

	peerToken, _ := input.AskPeerToken(reader)

	// A função GetPeer devolve peerLocal além do peerRemote e a chave pública
	peerLocal, peerRemote, peerPubKey := discovery.GetPeer(peerToken)

	info.PrintExecutingHolePunch()

	// Executa o Hole Punch usando o módulo connectivity
	punchCh := connectivity.UDPHolePunch(conn, peerLocal, peerRemote, &myPrivKey, peerPubKey)
	punchResult := <-punchCh // Aguarda a conclusão via channel

	if punchResult.Err != nil {
		clierr.PrintHolePunchFailure(punchResult.Err)
		os.Exit(1)
	}

	info.PrintHolePunchSuccess(punchResult.Addr.String())

	info.PrintConnectingQUIC()
	// Conecta ao endereço validado pelo Hole Punch (punchResult.Addr)
	quicConn, err := transport.QuicUpgrade(conn, punchResult.Addr, &myPrivKey, peerPubKey)
	if err != nil {
		clierr.PrintConnectionError(err)
		os.Exit(1)
	}
	defer quicConn.CloseWithError(0, "Conexão terminada")

	info.PrintConnectionSuccess()

	if option == "1" {
		err = protocol.SenderLoop(quicConn, reader)
	} else if option == "2" {
		err = protocol.ReceiverLoop(quicConn, reader)
	} else {
		clierr.PrintInvalidOption()
	}

	if err != nil {
		clierr.PrintExecutionError(err)
	}
}
