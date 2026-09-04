package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// HTTP-like protocol with JSON serialization in the QUIC control stream.

// BuildMetaDataSendFile builds the metadata payload for a SEND request.
func BuildMetaDataSendFile(fileStats os.FileInfo, hash string) ([]byte, error) {
	fileName := fileStats.Name()
	fileSize := fileStats.Size()
	request := Request{Method: SEND, FileName: fileName, FileSize: fileSize,
		FileHash: hash,
	}
	return json.Marshal(request)
}

// ReceiveMetaDataFile parses a Request from payload bytes.
func ReceiveMetaDataFile(payloadBytes []byte) (*Request, error) {
	var request Request
	err := json.Unmarshal(payloadBytes, &request)
	if err != nil {
		return nil, err
	}
	if request.Method == "" {
		return nil, errors.New("invalid request: missing method")
	}
	return &request, nil
}

// receiveMetaDataFile keeps backwards compatibility for unexported usage.
func receiveMetaDataFile(payloadBytes []byte) (*Request, error) {
	return ReceiveMetaDataFile(payloadBytes)
}

// BuildMetaDataCancelFile builds the metadata payload for a CANCEL request.
func BuildMetaDataCancelFile() ([]byte, error) {
	cancelRequest := Request{Method: CANCEL}
	return json.Marshal(cancelRequest)
}

// BuildMetadataResponse builds response payload with status code.
func BuildMetadataResponse(code StatusCode) ([]byte, error) {
	response := Response{Status: code}
	return json.Marshal(response)
}

// ReceiveMetadataResponse parses response payload and returns status code.
func ReceiveMetadataResponse(payloadBytes []byte) StatusCode {
	var response Response
	if err := json.Unmarshal(payloadBytes, &response); err != nil {
		return 0
	}
	return response.Status
}

// ParseMetadataResponse parses response payload and returns Response struct with error handling.
func ParseMetadataResponse(payloadBytes []byte) (*Response, error) {
	var response Response
	if err := json.Unmarshal(payloadBytes, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// WriteFrame writes length-prefixed data (4-byte big endian header) to a stream/writer.
func WriteFrame(w io.Writer, data []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadFrame reads length-prefixed data (4-byte big endian header) from a stream/reader.
func ReadFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if msgLen > 1024*1024 { // 1MB sanity limit for control messages
		return nil, fmt.Errorf("control frame exceeded limit: %d bytes", msgLen)
	}
	buf := make([]byte, msgLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
