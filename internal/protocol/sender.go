package protocol

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strings"

	"ftu/internal/transfer"

	"github.com/quic-go/quic-go"
)

// SenderLoop coordinates the sender workflow, prompting for files and orchestrating peer communication.
func SenderLoop(conn *quic.Conn, reader *bufio.Reader) error {
	for {
		fmt.Print("\nIntroduza o caminho do ficheiro a enviar (ou 'sair' para terminar): ")
		filePath, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		if strings.EqualFold(filePath, "sair") || strings.EqualFold(filePath, "exit") {
			break
		}

		err = SendFileToPeer(conn, filePath)
		if err != nil {
			fmt.Printf("[-] Falha no envio: %v\n", err)
		}

		fmt.Print("\nDeseja enviar outro ficheiro? (S/N): ")
		ans, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		ans = strings.TrimSpace(strings.ToUpper(ans))
		if ans != "S" && ans != "Y" {
			break
		}
	}
	return nil
}

// SendFileToPeer executes the peer communication protocol for sending a single file.
// Architecture:
// IDLE -> REQUEST -> REQUESTED -> RESPONSE received:
// ACCEPTED -> TRANSFERRING -> Stream FIN -> VERIFYING -> COMPLETED / REJECTED
func SendFileToPeer(conn *quic.Conn, filePath string) error {
	// 1. Prepare file and compute hash using transfer package
	file, stats, hash, err := transfer.SelectFileToSend(filePath)
	if err != nil {
		return fmt.Errorf("erro ao aceder ao ficheiro: %w", err)
	}
	defer file.Close()

	log.Printf("[Sender] Ficheiro selecionado: %s (%.2f MB, SHA256: %s)",
		stats.Name(), float64(stats.Size())/(1024*1024), hash)

	// 2. Open QUIC stream for transfer
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("erro ao abrir stream QUIC: %w", err)
	}
	defer stream.Close()

	// 3. Send metadata request to receiver
	metaPayload, err := BuildMetaDataSendFile(stats, hash)
	if err != nil {
		return fmt.Errorf("erro ao construir metadados: %w", err)
	}

	log.Println("[Sender] A enviar pedido de transferência ao destinatário...")
	if err := WriteFrame(stream, metaPayload); err != nil {
		return fmt.Errorf("erro ao enviar pedido: %w", err)
	}

	// 4. Read receiver response
	respBytes, err := ReadFrame(stream)
	if err != nil {
		return fmt.Errorf("erro ao aguardar resposta do destinatário: %w", err)
	}

	statusCode := ReceiveMetadataResponse(respBytes)
	if statusCode != StatusOK && statusCode != StatusContinue {
		return fmt.Errorf("transferência recusada pelo destinatário (Status: %d)", statusCode)
	}

	log.Println("[Sender] Destinatário aceitou a transferência! A iniciar envio do ficheiro...")

	// 5. Transfer file payload via transfer module
	_, err = transfer.SendFile(file, stream)
	if err != nil {
		return fmt.Errorf("falha durante o envio do ficheiro: %w", err)
	}

	// 6. Signal FIN on write stream (closes send direction so receiver knows file is complete)
	if err := stream.Close(); err != nil {
		return fmt.Errorf("erro ao sinalizar fim de stream: %w", err)
	}

	log.Println("[Sender] Payload enviado. A aguardar verificação de integridade do destinatário...")

	// 7. Await receiver's final checksum verification response
	finalRespBytes, err := ReadFrame(stream)
	if err != nil {
		return fmt.Errorf("erro ao aguardar verificação final: %w", err)
	}

	finalStatus := ReceiveMetadataResponse(finalRespBytes)
	if finalStatus != StatusOK {
		return fmt.Errorf("destinatário reportou falha na verificação de integridade (Status: %d)", finalStatus)
	}

	log.Println("[Sender] Destinatário validou com sucesso a integridade do ficheiro!")
	fmt.Println("\n[+] Transferência e verificação de integridade concluídas com sucesso!")
	return nil
}
