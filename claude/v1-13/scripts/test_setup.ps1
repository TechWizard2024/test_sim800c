Write-Host "Test de l'environnement SIM800C Supervisor" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# Test 1: Vérifier Go
$goVersion = go version
if ($goVersion) {
    Write-Host "[OK] Go installé: $goVersion" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Go non trouvé" -ForegroundColor Red
}

# Test 2: Vérifier MySQL
$mysqlTest = mysql -u sim800c_user -pSIM800c@2026! -e "SELECT 1" 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] MySQL accessible" -ForegroundColor Green
} else {
    Write-Host "[FAIL] MySQL inaccessible" -ForegroundColor Red
}

# Test 3: Vérifier les ports COM
$comPorts = @("COM5", "COM6", "COM7")
foreach ($port in $comPorts) {
    $portTest = Test-Path "\\.\$port"
    if ($portTest) {
        Write-Host "[OK] Port $port disponible" -ForegroundColor Green
    } else {
        Write-Host "[WARN] Port $port non trouvé" -ForegroundColor Yellow
    }
}

# Test 4: Vérifier les fichiers
$requiredFiles = @(
    "cmd\main.go",
    "config.yaml",
    "web\index.html"
)
foreach ($file in $requiredFiles) {
    if (Test-Path $file) {
        Write-Host "[OK] $file présent" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $file manquant" -ForegroundColor Red
    }
}

Write-Host "`nTest terminé" -ForegroundColor Cyan