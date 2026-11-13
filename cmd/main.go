package main

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
	"github.com/astianmuchui/supersimpleingestor/internal/mqtt"
	"github.com/astianmuchui/supersimpleingestor/internal/report"
	"github.com/astianmuchui/supersimpleingestor/internal/routes"
	"github.com/astianmuchui/supersimpleingestor/internal/scylla"
	"github.com/astianmuchui/supersimpleingestor/internal/store"
)

func main() {
	env.Load()

	if _, err := scylla.Connect(); err != nil {
		log.Warnf("ScyllaDB unavailable: %v", err)
	}

	st := store.NewStall(store.Scylla{}, env.GetStallAfter(), env.GetStallFor(), env.GetStallDelay())
	if st.For > 0 {
		log.Infof("stall: after %s, for %s, +%s per write", st.After, st.For, st.Delay)
	}

	go report.Run(context.Background(), st.Active)
	go mqtt.Connect(st)

	app := fiber.New(fiber.Config{
		AppName:               "IoT Ingester",
		DisableStartupMessage: true,
	})

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	routes.GetRoutes(app)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", env.GetHttpListenPort())))
}
