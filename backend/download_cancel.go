package backend

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrDownloadCancelled = errors.New("download cancelled")

var downloadCancelState = struct {
	sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	active   int
	stopping bool
}{}

func BeginDownloadCancellationScope() (context.Context, func()) {
	downloadCancelState.Lock()
	defer downloadCancelState.Unlock()

	if downloadCancelState.ctx == nil || downloadCancelState.active == 0 {
		downloadCancelState.ctx, downloadCancelState.cancel = context.WithCancel(context.Background())
		downloadCancelState.stopping = false
	}

	downloadCancelState.active++
	ctx := downloadCancelState.ctx
	once := sync.Once{}

	return ctx, func() {
		once.Do(func() {
			downloadCancelState.Lock()
			defer downloadCancelState.Unlock()

			if downloadCancelState.active > 0 {
				downloadCancelState.active--
			}
			if downloadCancelState.active == 0 {
				if downloadCancelState.cancel != nil {
					downloadCancelState.cancel()
				}
				downloadCancelState.ctx = nil
				downloadCancelState.cancel = nil
				downloadCancelState.stopping = false
			}
		})
	}
}

func ActiveDownloadContext() context.Context {
	downloadCancelState.Lock()
	defer downloadCancelState.Unlock()

	if downloadCancelState.ctx == nil {
		return context.Background()
	}
	return downloadCancelState.ctx
}

func ForceStopActiveDownloads() {
	downloadCancelState.Lock()
	cancel := downloadCancelState.cancel
	if cancel != nil {
		downloadCancelState.stopping = true
	}
	downloadCancelState.Unlock()

	if cancel != nil {
		cancel()
	}

	CancelQueuedAndDownloadingItems()
	SetDownloading(false)
}

func IsDownloadForceStopRequested() bool {
	downloadCancelState.Lock()
	defer downloadCancelState.Unlock()

	return downloadCancelState.stopping
}

func CheckDownloadCancelled() error {
	ctx := ActiveDownloadContext()
	select {
	case <-ctx.Done():
		return ErrDownloadCancelled
	default:
		return nil
	}
}

func SleepWithDownloadContext(delay time.Duration) error {
	if delay <= 0 {
		return CheckDownloadCancelled()
	}

	ctx := ActiveDownloadContext()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ErrDownloadCancelled
	case <-timer.C:
		return nil
	}
}

func IsDownloadCancelledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrDownloadCancelled) || errors.Is(err, context.Canceled)
}

func WrapDownloadCancelled(err error) error {
	if err == nil {
		return nil
	}
	if IsDownloadForceStopRequested() || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w", ErrDownloadCancelled)
	}
	return err
}
