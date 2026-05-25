@echo off
title Stop SIM800C Supervisor

echo ========================================
echo Arret de SIM800C Supervisor
echo ========================================
echo.

cd /d %%~dp0

set EXE_NAME=sim800c-supervisor.exe
set SERVICE_NAME=SIM800C_Supervisor

echo [1/3] Arret du service Windows si installe...
sc query %%SERVICE_NAME%% >nul 2>&1
if %%errorlevel%% equ 0 (
    echo   Service trouve : %%SERVICE_NAME%%
    net stop %%SERVICE_NAME%% >nul 2>&1
    if %%errorlevel%% equ 0 (
        echo   Service arrete avec succes
    ) else (
        echo   Impossible d'arreter le service ou il est deja arrete
    )
) else (
    echo   Aucun service %%SERVICE_NAME%% trouve
)

echo.
echo [2/3] Arret du processus %%EXE_NAME%%...
tasklist /FI IMAGENAME eq %%EXE_NAME%% | find /I %%EXE_NAME%% >nul
if %%errorlevel%% equ 0 (
    taskkill /F /IM %%EXE_NAME%% >nul 2>&1
    if %%errorlevel%% equ 0 (
        echo   Processus arrete avec succes
    ) else (
        echo   Erreur lors de l'arret du processus
    )
) else (
    echo   Aucun processus %%EXE_NAME%% en cours
)

echo.
echo [3/3] Verification finale...
tasklist /FI IMAGENAME eq %%EXE_NAME%% | find /I %%EXE_NAME%% >nul
if %%errorlevel%% equ 0 (
    echo   Impossible de stopper completement %%EXE_NAME%%
) else (
    echo   Application arretee avec succes
)

echo.
pause
