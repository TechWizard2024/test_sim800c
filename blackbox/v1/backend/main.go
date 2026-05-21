package main

import (
	"blackboxv1/backend/app"
	"blackboxv1/backend/config"
	"blackboxv1/backend/logging"
	"blackboxv1/backend/realtime"
	"blackboxv1/backend/server"
	"fmt"
	"os"
)

func main() {
	cfg := config.Load()
	l := logging.NewLogger(cfg)	
	_ = realtime.NewEventBus()
	if cfg.Mode == "dev" {
		fmt.Println("Starting in dev mode")
	}

	// Route & websocket server
	application := app.New(cfg, l)
	if err := server.Run(cfg, application, l); err != nil {
		l.Fatal("server failed", zapErrField(err))
		os.Exit(1)
	}
}

func zapErrField(err error) interface{} { return err }

