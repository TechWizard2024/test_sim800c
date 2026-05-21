package server

import (
	"blackboxv1/backend/app"
	"blackboxv1/backend/config"
	"blackboxv1/backend/realtime"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

func Run(cfg config.Config, application *app.Application, l *zap.Logger) error {
	appFiber := fiber.New(fiber.Config{})
	appFiber.Use(cors.New(cors.Config{AllowOrigins: cfg.App.FrontendOrigin, AllowMethods: "*", AllowHeaders: "*"}))

	// health
	appFiber.Get("/api/system/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"ok": true}) })

	// websocket gateway
	bus := realtime.NewEventBus()
	appFiber.Use("/ws", websocket.New(func(c *websocket.Conn) {
		// subscribe each client to event stream
		ch := bus.Subscribe(64)
		defer func() { _ = c.Close() }()

		// keep-alive
		go func() {
			for {
				if err := c.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
					return
				}
			}
		}()

		for evt := range ch {
			_ = c.WriteJSON(evt)
		}
	}))

	application.RegisterRoutes(appFiber)

	return appFiber.Listen(":" + strconv.Itoa(cfg.App.Port))
}
