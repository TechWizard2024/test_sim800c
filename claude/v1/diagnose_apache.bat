@echo off
setlocal enabledelayedexpansion
echo ============================================
echo  SIM800C - Diagnostic Complet
echo ============================================
echo.

echo [1] Verification des modules proxy dans httpd.conf...
set HTTPD_CONF=C:\xampp\apache\conf\httpd.conf
if not exist "%HTTPD_CONF%" (
    echo   [ERREUR] httpd.conf introuvable : %HTTPD_CONF%
    goto :check_backend
)

for %%M in (mod_proxy.so mod_proxy_http.so mod_proxy_wstunnel.so) do (
    powershell -Command "$lines = Get-Content '%HTTPD_CONF%' | Where-Object { $_ -match '%%M' -and $_ -notmatch '^\s*#' }; if ($lines) { Write-Host '  [OK] %%M active' } else { Write-Host '  [MANQUANT] %%M - a activer dans httpd.conf' }"
)

echo.
echo [2] Verification du vhosts conf...
set VHOSTS_CONF=C:\xampp\apache\conf\extra\httpd-vhosts.conf
if not exist "%VHOSTS_CONF%" (
    echo   [ERREUR] httpd-vhosts.conf introuvable
) else (
    powershell -Command "$c = Get-Content '%VHOSTS_CONF%' -Raw; if ($c -match 'ProxyPass /api/ws') { Write-Host '  [OK] ProxyPass /api/ws present' } elseif ($c -match 'ProxyPass /ws') { Write-Host '  [ANCIEN] ProxyPass /ws - doit etre mis a jour vers /api/ws' } else { Write-Host '  [MANQUANT] Aucun ProxyPass WebSocket' }"
    powershell -Command "$c = Get-Content '%VHOSTS_CONF%' -Raw; if ($c -match 'ProxyPass /api ') { Write-Host '  [OK] ProxyPass /api present' } else { Write-Host '  [MANQUANT] ProxyPass /api absent' }"
)

echo.
:check_backend
echo [3] Processus sim800c-supervisor.exe en cours ?
tasklist /FI "IMAGENAME eq sim800c-supervisor.exe" 2>NUL | find /I "sim800c-supervisor.exe" >NUL
if !errorlevel! equ 0 (
    echo   [OK] sim800c-supervisor.exe est en cours d'execution
    tasklist /FI "IMAGENAME eq sim800c-supervisor.exe" /FO TABLE /NH 2>NUL
) else (
    echo   [ERREUR] sim800c-supervisor.exe ne tourne PAS
    echo   Lancez .\start_app.bat pour demarrer l'application
)

echo.
echo [4] Port 8082 en ecoute ?
netstat -ano 2>NUL | find ":8082" | find "LISTENING" >NUL
if !errorlevel! equ 0 (
    echo   [OK] Port 8082 en ecoute
    netstat -ano 2>NUL | find ":8082" | find "LISTENING"
) else (
    echo   [ERREUR] Port 8082 pas en ecoute - le backend n'a pas demarre correctement
    echo   Verifiez les logs:
    if exist "storage\logs\runtime.log" (
        echo   --- runtime.log ^(10 dernieres lignes^) ---
        powershell -Command "Get-Content 'storage\logs\runtime.log' -Tail 10 | ForEach-Object { Write-Host '  ' $_ }"
    )
    if exist "storage\logs\runtime_err.log" (
        echo   --- runtime_err.log ^(erreurs critiques^) ---
        powershell -Command "Get-Content 'storage\logs\runtime_err.log' -Tail 10 | ForEach-Object { Write-Host '  ' $_ }"
    )
)

echo.
echo [5] Verification backend directement sur port 8082...
powershell -Command "try { $r = Invoke-WebRequest -Uri 'http://localhost:8082/api/health' -TimeoutSec 5 -UseBasicParsing; Write-Host '  [OK] Backend repond! Status:' $r.StatusCode '- Acces direct: http://localhost:8082/' } catch { Write-Host '  [ERREUR] Backend inaccessible sur 8082:' $_.Exception.Message }"

echo.
echo [6] Port 80 Apache en ecoute ?
netstat -ano 2>NUL | find ":80 " | find "LISTENING" >NUL
if !errorlevel! equ 0 (
    echo   [OK] Port 80 en ecoute
) else (
    echo   [ERREUR] Port 80 pas en ecoute - Apache non demarre ?
)

echo.
echo [7] Test proxy Apache...
powershell -Command "try { $r = Invoke-WebRequest -Uri 'http://test-sim800c.lan/api/health' -TimeoutSec 5 -UseBasicParsing; Write-Host '  [OK] Proxy Apache fonctionne! Status:' $r.StatusCode } catch { Write-Host '  [ERREUR] Proxy Apache echoue:' $_.Exception.Message }"

echo.
echo [8] Logs Apache recents...
set ERR_LOG=C:\xampp\apache\logs\sim800c_error.log
if exist "%ERR_LOG%" (
    echo   Dernieres lignes de sim800c_error.log:
    powershell -Command "Get-Content '%ERR_LOG%' -Tail 5 | ForEach-Object { Write-Host '  ' $_ }"
)

echo.
echo [9] Logs runtime Go recents...
if exist "storage\logs\runtime.log" (
    echo   Dernieres lignes de runtime.log:
    powershell -Command "Get-Content 'storage\logs\runtime.log' -Tail 15 | ForEach-Object { Write-Host '  ' $_ }"
)

echo.
echo ============================================
echo  FIN DU DIAGNOSTIC
echo ============================================
pause
