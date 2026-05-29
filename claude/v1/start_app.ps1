@echo off
setlocal enabledelayedexpansion
title SIM800C Supervisor v1
color 0A

echo ========================================
echo   SIM800C Supervisor v1
echo   Demarrage de l'application
echo ========================================
echo.

cd /d "%~dp0"

REM -----------------------------------------------
REM Lecture du port depuis .env
REM -----------------------------------------------
set SERVER_PORT=8082
if exist ".env" (
    for /F "usebackq tokens=1,* delims==" %%A in (".env") do (
        set "_KEY=%%A"
        set "_VAL=%%B"
        set "_KEY=!_KEY: =!"
        REM Ignorer les lignes de commentaires
        if not "!_KEY:~0,1!"=="#" (
            if /I "!_KEY!"=="SERVER_PORT" (
                set "_VAL=!_VAL: =!"
                if not "!_VAL!"=="" set SERVER_PORT=!_VAL!
            )
        )
    )
)
echo   [INFO] Port serveur : %SERVER_PORT%

REM -----------------------------------------------
REM PRE-CHECK : Verifier si deja en cours d'execution
REM -----------------------------------------------
tasklist /FI "IMAGENAME eq sim800c-supervisor.exe" 2>NUL | find /I "sim800c-supervisor.exe" >NUL
if !errorlevel! equ 0 (
    echo [AVERT] sim800c-supervisor.exe est deja en cours d'execution !
    echo.
    choice /C ONA /M "  [O]uvrir navigateur  [N]Nouvel instance  [A]rreter"
    if !errorlevel! equ 1 (
        echo   Ouverture du navigateur...
        start http://test-sim800c.lan
        exit /b 0
    )
    if !errorlevel! equ 3 (
        echo   Arret de l'instance existante...
        call stop_app.bat
        timeout /t 2 /nobreak >NUL
    )
)

echo.

REM -----------------------------------------------
REM PRE-CHECK : Verifier si port libre
REM -----------------------------------------------
netstat -ano 2>NUL | find ":%SERVER_PORT%" | find "LISTENING" >NUL
if !errorlevel! equ 0 (
    echo [AVERT] Le port %SERVER_PORT% est deja occupe par un autre processus.
    echo   Verifiez qu'aucune autre instance ne tourne.
    echo   L'application pourrait ne pas demarrer correctement.
    echo.
)

REM -----------------------------------------------
REM PRE-CHECK : Creer les dossiers necessaires
REM -----------------------------------------------
if not exist "storage" mkdir storage
if not exist "storage\logs" (
    mkdir storage\logs
    echo [OK] Dossier storage\logs cree
)
if not exist "storage\excel" mkdir storage\excel
if not exist "storage\backup" mkdir storage\backup

REM Copier Codes_USSD_CI.xlsx si absent du bon dossier
if not exist "storage\excel\Codes_USSD_CI.xlsx" (
    if exist "C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx" (
        copy "C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx" "storage\excel\" >NUL
        echo [OK] Codes_USSD_CI.xlsx copie dans storage\excel\
    ) else (
        echo [WARN] Codes_USSD_CI.xlsx introuvable - verifiez le chemin Excel dans config.yaml
    )
)

echo.

REM -----------------------------------------------
REM ETAPE 1/4 : Verification MySQL (XAMPP)
REM -----------------------------------------------
echo [1/4] Verification de MySQL (XAMPP)...
echo.

set MYSQL_RUNNING=0
tasklist /FI "IMAGENAME eq mysqld.exe" 2>NUL | find /I "mysqld.exe" >NUL
if !errorlevel! equ 0 (
    echo   [OK] MySQL est en cours d'execution
    set MYSQL_RUNNING=1
) else (
    echo   [WARN] MySQL non detecte - tentative de demarrage via XAMPP...
    if exist "C:\xampp\xampp_start.exe" (
        start "" /B "C:\xampp\xampp_start.exe"
        echo   Attente de MySQL ^(10 secondes^)...
        timeout /t 10 /nobreak >NUL
        tasklist /FI "IMAGENAME eq mysqld.exe" 2>NUL | find /I "mysqld.exe" >NUL
        if !errorlevel! equ 0 (
            echo   [OK] MySQL demarre avec succes
            set MYSQL_RUNNING=1
        ) else (
            echo   [WARN] MySQL toujours inactif - verifiez XAMPP manuellement
        )
    ) else (
        if exist "C:\xampp\mysql\bin\mysqld.exe" (
            echo   Demarrage direct de MySQL...
            start "" /B "C:\xampp\mysql\bin\mysqld.exe" --standalone
            echo   Attente de MySQL ^(8 secondes^)...
            timeout /t 8 /nobreak >NUL
            tasklist /FI "IMAGENAME eq mysqld.exe" 2>NUL | find /I "mysqld.exe" >NUL
            if !errorlevel! equ 0 (
                echo   [OK] MySQL demarre avec succes
                set MYSQL_RUNNING=1
            ) else (
                echo   [WARN] MySQL toujours inactif - verifiez XAMPP manuellement
            )
        ) else (
            echo   [WARN] XAMPP non trouve - demarrez MySQL manuellement
        )
    )
)

echo.

REM -----------------------------------------------
REM ETAPE 2/4 : Verification base de donnees
REM -----------------------------------------------
echo [2/4] Verification de la base de donnees...
echo.

if !MYSQL_RUNNING! equ 1 (
    C:\xampp\mysql\bin\mysql.exe -u root -e "SELECT 1;" sim800c_manager_deepseekv1 >NUL 2>&1
    if !errorlevel! equ 0 (
        echo   [OK] Base de donnees sim800c_manager_deepseekv1 accessible

        REM Appliquer migrations si elles n'ont pas ete faites
        if exist "scripts\migrate_v1-13.sql" (
            C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-13.sql >NUL 2>&1
        )
        if exist "scripts\migrate_v1-25.sql" (
            C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-25.sql >NUL 2>&1
        )
    ) else (
        echo   [INFO] Base de donnees inaccessible - tentative d'initialisation...
        if exist "scripts\init_db.sql" (
            C:\xampp\mysql\bin\mysql.exe -u root < scripts\init_db.sql >NUL 2>&1
            if !errorlevel! equ 0 (
                echo   [OK] Base de donnees initialisee avec succes
            ) else (
                echo   [WARN] Initialisation echouee - l'application tentera de creer la DB au demarrage
            )
        ) else (
            echo   [WARN] scripts\init_db.sql introuvable
        )
    )
) else (
    echo   [SKIP] MySQL inactif - base de donnees ignoree
    echo   [INFO] Demarrez XAMPP puis relancez start_app.bat
)

echo.

REM -----------------------------------------------
REM ETAPE 3/4 : Detection des modules SIM800C USB
REM -----------------------------------------------
echo [3/4] Detection des modules SIM800C USB...
echo.
echo   L'application effectuera un scan automatique de COM1 a COM99
echo   Les modules SIM800C ^(USB-SERIAL CH340^) seront detectes automatiquement
echo.

set COM_COUNT=0
for /L %%i in (1,1,30) do (
    if exist "\\.\COM%%i" (
        echo   [DETECTE] COM%%i disponible
        set /a COM_COUNT+=1
    )
)
if !COM_COUNT! equ 0 (
    echo   [INFO] Aucun port COM detecte dans COM1-COM30
    echo   [INFO] L'application continuera a scanner en arriere-plan
) else (
    echo.
    echo   [INFO] !COM_COUNT! port^(s^) COM detecte^(s^) - scan SIM800C en cours au demarrage
)

echo.

REM -----------------------------------------------
REM ETAPE 4/4 : Demarrage de l'application Go
REM -----------------------------------------------
echo [4/4] Demarrage de l'application...
echo.

REM Construire si l'executable n'existe pas OU si les sources sont plus recentes
set NEED_BUILD=0
if not exist "sim800c-supervisor.exe" set NEED_BUILD=1
if exist "cmd\main.go" (
    REM Verifier si les sources sont plus recentes que l'exe
    for /f %%i in ('powershell -Command "if ((Get-Item cmd\main.go).LastWriteTime -gt (Get-Item sim800c-supervisor.exe).LastWriteTime) { echo 1 } else { echo 0 }" 2^>NUL') do set SRC_NEWER=%%i
    if "!SRC_NEWER!"=="1" (
        echo   [INFO] Sources modifiees - recompilation necessaire
        set NEED_BUILD=1
    )
)
if !NEED_BUILD! equ 1 (
    echo   Compilation en cours...
    where go >NUL 2>&1
    if !errorlevel! equ 0 (
        go build -o sim800c-supervisor.exe ./cmd/
        if !errorlevel! neq 0 (
            echo   [ERREUR] Compilation echouee - verifiez les erreurs ci-dessus
            echo   Assurez-vous que Go est installe : https://go.dev/dl/
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
) else (
    echo   [OK] Executable sim800c-supervisor.exe trouve
)

echo.
echo ========================================
echo   Application en cours de demarrage...
echo   Frontend : http://test-sim800c.lan
echo   Backend  : http://localhost:%SERVER_PORT%
echo   WebSocket: ws://localhost:%SERVER_PORT%/ws
echo ========================================
echo.
echo   Connexion par defaut : admin / admin123
echo   Appuyez sur Ctrl+C dans cette fenetre pour arreter les logs
echo   (l'application continue en arriere-plan)
echo.

REM Sauvegarder le PID pour stop_app.bat
set PID_FILE=.pid

REM Demarrer l'application via PowerShell Start-Process pour un vrai detachement du processus
echo   Lancement du processus Go en arriere-plan...
powershell -Command "$p = Start-Process -FilePath '.\sim800c-supervisor.exe' -RedirectStandardOutput '.\storage\logs\runtime.log' -RedirectStandardError '.\storage\logs\runtime_err.log' -WindowStyle Hidden -PassThru; $p.Id | Out-File -Encoding ascii '.pid'"

REM Lire le PID depuis le fichier .pid
timeout /t 2 /nobreak >NUL
set APP_PID=
if exist ".pid" (
    set /P APP_PID=<.pid
    set APP_PID=!APP_PID: =!
)

if defined APP_PID (
    echo   [OK] Application demarree ^(PID: !APP_PID!^)
) else (
    echo   [WARN] Impossible de recuperer le PID
)

REM Attendre que le serveur soit pret ^(scan COM peut prendre quelques secondes^)
echo   Attente du serveur ^(15 secondes - scan des ports COM en cours^)...
timeout /t 15 /nobreak >NUL

REM Verifier que l'application tourne
tasklist /FI "IMAGENAME eq sim800c-supervisor.exe" 2>NUL | find /I "sim800c-supervisor.exe" >NUL
if !errorlevel! equ 0 (
    echo   [OK] Serveur en ecoute sur le port %SERVER_PORT%
    echo.
    echo   Ouverture du navigateur...
    start http://test-sim800c.lan
) else (
    echo   [ERREUR] L'application ne semble pas avoir demarre correctement
    echo   Logs stdout : storage\logs\runtime.log
    echo   Logs stderr : storage\logs\runtime_err.log
    echo.
    echo --- stdout ---
    type storage\logs\runtime.log 2>NUL
    echo.
    echo --- stderr ^(panics, erreurs critiques^) ---
    type storage\logs\runtime_err.log 2>NUL
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo   SIM800C Supervisor est actif !
echo   Frontend : http://test-sim800c.lan
echo   Logs     : storage\logs\runtime.log
echo ========================================
echo.

REM Afficher le log en temps reel
echo   --- Logs en temps reel ^(fermez cette fenetre pour arreter l'affichage^) ---
echo   --- L'application continue en arriere-plan meme si vous fermez cette fenetre ---
echo.
powershell -Command "Get-Content -Path 'storage\logs\runtime.log' -Wait -Tail 20"

endlocal
