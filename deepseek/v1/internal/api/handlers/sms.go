package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/serial"
	"sim800c-supervisor/internal/sms"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SMSHandler struct {
	serialManager *serial.Manager
	db            *db.DB
	logger        *logrus.Logger
	smsManager    *sms.SMSManager
}

func NewSMSHandler(serialManager *serial.Manager, dbConn *db.DB, logger *logrus.Logger) *SMSHandler {
	handler := &SMSHandler{
		serialManager: serialManager,
		db:            dbConn,
		logger:        logger,
	}

	// Initialiser le SMS Manager
	handler.smsManager = sms.NewSMSManager(logger, nil, dbConn, "Test")

	return handler
}

type SendSMSRequest struct {
	Number  string `json:"number"`
	Message string `json:"message"`
}

func (h *SMSHandler) SendSMS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	moduleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	var req SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// Trouver le module
	var targetModule *serial.SIM800C
	for _, module := range h.serialManager.GetAllModules() {
		if module.ModuleID == moduleID {
			targetModule = module
			break
		}
	}

	if targetModule == nil {
		http.Error(w, "Module non trouvé", http.StatusNotFound)
		return
	}

	if err := h.smsManager.SendSMS(targetModule, req.Number, req.Message); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "SMS envoyé avec succès",
	})
}

func (h *SMSHandler) GetSMS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	moduleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	includeTrash := r.URL.Query().Get("include_trash") == "true"

	smsList, err := h.smsManager.GetSMS(moduleID, includeTrash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(smsList)
}

func (h *SMSHandler) DeleteSMS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	indexStr := vars["index"]

	moduleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "Index invalide", http.StatusBadRequest)
		return
	}

	// Trouver le module
	var targetModule *serial.SIM800C
	for _, module := range h.serialManager.GetAllModules() {
		if module.ModuleID == moduleID {
			targetModule = module
			break
		}
	}

	if targetModule == nil {
		http.Error(w, "Module non trouvé", http.StatusNotFound)
		return
	}

	if err := h.smsManager.DeleteSMS(targetModule, index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "SMS supprimé avec succès",
	})
}

func (h *SMSHandler) MoveToTrash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	smsID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID invalide", http.StatusBadRequest)
		return
	}

	if err := h.smsManager.MoveToTrash(nil, smsID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "SMS déplacé vers la corbeille",
	})
}
