package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/excel"
	"sim800c-supervisor/internal/serial"
	"sim800c-supervisor/internal/ussd"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type USSDHandler struct {
	serialManager *serial.Manager
	db            *db.DB
	cfg           *config.Config
	logger        *logrus.Logger
	executor      *ussd.USSDExecutor
	explorer      *ussd.USSDExplorer
	excelReader   *excel.ExcelReader
}

func NewUSSDHandler(serialManager *serial.Manager, db *db.DB, cfg *config.Config, logger *logrus.Logger) *USSDHandler {
	executor := ussd.NewUSSDExecutor(logger)
	excelReader := excel.NewExcelReader(cfg.Excel.BasePath, cfg.Excel.FilenamePattern, logger)
	excelWriter := excel.NewExcelWriter(cfg.Excel.BasePath, logger)
	explorer := ussd.NewUSSDExplorer(executor, excelReader, excelWriter, logger, cfg.USSD.MaxMenuDepth)

	return &USSDHandler{
		serialManager: serialManager,
		db:            db,
		cfg:           cfg,
		logger:        logger,
		executor:      executor,
		explorer:      explorer,
		excelReader:   excelReader,
	}
}

type ExecuteUSSDRequest struct {
	ModuleID  int    `json:"module_id"`
	USSDCode  string `json:"ussd_code"`
	InputData string `json:"input_data"`
}

func (h *USSDHandler) ExecuteUSSD(w http.ResponseWriter, r *http.Request) {
	var req ExecuteUSSDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// Trouver le module
	var targetModule *serial.SIM800C
	for _, module := range h.serialManager.GetAllModules() {
		if module.ModuleID == req.ModuleID {
			targetModule = module
			break
		}
	}

	if targetModule == nil {
		http.Error(w, "Module non trouvé", http.StatusNotFound)
		return
	}

	// Exécuter l'USSD
	ussdReq := &ussd.USSDRequest{
		Module:    targetModule,
		Code:      req.USSDCode,
		InputData: req.InputData,
		ModuleID:  req.ModuleID,
	}

	startTime := time.Now()
	response, err := h.executor.Execute(ussdReq)
	duration := time.Since(startTime)

	if err != nil {
		// Sauvegarder l'historique d'erreur
		history := &db.USSDHistory{
			ModuleID:   req.ModuleID,
			USSSCode:   req.USSDCode,
			InputData:  req.InputData,
			OutputData: response.Error,
			Status:     "error",
			DurationMs: int(duration.Milliseconds()),
			ExecutedBy: r.RemoteAddr,
		}
		h.db.SaveUSSDHistory(history)

		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	// Sauvegarder l'historique
	history := &db.USSDHistory{
		ModuleID:   req.ModuleID,
		USSSCode:   req.USSDCode,
		InputData:  req.InputData,
		OutputData: response.Result,
		Status:     "success",
		DurationMs: int(response.Duration.Milliseconds()),
		ExecutedBy: r.RemoteAddr,
	}
	h.db.SaveUSSDHistory(history)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"result":   response.Result,
		"duration": response.Duration.Milliseconds(),
	})
}

func (h *USSDHandler) AutoStatusDiscovery(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Démarrage de SIM Status Auto-Discovery")

	modules := h.serialManager.GetAllModules()
	results := make(map[int]map[string]string)

	for _, module := range modules {
		moduleResults := make(map[string]string)

		// Charger les codes USSD de consultation pour cet opérateur
		codes := h.excelReader.GetConsultCodes(module.Carrier)

		for _, code := range codes {
			req := &ussd.USSDRequest{
				Module:   module,
				Code:     code.USSDCode,
				ModuleID: module.ModuleID,
			}

			response, err := h.executor.Execute(req)
			if err != nil {
				moduleResults[code.Operation] = "Erreur: " + err.Error()
			} else {
				moduleResults[code.Operation] = response.Result
			}

			// Petit délai entre les requêtes
			time.Sleep(time.Duration(h.cfg.USSD.ExploreDelayMs) * time.Millisecond)
		}

		results[module.ModuleID] = moduleResults
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *USSDHandler) AutoMenuDiscovery(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Démarrage de USSD Menu Auto-Discovery")

	modules := h.serialManager.GetAllModules()
	results := make(map[int]interface{})

	for _, module := range modules {
		codes := h.excelReader.GetServiceNCodes(module.Carrier)

		moduleResults := make(map[string]interface{})

		for _, code := range codes {
			explorationResult, err := h.explorer.ExploreMenu(module, code.USSDCode, code.ID)
			if err != nil {
				moduleResults[code.Operation] = map[string]interface{}{
					"error": err.Error(),
				}
			} else {
				moduleResults[code.Operation] = map[string]interface{}{
					"discovered_codes": len(explorationResult.DiscoveredCodes),
					"menu_tree":        h.explorer.FormatMenuTree(explorationResult.MenuTree, 0),
				}
			}

			time.Sleep(time.Duration(h.cfg.USSD.ExploreDelayMs) * time.Millisecond)
		}

		results[module.ModuleID] = moduleResults
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetStatusCodes returns USSD codes for Action=Consulter, Target=Interne, Scope=In for a module's carrier
func (h *USSDHandler) GetStatusCodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	var carrier string
	for _, module := range h.serialManager.GetAllModules() {
		if module.ModuleID == id {
			carrier = module.Carrier
			break
		}
	}

	codes := h.excelReader.GetConsultCodes(carrier)
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

// GetMenuCodes returns USSD codes for Action=Services_N1, Target=Interne, Scope=In for a module's carrier
func (h *USSDHandler) GetMenuCodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	var carrier string
	for _, module := range h.serialManager.GetAllModules() {
		if module.ModuleID == id {
			carrier = module.Carrier
			break
		}
	}

	codes := h.excelReader.GetServiceNCodes(carrier)
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
