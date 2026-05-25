package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time" // Ajoutez cette ligne

	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/serial"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ModuleHandler struct {
	serialManager *serial.Manager
	db            *db.DB
	logger        *logrus.Logger
}

func NewModuleHandler(serialManager *serial.Manager, db *db.DB, logger *logrus.Logger) *ModuleHandler {
	return &ModuleHandler{
		serialManager: serialManager,
		db:            db,
		logger:        logger,
	}
}

func (h *ModuleHandler) GetModules(w http.ResponseWriter, r *http.Request) {
	modules := h.serialManager.GetAllModules()

	response := make([]map[string]interface{}, 0, len(modules))
	for _, module := range modules {
		response = append(response, map[string]interface{}{
			"port":         module.Port,
			"module_id":    module.ModuleID,
			"imei":         module.IMEI,
			"phone_number": module.PhoneNumber,
			"carrier":      module.Carrier,
			"status":       "connected",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ModuleHandler) GetModule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	// Trouver le module par ID
	modules := h.serialManager.GetAllModules()
	for _, module := range modules {
		if module.ModuleID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"port":         module.Port,
				"module_id":    module.ModuleID,
				"imei":         module.IMEI,
				"phone_number": module.PhoneNumber,
				"carrier":      module.Carrier,
			})
			return
		}
	}

	http.Error(w, "Module non trouvé", http.StatusNotFound)
}

func (h *ModuleHandler) DiscoverModules(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Découverte des modules demandée")

	// Force la rediscovery
	modules := h.serialManager.GetAllModules()

	response := map[string]interface{}{
		"status":  "completed",
		"modules": len(modules),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
