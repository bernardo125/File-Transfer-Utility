package protocol

type Method string
type Header string
type StatusCode int

const (
	MethodSend   Method = "SEND"
	MethodCancel Method = "CANCEL"
)

const (
	HeaderFileName Header = "File-Name"
	HeaderFileSize Header = "File-Size" // size in bytes
	HeaderFileHash Header = "File-Hash" // hash to verify file integrity
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
