package ui

import (
	"context"
	"path/filepath"
	"time"

	"ftu/internal/protocol"
	"ftu/internal/transfer"

	"github.com/quic-go/quic-go"
)

const (
	// controlRequestTimeout bounds how long a peer may take to send its metadata request.
	controlRequestTimeout = 30 * time.Second
	// offerDecisionTimeout is how long the user has to answer an offer.
	offerDecisionTimeout = 120 * time.Second
)

func (a *App) startReceiverLoop() {
	for {
		a.mu.Lock()
		conn, confirmed := a.quicConn, a.confirmed
		a.mu.Unlock()

		if conn == nil || !confirmed { // the data channel only exists after pairing confirmation
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

	// A peer must send its request promptly; an idle control stream must not hold a goroutine.
	_ = controlStream.SetReadDeadline(time.Now().Add(controlRequestTimeout))
	req, err := protocol.ReceiveMetaDataRequest(controlStream)
	_ = controlStream.SetReadDeadline(time.Time{})
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusBadRequest, controlStream)
		return
	}
	// The name shown to the user is the name that will really be used on disk.
	safeName, err := transfer.SanitizeFileName(req.FileName)
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusBadRequest, controlStream)
		return
	}

	// One offer at a time: while a decision is pending or an accepted file is being received,
	// further offers are refused instead of sharing (and racing for) one decision.
	id, err := newOfferID()
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusInternalError, controlStream)
		return
	}
	offer := &pendingOffer{id: id, decision: make(chan bool, 1)}
	a.mu.Lock()
	if a.offer != nil || a.receiving {
		a.mu.Unlock()
		_ = protocol.SendCodeResponse(protocol.StatusBusy, controlStream)
		return
	}
	a.offer = offer
	a.mu.Unlock()

	a.emit(EventTransferIncoming, IncomingTransferDTO{
		OfferID:  offer.id,
		FileName: safeName,
		FileSize: req.FileSize,
		FileHash: req.FileHash,
	})

	var accepted bool
	select {
	case accepted = <-offer.decision:
	case <-time.After(offerDecisionTimeout):
	case <-conn.Context().Done():
	}
	a.mu.Lock()
	a.offer = nil
	if accepted {
		a.receiving = true
	}
	a.mu.Unlock()
	if accepted {
		defer func() {
			a.mu.Lock()
			a.receiving = false
			a.mu.Unlock()
		}()
	}

	if !accepted {
		_ = protocol.SendCodeResponse(protocol.StatusForbidden, controlStream)
		a.emit(EventTransferStatus, "rejected")
		return
	}

	a.mu.Lock()
	dir := a.downloadDir
	a.mu.Unlock()
	if err := transfer.EnsureFreeSpace(dir, req.FileSize); err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusNotAcceptable, controlStream)
		a.emit(EventTransferError, TransferErrorDTO{Code: "NO_SPACE", Message: err.Error()})
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

	a.mu.Lock()
	downloadDir := a.downloadDir
	a.mu.Unlock()

	start := time.Now()
	destPath, written, err := transfer.ReceiveFile(downloadDir, req.FileName, conn, req.FileSize, req.FileHash, a.newProgressEmitter(req.FileSize))
	if err != nil {
		_ = protocol.SendCodeResponse(protocol.StatusForReceiveError(err), controlStream)
		a.emit(EventTransferError, TransferErrorDTO{
			Code:    "RECEIVE_FAILED",
			Message: err.Error(),
		})
		return
	}
	destName := filepath.Base(destPath)

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
