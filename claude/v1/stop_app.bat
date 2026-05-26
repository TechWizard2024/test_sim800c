@echo off
title SIM800C Supervisor - Arret

echo ========================================
echo   SIM800C Supervisor - Arret
echo ========================================
echo.

echo [1/2] Arret de l'application Go...
taskkill /F /IM sim800c-supervisor.exe /T >NUL 2>&1
if %errorlevel% equ 0 (
    echo   [OK] Application arretee
) else (
    echo   [INFO] Application non trouvee (deja arretee ?)
)

echo.
echo [2/2] Nettoyage...
if exist ".pid" del /F /Q .pid >NUL 2>&1

echo   [OK] Nettoyage termine
echo.
echo ========================================
echo   Application arretee avec succes
echo ========================================
pause
