package transfer

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

const FILE_META_PAYLOAD_MIN_LENGTH = 10
const TRANSPORT_ERROR = 0x3

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

// Function to send file to other peer (initialization of stream on quic module)
func sendFile(fileName string, conn *quic.Conn, commStream *quic.Stream) error {
	// file selection
	file, fileStats, err := selectFileToSend(fileName, conn)
	if err != nil {
		return err
	}
	// metadata of file
	metaPayload := buildMetaDataPayload(fileStats)

	defer commStream.Close()

	// send file metadata
	_, writeMetaErr := commStream.Write(metaPayload)
	if writeMetaErr != nil {
		conn.CloseWithError(TRANSPORT_ERROR, writeMetaErr.Error())
		return writeMetaErr
	}

	// read if the reception of file has been accepted by other peer
	recvBuff := make([]byte, 10)
	_, err = commStream.Read(recvBuff)
	if err != nil {
		return err
	}
	log.Println("Sending data to other peer")
	start := time.Now()
	_, err = io.Copy(commStream, file)
	if err != nil {
		return err
	}
	fmt.Printf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	return nil

}
