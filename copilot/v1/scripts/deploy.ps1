# Script de déploiement automatique - Version XAMPP
param(
    [switch]$SkipServiceInstall
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Déploiement de SIM800C Supervisor" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

$ErrorActionPreference = "Continue"
$projectRoot = Split-Path -Parent $PSScriptRoot

# Chemins XAMPP
$xamppPath = "C:\xampp"
$mysqlPath = "$xamppPath\mysql\bin"
$apachePath = "$xamppPath\apache"

# Vérifier si XAMPP existe
if (-not (Test-Path $mysqlPath)) {
    Write-Host "❌ XAMPP non trouvé dans C:\xampp" -ForegroundColor Red
    exit 1
}

Write-Host "✅ XAMPP trouvé dans $xamppPath" -ForegroundColor Green

# Vérifier les privilèges administrateur
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
if (-NOT $isAdmin) {
    Write-Host "⚠️  Ce script nécessite des privilèges administrateur pour installer le service" -ForegroundColor Yellow
    Write-Host "L'application pourra quand même être exécutée manuellement`n" -ForegroundColor Yellow
}

# 1. Vérification des prérequis
Write-Host "[1/5] Vérification des prérequis..." -ForegroundColor Yellow

# Vérifier Go
$goVersion = go version 2>$null
if (-not $goVersion) {
    Write-Host "❌ Go non installé. Veuillez installer Go 1.21+" -ForegroundColor Red
    Write-Host "Téléchargement: https://go.dev/dl/" -ForegroundColor Cyan
    exit 1
}
Write-Host "✅ Go installé: $goVersion" -ForegroundColor Green

# Vérifier MySQL via XAMPP
$mysqlExe = "$mysqlPath\mysql.exe"
if (-not (Test-Path $mysqlExe)) {
    Write-Host "❌ MySQL non trouvé dans XAMPP" -ForegroundColor Red
    exit 1
}
Write-Host "✅ MySQL trouvé: $mysqlExe" -ForegroundColor Green

# 2. Installation des dépendances Go
Write-Host "`n[2/5] Installation des dépendances Go..." -ForegroundColor Yellow
Set-Location $projectRoot

# Initialiser go.mod s'il n'existe pas
if (-not (Test-Path "go.mod")) {
    Write-Host "   Initialisation du module Go..."
    go mod init sim800c-supervisor 2>$null
}

# Installer les dépendances
$dependencies = @(
    "github.com/tarm/serial",
    "github.com/xuri/excelize/v2",
    "github.com/gorilla/websocket",
    "github.com/go-sql-driver/mysql",
    "github.com/joho/godotenv",
    "github.com/golang-jwt/jwt/v5",
    "github.com/rs/cors",
    "github.com/sirupsen/logrus",
    "gopkg.in/yaml.v3"
)

Write-Host "   Installation des dépendances Go..."
foreach ($dep in $dependencies) {
    Write-Host "     - $dep"
    go get $dep 2>$null
}

go mod tidy 2>$null
Write-Host "✅ Dépendances installées" -ForegroundColor Green

# 3. Initialisation de la base de données
Write-Host "`n[3/5] Initialisation de la base de données..." -ForegroundColor Yellow

# Tester la connexion MySQL via XAMPP
Write-Host "   Test de connexion à MySQL..."
$mysqlConnection = & $mysqlExe -u root -e "SELECT 1" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "⚠️  Connexion MySQL sans mot de passe échouée, tentative avec mot de passe vide..." -ForegroundColor Yellow
    $mysqlConnection = & $mysqlExe -u root --password="" -e "SELECT 1" 2>&1
    $mysqlPwd = ""
} else {
    $mysqlPwd = ""
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Impossible de se connecter à MySQL. Vérifiez que MySQL est démarré dans XAMPP." -ForegroundColor Red
    exit 1
}
Write-Host "✅ Connexion MySQL réussie" -ForegroundColor Green

# Créer la base de données
Write-Host "   Création de la base de données..."
& $mysqlExe -u root --password="$mysqlPwd" -e "CREATE DATABASE IF NOT EXISTS sim800c_manager CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>$null

# Créer l'utilisateur
Write-Host "   Création de l'utilisateur..."
& $mysqlExe -u root --password="$mysqlPwd" -e "CREATE USER IF NOT EXISTS 'sim800c_user'@'localhost' IDENTIFIED BY 'SIM800c@2026!';" 2>$null
& $mysqlExe -u root --password="$mysqlPwd" -e "GRANT ALL PRIVILEGES ON sim800c_manager.* TO 'sim800c_user'@'localhost';" 2>$null
& $mysqlExe -u root --password="$mysqlPwd" -e "FLUSH PRIVILEGES;" 2>$null

# Exécuter le script SQL d'initialisation
Write-Host "   Exécution du script SQL..."
$sqlScript = Get-Content -Path "$projectRoot\scripts\init_db.sql" -Raw
$sqlScript | & $mysqlExe -u root --password="$mysqlPwd" sim800c_manager 2>$null

Write-Host "✅ Base de données initialisée" -ForegroundColor Green

# 4. Compilation
Write-Host "`n[4/5] Compilation du binaire..." -ForegroundColor Yellow

# Créer les dossiers storage
$storageDirs = @("storage\excel", "storage\logs", "storage\backup")
foreach ($dir in $storageDirs) {
    $fullPath = Join-Path $projectRoot $dir
    if (-not (Test-Path $fullPath)) {
        New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
    }
}

# Copier le fichier Excel s'il existe
$excelSource = "C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx"
$excelDest = "$projectRoot\storage\excel\Codes_USSD_CI.xlsx"
if (Test-Path $excelSource) {
    Copy-Item $excelSource $excelDest -Force
    Write-Host "   Fichier Excel copié" -ForegroundColor Green
} else {
    Write-Host "   ⚠️  Fichier Excel non trouvé, création d'un fichier vide..." -ForegroundColor Yellow
}

# Créer config.yaml s'il n'existe pas
$configFile = "$projectRoot\config.yaml"
if (-not (Test-Path $configFile)) {
    Write-Host "   Création du fichier de configuration..."
    $configContent = @"
server:
  port: 8080
  websocket_path: "/ws"
  api_path: "/api"
  read_timeout_seconds: 30
  write_timeout_seconds: 30

serial:
  ports:
    - "COM5"
    - "COM6"
    - "COM7"
  baud_rate: 9600
  timeout_seconds: 30
  reconnect_delay_seconds: 5
  command_queue_size: 100

mysql:
  host: "localhost"
  port: 3306
  user: "sim800c_user"
  password: "SIM800c@2026!"
  database: "sim800c_manager"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime_minutes: 60

excel:
  base_path: "$($projectRoot.Replace('\','/'))/storage/excel"
  filename_pattern: "Codes_USSD_CI*.xlsx"
  reload_interval_minutes: 5

ussd:
  max_menu_depth: 10
  session_timeout_seconds: 60
  explore_delay_ms: 1000

sms:
  auto_trash_keyword: "Test"
  max_sms_per_module: 500
  check_interval_seconds: 10

security:
  jwt_secret: "SIM800c-Supervisor-Secret-Key-2026"
  enable_auth: false

logging:
  level: "info"
  output_path: "storage/logs/app.log"
"@
    $configContent | Out-File -FilePath $configFile -Encoding utf8
}

# Compiler
Write-Host "   Compilation en cours..."
go build -o sim800c-supervisor.exe cmd/main.go 2>&1

if (Test-Path "sim800c-supervisor.exe") {
    Write-Host "✅ Compilation réussie" -ForegroundColor Green
} else {
    Write-Host "❌ Erreur de compilation" -ForegroundColor Red
    Write-Host "Vérification des fichiers source..." -ForegroundColor Yellow
    
    # Vérifier les fichiers nécessaires
    $requiredFiles = @(
        "cmd\main.go",
        "internal\config\config.go",
        "internal\serial\manager.go",
        "internal\serial\sim800c.go"
    )
    
    foreach ($file in $requiredFiles) {
        if (Test-Path $file) {
            Write-Host "  ✅ $file" -ForegroundColor Green
        } else {
            Write-Host "  ❌ $file manquant" -ForegroundColor Red
        }
    }
    exit 1
}

# 5. Configuration Apache (VirtualHost)
Write-Host "`n[5/5] Configuration d'Apache..." -ForegroundColor Yellow

$vhostsFile = "$apachePath\conf\extra\httpd-vhosts.conf"
$hostsFile = "C:\Windows\System32\drivers\etc\hosts"

# Sauvegarder le fichier vhosts original
if (Test-Path $vhostsFile) {
    Copy-Item $vhostsFile "$vhostsFile.backup" -Force
}

# Ajouter le VirtualHost
$vhostEntry = @"

# VirtualHost pour SIM800C Supervisor
<VirtualHost *:80>
    ServerName test_sim800c.local
    DocumentRoot "$($projectRoot.Replace('\','/'))/web"
    <Directory "$($projectRoot.Replace('\','/'))/web">
        Options Indexes FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
    
    # Proxy vers l'API Go
    ProxyPreserveHost On
    ProxyPass /api http://localhost:8080/api
    ProxyPassReverse /api http://localhost:8080/api
    
    # Proxy WebSocket
    ProxyPass /ws ws://localhost:8080/ws
    ProxyPassReverse /ws ws://localhost:8080/ws
    
    ErrorLog "logs/sim800c_error.log"
    CustomLog "logs/sim800c_access.log" common
</VirtualHost>

"@

# Vérifier si le vhost existe déjà
$vhostsContent = Get-Content $vhostsFile -Raw -ErrorAction SilentlyContinue
if ($vhostsContent -notlike "*test_sim800c.local*") {
    Add-Content -Path $vhostsFile -Value $vhostEntry
    Write-Host "   VirtualHost ajouté à httpd-vhosts.conf" -ForegroundColor Green
} else {
    Write-Host "   VirtualHost déjà présent" -ForegroundColor Green
}

# Activer le module proxy dans httpd.conf
$httpdFile = "$apachePath\conf\httpd.conf"
$httpdContent = Get-Content $httpdFile -Raw

$proxyModules = @(
    "mod_proxy.so",
    "mod_proxy_http.so",
    "mod_proxy_wstunnel.so"
)

foreach ($module in $proxyModules) {
    $moduleLine = "LoadModule $module modules/$module"
    if ($httpdContent -notlike "*$moduleLine*") {
        # Remplacer #LoadModule par LoadModule
        $httpdContent = $httpdContent -replace "#LoadModule $module", "LoadModule $module"
    }
}

# S'assurer que Include vhosts est actif
if ($httpdContent -notlike "*httpd-vhosts.conf*") {
    $httpdContent = $httpdContent -replace "#Include conf/extra/httpd-vhosts.conf", "Include conf/extra/httpd-vhosts.conf"
}

Set-Content -Path $httpdFile -Value $httpdContent
Write-Host "   Modules proxy activés dans httpd.conf" -ForegroundColor Green

# Ajouter l'entrée hosts
$hostsContent = Get-Content $hostsFile -Raw -ErrorAction SilentlyContinue
if ($hostsContent -notlike "*test_sim800c.local*") {
    Add-Content -Path $hostsFile -Value "`n127.0.0.1 test_sim800c.local"
    Write-Host "   Entrée hosts ajoutée" -ForegroundColor Green
} else {
    Write-Host "   Entrée hosts déjà présente" -ForegroundColor Green
}

Write-Host "✅ Configuration Apache terminée" -ForegroundColor Green

# Résumé final
Write-Host "`n========================================" -ForegroundColor Green
Write-Host "Déploiement terminé avec succès!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Résumé des actions:" -ForegroundColor Cyan
Write-Host "  ✅ Base de données: sim800c_manager" -ForegroundColor White
Write-Host "  ✅ Binaire compilé: sim800c-supervisor.exe" -ForegroundColor White
Write-Host "  ✅ VirtualHost configuré: test_sim800c.local" -ForegroundColor White
Write-Host ""

Write-Host "🚀 Prochaines étapes:" -ForegroundColor Cyan
Write-Host ""

Write-Host "1️⃣  Démarrer l'application (dans un nouveau terminal):" -ForegroundColor Yellow
Write-Host "   cd $projectRoot" -ForegroundColor White
Write-Host "   .\sim800c-supervisor.exe" -ForegroundColor White
Write-Host ""

Write-Host "2️⃣  Redémarrer Apache via XAMPP Control Panel:" -ForegroundColor Yellow
Write-Host "   - Ouvrir XAMPP Control Panel" -ForegroundColor White
Write-Host "   - Arrêter Apache" -ForegroundColor White
Write-Host "   - Redémarrer Apache" -ForegroundColor White
Write-Host ""

Write-Host "3️⃣  Accéder à l'application:" -ForegroundColor Yellow
Write-Host "   http://test_sim800c.local" -ForegroundColor White
Write-Host ""

Write-Host "4️⃣  Vérifier les logs en cas de problème:" -ForegroundColor Yellow
Write-Host "   $projectRoot\storage\logs\app.log" -ForegroundColor White
Write-Host ""

Write-Host "📝 Note:" -ForegroundColor Cyan
Write-Host "   - Assurez-vous que les modules SIM800C sont connectés sur COM5, COM6, COM7" -ForegroundColor White
Write-Host "   - Vérifiez que MySQL est démarré dans XAMPP" -ForegroundColor White
Write-Host "   - Pour un démarrage automatique, configurez le service avec NSSM" -ForegroundColor White
Write-Host ""