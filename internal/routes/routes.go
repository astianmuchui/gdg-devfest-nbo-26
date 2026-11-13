package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/astianmuchui/supersimpleingestor/internal/handlers/api"
)

func GetRoutes(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	app.Route("/api/v1", func(v1 fiber.Router) {
		v1.Get("/telemetry/latest", api.TelemetryLatestHandler)
		v1.Get("/telemetry/history", api.TelemetryHistoryHandler)
		v1.Get("/telemetry/count", api.TelemetryCountHandler)
	})
}
