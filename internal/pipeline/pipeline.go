package pipeline

import (
	"context"
	"hash/maphash"
	"sync"
	"time"

	"github.com/astianmuchui/supersimpleingestor/internal/report"
	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
	"github.com/astianmuchui/supersimpleingestor/internal/store"
)

var seed = maphash.MakeSeed()

type Pipeline struct {
	shards  []chan scylla.TelemetryRecord
	st      store.Store
	maxRows int
	every   time.Duration
	conc    int
}

func New(st store.Store, shards, depth, maxRows, conc int, every time.Duration) *Pipeline {
	p := &Pipeline{st: st, maxRows: maxRows, every: every, conc: conc}
	p.shards = make([]chan scylla.TelemetryRecord, shards)
	for i := range p.shards {
		p.shards[i] = make(chan scylla.TelemetryRecord, depth)
	}
	return p
}

func (p *Pipeline) Start(ctx context.Context, wg *sync.WaitGroup) {
	for _, q := range p.shards {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(ctx, q)
		}()
	}
}

func (p *Pipeline) Submit(rec scylla.TelemetryRecord) bool {
	h := maphash.String(seed, rec.DeviceID)
	q := p.shards[h%uint64(len(p.shards))]
	select {
	case q <- rec:
		return true
	default:
		report.Dropped.Add(1)
		return false
	}
}

func (p *Pipeline) QueueDepth() int {
	n := 0
	for _, q := range p.shards {
		n += len(q)
	}
	return n
}

func (p *Pipeline) worker(ctx context.Context, in <-chan scylla.TelemetryRecord) {
	buf := make([]scylla.TelemetryRecord, 0, p.maxRows)
	t := time.NewTicker(p.every)
	defer t.Stop()
	for {
		select {
		case r, ok := <-in:
			if !ok {
				p.flush(ctx, buf)
				return
			}
			buf = append(buf, r)
			if len(buf) < p.maxRows {
				continue
			}
		case <-t.C:
			if len(buf) == 0 {
				continue
			}
		}
		p.flush(ctx, buf)
		buf = buf[:0]
	}
}

func (p *Pipeline) flush(ctx context.Context, rows []scylla.TelemetryRecord) {
	groups := make(map[string][]scylla.TelemetryRecord)
	for _, r := range rows {
		groups[r.DeviceID] = append(groups[r.DeviceID], r)
	}
	sem := make(chan struct{}, p.conc)
	var wg sync.WaitGroup
	for _, g := range groups {
		sem <- struct{}{}
		wg.Add(1)
		report.InFlight.Add(1)
		go func() {
			defer func() { <-sem; wg.Done(); report.InFlight.Add(-1) }()
			if err := p.st.SaveBatch(ctx, g); err != nil {
				report.FailN(len(g), err)
				return
			}
			report.Stored.Add(uint64(len(g)))
		}()
	}
	wg.Wait()
}
