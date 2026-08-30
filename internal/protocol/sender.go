package protocol

import (
	transfer "cmd/ftu/main.go/internal/transfer"
	"os"

	"github.com/quic-go/quic-go"
)

func selectFileToSend(fileName string, conn *quic.Conn) (*os.File, os.FileInfo, error) {
	file, openErr := os.Open(fileName)
	if openErr != nil {
		conn.CloseWithError(quic.ApplicationErrorCode(101), openErr.Error())
		return nil, nil, openErr
	}

	fileStat, statErr := file.Stat()
	if statErr != nil {
		file.Close()
		conn.CloseWithError(quic.ApplicationErrorCode(101), openErr.Error())
		return nil, nil, statErr
	}

	file.Close()

	return file, fileStat, nil

}

func sendFileToPeer(fileName string, conn *quic.Conn, commStream *quic.Stream) error {
	// file selection
	file, fileStats, err := selectFileToSend(fileName, conn)
	if err != nil {
		return err
	}
	// metadata of file
	metaPayload := buildMetaDataPayload(fileStats)

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
