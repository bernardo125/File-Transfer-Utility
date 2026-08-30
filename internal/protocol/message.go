package protocol

import (
	"encoding/json"
	"os"
)

// HTTP-like protocol with json in control stream

// build metadata payload by request struct
func buildMetaDataSendFile(fileStats os.FileInfo, hash string) ([]byte, error) {
	fileName := fileStats.Name()
	fileSize := fileStats.Size()
	request := Request{Method: SEND, FileName: fileName, FileSize: fileSize, FileHash: hash}

	// Using JSON serialization exclusively on the control stream has no measurable impact on file transmission performance
	return json.Marshal(request)
}

func receiveMetaDataFile(payloadBytes []byte) (*Request, error) {
	request := Request{}
	err := json.Unmarshal(payloadBytes, &request)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func buildMetaDataCancelFile() ([]byte, error) {
	cancelRequest := Request{Method: CANCEL}
	return json.Marshal(cancelRequest)
}

func buildMetadataResponse(code StatusCode) ([]byte, error) {
	response := Response{Status: code}
	return json.Marshal(response)
}

func receiveMetadataResponse(payloadBytes []byte) StatusCode {
	response := Response{}
	json.Unmarshal(payloadBytes, &response)
	return response.Status
}
