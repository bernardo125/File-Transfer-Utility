package ui

import (
	"time"

	"ftu/internal/transfer"

	"github.com/quic-go/quic-go"
)

// progressEmitInterval caps how often transfer:progress events reach the frontend.
const progressEmitInterval = 150 * time.Millisecond

// newProgressEmitter returns a transfer.ProgressFunc that converts measured byte counts
// into throttled transfer:progress events. Speed and ETA are derived from real elapsed time.
func (a *App) newProgressEmitter(totalBytes int64) transfer.ProgressFunc {
	start := time.Now()
	var lastEmit time.Time

	return func(done int64) {
		now := time.Now()
		if done < totalBytes && now.Sub(lastEmit) < progressEmitInterval {
			return
		}
		lastEmit = now

		elapsed := now.Sub(start).Seconds()
		var speedMBs float64
		var etaSeconds int64
		if elapsed > 0 {
			bytesPerSec := float64(done) / elapsed
			speedMBs = bytesPerSec / (1024 * 1024)
			if bytesPerSec > 0 && totalBytes > done {
				etaSeconds = int64(float64(totalBytes-done)/bytesPerSec + 0.5)
			}
		}

		var percentage float64
		if totalBytes > 0 {
			percentage = float64(done) / float64(totalBytes) * 100
		}

		a.emit(EventTransferProgress, TransferProgressDTO{
			BytesTransferred: done,
			TotalBytes:       totalBytes,
			Percentage:       percentage,
			SpeedMBs:         speedMBs,
			EtaSeconds:       etaSeconds,
		})
	}
}

// notifyPeerLost tells the frontend that the QUIC session ended without a local Disconnect call.
// Disconnect clears a.quicConn first, so a still-current conn means the peer (or network) closed it.
func (a *App) notifyPeerLost(conn *quic.Conn) {
	a.mu.Lock()
	current := a.quicConn == conn
	a.mu.Unlock()
	if !current {
		return
	}
	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "error",
		Detail: "Peer disconnected",
	})
}
