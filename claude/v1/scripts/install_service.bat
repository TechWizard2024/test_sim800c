@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Installation du service SIM800C Supervisor
echo ========================================
echo.

set SERVICE_NAME=SIM800C_Supervisor
set SERVICE_DISPLAY=SIM800C Supervisor Service
set SERVICE_DESC=Service de supervision des modules SIM800C USB

cd /d "%~dp0.."

set APP_PATH=%CD%\sim800c-supervisor.exe

if not exist "%APP_PATH%" (
    echo Erreur: Binaire non trouve. Veuillez compiler le projet d'abord.
    echo Commande: go build -o sim800c-supervisor.exe cmd/main.go
    pause
    exit /b 1
)

echo Arret du service existant...
net stop %SERVICE_NAME% >nul 2>&1

echo Suppression du service existant...
sc delete %SERVICE_NAME% >nul 2>&1
timeout /t 2 >nul

echo Creation du nouveau service...
sc create %SERVICE_NAME% binPath= "%APP_PATH%" start= auto DisplayName= "%SERVICE_DISPLAY%"

if %errorlevel% neq 0 (
    echo Erreur lors de la creation du service
    pause
    exit /b 1
)

sc description %SERVICE_NAME% "%SERVICE_DESC%"

echo Configuration de la recuperation automatique...
sc failure %SERVICE_NAME% reset= 86400 actions= restart/5000/restart/10000/restart/30000

echo Demarrage du service...
net start %SERVICE_NAME%

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo Service installe et demarre avec succes!
    echo ========================================
) else (
    echo.
    echo Erreur lors du demarrage du service
)

pause