package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

type readingsPayload struct {
	DeviceID    string `json:"device_id"`
	Timestamp   int64  `json:"timestamp"`
	UptimeS     int64  `json:"uptime_s"`
	Environment struct {
		TempC         float64 `json:"temperature_c"`
		HumidityPct   float64 `json:"humidity_pct"`
		AirQualityPPM float64 `json:"air_quality_ppm"`
		AirQualityV   float64 `json:"air_quality_v"`
	} `json:"environment"`
}

func DeviceFromTopic(topic string) string {
	i := strings.IndexByte(topic, '/')
	if i < 0 {
		return ""
	}
	rest := topic[i+1:]
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		return rest[:j]
	}
	return rest
}

func Parse(dev string, raw []byte) (scylla.TelemetryRecord, error) {
	var p readingsPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return scylla.TelemetryRecord{}, fmt.Errorf("Parse: %w", err)
	}

	if p.DeviceID == "" {
		p.DeviceID = dev
	}
	if p.DeviceID == "" {
		return scylla.TelemetryRecord{}, fmt.Errorf("Parse: missing device_id")
	}

	return scylla.TelemetryRecord{
		DeviceID:      p.DeviceID,
		CreatedAt:     time.Now().UTC(),
		DeviceTS:      p.Timestamp,
		UptimeS:       p.UptimeS,
		TempC:         p.Environment.TempC,
		HumidityPct:   max(p.Environment.HumidityPct, 0),
		AirQualityPPM: max(p.Environment.AirQualityPPM, 0),
		AirQualityV:   max(p.Environment.AirQualityV, 0),
	}, nil
}
