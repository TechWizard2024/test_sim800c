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

func NewSMSManager(logger *logrus.Logger, hub *websocket.Hub, db *db.DB, autoTrashKeyword string) *SMSManager {
	return &SMSManager{
		logger:           logger,
		hub:              hub,
		db:               db,
		autoTrashKeyword: autoTrashKeyword,
	}
}

func (m *SMSManager) SendSMS(module *serial.SIM800C, number, message string) error {
	m.logger.Infof("Envoi SMS de %s à %s", module.Port, number)

	// Valider le numéro
	if err := m.validateNumber(number); err != nil {
		return err
	}

	// Envoyer via le module
	cmd := serial.Command{
		Type:       "sms_send",
		SMSNumber:  number,
		SMSMessage: message,
	}

	response, err := module.SendCommand(cmd)
	if err != nil {
		m.logger.Errorf("Erreur envoi SMS: %v", err)
		return err
	}

	// Sauvegarder dans la base de données
	sms := &db.SMSMessage{
		ModuleID:       module.ModuleID,
		ReceiverNumber: number,
		Message:        message,
		Direction:      "out",
		ReceivedAt:     time.Now(),
	}

	if err := m.db.SaveSMS(sms); err != nil {
		m.logger.Warnf("Erreur sauvegarde SMS: %v", err)
	}

	// Notifier via WebSocket
	m.hub.BroadcastEvent(websocket.Event{
		Type:      "sms_sent",
		ModuleID:  module.ModuleID,
		Data:      map[string]interface{}{"number": number, "message": message, "response": response},
		Timestamp: time.Now(),
	})

	m.logger.Info("SMS envoyé avec succès")
	return nil
}

func (m *SMSManager) ReadSMS(module *serial.SIM800C) ([]SMS, error) {
	m.logger.Infof("Lecture SMS du module %s", module.Port)

	smsList, err := module.ListSMS()
	if err != nil {
		return nil, fmt.Errorf("erreur lecture SMS: %w", err)
	}

	var smsMessages []SMS

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

		sms := SMS{
			ModuleID:     module.ModuleID,
			SenderNumber: sender,
			Message:      message,
			Direction:    "in",
			SMSIndex:     idx,
			ReceivedAt:   time.Now(),
		}

		// Vérifier si le SMS doit aller à la corbeille
		if !strings.Contains(message, m.autoTrashKeyword) {
			sms.IsTrash = true
			m.logger.Debugf("SMS déplacé vers corbeille: ne contient pas '%s'", m.autoTrashKeyword)
		}

		smsMessages = append(smsMessages, sms)

		// Sauvegarder en base
		dbSMS := &db.SMSMessage{
			ModuleID:     module.ModuleID,
			SenderNumber: sender,
			Message:      message,
			Direction:    "in",
			IsTrash:      sms.IsTrash,
			SMSIndex:     idx,
			ReceivedAt:   sms.ReceivedAt,
		}

		if err := m.db.SaveSMS(dbSMS); err != nil {
			m.logger.Warnf("Erreur sauvegarde SMS en base: %v", err)
		}

		// Notifier via WebSocket
		m.hub.BroadcastEvent(websocket.Event{
			Type:      "sms_received",
			ModuleID:  module.ModuleID,
			Data:      sms,
			Timestamp: time.Now(),
		})
	}

	return smsMessages, nil
}

func (m *SMSManager) DeleteSMS(module *serial.SIM800C, index int) error {
	m.logger.Infof("Suppression SMS index %d du module %s", index, module.Port)

	if err := module.DeleteSMS(index); err != nil {
		return fmt.Errorf("erreur suppression SMS: %w", err)
	}

	// Marquer comme supprimé en base
	if err := m.db.MarkSMSDeleted(module.ModuleID, index); err != nil {
		m.logger.Warnf("Erreur marquage SMS supprimé: %v", err)
	}

	m.hub.BroadcastEvent(websocket.Event{
		Type:      "sms_deleted",
		ModuleID:  module.ModuleID,
		Data:      map[string]interface{}{"index": index},
		Timestamp: time.Now(),
	})

	return nil
}

func (m *SMSManager) MoveToTrash(module *serial.SIM800C, smsID int) error {
	m.logger.Infof("Déplacement SMS %d vers corbeille", smsID)

	if err := m.db.MoveSMSToTrash(smsID); err != nil {
		return fmt.Errorf("erreur déplacement vers corbeille: %w", err)
	}

	m.hub.BroadcastEvent(websocket.Event{
		Type:      "sms_moved_to_trash",
		ModuleID:  module.ModuleID,
		Data:      map[string]interface{}{"sms_id": smsID},
		Timestamp: time.Now(),
	})

	return nil
}

func (m *SMSManager) GetSMS(moduleID int, includeTrash bool) ([]db.SMSMessage, error) {
	return m.db.GetSMSByModule(moduleID, includeTrash)
}

func (m *SMSManager) AutoFilterTrash(module *serial.SIM800C) error {
	m.logger.Infof("Filtrage automatique des SMS pour module %s", module.Port)

	smsList, err := m.ReadSMS(module)
	if err != nil {
		return err
	}

	for _, sms := range smsList {
		if sms.Direction == "in" && !strings.Contains(sms.Message, m.autoTrashKeyword) && !sms.IsTrash {
			if err := m.MoveToTrash(module, sms.ID); err != nil {
				m.logger.Warnf("Erreur filtrage automatique SMS %d: %v", sms.ID, err)
			}
		}
	}

	return nil
}

func (m *SMSManager) validateNumber(number string) error {
	// Validation basique du numéro
	if len(number) < 8 || len(number) > 15 {
		return fmt.Errorf("numéro de téléphone invalide")
	}

	// Vérifier les caractères
	for _, c := range number {
		if (c < '0' || c > '9') && c != '+' {
			return fmt.Errorf("le numéro ne doit contenir que des chiffres et éventuellement +")
		}
	}

	return nil
}

func (m *SMSManager) StartAutoFilterRoutine(modules []*serial.SIM800C, intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	go func() {
		for range ticker.C {
			for _, module := range modules {
				if err := m.AutoFilterTrash(module); err != nil {
					m.logger.Warnf("Erreur filtre auto module %s: %v", module.Port, err)
				}
			}
		}
	}()
}
