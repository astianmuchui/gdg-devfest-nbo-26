package store

import (
	"context"
	"fmt"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

type Store interface {
	Save(ctx context.Context, rec scylla.TelemetryRecord) error
}

type Scylla struct{}

func (Scylla) Save(ctx context.Context, rec scylla.TelemetryRecord) error {
	if scylla.Session == nil {
		return fmt.Errorf("scylla not connected")
	}
	return scylla.LogTelemetry(scylla.Session, rec)
}
