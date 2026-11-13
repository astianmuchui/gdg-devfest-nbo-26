package store

import (
	"context"
	"fmt"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

type Store interface {
	Save(ctx context.Context, rec scylla.TelemetryRecord) error
	SaveBatch(ctx context.Context, rows []scylla.TelemetryRecord) error
}

type Scylla struct{}

func (Scylla) Save(ctx context.Context, rec scylla.TelemetryRecord) error {
	if scylla.Session == nil {
		return fmt.Errorf("scylla not connected")
	}
	return scylla.LogTelemetry(ctx, scylla.Session, rec)
}

func (Scylla) SaveBatch(ctx context.Context, rows []scylla.TelemetryRecord) error {
	if scylla.Session == nil {
		return fmt.Errorf("scylla not connected")
	}
	if len(rows) == 1 {
		return scylla.LogTelemetry(ctx, scylla.Session, rows[0])
	}
	return scylla.LogTelemetryBatch(ctx, scylla.Session, rows)
}
