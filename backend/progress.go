package backend

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusSkipped     DownloadStatus = "skipped"
)

type DownloadItem struct {
	ID           string         `json:"id"`
	TrackName    string         `json:"track_name"`
	ArtistName   string         `json:"artist_name"`
	AlbumName    string         `json:"album_name"`
	SpotifyID    string         `json:"spotify_id"`
	Status       DownloadStatus `json:"status"`
	Progress     float64        `json:"progress"`
	TotalSize    float64        `json:"total_size"`
	Speed        float64        `json:"speed"`
	StartTime    int64          `json:"start_time"`
	EndTime      int64          `json:"end_time"`
	ErrorMessage string         `json:"error_message"`
	FilePath     string         `json:"file_path"`
}

var (
	currentProgress     float64
	currentProgressLock sync.RWMutex
	isDownloading       bool
	downloadingLock     sync.RWMutex
	currentSpeed        float64
	speedLock           sync.RWMutex

	rateLimitUntilMs int64
	rateLimitLock    sync.RWMutex

	cooldownUntilMs int64
	cooldownMessage string
	cooldownEventID int64
	cooldownLock    sync.RWMutex

	downloadQueue       []DownloadItem
	downloadQueueLock   sync.RWMutex
	currentItemID       string
	currentItemLock     sync.RWMutex
	totalDownloaded     float64
	totalDownloadedLock sync.RWMutex
	sessionStartTime    int64
	sessionStartLock    sync.RWMutex
)

type ProgressInfo struct {
	IsDownloading   bool    `json:"is_downloading"`
	MBDownloaded    float64 `json:"mb_downloaded"`
	SpeedMBps       float64 `json:"speed_mbps"`
	RateLimited     bool    `json:"rate_limited"`
	RateLimitSecs   int     `json:"rate_limit_secs"`
	Cooldown        bool    `json:"cooldown"`
	CooldownSecs    int     `json:"cooldown_secs"`
	CooldownMessage string  `json:"cooldown_message"`
	CooldownEventID int64   `json:"cooldown_event_id"`
}

type DownloadQueueInfo struct {
	IsDownloading    bool           `json:"is_downloading"`
	Queue            []DownloadItem `json:"queue"`
	CurrentSpeed     float64        `json:"current_speed"`
	TotalDownloaded  float64        `json:"total_downloaded"`
	SessionStartTime int64          `json:"session_start_time"`
	QueuedCount      int            `json:"queued_count"`
	CompletedCount   int            `json:"completed_count"`
	FailedCount      int            `json:"failed_count"`
	SkippedCount     int            `json:"skipped_count"`
}

func GetDownloadProgress() ProgressInfo {
	downloadingLock.RLock()
	downloading := isDownloading
	downloadingLock.RUnlock()

	currentProgressLock.RLock()
	progress := currentProgress
	currentProgressLock.RUnlock()

	speedLock.RLock()
	speed := currentSpeed
	speedLock.RUnlock()

	rateLimitLock.RLock()
	untilMs := rateLimitUntilMs
	rateLimitLock.RUnlock()

	rateLimited := false
	rateLimitSecs := 0
	if untilMs > 0 {
		remainingMs := untilMs - getCurrentTimeMillis()
		if remainingMs > 0 {
			rateLimited = true
			rateLimitSecs = int((remainingMs + 999) / 1000)
		}
	}

	cooldownLock.RLock()
	cdUntilMs := cooldownUntilMs
	cdMessage := cooldownMessage
	cdEventID := cooldownEventID
	cooldownLock.RUnlock()

	cooldown := false
	cooldownSecs := 0
	if cdUntilMs > 0 {
		remainingMs := cdUntilMs - getCurrentTimeMillis()
		if remainingMs > 0 {
			cooldown = true
			cooldownSecs = int((remainingMs + 999) / 1000)
		} else {
			cdMessage = ""
		}
	}

	return ProgressInfo{
		IsDownloading:   downloading,
		MBDownloaded:    progress,
		SpeedMBps:       speed,
		RateLimited:     rateLimited,
		RateLimitSecs:   rateLimitSecs,
		Cooldown:        cooldown,
		CooldownSecs:    cooldownSecs,
		CooldownMessage: cdMessage,
		CooldownEventID: cdEventID,
	}
}

func SetRateLimitCooldown(seconds float64) {
	rateLimitLock.Lock()
	if seconds <= 0 {
		rateLimitUntilMs = 0
	} else {
		rateLimitUntilMs = getCurrentTimeMillis() + int64(seconds*1000)
	}
	rateLimitLock.Unlock()
}

func ClearRateLimitCooldown() {
	rateLimitLock.Lock()
	rateLimitUntilMs = 0
	rateLimitLock.Unlock()
}

func SetCommunityCooldown(seconds float64, message string) {
	cooldownLock.Lock()
	if seconds <= 0 {
		cooldownUntilMs = 0
		cooldownMessage = ""
	} else {
		cooldownUntilMs = getCurrentTimeMillis() + int64(seconds*1000)
		cooldownMessage = message
		cooldownEventID++
	}
	cooldownLock.Unlock()
}

func ClearCommunityCooldown() {
	cooldownLock.Lock()
	cooldownUntilMs = 0
	cooldownMessage = ""
	cooldownLock.Unlock()
}

func SetDownloadSpeed(mbps float64) {
	speedLock.Lock()
	currentSpeed = mbps
	speedLock.Unlock()
}

func SetDownloadProgress(mbDownloaded float64) {
	currentProgressLock.Lock()
	currentProgress = mbDownloaded
	currentProgressLock.Unlock()
}

func SetDownloading(downloading bool) {
	downloadingLock.Lock()
	isDownloading = downloading
	downloadingLock.Unlock()

	if !downloading {

		SetDownloadProgress(0)
		SetDownloadSpeed(0)
		ClearRateLimitCooldown()
	}
}

type ProgressWriter struct {
	writer      io.Writer
	total       int64
	lastPrinted int64
	startTime   int64
	lastTime    int64
	lastBytes   int64
	itemID      string
}

func NewProgressWriter(writer io.Writer) *ProgressWriter {
	now := getCurrentTimeMillis()
	return &ProgressWriter{
		writer:      writer,
		total:       0,
		lastPrinted: 0,
		startTime:   now,
		lastTime:    now,
		lastBytes:   0,
		itemID:      "",
	}
}

func NewProgressWriterWithID(writer io.Writer, itemID string) *ProgressWriter {
	pw := NewProgressWriter(writer)
	pw.itemID = itemID
	return pw
}

func getCurrentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	if err := CheckDownloadCancelled(); err != nil {
		return 0, err
	}

	n, err := pw.writer.Write(p)
	pw.total += int64(n)

	if pw.total-pw.lastPrinted >= 256*1024 {
		mbDownloaded := float64(pw.total) / (1024 * 1024)

		now := getCurrentTimeMillis()
		timeDiff := float64(now-pw.lastTime) / 1000.0
		bytesDiff := float64(pw.total - pw.lastBytes)

		var speedMBps float64
		if timeDiff > 0 {
			speedMBps = (bytesDiff / (1024 * 1024)) / timeDiff
			SetDownloadSpeed(speedMBps)
			fmt.Printf("\rDownloaded: %.2f MB (%.2f MB/s)", mbDownloaded, speedMBps)
		} else {
			fmt.Printf("\rDownloaded: %.2f MB", mbDownloaded)
		}

		SetDownloadProgress(mbDownloaded)

		if pw.itemID != "" {
			UpdateItemProgress(pw.itemID, mbDownloaded, speedMBps)
		}

		pw.lastPrinted = pw.total
		pw.lastTime = now
		pw.lastBytes = pw.total
	}

	return n, err
}

func (pw *ProgressWriter) GetTotal() int64 {
	return pw.total
}

func AddToQueue(id, trackName, artistName, albumName, spotifyID string) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	item := DownloadItem{
		ID:         id,
		TrackName:  trackName,
		ArtistName: artistName,
		AlbumName:  albumName,
		SpotifyID:  spotifyID,
		Status:     StatusQueued,
		Progress:   0,
		TotalSize:  0,
		Speed:      0,
		StartTime:  0,
		EndTime:    0,
	}

	downloadQueue = append(downloadQueue, item)

	sessionStartLock.Lock()
	if sessionStartTime == 0 {
		sessionStartTime = time.Now().Unix()
	}
	sessionStartLock.Unlock()
}

func StartDownloadItem(id string) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].ID == id {
			downloadQueue[i].Status = StatusDownloading
			downloadQueue[i].StartTime = time.Now().Unix()
			downloadQueue[i].Progress = 0
			break
		}
	}

	currentItemLock.Lock()
	currentItemID = id
	currentItemLock.Unlock()
}

func UpdateItemProgress(id string, progress, speed float64) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].ID == id {
			downloadQueue[i].Progress = progress
			downloadQueue[i].Speed = speed
			break
		}
	}
}

func GetCurrentItemID() string {
	currentItemLock.RLock()
	defer currentItemLock.RUnlock()
	return currentItemID
}

func CompleteDownloadItem(id, filePath string, finalSize float64) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].ID == id {
			downloadQueue[i].Status = StatusCompleted
			downloadQueue[i].EndTime = time.Now().Unix()
			downloadQueue[i].FilePath = filePath
			downloadQueue[i].Progress = finalSize
			downloadQueue[i].TotalSize = finalSize

			totalDownloadedLock.Lock()
			totalDownloaded += finalSize
			totalDownloadedLock.Unlock()
			break
		}
	}
}

func FailDownloadItem(id, errorMsg string) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].ID == id {
			downloadQueue[i].Status = StatusFailed
			downloadQueue[i].EndTime = time.Now().Unix()
			downloadQueue[i].ErrorMessage = errorMsg
			break
		}
	}
}

func SkipDownloadItem(id, filePath string) {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].ID == id {
			downloadQueue[i].Status = StatusSkipped
			downloadQueue[i].EndTime = time.Now().Unix()
			downloadQueue[i].FilePath = filePath
			break
		}
	}
}

func GetDownloadQueue() DownloadQueueInfo {

	ResetSessionIfComplete()

	downloadQueueLock.RLock()
	defer downloadQueueLock.RUnlock()

	downloadingLock.RLock()
	downloading := isDownloading
	downloadingLock.RUnlock()

	speedLock.RLock()
	speed := currentSpeed
	speedLock.RUnlock()

	totalDownloadedLock.RLock()
	total := totalDownloaded
	totalDownloadedLock.RUnlock()

	sessionStartLock.RLock()
	sessionStart := sessionStartTime
	sessionStartLock.RUnlock()

	var queued, completed, failed, skipped int
	for _, item := range downloadQueue {
		switch item.Status {
		case StatusQueued:
			queued++
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		case StatusSkipped:
			skipped++
		}
	}

	queueCopy := make([]DownloadItem, len(downloadQueue))
	copy(queueCopy, downloadQueue)

	return DownloadQueueInfo{
		IsDownloading:    downloading,
		Queue:            queueCopy,
		CurrentSpeed:     speed,
		TotalDownloaded:  total,
		SessionStartTime: sessionStart,
		QueuedCount:      queued,
		CompletedCount:   completed,
		FailedCount:      failed,
		SkippedCount:     skipped,
	}
}

func ClearDownloadQueue() {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	newQueue := make([]DownloadItem, 0)
	for _, item := range downloadQueue {
		if item.Status == StatusQueued || item.Status == StatusDownloading {
			newQueue = append(newQueue, item)
		}
	}
	downloadQueue = newQueue
}

func ClearAllDownloads() {
	downloadQueueLock.Lock()
	downloadQueue = []DownloadItem{}
	downloadQueueLock.Unlock()

	totalDownloadedLock.Lock()
	totalDownloaded = 0
	totalDownloadedLock.Unlock()

	sessionStartLock.Lock()
	sessionStartTime = 0
	sessionStartLock.Unlock()

	currentItemLock.Lock()
	currentItemID = ""
	currentItemLock.Unlock()

	SetDownloadProgress(0)
	SetDownloadSpeed(0)
}

func CancelAllQueuedItems() {
	downloadQueueLock.Lock()
	defer downloadQueueLock.Unlock()

	for i := range downloadQueue {
		if downloadQueue[i].Status == StatusQueued {
			downloadQueue[i].Status = StatusSkipped
			downloadQueue[i].EndTime = time.Now().Unix()
			downloadQueue[i].ErrorMessage = "Cancelled"
		}
	}
}

func CancelQueuedAndDownloadingItems() {
	downloadQueueLock.Lock()
	for i := range downloadQueue {
		if downloadQueue[i].Status == StatusQueued || downloadQueue[i].Status == StatusDownloading {
			downloadQueue[i].Status = StatusSkipped
			downloadQueue[i].EndTime = time.Now().Unix()
			downloadQueue[i].ErrorMessage = "Cancelled"
		}
	}
	downloadQueueLock.Unlock()

	currentItemLock.Lock()
	currentItemID = ""
	currentItemLock.Unlock()

	SetDownloadProgress(0)
	SetDownloadSpeed(0)
}

func ResetSessionIfComplete() {
	downloadQueueLock.RLock()
	hasActiveOrQueued := false
	for _, item := range downloadQueue {
		if item.Status == StatusQueued || item.Status == StatusDownloading {
			hasActiveOrQueued = true
			break
		}
	}
	downloadQueueLock.RUnlock()

	if !hasActiveOrQueued {
		sessionStartLock.Lock()
		sessionStartTime = 0
		sessionStartLock.Unlock()

		totalDownloadedLock.Lock()
		totalDownloaded = 0
		totalDownloadedLock.Unlock()
	}
}
