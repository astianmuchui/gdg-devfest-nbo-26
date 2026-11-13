package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
	"github.com/astianmuchui/supersimpleingestor/internal/store"
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

func handleReadings(ctx context.Context, st store.Store, topic string, raw []byte) error {
	var p readingsPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("handleReadings: %w", err)
	}

	if p.DeviceID == "" {
		parts := strings.Split(topic, "/")
		if len(parts) >= 2 {
			p.DeviceID = parts[1]
		}
	}
	if p.DeviceID == "" {
		return fmt.Errorf("handleReadings: missing device_id")
	}

	rec := scylla.TelemetryRecord{
		DeviceID:      p.DeviceID,
		CreatedAt:     time.Now().UTC(),
		DeviceTS:      p.Timestamp,
		UptimeS:       p.UptimeS,
		TempC:         p.Environment.TempC,
		HumidityPct:   max(p.Environment.HumidityPct, 0),
		AirQualityPPM: max(p.Environment.AirQualityPPM, 0),
		AirQualityV:   max(p.Environment.AirQualityV, 0),
	}

	if err := st.Save(ctx, rec); err != nil {
		return fmt.Errorf("handleReadings save: %w", err)
	}
	return nil
}
