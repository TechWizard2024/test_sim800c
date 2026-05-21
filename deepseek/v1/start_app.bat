@echo off
title SIM800C Supervisor

echo ========================================
echo SIM800C Supervisor
echo ========================================
echo.

cd /d "%~dp0"

echo [1/3] Verification des modules SIM800C...
echo.

REM Vérifier les ports COM
echo Ports COM disponibles:
for %%p in (COM5 COM6 COM7) do (
    if exist "\\.\%%p" (
        echo   [OK] %%p disponible
    ) else (
        echo   [WARN] %%p non trouve
    )
)

echo.
echo [2/3] Verification de la base de donnees...
echo.

C:\xampp\mysql\bin\mysql.exe -u sim800c_user -pSIM800c@2026! -e "SELECT 1" sim800c_manager 2>nul
if %errorlevel% equ 0 (
    echo   [OK] Base de donnees accessible
) else (
    echo   [WARN] Base de donnees non accessible - verifiez que MySQL est demarre
)

echo.
echo [3/3] Demarrage de l'application...
echo.

echo L'application va demarrer sur http://localhost:8080
echo Frontend accessible sur http://test_sim800c.local
echo.
echo Appuyez sur Ctrl+C pour arreter l'application
echo.

sim800c-supervisor.exe

pause