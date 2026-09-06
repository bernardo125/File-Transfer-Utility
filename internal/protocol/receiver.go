package protocol

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"path/filepath"

	"ftu/internal/transfer"
	clierr "ftu/pkg/cli/errors"
	"ftu/pkg/cli/info"
	"ftu/pkg/cli/input"

	"github.com/quic-go/quic-go"
)

// ReceiverLoop waits for incoming QUIC streams from the peer and handles file transfer requests.
func ReceiverLoop(conn *quic.Conn, reader *bufio.Reader) error {
	info.PrintWaitingForTransfer()
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			return fmt.Errorf("ligação com o peer terminada: %w", err)
		}

		err = HandleIncomingTransfer(stream, reader)
		if err != nil {
			clierr.PrintTransferError(err)
		}

		continueReceiving, err := input.AskContinueReceiving(reader)
		if err != nil {
			return err
		}
		if !continueReceiving {
			break
		}
		info.PrintWaitingForNextFile()
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
	accepted, err := input.AskAcceptTransfer(reader, req.FileName, req.FileSize)
	if err != nil {
		return err
	}

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
	info.PrintTransferSuccess(destFileName)
	return nil
}
