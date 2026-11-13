package pipeline

import (
	"fmt"
	"hash/maphash"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

const (
	benchTopic   = "devices/DEV00001/readings"
	benchPayload = `{"device_id":"DEV00001","timestamp":1794572813855,"uptime_s":19030,"environment":{"temperature_c":21.92,"humidity_pct":51.46,"air_quality_ppm":429.7,"air_quality_v":0.709}}`
)

var (
	sinkHash   uint64
	sinkRec    scylla.TelemetryRecord
	sinkGroups int
)

func TestBenchPayloadSize(t *testing.T) {
	if len(benchPayload) != 172 {
		t.Fatalf("payload is %d bytes, want 172", len(benchPayload))
	}
}

func BenchmarkReceive(b *testing.B) {
	payload := []byte(benchPayload)

	b.Run("v3-parse-on-dispatcher", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rec, err := Parse("", payload)
			if err != nil {
				b.Fatal(err)
			}
			if rec.DeviceID == "" {
				rec.DeviceID = strings.Split(benchTopic, "/")[1]
			}
			sinkRec = rec
			sinkHash = maphash.String(seed, rec.DeviceID)
		}
	})

	b.Run("v4-topic-slice-hash", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkHash = maphash.String(seed, DeviceFromTopic(benchTopic))
		}
	})
}

func benchRows(n, devices int) []scylla.TelemetryRecord {
	r := rand.New(rand.NewPCG(1, 2))
	rows := make([]scylla.TelemetryRecord, n)
	for i := range rows {
		rows[i] = scylla.TelemetryRecord{DeviceID: fmt.Sprintf("DEV%05d", r.IntN(devices)), TempC: float64(i)}
	}
	return rows
}

func groupFreshMap(rows []scylla.TelemetryRecord) int {
	groups := make(map[string][]scylla.TelemetryRecord)
	for _, r := range rows {
		groups[r.DeviceID] = append(groups[r.DeviceID], r)
	}
	return len(groups)
}

func groupSortRuns(buf []scylla.TelemetryRecord, runs [][]scylla.TelemetryRecord) [][]scylla.TelemetryRecord {
	slices.SortFunc(buf, func(a, b scylla.TelemetryRecord) int { return strings.Compare(a.DeviceID, b.DeviceID) })
	runs = runs[:0]
	start := 0
	for i := 1; i <= len(buf); i++ {
		if i == len(buf) || buf[i].DeviceID != buf[start].DeviceID {
			runs = append(runs, buf[start:i])
			start = i
		}
	}
	return runs
}

func BenchmarkGroup500(b *testing.B) {
	for _, tc := range []struct {
		name    string
		devices int
	}{
		{"fleet-50000-devices", 50000},
		{"chatty-50-devices", 50},
	} {
		rows := benchRows(500, tc.devices)
		buf := make([]scylla.TelemetryRecord, len(rows))

		b.Run(tc.name+"/fresh-map", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				copy(buf, rows)
				sinkGroups = groupFreshMap(buf)
			}
		})

		b.Run(tc.name+"/sort-and-runs", func(b *testing.B) {
			b.ReportAllocs()
			var runs [][]scylla.TelemetryRecord
			for b.Loop() {
				copy(buf, rows)
				runs = groupSortRuns(buf, runs)
				sinkGroups = len(runs)
			}
		})

		b.Run(tc.name+"/reusable-grouper", func(b *testing.B) {
			b.ReportAllocs()
			g := newGrouper(len(rows))
			for b.Loop() {
				copy(buf, rows)
				sinkGroups = len(g.group(buf))
			}
		})
	}
}
