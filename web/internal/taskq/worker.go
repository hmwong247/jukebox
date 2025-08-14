package taskq

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
)

type Task interface {
	Process(context.Context)

	// std interface
	fmt.Stringer
}

type Worker struct {
	ID    int
	taskq chan Task
	poolq chan chan Task
}

func newWorker(id int, workerq chan chan Task) *Worker {
	return &Worker{
		ID:    id,
		taskq: make(chan Task),
		poolq: workerq,
	}
}

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("id", w.ID).Msg("worker ctx cancelled")
			return
		case w.poolq <- w.taskq:
			// join the pool when available
			// wait for task
			task := <-w.taskq
			task.Process(ctx)
		}
	}
}
