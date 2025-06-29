package weaksync

import (
	"sync"
	"sync/atomic"
)

type WaitGroupCnt struct {
	wg  *sync.WaitGroup
	cnt int64
}

func CreateWaitGroupCnt() *WaitGroupCnt {
	return &WaitGroupCnt{
		wg:  &sync.WaitGroup{},
		cnt: 0,
	}
}

func (wgcnt WaitGroupCnt) Add(delta int) {
	atomic.AddInt64(&wgcnt.cnt, 1)
	wgcnt.wg.Add(delta)
}

func (wgcnt WaitGroupCnt) Done() {
	atomic.AddInt64(&wgcnt.cnt, -1)
	wgcnt.wg.Done()
}

func (wgcnt WaitGroupCnt) Wait() {
	wgcnt.wg.Wait()
}

func (wgcnt WaitGroupCnt) Count() int {
	return int(atomic.LoadInt64(&wgcnt.cnt))
}
