package executor

import (
	"errors"
	"runtime"
	"sync"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// CallQueueCap defines the capacity of the queue used to store function calls in the main thread's call queue.
const CallQueueCap = 16

// Thread is a global pointer to the main thread, used for executing thread-safe OpenGL and GLFW operations.
var Thread *MainThread

// init initializes the main thread and locks the current OS thread for proper thread management.
func init() {
	runtime.LockOSThread()
	Thread = NewMainThread()
}

// MainThread provides a mechanism for serializing function execution on a single thread.
// It ensures thread-safe operations using a call queue and synchronization primitives.
// Functions can be posted or called with optional return values or errors.
type MainThread struct {
	callQueue chan func()
	respMutex sync.Mutex
	respChan  chan interface{}
}

// NewMainThread creates and returns a new instance of MainThread with initialized callQueue and respChan channels.
func NewMainThread() *MainThread {
	return &MainThread{
		callQueue: make(chan func(), CallQueueCap),
		respChan:  make(chan interface{}),
	}
}

// Init initializes the essential OpenGL state, enabling blending, multisampling, and optionally scissor testing.
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
	gl.Enable(gl.MULTISAMPLE)
}

// Run initializes the GLFW library, executes the provided function within the main thread, and terminates the GLFW context.
func (m *MainThread) Run(run func()) {
	err := glfw.Init()
	if err != nil {
		panic(errors.New("failed to initialize glfw"))
	}
	m.doRun(run)
	glfw.Terminate()
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
		m.respChan <- true
	}
	<-m.respChan
	m.respMutex.Unlock()
}

// CallErr executes the provided function on the main thread and returns any error it produces.
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

// CallVal executes the provided function on the main thread and returns its result through a synchronized call.
func (m *MainThread) CallVal(f func() interface{}) interface{} {
	m.respMutex.Lock()
	m.callQueue <- func() {
		m.respChan <- f()
	}
	val := <-m.respChan
	m.respMutex.Unlock()
	return val
}

// doRun executes the provided function on a separate goroutine and processes queued functions on the main thread.
// It blocks until the provided function completes execution and cleans up resources before returning.
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
