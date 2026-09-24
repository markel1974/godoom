package executor

import (
	"errors"
	"runtime"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// CallQueueCap defines the capacity of the channel used for managing function calls in a MainThread instance.
const CallQueueCap = 16

// Thread is a pointer to the main thread's execution context, facilitating thread-safe operations using a queue-based model.
var Thread *MainThread

// init initializes the main thread by locking the OS thread and creating a new instance of MainThread.
func init() {
	runtime.LockOSThread()
	Thread = NewMainThread()
}

// MainThread is a type that manages function calls serialized onto a single thread, typically for thread-safe operations.
type MainThread struct {
	callQueue chan func()
	respMutex sync.Mutex
	respChan  chan interface{}
}

// NewMainThread creates and returns a new instance of MainThread with initialized call queue and response channel.
func NewMainThread() *MainThread {
	return &MainThread{
		callQueue: make(chan func(), CallQueueCap),
		respChan:  make(chan interface{}),
	}
}

// Init initializes the OpenGL context and sets up blending modes and scissor test based on the provided configuration.
func (m *MainThread) Init(disableScissorTest bool) {
	err := gl.Init()
	if err != nil {
		panic(err)
	}
	gl.Enable(gl.BLEND)
	if !disableScissorTest {
		gl.Enable(gl.SCISSOR_TEST)
	}
	gl.BlendEquation(gl.FUNC_ADD)
}

func (m *MainThread) Run(run func()) {
	err := glfw.Init()
	if err != nil {
		panic(errors.New("failed to initialize glfw"))
	}
	defer glfw.Terminate()
	m.doRun(run)
}

// Post schedules the provided function to be executed on the main thread by adding it to the call queue.
func (m *MainThread) Post(f func()) {
	m.callQueue <- f
}

// Call schedules a function to run on the main thread and waits for its execution to complete.
func (m *MainThread) Call(f func()) {
	m.respMutex.Lock()
	m.callQueue <- func() {
		f()
		m.respChan <- true
	}
	<-m.respChan
	m.respMutex.Unlock()
}

// CallErr schedules a function that returns an error for execution on the main thread and returns the resulting error.
func (m *MainThread) CallErr(f func() error) error {
	m.respMutex.Lock()
	m.callQueue <- func() {
		m.respChan <- f()
	}
	resp := <-m.respChan
	m.respMutex.Unlock()
	if resp == nil {
		return nil
	}
	if err, ok := resp.(error); ok {
		return err
	}
	return errors.New("invalid response")
}

// CallVal schedules a function to be executed on the main thread and returns its result.
func (m *MainThread) CallVal(f func() interface{}) interface{} {
	m.respMutex.Lock()
	m.callQueue <- func() {
		m.respChan <- f()
	}
	val := <-m.respChan
	m.respMutex.Unlock()
	return val
}

// Run executes the provided function in a separate goroutine while managing a call queue for synchronized execution.
func (m *MainThread) doRun(run func()) {
	done := make(chan bool)
	go func() {
		run()
		done <- true
	}()

	for {
		select {
		case f := <-m.callQueue:
			f()
		case <-done:
			return
		}
	}
}
