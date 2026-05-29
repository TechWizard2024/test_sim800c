# ========================================
# GESTION MODULES SIM800C
# ========================================
# ETAPES :
# 1 - Detection module SIM800C
# 2 - Verification / Deblocage PIN SIM
# 3 - Menu USSD interactif
# ========================================

Clear-Host

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "GESTION DES MODULES SIM800C" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# ========================================
# CODES PIN PAR DEFAUT
# ========================================

$pinCodes = @(
    @{ Operator = "Orange CI"; Pin = "0000"  },
    @{ Operator = "MTN CI";    Pin = "12345" },
    @{ Operator = "Moov CI";   Pin = "0101"  }
)

# ========================================
# MENU USSD
# ========================================

$ussdMenu = @{
    "1" = "*99#"
    "2" = "#99#"
    "3" = "#122#"
    "4" = "*100#"
    "5" = "#100#"
}

# ========================================
# FONCTION LECTURE SERIAL
# ========================================

function Read-SerialResponse {

    param(
        $Serial,
        $WaitSeconds = 2
    )

    $response = ""

    for ($i = 0; $i -lt $WaitSeconds; $i++) {

        Start-Sleep -Seconds 1

        try {

            $data = $Serial.ReadExisting()

            if ($data -ne "") {
                $response += $data
            }
        }
        catch {
        }
    }

    return $response
}

# ========================================
# DETECTION PORTS COM
# ========================================

$ports = [System.IO.Ports.SerialPort]::GetPortNames() |
         Sort-Object -Unique

if ($ports.Count -eq 0) {

    Write-Host "Aucun port COM detecte." -ForegroundColor Red

    Read-Host "Appuyez sur ENTREE pour quitter"

    exit
}

Write-Host "Ports detectes :" -ForegroundColor White
Write-Host ($ports -join ", ") -ForegroundColor Gray
Write-Host ""

# ========================================
# TEST PORTS
# ========================================

foreach ($port in $ports) {

    Write-Host "----------------------------------------" -ForegroundColor DarkCyan
    Write-Host "Test du port $port..." -ForegroundColor Yellow

    $serial = $null

    try {

        # ========================================
        # CONFIGURATION PORT
        # ========================================

        $serial = New-Object System.IO.Ports.SerialPort

        $serial.PortName = $port
        $serial.BaudRate = 9600
        $serial.Parity   = [System.IO.Ports.Parity]::None
        $serial.DataBits = 8
        $serial.StopBits = [System.IO.Ports.StopBits]::One

        $serial.ReadTimeout  = 5000
        $serial.WriteTimeout = 5000

        $serial.DtrEnable = $true
        $serial.RtsEnable = $true

        # ========================================
        # OUVERTURE
        # ========================================

        $serial.Open()

        Start-Sleep -Milliseconds 1500

        $serial.DiscardInBuffer()
        $serial.DiscardOutBuffer()

        # ========================================
        # TEST AT
        # ========================================

        Write-Host "  Verification module..." -ForegroundColor Gray

        $serial.Write("AT`r")

        $response = Read-SerialResponse -Serial $serial -WaitSeconds 2

        if ($response -match "OK") {

            Write-Host "  ✅ SIM800C detecte sur $port" -ForegroundColor Green

            # ========================================
            # DESACTIVER ECHO
            # ========================================

            $serial.DiscardInBuffer()

            $serial.Write("ATE0`r")

            Start-Sleep -Milliseconds 500

            $null = $serial.ReadExisting()

            # ========================================
            # VERIFICATION PIN
            # ========================================

            Write-Host ""
            Write-Host "  Verification etat SIM..." -ForegroundColor Cyan

            $serial.DiscardInBuffer()

            $serial.Write("AT+CPIN?`r")

            $pinStatus = Read-SerialResponse -Serial $serial -WaitSeconds 2

            Write-Host $pinStatus.Trim() -ForegroundColor White

            # ========================================
            # SIM PIN
            # ========================================

            if ($pinStatus -match "SIM PIN") {

                Write-Host ""
                Write-Host "  ⚠️ SIM verrouillee par PIN" -ForegroundColor Yellow

                $pinUnlocked = $false

                # ========================================
                # TEST PINS AUTOMATIQUES
                # ========================================

                foreach ($pinEntry in $pinCodes) {

                    $operator = $pinEntry.Operator
                    $pinCode  = $pinEntry.Pin

                    Write-Host ""
                    Write-Host "  Test PIN $operator : $pinCode" -ForegroundColor Gray

                    $serial.DiscardInBuffer()

                    $serial.Write("AT+CPIN=`"$pinCode`"`r")

                    $unlockResponse = Read-SerialResponse -Serial $serial -WaitSeconds 4

                    Write-Host $unlockResponse.Trim() -ForegroundColor White

                    if ($unlockResponse -match "OK") {

                        Write-Host ""
                        Write-Host "  ✅ PIN valide : $pinCode ($operator)" -ForegroundColor Green

                        $pinUnlocked = $true

                        Start-Sleep -Seconds 5

                        break
                    }
                }

                # ========================================
                # SAISIE MANUELLE SI ECHEC
                # ========================================

                if (-not $pinUnlocked) {

                    Write-Host ""
                    Write-Host "  ❌ Aucun PIN automatique valide" -ForegroundColor Red

                    $manualPin = Read-Host "  Entrez le code PIN SIM"

                    if ($manualPin -ne "") {

                        $serial.DiscardInBuffer()

                        $serial.Write("AT+CPIN=`"$manualPin`"`r")

                        $manualResponse = Read-SerialResponse -Serial $serial -WaitSeconds 4

                        Write-Host $manualResponse.Trim() -ForegroundColor White

                        if ($manualResponse -match "OK") {

                            Write-Host ""
                            Write-Host "  ✅ SIM deverrouillee" -ForegroundColor Green

                            Start-Sleep -Seconds 5
                        }
                        else {

                            Write-Host ""
                            Write-Host "  ❌ PIN incorrect" -ForegroundColor Red

                            continue
                        }
                    }
                    else {

                        continue
                    }
                }
            }

            # ========================================
            # IMEI
            # ========================================

            Write-Host ""
            Write-Host "  Recuperation IMEI..." -ForegroundColor Cyan

            $serial.DiscardInBuffer()

            $serial.Write("AT+CGSN`r")

            $imei = Read-SerialResponse -Serial $serial -WaitSeconds 2

            Write-Host $imei.Trim() -ForegroundColor White

            # ========================================
            # RESEAU
            # ========================================

            Write-Host ""
            Write-Host "  Verification reseau..." -ForegroundColor Cyan

            $serial.DiscardInBuffer()

            $serial.Write("AT+CREG?`r")

            $network = Read-SerialResponse -Serial $serial -WaitSeconds 2

            Write-Host $network.Trim() -ForegroundColor White

            # ========================================
            # SIGNAL
            # ========================================

            Write-Host ""
            Write-Host "  Verification signal..." -ForegroundColor Cyan

            $serial.DiscardInBuffer()

            $serial.Write("AT+CSQ`r")

            $signal = Read-SerialResponse -Serial $serial -WaitSeconds 2

            Write-Host $signal.Trim() -ForegroundColor White

            # ========================================
            # MENU USSD
            # ========================================

            Write-Host ""
            Write-Host "========================================" -ForegroundColor Magenta
            Write-Host "MENU USSD - $port" -ForegroundColor Magenta
            Write-Host "========================================" -ForegroundColor Magenta

            Write-Host "1 - *99#"
            Write-Host "2 - #99#"
            Write-Host "3 - #122#"
            Write-Host "4 - *100#"
            Write-Host "5 - #100#"

            Write-Host ""

            $choice = Read-Host "Choisissez une option"

            if ($ussdMenu.ContainsKey($choice)) {

                $ussdCode = $ussdMenu[$choice]

                Write-Host ""
                Write-Host "  Envoi USSD : $ussdCode" -ForegroundColor Cyan

                $serial.DiscardInBuffer()

                # Activation USSD
                $serial.Write("AT+CUSD=1`r")

                Start-Sleep -Milliseconds 500

                $null = $serial.ReadExisting()

                # Envoi USSD
                $serial.Write("AT+CUSD=1,""`"$ussdCode`"`",15`r")

                $ussdResponse = ""

                # Lecture longue
                for ($i = 0; $i -lt 20; $i++) {

                    Start-Sleep -Seconds 1

                    $data = $serial.ReadExisting()

                    if ($data -ne "") {

                        $ussdResponse += $data

                        Write-Host "  [RX] $data" -ForegroundColor DarkGray

                        if ($ussdResponse -match "\+CUSD") {
                            break
                        }
                    }
                }

                Write-Host ""
                Write-Host "========================================" -ForegroundColor Green
                Write-Host "REPONSE USSD" -ForegroundColor Green
                Write-Host "========================================" -ForegroundColor Green

                Write-Host $ussdResponse.Trim() -ForegroundColor White
            }
            else {

                Write-Host ""
                Write-Host "❌ Option invalide" -ForegroundColor Red
            }

        }
        else {

            Write-Host "  ❌ Aucun module SIM800C detecte" -ForegroundColor Red
        }

    }
    catch {

        Write-Host "  ❌ Erreur : $($_.Exception.Message)" -ForegroundColor Red
    }
    finally {

        if ($serial -ne $null) {

            if ($serial.IsOpen) {
                $serial.Close()
            }

            $serial.Dispose()
        }
    }

    Write-Host ""
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "FIN DES TESTS" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

Write-Host ""
Read-Host "Appuyez sur ENTREE pour quitter"