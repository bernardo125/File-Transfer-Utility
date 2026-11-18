package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/quic-go/quic-go"
)

// MaxControlMessageSize bounds a single control frame. Control frames carry tiny JSON
// documents (file name, size, hash, status); without a bound a peer could announce a
// 4 GiB length prefix and force the receiver to allocate it.
const MaxControlMessageSize = 64 * 1024

var (
	// ErrFrameTooLarge is returned when a control frame announces more than MaxControlMessageSize bytes.
	ErrFrameTooLarge = errors.New("control frame exceeds maximum size")
	// ErrInvalidRequest is returned when a metadata request fails validation.
	ErrInvalidRequest = errors.New("invalid metadata request")
)

// HTTP-like protocol with JSON serialization in the QUIC control stream
// SENDER
func SendMetaDataRequest(fileStats os.FileInfo, hash string, controlStream *quic.Stream) error {
	request, err := BuildMetaDataRequest(fileStats, hash)
	if err != nil {
		return err
	}
	return sendControlMessage(request, controlStream)
}

// BuildMetaDataRequest builds the metadata payload for a SEND request
func BuildMetaDataRequest(fileStats os.FileInfo, hash string) ([]byte, error) {
	fileName := fileStats.Name()
	fileSize := fileStats.Size()
	request := Request{FileName: fileName, FileSize: fileSize,
		FileHash: hash,
	}
	return json.Marshal(request)
}

// ReceiveMetadataResponse parses response payload and returns Response struct with error handling
func ReceiveMetadataResponse(controlStream *quic.Stream) (*Response, error) {
	var response Response
	respPayload, err := receiveControlMessage(controlStream)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respPayload, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// RECEIVER
// ReceiveMetaDataRequest parses a Request from payload bytes
func ReceiveMetaDataRequest(controlStream *quic.Stream) (*Request, error) {
	resqPayload, err := receiveControlMessage(controlStream)
	if err != nil {
		return nil, err
	}
	return DecodeRequest(resqPayload)
}

// DecodeRequest parses and validates a metadata request payload. A request that decodes
// but carries a negative size or a malformed hash is rejected, because the receiver uses
// those fields to decide whether size and integrity checks are enforced.
func DecodeRequest(payload []byte) (*Request, error) {
	var request Request
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if request.FileSize < 0 {
		return nil, fmt.Errorf("%w: negative file size %d", ErrInvalidRequest, request.FileSize)
	}
	if len(request.FileHash) != 64 || strings.Trim(request.FileHash, "0123456789abcdef") != "" {
		return nil, fmt.Errorf("%w: file hash must be 64 lowercase hex characters", ErrInvalidRequest)
	}
	return &request, nil
}

func SendCodeResponse(code StatusCode, controlStream *quic.Stream) error {
	responsePayload, err := BuildCodeResponse(code)
	if err != nil {
		return err
	}
	return sendControlMessage(responsePayload, controlStream)
}

// BuildMetadataResponse builds response payload with status code
func BuildCodeResponse(code StatusCode) ([]byte, error) {
	response := Response{Status: code}
	return json.Marshal(response)
}

// SHARED - SENDER AND RECEIVER
func sendControlMessage(metaPayload []byte, controlStream io.Writer) error {
	if len(metaPayload) > MaxControlMessageSize {
		return ErrFrameTooLarge
	}
	// send payload length
	length := uint32(len(metaPayload))
	err := binary.Write(controlStream, binary.BigEndian, length)
	if err != nil {
		return err
	}
	_, err = controlStream.Write(metaPayload)
	return err
}

func receiveControlMessage(controlStream io.Reader) (response []byte, err error) {
	// read payload length
	var length uint32
	err = binary.Read(controlStream, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	if length > MaxControlMessageSize {
		return nil, fmt.Errorf("%w: announced %d bytes", ErrFrameTooLarge, length)
	}
	recvBuff := make([]byte, length)
	_, err = io.ReadFull(controlStream, recvBuff) // ReadFull garantees full read of length bytes (parcial reading problem)
	if err != nil {
		return nil, err
	}
	return recvBuff, nil
}
