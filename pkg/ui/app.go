package ui

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"

	"ftu/internal/connectivity"
	"ftu/internal/discovery"
	"ftu/internal/protocol"
	"ftu/internal/transfer"
	"ftu/internal/transport"

	"github.com/quic-go/quic-go"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the central Wails application struct handling IPC bindings and state.
type App struct {
	ctx                context.Context
	mu                 sync.Mutex
	conn               *net.UDPConn
	myToken            string
	myPrivKey          ed25519.PrivateKey
	peerPubKey         *ed25519.PublicKey
	quicConn           *quic.Conn
	downloadDir        string
	incomingDecisionCh chan bool
	cancelConnect      context.CancelFunc
	connectMu          sync.Mutex
	emitter            func(event string, data ...any) // nil in production: events go through the Wails runtime
}

// NewApp creates a new App instance.
func NewApp() *App {
	downloadDir := ""
	if usr, err := user.Current(); err == nil {
		downloads := filepath.Join(usr.HomeDir, "Downloads")
		if _, err := os.Stat(downloads); err == nil {
			downloadDir = downloads
		} else {
			downloadDir = usr.HomeDir
		}
	} else {
		downloadDir = "."
	}

	return &App{
		downloadDir:        downloadDir,
		incomingDecisionCh: make(chan bool, 1),
	}
}

// Startup is invoked by Wails when the runtime is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is invoked by Wails when the application is closing.
func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.quicConn != nil {
		_ = a.quicConn.CloseWithError(0, "Application shutdown")
		a.quicConn = nil
	}
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn = nil
	}
}

// GetMyToken initializes the UDP socket and retrieves or generates the local signaling token.
func (a *App) GetMyToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.myToken != "" && a.conn != nil {
		return a.myToken, nil
	}

	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "discovery",
		Detail: "Resolving STUN and preparing local keys...",
	})

	conn := transport.InitializeUDPSocket()
	token, privKey := discovery.CreateKey(conn)

	a.conn = conn
	a.myToken = token
	a.myPrivKey = privKey

	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "ready",
		Detail: "Signaling token ready to share",
	})

	return token, nil
}

// GetLocalToken retrieves or generates the local signaling token via STUN.
func (a *App) GetLocalToken() (string, error) {
	return a.GetMyToken()
}

// CancelConnection interrupts any in-progress UDP hole punching or handshake.
func (a *App) CancelConnection() error {
	a.connectMu.Lock()
	if a.cancelConnect != nil {
		a.cancelConnect()
		a.cancelConnect = nil
	}
	a.connectMu.Unlock()

	a.mu.Lock()
	if a.conn != nil {
		_ = a.conn.SetReadDeadline(time.Now())
		time.Sleep(20 * time.Millisecond)
		_ = a.conn.SetReadDeadline(time.Time{})
	}
	a.mu.Unlock()

	a.emit(EventConnectionFailed, ConnectionFailedDTO{
		Reason: "Ligação cancelada pelo utilizador",
	})
	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "ready",
		Detail: "Tentativa de ligação cancelada",
	})

	return nil
}

// ConnectPeer initiates UDP hole punching and QUIC handshake with the remote peer asynchronously.
func (a *App) ConnectPeer(peerToken string) error {
	a.mu.Lock()
	if a.conn == nil || a.myToken == "" {
		a.mu.Unlock()
		_, err := a.GetMyToken()
		if err != nil {
			return fmt.Errorf("failed to initialize local connection: %w", err)
		}
		a.mu.Lock()
	}

	conn := a.conn
	privKey := a.myPrivKey
	a.mu.Unlock()

	if len(peerToken) <= 12 {
		return fmt.Errorf("invalid token length: must be longer than 12 characters")
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.connectMu.Lock()
	if a.cancelConnect != nil {
		a.cancelConnect()
	}
	a.cancelConnect = cancel
	a.connectMu.Unlock()

	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "hole_punching",
		Detail: "Decoding token and starting UDP Hole Punching...",
	})

	go func() {
		peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, peerPubKey, err := discovery.GetPeer(peerToken)
		if err != nil {
			a.emit(EventConnectionFailed, ConnectionFailedDTO{
				Reason: fmt.Sprintf("Token inválido: %v", err),
			})
			a.emit(EventConnectionState, ConnectionStateDTO{
				State:  "error",
				Detail: fmt.Sprintf("Token inválido: %v", err),
			})
			return
		}

		a.mu.Lock()
		a.peerPubKey = peerPubKey
		a.mu.Unlock()

		punchCh := connectivity.UDPHolePunch(conn, peerLocalIPV4, peerRemoteIPV4, peerPublicIPV6, &privKey, peerPubKey)

		var punchResult connectivity.HolePunchResult
		select {
		case <-ctx.Done():
			a.emit(EventConnectionFailed, ConnectionFailedDTO{
				Reason: "Ligação cancelada pelo utilizador",
			})
			return
		case punchResult = <-punchCh:
		}

		if punchResult.Err != nil {
			a.emit(EventConnectionFailed, ConnectionFailedDTO{
				Reason: fmt.Sprintf("Falha no Hole Punch: %v", punchResult.Err),
			})
			a.emit(EventConnectionState, ConnectionStateDTO{
				State:  "error",
				Detail: fmt.Sprintf("Falha no Hole Punch: %v", punchResult.Err),
			})
			return
		}

		a.emit(EventConnectionState, ConnectionStateDTO{
			State:  "upgrading",
			Detail: fmt.Sprintf("Hole punch com sucesso (%v). Upgrading para QUIC...", punchResult.Rtt),
			RttMs:  punchResult.Rtt.Milliseconds(),
		})

		quicConn, err := transport.QuicUpgrade(conn, punchResult.Addr, &privKey, peerPubKey)
		if err != nil {
			a.emit(EventConnectionFailed, ConnectionFailedDTO{
				Reason: fmt.Sprintf("Falha no upgrade QUIC: %v", err),
			})
			a.emit(EventConnectionState, ConnectionStateDTO{
				State:  "error",
				Detail: fmt.Sprintf("Falha no upgrade QUIC: %v", err),
			})
			return
		}

		a.mu.Lock()
		a.quicConn = quicConn
		a.mu.Unlock()

		a.emit(EventConnectionSuccess, ConnectionSuccessDTO{
			Addr:  punchResult.Addr.String(),
			RttMs: punchResult.Rtt.Milliseconds(),
		})

		a.emit(EventConnectionState, ConnectionStateDTO{
			State:  "connected",
			Detail: fmt.Sprintf("Conectado a %s", punchResult.Addr.String()),
			RttMs:  punchResult.Rtt.Milliseconds(),
		})

		go a.startReceiverLoop()
	}()

	return nil
}

// Disconnect cleanly shuts down the active connection and socket.
func (a *App) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.quicConn != nil {
		_ = a.quicConn.CloseWithError(0, "Disconnected by user")
		a.quicConn = nil
	}
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn = nil
	}
	a.myToken = ""

	a.emit(EventConnectionState, ConnectionStateDTO{
		State:  "idle",
		Detail: "Disconnected",
	})
	return nil
}

// SelectFileNative opens the native OS file picker and extracts file metadata.
func (a *App) SelectFileNative() (*FileInfoDTO, error) {
	filePath, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select File to Send",
	})
	if err != nil || filePath == "" {
		return nil, err
	}

	return a.InspectFile(filePath)
}

// InspectFile inspects a file path and calculates its SHA256 checksum and metadata.
func (a *App) InspectFile(filePath string) (*FileInfoDTO, error) {
	file, stats, hash, err := transfer.SelectFileToSend(filePath)
	if err != nil {
		return nil, err
	}
	_ = file.Close()

	return &FileInfoDTO{
		Path: filePath,
		Name: stats.Name(),
		Size: stats.Size(),
		Hash: hash,
	}, nil
}

// SelectDownloadDirectory opens the native directory dialog to choose where received files are saved.
func (a *App) SelectDownloadDirectory() (string, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Download Directory",
	})
	if err != nil || dir == "" {
		return a.downloadDir, err
	}

	a.mu.Lock()
	a.downloadDir = dir
	a.mu.Unlock()

	return dir, nil
}

// GetDownloadDirectory returns the currently configured download folder.
func (a *App) GetDownloadDirectory() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.downloadDir
}

// SendSelectedFile sends the selected file to the connected peer over QUIC.
func (a *App) SendSelectedFile(filePath string) error {
	a.mu.Lock()
	conn := a.quicConn
	a.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("no active peer connection")
	}

	file, stats, hash, err := transfer.SelectFileToSend(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	a.emit(EventTransferStatus, "requesting")

	controlStream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open QUIC control stream: %w", err)
	}
	defer controlStream.Close()

	err = protocol.SendMetaDataRequest(stats, hash, controlStream)
	if err != nil {
		return fmt.Errorf("failed to send metadata request: %w", err)
	}

	a.emit(EventTransferStatus, "waiting_approval")

	response, err := protocol.ReceiveMetadataResponse(controlStream)
	if err != nil {
		return fmt.Errorf("failed to receive metadata response: %w", err)
	}

	if response.Status != protocol.StatusOK && response.Status != protocol.StatusContinue {
		a.emit(EventTransferStatus, "rejected")
		return fmt.Errorf("transfer rejected by peer (status: %d)", response.Status)
	}

	a.emit(EventTransferStatus, "transferring")
	a.emit(EventTransferProgress, TransferProgressDTO{
		BytesTransferred: 0,
		TotalBytes:       stats.Size(),
		Percentage:       0,
	})

	startTime := time.Now()
	err = transfer.SendFile(file, conn, a.newProgressEmitter(stats.Size()))
	if err != nil {
		a.emit(EventTransferError, TransferErrorDTO{
			Code:    "SEND_FAILED",
			Message: err.Error(),
		})
		return fmt.Errorf("error during file streaming: %w", err)
	}

	a.emit(EventTransferStatus, "verifying")

	finalStatus, err := protocol.ReceiveMetadataResponse(controlStream)
	if err != nil {
		return fmt.Errorf("failed to receive integrity verification: %w", err)
	}

	if finalStatus.Status != protocol.StatusOK {
		a.emit(EventTransferError, TransferErrorDTO{
			Code:    "INTEGRITY_FAILED",
			Message: "Peer reported checksum verification failure",
		})
		return fmt.Errorf("peer reported integrity verification failure")
	}

	elapsed := time.Since(startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	speedMBs := float64(stats.Size()) / (1024 * 1024 * elapsed)

	a.emit(EventTransferProgress, TransferProgressDTO{
		BytesTransferred: stats.Size(),
		TotalBytes:       stats.Size(),
		Percentage:       100,
		SpeedMBs:         speedMBs,
		EtaSeconds:       0,
	})

	a.emit(EventTransferCompleted, TransferCompletedDTO{
		FileName:     stats.Name(),
		DestPath:     filePath,
		FileSize:     stats.Size(),
		DurationSec:  elapsed,
		SpeedMBs:     speedMBs,
		HashVerified: true,
	})

	return nil
}

// RespondIncomingTransfer signals the receiver loop whether to accept or reject an incoming file.
func (a *App) RespondIncomingTransfer(accept bool) error {
	select {
	case a.incomingDecisionCh <- accept:
		return nil
	default:
		return fmt.Errorf("no pending incoming transfer to respond to")
	}
}

// OpenFolder opens the containing folder of a file or a directory in the native file manager.
func (a *App) OpenFolder(path string) error {
	info, err := os.Stat(path)
	targetDir := path
	if err == nil && !info.IsDir() {
		targetDir = filepath.Dir(path)
	}

	switch goruntime.GOOS {
	case "windows":
		return exec.Command("explorer", targetDir).Start()
	case "darwin":
		return exec.Command("open", targetDir).Start()
	default:
		return exec.Command("xdg-open", targetDir).Start()
	}
}

// CopyToClipboard copies text to the native clipboard.
func (a *App) CopyToClipboard(text string) error {
	return wailsruntime.ClipboardSetText(a.ctx, text)
}

// GetClipboardText retrieves text from the native clipboard.
func (a *App) GetClipboardText() (string, error) {
	return wailsruntime.ClipboardGetText(a.ctx)
}

// QuitApplication terminates the desktop application cleanly.
func (a *App) QuitApplication() {
	if a.ctx != nil {
		wailsruntime.Quit(a.ctx)
	}
}

// OpenGitHub opens the project GitHub repository in the default web browser.
func (a *App) OpenGitHub() {
	if a.ctx != nil {
		wailsruntime.BrowserOpenURL(a.ctx, "https://github.com/bernardo125/File-Transfer-Utility")
	}
}
