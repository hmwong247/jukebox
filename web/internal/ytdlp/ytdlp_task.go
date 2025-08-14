package ytdlp

import (
	"context"
	"fmt"
	"main/internal/taskq"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	TIMEOUT_JSON  = 1 * time.Minute
	TIMEOUT_AUDIO = 3 * time.Minute
)

var (
	JsonDownloader, AudioDownloader *taskq.WorkerPool

	MAX_CONCURRENT_WORKER_PER_POOL = func() int {
		envar := os.Getenv("MAX_CONCURRENT_WORKER_PER_POOL")
		var ret int
		if envar == "" {
			log.Warn().Msg("Conncurrent download is not enabled")
			ret = 1
		} else {
			_ret, err := strconv.Atoi(envar)
			if err != nil {
				log.Error().Err(err).Msg("Invalid env: MAX_CONCURRENT_WORKER_PER_POOL")
				ret = 1
			}
			ret = _ret
		}
		return ret
	}()

	MAX_TASK_QUEUE_SIZE = func() int {
		envar := os.Getenv("MAX_TASK_QUEUE_SIZE")
		var ret int
		if envar == "" {
			log.Warn().Msg("Download queue is not configured, default to 1")
			ret = 1
		} else {
			_ret, err := strconv.Atoi(envar)
			if err != nil {
				log.Error().Err(err).Msg("Invalid env: MAX_TASK_QUEUE_SIZE")
				ret = 1
			}
			ret = _ret
		}
		return ret
	}()
)

type RequestInfojson struct {
	Ctx      context.Context
	ErrCh    chan error
	FinCh    chan struct{}
	URL      string
	Response InfoJson
}

func (r *RequestInfojson) Process(workerctx context.Context) {
	select {
	case <-workerctx.Done():
		r.ErrCh <- workerctx.Err()
		return
	case <-r.Ctx.Done():
		r.ErrCh <- r.Ctx.Err()
		return
	default:
		json, err := DownloadInfoJson(r.Ctx, r.URL)
		if err != nil {
			log.Info().Err(err).Msg("[task] failed to fetch infojson")
			select {
			case <-r.Ctx.Done():
				// check closed channel, ctx already timeout on the top-level
				log.Debug().Msg("[task] ctx already timeout on the top-level")
				return
			default:
			}
			r.ErrCh <- err
			return
		}

		if workerctx.Err() == nil && r.Ctx.Err() == nil {
			r.Response = json
			r.FinCh <- struct{}{}
		}
	}
}

func (r *RequestInfojson) String() string {
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
		audioBytes, err := DownloadAudio(r.Ctx, r.URL)
		if err != nil {
			log.Error().Err(err).Msg("[task] failed to fetch audio")
			select {
			case <-r.Ctx.Done():
				// check closed channel, ctx already timeout on the top-level
				log.Debug().Msg("[task] ctx already timeout on the top-level")
				return
			default:
			}
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
	JsonDownloader, _ = taskq.NewWorkerPool(MAX_CONCURRENT_WORKER_PER_POOL, MAX_TASK_QUEUE_SIZE)
	AudioDownloader, _ = taskq.NewWorkerPool(MAX_CONCURRENT_WORKER_PER_POOL, MAX_TASK_QUEUE_SIZE)

	// use less resources for testing
	// JsonDownloader, _ = taskq.NewWorkerPool(2, 2)
	// AudioDownloader, _ = taskq.NewWorkerPool(2, 2)
}
