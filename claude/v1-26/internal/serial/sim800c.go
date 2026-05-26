package serial

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"sim800c-supervisor/internal/websocket"

	tserial "github.com/tarm/serial"
)

// --- internal buffer used for single-reader synchronization ---

type syncReadBuffer struct {
	mu    sync.Mutex
	lines []string
}

func newSyncReadBuffer() *syncReadBuffer {
	rb := &syncReadBuffer{}
	return rb
}

func (rb *syncReadBuffer) push(line string) {
	rb.mu.Lock()
	rb.lines = append(rb.lines, line)
	rb.mu.Unlock()
}

func (rb *syncReadBuffer) waitReadUntil(startIdx *int, expected string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var out strings.Builder

	for {
		if time.Now().After(deadline) {
			return out.String(), fmt.Errorf("timeout en attente de %s", expected)
		}

		rb.mu.Lock()
		if *startIdx < len(rb.lines) {
			line := rb.lines[*startIdx]
			*startIdx++
			rb.mu.Unlock()

			out.WriteString(line + "\n")

			if expected != "" && strings.Contains(line, expected) {
				return out.String(), nil
			}
			if expected == "" && (strings.Contains(line, "OK") || strings.Contains(line, "ERROR")) {
				return out.String(), nil
			}
			if strings.Contains(line, "ERROR") {
				return out.String(), fmt.Errorf("erreur commande: %s", line)
			}

			continue
		}

		// No new line yet — release lock and sleep briefly before retrying
		rb.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}

// SIM800C methods

func (s *SIM800C) startSingleReader() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readerStarted {
		return
	}
	s.readerStarted = true
	s.rb = newSyncReadBuffer()

	go func() {
		reader := bufio.NewReader(s.SerialPort)
		for {
			select {
			case <-s.stopChan:
				return
			default:
			}

			lineBytes, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			line := strings.TrimSpace(string(lineBytes))
			if line == "" {
				continue
			}
			s.Logger.Debugf("RX[%s]: %s", s.Port, line)
			s.rb.push(line)
		}
	}()
}

// sendCommandRaw sends a command and reads response — caller must NOT hold s.mu
func (s *SIM800C) sendCommandRaw(cmd string, expected string, timeout time.Duration) (string, error) {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	startIdx := 0
	s.rb.mu.Lock()
	startIdx = len(s.rb.lines)
	s.rb.mu.Unlock()

	if _, err := s.SerialPort.Write([]byte(cmd + "\r\n")); err != nil {
		return "", err
	}

	idx := startIdx
	return s.rb.waitReadUntil(&idx, expected, timeout)
}

func (s *SIM800C) getIMEI() (string, error) {
	resp, err := s.sendCommandRaw("AT+CGSN", "OK", 20*time.Second)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 15 && isDigits(line) {
			return line, nil
		}
		if isDigits(line) && len(line) >= 14 {
			return line, nil
		}
	}
	return "", fmt.Errorf("IMEI non trouvé")
}

func (s *SIM800C) getPhoneNumber() (string, error) {
	resp, err := s.sendCommandRaw("AT+CNUM", "OK", 20*time.Second)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(resp, "\n") {
		if !strings.Contains(line, "+CNUM") {
			continue
		}
		firstQ := strings.Index(line, "\"")
		if firstQ == -1 {
			continue
		}
		secondQRel := strings.Index(line[firstQ+1:], "\"")
		if secondQRel == -1 {
			continue
		}
		secondQ := firstQ + 1 + secondQRel

		thirdQRel := strings.Index(line[secondQ+1:], "\"")
		if thirdQRel == -1 {
			continue
		}
		thirdQ := secondQ + 1 + thirdQRel

		numStart := secondQ + 1
		num := strings.TrimSpace(line[numStart:thirdQ])
		if num != "" {
			return num, nil
		}
	}
	return "", fmt.Errorf("numéro non trouvé")
}

// detectCarrierFromNumber determines carrier from phone prefix.
// If dialPlan is non-nil (loaded from DB), it is used for dynamic lookup.
// Falls back to hardcoded CI plan if dialPlan is nil.
func detectCarrierFromNumber(phoneNumber string, dialPlan []DialPlanEntry) string {
	// Strip calling code prefixes (+225, 00225, 225)
	num := phoneNumber
	for _, pfx := range []string{"+225", "00225", "225"} {
		if strings.HasPrefix(num, pfx) {
			num = num[len(pfx):]
			break
		}
	}
	num = strings.TrimSpace(num)

	// Dynamic lookup from DB dial plan
	if len(dialPlan) > 0 {
		for _, entry := range dialPlan {
			if strings.HasPrefix(num, entry.Prefix) {
				return entry.Operator
			}
		}
		return "Universel"
	}

	// Hardcoded fallback (CI)
	switch {
	case strings.HasPrefix(num, "07"):
		return "Orange CI"
	case strings.HasPrefix(num, "05"):
		return "MTN CI"
	case strings.HasPrefix(num, "01"):
		return "Moov Africa CI"
	}
	return "Universel"
}

// defaultPINForCarrier returns the default PIN for a carrier
func defaultPINForCarrier(carrier string) string {
	switch carrier {
	case "Orange CI":
		return "0000"
	case "MTN CI":
		return "12345"
	case "Moov Africa CI":
		return "0101"
	}
	return "0000"
}

// checkAndUnlockPIN checks if SIM is PIN-locked and attempts to unlock
// Must NOT hold s.mu or s.cmdMu when called
func (s *SIM800C) checkAndUnlockPIN() error {
	resp, err := s.sendCommandRaw("AT+CPIN?", "OK", 10*time.Second)
	if err != nil {
		return err
	}

	if !strings.Contains(resp, "SIM PIN") {
		s.PINUnlocked = true
		return nil
	}

	s.Logger.Warnf("Module %s: SIM PIN requis - tentative de déverrouillage automatique", s.Port)

	pinsToTry := []string{"0000", "12345", "0101"}
	if s.Carrier != "" && s.Carrier != "Universel" {
		pin := defaultPINForCarrier(s.Carrier)
		pinsToTry = append([]string{pin}, pinsToTry...)
	}

	for _, pin := range pinsToTry {
		unlockResp, err := s.sendCommandRaw(fmt.Sprintf("AT+CPIN=\"%s\"", pin), "OK", 10*time.Second)
		if err == nil && (strings.Contains(unlockResp, "OK") || strings.Contains(unlockResp, "READY")) {
			s.Logger.Infof("Module %s: PIN déverrouillé avec succès (PIN: %s)", s.Port, pin)
			s.PINUnlocked = true
			time.Sleep(3 * time.Second)
			// Notify frontend via WebSocket
			if s.hub != nil {
				s.hub.BroadcastEvent(websocket.Event{
					Type:     "pin_unlocked",
					ModuleID: s.ModuleID,
					Data: map[string]interface{}{
						"port":        s.Port,
						"pin_entered": pin,
					},
					Timestamp: time.Now(),
				})
			}
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("impossible de déverrouiller le PIN - vérifiez les codes PIN par défaut")
}

// markPINFailed sets PINFailed and notifies the frontend
func (s *SIM800C) markPINFailed() {
	s.PINFailed = true
	if s.hub != nil {
		s.hub.BroadcastEvent(websocket.Event{
			Type:     "pin_failed",
			ModuleID: s.ModuleID,
			Data: map[string]interface{}{
				"port":    s.Port,
				"message": "Impossible de déverrouiller le PIN - aucun code par défaut n'a fonctionné",
			},
			Timestamp: time.Now(),
		})
	}
}

func (s *SIM800C) getPhoneNumberViaUSSD() (string, error) {
	universalCodes := []string{"#99#", "*99#", "#06#"}
	for _, code := range universalCodes {
		resp, err := s.sendCommandRaw(fmt.Sprintf("AT+CUSD=1,\"%s\",15", code), "+CUSD:", 30*time.Second)
		if err != nil {
			continue
		}

		for _, line := range strings.Split(resp, "\n") {
			if !strings.Contains(line, "+CUSD:") {
				continue
			}
			start := strings.Index(line, "\"")
			if start == -1 {
				continue
			}
			endRel := strings.Index(line[start+1:], "\"")
			if endRel == -1 {
				continue
			}
			end := start + 1 + endRel
			num := strings.TrimSpace(line[start+1 : end])
			phoneRe := regexp.MustCompile(`(?:\+225|00225|225)?0[157]\d{8}`)
			if match := phoneRe.FindString(num); match != "" {
				return match, nil
			}
			if strings.Contains(num, "225") {
				return num, nil
			}
		}
	}
	return "", fmt.Errorf("numéro non trouvé via USSD")
}

// getSignalQuality fetches AT+CSQ and returns the CSQ value (0-31, 99=unknown)
func (s *SIM800C) getSignalQuality() (int, error) {
	resp, err := s.sendCommandRaw("AT+CSQ", "OK", 5*time.Second)
	if err != nil {
		return 99, err
	}
	for _, line := range strings.Split(resp, "\n") {
		if !strings.Contains(line, "+CSQ:") {
			continue
		}
		// +CSQ: 29,0
		after := strings.TrimPrefix(strings.TrimSpace(line), "+CSQ:")
		after = strings.TrimSpace(after)
		parts := strings.Split(after, ",")
		if len(parts) >= 1 {
			var csq int
			fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &csq)
			return csq, nil
		}
	}
	return 99, fmt.Errorf("CSQ non trouvé")
}

// getNetworkStatus fetches AT+CREG? and returns a human-readable status
func (s *SIM800C) getNetworkStatus() string {
	resp, err := s.sendCommandRaw("AT+CREG?", "OK", 5*time.Second)
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(resp, "\n") {
		if !strings.Contains(line, "+CREG:") {
			continue
		}
		// +CREG: 0,1  or  +CREG: 1
		after := strings.TrimPrefix(strings.TrimSpace(line), "+CREG:")
		after = strings.TrimSpace(after)
		parts := strings.Split(after, ",")
		statStr := strings.TrimSpace(parts[len(parts)-1])
		var stat int
		fmt.Sscanf(statStr, "%d", &stat)
		switch stat {
		case 0:
			return "not_registered"
		case 1:
			return "registered"
		case 2:
			return "searching"
		case 3:
			return "denied"
		case 5:
			return "roaming"
		default:
			return "unknown"
		}
	}
	return "unknown"
}

// CSQToRSSI converts CSQ value to approximate dBm
func CSQToRSSI(csq int) string {
	if csq == 99 || csq == 0 {
		return "N/A"
	}
	dbm := -113 + csq*2
	return fmt.Sprintf("%d dBm", dbm)
}

// GetSignalQuality is the public wrapper for getSignalQuality
func (s *SIM800C) GetSignalQuality() (int, error) {
	return s.getSignalQuality()
}

// GetNetworkStatus is the public wrapper for getNetworkStatus
func (s *SIM800C) GetNetworkStatus() string {
	return s.getNetworkStatus()
}

func (s *SIM800C) SendAT() error {
	_, err := s.sendCommandRaw("AT", "OK", 10*time.Second)
	return err
}

func (s *SIM800C) initialize() {
	// Start the single reader goroutine first (no lock needed — it self-guards)
	s.startSingleReader()

	if _, err := s.sendCommandRaw("AT", "OK", 10*time.Second); err != nil {
		s.Logger.Errorf("Module %s: AT test échoué: %v", s.Port, err)
		return
	}

	// Check and unlock PIN before proceeding
	if err := s.checkAndUnlockPIN(); err != nil {
		s.Logger.Warnf("Module %s: %v", s.Port, err)
		s.markPINFailed()
		// Continue anyway
	}

	if _, err := s.sendCommandRaw("AT+CMGF=1", "OK", 10*time.Second); err != nil {
		s.Logger.Warnf("Module %s: mode SMS texte échoué: %v", s.Port, err)
	}
	s.sendCommandRaw("AT+CNMI=2,1,0,0,0", "OK", 5*time.Second)

	if imei, err := s.getIMEI(); err == nil && imei != "" {
		s.IMEI = imei
		s.Logger.Infof("Module %s - IMEI: %s", s.Port, imei)
	}

	if phoneNumber, err := s.getPhoneNumber(); err == nil && phoneNumber != "" && phoneNumber != "ERROR" {
		s.PhoneNumber = phoneNumber
		s.Carrier = detectCarrierFromNumber(phoneNumber, s.dialPlan)
		s.Logger.Infof("Module %s - Numéro (AT+CNUM): %s, Opérateur: %s", s.Port, phoneNumber, s.Carrier)
	} else {
		if number, err := s.getPhoneNumberViaUSSD(); err == nil && number != "" {
			s.PhoneNumber = number
			s.Carrier = detectCarrierFromNumber(number, s.dialPlan)
			s.Logger.Infof("Module %s - Numéro (USSD): %s, Opérateur: %s", s.Port, number, s.Carrier)
		}
	}

	s.Logger.Infof("Module %s: initialisation terminée (IMEI=%s, SIM=%s, Carrier=%s, PINUnlocked=%v)",
		s.Port, s.IMEI, s.PhoneNumber, s.Carrier, s.PINUnlocked)

	// Fetch signal quality and network status
	if csq, err := s.getSignalQuality(); err == nil {
		s.SignalQuality = csq
		s.Logger.Infof("Module %s - Signal: CSQ=%d (%s)", s.Port, csq, CSQToRSSI(csq))
		// MICRO-BLOC C5 — Enregistrer la mesure de signal en DB
		if s.dbLogger != nil && s.DBID > 0 {
			rssiVal := float64(-113 + csq*2)
			go s.dbLogger.LogSignal(s.DBID, csq, rssiVal, s.NetworkStatus)
		}
	}
	s.NetworkStatus = s.getNetworkStatus()
	s.Logger.Infof("Module %s - Réseau: %s", s.Port, s.NetworkStatus)

	// Broadcast module_initialized event so the frontend can refresh the dashboard
	if s.hub != nil {
		s.hub.BroadcastEvent(websocket.Event{
			Type:     "module_initialized",
			ModuleID: s.ModuleID,
			Data: map[string]interface{}{
				"port":           s.Port,
				"imei":           s.IMEI,
				"phone_number":   s.PhoneNumber,
				"carrier":        s.Carrier,
				"pin_unlocked":   s.PINUnlocked,
				"pin_failed":     s.PINFailed,
				"signal_quality": s.SignalQuality,
				"signal_rssi":    CSQToRSSI(s.SignalQuality),
				"network_status": s.NetworkStatus,
			},
			Timestamp: time.Now(),
		})
	}

	// Call post-init callback if set (used for DB persistence)
	if s.onInitDone != nil {
		s.onInitDone()
	}
}

// FormatUSSDResponse cleans up SIM800C raw USSD menu text.
// The modem returns text with unusual whitespace/alignment used for display on old phones.
func FormatUSSDResponse(raw string) string {
	// First, replace all sequences of whitespace-only that appear between option lines
	// The SIM800C aligns text by padding with spaces — collapse them
	spaceRe := regexp.MustCompile(`[ \t]{2,}`)
	raw = spaceRe.ReplaceAllString(raw, " ")

	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Normalize separator lines like "- - -" or "---"
		if regexp.MustCompile(`^[-\s]+$`).MatchString(line) {
			cleaned = append(cleaned, "---")
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

// ExecuteUSSDRaw sends a USSD command (or menu choice) directly on the serial port.
// This is the low-level call used for menu navigation.
func (s *SIM800C) ExecuteUSSDRaw(code string) (string, error) {
	cmd := fmt.Sprintf("AT+CUSD=1,\"%s\",15", code)
	resp, err := s.sendCommandRaw(cmd, "+CUSD:", 30*time.Second)
	if err != nil {
		// On ERROR, check PIN
		if strings.Contains(err.Error(), "erreur commande") {
			pinResp, _ := s.sendCommandRaw("AT+CPIN?", "OK", 5*time.Second)
			if strings.Contains(pinResp, "SIM PIN") {
				if unlockErr := s.checkAndUnlockPIN(); unlockErr == nil {
					resp, err = s.sendCommandRaw(cmd, "+CUSD:", 30*time.Second)
				}
			}
		}
		if err != nil {
			return "", err
		}
	}

	return parseUSSDResponse(resp), nil
}

// parseUSSDResponse extracts the text from a +CUSD: response and formats it.
func parseUSSDResponse(resp string) string {
	if !strings.Contains(resp, "+CUSD:") {
		return strings.TrimSpace(resp)
	}
	start := strings.Index(resp, "\"")
	if start == -1 {
		return strings.TrimSpace(resp)
	}
	endRel := strings.LastIndex(resp[start+1:], "\"")
	if endRel == -1 {
		return strings.TrimSpace(resp)
	}
	end := start + 1 + endRel
	rawText := resp[start+1 : end]
	return FormatUSSDResponse(rawText)
}

// ExecuteUSSD is called from the command channel handler.
func (s *SIM800C) ExecuteUSSD(code string, inputData string) (string, error) {
	_ = inputData
	return s.ExecuteUSSDRaw(code)
}

func (s *SIM800C) SendSMS(number, message string) error {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	startIdx := 0
	s.rb.mu.Lock()
	startIdx = len(s.rb.lines)
	s.rb.mu.Unlock()

	cmd := fmt.Sprintf("AT+CMGS=\"%s\"", number)
	if _, err := s.SerialPort.Write([]byte(cmd + "\r\n")); err != nil {
		return err
	}

	idx := startIdx
	deadline := time.Now().Add(20 * time.Second)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout prompt SMS")
		}
		s.rb.mu.Lock()
		if idx < len(s.rb.lines) {
			line := s.rb.lines[idx]
			idx++
			s.rb.mu.Unlock()
			if strings.Contains(line, ">") {
				break
			}
			if strings.Contains(line, "ERROR") {
				return fmt.Errorf("erreur envoi SMS prompt: %s", line)
			}
			continue
		}
		s.rb.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}

	if _, err := s.SerialPort.Write([]byte(message + "\x1A")); err != nil {
		return err
	}

	idx2 := idx
	_, err := s.rb.waitReadUntil(&idx2, "+CMGS:", 30*time.Second)
	return err
}

func (s *SIM800C) ReadSMS(index int) (string, string, error) {
	cmd := fmt.Sprintf("AT+CMGR=%d", index)
	resp, err := s.sendCommandRaw(cmd, "OK", 20*time.Second)
	if err != nil {
		return "", "", err
	}

	var sender, message string
	for i, line := range strings.Split(resp, "\n") {
		if strings.Contains(line, "+CMGR:") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				sender = strings.Trim(parts[1], "\"")
			}
		}
		if i > 0 && !strings.Contains(line, "+CMGR:") && !strings.Contains(line, "OK") && !strings.Contains(line, "ERROR") {
			line = strings.TrimSpace(line)
			if line != "" {
				message = line
			}
		}
	}

	if sender == "" && message == "" {
		return "", "", fmt.Errorf("SMS introuvable index %d", index)
	}
	return sender, message, nil
}

func (s *SIM800C) DeleteSMS(index int) error {
	_, err := s.sendCommandRaw(fmt.Sprintf("AT+CMGD=%d", index), "OK", 15*time.Second)
	return err
}

func (s *SIM800C) ListSMS() ([]map[string]interface{}, error) {
	resp, err := s.sendCommandRaw("AT+CMGL=\"ALL\"", "OK", 30*time.Second)
	if err != nil {
		// No messages returns ERROR on some modems — treat as empty list
		if strings.Contains(err.Error(), "erreur commande") {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	var smsList []map[string]interface{}
	lines := strings.Split(resp, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.Contains(line, "+CMGL:") {
			continue
		}
		// Parse: +CMGL: <index>,<status>,<number>,,[<date>]
		parts := strings.SplitN(strings.TrimPrefix(line, "+CMGL: "), ",", 5)
		if len(parts) < 3 {
			continue
		}
		indexStr := strings.TrimSpace(parts[0])
		status := strings.TrimSpace(parts[1])
		sender := strings.Trim(strings.TrimSpace(parts[2]), "\"")

		// Next line is the message body
		var message string
		if i+1 < len(lines) {
			message = strings.TrimSpace(lines[i+1])
			// Skip if it looks like another AT response line
			if strings.HasPrefix(message, "+") || message == "OK" || message == "ERROR" {
				message = ""
			} else {
				i++ // consumed
			}
		}

		smsList = append(smsList, map[string]interface{}{
			"index":   indexStr,
			"status":  status,
			"sender":  sender,
			"message": message,
		})
	}
	if smsList == nil {
		smsList = []map[string]interface{}{}
	}
	return smsList, nil
}

func (s *SIM800C) handleCommands() {
	for {
		select {
		case cmd := <-s.commandChan:
			switch cmd.Type {
			case "ussd":
				res, err := s.ExecuteUSSD(cmd.USSDCode, cmd.InputData)
				if err != nil {
					cmd.Response <- fmt.Sprintf("Erreur: %v", err)
				} else {
					cmd.Response <- res
				}
			case "sms_send":
				err := s.SendSMS(cmd.SMSNumber, cmd.SMSMessage)
				if err != nil {
					cmd.Response <- fmt.Sprintf("Erreur: %v", err)
				} else {
					cmd.Response <- "SMS envoyé avec succès"
				}
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *SIM800C) SendCommand(cmd Command) (string, error) {
	cmd.Response = make(chan string, 1)
	s.commandChan <- cmd

	select {
	case response := <-cmd.Response:
		return response, nil
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("timeout commande")
	}
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ensure imports used
var _ = tserial.Port{}
