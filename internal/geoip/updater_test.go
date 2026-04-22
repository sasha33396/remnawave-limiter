package geoip

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type recordingDownloader struct {
	mu           sync.Mutex
	calls        int32
	errOnNthCall int32
	errValue     error
}

func (r *recordingDownloader) Download(ctx context.Context, dst string) error {
	n := atomic.AddInt32(&r.calls, 1)
	if r.errOnNthCall > 0 && n == r.errOnNthCall {
		return r.errValue
	}
	return nil
}

type recordingReloader struct {
	mu    sync.Mutex
	calls int32
	err   error
}

func (r *recordingReloader) Reload(path string) error {
	atomic.AddInt32(&r.calls, 1)
	return r.err
}

func TestUpdater_RunsOnTickThenStopsOnContextCancel(t *testing.T) {
	d := &recordingDownloader{}
	rel := &recordingReloader{}
	u := &Updater{
		Downloader: d,
		Reloader:   rel,
		DstPath:    "/tmp/fake.mmdb",
		Interval:   30 * time.Millisecond,
		Logger:     logrus.New(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		u.Run(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Updater.Run did not return after context cancel")
	}

	if atomic.LoadInt32(&d.calls) == 0 {
		t.Error("Downloader.Download was not called")
	}
	if atomic.LoadInt32(&rel.calls) != atomic.LoadInt32(&d.calls) {
		t.Errorf("Reloader.Reload called %d times, Download called %d times; want equal",
			rel.calls, d.calls)
	}
}

func TestUpdater_DownloadErrorDoesNotStopLoop(t *testing.T) {
	d := &recordingDownloader{errOnNthCall: 1, errValue: errors.New("network down")}
	rel := &recordingReloader{}
	u := &Updater{
		Downloader: d,
		Reloader:   rel,
		DstPath:    "/tmp/fake.mmdb",
		Interval:   30 * time.Millisecond,
		Logger:     logrus.New(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go u.Run(ctx)

	time.Sleep(120 * time.Millisecond)
	cancel()

	calls := atomic.LoadInt32(&d.calls)
	if calls < 2 {
		t.Errorf("Download was called %d times; expected >=2 even after an error", calls)
	}
	if atomic.LoadInt32(&rel.calls) == calls {
		t.Errorf("Reloader was called for a failed download; want Reload < Download when errors occur")
	}
}
