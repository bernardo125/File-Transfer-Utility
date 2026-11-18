package ui

import (
	"context"
	"path/filepath"
	"time"

	"ftu/internal/protocol"
	"ftu/internal/transfer"

	"github.com/quic-go/quic-go"
)

func (a *App) startReceiverLoop() {
	for {
		a.mu.Lock()
		conn := a.quicConn
		a.mu.Unlock()

		if conn == nil {
			return
		}

		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			a.notifyPeerLost(conn)
			return
		}

		go a.handleIncomingStream(stream, conn)
	}
}

func (a *App) handleIncomingStream(controlStream *quic.Stream, conn *quic.Conn) {
	defer controlStream.Close()

	req, err := protocol.ReceiveMetaDataRequest(controlStream)
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusBadRequest, controlStream)
		return
	}

	a.emit(EventTransferIncoming, IncomingTransferDTO{
		FileName: req.FileName,
		FileSize: req.FileSize,
		FileHash: req.FileHash,
	})

	var accepted bool
	select {
	case accepted = <-a.incomingDecisionCh:
	case <-time.After(120 * time.Second):
		accepted = false
	}

	if !accepted {
		_ = protocol.SendCodeResponse(protocol.StatusForbidden, controlStream)
		a.emit(EventTransferStatus, "rejected")
		return
	}

	err = protocol.SendCodeResponse(protocol.StatusContinue, controlStream)
	if err != nil {
		return
	}

	a.emit(EventTransferStatus, "transferring")
	a.emit(EventTransferProgress, TransferProgressDTO{
		BytesTransferred: 0,
		TotalBytes:       req.FileSize,
		Percentage:       0,
	})

	destName := filepath.Base(req.FileName)
	if destName == "" || destName == "." || destName == "/" {
		destName = "received_file"
	}

	a.mu.Lock()
	destPath := filepath.Join(a.downloadDir, destName)
	a.mu.Unlock()

	start := time.Now()
	written, err := transfer.ReceiveFile(destPath, conn, req.FileSize, req.FileHash, a.newProgressEmitter(req.FileSize))
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusConflict, controlStream)
		a.emit(EventTransferError, TransferErrorDTO{
			Code:    "RECEIVE_FAILED",
			Message: err.Error(),
		})
		return
	}

	err = protocol.SendCodeResponse(protocol.StatusOK, controlStream)
	if err != nil {
		return
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speedMBs := float64(written) / (1024 * 1024 * elapsed)

	a.emit(EventTransferProgress, TransferProgressDTO{
		BytesTransferred: written,
		TotalBytes:       req.FileSize,
		Percentage:       100,
		SpeedMBs:         speedMBs,
		EtaSeconds:       0,
	})

	a.emit(EventTransferCompleted, TransferCompletedDTO{
		FileName:     destName,
		DestPath:     destPath,
		FileSize:     written,
		DurationSec:  elapsed,
		SpeedMBs:     speedMBs,
		HashVerified: true,
	})
}
