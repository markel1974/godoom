package collisions

import (
	"runtime"
	"sync"
	"sync/atomic"
)

const maxPoolHelpers = 7

type parallelWorkerPool struct {
	mu     sync.Mutex
	init   sync.Once
	notify []chan struct{}
	wg     sync.WaitGroup
	n      int64
	chunk  int64
	cursor atomic.Int64
	fn     func(workerID, start, end int)
}

var sharedWorkerPool parallelWorkerPool

func (p *parallelWorkerPool) ensureInit() {
	p.init.Do(func() {
		p.notify = make([]chan struct{}, maxPoolHelpers)
		for w := range maxPoolHelpers {
			ch := make(chan struct{}, 1)
			p.notify[w] = ch
			workerID := w + 1
			go p.workerLoop(workerID, ch)
		}
	})
}

func (p *parallelWorkerPool) workerLoop(workerID int, ch <-chan struct{}) {
	for range ch {
		p.runChunks(workerID)
		p.wg.Done()
	}
}

func (p *parallelWorkerPool) runChunks(workerID int) {
	n := p.n
	chunk := p.chunk
	fn := p.fn
	for {
		end := p.cursor.Add(chunk)
		start := end - chunk
		if start >= n {
			return
		}
		if end > n {
			end = n
		}
		fn(workerID, int(start), int(end))
	}
}

// WorkerCount returns the maximum worker slots (caller plus persistent helper workers).
func (p *parallelWorkerPool) WorkerCount() int {
	return min(maxPoolHelpers+1, max(1, runtime.GOMAXPROCS(0)))
}

// Dispatch executes fn(workerID, start, end) across contiguous chunked subranges of [0, n)
// using persistent helper goroutines with zero per-call heap allocations or goroutine spawns.
func (p *parallelWorkerPool) Dispatch(n, minPerWorker, chunk int, fn func(workerID, start, end int)) {
	if n <= 0 {
		return
	}
	procs := runtime.GOMAXPROCS(0)
	if procs <= 1 || n < minPerWorker*2 || !p.mu.TryLock() {
		fn(0, 0, n)
		return
	}
	p.ensureInit()
	helpersAvail := min(len(p.notify), procs-1)
	wantHelpers := (n+minPerWorker-1)/minPerWorker - 1
	activeHelpers := min(helpersAvail, max(0, wantHelpers))
	if activeHelpers == 0 {
		p.mu.Unlock()
		fn(0, 0, n)
		return
	}
	p.n = int64(n)
	p.chunk = int64(max(1, chunk))
	p.fn = fn
	p.cursor.Store(0)
	p.wg.Add(activeHelpers)
	for w := range activeHelpers {
		p.notify[w] <- struct{}{}
	}
	p.runChunks(0)
	p.wg.Wait()
	p.fn = nil
	p.mu.Unlock()
}
