@echo off
setlocal enabledelayedexpansion
title SIM800C Supervisor - Demarrage

echo ========================================
echo   SIM800C Supervisor v1
echo ========================================
echo.

cd /d "%~dp0"

REM -----------------------------------------------
REM ETAPE 1/4 : Verification MySQL (XAMPP)
REM -----------------------------------------------
echo [1/4] Verification de MySQL (XAMPP)...
echo.

set MYSQL_RUNNING=0
tasklist /FI "IMAGENAME eq mysqld.exe" 2>NUL | find /I "mysqld.exe" >NUL
if %errorlevel% equ 0 (
    echo   [OK] MySQL est en cours d'execution
    set MYSQL_RUNNING=1
) else (
    echo   [WARN] MySQL non detecte - tentative de demarrage via XAMPP...
    if exist "C:\xampp\xampp_start.exe" (
        start "" /B "C:\xampp\xampp_start.exe"
        echo   Attente de MySQL (10 secondes)...
        timeout /t 10 /nobreak >NUL
        tasklist /FI "IMAGENAME eq mysqld.exe" 2>NUL | find /I "mysqld.exe" >NUL
        if %errorlevel% equ 0 (
            echo   [OK] MySQL demarre avec succes
            set MYSQL_RUNNING=1
        ) else (
            echo   [WARN] MySQL toujours inactif - verifiez XAMPP manuellement
        )
    ) else (
        echo   [WARN] XAMPP non trouve - demarrez MySQL manuellement
    )
)

echo.

REM -----------------------------------------------
REM ETAPE 2/4 : Verification base de donnees
REM -----------------------------------------------
echo [2/4] Verification de la base de donnees...
echo.

if %MYSQL_RUNNING% equ 1 (
    C:\xampp\mysql\bin\mysql.exe -u root -e "SELECT 1" sim800c_manager_deepseekv1 2>NUL
    if %errorlevel% equ 0 (
        echo   [OK] Base de donnees sim800c_manager_deepseekv1 accessible
    ) else (
        echo   [WARN] Base de donnees inaccessible - initialisation...
        C:\xampp\mysql\bin\mysql.exe -u root < scripts\init_db.sql 2>NUL
        if %errorlevel% equ 0 (
            echo   [OK] Base de donnees initialisee
        ) else (
            echo   [WARN] Initialisation echouee - verifiez scripts\init_db.sql
        )
    )
) else (
    echo   [SKIP] MySQL inactif - base de donnees ignoree
)

echo.

REM -----------------------------------------------
REM ETAPE 3/4 : Detection des modules SIM800C USB
REM -----------------------------------------------
echo [3/4] Detection des modules SIM800C USB...
echo.
echo   L'application effectuera un scan automatique de COM1 a COM99
echo   Les modules SIM800C (USB-SERIAL CH340) seront detectes automatiquement
echo.

REM Apercu rapide des ports COM disponibles
set COM_COUNT=0
for /L %%i in (1,1,20) do (
    if exist "\\.\COM%%i" (
        echo   [DETECTE] COM%%i
        set /a COM_COUNT+=1
    )
)
if %COM_COUNT% equ 0 (
    echo   [INFO] Aucun port COM detecte dans COM1-COM20
    echo   [INFO] L'application continuera a scanner en arriere-plan
)

echo.

REM -----------------------------------------------
REM ETAPE 4/4 : Demarrage de l'application Go
REM -----------------------------------------------
echo [4/4] Demarrage de l'application...
echo.

REM Construire si l'executable n'existe pas ou si les sources sont plus recentes
if not exist "sim800c-supervisor.exe" (
    echo   Compilation en cours...
    where go >NUL 2>&1
    if %errorlevel% equ 0 (
        go build -o sim800c-supervisor.exe ./cmd/
        if %errorlevel% neq 0 (
            echo   [ERREUR] Compilation echouee - verifiez les erreurs ci-dessus
            pause
            exit /b 1
        )
        echo   [OK] Compilation reussie
    ) else (
        echo   [ERREUR] Go n'est pas installe ou pas dans le PATH
        echo   Telechargez Go depuis https://go.dev/dl/
        pause
        exit /b 1
    )
)

echo.
echo ========================================
echo   Application en cours de demarrage...
echo   Frontend : http://test-sim800c.lan:8082
echo   Backend  : http://localhost:8082
echo   WebSocket: ws://localhost:8082/ws
echo ========================================
echo.
echo   Connexion par defaut : admin / admin123
echo   Appuyez sur Ctrl+C pour arreter
echo.

REM Creer le fichier PID pour stop_app.bat
start /B "" cmd /c "sim800c-supervisor.exe > storage\logs\runtime.log 2>&1 & echo !errorlevel! > .pid"

REM Attendre que l'app demarre
timeout /t 3 /nobreak >NUL

REM Afficher le log en temps reel
echo   --- Logs en temps reel (storage\logs\runtime.log) ---
powershell -Command "Get-Content -Path 'storage\logs\runtime.log' -Wait -Tail 10"

endlocal
