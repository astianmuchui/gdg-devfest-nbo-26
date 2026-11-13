package store

import (
	"context"
	"time"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

type Stall struct {
	Inner Store
	Start time.Time
	After time.Duration
	For   time.Duration
	Delay time.Duration
}

func NewStall(inner Store, after, dur, delay time.Duration) *Stall {
	return &Stall{Inner: inner, Start: time.Now(), After: after, For: dur, Delay: delay}
}

func (s *Stall) Active() bool {
	if s.For <= 0 || s.Delay <= 0 {
		return false
	}
	elapsed := time.Since(s.Start)
	return elapsed >= s.After && elapsed < s.After+s.For
}

func (s *Stall) Save(ctx context.Context, rec scylla.TelemetryRecord) error {
	if s.Active() {
		t := time.NewTimer(s.Delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return s.Inner.Save(ctx, rec)
}
