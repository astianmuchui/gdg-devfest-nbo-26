package scylla

import "time"

type TelemetryRecord struct {
	DeviceID      string    `json:"device_id"`
	CreatedAt     time.Time `json:"created_at"`
	DeviceTS      int64     `json:"device_ts"`
	UptimeS       int64     `json:"uptime_s"`
	TempC         float64   `json:"temp_c"`
	HumidityPct   float64   `json:"humidity_pct"`
	AirQualityPPM float64   `json:"air_quality_ppm"`
	AirQualityV   float64   `json:"air_quality_v"`
}
