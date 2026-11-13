package pipeline

import (
	"hash/maphash"
	"testing"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

func rec(dev string, v float64) scylla.TelemetryRecord {
	return scylla.TelemetryRecord{DeviceID: dev, TempC: v}
}

func checkGroups(t *testing.T, got [][]scylla.TelemetryRecord, want map[string][]float64, order []string) {
	t.Helper()
	if len(got) != len(order) {
		t.Fatalf("got %d groups, want %d", len(got), len(order))
	}
	for i, g := range got {
		if len(g) == 0 {
			t.Fatalf("group %d is empty", i)
		}
		dev := g[0].DeviceID
		if dev != order[i] {
			t.Fatalf("group %d is %s, want %s", i, dev, order[i])
		}
		vals := want[dev]
		if len(g) != len(vals) {
			t.Fatalf("group %s has %d rows, want %d", dev, len(g), len(vals))
		}
		for j, r := range g {
			if r.DeviceID != dev || r.TempC != vals[j] {
				t.Fatalf("group %s row %d = %s/%v, want %s/%v", dev, j, r.DeviceID, r.TempC, dev, vals[j])
			}
		}
	}
}

func TestGrouperAcrossFlushes(t *testing.T) {
	g := newGrouper(8)

	first := []scylla.TelemetryRecord{rec("A", 1), rec("B", 2), rec("A", 3), rec("C", 4), rec("B", 5)}
	checkGroups(t, g.group(first),
		map[string][]float64{"A": {1, 3}, "B": {2, 5}, "C": {4}},
		[]string{"A", "B", "C"})

	second := []scylla.TelemetryRecord{rec("C", 10), rec("D", 11), rec("C", 12)}
	checkGroups(t, g.group(second),
		map[string][]float64{"C": {10, 12}, "D": {11}},
		[]string{"C", "D"})

	checkGroups(t, g.group(nil), map[string][]float64{}, nil)
}

func TestDeviceFromTopic(t *testing.T) {
	for topic, want := range map[string]string{
		"devices/DEV00001/readings": "DEV00001",
		"devices/DEV00001":          "DEV00001",
		"devices":                   "",
		"bench/X/readings/extra":    "X",
	} {
		if got := DeviceFromTopic(topic); got != want {
			t.Errorf("DeviceFromTopic(%q) = %q, want %q", topic, got, want)
		}
	}
}

func TestDeviceFromTopicAllocs(t *testing.T) {
	if n := testing.AllocsPerRun(1000, func() { sinkHash = maphash.String(seed, DeviceFromTopic(benchTopic)) }); n != 0 {
		t.Fatalf("DeviceFromTopic + hash allocates %.0f times, want 0", n)
	}
}
