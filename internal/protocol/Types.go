package protocol

import (
	"errors"

	"ftu/internal/transfer"
)

type Method string
type StatusCode int

type Request struct {
	FileName string
	FileSize int64
	FileHash string
}

type Response struct {
	Status StatusCode
}

const (
	StatusContinue      StatusCode = 100 // Metadata accepted
	StatusOK            StatusCode = 200 // OK
	StatusBadRequest    StatusCode = 400 // Metadata error
	StatusForbidden     StatusCode = 403 // Rejected by other peer
	StatusNotAcceptable StatusCode = 406 // Rejected by the SO (ex: no space)
	StatusBusy          StatusCode = 429 // another offer is pending or being received
	StatusConflict      StatusCode = 409 // file integrity failure (Hash)
	StatusClientClosed  StatusCode = 499 // cancelled by sender
	StatusInternalError StatusCode = 500 // I/O failure
)

// StatusForReceiveError maps a receive failure to the status sent back to the sender:
// 409 only for an integrity failure (hash/size mismatch), 406 when the disk is full,
// 500 for every other I/O problem.
func StatusForReceiveError(err error) StatusCode {
	switch {
	case errors.Is(err, transfer.ErrIntegrity):
		return StatusConflict
	case errors.Is(err, transfer.ErrInsufficientSpace):
		return StatusNotAcceptable
	default:
		return StatusInternalError
	}
}
