package transfer

import (
	"encoding/binary"
	"os"

	"github.com/quic-go/quic-go"
)

const FILE_META_PAYLOAD_MIN_LENGTH = 10

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

// payload size (8 size + 2 byte filename_len +  n bytes filename = 8 + 2 + n bytes)
func buildMetaDataPayload(fileStats os.FileInfo) []byte {

	nameBytes := []byte(fileStats.Name())
	fileNameLength := len(nameBytes)
	payload := make([]byte, FILE_META_PAYLOAD_MIN_LENGTH+fileNameLength)
	// file size
	binary.BigEndian.PutUint64(payload[0:8], uint64(fileStats.Size()))
	// filename Length to avoid network buffer errors
	binary.BigEndian.PutUint16(payload[8:10], uint16(fileNameLength))

	// filename
	copy(payload[10:], nameBytes)

	return payload
}

func sendFile(fileName string, conn *quic.Conn, commStream quic.Stream) error /*{
	file, fileStats, err := selectFileToSend(fileName, conn)
	if err != nil {
		return err
	}

}*/
