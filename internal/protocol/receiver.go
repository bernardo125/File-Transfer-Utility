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
		// the sender opens a new control stream for every file, and HandleIncomingTransfer
		// closes the one it was given, so each iteration must accept its own stream
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

	if err := transfer.EnsureFreeSpace(".", req.FileSize); err != nil {
		SendCodeResponse(StatusNotAcceptable, controlStream)
		return fmt.Errorf("sem espaço em disco: %w", err)
	}

	// send acceptance response
	err = SendCodeResponse(StatusContinue, controlStream)
	if err != nil {
		return err
	}

	log.Printf("[Receiver] A receber %q...", req.FileName)

	// receive file body via transfer module (temp file, size + SHA-256 verified, never overwrites)
	finalPath, _, err := transfer.ReceiveFile(".", req.FileName, conn, req.FileSize, req.FileHash, nil)
	if err != nil {
		log.Printf("[Receiver] Erro na recepção ou verificação de integridade: %v", err)
		SendCodeResponse(StatusForReceiveError(err), controlStream)
		return fmt.Errorf("falha ao gravar ou validar ficheiro: %w", err)
	}
	destFileName := filepath.Base(finalPath)

	// notify sender that transfer and checksum verification completed successfully
	err = SendCodeResponse(StatusOK, controlStream)
	if err != nil {
		return err
	}

	log.Printf("[Receiver] Ficheiro %q transferido e validado com sucesso!", destFileName)
	info.PrintTransferSuccess(destFileName)
	return nil
}
