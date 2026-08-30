package protocol

type Method string
type StatusCode int

type Request struct {
	Method   Method
	FileName string
	FileSize int64
	FileHash string
}

type Response struct {
	Status StatusCode
}

const (
	SEND   Method = "SEND"
	CANCEL Method = "CANCEL"
)

const (
	StatusContinue      StatusCode = 100 // Metadata accepted
	StatusOK            StatusCode = 200 // OK
	StatusBadRequest    StatusCode = 400 // Metadata error
	StatusForbidden     StatusCode = 403 // Rejected by other peer
	StatusNotAcceptable StatusCode = 406 // Rejected by the SO (ex: no space)
	StatusConflict      StatusCode = 409 // file integrity failure (Hash)
	StatusClientClosed  StatusCode = 499 // cancelled by sender
	StatusInternalError StatusCode = 500 // I/O failure
)
