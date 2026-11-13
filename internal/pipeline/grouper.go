package pipeline

import "github.com/astianmuchui/supersimpleingestor/internal/scylla"

type grouper struct {
	idx    map[string]int
	groups [][]scylla.TelemetryRecord
	n      int
}

func newGrouper(capacity int) *grouper {
	return &grouper{idx: make(map[string]int, capacity)}
}

func (g *grouper) group(rows []scylla.TelemetryRecord) [][]scylla.TelemetryRecord {
	clear(g.idx)
	g.n = 0
	for _, r := range rows {
		i, ok := g.idx[r.DeviceID]
		if !ok {
			i = g.n
			g.idx[r.DeviceID] = i
			if i < len(g.groups) {
				g.groups[i] = g.groups[i][:0]
			} else {
				g.groups = append(g.groups, make([]scylla.TelemetryRecord, 0, 1))
			}
			g.n++
		}
		g.groups[i] = append(g.groups[i], r)
	}
	return g.groups[:g.n]
}
