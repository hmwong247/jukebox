package ytdlp

import (
	"context"
	"fmt"
	"log/slog"
	"main/core/mq"
)

var (
	JsonDownloader, AudioDownloader *mq.WorkerPool
)

type RequestJson struct {
	URL      string
	Response chan any
}

func (r *RequestJson) Accepted(ctx context.Context) {
	// if ctx.Err() != nil {
	// 	r.Response <- ctx.Err()
	// 	slog.Debug("ctx err", "err", ctx.Err())
	// 	return
	// }

	r.Response <- 202
}

func (r *RequestJson) Rejected(ctx context.Context) {
	// if ctx.Err() != nil {
	// 	r.Response <- ctx.Err()
	// 	slog.Debug("ctx err", "err", ctx.Err())
	// 	return
	// }

	r.Response <- 429
}

func (r *RequestJson) Process() {
	json, err := DownloadInfoJson(r.URL)
	if err != nil {
		slog.Error("requestjson err", "err", err)
		r.Response <- err
		return
	}

	slog.Debug("requestjson response")
	r.Response <- json
}

func (r *RequestJson) String() string {
	return fmt.Sprintf("")
}

func init() {
	// JsonDownloader, _ = mq.NewWorkerPool(mq.MAX_CONCURRENT_WORKER_PER_POOL, mq.MAX_TASK_QUEUE_SIZE)
	// AudioDownloader, _ = mq.NewWorkerPool(mq.MAX_CONCURRENT_WORKER_PER_POOL, mq.MAX_TASK_QUEUE_SIZE)

	// use less resources for testing
	JsonDownloader, _ = mq.NewWorkerPool(2, 2)
	AudioDownloader, _ = mq.NewWorkerPool(2, 2)
}
