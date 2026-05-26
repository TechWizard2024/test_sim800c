@echo off
setlocal enabledelayedexpansion
title SIM800C Supervisor - Arret

echo ========================================
echo   SIM800C Supervisor - Arret
echo ========================================
echo.

echo [1/2] Arret de l'application Go...

REM Essayer d'abord par PID si disponible
set APP_PID=
if exist ".pid" (
    set /P APP_PID=<.pid
    set APP_PID=!APP_PID: =!
)

if defined APP_PID (
    echo   Arret du processus PID: !APP_PID!...
    taskkill /F /PID !APP_PID! >NUL 2>&1
    if !errorlevel! equ 0 (
        echo   [OK] Processus !APP_PID! arrete
    ) else (
        echo   [INFO] PID !APP_PID! introuvable - tentative par nom...
        taskkill /F /IM sim800c-supervisor.exe /T >NUL 2>&1
    )
) else (
    taskkill /F /IM sim800c-supervisor.exe /T >NUL 2>&1
    if !errorlevel! equ 0 (
        echo   [OK] Application arretee
    ) else (
        echo   [INFO] Application non trouvee ^(deja arretee ?^)
    )
)

echo.
echo [2/2] Nettoyage...
if exist ".pid" del /F /Q .pid >NUL 2>&1

echo   [OK] Nettoyage termine
echo.
echo ========================================
echo   Application arretee avec succes
echo ========================================
echo.
pause
endlocal
