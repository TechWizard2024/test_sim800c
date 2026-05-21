package serial

import (
	"sync"
	"time"

	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/websocket"

	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

type Manager struct {
	cfg      *config.Config
	logger   *logrus.Logger
	hub      *websocket.Hub
	modules  map[string]*SIM800C
	mu       sync.RWMutex
	stopChan chan struct{}
}

type SIM800C struct {
	Port        string
	SerialPort  *serial.Port
	Logger      *logrus.Logger
	ModuleID    int
	PhoneNumber string
	IMEI        string
	Carrier     string
	mu          sync.Mutex
	commandChan chan Command
	stopChan    chan struct{}
}

type Command struct {
	Type       string      `json:"type"`
	USSDCode   string      `json:"ussd_code,omitempty"`
	InputData  string      `json:"input_data,omitempty"`
	SMSNumber  string      `json:"sms_number,omitempty"`
	SMSMessage string      `json:"sms_message,omitempty"`
	Response   chan string `json:"-"`
}

func NewManager(cfg *config.Config, logger *logrus.Logger, hub *websocket.Hub) *Manager {
	return &Manager{
		cfg:      cfg,
		logger:   logger,
		hub:      hub,
		modules:  make(map[string]*SIM800C),
		stopChan: make(chan struct{}),
	}
}

func (m *Manager) Start() error {
	m.logger.Info("Démarrage du gestionnaire série")

	for _, port := range m.cfg.Serial.Ports {
		go m.connectModule(port)
	}

	go m.monitorModules()

	return nil
}

func (m *Manager) connectModule(port string) {
	m.logger.Infof("Tentative de connexion au module sur %s", port)

	serialConfig := &serial.Config{
		Name:        port,
		Baud:        m.cfg.Serial.BaudRate,
		ReadTimeout: m.cfg.GetConnectionTimeout(),
	}

	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		m.logger.Errorf("Erreur ouverture port %s: %v", port, err)
		return
	}

	module := &SIM800C{
		Port:        port,
		SerialPort:  serialPort,
		Logger:      m.logger,
		commandChan: make(chan Command, m.cfg.Serial.CommandQueueSize),
		stopChan:    make(chan struct{}),
	}

	m.mu.Lock()
	m.modules[port] = module
	m.mu.Unlock()

	// Initialiser le module
	go module.initialize()
	go module.handleCommands()
	go module.readResponses()

	m.logger.Infof("Module connecté sur %s", port)

	// Broadcast l'événement
	m.hub.BroadcastEvent(websocket.Event{
		Type:      "module_connected",
		ModuleID:  module.ModuleID,
		Data:      map[string]interface{}{"port": port},
		Timestamp: time.Now(),
	})
}

func (m *Manager) monitorModules() {
	ticker := time.NewTicker(time.Duration(m.cfg.Monitoring.CheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkModulesHealth()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Manager) checkModulesHealth() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for port, module := range m.modules {
		if err := module.SendAT(); err != nil {
			m.logger.Warnf("Module %s non responsive: %v", port, err)
			m.hub.BroadcastEvent(websocket.Event{
				Type:      "module_disconnected",
				ModuleID:  module.ModuleID,
				Data:      map[string]interface{}{"port": port, "error": err.Error()},
				Timestamp: time.Now(),
			})
		}
	}
}

func (m *Manager) Stop() {
	m.logger.Info("Arrêt du gestionnaire série")
	close(m.stopChan)

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, module := range m.modules {
		module.stopChan <- struct{}{}
		module.SerialPort.Close()
	}
}

func (m *Manager) GetModule(port string) (*SIM800C, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	module, ok := m.modules[port]
	return module, ok
}

func (m *Manager) GetAllModules() []*SIM800C {
	m.mu.RLock()
	defer m.mu.RUnlock()

	modules := make([]*SIM800C, 0, len(m.modules))
	for _, module := range m.modules {
		modules = append(modules, module)
	}
	return modules
}
