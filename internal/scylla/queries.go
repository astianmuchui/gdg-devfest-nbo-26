package scylla

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

const selectColumns = `device_id, created_at, device_ts, uptime_s, temp_c, humidity_pct, air_quality_ppm, air_quality_v`

const insertTelemetry = `INSERT INTO telemetry (` + selectColumns + `) VALUES (?,?,?,?,?,?,?,?)`

func scanAll(iter *gocql.Iter) ([]TelemetryRecord, error) {
	results := []TelemetryRecord{}
	var r TelemetryRecord
	for iter.Scan(&r.DeviceID, &r.CreatedAt, &r.DeviceTS, &r.UptimeS, &r.TempC, &r.HumidityPct, &r.AirQualityPPM, &r.AirQualityV) {
		results = append(results, r)
		r = TelemetryRecord{}
	}
	return results, iter.Close()
}

func LogTelemetry(session *gocql.Session, r TelemetryRecord) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return session.Query(insertTelemetry,
		r.DeviceID, r.CreatedAt, r.DeviceTS, r.UptimeS,
		r.TempC, r.HumidityPct, r.AirQualityPPM, r.AirQualityV,
	).Exec()
}

func LogTelemetryBatch(ctx context.Context, s *gocql.Session, rows []TelemetryRecord) error {
	b := s.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
	for _, r := range rows {
		b.Query(insertTelemetry, r.DeviceID, r.CreatedAt, r.DeviceTS, r.UptimeS,
			r.TempC, r.HumidityPct, r.AirQualityPPM, r.AirQualityV)
	}
	return s.ExecuteBatch(b)
}

func GetLatestTelemetry(session *gocql.Session, deviceID string) (TelemetryRecord, error) {
	var r TelemetryRecord
	err := session.Query(`
		SELECT `+selectColumns+`
		FROM telemetry
		WHERE device_id = ?
		LIMIT 1`,
		deviceID,
	).Scan(&r.DeviceID, &r.CreatedAt, &r.DeviceTS, &r.UptimeS, &r.TempC, &r.HumidityPct, &r.AirQualityPPM, &r.AirQualityV)
	if err != nil {
		return r, fmt.Errorf("GetLatestTelemetry: %w", err)
	}
	return r, nil
}

func GetTelemetryRange(session *gocql.Session, deviceID string, start, end time.Time) ([]TelemetryRecord, error) {
	iter := session.Query(`
		SELECT `+selectColumns+`
		FROM telemetry
		WHERE device_id = ? AND created_at >= ? AND created_at <= ?
		ORDER BY created_at ASC`,
		deviceID, start, end,
	).Iter()
	results, err := scanAll(iter)
	if err != nil {
		return nil, fmt.Errorf("GetTelemetryRange: %w", err)
	}
	return results, nil
}

func GetTelemetryCount(session *gocql.Session, deviceID string) (int64, error) {
	var count int64
	var err error
	if deviceID != "" {
		err = session.Query(`SELECT COUNT(*) FROM telemetry WHERE device_id = ?`, deviceID).Scan(&count)
	} else {
		err = session.Query(`SELECT COUNT(*) FROM telemetry`).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("GetTelemetryCount: %w", err)
	}
	return count, nil
}
