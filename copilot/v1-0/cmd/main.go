package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sim800c-supervisor/internal/api/handlers"
	"sim800c-supervisor/internal/auth"
	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/excel"
	"sim800c-supervisor/internal/serial"
	"sim800c-supervisor/internal/sms"
	"sim800c-supervisor/internal/ussd"
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
	logger.Info("Démarrage de SIM800C Supervisor v2.0")
	logger.Infof("Config MySQL: user=%s db=%s host=%s", cfg.MySQL.User, cfg.MySQL.Database, cfg.MySQL.Host)

	// Initialiser la base de données
	dbConn, err := db.InitDB(cfg)
	if err != nil {
		logger.Fatalf("Erreur connexion DB: %v", err)
	}
	defer dbConn.Close()

	// Initialiser le gestionnaire d'authentification
	authManager := auth.NewAuthManager(dbConn, cfg, logger)
	authManager.CreateDefaultAdmin()

	// Initialiser le gestionnaire WebSocket
	hub := websocket.NewHub()
	go hub.Run()

	// Initialiser le gestionnaire série (communication réelle)
	serialManager := serial.NewManager(cfg, logger, hub)
	if err := serialManager.Start(); err != nil {
		logger.Errorf("Erreur démarrage serial manager: %v", err)
	}

	// Initialiser le gestionnaire Excel
	excelReader := excel.NewExcelReader(cfg.Excel.BasePath, cfg.Excel.FilenamePattern, logger)
	if err := excelReader.Load(); err != nil {
		logger.Warnf("Erreur chargement Excel: %v", err)
	}
	excelWriter := excel.NewExcelWriter(cfg.Excel.BasePath, logger)

	// Initialiser le gestionnaire SMS
	smsManager := sms.NewSMSManager(logger, hub, dbConn, cfg.SMS.AutoTrashKeyword)

	// Initialiser le gestionnaire USSD
	ussdExecutor := ussd.NewUSSDExecutor(logger)
	ussdExplorer := ussd.NewUSSDExplorer(ussdExecutor, excelReader, excelWriter, logger, cfg.USSD.MaxMenuDepth)

	// Configurer le routeur
	router := mux.NewRouter()

	// Middleware
	router.Use(loggingMiddleware(logger))
	router.Use(recoveryMiddleware(logger))

	// Servir les fichiers statiques (sans embed)
	webDir := "./web"
	if _, err := os.Stat(webDir); err == nil {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(webDir)))
		logger.Info("Frontend servi depuis le dossier web/")
	} else {
		logger.Warn("Dossier web/ non trouvé")
	}

	// Routes API
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Routes publiques
	apiRouter.HandleFunc("/health", healthCheck).Methods("GET")
	apiRouter.HandleFunc("/login", authManager.LoginHandler).Methods("POST")
	apiRouter.HandleFunc("/logout", authManager.LogoutHandler).Methods("POST")

	// Routes protégées (sauf login/logout/health)
	apiRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ignore auth for public endpoints
			if r.URL.Path == "/api/health" || r.URL.Path == "/api/login" || r.URL.Path == "/api/logout" {
				next.ServeHTTP(w, r)
				return
			}
			authManager.AuthMiddlewareMux(next).ServeHTTP(w, r)
		})
	})

	// Modules
	apiRouter.HandleFunc("/modules", getModulesHandler(serialManager, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}", getModuleHandler(serialManager, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/codes", getModuleUSSDCodesHandler(serialManager, excelReader, logger)).Methods("GET")
	apiRouter.HandleFunc("/discover", discoverModulesHandler(serialManager, logger)).Methods("POST")

	// USSD
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/execute", executeUSSDHandler(serialManager, dbConn, ussdExecutor, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/auto-status", autoStatusHandler(serialManager, excelReader, ussdExecutor, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/auto-menu", autoMenuHandler(serialManager, excelReader, ussdExplorer, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/explore/{id:[0-9]+}/{code}", exploreMenuHandler(serialManager, ussdExplorer, logger)).Methods("POST")

	// SMS
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms", getSMSHandler(smsManager, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/send", sendSMSHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/{index:[0-9]+}", deleteSMSHandler(smsManager, logger)).Methods("DELETE")
	apiRouter.HandleFunc("/sms/trash/{id:[0-9]+}", moveToTrashHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/sms/read-all", readAllSMSHandler(smsManager, serialManager, logger)).Methods("POST")

	// Authentification
	apiRouter.HandleFunc("/user/profile", authManager.GetProfile).Methods("GET")
	apiRouter.HandleFunc("/user/password", authManager.ChangePassword).Methods("POST")
	apiRouter.HandleFunc("/audit/logs", getAuditLogsHandler(dbConn, logger)).Methods("GET")

	// Excel
	apiRouter.HandleFunc("/excel/reload", reloadExcelHandler(excelReader, logger)).Methods("POST")
	apiRouter.HandleFunc("/excel/versions", getExcelVersionsHandler(dbConn, logger)).Methods("GET")

	// WebSocket (auth JWT via Authorization header)
	wsHandler := handlers.NewWebSocketHandler(hub, logger, authManager)
	apiRouter.HandleFunc("/ws", wsHandler.HandleWebSocket).Methods("GET")

	// Configurer CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8082", "http://127.0.0.1:8082"},
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

	// Démarrer la routine de surveillance des SMS
	go smsManager.StartMonitoring(serialManager, cfg.SMS.CheckIntervalSeconds)

	// Démarrer le serveur
	go func() {
		logger.Infof("Serveur démarré sur http://localhost:%d", cfg.Server.Port)
		logger.Infof("API Health: http://localhost:%d/api/health", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Erreur serveur: %v", err)
		}
	}()

	// Attendre l'arrêt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Arrêt du serveur...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	serialManager.Stop()
	server.Shutdown(ctx)
	logger.Info("Serveur arrêté")
}

func initLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
	level, _ := logrus.ParseLevel(cfg.Logging.Level)
	logger.SetLevel(level)
	return logger
}

func loggingMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Infof("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

func recoveryMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Errorf("Panic: %v", err)
					http.Error(w, "Erreur interne", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": "2.0",
	})
}

func getModulesHandler(sm *serial.Manager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modules := sm.GetAllModules()
		result := make([]map[string]interface{}, 0)
		for _, m := range modules {
			result = append(result, map[string]interface{}{
				"id":           m.ModuleID,
				"port":         m.Port,
				"imei":         m.IMEI,
				"phone_number": m.PhoneNumber,
				"carrier":      m.Carrier,
				"status":       "connected",
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func getModuleHandler(sm *serial.Manager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var id int
		fmt.Sscanf(vars["id"], "%d", &id)

		for _, m := range sm.GetAllModules() {
			if m.ModuleID == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(m)
				return
			}
		}
		http.Error(w, "Module non trouvé", http.StatusNotFound)
	}
}

func getModuleUSSDCodesHandler(sm *serial.Manager, reader *excel.ExcelReader, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		var targetModule *serial.SIM800C
		for _, m := range sm.GetAllModules() {
			if m.ModuleID == moduleID {
				targetModule = m
				break
			}
		}
		if targetModule == nil {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		consult := reader.GetConsultCodes(targetModule.Carrier)
		services := reader.GetServiceNCodes(targetModule.Carrier)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"consult_codes":   consult,
			"service_n_codes": services,
		})
	}
}

func discoverModulesHandler(sm *serial.Manager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modules := sm.GetAllModules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "completed",
			"modules": len(modules),
		})
	}
}

func executeUSSDHandler(sm *serial.Manager, dbConn *db.DB, executor *ussd.USSDExecutor, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		var req struct {
			USSDCode  string `json:"ussd_code"`
			InputData string `json:"input_data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}

		var targetModule *serial.SIM800C
		for _, m := range sm.GetAllModules() {
			if m.ModuleID == moduleID {
				targetModule = m
				break
			}
		}
		if targetModule == nil {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		ussdReq := &ussd.USSDRequest{
			Module:    targetModule,
			Code:      req.USSDCode,
			InputData: req.InputData,
			ModuleID:  moduleID,
		}

		startTime := time.Now()
		response, err := executor.Execute(ussdReq)
		duration := time.Since(startTime)

		status := "success"
		if err != nil {
			status = "error"
		}
		history := &db.USSDHistory{
			ModuleID:   moduleID,
			USSDCode:   req.USSDCode,
			InputData:  req.InputData,
			OutputData: response.Result,
			Status:     status,
			DurationMs: int(duration.Milliseconds()),
			ExecutedBy: r.Header.Get("X-User-ID"),
		}
		dbConn.SaveUSSDHistory(history)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"result":   response.Result,
			"duration": duration.Milliseconds(),
		})
	}
}

func autoStatusHandler(sm *serial.Manager, reader *excel.ExcelReader, executor *ussd.USSDExecutor, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modules := sm.GetAllModules()
		results := make(map[int]map[string]string)

		for _, module := range modules {
			moduleResults := make(map[string]string)
			codes := reader.GetConsultCodes(module.Carrier)

			for _, code := range codes {
				req := &ussd.USSDRequest{
					Module:   module,
					Code:     code.USSDCode,
					ModuleID: module.ModuleID,
				}
				response, err := executor.Execute(req)
				if err != nil {
					moduleResults[code.Operation] = "Erreur: " + err.Error()
				} else {
					moduleResults[code.Operation] = response.Result
				}
				time.Sleep(1000 * time.Millisecond)
			}
			results[module.ModuleID] = moduleResults
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func autoMenuHandler(sm *serial.Manager, reader *excel.ExcelReader, explorer *ussd.USSDExplorer, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modules := sm.GetAllModules()
		results := make(map[int]interface{})

		for _, module := range modules {
			codes := reader.GetServiceNCodes(module.Carrier)
			moduleResults := make(map[string]interface{})

			for _, code := range codes {
				result, err := explorer.ExploreMenu(module, code.USSDCode, code.ID)
				if err != nil {
					moduleResults[code.Operation] = map[string]interface{}{
						"error": err.Error(),
					}
				} else {
					moduleResults[code.Operation] = map[string]interface{}{
						"discovered_codes": len(result.DiscoveredCodes),
						"menu_tree":        explorer.FormatMenuTree(result.MenuTree, 0),
					}
				}
				time.Sleep(1000 * time.Millisecond)
			}
			results[module.ModuleID] = moduleResults
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func exploreMenuHandler(sm *serial.Manager, explorer *ussd.USSDExplorer, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)
		code := vars["code"]

		var targetModule *serial.SIM800C
		for _, m := range sm.GetAllModules() {
			if m.ModuleID == moduleID {
				targetModule = m
				break
			}
		}
		if targetModule == nil {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		result, err := explorer.ExploreMenu(targetModule, code, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":          true,
			"discovered_codes": len(result.DiscoveredCodes),
			"menu_tree":        explorer.FormatMenuTree(result.MenuTree, 0),
		})
	}
}

func getSMSHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		includeTrash := r.URL.Query().Get("include_trash") == "true"
		smsList, err := smsManager.GetSMS(moduleID, includeTrash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(smsList)
	}
}

func sendSMSHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		var req struct {
			Number  string `json:"number"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}

		if err := smsManager.SendSMS(moduleID, req.Number, req.Message); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "SMS envoyé"})
	}
}

func deleteSMSHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID, index int
		fmt.Sscanf(vars["id"], "%d", &moduleID)
		fmt.Sscanf(vars["index"], "%d", &index)

		if err := smsManager.DeleteSMS(moduleID, index); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "SMS supprimé"})
	}
}

func moveToTrashHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var smsID int
		fmt.Sscanf(vars["id"], "%d", &smsID)

		if err := smsManager.MoveToTrash(smsID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "SMS déplacé vers corbeille"})
	}
}

func readAllSMSHandler(smsManager *sms.SMSManager, sm *serial.Manager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, module := range sm.GetAllModules() {
			if err := smsManager.ReadSMS(module); err != nil {
				logger.Errorf("Erreur lecture SMS module %s: %v", module.Port, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Lecture SMS terminée"})
	}
}

func getAuditLogsHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		logs, err := dbConn.GetAuditLogs(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
	}
}

func reloadExcelHandler(reader *excel.ExcelReader, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := reader.Load(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Excel rechargé"})
	}
}

func getExcelVersionsHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versions, err := dbConn.GetExcelVersions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(versions)
	}
}
