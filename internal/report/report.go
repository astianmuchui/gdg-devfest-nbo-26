package report

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	Received atomic.Uint64
	Stored   atomic.Uint64
	Failed   atomic.Uint64
	InFlight atomic.Int64
	Dropped  atomic.Uint64
	lastErr  atomic.Value

	QueueDepth func() int
)

func Fail(err error) {
	Failed.Add(1)
	lastErr.Store(err.Error())
}

func FailN(n int, err error) {
	Failed.Add(uint64(n))
	lastErr.Store(err.Error())
}

func Run(ctx context.Context, stalling func() bool) {
	start := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastRecv, lastStored, lastFailed uint64
	var m runtime.MemStats

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		recv, stored, failed := Received.Load(), Stored.Load(), Failed.Load()
		runtime.ReadMemStats(&m)

		q := 0
		if QueueDepth != nil {
			q = QueueDepth()
		}

		stall := ""
		if stalling != nil && stalling() {
			stall = "  STALL"
		}

		fmt.Printf("t=%4ds  recv=%-9d recv/s=%-7d stored=%-9d stored/s=%-7d failed=%-7d dropped=%-8d queued=%-7d inflight=%-8d goroutines=%-8d heap=%7.1fMB sys=%7.1fMB%s\n",
			int(time.Since(start).Seconds()),
			recv, recv-lastRecv,
			stored, stored-lastStored,
			failed,
			Dropped.Load(),
			q,
			InFlight.Load(),
			runtime.NumGoroutine(),
			float64(m.HeapAlloc)/(1<<20),
			float64(m.Sys)/(1<<20),
			stall,
		)

		if failed > lastFailed {
			if e, ok := lastErr.Load().(string); ok {
				fmt.Printf("        last error: %s\n", e)
			}
		}

		lastRecv, lastStored, lastFailed = recv, stored, failed
	}
}
