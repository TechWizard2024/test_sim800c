package sms

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"sim800c-supervisor/internal/db"
	"sim800c-supervisor/internal/serial"
	"sim800c-supervisor/internal/websocket"

	"github.com/sirupsen/logrus"
)

type SMSManager struct {
	logger           *logrus.Logger
	hub              *websocket.Hub
	db               *db.DB
	autoTrashKeyword string
	mu               sync.RWMutex
}

type SMS struct {
	ID             int       `json:"id"`
	ModuleID       int       `json:"module_id"`
	SenderNumber   string    `json:"sender_number"`
	ReceiverNumber string    `json:"receiver_number"`
	Message        string    `json:"message"`
	Direction      string    `json:"direction"`
	IsDeleted      bool      `json:"is_deleted"`
	IsTrash        bool      `json:"is_trash"`
	SMSIndex       int       `json:"sms_index"`
	ReceivedAt     time.Time `json:"received_at"`
}

func NewSMSManager(logger *logrus.Logger, hub *websocket.Hub, dbConn *db.DB, autoTrashKeyword string) *SMSManager {
	return &SMSManager{
		logger:           logger,
		hub:              hub,
		db:               dbConn,
		autoTrashKeyword: autoTrashKeyword,
	}
}

func (m *SMSManager) SendSMS(moduleID int, number, message string) error {
	m.logger.Infof("Envoi SMS du module %d à %s", moduleID, number)

	// Validation du numéro
	if err := m.validateNumber(number); err != nil {
		return err
	}

	// Sauvegarder dans la base
	sms := &db.SMSMessage{
		ModuleID:       moduleID,
		ReceiverNumber: number,
		Message:        message,
		Direction:      "out",
		ReceivedAt:     time.Now(),
	}

	if err := m.db.SaveSMS(sms); err != nil {
		m.logger.Warnf("Erreur sauvegarde SMS: %v", err)
	}

	// Notifier via WebSocket
	if m.hub != nil {
		m.hub.BroadcastEvent(websocket.Event{
			Type:      "sms_sent",
			ModuleID:  moduleID,
			Data:      sms,
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (m *SMSManager) ReadSMS(module *serial.SIM800C) error {
	m.logger.Infof("Lecture SMS du module %s", module.Port)

	smsList, err := module.ListSMS()
	if err != nil {
		return fmt.Errorf("erreur lecture SMS: %w", err)
	}

	for _, smsInfo := range smsList {
		index, ok := smsInfo["index"].(string)
		if !ok {
			continue
		}

		var idx int
		fmt.Sscanf(index, "%d", &idx)

		sender, message, err := module.ReadSMS(idx)
		if err != nil {
			m.logger.Warnf("Erreur lecture SMS index %d: %v", idx, err)
			continue
		}

		// Vérifier si déjà en base
		exists, _ := m.db.SMSExists(module.ModuleID, idx)
		if exists {
			continue
		}

		isTrash := !strings.Contains(message, m.autoTrashKeyword)

		sms := &db.SMSMessage{
			ModuleID:     module.ModuleID,
			SenderNumber: sender,
			Message:      message,
			Direction:    "in",
			IsTrash:      isTrash,
			SMSIndex:     idx,
			ReceivedAt:   time.Now(),
		}

		if err := m.db.SaveSMS(sms); err != nil {
			m.logger.Warnf("Erreur sauvegarde SMS: %v", err)
		}

		// Notifier via WebSocket
		if m.hub != nil {
			m.hub.BroadcastEvent(websocket.Event{
				Type:      "sms_received",
				ModuleID:  module.ModuleID,
				Data:      sms,
				Timestamp: time.Now(),
			})
		}

		m.logger.Infof("SMS reçu de %s: %s", sender, message[:min(50, len(message))])
	}

	return nil
}

func (m *SMSManager) DeleteSMS(moduleID int, index int) error {
	m.logger.Infof("Suppression SMS index %d du module %d", index, moduleID)

	if err := m.db.MarkSMSDeleted(moduleID, index); err != nil {
		return fmt.Errorf("erreur suppression SMS: %w", err)
	}

	if m.hub != nil {
		m.hub.BroadcastEvent(websocket.Event{
			Type:      "sms_deleted",
			ModuleID:  moduleID,
			Data:      map[string]interface{}{"index": index},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (m *SMSManager) MoveToTrash(smsID int) error {
	m.logger.Infof("Déplacement SMS %d vers corbeille", smsID)

	if err := m.db.MoveSMSToTrash(smsID); err != nil {
		return fmt.Errorf("erreur déplacement vers corbeille: %w", err)
	}

	if m.hub != nil {
		m.hub.BroadcastEvent(websocket.Event{
			Type:      "sms_moved_to_trash",
			Data:      map[string]interface{}{"sms_id": smsID},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (m *SMSManager) GetSMS(moduleID int, includeTrash bool) ([]db.SMSMessage, error) {
	return m.db.GetSMSByModule(moduleID, includeTrash)
}

func (m *SMSManager) AutoFilterTrash(module *serial.SIM800C) error {
	m.logger.Infof("Filtrage automatique des SMS pour module %s", module.Port)

	smsList, err := m.db.GetSMSByModule(module.ModuleID, false)
	if err != nil {
		return err
	}

	for _, sms := range smsList {
		if sms.Direction == "in" && !strings.Contains(sms.Message, m.autoTrashKeyword) && !sms.IsTrash {
			if err := m.MoveToTrash(sms.ID); err != nil {
				m.logger.Warnf("Erreur filtrage SMS %d: %v", sms.ID, err)
			}
		}
	}

	return nil
}

func (m *SMSManager) StartMonitoring(serialManager *serial.Manager, intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	go func() {
		for range ticker.C {
			for _, module := range serialManager.GetAllModules() {
				if err := m.ReadSMS(module); err != nil {
					m.logger.Warnf("Erreur monitoring SMS module %s: %v", module.Port, err)
				}
				if err := m.AutoFilterTrash(module); err != nil {
					m.logger.Warnf("Erreur filtre auto module %s: %v", module.Port, err)
				}
			}
		}
	}()
}

func (m *SMSManager) validateNumber(number string) error {
	if len(number) < 8 || len(number) > 15 {
		return fmt.Errorf("numéro invalide")
	}
	for _, c := range number {
		if (c < '0' || c > '9') && c != '+' {
			return fmt.Errorf("caractère invalide dans le numéro")
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
