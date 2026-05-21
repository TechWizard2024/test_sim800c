package app

import (
	"blackboxv1/backend/config"
	"blackboxv1/backend/logging"
	"github.com/gofiber/fiber/v2"
)

type Application struct {
	cfg config.Config
	log *zap.Logger
}

func New(cfg config.Config, l *zap.Logger) *Application {
	return &Application{cfg: cfg, log: l}
}

func (a *Application) RegisterRoutes(f *fiber.App) {
	// placeholder
}

