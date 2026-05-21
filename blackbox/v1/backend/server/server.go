package server

import (
	"blackboxv1/backend/config"
	"blackboxv1/backend/logging"
	"blackboxv1/backend/app"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func Run(cfg config.Config, application *app.Application, l *zap.Logger) error {
	appFiber := fiber.New(fiber.Config{})
	appFiber.Use(cors.New(cors.Config{AllowOrigins: cfg.App.FrontendOrigin, AllowMethods: "*", AllowHeaders: "*"}))

	// health
	appFiber.Get("/api/system/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })

	application.RegisterRoutes(appFiber)

	return appFiber.Listen(":" + strconv.Itoa(cfg.App.Port))
}

