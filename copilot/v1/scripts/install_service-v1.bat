@echo off
echo Installation du service SIM800C Supervisor...
echo.

set SERVICE_NAME=SIM800C_Supervisor
set APP_PATH=%CD%\sim800c-supervisor.exe

sc query %SERVICE_NAME% >nul 2>&1
if %errorlevel% equ 0 (
    echo Arret et suppression de l'ancien service...
    net stop %SERVICE_NAME% >nul 2>&1
    sc delete %SERVICE_NAME% >nul 2>&1
    timeout /t 2 >nul
)

echo Creation du nouveau service...
sc create %SERVICE_NAME% binPath= "%APP_PATH%" start= auto DisplayName= "SIM800C Supervisor Service"

sc description %SERVICE_NAME% "Service de supervision des modules SIM800C"

sc failure %SERVICE_NAME% reset= 86400 actions= restart/5000/restart/10000/restart/30000

echo Demarrage du service...
net start %SERVICE_NAME%

echo Service installe et demarre avec succes!
pause