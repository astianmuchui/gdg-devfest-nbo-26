package report

import (
	"runtime/metrics"
	"syscall"
	"time"
)

const (
	mAllocObjects = "/gc/heap/allocs:objects"
	mAllocBytes   = "/gc/heap/allocs:bytes"
	mGCCycles     = "/gc/cycles/total:gc-cycles"
	mGCCPU        = "/cpu/classes/gc/total:cpu-seconds"
)

type goSample struct {
	allocObjects uint64
	allocBytes   uint64
	gcCycles     uint64
	gcCPU        float64
	procCPU      time.Duration
}

type goMetrics struct {
	samples []metrics.Sample
	last    goSample
}

func newGoMetrics() *goMetrics {
	g := &goMetrics{samples: []metrics.Sample{
		{Name: mAllocObjects},
		{Name: mAllocBytes},
		{Name: mGCCycles},
		{Name: mGCCPU},
	}}
	g.last = g.read()
	return g
}

func processCPU() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

func (g *goMetrics) read() goSample {
	metrics.Read(g.samples)
	return goSample{
		allocObjects: g.samples[0].Value.Uint64(),
		allocBytes:   g.samples[1].Value.Uint64(),
		gcCycles:     g.samples[2].Value.Uint64(),
		gcCPU:        g.samples[3].Value.Float64(),
		procCPU:      processCPU(),
	}
}

func (g *goMetrics) line(storedDelta uint64) string {
	cur := g.read()
	prev := g.last
	g.last = cur

	cpu := cur.procCPU - prev.procCPU
	cpuMsPer1k, allocsPerMsg := 0.0, 0.0
	if storedDelta > 0 {
		cpuMsPer1k = float64(cpu.Milliseconds()) * 1000 / float64(storedDelta)
		allocsPerMsg = float64(cur.allocObjects-prev.allocObjects) / float64(storedDelta)
	}
	gcCPUPct := 0.0
	if cpu > 0 {
		gcCPUPct = (cur.gcCPU - prev.gcCPU) / cpu.Seconds() * 100
	}

	return fmtGo(cpuMsPer1k, allocsPerMsg,
		float64(cur.allocBytes-prev.allocBytes)/(1<<20),
		cur.gcCycles-prev.gcCycles, gcCPUPct)
}
