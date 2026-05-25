# Test de communication avec les modules SIM800C
$ports = @("COM5", "COM6", "COM7")

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test des modules SIM800C" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

foreach ($port in $ports) {
    Write-Host "Test du port $port..." -ForegroundColor Yellow
    
    try {
        $serial = New-Object System.IO.Ports.SerialPort $port, 9600, None, 8, One
        $serial.ReadTimeout = 3000
        $serial.WriteTimeout = 3000
        
        $serial.Open()
        Start-Sleep -Milliseconds 500
        
        # Envoyer AT
        $serial.WriteLine("AT")
        Start-Sleep -Milliseconds 500
        
        $response = $serial.ReadExisting()
        
        if ($response -match "OK") {
            Write-Host "  ✅ Module repond sur $port" -ForegroundColor Green
            
            # Obtenir IMEI
            $serial.WriteLine("AT+CGSN")
            Start-Sleep -Milliseconds 500
            $imei = $serial.ReadExisting()
            Write-Host "     IMEI: $imei" -ForegroundColor White
            
            # Obtenir le numéro
            $serial.WriteLine("AT+CNUM")
            Start-Sleep -Milliseconds 500
            $number = $serial.ReadExisting()
            Write-Host "     Numero: $number" -ForegroundColor White
        }
        else {
            Write-Host "  ❌ Pas de reponse sur $port" -ForegroundColor Red
        }
        
        $serial.Close()
    }
    catch {
        Write-Host "  ❌ Erreur: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    Write-Host ""
}

Write-Host "========================================" -ForegroundColor Cyan