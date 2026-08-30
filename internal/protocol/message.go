package protocol

import (
	"encoding/binary"
	"os"
)

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

/*
func receiveMetaDataPayload(payloadBytes []byte) (fileStats os.FileInfo) {
	fileSize := int64(payloadBytes[0:8])
	filenameLen := int16(payloadBytes[8:10])
	filename := string(payloadBytes[10:filenameLen])

}
*/
