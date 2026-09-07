package protocol

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"

	"github.com/quic-go/quic-go"
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
	var request Request
	resqPayload, err := receiveControlMessage(controlStream)
	err = json.Unmarshal(resqPayload, &request)
	if err != nil {
		return nil, err
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
func sendControlMessage(metaPayload []byte, controlStream *quic.Stream) error {
	// send payload length
	length := uint32(len(metaPayload))
	err := binary.Write(controlStream, binary.BigEndian, length)
	if err != nil {
		return err
	}
	_, err = controlStream.Write(metaPayload)
	return err
}

func receiveControlMessage(controlStream *quic.Stream) (response []byte, err error) {
	// read payload length
	var length uint32
	err = binary.Read(controlStream, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	recvBuff := make([]byte, length)
	_, err = io.ReadFull(controlStream, recvBuff) // ReadFull garantees full read of length bytes (parcial reading problem)
	if err != nil {
		return nil, err
	}
	return recvBuff, nil
}
