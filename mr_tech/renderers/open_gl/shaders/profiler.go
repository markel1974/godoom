package shaders

import (
	"fmt"
	"sort"
	"time"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// gpuTimer represents a timer for GPU queries, managing elapsed time measurement and query state tracking.
type gpuTimer struct {
	id      uint32
	pending bool
	totalNS uint64
	samples uint64
}

// cpuTimer represents a structure for tracking CPU timing data, including total elapsed time and the number of invocations.
type cpuTimer struct {
	total time.Duration
	count uint64
}

// GpuProfiler is a tool for profiling GPU and CPU performance across multiple frames in a graphics application.
type GpuProfiler struct {
	ctx      api.IContext
	gpu      map[string]*gpuTimer
	cpu      map[string]*cpuTimer
	frame    uint64
	reportAt uint64
	start    time.Time
}

// NewGpuProfiler creates and returns a new instance of GpuProfiler configured with the provided IContext.
func NewGpuProfiler(ctx api.IContext, fps uint64) *GpuProfiler {
	return &GpuProfiler{
		ctx:      ctx,
		gpu:      make(map[string]*gpuTimer),
		cpu:      make(map[string]*cpuTimer),
		reportAt: fps,
		start:    time.Now(),
	}
}

// MeasureGPU measures the GPU execution time for the given named section by wrapping the provided function `fn`.
func (p *GpuProfiler) MeasureGPU(name string, fn func()) {
	t, exists := p.gpu[name]
	if !exists {
		t = &gpuTimer{}
		p.ctx.GenQueries(1, &t.id)
		p.gpu[name] = t
	}

	canMeasure := true

	if t.pending {
		var available int32
		p.ctx.GetQueryObjectiv(t.id, api.QUERY_RESULT_AVAILABLE, &available)
		if available == 0 {
			canMeasure = false
		} else {
			var ns uint64
			p.ctx.GetQueryObjectui64v(t.id, api.QUERY_RESULT, &ns)
			t.totalNS += ns
			t.samples++
			t.pending = false
		}
	}

	if !canMeasure {
		fn()
		return
	}

	p.ctx.BeginQuery(api.TIME_ELAPSED, t.id)
	fn()
	p.ctx.EndQuery(api.TIME_ELAPSED)

	t.pending = true
}

// MeasureCPU measures the execution time of a function on the CPU and tracks the total time and invocation count by name.
func (p *GpuProfiler) MeasureCPU(name string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start)

	t := p.cpu[name]
	if t == nil {
		t = &cpuTimer{}
		p.cpu[name] = t
	}

	t.total += elapsed
	t.count++
}

// EndFrame processes profiling data for GPU and CPU, prints results if needed, and resets collected statistics.
func (p *GpuProfiler) EndFrame() bool {
	p.frame++

	if p.frame%p.reportAt != 0 {
		return false
	}

	now := time.Now()
	fmt.Printf("\n===== PROFILER frame=%d (elapsed=%s) =====\n", p.frame, now.Sub(p.start))

	gpuNames := make([]string, 0, len(p.gpu))
	for name := range p.gpu {
		gpuNames = append(gpuNames, name)
	}
	sort.Strings(gpuNames)

	fmt.Println("-- GPU --")
	for _, name := range gpuNames {
		t := p.gpu[name]
		if t.samples == 0 {
			fmt.Printf("%-24s n/a\n", name)
			continue
		}

		avgMS := float64(t.totalNS) /
			float64(t.samples) / 1e6

		fmt.Printf(
			"%-24s %8.3f ms (%d samples)\n",
			name, avgMS, t.samples,
		)

		t.totalNS = 0
		t.samples = 0
	}

	cpuNames := make([]string, 0, len(p.cpu))
	for name := range p.cpu {
		cpuNames = append(cpuNames, name)
	}
	sort.Strings(cpuNames)

	fmt.Println("-- CPU --")
	for _, name := range cpuNames {
		t := p.cpu[name]
		if t.count == 0 {
			continue
		}

		avgMS := float64(t.total) /
			float64(t.count) / float64(time.Millisecond)

		fmt.Printf(
			"%-24s %8.3f ms (%d samples)\n",
			name, avgMS, t.count,
		)

		t.total = 0
		t.count = 0
	}

	fmt.Println("============================")
	p.frame = 0
	p.start = time.Now()

	return true
}

// Close releases all GPU queries associated with the profiler, cleaning up resources allocated during profiling activities.
func (p *GpuProfiler) Close() {
	for _, t := range p.gpu {
		if t.id != 0 {
			p.ctx.DeleteQueries(1, &t.id)
		}
	}
}
