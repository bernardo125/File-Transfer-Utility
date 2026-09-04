package protocol

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"ftu/internal/transfer"

	"github.com/quic-go/quic-go"
)

// ReceiverLoop waits for incoming QUIC streams from the peer and handles file transfer requests.
func ReceiverLoop(conn *quic.Conn, reader *bufio.Reader) error {
	fmt.Println("\n[*] À espera de pedidos de transferência do peer...")
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return fmt.Errorf("ligação com o peer terminada: %w", err)
		}

		err = HandleIncomingTransfer(stream, reader)
		if err != nil {
			fmt.Printf("[-] Erro na transferência: %v\n", err)
		}

		fmt.Print("\nDeseja continuar na sessão a aguardar mais ficheiros? (S/N): ")
		ans, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		ans = strings.TrimSpace(strings.ToUpper(ans))
		if ans != "S" && ans != "Y" {
			break
		}
		fmt.Println("\n[*] À espera do próximo ficheiro...")
	}
	return nil
}

// HandleIncomingTransfer processes an incoming transfer request according to the protocol state machine:
// RECEIVE_REQUEST -> ACCEPT/REJECT -> RECEIVE_BODY -> VERIFY_CHECKSUM -> COMPLETE/FAILED
func HandleIncomingTransfer(stream *quic.Stream, reader *bufio.Reader) error {
	defer stream.Close()

	// 1. Read metadata request frame
	reqBytes, err := ReadFrame(stream)
	if err != nil {
		return fmt.Errorf("erro ao ler pedido de metadados: %w", err)
	}

	req, err := ReceiveMetaDataFile(reqBytes)
	if err != nil {
		resp, _ := BuildMetadataResponse(StatusBadRequest)
		_ = WriteFrame(stream, resp)
		return fmt.Errorf("metadados inválidos: %w", err)
	}

	if req.Method != SEND {
		resp, _ := BuildMetadataResponse(StatusBadRequest)
		_ = WriteFrame(stream, resp)
		return fmt.Errorf("método de protocolo não suportado: %s", req.Method)
	}

	// 2. Prompt user to accept or reject transfer
	fmt.Printf("\n[?] Peer quer enviar o ficheiro '%s' (%.2f MB). Aceitar? (S/N): ",
		req.FileName, float64(req.FileSize)/(1024*1024))

	ans, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	ans = strings.TrimSpace(strings.ToUpper(ans))
	accepted := (ans == "S" || ans == "Y")

	if !accepted {
		log.Println("[Receiver] Transferência recusada pelo utilizador.")
		rejectResp, _ := BuildMetadataResponse(StatusForbidden)
		_ = WriteFrame(stream, rejectResp)
		return nil
	}

	// 3. Send acceptance response
	acceptResp, _ := BuildMetadataResponse(StatusOK)
	if err := WriteFrame(stream, acceptResp); err != nil {
		return fmt.Errorf("falha ao responder aceitação ao sender: %w", err)
	}

	log.Printf("[Receiver] A receber '%s'...", req.FileName)

	// 4. Sanitize destination file name (save in current directory)
	destFileName := filepath.Base(req.FileName)
	if destFileName == "." || destFileName == "/" || destFileName == "" {
		destFileName = "received_file"
	}

	// 5. Receive file body via transfer module and verify checksum on the fly
	_, err = transfer.ReceiveFile(destFileName, stream, req.FileSize, req.FileHash)
	if err != nil {
		log.Printf("[Receiver] Erro na recepção ou verificação de integridade: %v", err)
		failResp, _ := BuildMetadataResponse(StatusConflict)
		_ = WriteFrame(stream, failResp)
		return fmt.Errorf("falha ao gravar ou validar ficheiro: %w", err)
	}

	// 6. Notify sender that transfer and checksum verification completed successfully
	okResp, _ := BuildMetadataResponse(StatusOK)
	if err := WriteFrame(stream, okResp); err != nil {
		return fmt.Errorf("falha ao enviar confirmação de sucesso: %w", err)
	}

	log.Printf("[Receiver] Ficheiro '%s' transferido e validado com sucesso!", destFileName)
	fmt.Printf("\n[+] Ficheiro '%s' descarregado e verificado com sucesso!\n", destFileName)
	return nil
}
