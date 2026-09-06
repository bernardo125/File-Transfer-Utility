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

		err = HandleIncomingTransfer(stream, conn, reader)
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
func HandleIncomingTransfer(controlStream *quic.Stream, conn *quic.Conn, reader *bufio.Reader) error {
	defer controlStream.Close()

	// read metadata request frame
	req, err := ReceiveMetaDataRequest(controlStream)
	if err != nil {
		SendCodeResponse(StatusBadRequest, controlStream)
		return fmt.Errorf("metadados inválidos: %w", err)
	}

	// prompt user to accept or reject transfer
	accepted, err := input.AskAcceptTransfer(reader, req.FileName, req.FileSize)
	if err != nil {
		return err
	}

	if !accepted {
		log.Println("[Receiver] Transferência recusada pelo utilizador.")
		SendCodeResponse(StatusForbidden, controlStream)
		return nil
	}

	// send acceptance response
	err = SendCodeResponse(StatusContinue, controlStream)
	if err != nil {
		return err
	}

	log.Printf("[Receiver] A receber '%s'...", req.FileName)

	// sanitize destination file name (save in current directory)
	destFileName := filepath.Base(req.FileName)
	if destFileName == "." || destFileName == "/" || destFileName == "" {
		destFileName = "received_file"
	}

	// receive file body via transfer module and verify checksum on the fly
	_, err = transfer.ReceiveFile(destFileName, conn, req.FileSize, req.FileHash)
	if err != nil {
		log.Printf("[Receiver] Erro na recepção ou verificação de integridade: %v", err)
		SendCodeResponse(StatusConflict, controlStream)
		return fmt.Errorf("falha ao gravar ou validar ficheiro: %w", err)
	}

	// notify sender that transfer and checksum verification completed successfully
	err = SendCodeResponse(StatusOK, controlStream)
	if err != nil {
		return err
	}

	log.Printf("[Receiver] Ficheiro '%s' transferido e validado com sucesso!", destFileName)
	info.PrintTransferSuccess(destFileName)
	return nil
}
