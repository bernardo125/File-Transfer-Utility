package protocol

import (
	transfer "cmd/ftu/main.go/internal/transfer"

	"github.com/quic-go/quic-go"
)

func sendFileToPeer(fileAddr string, conn *quic.Conn, commStream *quic.Stream) error {
	// file selection
	file, fileStats, hashFile, err := transfer.SelectFileToSend(fileAddr, conn)
	if err != nil {
		return err
	}
	// metadata of file
	metaPayload, err := buildMetaDataSendFile(fileStats, hashFile)
	if err != nil {
		return err
	}

	//send metadata
	err = transfer.SendMetadata(metaPayload, commStream)
	if err != nil {
		return err
	}

	//send data
	err = transfer.SendFile(file, conn, commStream)
	if err != nil {
		return err
	}

	return nil

}
