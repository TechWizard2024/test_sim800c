# ========================================
# Test automatique de tous les ports COM
# pour modules SIM800C
# ========================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Detection et test des modules SIM800C" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# Recuperer tous les ports COM disponibles
$ports = [System.IO.Ports.SerialPort]::GetPortNames() | Sort-Object

if ($ports.Count -eq 0) {
    Write-Host "Aucun port COM detecte." -ForegroundColor Red

    Write-Host ""
    Write-Host "Appuyez sur ENTREE pour quitter..."
    Read-Host
    exit
}

Write-Host "Ports detectes : $($ports -join ', ')" -ForegroundColor White
Write-Host ""

foreach ($port in $ports) {

    Write-Host "----------------------------------------" -ForegroundColor DarkCyan
    Write-Host "Test du port $port..." -ForegroundColor Yellow

    $serial = $null

    try {

        # Creation du port serie
        $serial = New-Object System.IO.Ports.SerialPort
        $serial.PortName = $port
        $serial.BaudRate = 9600
        $serial.Parity = [System.IO.Ports.Parity]::None
        $serial.DataBits = 8
        $serial.StopBits = [System.IO.Ports.StopBits]::One

        $serial.ReadTimeout = 3000
        $serial.WriteTimeout = 3000

        $serial.DtrEnable = $true
        $serial.RtsEnable = $true

        # Ouvrir le port
        $serial.Open()

        Start-Sleep -Milliseconds 1000

        # Nettoyage buffer
        $serial.DiscardInBuffer()
        $serial.DiscardOutBuffer()

        # Test AT
        Write-Host "  Envoi: AT" -ForegroundColor Gray

        $serial.Write("AT`r")

        Start-Sleep -Milliseconds 1000

        $response = $serial.ReadExisting()

        if ($response -match "OK") {

            Write-Host "  ✅ Module SIM800C detecte sur $port" -ForegroundColor Green

            # ==========================
            # IMEI
            # ==========================
            try {

                $serial.DiscardInBuffer()

                Write-Host "  Envoi: AT+CGSN" -ForegroundColor Gray

                $serial.Write("AT+CGSN`r")

                Start-Sleep -Milliseconds 1000

                $imei = $serial.ReadExisting()

                Write-Host "  IMEI :" -ForegroundColor Cyan
                Write-Host $imei.Trim() -ForegroundColor White
            }
            catch {
                Write-Host "  Impossible de lire IMEI" -ForegroundColor DarkYellow
            }

            # ==========================
            # Numero SIM
            # ==========================
            try {

                $serial.DiscardInBuffer()

                Write-Host "  Envoi: AT+CNUM" -ForegroundColor Gray

                $serial.Write("AT+CNUM`r")

                Start-Sleep -Milliseconds 1500

                $number = $serial.ReadExisting()

                Write-Host "  Numero :" -ForegroundColor Cyan
                Write-Host $number.Trim() -ForegroundColor White
            }
            catch {
                Write-Host "  Impossible de lire le numero" -ForegroundColor DarkYellow
            }

            # ==========================
            # Signal
            # ==========================
            try {

                $serial.DiscardInBuffer()

                Write-Host "  Envoi: AT+CSQ" -ForegroundColor Gray

                $serial.Write("AT+CSQ`r")

                Start-Sleep -Milliseconds 1000

                $signal = $serial.ReadExisting()

                Write-Host "  Signal :" -ForegroundColor Cyan
                Write-Host $signal.Trim() -ForegroundColor White
            }
            catch {
                Write-Host "  Impossible de lire le signal" -ForegroundColor DarkYellow
            }

        }
        else {

            Write-Host "  ❌ Aucune reponse valide sur $port" -ForegroundColor Red

            if ($response.Trim() -ne "") {

                Write-Host "  Reponse recue :" -ForegroundColor DarkGray
                Write-Host $response.Trim() -ForegroundColor White
            }
        }

    }
    catch {

        Write-Host "  ❌ Erreur sur $port" -ForegroundColor Red
        Write-Host "     $($_.Exception.Message)" -ForegroundColor DarkRed

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
Write-Host "Test termine." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# ========================================
# Empêcher fermeture automatique
# ========================================

Write-Host ""
Write-Host "Appuyez sur ENTREE pour fermer la fenetre..."
Read-Host