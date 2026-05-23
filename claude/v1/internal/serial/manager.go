package serial

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/websocket"

	"github.com/sirupsen/logrus"
	tserial "github.com/tarm/serial"
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
	SerialPort  *tserial.Port
	Logger      *logrus.Logger
	ModuleID    int
	PhoneNumber string
	IMEI        string
	Carrier     string

	mu sync.Mutex

	// Single reader state
	readerStarted bool
	rb            *syncReadBuffer

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

// scanCOMPorts detects all available COM ports dynamically on Windows (COM1..COM99)
// plus any explicitly configured ports.
func (m *Manager) scanCOMPorts() []string {
	found := []string{}
	seen := map[string]bool{}

	// First, try configured ports
	for _, p := range m.cfg.Serial.Ports {
		if !seen[p] {
			seen[p] = true
			found = append(found, p)
		}
	}

	// Dynamic scan COM1..COM99 for Windows
	for i := 1; i <= 99; i++ {
		port := fmt.Sprintf("COM%d", i)
		if seen[port] {
			continue
		}
		cfg := &tserial.Config{
			Name:        port,
			Baud:        m.cfg.Serial.BaudRate,
			ReadTimeout: 1 * time.Second,
		}
		sp, err := tserial.OpenPort(cfg)
		if err == nil {
			sp.Close()
			found = append(found, port)
			seen[port] = true
			m.logger.Infof("Port COM détecté: %s", port)
		}
	}

	// Also scan /dev/ttyUSB* and /dev/ttyACM* for Linux
	for i := 0; i <= 9; i++ {
		for _, prefix := range []string{"/dev/ttyUSB", "/dev/ttyACM", "/dev/ttyS"} {
			port := fmt.Sprintf("%s%d", prefix, i)
			if seen[port] {
				continue
			}
			cfg := &tserial.Config{
				Name:        port,
				Baud:        m.cfg.Serial.BaudRate,
				ReadTimeout: 1 * time.Second,
			}
			sp, err := tserial.OpenPort(cfg)
			if err == nil {
				sp.Close()
				found = append(found, port)
				seen[port] = true
				m.logger.Infof("Port série détecté: %s", port)
			}
		}
	}

	return found
}

// isSIM800C sends AT command and checks if the port responds like a SIM800C modem
func (m *Manager) isSIM800C(port string) bool {
	cfg := &tserial.Config{
		Name:        port,
		Baud:        m.cfg.Serial.BaudRate,
		ReadTimeout: 2 * time.Second,
	}
	sp, err := tserial.OpenPort(cfg)
	if err != nil {
		return false
	}
	defer sp.Close()

	sp.Write([]byte("AT\r\n"))
	time.Sleep(500 * time.Millisecond)
	buf := make([]byte, 64)
	n, _ := sp.Read(buf)
	response := strings.TrimSpace(string(buf[:n]))
	return strings.Contains(response, "OK") || strings.Contains(response, "AT")
}

func (m *Manager) Start() error {
	m.logger.Info("Démarrage du gestionnaire série avec auto-discovery des ports COM")

	ports := m.scanCOMPorts()
	if len(ports) == 0 {
		m.logger.Warn("Aucun port COM trouvé. En attente de connexion...")
	}

	for _, port := range ports {
		go m.connectModule(port)
	}

	go m.monitorModules()
	return nil
}

func (m *Manager) connectModule(port string) {
	m.logger.Infof("Tentative de connexion au module sur %s", port)

	serialConfig := &tserial.Config{
		Name:        port,
		Baud:        m.cfg.Serial.BaudRate,
		ReadTimeout: m.cfg.GetConnectionTimeout(),
	}

	serialPort, err := tserial.OpenPort(serialConfig)
	if err != nil {
		m.logger.Errorf("Erreur ouverture port %s: %v", port, err)
		return
	}

	// Assign a module ID
	m.mu.Lock()
	moduleID := len(m.modules) + 1
	m.mu.Unlock()

	module := &SIM800C{
		Port:        port,
		SerialPort:  serialPort,
		Logger:      m.logger,
		ModuleID:    moduleID,
		commandChan: make(chan Command, m.cfg.Serial.CommandQueueSize),
		stopChan:    make(chan struct{}),
	}

	m.mu.Lock()
	m.modules[port] = module
	m.mu.Unlock()

	go module.initialize()
	go module.handleCommands()

	m.logger.Infof("Module connecté sur %s (ID=%d)", port, moduleID)

	m.hub.BroadcastEvent(websocket.Event{
		Type:      "module_connected",
		ModuleID:  module.ModuleID,
		Data:      map[string]interface{}{"port": port},
		Timestamp: time.Now(),
	})
}

func (m *Manager) monitorModules() {
	interval := m.cfg.Monitoring.CheckIntervalSeconds
	if interval <= 0 {
		interval = 10
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkModulesHealth()
			// Re-scan for newly connected modules
			m.discoverNewModules()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Manager) discoverNewModules() {
	ports := m.scanCOMPorts()
	m.mu.RLock()
	existing := make(map[string]bool, len(m.modules))
	for p := range m.modules {
		existing[p] = true
	}
	m.mu.RUnlock()

	for _, port := range ports {
		if !existing[port] {
			m.logger.Infof("Nouveau port détecté: %s - connexion en cours...", port)
			go m.connectModule(port)
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
