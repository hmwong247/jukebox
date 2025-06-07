package taskq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	PoolID = autoIncID{id: -1}
)

type TaskStatus struct {
	TaskID int64
	Status string
}

type autoIncID struct {
	sync.Mutex
	id int
}

func (a *autoIncID) ID() int {
	a.Lock()
	defer a.Unlock()
	a.id++
	return a.id
}

type WorkerPool struct {
	ID            int
	snowflakeNode *snowflake.Node
	workers       []*Worker
	workerq       chan chan Task
	taskq         chan Task
}

func NewWorkerPool(workernum int, qbuffer int) (*WorkerPool, error) {
	if workernum <= 0 {
		return &WorkerPool{}, fmt.Errorf("number of workers should be non-zero +ve number, given: %v", workernum)
	}
	if qbuffer <= 0 {
		return &WorkerPool{}, fmt.Errorf("number of buffers should be non-zero +ve number, given: %v", qbuffer)
	}
	// workerNum := min(workernum, MAX_CONCURRENT_WORKER_PER_POOL)
	// qBuffer := min(qbuffer-1, MAX_TASK_QUEUE_SIZE)
	id := PoolID.ID()
	node, err := snowflake.NewNode(int64(id))
	if err != nil {
		errStr := fmt.Sprintf("failed to create snowflake node, err: %v", err)
		newErr := errors.New(errStr)
		return &WorkerPool{}, newErr
	}

	return &WorkerPool{
		ID:            id,
		snowflakeNode: node,
		workers:       make([]*Worker, 0, workernum),
		workerq:       make(chan chan Task),
		taskq:         make(chan Task, qbuffer),
	}, nil
}

func (wp *WorkerPool) Run(ctx context.Context) {
	if name, ok := ctx.Value("name").(string); ok {
		slog.Debug("worker pool running", "name", name, "id", wp.ID)
	} else {
		slog.Debug("worker pool ctx err", "name", name, "id", wp.ID)
		return
	}
	// create workers
	for i := 0; i < cap(wp.workers); i++ {
		w := newWorker(i, wp.workerq)
		wp.workers = append(wp.workers, w)
		go w.Start(ctx)
		slog.Debug("created worker", "id", i)
	}
	for {
		select {
		case <-ctx.Done():
			if name, ok := ctx.Value("name").(string); ok {
				slog.Info("worker pool ctx cancelled", "ctx", ctx.Err(), "name", name)
			}
			return
		case task := <-wp.taskq:
			// recieved a task
			// wait for an availble worker and dispatch
			workerTaskq := <-wp.workerq
			workerTaskq <- task
		}
	}
}

func (wp *WorkerPool) Submit(ctx context.Context, t Task) (int, int64) {
	select {
	case <-ctx.Done():
		slog.Info("submit cancelled", "ctx", ctx.Err())
		return http.StatusRequestTimeout, -1
	case wp.taskq <- t:
		// signal to the caller
		// t.Accepted(ctx)
		// slog.Debug("submit ok")
		taskID := wp.snowflakeNode.Generate().Int64()
		return http.StatusAccepted, taskID
	default:
		// t.Rejected(ctx)
		// slog.Debug("submit err task queue is full", "task", t)
		return http.StatusTooManyRequests, -1
	}
}
