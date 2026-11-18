package ui

const (
	EventConnectionState   = "p2p:connection-state"
	EventConnectionSuccess = "connection:success"
	EventConnectionFailed  = "connection:failed"
	EventTransferIncoming  = "transfer:incoming"
	EventTransferProgress  = "transfer:progress"
	EventTransferStatus    = "transfer:status"
	EventTransferCompleted = "transfer:completed"
	EventTransferError     = "transfer:error"
)

type ConnectionStateDTO struct {
	State  string `json:"state"` // "idle" | "discovery" | "ready" | "hole_punching" | "upgrading" | "connected" | "error"
	Detail string `json:"detail"`
	RttMs  int64  `json:"rttMs"`
}

type ConnectionSuccessDTO struct {
	Addr  string `json:"addr"`
	RttMs int64  `json:"rttMs"`
}

type ConnectionFailedDTO struct {
	Reason string `json:"reason"`
}

type FileInfoDTO struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Hash string `json:"hash"`
}

type IncomingTransferDTO struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileHash string `json:"fileHash"`
}

type TransferProgressDTO struct {
	BytesTransferred int64   `json:"bytesTransferred"`
	TotalBytes       int64   `json:"totalBytes"`
	Percentage       float64 `json:"percentage"`
	SpeedMBs         float64 `json:"speedMBs"`
	EtaSeconds       int64   `json:"etaSeconds"`
}

type TransferCompletedDTO struct {
	FileName     string  `json:"fileName"`
	DestPath     string  `json:"destPath"`
	FileSize     int64   `json:"fileSize"`
	DurationSec  float64 `json:"durationSec"`
	SpeedMBs     float64 `json:"speedMBs"`
	HashVerified bool    `json:"hashVerified"`
}

type TransferErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
