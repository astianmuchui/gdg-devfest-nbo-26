package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
	"github.com/astianmuchui/supersimpleingestor/internal/mqtt"
	"github.com/astianmuchui/supersimpleingestor/internal/routes"
	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
)

func main() {
	env.Load()

	if _, err := scylla.Connect(); err != nil {
		log.Warnf("ScyllaDB unavailable: %v", err)
	}

	go mqtt.Connect()

	app := fiber.New(fiber.Config{
		AppName: "IoT Ingester",
	})

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	routes.GetRoutes(app)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", env.GetHttpListenPort())))
}
