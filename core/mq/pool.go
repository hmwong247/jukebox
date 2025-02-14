package mq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

const (
	MAX_CONCURRENT_WORKER_PER_POOL = 2
	MAX_TASK_QUEUE_SIZE            = 16
)

var (
// PoolID = autoIncID{id: -1}
)

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
	ID      autoIncID
	workers []*Worker
	workerq chan chan Task
	taskq   chan Task
}

func NewWorkerPool(workernum uint, qbuffer int) (*WorkerPool, error) {
	if workernum <= 0 {
		return &WorkerPool{}, fmt.Errorf("number of workers should be non-zero +ve number, given: %v", workernum)
	}
	if qbuffer <= 0 {
		return &WorkerPool{}, fmt.Errorf("number of buffers should be non-zero +ve number, given: %v", qbuffer)
	}
	workerNum := min(workernum, MAX_CONCURRENT_WORKER_PER_POOL)
	qBuffer := min(qbuffer-1, MAX_TASK_QUEUE_SIZE)

	return &WorkerPool{
		ID:      autoIncID{id: -1},
		workers: make([]*Worker, 0, workerNum),
		workerq: make(chan chan Task),
		taskq:   make(chan Task, qBuffer),
	}, nil
}

func (wp *WorkerPool) Run(ctx context.Context) {
	if name, ok := ctx.Value("name").(string); ok {
		slog.Debug("worker pool running", "name", name)
	} else {
		slog.Debug("worker pool ctx err", "name", name)
		return
	}
	// create workers
	for i := 0; i < cap(wp.workers); i++ {
		w := newWorker(i, wp.workerq)
		wp.workers = append(wp.workers, w)
		go w.Start(ctx)
		slog.Debug("create workers", "cap", cap(wp.workers))
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

func (wp *WorkerPool) Submit(ctx context.Context, t Task) {
	select {
	case <-ctx.Done():
		slog.Info("submit cancelled", "ctx", ctx.Err())
		return
	case wp.taskq <- t:
		// signal to the caller
		t.Accepted(ctx)
		// slog.Debug("submit ok")
	default:
		t.Rejected(ctx)
		// slog.Debug("submit err task queue is full", "task", t)
	}
}
