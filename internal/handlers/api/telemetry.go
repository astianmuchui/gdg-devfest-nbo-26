package api

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

func parseRangeTime(s string, endOfDay bool) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Second)
		}
		return t.UTC()
	}
	return time.Time{}
}

func scyllaUnavailable(c *fiber.Ctx) error {
	return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "ScyllaDB not connected"})
}

func TelemetryLatestHandler(c *fiber.Ctx) error {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device_id is required"})
	}
	if scylla.Session == nil {
		return scyllaUnavailable(c)
	}

	rec, err := scylla.GetLatestTelemetry(scylla.Session, deviceID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rec)
}

func TelemetryHistoryHandler(c *fiber.Ctx) error {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device_id is required"})
	}
	if scylla.Session == nil {
		return scyllaUnavailable(c)
	}

	hours := c.QueryInt("hours", 24)
	if hours <= 0 {
		hours = 24
	}
	startStr := c.Query("start")
	endStr := c.Query("end")

	end := parseRangeTime(endStr, true)
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := parseRangeTime(startStr, false)
	if start.IsZero() {
		start = end.Add(-time.Duration(hours) * time.Hour)
	}
	if end.Before(start) {
		start, end = end, start
	}

	recs, err := scylla.GetTelemetryRange(scylla.Session, deviceID, start, end)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"device_id": deviceID,
		"start":     start,
		"end":       end,
		"count":     len(recs),
		"data":      recs,
	})
}

func TelemetryCountHandler(c *fiber.Ctx) error {
	if scylla.Session == nil {
		return scyllaUnavailable(c)
	}

	deviceID := c.Query("device_id")
	count, err := scylla.GetTelemetryCount(scylla.Session, deviceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"device_id": deviceID, "count": count})
}
