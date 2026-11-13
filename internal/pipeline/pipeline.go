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

type job struct {
	dev     string
	payload []byte
}

type shard struct {
	in    chan job
	jobs  chan []scylla.TelemetryRecord
	wg    sync.WaitGroup
	group *grouper
}

type Pipeline struct {
	shards  []*shard
	st      store.Store
	maxRows int
	every   time.Duration
	conc    int
}

func New(st store.Store, shards, depth, maxRows, conc int, every time.Duration) *Pipeline {
	p := &Pipeline{st: st, maxRows: maxRows, every: every, conc: conc}
	p.shards = make([]*shard, shards)
	for i := range p.shards {
		p.shards[i] = &shard{
			in:    make(chan job, depth),
			jobs:  make(chan []scylla.TelemetryRecord),
			group: newGrouper(maxRows),
		}
	}
	return p
}

func (p *Pipeline) Start(ctx context.Context, wg *sync.WaitGroup) {
	for _, s := range p.shards {
		for i := 0; i < p.conc; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p.writer(ctx, s)
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(ctx, s)
		}()
	}
}

func (p *Pipeline) Submit(dev string, payload []byte) bool {
	h := maphash.String(seed, dev)
	s := p.shards[h%uint64(len(p.shards))]
	select {
	case s.in <- job{dev: dev, payload: payload}:
		return true
	default:
		report.Dropped.Add(1)
		return false
	}
}

func (p *Pipeline) QueueDepth() int {
	n := 0
	for _, s := range p.shards {
		n += len(s.in)
	}
	return n
}

func (p *Pipeline) worker(ctx context.Context, s *shard) {
	defer close(s.jobs)
	buf := make([]scylla.TelemetryRecord, 0, p.maxRows)
	t := time.NewTicker(p.every)
	defer t.Stop()
	for {
		select {
		case j, ok := <-s.in:
			if !ok {
				p.flush(s, buf)
				return
			}
			r, err := Parse(j.dev, j.payload)
			if err != nil {
				report.Fail(err)
				continue
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
		p.flush(s, buf)
		buf = buf[:0]
	}
}

func (p *Pipeline) flush(s *shard, rows []scylla.TelemetryRecord) {
	for _, g := range s.group.group(rows) {
		s.wg.Add(1)
		s.jobs <- g
	}
	s.wg.Wait()
}

func (p *Pipeline) writer(ctx context.Context, s *shard) {
	for g := range s.jobs {
		report.InFlight.Add(1)
		if err := p.st.SaveBatch(ctx, g); err != nil {
			report.FailN(len(g), err)
		} else {
			report.Stored.Add(uint64(len(g)))
		}
		report.InFlight.Add(-1)
		s.wg.Done()
	}
}
