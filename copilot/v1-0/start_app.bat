@echo off
title SIM800C Supervisor

echo ========================================
echo SIM800C Supervisor
echo ========================================
echo.

cd /d  %%~dp0

set EXE_NAME=sim800c-supervisor.exe
set SERVICE_NAME=SIM800C_Supervisor

echo [1/4] Preparation de l'application...
if not exist %%EXE_NAME%% (
    echo   Binaire non trouve : compilation en cours...
    go build -o %%EXE_NAME%% cmd/main.go
    if %%errorlevel%% neq 0 (
        echo Erreur : la compilation a echoue.
        pause
        exit /b 1
    )
    echo   Binaire cree : %%EXE_NAME%%
) else (
    echo   Binaire trouve : %%EXE_NAME%%
)

echo.
echo [2/4] Verification de la base de donnees...
C:\xampp\mysql\bin\mysql.exe -u sim800c_user -pSIM800c@2026! -e SELECT 1 sim800c_manager_deepseekv1 2>nul
if %%errorlevel%% equ 0 (
    echo   [OK] Base de donnees accessible
) else (
    echo   [WARN] Base de donnees non accessible - verifiez que MySQL est demarre
)

echo.
echo [3/4] Verification des modules SIM800C...
echo.
echo Ports COM disponibles:
for %%p in (COM5 COM6 COM7) do (
    if exist \\.\%%p (
        echo   [OK] %%p disponible
    ) else (
        echo   [WARN] %%p non trouve
    )
)

echo.
echo [4/4] Demarrage de l'application...
echo.
echo   Application : %%EXE_NAME%%
echo   Adresse     : http://localhost:8080
echo   Frontend    : http://test-sim800c.lan
echo.
echo Appuyez sur Ctrl+C pour arreter l'application dans cette fenetre.
echo Pour arreter depuis un autre terminal, executez stop_app.bat.
echo.

%%EXE_NAME%%

echo.
echo L'application est terminee.
pause
