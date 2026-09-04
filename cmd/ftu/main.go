package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"ftu/internal/connectivity"
	"ftu/internal/discovery"
	"ftu/internal/protocol"
	"ftu/internal/transport"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== File Transfer Utility ===")
	fmt.Println("[1] Enviar Ficheiro")
	fmt.Println("[2] Receber Ficheiro")
	fmt.Println("[3] Sair")
	fmt.Print("Escolha uma opção: ")

	option, _ := reader.ReadString('\n')
	option = strings.TrimSpace(option)

	if option == "3" {
		os.Exit(0)
	}
	localIP := discovery.GetLocalIP()
	conn := transport.InitializeUDPSocket(localIP)
	defer conn.Close()

	fmt.Println("A contactar servidor STUN para gerar token...")
	myToken, myPrivKey := discovery.CreateKey(conn)

	fmt.Printf("\nO SEU TOKEN:\n%s\n\n", myToken)

	fmt.Print("Insira o token do outro peer: ")
	peerToken, _ := reader.ReadString('\n')
	peerToken = strings.TrimSpace(peerToken)

	// A função GetPeer devolve peerLocal além do peerRemote e a chave pública
	peerLocal, peerRemote, peerPubKey := discovery.GetPeer(peerToken)

	fmt.Println("A executar UDP Hole Punching...")

	// Executa o Hole Punch usando o módulo connectivity
	punchCh := connectivity.UDPHolePunch(conn, peerLocal, peerRemote, &myPrivKey, peerPubKey)
	punchResult := <-punchCh // Aguarda a conclusão via channel

	if punchResult.Err != nil {
		fmt.Printf("Falha no Hole Punch: %v\n", punchResult.Err)
		os.Exit(1)
	}

	fmt.Printf("Hole Punch bem-sucedido para: %s\n", punchResult.Addr.String())

	fmt.Println("A estabelecer ligação QUIC...")
	// Conecta ao endereço validado pelo Hole Punch (punchResult.Addr)
	quicConn, err := transport.QuicUpgrade(conn, punchResult.Addr, &myPrivKey, peerPubKey)
	if err != nil {
		fmt.Printf("Erro na ligação: %v\n", err)
		os.Exit(1)
	}
	defer quicConn.CloseWithError(0, "Conexão terminada")

	fmt.Println("Ligação estabelecida com sucesso!")

	if option == "1" {
		err = protocol.SenderLoop(quicConn, reader)
	} else if option == "2" {
		err = protocol.ReceiverLoop(quicConn, reader)
	} else {
		fmt.Println("Opção inválida.")
	}

	if err != nil {
		fmt.Printf("Erro durante a execução: %v\n", err)
	}
}
