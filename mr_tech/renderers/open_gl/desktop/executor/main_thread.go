package executor

import (
	"runtime"
	"sync"
)

// CallQueueCap defines the capacity of the queue used to store function calls in the main thread's call queue.
const CallQueueCap = 16

// MainThread provides a mechanism for serializing function execution on a single thread.
// It ensures thread-safe operations using a call queue and synchronization primitives.
// Functions can be posted or called with optional return values or errors.
type MainThread struct {
	callQueue         chan func()
	respMutex         sync.Mutex
	respChanInterface chan interface{}
	respChanError     chan error
	respChanRes       chan bool
}

// NewMainThread creates and returns a new instance of MainThread with initialized callQueue and respChan channels.
func NewMainThread() *MainThread {
	runtime.LockOSThread()
	th := &MainThread{
		callQueue:         make(chan func(), CallQueueCap),
		respChanInterface: make(chan interface{}),
		respChanError:     make(chan error),
		respChanRes:       make(chan bool),
	}
	return th
}

// Start begins the main execution loop for processing queued functions in the call queue. It blocks until termination.w.doRun
func (m *MainThread) Start(done chan bool) {
	for {
		select {
		case f := <-m.callQueue:
			f()
		case <-done:
			return
		}
	}
}

// Post queues the provided function to be executed on the main thread.
func (m *MainThread) Post(f func()) {
	m.callQueue <- f
}

// Call executes the provided function on the main thread and blocks until the function completes.
func (m *MainThread) Call(f func()) {
	m.respMutex.Lock()
	m.callQueue <- func() {
		f()
		m.respChanRes <- true
	}
	<-m.respChanRes
	m.respMutex.Unlock()
}

// CallErr executes the provided function on the main thread and returns any error it produces.
func (m *MainThread) CallErr(f func() error) error {
	m.respMutex.Lock()
	m.callQueue <- func() {
		m.respChanError <- f()
	}
	err := <-m.respChanError
	m.respMutex.Unlock()
	return err
}

// CallVal executes the provided function on the main thread and returns its result through a synchronized call.
func (m *MainThread) CallVal(f func() interface{}) interface{} {
	m.respMutex.Lock()
	m.callQueue <- func() {
		m.respChanInterface <- f()
	}
	val := <-m.respChanInterface
	m.respMutex.Unlock()
	return val
}
