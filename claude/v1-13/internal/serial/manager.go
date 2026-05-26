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

// DialPlanEntry is a minimal copy of db.DialPlan to avoid import cycle.
type DialPlanEntry struct {
	Operator     string
	Prefix       string
	CallingCode  string
	NumberLength int
}

type Manager struct {
	cfg                 *config.Config
	logger              *logrus.Logger
	hub                 *websocket.Hub
	modules             map[string]*SIM800C
	mu                  sync.RWMutex
	stopChan            chan struct{}
	DialPlan            []DialPlanEntry       // loaded from DB at startup
	OnModuleInitialized func(module *SIM800C) // callback called after a module finishes init
}

// SIM800C struct — note TWO mutexes:
//
//	mu    : protects struct fields (PhoneNumber, Carrier, IMEI, readerStarted, rb)
//	cmdMu : serializes AT command send/receive (prevents interleaved commands)
type SIM800C struct {
	Port          string
	SerialPort    *tserial.Port
	Logger        *logrus.Logger
	ModuleID      int
	DBID          int // ID dans la table modules (synced from DB after SaveModule)
	PhoneNumber   string
	IMEI          string
	Carrier       string
	PINUnlocked   bool   // true if PIN was required and unlocked successfully, or no PIN required
	PINFailed     bool   // true if PIN unlock was attempted but all codes failed
	SignalQuality int    // valeur CSQ (0-31, 99=inconnu)
	NetworkStatus string // "registered", "searching", "denied", "unknown", "roaming"

	dialPlan []DialPlanEntry // injected from DB at connect time

	mu    sync.RWMutex // protects fields above + readerStarted
	cmdMu sync.Mutex   // serializes serial commands

	// Single reader state
	readerStarted bool
	rb            *syncReadBuffer

	commandChan chan Command
	stopChan    chan struct{}
	hub         *websocket.Hub // for real-time notifications
	onInitDone  func()         // called after initialize() completes (for DB persistence etc.)
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

// scanCOMPorts detects all available serial ports dynamically.
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
		dialPlan:    m.DialPlan,
		commandChan: make(chan Command, m.cfg.Serial.CommandQueueSize),
		stopChan:    make(chan struct{}),
		hub:         m.hub,
	}

	m.mu.Lock()
	m.modules[port] = module
	m.mu.Unlock()

	// Start reader then initialize (no deadlock — these use cmdMu, not mu)
	module.startSingleReader()

	// Set post-init callback before launching goroutine
	module.onInitDone = func() {
		if m.OnModuleInitialized != nil {
			m.OnModuleInitialized(module)
		}
	}

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
		interval = 30
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkModulesHealth()
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

	newFound := 0
	for _, port := range ports {
		if !existing[port] {
			m.logger.Infof("Nouveau port détecté: %s - connexion en cours...", port)
			go m.connectModule(port)
			newFound++
		}
	}

	// Broadcast scan complete event with current module count
	if m.hub != nil {
		m.mu.RLock()
		total := len(m.modules)
		m.mu.RUnlock()
		m.hub.BroadcastEvent(websocket.Event{
			Type: "discovery_scan_complete",
			Data: map[string]interface{}{
				"modules_total": total,
				"new_found":     newFound,
				"ports_scanned": len(ports),
			},
			Timestamp: time.Now(),
		})
	}
}

func (m *Manager) checkModulesHealth() {
	// Collect modules snapshot under read lock, then release before AT commands
	m.mu.RLock()
	modulesCopy := make([]*SIM800C, 0, len(m.modules))
	for _, module := range m.modules {
		modulesCopy = append(modulesCopy, module)
	}
	m.mu.RUnlock()

	for _, module := range modulesCopy {
		port := module.Port
		if err := module.SendAT(); err != nil {
			m.logger.Warnf("Module %s non responsive: %v", port, err)
			m.hub.BroadcastEvent(websocket.Event{
				Type:      "module_disconnected",
				ModuleID:  module.ModuleID,
				Data:      map[string]interface{}{"port": port, "error": err.Error()},
				Timestamp: time.Now(),
			})
		} else {
			// Refresh signal quality periodically
			if csq, err := module.getSignalQuality(); err == nil {
				module.SignalQuality = csq
			}
			module.NetworkStatus = module.getNetworkStatus()
			m.hub.BroadcastEvent(websocket.Event{
				Type:     "signal_update",
				ModuleID: module.ModuleID,
				Data: map[string]interface{}{
					"port":           port,
					"signal_quality": module.SignalQuality,
					"signal_rssi":    CSQToRSSI(module.SignalQuality),
					"network_status": module.NetworkStatus,
				},
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
		select {
		case module.stopChan <- struct{}{}:
		default:
		}
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

// ReloadDialPlan updates the dial plan in the Manager and propagates it to all
// currently connected modules. Call this after a CRUD operation on /api/dialplan.
func (m *Manager) ReloadDialPlan(newPlan []DialPlanEntry) {
	m.mu.Lock()
	m.DialPlan = newPlan
	for _, module := range m.modules {
		module.mu.Lock()
		module.dialPlan = newPlan
		module.mu.Unlock()
	}
	m.mu.Unlock()
	m.logger.Infof("Plan de numérotation rechargé: %d entrées propagées à %d modules", len(newPlan), len(m.modules))
}

// GetModuleByDBID returns the module with the given DB ID (DBID field).
// Falls back to ModuleID lookup if no match found by DBID.
func (m *Manager) GetModuleByDBID(dbID int) (*SIM800C, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, module := range m.modules {
		if module.DBID == dbID {
			return module, true
		}
	}
	// Fallback: search by ModuleID
	for _, module := range m.modules {
		if module.ModuleID == dbID {
			return module, true
		}
	}
	return nil, false
}

// GetEffectiveID returns the DB ID if available (>0), otherwise the in-memory ModuleID.
// Use this for all DB foreign key references (ussd_history.module_id, sms_messages.module_id).
func (s *SIM800C) GetEffectiveID() int {
	if s.DBID > 0 {
		return s.DBID
	}
	return s.ModuleID
}
