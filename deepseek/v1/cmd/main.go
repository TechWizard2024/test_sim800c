package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sim800c-supervisor/internal/api/handlers"
	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/serial"
	"sim800c-supervisor/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

func main() {
	// Charger la configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Erreur chargement config: %v", err)
	}

	// Initialiser les logs
	logger := initLogger(cfg)

	logger.Info("Démarrage de SIM800C Supervisor")

	// Initialiser la base de données
	dbConn, err := db.InitDB(cfg)
	if err != nil {
		logger.Fatalf("Erreur connexion DB: %v", err)
	}
	defer dbConn.Close()

	// Initialiser le gestionnaire WebSocket
	hub := websocket.NewHub()
	go hub.Run()

	// Initialiser le gestionnaire série
	serialManager := serial.NewManager(cfg, logger, hub)
	if err := serialManager.Start(); err != nil {
		logger.Errorf("Erreur démarrage serial manager: %v", err)
	}

	// Initialiser les handlers API
	moduleHandler := handlers.NewModuleHandler(serialManager, dbConn, logger)
	ussdHandler := handlers.NewUSSDHandler(serialManager, dbConn, cfg, logger)
	smsHandler := handlers.NewSMSHandler(serialManager, dbConn, logger)
	websocketHandler := handlers.NewWebSocketHandler(hub, logger)

	// Configurer le routeur
	router := mux.NewRouter()

	// Middleware simple pour logging
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger.Infof("Requête: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			logger.Infof("Réponse: %s %s - %v", r.Method, r.URL.Path, time.Since(start))
		})
	})

	// Middleware pour recovery
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf("Panic récupéré: %v", err)
					http.Error(w, "Erreur interne", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})

	// Routes API
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Routes publiques
	apiRouter.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
	apiRouter.HandleFunc("/ws", websocketHandler.HandleWebSocket)

	// Routes des modules
	apiRouter.HandleFunc("/modules", moduleHandler.GetModules).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}", moduleHandler.GetModule).Methods("GET")
	apiRouter.HandleFunc("/discover", moduleHandler.DiscoverModules).Methods("POST")

	// Routes USSD
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/execute", ussdHandler.ExecuteUSSD).Methods("POST")
	apiRouter.HandleFunc("/ussd/auto-status", ussdHandler.AutoStatusDiscovery).Methods("POST")
	apiRouter.HandleFunc("/ussd/auto-menu", ussdHandler.AutoMenuDiscovery).Methods("POST")

	// Routes SMS
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms", smsHandler.GetSMS).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/send", smsHandler.SendSMS).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/{index:[0-9]+}", smsHandler.DeleteSMS).Methods("DELETE")
	apiRouter.HandleFunc("/sms/trash/{id:[0-9]+}", smsHandler.MoveToTrash).Methods("POST")

	// Configurer CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://test_sim800c.local", "http://localhost"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Créer le serveur HTTP
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
	}

	// Démarrer le serveur dans une goroutine
	go func() {
		logger.Infof("Serveur démarré sur le port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Erreur serveur: %v", err)
		}
	}()

	// Attendre les signaux d'arrêt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Arrêt du serveur en cours...")

	// Contexte avec timeout pour l'arrêt
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrêter le gestionnaire série
	serialManager.Stop()

	// Arrêter le serveur HTTP
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Erreur arrêt serveur: %v", err)
	}

	logger.Info("Serveur arrêté avec succès")
}

func initLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Configurer le format
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	// Configurer le niveau
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Configurer la sortie
	if cfg.Logging.OutputPath != "" {
		// Créer le dossier logs s'il n'existe pas
		logDir := "storage/logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logger.Warnf("Impossible de créer le dossier logs: %v", err)
		} else {
			file, err := os.OpenFile(cfg.Logging.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				logger.SetOutput(file)
			} else {
				logger.Warnf("Impossible d'ouvrir le fichier de log: %v", err)
			}
		}
	}

	return logger
}
