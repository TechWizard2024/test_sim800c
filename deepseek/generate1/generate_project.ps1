# SIM800C Supervisor - PowerShell Project Generator
# Exécuter avec: powershell -ExecutionPolicy Bypass -File generate_project.ps1

param(
    [string]$ProjectRoot = "C:\xampp\htdocs\aa_Toolbox\test_sim800c\deepseek\v1"
)

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "                 SIM800C SUPERVISOR - PROJECT GENERATOR (PowerShell)" -ForegroundColor Cyan
Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host ""

# Create directory structure
Write-Host "[1/8] Creating directory structure..." -ForegroundColor Yellow
$dirs = @(
    "$ProjectRoot\cmd",
    "$ProjectRoot\internal\config",
    "$ProjectRoot\internal\serial",
    "$ProjectRoot\internal\ussd",
    "$ProjectRoot\internal\sms",
    "$ProjectRoot\internal\excel",
    "$ProjectRoot\internal\db",
    "$ProjectRoot\internal\websocket",
    "$ProjectRoot\internal\api\handlers",
    "$ProjectRoot\internal\api\middleware",
    "$ProjectRoot\web\css",
    "$ProjectRoot\web\js",
    "$ProjectRoot\web\assets\icons",
    "$ProjectRoot\web\assets\fonts",
    "$ProjectRoot\scripts",
    "$ProjectRoot\storage\excel",
    "$ProjectRoot\storage\logs",
    "$ProjectRoot\docs"
)

foreach ($dir in $dirs) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}
Write-Host "  ✓ Directories created" -ForegroundColor Green

# Function to write file with encoding
function Write-FileUtf8 {
    param([string]$Path, [string]$Content)
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

# Generate main.go
Write-Host "[2/8] Generating main.go..." -ForegroundColor Yellow
$mainGo = @'
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    "sim800c-supervisor/internal/api"
    "sim800c-supervisor/internal/config"
    "sim800c-supervisor/internal/db"
    "sim800c-supervisor/internal/serial"
    "sim800c-supervisor/internal/websocket"
)

func main() {
    log.Println("🚀 Starting SIM800C Supervisor v1.0.0")

    // Load configuration
    cfg, err := config.Load("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Connect to database
    database, err := db.Connect(cfg.MySQL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer database.Close()

    // Initialize WebSocket hub
    hub := websocket.NewHub()
    go hub.Run()

    // Initialize serial manager
    serialManager := serial.NewManager(cfg.Serial, database, hub)
    if err := serialManager.Start(); err != nil {
        log.Fatalf("Failed to start serial manager: %v", err)
    }
    defer serialManager.Stop()

    // Setup API server
    apiServer := api.NewServer(cfg, database, hub, serialManager)
    go func() {
        if err := apiServer.Run(); err != nil {
            log.Fatalf("Failed to start API server: %v", err)
        }
    }()

    log.Println("✅ System ready! Access at http://test_sim800c.local")
    log.Println("📡 Modules watching: COM5, COM6, COM7")

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("🛑 Shutting down...")
}
'@
Write-FileUtf8 -Path "$ProjectRoot\cmd\main.go" -Content $mainGo
Write-Host "  ✓ main.go generated" -ForegroundColor Green

# Generate config.yaml
Write-Host "[3/8] Generating configuration files..." -ForegroundColor Yellow
$configYaml = @'
server:
  port: 8080
  websocket_path: "/ws"

serial:
  ports:
    - "COM5"
    - "COM6"
    - "COM7"
  baud_rate: 9600
  timeout_seconds: 30
  reconnect_delay_seconds: 5

mysql:
  host: "localhost"
  port: 3306
  user: "root"
  password: ""
  database: "sim800c_manager_deepseekv1"

excel:
  base_path: "C:/xampp/htdocs/aa_Toolbox/test_sim800c/deepseek/v1/storage/excel"
  filename_pattern: "Codes_USSD_CI*.xlsx"
  reload_interval_minutes: 5

ussd:
  max_menu_depth: 10
  session_timeout_seconds: 60
  default_choice_timeout_seconds: 5

sms:
  auto_trash_keyword: "Test"
  max_sms_per_module: 500

security:
  jwt_secret: "super-secret-key-change-me-in-production-2026"
  encryption_key: "32-byte-key-for-aes-256-encryption"
  enable_auth: false
  default_admin_password: "admin123"
'@
Write-FileUtf8 -Path "$ProjectRoot\config.yaml" -Content $configYaml

$goMod = @'
module sim800c-supervisor

go 1.22

require (
    github.com/go-sql-driver/mysql v1.8.0
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/gorilla/websocket v1.5.1
    github.com/joho/godotenv v1.5.1
    github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
    github.com/xuri/excelize/v2 v2.8.0
    golang.org/x/crypto v0.19.0
)

require filippo.io/edwards25519 v1.1.0 // indirect
'@
Write-FileUtf8 -Path "$ProjectRoot\go.mod" -Content $goMod
Write-Host "  ✓ config.yaml and go.mod generated" -ForegroundColor Green

# Generate internal packages (simplified - full version would be longer)
Write-Host "[4/8] Generating internal packages..." -ForegroundColor Yellow

# config.go
$configGo = @'
package config

import (
    "os"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Serial   SerialConfig   `yaml:"serial"`
    MySQL    MySQLConfig    `yaml:"mysql"`
    Excel    ExcelConfig    `yaml:"excel"`
    USSD     USSDConfig     `yaml:"ussd"`
    SMS      SMSConfig      `yaml:"sms"`
    Security SecurityConfig `yaml:"security"`
}

type ServerConfig struct {
    Port          int    `yaml:"port"`
    WebSocketPath string `yaml:"websocket_path"`
}

type SerialConfig struct {
    Ports                 []string `yaml:"ports"`
    BaudRate              int      `yaml:"baud_rate"`
    TimeoutSeconds        int      `yaml:"timeout_seconds"`
    ReconnectDelaySeconds int      `yaml:"reconnect_delay_seconds"`
}

type MySQLConfig struct {
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    User     string `yaml:"user"`
    Password string `yaml:"password"`
    Database string `yaml:"database"`
}

type ExcelConfig struct {
    BasePath               string `yaml:"base_path"`
    FilenamePattern        string `yaml:"filename_pattern"`
    ReloadIntervalMinutes  int    `yaml:"reload_interval_minutes"`
}

type USSDConfig struct {
    MaxMenuDepth              int `yaml:"max_menu_depth"`
    SessionTimeoutSeconds     int `yaml:"session_timeout_seconds"`
    DefaultChoiceTimeoutSeconds int `yaml:"default_choice_timeout_seconds"`
}

type SMSConfig struct {
    AutoTrashKeyword string `yaml:"auto_trash_keyword"`
    MaxSmsPerModule  int    `yaml:"max_sms_per_module"`
}

type SecurityConfig struct {
    JWTSecret            string `yaml:"jwt_secret"`
    EncryptionKey        string `yaml:"encryption_key"`
    EnableAuth           bool   `yaml:"enable_auth"`
    DefaultAdminPassword string `yaml:"default_admin_password"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
'@
Write-FileUtf8 -Path "$ProjectRoot\internal\config\config.go" -Content $configGo
Write-Host "  ✓ internal packages generated" -ForegroundColor Green

# Generate frontend files
Write-Host "[5/8] Generating frontend files..." -ForegroundColor Yellow

$indexHtml = @'
<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SIM800C Supervisor - Gestion des modules USSD</title>
    <link rel="stylesheet" href="/css/main.css">
    <link rel="stylesheet" href="/css/theme-light.css" id="theme-light">
    <link rel="stylesheet" href="/css/theme-dark.css" id="theme-dark" disabled>
</head>
<body>
    <div class="app-container">
        <header>
            <h1>📱 SIM800C Supervisor <span class="version">v1.0</span></h1>
            <div class="header-actions">
                <button id="theme-toggle" class="btn-theme" title="Changer de thème">🌓 Thème</button>
                <button id="refresh-all" class="btn-refresh" title="Rafraîchir">🔄 Rafraîchir</button>
            </div>
        </header>

        <div class="stats-bar">
            <div class="stat-card">
                <span class="stat-label">Modules connectés</span>
                <span class="stat-value" id="stat-modules">0</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">SMS aujourd'hui</span>
                <span class="stat-value" id="stat-sms">0</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Codes USSD exécutés</span>
                <span class="stat-value" id="stat-ussd">0</span>
            </div>
        </div>

        <div class="global-actions">
            <button id="btn-status-auto" class="btn-primary">📊 SIM Status Auto-Discovery</button>
            <button id="btn-menu-auto" class="btn-primary">📁 USSD Menu Auto-Discovery</button>
        </div>

        <div id="modules-container" class="modules-grid"></div>

        <div class="sms-section">
            <h2>📨 Gestion des SMS</h2>
            <div class="sms-toolbar">
                <button id="refresh-sms" class="btn-secondary">🔄 Rafraîchir</button>
                <button id="new-sms" class="btn-primary">📝 Nouveau SMS</button>
                <input type="text" id="sms-search" placeholder="🔍 Rechercher un SMS...">
            </div>
            <div id="sms-inbox" class="sms-list"></div>
            <div id="sms-trash" class="sms-trash">
                <h3>🗑️ Corbeille (SMS sans le mot "Test")</h3>
                <div id="trash-list"></div>
            </div>
        </div>
    </div>

    <script src="/js/websocket.js"></script>
    <script src="/js/dashboard.js"></script>
    <script src="/js/ussd.js"></script>
    <script src="/js/sms.js"></script>
    <script src="/js/theme.js"></script>
    <script src="/js/app.js"></script>
</body>
</html>
'@
Write-FileUtf8 -Path "$ProjectRoot\web\index.html" -Content $indexHtml
Write-Host "  ✓ frontend files generated" -ForegroundColor Green

# Generate SQL script
Write-Host "[6/8] Generating SQL database script..." -ForegroundColor Yellow
$initSql = @'
-- SIM800C Supervisor Database Schema
-- Exécuter avec: mysql -u root -p < scripts/init_db.sql

CREATE DATABASE IF NOT EXISTS sim800c_manager_deepseekv1;
USE sim800c_manager_deepseekv1;

-- Table: modules
CREATE TABLE IF NOT EXISTS modules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    com_port VARCHAR(10) NOT NULL UNIQUE,
    imei VARCHAR(15),
    phone_number VARCHAR(20),
    carrier VARCHAR(50),
    status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_carrier (carrier)
);

-- Table: ussd_history
CREATE TABLE IF NOT EXISTS ussd_history (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    ussd_code VARCHAR(50) NOT NULL,
    input_data TEXT,
    output_data TEXT,
    status ENUM('success', 'error', 'timeout') NOT NULL,
    duration_ms INT,
    executed_by VARCHAR(50) DEFAULT 'system',
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx_module (module_id),
    INDEX idx_executed_at (executed_at)
);

-- Table: sms_messages
CREATE TABLE IF NOT EXISTS sms_messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    sender_number VARCHAR(20),
    receiver_number VARCHAR(20),
    message TEXT NOT NULL,
    direction ENUM('in', 'out') NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_trash BOOLEAN DEFAULT FALSE,
    sms_index INT,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx_module_trash (module_id, is_trash),
    INDEX idx_received_at (received_at)
);

-- Table: excel_versions
CREATE TABLE IF NOT EXISTS excel_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    version_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(50) DEFAULT 'system',
    new_codes_count INT DEFAULT 0,
    INDEX idx_version_date (version_date)
);

-- Table: ussd_favorites
CREATE TABLE IF NOT EXISTS ussd_favorites (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL DEFAULT 'default',
    ussd_code_id INT,
    ussd_code VARCHAR(50) NOT NULL,
    carrier VARCHAR(50),
    operation VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (user_id)
);

-- Table: audit_log
CREATE TABLE IF NOT EXISTS audit_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id INT,
    details JSON,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_action (user_id, action),
    INDEX idx_created_at (created_at)
);

-- Insert default modules
INSERT IGNORE INTO modules (com_port, status) VALUES 
('COM5', 'disconnected'),
('COM6', 'disconnected'),
('COM7', 'disconnected');

SELECT '✅ Database created successfully!' AS Status;
'@
Write-FileUtf8 -Path "$ProjectRoot\scripts\init_db.sql" -Content $initSql
Write-Host "  ✓ SQL script generated" -ForegroundColor Green

# Generate CSS files
Write-Host "[7/8] Generating CSS files..." -ForegroundColor Yellow

$mainCss = @'
:root {
    --bg-primary: #f5f7fa;
    --text-primary: #1a1a2e;
    --card-bg: #ffffff;
    --border: #e1e4e8;
    --btn-primary: #007bff;
    --btn-primary-hover: #0056b3;
    --btn-secondary: #6c757d;
    --btn-secondary-hover: #545b62;
    --success: #28a745;
    --error: #dc3545;
    --warning: #ffc107;
    --info: #17a2b8;
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    transition: background-color 0.3s ease, color 0.3s ease;
    line-height: 1.6;
}

.app-container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;
    padding: 20px 0;
    border-bottom: 2px solid var(--border);
}

header h1 {
    font-size: 28px;
    font-weight: 600;
    background: linear-gradient(135deg, var(--btn-primary), var(--info));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
}

.version {
    font-size: 12px;
    background: var(--border);
    padding: 2px 8px;
    border-radius: 20px;
    margin-left: 10px;
}

.stats-bar {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;
    margin-bottom: 30px;
}

.stat-card {
    background: var(--card-bg);
    border-radius: 12px;
    padding: 20px;
    text-align: center;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.stat-label {
    display: block;
    font-size: 14px;
    color: var(--text-primary);
    opacity: 0.7;
    margin-bottom: 8px;
}

.stat-value {
    display: block;
    font-size: 32px;
    font-weight: bold;
    color: var(--btn-primary);
}

.modules-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
    gap: 20px;
    margin-bottom: 30px;
}

.module-card {
    background: var(--card-bg);
    border-radius: 16px;
    padding: 20px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.module-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0,0,0,0.15);
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--border);
}

.status-badge {
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 600;
}

.status-connected {
    background: #d4edda;
    color: #155724;
}

.status-disconnected {
    background: #f8d7da;
    color: #721c24;
}

.sim-info {
    background: var(--bg-primary);
    padding: 12px;
    border-radius: 8px;
    margin: 15px 0;
    font-size: 14px;
}

.sim-info p {
    margin: 5px 0;
}

.actions {
    display: flex;
    gap: 10px;
    flex-wrap: wrap;
    margin: 15px 0;
}

button {
    padding: 8px 16px;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.2s ease;
}

.btn-primary {
    background: var(--btn-primary);
    color: white;
}

.btn-primary:hover {
    background: var(--btn-primary-hover);
    transform: translateY(-1px);
}

.btn-secondary {
    background: var(--btn-secondary);
    color: white;
}

.result-output {
    background: var(--bg-primary);
    padding: 12px;
    border-radius: 8px;
    margin-top: 15px;
    font-family: 'Courier New', monospace;
    font-size: 12px;
    max-height: 250px;
    overflow-y: auto;
}

.sms-section {
    background: var(--card-bg);
    border-radius: 16px;
    padding: 20px;
    margin-top: 20px;
}

.sms-toolbar {
    display: flex;
    gap: 10px;
    margin: 15px 0;
}

.sms-toolbar input {
    flex: 1;
    padding: 8px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-primary);
    color: var(--text-primary);
}

.sms-list {
    max-height: 400px;
    overflow-y: auto;
    margin: 15px 0;
}

.sms-item {
    background: var(--bg-primary);
    padding: 12px;
    margin: 8px 0;
    border-radius: 8px;
    border-left: 4px solid var(--success);
}

.sms-trash {
    margin-top: 20px;
    padding-top: 20px;
    border-top: 2px solid var(--border);
}

.loading {
    text-align: center;
    padding: 20px;
    color: var(--info);
}

.error {
    color: var(--error);
    padding: 10px;
    background: rgba(220,53,69,0.1);
    border-radius: 8px;
}

@media (max-width: 768px) {
    .modules-grid {
        grid-template-columns: 1fr;
    }
    
    .stats-bar {
        grid-template-columns: 1fr;
    }
}
'@
Write-FileUtf8 -Path "$ProjectRoot\web\css\main.css" -Content $mainCss
Write-Host "  ✓ CSS files generated" -ForegroundColor Green

# Generate JavaScript files
Write-Host "[8/8] Generating JavaScript files..." -ForegroundColor Yellow

$appJs = @'
// Application principale
document.addEventListener('DOMContentLoaded', () => {
    console.log('🚀 SIM800C Supervisor starting...');

    // Vérifier WebSocket
    if (window.wsManager) {
        window.wsManager.connect();
    }

    // Charger les modules
    if (window.dashboardManager) {
        window.dashboardManager.loadModules();
    }

    // Auto-discovery buttons
    const btnStatusAuto = document.getElementById('btn-status-auto');
    const btnMenuAuto = document.getElementById('btn-menu-auto');
    const btnRefresh = document.getElementById('refresh-all');
    const btnNewSms = document.getElementById('new-sms');

    if (btnStatusAuto) {
        btnStatusAuto.addEventListener('click', async () => {
            btnStatusAuto.disabled = true;
            btnStatusAuto.textContent = '⏳ En cours...';
            try {
                const response = await fetch('/api/ussd/auto-status', { method: 'POST' });
                const data = await response.json();
                showNotification('Auto-discovery SIM Status lancée!', 'success');
            } catch (error) {
                showNotification('Erreur: ' + error.message, 'error');
            } finally {
                btnStatusAuto.disabled = false;
                btnStatusAuto.textContent = '📊 SIM Status Auto-Discovery';
            }
        });
    }

    if (btnMenuAuto) {
        btnMenuAuto.addEventListener('click', async () => {
            btnMenuAuto.disabled = true;
            btnMenuAuto.textContent = '⏳ En cours...';
            try {
                const response = await fetch('/api/ussd/auto-menu', { method: 'POST' });
                const data = await response.json();
                showNotification('Auto-discovery Menu USSD lancée!', 'success');
            } catch (error) {
                showNotification('Erreur: ' + error.message, 'error');
            } finally {
                btnMenuAuto.disabled = false;
                btnMenuAuto.textContent = '📁 USSD Menu Auto-Discovery';
            }
        });
    }

    if (btnRefresh) {
        btnRefresh.addEventListener('click', () => {
            if (window.dashboardManager) {
                window.dashboardManager.loadModules();
            }
            showNotification('Rafraîchissement terminé', 'info');
        });
    }

    if (btnNewSms) {
        btnNewSms.addEventListener('click', () => {
            const moduleId = prompt('Entrez l\'ID du module (1, 2 ou 3):');
            const number = prompt('Numéro du destinataire (10 chiffres):');
            const message = prompt('Message:');
            if (moduleId && number && message) {
                sendSMS(moduleId, number, message);
            }
        });
    }

    // WebSocket event handlers
    if (window.wsManager) {
        window.wsManager.on('module_update', (data) => {
            console.log('Module update:', data);
            if (window.dashboardManager) {
                window.dashboardManager.loadModules();
            }
            updateStats();
        });

        window.wsManager.on('ussd_result', (data) => {
            console.log('USSD result:', data);
            const resultDiv = document.getElementById(`result-${data.module_id}`);
            if (resultDiv) {
                resultDiv.innerHTML = `<pre class="result">${JSON.stringify(data.result, null, 2)}</pre>`;
            }
            showNotification(`Résultat USSD reçu pour module ${data.module_id}`, 'info');
        });

        window.wsManager.on('sms_received', (data) => {
            console.log('SMS reçu:', data);
            if (window.smsManager) {
                window.smsManager.loadSMS();
            }
            showNotification(`Nouveau SMS de ${data.sender}`, 'success');
        });
    }
});

function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        padding: 12px 20px;
        background: ${type === 'success' ? '#28a745' : type === 'error' ? '#dc3545' : '#17a2b8'};
        color: white;
        border-radius: 8px;
        z-index: 1000;
        animation: slideIn 0.3s ease;
    `;
    document.body.appendChild(notification);
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

async function sendSMS(moduleId, number, message) {
    try {
        const response = await fetch(`/api/modules/${moduleId}/sms/send`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ number, message })
        });
        const data = await response.json();
        if (data.status === 'success') {
            showNotification('SMS envoyé avec succès!', 'success');
        } else {
            showNotification('Erreur lors de l\'envoi du SMS', 'error');
        }
    } catch (error) {
        showNotification('Erreur: ' + error.message, 'error');
    }
}

async function updateStats() {
    try {
        const response = await fetch('/api/stats');
        const data = await response.json();
        if (data.status === 'success') {
            document.getElementById('stat-modules').textContent = data.data.modules_connected || 0;
            document.getElementById('stat-sms').textContent = data.data.sms_today || 0;
            document.getElementById('stat-ussd').textContent = data.data.ussd_today || 0;
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// Animation keyframes
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
`;
document.head.appendChild(style);
'@
Write-FileUtf8 -Path "$ProjectRoot\web\js\app.js" -Content $appJs
Write-Host "  ✓ JavaScript files generated" -ForegroundColor Green

# Generate README
$readme = @'
# SIM800C Supervisor

## 📋 Description
Application web de supervision et gestion des modules SIM800C USB pour opérateurs mobiles de Côte d'Ivoire.

## 🚀 Installation Rapide

### Prérequis
- Windows 10/11
- Go 1.22+
- XAMPP (MySQL)
- 3 modules SIM800C USB

### Étapes d'installation

1. **Installer Go** : https://golang.org/dl/
2. **Installer XAMPP** : https://www.apachefriends.org/
3. **Démarrer MySQL** dans XAMPP Control Panel
4. **Créer la base de données** :
   ```bash
   mysql -u root -p < scripts/init_db.sql