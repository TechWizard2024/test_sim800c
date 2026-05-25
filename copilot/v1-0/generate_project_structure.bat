@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Generation du projet SIM800C Supervisor
echo ========================================
echo.

set PROJECT_ROOT=C:\xampp\htdocs\aa_Toolbox\test_sim800c\deepseek\v1

echo Creation de la structure de dossiers dans %PROJECT_ROOT%
echo.

REM Creation des dossiers principaux
mkdir "%PROJECT_ROOT%" 2>nul
mkdir "%PROJECT_ROOT%\cmd" 2>nul
mkdir "%PROJECT_ROOT%\internal" 2>nul
mkdir "%PROJECT_ROOT%\internal\config" 2>nul
mkdir "%PROJECT_ROOT%\internal\serial" 2>nul
mkdir "%PROJECT_ROOT%\internal\ussd" 2>nul
mkdir "%PROJECT_ROOT%\internal\sms" 2>nul
mkdir "%PROJECT_ROOT%\internal\excel" 2>nul
mkdir "%PROJECT_ROOT%\internal\db" 2>nul
mkdir "%PROJECT_ROOT%\internal\websocket" 2>nul
mkdir "%PROJECT_ROOT%\internal\api" 2>nul
mkdir "%PROJECT_ROOT%\internal\api\handlers" 2>nul
mkdir "%PROJECT_ROOT%\internal\api\middleware" 2>nul
mkdir "%PROJECT_ROOT%\pkg" 2>nul
mkdir "%PROJECT_ROOT%\pkg\logger" 2>nul
mkdir "%PROJECT_ROOT%\pkg\errors" 2>nul
mkdir "%PROJECT_ROOT%\web" 2>nul
mkdir "%PROJECT_ROOT%\web\css" 2>nul
mkdir "%PROJECT_ROOT%\web\js" 2>nul
mkdir "%PROJECT_ROOT%\web\assets" 2>nul
mkdir "%PROJECT_ROOT%\web\assets\icons" 2>nul
mkdir "%PROJECT_ROOT%\web\assets\fonts" 2>nul
mkdir "%PROJECT_ROOT%\scripts" 2>nul
mkdir "%PROJECT_ROOT%\docs" 2>nul
mkdir "%PROJECT_ROOT%\storage" 2>nul
mkdir "%PROJECT_ROOT%\storage\excel" 2>nul
mkdir "%PROJECT_ROOT%\storage\logs" 2>nul
mkdir "%PROJECT_ROOT%\storage\backup" 2>nul
mkdir "%PROJECT_ROOT%\tests" 2>nul

echo [OK] Structure de dossiers creee avec succes
echo.

REM Copier le fichier Excel s'il existe
if exist "C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx" (
    copy "C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx" "%PROJECT_ROOT%\storage\excel\" 2>nul
    echo [OK] Fichier Excel copie
) else (
    echo [WARN] Fichier Excel non trouve, veuillez le copier manuellement
)

echo.
echo Structure generee avec succes dans %PROJECT_ROOT%
pause