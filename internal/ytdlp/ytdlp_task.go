package ytdlp

import (
	"context"
	"fmt"
	"log/slog"
	"main/internal/mq"
	"time"
)

const (
	TIMEOUT_JSON  = 30 * time.Second
	TIMEOUT_AUDIO = 1 * time.Minute
)

var (
	JsonDownloader, AudioDownloader *mq.WorkerPool
)

type RequestJson struct {
	Ctx      context.Context
	ErrCh    chan error
	FinCh    chan struct{}
	URL      string
	Response InfoJson
}

func (r *RequestJson) Process(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.ErrCh <- workerctx.Err()
		return
	case <-r.Ctx.Done():
		r.ErrCh <- r.Ctx.Err()
		return
	default:
		json, err := DownloadInfoJson(r.URL)
		if err != nil {
			slog.Info("[task] failed to fetch infojson", "err", err)
			r.ErrCh <- err
			return
		}

		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response = json
			r.FinCh <- struct{}{}
		}
	}
}

func (r *RequestJson) String() string {
	return fmt.Sprintf("request: json, url: %v", r.URL)
}

type RequestAudio struct {
	Ctx      context.Context
	ErrCh    chan error
	FinCh    chan struct{}
	URL      string
	Response []byte
}

func (r *RequestAudio) Process(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.ErrCh <- workerctx.Err()
		return
	case <-r.Ctx.Done():
		r.ErrCh <- r.Ctx.Err()
		return
	default:
		audioBytes, err := DownloadAudio(r.URL)
		if err != nil {
			slog.Info("[task] failed to fetch infojson", "err", err)
			r.ErrCh <- err
			return
		}

		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response = audioBytes
			r.FinCh <- struct{}{}
		}
	}
}

func (r *RequestAudio) String() string {
	return fmt.Sprintf("request: audio, url: %v", r.URL)
}

func init() {
	// JsonDownloader, _ = mq.NewWorkerPool(mq.MAX_CONCURRENT_WORKER_PER_POOL, mq.MAX_TASK_QUEUE_SIZE)
	// AudioDownloader, _ = mq.NewWorkerPool(mq.MAX_CONCURRENT_WORKER_PER_POOL, mq.MAX_TASK_QUEUE_SIZE)

	// use less resources for testing
	JsonDownloader, _ = mq.NewWorkerPool(2, 2)
	AudioDownloader, _ = mq.NewWorkerPool(2, 2)
}
