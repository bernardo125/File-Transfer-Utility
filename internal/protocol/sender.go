package protocol

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strings"

	"ftu/internal/transfer"
	clierr "ftu/pkg/cli/errors"
	"ftu/pkg/cli/info"
	"ftu/pkg/cli/input"

	"github.com/quic-go/quic-go"
)

// SenderLoop coordinates the sender workflow, prompting for files and orchestrating peer communication.
func SenderLoop(conn *quic.Conn, reader *bufio.Reader) error {
	for {
		filePath, err := input.AskFilePath(reader)
		if err != nil {
			return err
		}
		if filePath == "" {
			continue
		}
		if strings.EqualFold(filePath, "exit") {
			break
		}

		err = SendFileToPeer(conn, filePath)
		if err != nil {
			clierr.PrintSendFailure(err)
		}

		sendAnother, err := input.AskSendAnother(reader)
		if err != nil {
			return err
		}
		if !sendAnother {
			break
		}
	}
	return nil
}

// sendFileToPeer executes the peer communication protocol for sending a single file.
// Arch:
// IDLE -> REQUEST -> REQUESTED -> RESPONSE received:
// ACCEPTED -> TRANSFERRING -> Stream FIN -> VERIFYING -> COMPLETED / REJECTED
func SendFileToPeer(conn *quic.Conn, filePath string) error {
	// prepare file and compute hash using transfer package
	file, stats, hash, err := transfer.SelectFileToSend(filePath)
	if err != nil {
		return fmt.Errorf("erro ao aceder ao ficheiro: %w", err)
	}
	defer file.Close()

	log.Printf("[Sender] Ficheiro selecionado: %s (%.2f MB, SHA256: %s)",
		stats.Name(), float64(stats.Size())/(1024*1024), hash)

	// open QUIC stream for metadata
	controlStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("erro ao abrir stream QUIC: %w", err)
	}
	defer controlStream.Close()

	// send metadata request to receiver
	err = SendMetaDataRequest(stats, hash, controlStream)
	if err != nil {
		return err
	}

	// read receiver response code
	response, err := ReceiveMetadataResponse(controlStream)
	if response.Status != StatusOK && response.Status != StatusContinue {
		return fmt.Errorf("transferência recusada pelo destinatário (Status: %d)", response.Status)
	}

	log.Println("[Sender] Destinatário aceitou a transferência! A iniciar envio do ficheiro...")

	// transfer file payload via transfer module
	err = transfer.SendFile(file, conn)
	if err != nil {
		return fmt.Errorf("falha durante o envio do ficheiro: %w", err)
	}

	log.Println("[Sender] A aguardar verificação de integridade do destinatário...")

	// await receiver's final checksum verification response
	finalStatus, err := ReceiveMetadataResponse(controlStream)

	if finalStatus.Status != StatusOK {
		return fmt.Errorf("destinatário reportou falha na verificação de integridade (Status: %d)", finalStatus)
	}

	log.Println("[Sender] Destinatário validou com sucesso a integridade do ficheiro!")
	info.PrintSendSuccess()
	return nil
}
