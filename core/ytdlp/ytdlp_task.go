package ytdlp

import (
	"context"
	"fmt"
	"log/slog"
	"main/core/mq"
	"net/http"
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
	URL      string
	Response chan any
}

func (r *RequestJson) Accepted(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	default:
		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- http.StatusAccepted
		}
	}
}

func (r *RequestJson) Rejected(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	default:
		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- http.StatusTooManyRequests
		}
	}
}

func (r *RequestJson) Process(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	case <-r.Ctx.Done():
		r.Response <- r.Ctx.Err()
		return
	default:
		json, err := DownloadInfoJson(r.URL)
		if err != nil {
			slog.Info("[task] failed to fetch infojson", "err", err)
			r.Response <- err
			return
		}

		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- json
		}
	}
}

func (r *RequestJson) String() string {
	return fmt.Sprintf("request: json, url: %v", r.URL)
}

type RequestAudio struct {
	Ctx      context.Context
	URL      string
	Response chan any
}

func (r *RequestAudio) Accepted(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	default:
		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- true
		}
	}
}

func (r *RequestAudio) Rejected(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	default:
		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- false
		}
	}
}

func (r *RequestAudio) Process(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.Response <- workerctx.Err()
		return
	case <-r.Ctx.Done():
		r.Response <- r.Ctx.Err()
		return
	default:
		audioBytes, err := DownloadAudio(r.URL)
		if err != nil {
			slog.Info("[task] failed to fetch infojson", "err", err)
			r.Response <- err
			return
		}

		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response <- audioBytes
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
