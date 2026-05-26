package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	// MICRO-BLOC C5 — Injecter la DB pour le logging du signal
	serialManager.SetDB(dbConn)

	// Charger les paramètres persistants depuis la DB (délais USSD, etc.)
	if val, err := dbConn.GetSetting("ussd.explore_delay_ms"); err == nil && val != "" {
		var v int
		fmt.Sscanf(val, "%d", &v)
		if v >= 500 {
			cfg.USSD.ExploreDelayMs = v
			logger.Infof("Délai exploration USSD restauré depuis DB: %dms", v)
		}
	}
	if val, err := dbConn.GetSetting("ussd.nav_delay_ms"); err == nil && val != "" {
		var v int
		fmt.Sscanf(val, "%d", &v)
		if v >= 100 {
			cfg.USSD.NavDelayMs = v
			logger.Infof("Délai navigation USSD restauré depuis DB: %dms", v)
		}
	}
	if val, err := dbConn.GetSetting("sms.auto_trash_keyword"); err == nil && val != "" {
		cfg.SMS.AutoTrashKeyword = val
		logger.Infof("Mot-clé corbeille SMS restauré depuis DB: %q", val)
	}
	if val, err := dbConn.GetSetting("ussd.retry_on_error"); err == nil && val != "" {
		cfg.USSD.RetryOnError = val == "true"
	}
	if val, err := dbConn.GetSetting("ussd.max_retries"); err == nil && val != "" {
		var v int
		fmt.Sscanf(val, "%d", &v)
		if v > 0 {
			cfg.USSD.MaxRetries = v
		}
	}
	if val, err := dbConn.GetSetting("ussd.max_menu_depth"); err == nil && val != "" {
		var v int
		fmt.Sscanf(val, "%d", &v)
		if v > 0 {
			cfg.USSD.MaxMenuDepth = v
		}
	}
	// Restaurer la whitelist des ports COM depuis la DB
	if val, err := dbConn.GetSetting("serial.ports_whitelist"); err == nil && val != "" && val != "[]" {
		// val est une liste séparée par des virgules (ex: "COM5,COM6,COM7")
		parts := strings.Split(val, ",")
		ports := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				ports = append(ports, p)
			}
		}
		if len(ports) > 0 {
			cfg.Serial.Ports = ports
			logger.Infof("Whitelist ports COM restaurée depuis DB: %v", ports)
		}
	}

	// Charger le plan de numérotation depuis la DB et l'injecter dans le serial Manager
	if dialPlanEntries, err := dbConn.GetDialPlan(); err == nil && len(dialPlanEntries) > 0 {
		for _, dp := range dialPlanEntries {
			serialManager.DialPlan = append(serialManager.DialPlan, serial.DialPlanEntry{
				Operator:     dp.Operator,
				Prefix:       dp.Prefix,
				CallingCode:  dp.CallingCode,
				NumberLength: dp.NumberLength,
			})
		}
		logger.Infof("Plan de numérotation chargé depuis DB: %d entrées", len(serialManager.DialPlan))
	} else {
		logger.Warn("Plan de numérotation non disponible depuis DB — fallback hardcodé CI actif")
	}

	// Persist module to DB after initialization
	serialManager.OnModuleInitialized = func(module *serial.SIM800C) {
		dbModule := &db.Module{
			COMPort:     module.Port,
			IMEI:        module.IMEI,
			PhoneNumber: module.PhoneNumber,
			Carrier:     module.Carrier,
			Status:      "connected",
		}
		if err := dbConn.SaveModule(dbModule); err != nil {
			logger.Warnf("Erreur sauvegarde module %s en DB: %v", module.Port, err)
		} else {
			// Sync back the DB ID into the in-memory struct
			if dbMod, err := dbConn.GetModuleByCOMPort(module.Port); err == nil && dbMod != nil {
				module.DBID = dbMod.ID
				logger.Infof("Module %s persisté en DB (DB_ID=%d, Mem_ID=%d)", module.Port, dbMod.ID, module.ModuleID)
			}
		}
	}
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
	apiRouter.HandleFunc("/discover", discoverModulesHandler(serialManager, logger)).Methods("POST")

	// USSD
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/execute", executeUSSDHandler(serialManager, dbConn, ussdExecutor, logger)).Methods("POST")
	// Update 23052026-0937
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/status-codes", statusCodesHandler(serialManager, excelReader, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/menu-codes", menuCodesHandler(serialManager, excelReader, logger)).Methods("GET")
	// Update 23052026-0937
	apiRouter.HandleFunc("/ussd/auto-status", autoStatusHandler(serialManager, excelReader, ussdExecutor, hub, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/auto-menu", autoMenuHandler(serialManager, excelReader, ussdExplorer, hub, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/explore/{id:[0-9]+}/{code}", exploreMenuHandler(serialManager, ussdExplorer, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/navigate", navigateUSSDHandler(serialManager, dbConn, ussdExecutor, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/signal", getModuleSignalHandler(serialManager, logger)).Methods("GET")
	// MICRO-BLOC C6 — Historique signal
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/signal/history", getSignalHistoryHandler(dbConn, logger)).Methods("GET")
	// Update 24052026 — per-module auto-status
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/auto-status", moduleAutoStatusHandler(serialManager, excelReader, ussdExecutor, hub, logger)).Methods("POST")
	// Update 24052026 — per-module auto-menu
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/auto-menu", moduleAutoMenuHandler(serialManager, excelReader, ussdExplorer, hub, logger)).Methods("POST")

	// SMS
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms", getSMSHandler(smsManager, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/send", sendSMSHandler(serialManager, smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/export", exportSMSCSVHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/{index:[0-9]+}", deleteSMSHandler(smsManager, logger)).Methods("DELETE")
	apiRouter.HandleFunc("/sms/trash/{id:[0-9]+}", moveToTrashHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/sms/restore/{id:[0-9]+}", restoreFromTrashHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/sms/delete-permanent/{id:[0-9]+}", deletePermanentHandler(smsManager, logger)).Methods("DELETE")
	apiRouter.HandleFunc("/sms/read-all", readAllSMSHandler(smsManager, serialManager, logger)).Methods("POST")
	// MICRO-BLOC C3 — Export SMS tous modules
	apiRouter.HandleFunc("/sms/export", exportAllSMSCSVHandler(dbConn, logger)).Methods("GET")
	// MICRO-BLOC A2 — SMS is_read routes (v1-14)
	apiRouter.HandleFunc("/sms/mark-read/{id:[0-9]+}", markSMSReadHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/mark-all-read", markAllSMSReadHandler(smsManager, logger)).Methods("POST")
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/sms/unread-count", getUnreadSMSCountHandler(smsManager, logger)).Methods("GET")

	// Authentification
	apiRouter.HandleFunc("/user/profile", authManager.GetProfile).Methods("GET")
	apiRouter.HandleFunc("/user/password", authManager.ChangePassword).Methods("POST")
	apiRouter.HandleFunc("/audit/logs", getAuditLogsHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/system/status", systemStatusHandler(serialManager, dbConn, cfg, logger)).Methods("GET")

	// Excel
	apiRouter.HandleFunc("/excel/reload", reloadExcelHandler(excelReader, logger)).Methods("POST")
	apiRouter.HandleFunc("/excel/versions", getExcelVersionsHandler(dbConn, logger)).Methods("GET")

	// Dial Plan
	apiRouter.HandleFunc("/dialplan", getDialPlanHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/dialplan", createDialPlanHandler(dbConn, logger)).Methods("POST")
	apiRouter.HandleFunc("/dialplan/reload", reloadDialPlanHandler(dbConn, serialManager, hub, logger)).Methods("POST")
	apiRouter.HandleFunc("/dialplan/export", exportDialPlanCSVHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/dialplan/{id:[0-9]+}", updateDialPlanHandler(dbConn, logger)).Methods("PUT")
	apiRouter.HandleFunc("/dialplan/{id:[0-9]+}", deleteDialPlanHandler(dbConn, logger)).Methods("DELETE")

	// Configuration (read + update delays)
	apiRouter.HandleFunc("/config", getConfigHandler(cfg, logger)).Methods("GET")
	apiRouter.HandleFunc("/config/delays", updateDelaysHandler(cfg, dbConn, logger)).Methods("PUT")
	apiRouter.HandleFunc("/config/advanced", getAdvancedSettingsHandler(cfg, dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/config/advanced", updateAdvancedSettingsHandler(cfg, dbConn, smsManager, ussdExplorer, hub, logger)).Methods("PUT")
	apiRouter.HandleFunc("/config/ports", updatePortsWhitelistHandler(cfg, dbConn, logger)).Methods("PUT")
	apiRouter.HandleFunc("/config/ports", getPortsWhitelistHandler(cfg, logger)).Methods("GET")

	// USSD Favorites
	apiRouter.HandleFunc("/ussd/favorites", getUSSDFavoritesHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/ussd/favorites", addUSSDFavoriteHandler(dbConn, logger)).Methods("POST")
	apiRouter.HandleFunc("/ussd/favorites/{id:[0-9]+}", deleteUSSDFavoriteHandler(dbConn, logger)).Methods("DELETE")

	// USSD History
	apiRouter.HandleFunc("/ussd/history", getUSSDHistoryHandler(dbConn, logger)).Methods("GET")
	apiRouter.HandleFunc("/ussd/history/export", exportUSSDHistoryCSVHandler(dbConn, logger)).Methods("GET")
	// MICRO-BLOC B1 — codes USSD récents par module
	apiRouter.HandleFunc("/modules/{id:[0-9]+}/ussd/recent", getRecentUSSDCodesHandler(dbConn, logger)).Methods("GET")

	// WebSocket (auth JWT via Authorization header)
	wsHandler := handlers.NewWebSocketHandler(hub, logger, authManager)
	apiRouter.HandleFunc("/ws", wsHandler.HandleWebSocket).Methods("GET")

	// Configurer CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			fmt.Sprintf("http://localhost:%d", cfg.Server.Port),
			fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port),
			fmt.Sprintf("http://test-sim800c.lan:%d", cfg.Server.Port),
			"http://test-sim800c.lan",
		},
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

// startupTime records when the application started (used by /api/system/status)
var startupTime = time.Now()

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
				"id":             m.ModuleID,
				"module_id":      m.ModuleID,
				"db_id":          m.DBID,
				"port":           m.Port,
				"imei":           m.IMEI,
				"phone_number":   m.PhoneNumber,
				"carrier":        m.Carrier,
				"pin_unlocked":   m.PINUnlocked,
				"pin_failed":     m.PINFailed,
				"signal_quality": m.SignalQuality,
				"signal_rssi":    serial.CSQToRSSI(m.SignalQuality),
				"network_status": m.NetworkStatus,
				"status":         "connected",
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

		if m, ok := sm.GetModuleByDBID(id); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(m)
			return
		}
		http.Error(w, "Module non trouvé", http.StatusNotFound)
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
		if m, ok := sm.GetModuleByDBID(moduleID); ok {
			targetModule = m
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
			ModuleID:   targetModule.GetEffectiveID(),
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

func autoStatusHandler(sm *serial.Manager, reader *excel.ExcelReader, executor *ussd.USSDExecutor, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
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
				// Broadcast real-time progress via WebSocket
				if hub != nil {
					hub.BroadcastEvent(websocket.Event{
						Type:     "auto_status_progress",
						ModuleID: module.ModuleID,
						Data: map[string]interface{}{
							"port":      module.Port,
							"operation": code.Operation,
							"ussd_code": code.USSDCode,
							"result":    moduleResults[code.Operation],
						},
						Timestamp: time.Now(),
					})
				}
				time.Sleep(1000 * time.Millisecond)
			}
			results[module.ModuleID] = moduleResults
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func autoMenuHandler(sm *serial.Manager, reader *excel.ExcelReader, explorer *ussd.USSDExplorer, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		modules := sm.GetAllModules()
		results := make(map[int]interface{})

		for _, module := range modules {
			codes := reader.GetServiceNCodes(module.Carrier)
			moduleResults := make(map[string]interface{})

			for _, code := range codes {
				// Broadcast start of exploration
				if hub != nil {
					hub.BroadcastEvent(websocket.Event{
						Type:     "auto_menu_progress",
						ModuleID: module.ModuleID,
						Data: map[string]interface{}{
							"port":      module.Port,
							"operation": code.Operation,
							"ussd_code": code.USSDCode,
							"status":    "exploring",
						},
						Timestamp: time.Now(),
					})
				}
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
				// Broadcast result
				if hub != nil {
					hub.BroadcastEvent(websocket.Event{
						Type:     "auto_menu_progress",
						ModuleID: module.ModuleID,
						Data: map[string]interface{}{
							"port":      module.Port,
							"operation": code.Operation,
							"ussd_code": code.USSDCode,
							"status":    "done",
							"result":    moduleResults[code.Operation],
						},
						Timestamp: time.Now(),
					})
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

		targetModule, ok := sm.GetModuleByDBID(moduleID)
		if !ok {
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

func sendSMSHandler(sm *serial.Manager, smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
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

		// Find the actual serial module to use real serial port + DBID
		targetModule, ok := sm.GetModuleByDBID(moduleID)
		if !ok {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		if err := smsManager.SendSMSWithModule(targetModule, req.Number, req.Message); err != nil {
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

// POST /api/sms/restore/{id} — Restaure un SMS depuis la corbeille
func restoreFromTrashHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var smsID int
		fmt.Sscanf(vars["id"], "%d", &smsID)

		if err := smsManager.RestoreFromTrash(smsID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "SMS restauré"})
	}
}

// DELETE /api/sms/delete-permanent/{id} — Supprime définitivement un SMS
func deletePermanentHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var smsID int
		fmt.Sscanf(vars["id"], "%d", &smsID)

		if err := smsManager.DeletePermanent(smsID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "SMS supprimé définitivement"})
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

// POST /api/sms/mark-read/{id} — Marque un SMS comme lu (MICRO-BLOC A2 v1-14)
func markSMSReadHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var smsID int
		fmt.Sscanf(vars["id"], "%d", &smsID)
		if smsID <= 0 {
			http.Error(w, "ID SMS invalide", http.StatusBadRequest)
			return
		}

		// Récupérer le module_id depuis le body (optionnel, utilisé pour le broadcast WS)
		var req struct {
			ModuleID int `json:"module_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if err := smsManager.MarkRead(smsID, req.ModuleID); err != nil {
			logger.Errorf("Erreur mark SMS %d read: %v", smsID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "SMS marqué comme lu",
			"sms_id": smsID,
		})
	}
}

// POST /api/modules/{id}/sms/mark-all-read — Marque tous les SMS d'un module comme lus (MICRO-BLOC A2 v1-14)
func markAllSMSReadHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)
		if moduleID <= 0 {
			http.Error(w, "ID module invalide", http.StatusBadRequest)
			return
		}

		if err := smsManager.MarkAllRead(moduleID); err != nil {
			logger.Errorf("Erreur mark all SMS read module %d: %v", moduleID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "Tous les SMS marqués comme lus",
			"module_id": moduleID,
		})
	}
}

// GET /api/modules/{id}/sms/unread-count — Retourne le nombre de SMS non lus d'un module (MICRO-BLOC A2 v1-14)
func getUnreadSMSCountHandler(smsManager *sms.SMSManager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)
		if moduleID <= 0 {
			http.Error(w, "ID module invalide", http.StatusBadRequest)
			return
		}

		count, err := smsManager.GetUnreadCount(moduleID)
		if err != nil {
			logger.Errorf("Erreur GetUnreadCount module %d: %v", moduleID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"module_id":    moduleID,
			"unread_count": count,
		})
	}
}

func getAuditLogsHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// MICRO-BLOC C1 — Pagination + filtres
		pageSize := 50
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		if page < 1 {
			page = 1
		}
		action := r.URL.Query().Get("action")
		userID := r.URL.Query().Get("user")

		offset := (page - 1) * pageSize
		logs, err := dbConn.GetAuditLogs(pageSize, offset, action, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		total, _ := dbConn.GetAuditLogsCount(action, userID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
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

func statusCodesHandler(sm *serial.Manager, reader *excel.ExcelReader, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "ID invalide", http.StatusBadRequest)
			return
		}

		var carrier string
		if module, ok := sm.GetModuleByDBID(id); ok {
			carrier = module.Carrier
		} else {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		codes := reader.GetConsultCodes(carrier)
		result := make([]map[string]interface{}, 0, len(codes))
		for _, c := range codes {
			result = append(result, map[string]interface{}{
				"id":          c.ID,
				"carrier":     c.Carrier,
				"action":      c.Action,
				"target":      c.Target,
				"operation":   c.Operation,
				"ussd_code":   c.USSDCode,
				"info_input":  c.InformationINPUT,
				"info_output": c.InformationOUTPUT,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func menuCodesHandler(sm *serial.Manager, reader *excel.ExcelReader, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "ID invalide", http.StatusBadRequest)
			return
		}

		var carrier string
		if module, ok := sm.GetModuleByDBID(id); ok {
			carrier = module.Carrier
		} else {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		codes := reader.GetServiceNCodes(carrier)
		result := make([]map[string]interface{}, 0, len(codes))
		for _, c := range codes {
			result = append(result, map[string]interface{}{
				"id":          c.ID,
				"carrier":     c.Carrier,
				"action":      c.Action,
				"target":      c.Target,
				"operation":   c.Operation,
				"ussd_code":   c.USSDCode,
				"info_input":  c.InformationINPUT,
				"info_output": c.InformationOUTPUT,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
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

// ─── USSD Favorites handlers ────────────────────────────────────────────────

func getUSSDFavoritesHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		favs, err := dbConn.GetUSSDFavorites()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(favs)
	}
}

func addUSSDFavoriteHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			USSDCode  string `json:"ussd_code"`
			Operation string `json:"operation"`
			Carrier   string `json:"carrier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}
		if body.USSDCode == "" {
			http.Error(w, "ussd_code requis", http.StatusBadRequest)
			return
		}
		if err := dbConn.SaveUSSDFavorite(body.USSDCode, body.Operation, body.Carrier); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Favori ajouté"})
	}
}

func deleteUSSDFavoriteHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "ID invalide", http.StatusBadRequest)
			return
		}
		if err := dbConn.DeleteUSSDFavorite(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Favori supprimé"})
	}
}

// ─── USSD Navigate handler (interactive step-by-step) ──────────────────────
// POST /api/modules/{id}/ussd/navigate
// Body: { "choice": "1" }   — sends a menu choice in the ongoing USSD session
func navigateUSSDHandler(sm *serial.Manager, dbConn *db.DB, executor *ussd.USSDExecutor, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		moduleID, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "ID invalide", http.StatusBadRequest)
			return
		}

		var body struct {
			Choice string `json:"choice"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Choice == "" {
			http.Error(w, "Champ 'choice' requis", http.StatusBadRequest)
			return
		}

		var targetModule *serial.SIM800C
		if m, ok := sm.GetModuleByDBID(moduleID); ok {
			targetModule = m
		}
		if targetModule == nil {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		req := &ussd.USSDRequest{
			Module:   targetModule,
			ModuleID: moduleID,
		}
		resp, err := executor.ExecuteWithMenu(req, body.Choice)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save in history
		dbConn.SaveUSSDHistory(&db.USSDHistory{
			ModuleID:   targetModule.GetEffectiveID(),
			USSDCode:   fmt.Sprintf("CHOICE:%s", body.Choice),
			OutputData: resp.Result,
			Status:     "success",
			DurationMs: int(resp.Duration.Milliseconds()),
			ExecutedBy: r.RemoteAddr,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"result":   resp.Result,
			"duration": resp.Duration.Milliseconds(),
		})
	}
}

// ─── USSD History handler ────────────────────────────────────────────────────
// GET /api/ussd/history?module_id=N&limit=2000
// MICRO-BLOC B1 : si module_id absent ou 0 → retourne tous les modules
func getUSSDHistoryHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 2000 // Élevé pour pagination côté frontend
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		moduleID := 0 // 0 = tous les modules
		if m := r.URL.Query().Get("module_id"); m != "" {
			fmt.Sscanf(m, "%d", &moduleID)
		}
		history, err := dbConn.GetUSSDHistory(moduleID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if history == nil {
			history = []db.USSDHistory{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}

// ─── Signal Quality handler ───────────────────────────────────────────────────
// GET /api/modules/{id}/signal — refreshes CSQ and CREG in real time
func getModuleSignalHandler(sm *serial.Manager, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var id int
		fmt.Sscanf(vars["id"], "%d", &id)

		found, ok := sm.GetModuleByDBID(id)
		if !ok {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		csq, err := found.GetSignalQuality()
		if err != nil {
			csq = 99
		}
		netStatus := found.GetNetworkStatus()
		found.SignalQuality = csq
		found.NetworkStatus = netStatus

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"module_id":      id,
			"signal_quality": csq,
			"signal_rssi":    serial.CSQToRSSI(csq),
			"network_status": netStatus,
		})
	}
}

// ─── Per-module Auto-Status handler ──────────────────────────────────────────
// POST /api/modules/{id}/ussd/auto-status — exécute les codes Consulter/Interne sur UN seul module
func moduleAutoStatusHandler(sm *serial.Manager, reader *excel.ExcelReader, executor *ussd.USSDExecutor, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		// Try DBID first, then ModuleID fallback (via GetModuleByDBID)
		targetModule, ok := sm.GetModuleByDBID(moduleID)
		if !ok {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		moduleResults := make(map[string]string)
		codes := reader.GetConsultCodes(targetModule.Carrier)

		for _, code := range codes {
			req := &ussd.USSDRequest{
				Module:   targetModule,
				Code:     code.USSDCode,
				ModuleID: targetModule.ModuleID,
			}
			response, err := executor.Execute(req)
			if err != nil {
				moduleResults[code.Operation] = "Erreur: " + err.Error()
			} else {
				moduleResults[code.Operation] = response.Result
			}
			if hub != nil {
				hub.BroadcastEvent(websocket.Event{
					Type:     "auto_status_progress",
					ModuleID: targetModule.ModuleID,
					Data: map[string]interface{}{
						"port":      targetModule.Port,
						"operation": code.Operation,
						"ussd_code": code.USSDCode,
						"result":    moduleResults[code.Operation],
					},
					Timestamp: time.Now(),
				})
			}
			time.Sleep(1000 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"module_id": moduleID,
			"port":      targetModule.Port,
			"results":   moduleResults,
		})
	}
}

// ─── Recent USSD codes handler ────────────────────────────────────────────────
// GET /api/modules/{id}/ussd/recent?limit=5 (MICRO-BLOC B1)
// Retourne les N derniers codes USSD distincts exécutés sur ce module
func getRecentUSSDCodesHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		limit := 5
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
			if limit < 1 {
				limit = 1
			}
			if limit > 20 {
				limit = 20
			}
		}

		codes, err := dbConn.GetRecentUSSDCodes(moduleID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"module_id": moduleID,
			"codes":     codes,
		})
	}
}

// ─── Export USSD History CSV handler ─────────────────────────────────────────
// GET /api/ussd/history/export?module_id=N&limit=1000
func exportUSSDHistoryCSVHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		moduleID := 0
		limit := 1000
		if m := r.URL.Query().Get("module_id"); m != "" {
			fmt.Sscanf(m, "%d", &moduleID)
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}

		history, err := dbConn.GetUSSDHistory(moduleID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("ussd_history_%s.csv", time.Now().Format("20060102_150405"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		// UTF-8 BOM for Excel compatibility
		w.Write([]byte("\xEF\xBB\xBF"))
		// Header row
		fmt.Fprintf(w, "ID,Module_ID,USSD_Code,Input_Data,Output_Data,Status,Duration_ms,Executed_By,Executed_At\r\n")
		for _, h := range history {
			// Escape double quotes in fields
			outputData := fmt.Sprintf("%v", h.OutputData)
			outputData = escapeCSV(outputData)
			inputData := escapeCSV(fmt.Sprintf("%v", h.InputData))
			fmt.Fprintf(w, "%d,%d,%s,%s,%s,%s,%d,%s,%s\r\n",
				h.ID,
				h.ModuleID,
				escapeCSV(h.USSDCode),
				inputData,
				outputData,
				h.Status,
				h.DurationMs,
				escapeCSV(h.ExecutedBy),
				h.ExecutedAt.Format("2006-01-02 15:04:05"),
			)
		}
	}
}

// ─── System Status handler ────────────────────────────────────────────────────
// GET /api/system/status — état global de l'application (uptime, modules, DB, version)
func systemStatusHandler(sm *serial.Manager, dbConn *db.DB, cfg *config.Config, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(startupTime)
		uptimeStr := fmt.Sprintf("%dd %02dh %02dm %02ds",
			int(uptime.Hours())/24,
			int(uptime.Hours())%24,
			int(uptime.Minutes())%60,
			int(uptime.Seconds())%60,
		)

		// Modules
		allModules := sm.GetAllModules()
		modulesSummary := make([]map[string]interface{}, 0, len(allModules))
		connectedCount := 0
		pinFailedCount := 0
		for _, m := range allModules {
			connectedCount++
			if m.PINFailed {
				pinFailedCount++
			}
			modulesSummary = append(modulesSummary, map[string]interface{}{
				"port":           m.Port,
				"carrier":        m.Carrier,
				"phone_number":   m.PhoneNumber,
				"signal_quality": m.SignalQuality,
				"network_status": m.NetworkStatus,
				"pin_unlocked":   m.PINUnlocked,
				"pin_failed":     m.PINFailed,
				"db_id":          m.DBID,
			})
		}

		// DB ping
		dbOk := true
		dbErr := ""
		if err := dbConn.Ping(); err != nil {
			dbOk = false
			dbErr = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":          "2.0",
			"startup_time":     startupTime.Format(time.RFC3339),
			"uptime":           uptimeStr,
			"uptime_seconds":   int(uptime.Seconds()),
			"modules_total":    connectedCount,
			"modules_pin_ok":   connectedCount - pinFailedCount,
			"modules_pin_fail": pinFailedCount,
			"modules":          modulesSummary,
			"database": map[string]interface{}{
				"ok":       dbOk,
				"error":    dbErr,
				"host":     cfg.MySQL.Host,
				"database": cfg.MySQL.Database,
			},
			"config": map[string]interface{}{
				"explore_delay_ms": cfg.USSD.ExploreDelayMs,
				"nav_delay_ms":     cfg.USSD.NavDelayMs,
				"max_menu_depth":   cfg.USSD.MaxMenuDepth,
				"auto_trash_kw":    cfg.SMS.AutoTrashKeyword,
			},
			"server_time": time.Now().Format(time.RFC3339),
		})
	}
}

func escapeCSV(s string) string {
	if len(s) == 0 {
		return ""
	}
	// Wrap in quotes if contains comma, newline, or quote
	needsQuote := false
	for _, c := range s {
		if c == ',' || c == '\n' || c == '\r' || c == '"' {
			needsQuote = true
			break
		}
	}
	if needsQuote {
		// Escape internal double quotes by doubling them
		escaped := ""
		for _, c := range s {
			if c == '"' {
				escaped += "\"\""
			} else {
				escaped += string(c)
			}
		}
		return "\"" + escaped + "\""
	}
	return s
}

// ─── Dial Plan handler ────────────────────────────────────────────────────────
// GET /api/dialplan — retourne le plan de numérotation depuis la DB
func getDialPlanHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dialPlans, err := dbConn.GetDialPlan()
		if err != nil {
			logger.Warnf("Erreur récupération dial plan: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dialPlans)
	}
}

// POST /api/dialplan — ajouter une entrée
func createDialPlanHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entry db.DialPlan
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}
		if entry.CountryCode == "" || entry.Operator == "" || entry.Prefix == "" {
			http.Error(w, "country_code, operator et prefix sont requis", http.StatusBadRequest)
			return
		}
		if err := dbConn.CreateDialPlanEntry(&entry); err != nil {
			logger.Warnf("Erreur création dial plan: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(entry)
	}
}

// PUT /api/dialplan/{id} — modifier une entrée
func updateDialPlanHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var id int
		fmt.Sscanf(vars["id"], "%d", &id)
		var entry db.DialPlan
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}
		entry.ID = id
		if err := dbConn.UpdateDialPlanEntry(&entry); err != nil {
			logger.Warnf("Erreur mise à jour dial plan: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}
}

// DELETE /api/dialplan/{id} — supprimer une entrée
func deleteDialPlanHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var id int
		fmt.Sscanf(vars["id"], "%d", &id)
		if err := dbConn.DeleteDialPlanEntry(id); err != nil {
			logger.Warnf("Erreur suppression dial plan: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// POST /api/modules/{id}/ussd/auto-menu (par module)
func moduleAutoMenuHandler(sm *serial.Manager, reader *excel.ExcelReader, explorer *ussd.USSDExplorer, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		// Try DBID first, then ModuleID fallback (via GetModuleByDBID)
		targetModule, ok := sm.GetModuleByDBID(moduleID)
		if !ok {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		menuCodes := reader.GetServiceNCodes(targetModule.Carrier)
		allResults := make(map[string]interface{})

		for _, code := range menuCodes {
			result, err := explorer.ExploreMenu(targetModule, code.USSDCode, code.ID)
			entry := map[string]interface{}{
				"operation": code.Operation,
				"ussd_code": code.USSDCode,
			}
			if err != nil {
				entry["error"] = err.Error()
				entry["menu"] = ""
				entry["new_codes"] = []string{}
			} else if result != nil {
				entry["menu"] = explorer.FormatMenuTree(result.MenuTree, 0)
				newCodesList := make([]string, 0, len(result.DiscoveredCodes))
				for _, nc := range result.DiscoveredCodes {
					newCodesList = append(newCodesList, nc.USSDCode)
				}
				entry["new_codes"] = newCodesList
			}
			allResults[code.USSDCode] = entry

			discoveredCount := 0
			menuText := ""
			if v, ok := entry["menu"]; ok {
				menuText, _ = v.(string)
			}
			if v, ok := entry["new_codes"]; ok {
				if sl, ok2 := v.([]string); ok2 {
					discoveredCount = len(sl)
				}
			}

			if hub != nil {
				hub.BroadcastEvent(websocket.Event{
					Type:     "auto_menu_progress",
					ModuleID: targetModule.ModuleID,
					Data: map[string]interface{}{
						"port":            targetModule.Port,
						"operation":       code.Operation,
						"ussd_code":       code.USSDCode,
						"result":          menuText,
						"new_codes_count": discoveredCount,
					},
					Timestamp: time.Now(),
				})
			}
			time.Sleep(2000 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"module_id": moduleID,
			"port":      targetModule.Port,
			"results":   allResults,
		})
	}
}

// GET /api/modules/{id}/sms/export — export SMS en CSV
func exportSMSCSVHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)

		messages, err := dbConn.GetSMSMessages(moduleID, 2000)
		if err != nil {
			logger.Warnf("Erreur récupération SMS pour export: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("sms_module%d_%s.csv", moduleID, time.Now().Format("20060102_150405"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write([]byte("\xef\xbb\xbf"))
		w.Write([]byte("ID,Module_ID,Direction,Sender,Receiver,Message,Is_Trash,Received_At\n"))
		for _, msg := range messages {
			line := fmt.Sprintf("%d,%d,%s,%s,%s,%s,%v,%s\n",
				msg.ID,
				msg.ModuleID,
				escapeCSV(msg.Direction),
				escapeCSV(msg.SenderNumber),
				escapeCSV(msg.ReceiverNumber),
				escapeCSV(msg.Message),
				msg.IsTrash,
				msg.ReceivedAt.Format("2006-01-02 15:04:05"),
			)
			w.Write([]byte(line))
		}
	}
}

// POST /api/dialplan/reload — recharge le plan de numérotation depuis la DB
// et le propage immédiatement à tous les modules connectés.
func reloadDialPlanHandler(dbConn *db.DB, sm *serial.Manager, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := dbConn.GetDialPlan()
		if err != nil {
			logger.Errorf("Erreur reload dial plan: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert db.DialPlan → serial.DialPlanEntry
		plan := make([]serial.DialPlanEntry, 0, len(entries))
		for _, e := range entries {
			plan = append(plan, serial.DialPlanEntry{
				Operator:     e.Operator,
				Prefix:       e.Prefix,
				CallingCode:  e.CallingCode,
				NumberLength: e.NumberLength,
			})
		}

		sm.ReloadDialPlan(plan)
		logger.Infof("Dial plan rechargé manuellement: %d entrées", len(plan))

		// Broadcast aux clients WebSocket connectés
		if hub != nil {
			hub.BroadcastEvent(websocket.Event{
				Type: "dialplan_reloaded",
				Data: map[string]interface{}{
					"count":   len(plan),
					"message": fmt.Sprintf("Plan de numérotation rechargé: %d entrées", len(plan)),
				},
				Timestamp: time.Now(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"count":   len(plan),
			"message": fmt.Sprintf("Plan de numérotation rechargé: %d entrées propagées", len(plan)),
		})
	}
}

// GET /api/config — retourne la configuration courante (sans secrets)
func getConfigHandler(cfg *config.Config, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		safe := map[string]interface{}{
			"server": map[string]interface{}{
				"port":           cfg.Server.Port,
				"websocket_path": cfg.Server.WebsocketPath,
			},
			"serial": map[string]interface{}{
				"ports":     cfg.Serial.Ports,
				"baud_rate": cfg.Serial.BaudRate,
			},
			"mysql": map[string]interface{}{
				"host":     cfg.MySQL.Host,
				"port":     cfg.MySQL.Port,
				"database": cfg.MySQL.Database,
			},
			"ussd": map[string]interface{}{
				"explore_delay_ms":               cfg.USSD.ExploreDelayMs,
				"nav_delay_ms":                   cfg.USSD.NavDelayMs,
				"max_menu_depth":                 cfg.USSD.MaxMenuDepth,
				"session_timeout_seconds":        cfg.USSD.SessionTimeoutSeconds,
				"default_choice_timeout_seconds": cfg.USSD.DefaultChoiceTimeoutSeconds,
			},
			"sms": map[string]interface{}{
				"auto_trash_keyword":     cfg.SMS.AutoTrashKeyword,
				"max_sms_per_module":     cfg.SMS.MaxSMSPerModule,
				"check_interval_seconds": cfg.SMS.CheckIntervalSeconds,
			},
			"monitoring": map[string]interface{}{
				"check_interval_seconds": cfg.Monitoring.CheckIntervalSeconds,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(safe)
	}
}

// PUT /api/config/delays — met à jour les délais USSD en mémoire ET en DB (persistant)
func updateDelaysHandler(cfg *config.Config, dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ExploreDelayMs int `json:"explore_delay_ms"`
			NavDelayMs     int `json:"nav_delay_ms"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}
		if payload.ExploreDelayMs < 500 {
			http.Error(w, "explore_delay_ms minimum 500ms", http.StatusBadRequest)
			return
		}
		if payload.NavDelayMs < 100 {
			http.Error(w, "nav_delay_ms minimum 100ms", http.StatusBadRequest)
			return
		}
		cfg.USSD.ExploreDelayMs = payload.ExploreDelayMs
		cfg.USSD.NavDelayMs = payload.NavDelayMs

		// Persistance en DB
		dbConn.SetSetting("ussd.explore_delay_ms", fmt.Sprintf("%d", payload.ExploreDelayMs))
		dbConn.SetSetting("ussd.nav_delay_ms", fmt.Sprintf("%d", payload.NavDelayMs))

		// Audit log
		dbConn.SaveAuditLog("system", "config_update_delays",
			"config", 0,
			map[string]interface{}{
				"explore_delay_ms": payload.ExploreDelayMs,
				"nav_delay_ms":     payload.NavDelayMs,
			},
			r.RemoteAddr,
		)

		logger.Infof("Délais USSD mis à jour et persistés: explore=%dms, nav=%dms", payload.ExploreDelayMs, payload.NavDelayMs)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":          true,
			"explore_delay_ms": cfg.USSD.ExploreDelayMs,
			"nav_delay_ms":     cfg.USSD.NavDelayMs,
		})
	}
}

// GET /api/config/advanced — paramètres avancés (auto_trash_keyword, retry_on_error, etc.)
func getAdvancedSettingsHandler(cfg *config.Config, dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, _ := dbConn.GetAllSettings()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"auto_trash_keyword":     cfg.SMS.AutoTrashKeyword,
			"retry_on_error":         cfg.USSD.RetryOnError,
			"max_retries":            cfg.USSD.MaxRetries,
			"max_menu_depth":         cfg.USSD.MaxMenuDepth,
			"check_interval_seconds": cfg.SMS.CheckIntervalSeconds,
			"persisted":              settings,
		})
	}
}

// PUT /api/config/advanced — met à jour les paramètres avancés + persiste en DB
func updateAdvancedSettingsHandler(cfg *config.Config, dbConn *db.DB, smsManager *sms.SMSManager, ussdExp *ussd.USSDExplorer, hub *websocket.Hub, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			AutoTrashKeyword string `json:"auto_trash_keyword"`
			RetryOnError     *bool  `json:"retry_on_error"`
			MaxRetries       int    `json:"max_retries"`
			MaxMenuDepth     int    `json:"max_menu_depth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		changed := []string{}

		if payload.AutoTrashKeyword != "" && payload.AutoTrashKeyword != cfg.SMS.AutoTrashKeyword {
			cfg.SMS.AutoTrashKeyword = payload.AutoTrashKeyword
			smsManager.UpdateAutoTrashKeyword(payload.AutoTrashKeyword)
			dbConn.SetSetting("sms.auto_trash_keyword", payload.AutoTrashKeyword)
			changed = append(changed, "auto_trash_keyword")
			logger.Infof("Mot-clé corbeille SMS mis à jour: %q", payload.AutoTrashKeyword)
		}
		if payload.RetryOnError != nil {
			cfg.USSD.RetryOnError = *payload.RetryOnError
			dbConn.SetSetting("ussd.retry_on_error", fmt.Sprintf("%v", *payload.RetryOnError))
			changed = append(changed, "retry_on_error")
		}
		if payload.MaxRetries > 0 {
			cfg.USSD.MaxRetries = payload.MaxRetries
			dbConn.SetSetting("ussd.max_retries", fmt.Sprintf("%d", payload.MaxRetries))
			changed = append(changed, "max_retries")
		}
		if payload.MaxMenuDepth > 0 && payload.MaxMenuDepth != cfg.USSD.MaxMenuDepth {
			cfg.USSD.MaxMenuDepth = payload.MaxMenuDepth
			ussdExp.SetMaxDepth(payload.MaxMenuDepth) // propagation immédiate à l'instance existante
			dbConn.SetSetting("ussd.max_menu_depth", fmt.Sprintf("%d", payload.MaxMenuDepth))
			changed = append(changed, "max_menu_depth")
			logger.Infof("max_menu_depth mis à jour et propagé: %d", payload.MaxMenuDepth)
		}

		// Broadcast WebSocket aux clients connectés
		if hub != nil && len(changed) > 0 {
			hub.BroadcastEvent(websocket.Event{
				Type: "config_updated",
				Data: map[string]interface{}{
					"changed":            changed,
					"auto_trash_keyword": cfg.SMS.AutoTrashKeyword,
					"retry_on_error":     cfg.USSD.RetryOnError,
					"max_retries":        cfg.USSD.MaxRetries,
					"max_menu_depth":     cfg.USSD.MaxMenuDepth,
					"message":            fmt.Sprintf("Configuration mise à jour: %s", strings.Join(changed, ", ")),
				},
				Timestamp: time.Now(),
			})
		}

		// Audit log
		if len(changed) > 0 {
			dbConn.SaveAuditLog("system", "config_update_advanced",
				"config", 0,
				map[string]interface{}{
					"changed":            changed,
					"auto_trash_keyword": cfg.SMS.AutoTrashKeyword,
					"max_menu_depth":     cfg.USSD.MaxMenuDepth,
					"retry_on_error":     cfg.USSD.RetryOnError,
					"max_retries":        cfg.USSD.MaxRetries,
				},
				r.RemoteAddr,
			)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":            true,
			"changed":            changed,
			"auto_trash_keyword": cfg.SMS.AutoTrashKeyword,
			"retry_on_error":     cfg.USSD.RetryOnError,
			"max_retries":        cfg.USSD.MaxRetries,
			"max_menu_depth":     cfg.USSD.MaxMenuDepth,
		})
	}
}

// GET /api/config/ports — retourne la whitelist des ports COM
func getPortsWhitelistHandler(cfg *config.Config, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ports": cfg.Serial.Ports,
		})
	}
}

// PUT /api/config/ports — met à jour la whitelist des ports COM
// Body: { "ports": ["COM5", "COM6"] }  ou  { "ports_csv": "COM5, COM6" }
func updatePortsWhitelistHandler(cfg *config.Config, dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Ports    []string `json:"ports"`
			PortsCSV string   `json:"ports_csv"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		// Accepter soit un tableau soit une chaîne CSV
		ports := payload.Ports
		if len(ports) == 0 && payload.PortsCSV != "" {
			for _, p := range strings.Split(payload.PortsCSV, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					ports = append(ports, p)
				}
			}
		}

		cfg.Serial.Ports = ports
		// Persister en DB (CSV)
		dbConn.SetSetting("serial.ports_whitelist", strings.Join(ports, ","))
		logger.Infof("Whitelist ports COM mise à jour: %v", ports)

		// Audit log
		dbConn.SaveAuditLog("system", "config_update_ports_whitelist",
			"config", 0,
			map[string]interface{}{"ports": ports},
			r.RemoteAddr,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"ports":   ports,
			"message": fmt.Sprintf("%d port(s) enregistré(s). Effectif au prochain démarrage ou scan.", len(ports)),
		})
	}
}

// ─── Dial Plan Export CSV handler ───────────────────────────────────────────
// GET /api/dialplan/export — export du plan de numérotation en CSV
func exportDialPlanCSVHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := dbConn.GetDialPlan()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("dialplan_%s.csv", time.Now().Format("20060102_150405"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		// UTF-8 BOM for Excel compatibility
		w.Write([]byte("\xEF\xBB\xBF"))
		fmt.Fprintf(w, "ID,Country_Code,Country_Name,Calling_Code,Operator,Prefix,Number_Length,Is_Active\r\n")
		for _, e := range entries {
			active := "0"
			if e.IsActive {
				active = "1"
			}
			fmt.Fprintf(w, "%d,%s,%s,%s,%s,%s,%d,%s\r\n",
				e.ID,
				escapeCSV(e.CountryCode),
				escapeCSV(e.CountryName),
				escapeCSV(e.CallingCode),
				escapeCSV(e.Operator),
				escapeCSV(e.Prefix),
				e.NumberLength,
				active,
			)
		}
	}
}

// GET /api/sms/export — export SMS de TOUS les modules en CSV (MICRO-BLOC C3)
func exportAllSMSCSVHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messages, err := dbConn.GetAllSMS(10000)
		if err != nil {
			logger.Warnf("Erreur récupération SMS pour export global: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filename := fmt.Sprintf("sms_tous_modules_%s.csv", time.Now().Format("20060102_150405"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write([]byte("\xef\xbb\xbf"))
		w.Write([]byte("ID,Module_ID,Direction,Sender,Receiver,Message,Is_Trash,Is_Read,Received_At\n"))
		for _, msg := range messages {
			line := fmt.Sprintf("%d,%d,%s,%s,%s,%s,%v,%v,%s\n",
				msg.ID,
				msg.ModuleID,
				escapeCSV(msg.Direction),
				escapeCSV(msg.SenderNumber),
				escapeCSV(msg.ReceiverNumber),
				escapeCSV(msg.Message),
				msg.IsTrash,
				msg.IsRead,
				msg.ReceivedAt.Format("2006-01-02 15:04:05"),
			)
			w.Write([]byte(line))
		}
	}
}

// GET /api/modules/{id}/signal/history — historique signal d'un module (MICRO-BLOC C6)
func getSignalHistoryHandler(dbConn *db.DB, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var moduleID int
		fmt.Sscanf(vars["id"], "%d", &moduleID)
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 200 {
			limit = 200
		}
		history, err := dbConn.GetSignalHistory(moduleID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}
