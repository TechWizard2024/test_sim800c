@echo off
title SIM800C Supervisor - Project Generator
echo ================================================================================
echo                 SIM800C SUPERVISOR - PROJECT GENERATOR
echo ================================================================================
echo.
echo Generating project in: C:\xampp\htdocs\aa_Toolbox\test_sim800c\deepseek\v1
echo.

set PROJECT_ROOT=C:\xampp\htdocs\aa_Toolbox\test_sim800c\deepseek\v1

:: Create directory structure
echo [1/8] Creating directory structure...
mkdir "%PROJECT_ROOT%" 2>nul
mkdir "%PROJECT_ROOT%\cmd" 2>nul
mkdir "%PROJECT_ROOT%\internal\config" 2>nul
mkdir "%PROJECT_ROOT%\internal\serial" 2>nul
mkdir "%PROJECT_ROOT%\internal\ussd" 2>nul
mkdir "%PROJECT_ROOT%\internal\sms" 2>nul
mkdir "%PROJECT_ROOT%\internal\excel" 2>nul
mkdir "%PROJECT_ROOT%\internal\db" 2>nul
mkdir "%PROJECT_ROOT%\internal\websocket" 2>nul
mkdir "%PROJECT_ROOT%\internal\api\handlers" 2>nul
mkdir "%PROJECT_ROOT%\internal\api\middleware" 2>nul
mkdir "%PROJECT_ROOT%\web\css" 2>nul
mkdir "%PROJECT_ROOT%\web\js" 2>nul
mkdir "%PROJECT_ROOT%\web\assets\icons" 2>nul
mkdir "%PROJECT_ROOT%\web\assets\fonts" 2>nul
mkdir "%PROJECT_ROOT%\scripts" 2>nul
mkdir "%PROJECT_ROOT%\storage\excel" 2>nul
mkdir "%PROJECT_ROOT%\storage\logs" 2>nul
mkdir "%PROJECT_ROOT%\docs" 2>nul
echo OK.

:: Generate main.go
echo [2/8] Generating main.go...
(
echo package main
echo.
echo import (
echo     "log"
echo     "os"
echo     "os/signal"
echo     "syscall"
echo     "sim800c-supervisor/internal/api"
echo     "sim800c-supervisor/internal/config"
echo     "sim800c-supervisor/internal/db"
echo     "sim800c-supervisor/internal/serial"
echo     "sim800c-supervisor/internal/websocket"
echo )
echo.
echo func main^(^) {
echo     log.Println("Starting SIM800C Supervisor v1.0.0")
echo.
echo     // Load configuration
echo     cfg, err := config.Load("config.yaml")
echo     if err != nil {
echo         log.Fatalf("Failed to load config: %%v", err)
echo     }
echo.
echo     // Connect to database
echo     database, err := db.Connect(cfg.MySQL)
echo     if err != nil {
echo         log.Fatalf("Failed to connect to database: %%v", err)
echo     }
echo     defer database.Close()
echo.
echo     // Initialize WebSocket hub
echo     hub := websocket.NewHub()
echo     go hub.Run()
echo.
echo     // Initialize serial manager
echo     serialManager := serial.NewManager(cfg.Serial, database, hub)
echo     if err := serialManager.Start(); err != nil {
echo         log.Fatalf("Failed to start serial manager: %%v", err)
echo     }
echo     defer serialManager.Stop()
echo.
echo     // Setup API server
echo     apiServer := api.NewServer(cfg, database, hub, serialManager)
echo     go func^() {
echo         if err := apiServer.Run(); err != nil {
echo             log.Fatalf("Failed to start API server: %%v", err)
echo         }
echo     }()
echo.
echo     log.Println("System ready! Access at http://test_sim800c.local")
echo.
echo     // Wait for interrupt signal
echo     quit := make(chan os.Signal, 1)
echo     signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
echo     ^<-quit
echo.
echo     log.Println("Shutting down...")
echo }
) > "%PROJECT_ROOT%\cmd\main.go"
echo OK.

:: Generate config files
echo [3/8] Generating configuration files...
(
echo server:
echo   port: 8080
echo   websocket_path: "/ws"
echo.
echo serial:
echo   ports:
echo     - "COM5"
echo     - "COM6"
echo     - "COM7"
echo   baud_rate: 9600
echo   timeout_seconds: 30
echo   reconnect_delay_seconds: 5
echo.
echo mysql:
echo   host: "localhost"
echo   port: 3306
echo   user: "root"
echo   password: ""
echo   database: "sim800c_manager_deepseekv1"
echo.
echo excel:
echo   base_path: "C:/xampp/htdocs/aa_Toolbox/test_sim800c/deepseek/v1/storage/excel"
echo   filename_pattern: "Codes_USSD_CI*.xlsx"
echo   reload_interval_minutes: 5
echo.
echo ussd:
echo   max_menu_depth: 10
echo   session_timeout_seconds: 60
echo   default_choice_timeout_seconds: 5
echo.
echo sms:
echo   auto_trash_keyword: "Test"
echo   max_sms_per_module: 500
echo.
echo security:
echo   jwt_secret: "super-secret-key-change-me-in-production-2026"
echo   encryption_key: "32-byte-key-for-aes-256-encryption"
echo   enable_auth: false
echo   default_admin_password: "admin123"
) > "%PROJECT_ROOT%\config.yaml"

(
echo module sim800c-supervisor
echo.
echo go 1.22
echo.
echo require (
echo     github.com/go-sql-driver/mysql v1.8.0
echo     github.com/golang-jwt/jwt/v5 v5.2.0
echo     github.com/gorilla/websocket v1.5.1
echo     github.com/joho/godotenv v1.5.1
echo     github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
echo     github.com/xuri/excelize/v2 v2.8.0
echo     golang.org/x/crypto v0.19.0
echo )
echo.
echo require filippo.io/edwards25519 v1.1.0 // indirect
) > "%PROJECT_ROOT%\go.mod"
echo OK.

:: Generate internal/config/config.go
echo [4/8] Generating internal packages...
(
echo package config
echo.
echo import (
echo     "io/ioutil"
echo     "gopkg.in/yaml.v3"
echo )
echo.
echo type Config struct {
echo     Server  ServerConfig  `yaml:"server"`
echo     Serial  SerialConfig  `yaml:"serial"`
echo     MySQL   MySQLConfig   `yaml:"mysql"`
echo     Excel   ExcelConfig   `yaml:"excel"`
echo     USSD    USSDConfig    `yaml:"ussd"`
echo     SMS     SMSConfig     `yaml:"sms"`
echo     Security SecurityConfig `yaml:"security"`
echo }
echo.
echo type ServerConfig struct {
echo     Port            int    `yaml:"port"`
echo     WebSocketPath   string `yaml:"websocket_path"`
echo }
echo.
echo type SerialConfig struct {
echo     Ports               []string `yaml:"ports"`
echo     BaudRate            int      `yaml:"baud_rate"`
echo     TimeoutSeconds      int      `yaml:"timeout_seconds"`
echo     ReconnectDelaySeconds int    `yaml:"reconnect_delay_seconds"`
echo }
echo.
echo type MySQLConfig struct {
echo     Host     string `yaml:"host"`
echo     Port     int    `yaml:"port"`
echo     User     string `yaml:"user"`
echo     Password string `yaml:"password"`
echo     Database string `yaml:"database"`
echo }
echo.
echo type ExcelConfig struct {
echo     BasePath           string `yaml:"base_path"`
echo     FilenamePattern    string `yaml:"filename_pattern"`
echo     ReloadIntervalMinutes int `yaml:"reload_interval_minutes"`
echo }
echo.
echo type USSDConfig struct {
echo     MaxMenuDepth              int `yaml:"max_menu_depth"`
echo     SessionTimeoutSeconds     int `yaml:"session_timeout_seconds"`
echo     DefaultChoiceTimeoutSeconds int `yaml:"default_choice_timeout_seconds"`
echo }
echo.
echo type SMSConfig struct {
echo     AutoTrashKeyword string `yaml:"auto_trash_keyword"`
echo     MaxSmsPerModule  int    `yaml:"max_sms_per_module"`
echo }
echo.
echo type SecurityConfig struct {
echo     JWTSecret            string `yaml:"jwt_secret"`
echo     EncryptionKey        string `yaml:"encryption_key"`
echo     EnableAuth           bool   `yaml:"enable_auth"`
echo     DefaultAdminPassword string `yaml:"default_admin_password"`
echo }
echo.
echo func Load(path string) (*Config, error) {
echo     data, err := ioutil.ReadFile(path)
echo     if err != nil {
echo         return nil, err
echo     }
echo     var cfg Config
echo     if err := yaml.Unmarshal(data, ^&cfg); err != nil {
echo         return nil, err
echo     }
echo     return ^&cfg, nil
echo }
) > "%PROJECT_ROOT%\internal\config\config.go"

:: Generate internal/serial/sim800c.go
(
echo package serial
echo.
echo import (
echo     "fmt"
echo     "log"
echo     "strings"
echo     "sync"
echo     "time"
echo     "github.com/tarm/serial"
echo )
echo.
echo type SIM800C struct {
echo     port        *serial.Port
echo     comPort     string
echo     baudRate    int
echo     mu          sync.Mutex
echo     isConnected bool
echo     imei        string
echo     phoneNumber string
echo     carrier     string
echo     lastSeen    time.Time
echo }
echo.
echo func NewSIM800C(comPort string, baudRate int) *SIM800C {
echo     return ^&SIM800C{
echo         comPort:  comPort,
echo         baudRate: baudRate,
echo     }
echo }
echo.
echo func (s *SIM800C) Connect() error {
echo     s.mu.Lock()
echo     defer s.mu.Unlock()
echo.
echo     config := ^&serial.Config{
echo         Name: s.comPort,
echo         Baud: s.baudRate,
echo         ReadTimeout: time.Second * 5,
echo     }
echo.
echo     port, err := serial.OpenPort(config)
echo     if err != nil {
echo         return fmt.Errorf("failed to open port %%s: %%v", s.comPort, err)
echo     }
echo.
echo     s.port = port
echo     s.isConnected = true
echo     s.lastSeen = time.Now()
echo.
echo     // Test communication
echo     if err := s.sendCommand("AT"); err != nil {
echo         s.port.Close()
echo         return fmt.Errorf("module not responding: %%v", err)
echo     }
echo.
echo     return nil
echo }
echo.
echo func (s *SIM800C) Disconnect() error {
echo     s.mu.Lock()
echo     defer s.mu.Unlock()
echo.
echo     if s.port != nil {
echo         if err := s.sendCommand("AT+CUSD=2"); err != nil {
echo             log.Printf("Failed to end USSD session: %%v", err)
echo         }
echo         return s.port.Close()
echo     }
echo     return nil
echo }
echo.
echo func (s *SIM800C) sendCommand(cmd string) error {
echo     if s.port == nil {
echo         return fmt.Errorf("port not connected")
echo     }
echo.
echo     _, err := s.port.Write([]byte(cmd + "\r"))
echo     if err != nil {
echo         return err
echo     }
echo.
echo     // Wait for response
echo     time.Sleep(time.Millisecond * 500)
echo     return nil
echo }
echo.
echo func (s *SIM800C) readResponse() (string, error) {
echo     buf := make([]byte, 1024)
echo     n, err := s.port.Read(buf)
echo     if err != nil {
echo         return "", err
echo     }
echo     return string(buf[:n]), nil
echo }
echo.
echo func (s *SIM800C) ExecuteUSSD(code string, inputData string) (string, error) {
echo     s.mu.Lock()
echo     defer s.mu.Unlock()
echo.
echo     if !s.isConnected {
echo         return "", fmt.Errorf("module not connected")
echo     }
echo.
echo     // Send USSD command
echo     cmd := fmt.Sprintf("AT+CUSD=1,\"%%s\",15", code)
echo     if err := s.sendCommand(cmd); err != nil {
echo         return "", err
echo     }
echo.
echo     time.Sleep(time.Second * 2)
echo.
echo     response, err := s.readResponse()
echo     if err != nil {
echo         return "", err
echo     }
echo.
echo     // Parse response
echo     if strings.Contains(response, "+CUSD:") {
echo         // Extract the USSD response
echo         parts := strings.Split(response, ",")
echo         if len(parts) >= 2 {
echo             result := strings.Trim(parts[1], "\"")
echo             return result, nil
echo         }
echo     }
echo.
echo     return response, nil
echo }
echo.
echo func (s *SIM800C) GetIMEI() (string, error) {
echo     s.mu.Lock()
echo     defer s.mu.Unlock()
echo.
echo     if err := s.sendCommand("AT+CGSN"); err != nil {
echo         return "", err
echo     }
echo.
echo     time.Sleep(time.Millisecond * 500)
echo     response, err := s.readResponse()
echo     if err != nil {
echo         return "", err
echo     }
echo.
echo     lines := strings.Split(response, "\r\n")
echo     for _, line := range lines {
echo         line = strings.TrimSpace(line)
echo         if len(line) == 15 ^&^ strings.HasPrefix(line, "1") {
echo             return line, nil
echo         }
echo     }
echo     return "", fmt.Errorf("IMEI not found")
echo }
echo.
echo func (s *SIM800C) GetPhoneNumber() (string, error) {
echo     return s.ExecuteUSSD("#99#", "")
echo }
echo.
func (s *SIM800C) SendSMS(number, message string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if err := s.sendCommand("AT+CMGF=1"); err != nil {
        return err
    }

    cmd := fmt.Sprintf("AT+CMGS=\"%s\"", number)
    if err := s.sendCommand(cmd); err != nil {
        return err
    }

    time.Sleep(time.Millisecond * 500)

    if _, err := s.port.Write([]byte(message + "\x1A")); err != nil {
        return err
    }

    return nil
}
) > "%PROJECT_ROOT%\internal\serial\sim800c.go"

echo OK.

:: Generate internal/ussd/validator.go
(
echo package ussd
echo.
echo import (
echo     "fmt"
echo     "regexp"
echo     "strconv"
echo )
echo.
echo type InputValidator struct{}
echo.
echo func NewInputValidator() *InputValidator {
echo     return ^&InputValidator{}
echo }
echo.
echo func (v *InputValidator) Validate(inputType, value string) error {
echo     switch inputType {
echo     case "Choix":
echo         // Check if value is a number
echo         num, err := strconv.Atoi(value)
echo         if err != nil || num ^< 1 {
echo             return fmt.Errorf("Choix invalide: doit être un nombre positif")
echo         }
echo.
echo     case "PIN":
echo         matched, _ := regexp.MatchString("^[0-9]{4}$", value)
echo         if !matched {
echo             return fmt.Errorf("PIN invalide: doit être 4 chiffres")
echo         }
echo.
echo     case "Code de carte recharge", "Référence":
echo         matched, _ := regexp.MatchString("^[0-9]{14}$", value)
echo         if !matched {
echo             return fmt.Errorf("%%s invalide: doit être 14 chiffres", inputType)
echo         }
echo.
echo     case "Numéro", "numero de téléphone":
echo         matched, _ := regexp.MatchString("^[0-9]{10}$", value)
echo         if !matched {
echo             return fmt.Errorf("Numéro invalide: doit être 10 chiffres")
echo         }
echo.
echo     case "Montant":
echo         montant, err := strconv.ParseFloat(value, 64)
echo         if err != nil || montant ^< 50 {
echo             return fmt.Errorf("Montant invalide: doit être >= 50")
echo         }
echo.
echo     default:
echo         // No validation needed
echo         return nil
echo     }
echo     return nil
echo }
) > "%PROJECT_ROOT%\internal\ussd\validator.go"

:: Generate internal/db/database.go
(
echo package db
echo.
echo import (
echo     "database/sql"
echo     "fmt"
echo     "log"
echo     "time"
echo     _ "github.com/go-sql-driver/mysql"
echo     "sim800c-supervisor/internal/config"
echo )
echo.
echo type Database struct {
echo     Connection *sql.DB
echo }
echo.
echo func Connect(cfg config.MySQLConfig) (*Database, error) {
echo     dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4^&parseTime=True^&loc=Local",
echo         cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
echo.
echo     db, err := sql.Open("mysql", dsn)
echo     if err != nil {
echo         return nil, err
echo     }
echo.
echo     db.SetMaxOpenConns(25)
echo     db.SetMaxIdleConns(10)
echo     db.SetConnMaxLifetime(5 * time.Minute)
echo.
echo     if err := db.Ping(); err != nil {
echo         return nil, err
echo     }
echo.
echo     log.Println("Database connected successfully")
echo     return ^&Database{Connection: db}, nil
echo }
echo.
echo func (d *Database) Close() error {
echo     return d.Connection.Close()
echo }
echo.
echo // Module operations
echo type Module struct {
echo     ID          int
echo     ComPort     string
echo     IMEI        string
echo     PhoneNumber string
echo     Carrier     string
echo     Status      string
echo     LastSeen    time.Time
echo }
echo.
echo func (d *Database) SaveModule(module *Module) error {
echo     query := `
echo         INSERT INTO modules (com_port, imei, phone_number, carrier, status, last_seen)
echo         VALUES (?, ?, ?, ?, ?, NOW())
echo         ON DUPLICATE KEY UPDATE
echo             imei = VALUES(imei),
echo             phone_number = VALUES(phone_number),
echo             carrier = VALUES(carrier),
echo             status = VALUES(status),
echo             last_seen = NOW()
echo     `
echo     _, err := d.Connection.Exec(query, module.ComPort, module.IMEI, module.PhoneNumber, module.Carrier, module.Status)
echo     return err
echo }
echo.
echo func (d *Database) GetModules() ([]Module, error) {
echo     rows, err := d.Connection.Query("SELECT id, com_port, imei, phone_number, carrier, status, last_seen FROM modules ORDER BY id")
echo     if err != nil {
echo         return nil, err
echo     }
echo     defer rows.Close()
echo.
echo     var modules []Module
echo     for rows.Next() {
echo         var m Module
echo         err := rows.Scan(^&m.ID, ^&m.ComPort, ^&m.IMEI, ^&m.PhoneNumber, ^&m.Carrier, ^&m.Status, ^&m.LastSeen)
echo         if err != nil {
echo             return nil, err
echo         }
echo         modules = append(modules, m)
echo     }
echo     return modules, nil
echo }
echo.
echo // SMS operations
echo type SMS struct {
echo     ID       int
echo     ModuleID int
echo     Sender   string
echo     Message  string
echo     Direction string
echo     IsTrash  bool
echo     ReceivedAt time.Time
echo }
echo.
echo func (d *Database) SaveSMS(sms *SMS) error {
echo     query := `
echo         INSERT INTO sms_messages (module_id, sender_number, message, direction, is_trash, received_at)
echo         VALUES (?, ?, ?, ?, ?, NOW())
echo     `
echo     _, err := d.Connection.Exec(query, sms.ModuleID, sms.Sender, sms.Message, sms.Direction, sms.IsTrash)
echo     return err
echo }
echo.
echo func (d *Database) MoveToTrash(smsID int) error {
echo     _, err := d.Connection.Exec("UPDATE sms_messages SET is_trash = true WHERE id = ?", smsID)
echo     return err
echo }
) > "%PROJECT_ROOT%\internal\db\database.go"

echo OK.

:: Generate frontend files
echo [5/8] Generating frontend files...
(
echo ^<!DOCTYPE html^>
echo ^<html lang="fr"^>
echo ^<head^>
echo     ^<meta charset="UTF-8"^>
echo     ^<meta name="viewport" content="width=device-width, initial-scale=1.0"^>
echo     ^<title^>SIM800C Supervisor - Gestion des modules USSD^</title^>
echo     ^<link rel="stylesheet" href="/css/main.css"^>
echo     ^<link rel="stylesheet" href="/css/theme-light.css" id="theme-light"^>
echo     ^<link rel="stylesheet" href="/css/theme-dark.css" id="theme-dark" disabled^>
echo ^</head^>
echo ^<body^>
echo     ^<div class="app-container"^>
echo         ^<header^>
echo             ^<h1^>📱 SIM800C Supervisor^</h1^>
echo             ^<div class="header-actions"^>
echo                 ^<button id="theme-toggle" class="btn-theme"^>🌓 Thème^</button^>
echo                 ^<button id="refresh-all" class="btn-refresh"^>🔄 Rafraîchir^</button^>
echo             ^</div^>
echo         ^</header^>
echo.
echo         ^<div class="global-actions"^>
echo             ^<button id="btn-status-auto" class="btn-primary"^>📊 SIM Status Auto-Discovery^</button^>
echo             ^<button id="btn-menu-auto" class="btn-primary"^>📁 USSD Menu Auto-Discovery^</button^>
echo         ^</div^>
echo.
echo         ^<div id="modules-container" class="modules-grid"^>^</div^>
echo.
echo         ^<div class="sms-section"^>
echo             ^<h2^>📨 Gestion des SMS^</h2^>
echo             ^<div class="sms-toolbar"^>
echo                 ^<button id="refresh-sms"^>Rafraîchir^</button^>
echo                 ^<button id="new-sms"^>📝 Nouveau SMS^</button^>
echo                 ^<input type="text" id="sms-search" placeholder="Rechercher..."^>
echo             ^</div^>
echo             ^<div id="sms-inbox" class="sms-list"^>^</div^>
echo             ^<div id="sms-trash" class="sms-trash"^>
echo                 ^<h3^>🗑️ Corbeille (Sans "Test")^</h3^>
echo                 ^<div id="trash-list"^>^</div^>
echo             ^</div^>
echo         ^</div^>
echo     ^</div^>
echo.
echo     ^<script src="/js/websocket.js"^>^</script^>
echo     ^<script src="/js/dashboard.js"^>^</script^>
echo     ^<script src="/js/ussd.js"^>^</script^>
echo     ^<script src="/js/sms.js"^>^</script^>
echo     ^<script src="/js/theme.js"^>^</script^>
echo     ^<script src="/js/app.js"^>^</script^>
echo ^</body^>
echo ^</html^>
) > "%PROJECT_ROOT%\web\index.html"
echo OK.

:: Generate CSS files
echo [6/8] Generating CSS files...
(
echo :root {
echo     --bg-primary: #f5f5f5;
echo     --text-primary: #333;
echo     --card-bg: white;
echo     --border: #ddd;
echo     --btn-primary: #007bff;
echo     --btn-primary-hover: #0056b3;
echo     --success: #28a745;
echo     --error: #dc3545;
echo     --warning: #ffc107;
echo }
echo.
echo * {
echo     margin: 0;
echo     padding: 0;
echo     box-sizing: border-box;
echo }
echo.
echo body {
echo     font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
echo     background: var(--bg-primary);
echo     color: var(--text-primary);
echo     transition: all 0.3s ease;
echo }
echo.
echo .app-container {
echo     max-width: 1400px;
echo     margin: 0 auto;
echo     padding: 20px;
echo }
echo.
echo header {
echo     display: flex;
echo     justify-content: space-between;
echo     align-items: center;
echo     margin-bottom: 30px;
echo     padding-bottom: 20px;
echo     border-bottom: 2px solid var(--border);
echo }
echo.
echo .modules-grid {
echo     display: grid;
echo     grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
echo     gap: 20px;
echo     margin-bottom: 30px;
echo }
echo.
echo .module-card {
echo     background: var(--card-bg);
echo     border-radius: 12px;
echo     padding: 20px;
echo     box-shadow: 0 2px 8px rgba(0,0,0,0.1);
echo     border: 1px solid var(--border);
echo }
echo.
echo .card-header {
echo     display: flex;
echo     justify-content: space-between;
echo     align-items: center;
echo     margin-bottom: 15px;
echo     padding-bottom: 10px;
echo     border-bottom: 1px solid var(--border);
echo }
echo.
echo .status-badge {
echo     padding: 4px 12px;
echo     border-radius: 20px;
echo     font-size: 12px;
echo     font-weight: bold;
echo }
echo.
echo .status-connected {
echo     background: #d4edda;
echo     color: #155724;
echo }
echo.
echo .status-disconnected {
echo     background: #f8d7da;
echo     color: #721c24;
echo }
echo.
echo .sim-info {
echo     background: var(--bg-primary);
echo     padding: 10px;
echo     border-radius: 8px;
echo     margin: 15px 0;
echo     font-size: 14px;
echo }
echo.
echo .actions {
echo     display: flex;
echo     gap: 10px;
echo     flex-wrap: wrap;
echo     margin: 15px 0;
echo }
echo.
echo button {
echo     padding: 8px 16px;
echo     border: none;
echo     border-radius: 6px;
echo     cursor: pointer;
echo     font-size: 14px;
echo     transition: all 0.2s ease;
echo }
echo.
echo .btn-primary {
echo     background: var(--btn-primary);
echo     color: white;
echo }
echo.
echo .btn-primary:hover {
echo     background: var(--btn-primary-hover);
echo     transform: translateY(-1px);
echo }
echo.
echo .result-output {
echo     background: var(--bg-primary);
echo     padding: 10px;
echo     border-radius: 6px;
echo     margin-top: 10px;
echo     font-family: monospace;
echo     font-size: 12px;
echo     max-height: 200px;
echo     overflow-y: auto;
echo }
echo.
echo .sms-section {
echo     background: var(--card-bg);
echo     border-radius: 12px;
echo     padding: 20px;
echo     margin-top: 20px;
echo }
echo.
echo .sms-list {
echo     max-height: 400px;
echo     overflow-y: auto;
echo     margin: 15px 0;
echo }
echo.
echo .sms-item {
echo     background: var(--bg-primary);
echo     padding: 12px;
echo     margin: 8px 0;
echo     border-radius: 8px;
echo     border-left: 4px solid var(--success);
echo }
echo.
echo .sms-trash {
echo     margin-top: 20px;
echo     padding-top: 20px;
echo     border-top: 2px solid var(--border);
echo }
echo.
echo @media (max-width: 768px) {
echo     .modules-grid {
echo         grid-template-columns: 1fr;
echo     }
echo }
) > "%PROJECT_ROOT%\web\css\main.css"

(
echo /* Theme Light - Default */
echo :root {
echo     --bg-primary: #f5f5f5;
echo     --text-primary: #333;
echo     --card-bg: white;
echo     --border: #ddd;
echo     --btn-primary: #007bff;
echo     --btn-primary-hover: #0056b3;
echo }
) > "%PROJECT_ROOT%\web\css\theme-light.css"

(
echo /* Theme Dark */
echo [data-theme="dark"] {
echo     --bg-primary: #1a1a2e;
echo     --text-primary: #f0f0f0;
echo     --card-bg: #16213e;
echo     --border: #0f3460;
echo     --btn-primary: #e94560;
echo     --btn-primary-hover: #c62a47;
echo }
echo.
echo body {
echo     background: var(--bg-primary);
echo     color: var(--text-primary);
echo }
) > "%PROJECT_ROOT%\web\css\theme-dark.css"
echo OK.

:: Generate JavaScript files
echo [7/8] Generating JavaScript files...
(
echo // WebSocket Manager
echo class WebSocketManager {
echo     constructor() {
echo         this.socket = null;
echo         this.reconnectAttempts = 0;
echo         this.maxReconnectAttempts = 10;
echo         this.listeners = {};
echo     }
echo.
echo     connect() {
echo         const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
echo         const wsUrl = `${protocol}//${window.location.host}/ws`;
echo.
echo         this.socket = new WebSocket(wsUrl);
echo.
echo         this.socket.onopen = () => {
echo             console.log('WebSocket connected');
echo             this.reconnectAttempts = 0;
echo             this.dispatchEvent('connected', {});
echo         };
echo.
echo         this.socket.onmessage = (event) => {
echo             try {
echo                 const data = JSON.parse(event.data);
echo                 this.dispatchEvent(data.type, data);
echo             } catch (e) {
echo                 console.error('Failed to parse message:', e);
echo             }
echo         };
echo.
echo         this.socket.onclose = () => {
echo             console.log('WebSocket disconnected');
echo             this.reconnect();
echo         };
echo.
echo         this.socket.onerror = (error) => {
echo             console.error('WebSocket error:', error);
echo         };
echo     }
echo.
echo     reconnect() {
echo         if (this.reconnectAttempts < this.maxReconnectAttempts) {
echo             this.reconnectAttempts++;
echo             setTimeout(() => this.connect(), 3000 * this.reconnectAttempts);
echo         }
echo     }
echo.
echo     on(event, callback) {
echo         if (!this.listeners[event]) {
echo             this.listeners[event] = [];
echo         }
echo         this.listeners[event].push(callback);
echo     }
echo.
echo     dispatchEvent(event, data) {
echo         if (this.listeners[event]) {
echo             this.listeners[event].forEach(callback => callback(data));
echo         }
echo     }
echo.
echo     send(data) {
echo         if (this.socket ^&^& this.socket.readyState === WebSocket.OPEN) {
echo             this.socket.send(JSON.stringify(data));
echo         }
echo     }
echo }
echo.
echo // Initialize global WebSocket manager
echo window.wsManager = new WebSocketManager();
) > "%PROJECT_ROOT%\web\js\websocket.js"

(
echo // Dashboard Manager
echo class DashboardManager {
echo     constructor() {
echo         this.modules = {};
echo     }
echo.
echo     async loadModules() {
echo         try {
echo             const response = await fetch('/api/modules');
echo             const data = await response.json();
echo             if (data.status === 'success') {
echo                 this.modules = data.data.modules;
echo                 this.render();
echo             }
echo         } catch (error) {
echo             console.error('Failed to load modules:', error);
echo         }
echo     }
echo.
echo     render() {
echo         const container = document.getElementById('modules-container');
echo         if (!container) return;
echo.
echo         if (Object.keys(this.modules).length === 0) {
echo             container.innerHTML = '<div class="no-modules">Aucun module trouvé. Vérifiez les connexions SIM800C.</div>';
echo             return;
echo         }
echo.
echo         container.innerHTML = '';
echo         for (const [id, module] of Object.entries(this.modules)) {
echo             container.innerHTML += this.renderModuleCard(id, module);
echo         }
echo.
echo         this.attachEventListeners();
echo     }
echo.
echo     renderModuleCard(id, module) {
echo         const statusClass = module.status === 'connected' ? 'status-connected' : 'status-disconnected';
echo         const statusText = module.status === 'connected' ? '✅ Connecté' : '❌ Déconnecté';
echo.
echo         return `
echo             <div class="module-card" data-module-id="${id}">
echo                 <div class="card-header">
echo                     <h3>📡 Module ${module.com_port}</h3>
echo                     <span class="status-badge ${statusClass}">${statusText}</span>
echo                 </div>
echo                 <div class="sim-info">
echo                     <p><strong>IMEI:</strong> ${module.imei || 'Non détecté'}</p>
echo                     <p><strong>Numéro:</strong> ${module.phone_number || 'Non détecté'}</p>
echo                     <p><strong>Opérateur:</strong> ${module.carrier || 'Non détecté'}</p>
echo                 </div>
echo                 <div class="actions">
echo                     <button class="btn-status-manual btn-primary" data-module="${id}">📊 SIM Status</button>
echo                     <button class="btn-menu-manual btn-primary" data-module="${id}">📁 Menu USSD</button>
echo                     <button class="btn-ussd-custom btn-primary" data-module="${id}">⌨️ USSD Perso</button>
echo                 </div>
echo                 <div class="result-output" id="result-${id}"></div>
echo             </div>
echo         `;
echo     }
echo.
echo     attachEventListeners() {
echo         document.querySelectorAll('.btn-status-manual').forEach(btn => {
echo             btn.onclick = () => this.executeStatusManual(btn.dataset.module);
echo         });
echo.
echo         document.querySelectorAll('.btn-menu-manual').forEach(btn => {
echo             btn.onclick = () => this.executeMenuManual(btn.dataset.module);
echo         });
echo.
echo         document.querySelectorAll('.btn-ussd-custom').forEach(btn => {
echo             btn.onclick = () => this.showCustomUSSDDialog(btn.dataset.module);
echo         });
echo     }
echo.
echo     async executeStatusManual(moduleId) {
echo         const resultDiv = document.getElementById(`result-${moduleId}`);
echo         resultDiv.innerHTML = '<div class="loading">⏳ Exécution en cours...</div>';
echo.
echo         try {
echo             const response = await fetch(`/api/modules/${moduleId}/ussd/status`, { method: 'POST' });
echo             const data = await response.json();
echo             resultDiv.innerHTML = `<pre class="result">${JSON.stringify(data, null, 2)}</pre>`;
echo         } catch (error) {
echo             resultDiv.innerHTML = `<div class="error">❌ Erreur: ${error.message}</div>`;
echo         }
echo     }
echo.
echo     async executeMenuManual(moduleId) {
echo         const resultDiv = document.getElementById(`result-${moduleId}`);
echo         resultDiv.innerHTML = '<div class="loading">⏳ Exploration du menu en cours...</div>';
echo.
echo         try {
echo             const response = await fetch(`/api/modules/${moduleId}/ussd/menu`, { method: 'POST' });
echo             const data = await response.json();
echo             resultDiv.innerHTML = `<pre class="result">${JSON.stringify(data, null, 2)}</pre>`;
echo         } catch (error) {
echo             resultDiv.innerHTML = `<div class="error">❌ Erreur: ${error.message}</div>`;
echo         }
echo     }
echo.
echo     showCustomUSSDDialog(moduleId) {
echo         const code = prompt('Entrez le code USSD à exécuter (ex: #144#):');
echo         if (code) {
echo             this.executeCustomUSSD(moduleId, code);
echo         }
echo     }
echo.
echo     async executeCustomUSSD(moduleId, code) {
echo         const resultDiv = document.getElementById(`result-${moduleId}`);
echo         resultDiv.innerHTML = '<div class="loading">⏳ Exécution en cours...</div>';
echo.
echo         try {
echo             const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
echo                 method: 'POST',
echo                 headers: { 'Content-Type': 'application/json' },
echo                 body: JSON.stringify({ code: code })
echo             });
echo             const data = await response.json();
echo             resultDiv.innerHTML = `<pre class="result">${JSON.stringify(data, null, 2)}</pre>`;
echo         } catch (error) {
echo             resultDiv.innerHTML = `<div class="error">❌ Erreur: ${error.message}</div>`;
echo         }
echo     }
echo }
echo.
echo // Initialize dashboard
echo window.dashboardManager = new DashboardManager();
) > "%PROJECT_ROOT%\web\js\dashboard.js"

(
echo // Theme Manager
echo class ThemeManager {
echo     constructor() {
echo         this.currentTheme = localStorage.getItem('theme') || 'light';
echo         this.applyTheme(this.currentTheme);
echo     }
echo.
echo     applyTheme(theme) {
echo         if (theme === 'dark') {
echo             document.documentElement.setAttribute('data-theme', 'dark');
echo             document.getElementById('theme-light').disabled = true;
echo             document.getElementById('theme-dark').disabled = false;
echo         } else {
echo             document.documentElement.removeAttribute('data-theme');
echo             document.getElementById('theme-light').disabled = false;
echo             document.getElementById('theme-dark').disabled = true;
echo         }
echo         localStorage.setItem('theme', theme);
echo         this.currentTheme = theme;
echo     }
echo.
echo     toggle() {
echo         const newTheme = this.currentTheme === 'light' ? 'dark' : 'light';
echo         this.applyTheme(newTheme);
echo     }
echo }
echo.
echo // Initialize theme manager
echo window.themeManager = new ThemeManager();
echo.
echo // Theme toggle button
echo document.getElementById('theme-toggle')?.addEventListener('click', () => {
echo     window.themeManager.toggle();
echo });
) > "%PROJECT_ROOT%\web\js\theme.js"

(
echo // Application entry point
echo document.addEventListener('DOMContentLoaded', () => {
echo     console.log('SIM800C Supervisor starting...');
echo.
echo     // Connect WebSocket
echo     window.wsManager.connect();
echo.
echo     // Load modules
echo     window.dashboardManager.loadModules();
echo.
echo     // Auto-discovery buttons
echo     document.getElementById('btn-status-auto')?.addEventListener('click', async () => {
echo         const response = await fetch('/api/ussd/auto-status', { method: 'POST' });
echo         const data = await response.json();
echo         alert('Auto-discovery SIM Status lancée! Résultats dans le dashboard.');
echo     });
echo.
echo     document.getElementById('btn-menu-auto')?.addEventListener('click', async () => {
echo         const response = await fetch('/api/ussd/auto-menu', { method: 'POST' });
echo         const data = await response.json();
echo         alert('Auto-discovery Menu USSD lancée! Résultats dans le dashboard.');
echo     });
echo.
echo     // Refresh button
echo     document.getElementById('refresh-all')?.addEventListener('click', () => {
echo         window.dashboardManager.loadModules();
echo     });
echo.
echo     // WebSocket event handlers
echo     window.wsManager.on('module_update', (data) => {
echo         console.log('Module update:', data);
echo         window.dashboardManager.loadModules();
echo     });
echo.
echo     window.wsManager.on('ussd_result', (data) => {
echo         console.log('USSD result:', data);
echo         const resultDiv = document.getElementById(`result-${data.module_id}`);
echo         if (resultDiv) {
echo             resultDiv.innerHTML = `<pre class="result">${JSON.stringify(data.result, null, 2)}</pre>`;
echo         }
echo     });
echo });
) > "%PROJECT_ROOT%\web\js\app.js"

:: Create placeholder for other JS files
echo // SMS Manager placeholder > "%PROJECT_ROOT%\web\js\sms.js"
echo // USSD Manager placeholder > "%PROJECT_ROOT%\web\js\ussd.js"

echo OK.

:: Generate SQL script
echo [8/8] Generating SQL database script...
(
echo -- SIM800C Supervisor Database Schema
echo CREATE DATABASE IF NOT EXISTS sim800c_manager_deepseekv1;
echo USE sim800c_manager_deepseekv1;
echo.
echo -- Modules table
echo CREATE TABLE IF NOT EXISTS modules (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     com_port VARCHAR(10) NOT NULL UNIQUE,
echo     imei VARCHAR(15),
echo     phone_number VARCHAR(20),
echo     carrier VARCHAR(50),
echo     status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
echo     last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     INDEX idx_status (status),
echo     INDEX idx_carrier (carrier)
echo );
echo.
echo -- USSD History
echo CREATE TABLE IF NOT EXISTS ussd_history (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     module_id INT NOT NULL,
echo     ussd_code VARCHAR(50) NOT NULL,
echo     input_data TEXT,
echo     output_data TEXT,
echo     status ENUM('success', 'error', 'timeout') NOT NULL,
echo     duration_ms INT,
echo     executed_by VARCHAR(50) DEFAULT 'system',
echo     executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
echo     INDEX idx_module (module_id),
echo     INDEX idx_executed_at (executed_at)
echo );
echo.
echo -- SMS Messages
echo CREATE TABLE IF NOT EXISTS sms_messages (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     module_id INT NOT NULL,
echo     sender_number VARCHAR(20),
echo     receiver_number VARCHAR(20),
echo     message TEXT NOT NULL,
echo     direction ENUM('in', 'out') NOT NULL,
echo     is_deleted BOOLEAN DEFAULT FALSE,
echo     is_trash BOOLEAN DEFAULT FALSE,
echo     sms_index INT,
echo     received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
echo     INDEX idx_module_trash (module_id, is_trash),
echo     INDEX idx_received_at (received_at)
echo );
echo.
echo -- Excel Versions
echo CREATE TABLE IF NOT EXISTS excel_versions (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     filename VARCHAR(255) NOT NULL,
echo     version_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     created_by VARCHAR(50) DEFAULT 'system',
echo     new_codes_count INT DEFAULT 0,
echo     INDEX idx_version_date (version_date)
echo );
echo.
echo -- USSD Favorites
echo CREATE TABLE IF NOT EXISTS ussd_favorites (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     user_id VARCHAR(50) NOT NULL DEFAULT 'default',
echo     ussd_code_id INT,
echo     ussd_code VARCHAR(50) NOT NULL,
echo     carrier VARCHAR(50),
echo     operation VARCHAR(100),
echo     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     INDEX idx_user (user_id)
echo );
echo.
echo -- Audit Log
echo CREATE TABLE IF NOT EXISTS audit_log (
echo     id INT AUTO_INCREMENT PRIMARY KEY,
echo     user_id VARCHAR(50),
echo     action VARCHAR(100) NOT NULL,
echo     target_type VARCHAR(50),
echo     target_id INT,
echo     details JSON,
echo     ip_address VARCHAR(45),
echo     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     INDEX idx_user_action (user_id, action),
echo     INDEX idx_created_at (created_at)
echo );
echo.
echo -- Insert default modules
echo INSERT IGNORE INTO modules (com_port, status) VALUES 
echo ('COM5', 'disconnected'),
echo ('COM6', 'disconnected'),
echo ('COM7', 'disconnected');
echo.
echo SELECT 'Database created successfully!' AS Status;
) > "%PROJECT_ROOT%\scripts\init_db.sql"

:: Generate service installation script
(
echo @echo off
echo echo Installing SIM800C Supervisor as Windows Service...
echo.
echo set SERVICE_NAME=SIM800CSupervisor
echo set BINARY_PATH=%~dp0..\sim800c.exe
echo set CONFIG_PATH=%~dp0..\config.yaml
echo.
echo sc create "%SERVICE_NAME%" binPath= "%BINARY_PATH% --config %CONFIG_PATH%" start= auto
echo sc description "%SERVICE_NAME%" "Gestion et supervision des modules SIM800C"
echo sc start "%SERVICE_NAME%"
echo.
echo echo Service installed successfully!
echo pause
) > "%PROJECT_ROOT%\scripts\install_service.bat"

:: Generate README
(
echo # SIM800C Supervisor
echo.
echo ## Description
echo Application de supervision et gestion des modules SIM800C USB.
echo.
echo ## Installation Rapide
echo.
echo 1. Installer Go 1.22+ ^(https://golang.org/dl/^)
echo 2. Installer XAMPP ^(https://www.apachefriends.org/^)
echo 3. Démarrer MySQL dans XAMPP
echo 4. Exécuter le script SQL: `mysql -u root ^< scripts/init_db.sql`
echo 5. Installer les dépendances Go: `go mod tidy`
echo 6. Compiler: `go build -o sim800c.exe cmd/main.go`
echo 7. Lancer: `sim800c.exe --config config.yaml`
echo.
echo ## Accès
echo - Frontend: http://test_sim800c.local
echo - API: http://localhost:8080/api/modules
echo.
echo ## Configuration
echo Modifier `config.yaml` pour ajuster les ports COM et les paramètres.
echo.
echo ## Support
echo Voir la documentation complète dans le dossier `docs/`
) > "%PROJECT_ROOT%\README.md"

echo.
echo ================================================================================
echo                        PROJECT GENERATED SUCCESSFULLY!
echo ================================================================================
echo.
echo Location: %PROJECT_ROOT%
echo.
echo Next steps:
echo 1. cd %PROJECT_ROOT%
echo 2. Copy your Excel file to storage/excel/Codes_USSD_CI.xlsx
echo 3. Run: mysql -u root -p ^< scripts/init_db.sql
echo 4. Run: go mod tidy
echo 5. Run: go build -o sim800c.exe cmd/main.go
echo 6. Run: sim800c.exe --config config.yaml
echo 7. Open browser: http://test_sim800c.local
echo.
echo For more details, read the README.md file.
echo.
pause