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
	cond  *sync.Cond
	lines []string
}

func newSyncReadBuffer() *syncReadBuffer {
	rb := &syncReadBuffer{}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

func (rb *syncReadBuffer) push(line string) {
	rb.mu.Lock()
	rb.lines = append(rb.lines, line)
	rb.mu.Unlock()
	rb.cond.Broadcast()
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

		rb.cond.Wait()
		rb.mu.Unlock()
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
			s.Logger.Debugf("RX: %s", line)
			s.rb.push(line)
		}
	}()
}

func (s *SIM800C) sendCommandWithResponse(cmd string, expected string, timeout time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startSingleReader()

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
	resp, err := s.sendCommandWithResponse("AT+CGSN", "OK", 20*time.Second)
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
	resp, err := s.sendCommandWithResponse("AT+CNUM", "OK", 20*time.Second)
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

// detectCarrierFromNumber determines carrier from CI phone prefix
func detectCarrierFromNumber(phoneNumber string) string {
	// Strip country code prefix (+225, 00225, 225)
	num := phoneNumber
	for _, prefix := range []string{"+225", "00225", "225"} {
		if strings.HasPrefix(num, prefix) {
			num = num[len(prefix):]
			break
		}
	}
	num = strings.TrimSpace(num)

	if strings.HasPrefix(num, "07") {
		return "Orange"
	}
	if strings.HasPrefix(num, "05") {
		return "MTN"
	}
	if strings.HasPrefix(num, "01") {
		return "Moov"
	}
	return "Universel"
}

// defaultPINForCarrier returns the default PIN for a carrier
func defaultPINForCarrier(carrier string) string {
	switch carrier {
	case "Orange":
		return "0000"
	case "MTN":
		return "12345"
	case "Moov":
		return "0101"
	}
	return "0000"
}

// checkAndUnlockPIN checks if SIM is PIN-locked and attempts to unlock
func (s *SIM800C) checkAndUnlockPIN() error {
	resp, err := s.sendCommandWithResponse("AT+CPIN?", "OK", 10*time.Second)
	if err != nil {
		return err
	}

	if !strings.Contains(resp, "SIM PIN") {
		// Not PIN-locked, or already ready
		return nil
	}

	s.Logger.Warnf("Module %s: SIM PIN requis - tentative de déverrouillage automatique", s.Port)

	// Try to detect carrier from partial info or try all default PINs
	pinsToTry := []string{"0000", "12345", "0101"}
	if s.Carrier != "" && s.Carrier != "Universel" {
		pin := defaultPINForCarrier(s.Carrier)
		// Put carrier-specific pin first
		pinsToTry = append([]string{pin}, pinsToTry...)
	}

	for _, pin := range pinsToTry {
		unlockResp, err := s.sendCommandWithResponse(fmt.Sprintf("AT+CPIN=\"%s\"", pin), "OK", 10*time.Second)
		if err == nil && (strings.Contains(unlockResp, "OK") || strings.Contains(unlockResp, "READY")) {
			s.Logger.Infof("Module %s: PIN déverrouillé avec succès (PIN: %s)", s.Port, pin)
			// Wait for SIM to become ready
			time.Sleep(3 * time.Second)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("impossible de déverrouiller le PIN - vérifiez les codes PIN par défaut")
}

func (s *SIM800C) getPhoneNumberViaUSSD() (string, error) {
	// Universal USSD to get phone number
	universalCodes := []string{"#99#", "*99#", "#06#"}
	for _, code := range universalCodes {
		resp, err := s.sendCommandWithResponse(fmt.Sprintf("AT+CUSD=1,\"%s\",15", code), "+CUSD:", 30*time.Second)
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
			// Extract phone number from response (may contain +225...)
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

func (s *SIM800C) SendAT() error {
	_, err := s.sendCommandWithResponse("AT", "OK", 10*time.Second)
	return err
}

func (s *SIM800C) initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.sendCommandWithResponse("AT", "OK", 10*time.Second); err != nil {
		return fmt.Errorf("AT test échoué: %w", err)
	}

	// Check and unlock PIN before proceeding
	if err := s.checkAndUnlockPIN(); err != nil {
		s.Logger.Warnf("Module %s: %v", s.Port, err)
		// Continue anyway - some operations may still work
	}

	if _, err := s.sendCommandWithResponse("AT+CMGF=1", "OK", 10*time.Second); err != nil {
		return fmt.Errorf("mode SMS texte échoué: %w", err)
	}
	_, _ = s.sendCommandWithResponse("AT+CNMI=2,1,0,0,0", "OK", 5*time.Second)

	if imei, err := s.getIMEI(); err == nil && imei != "" {
		s.IMEI = imei
		s.Logger.Infof("IMEI: %s", imei)
	}

	if phoneNumber, err := s.getPhoneNumber(); err == nil && phoneNumber != "" && phoneNumber != "ERROR" {
		s.PhoneNumber = phoneNumber
		s.Carrier = detectCarrierFromNumber(phoneNumber)
		s.Logger.Infof("Numéro (AT+CNUM): %s, Opérateur: %s", phoneNumber, s.Carrier)
	} else {
		if number, err := s.getPhoneNumberViaUSSD(); err == nil && number != "" {
			s.PhoneNumber = number
			s.Carrier = detectCarrierFromNumber(number)
			s.Logger.Infof("Numéro (USSD): %s, Opérateur: %s", number, s.Carrier)
		}
	}

	return nil
}

// FormatUSSDResponse cleans up SIM800C raw USSD menu text
// The modem returns text with unusual whitespace/alignment used for display on old phones.
// We normalize it to clean, readable lines.
func FormatUSSDResponse(raw string) string {
	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, line := range lines {
		// Collapse multiple spaces/tabs to single space, trim
		spaceRe := regexp.MustCompile(`\s{2,}`)
		line = spaceRe.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

func (s *SIM800C) ExecuteUSSD(code string, inputData string) (string, error) {
	_ = inputData
	cmd := fmt.Sprintf("AT+CUSD=1,\"%s\",15", code)
	resp, err := s.sendCommandWithResponse(cmd, "+CUSD:", 30*time.Second)
	if err != nil {
		// Check if it's a PIN issue
		if strings.Contains(err.Error(), "ERROR") || strings.Contains(resp, "ERROR") {
			pinResp, _ := s.sendCommandWithResponse("AT+CPIN?", "OK", 5*time.Second)
			if strings.Contains(pinResp, "SIM PIN") {
				// Try PIN unlock
				if unlockErr := s.checkAndUnlockPIN(); unlockErr == nil {
					// Retry
					resp, err = s.sendCommandWithResponse(cmd, "+CUSD:", 30*time.Second)
				}
			}
		}
		if err != nil {
			return "", err
		}
	}

	if strings.Contains(resp, "+CUSD:") {
		start := strings.Index(resp, "\"")
		if start != -1 {
			endRel := strings.LastIndex(resp[start+1:], "\"")
			if endRel != -1 {
				end := start + 1 + endRel
				rawText := resp[start+1 : end]
				return FormatUSSDResponse(rawText), nil
			}
		}
	}
	return resp, fmt.Errorf("pas de réponse CUSD")
}

func (s *SIM800C) SendSMS(number, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startSingleReader()
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
		s.rb.cond.Wait()
		s.rb.mu.Unlock()
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
	resp, err := s.sendCommandWithResponse(cmd, "OK", 20*time.Second)
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
	_, err := s.sendCommandWithResponse(fmt.Sprintf("AT+CMGD=%d", index), "OK", 15*time.Second)
	return err
}

func (s *SIM800C) ListSMS() ([]map[string]interface{}, error) {
	resp, err := s.sendCommandWithResponse("AT+CMGL=\"ALL\"", "+CMGL:", 30*time.Second)
	if err != nil {
		return nil, err
	}

	var smsList []map[string]interface{}
	for _, line := range strings.Split(resp, "\n") {
		if !strings.Contains(line, "+CMGL:") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			indexStr := strings.TrimSpace(parts[0])
			indexStr = strings.TrimPrefix(indexStr, "+CMGL: ")
			smsList = append(smsList, map[string]interface{}{
				"index":  indexStr,
				"status": strings.TrimSpace(parts[1]),
				"sender": strings.Trim(parts[2], "\""),
			})
		}
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
	case <-time.After(30 * time.Second):
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

// ensure websocket import not removed by gofmt
var _ = websocket.Event{}
var _ = tserial.Port{}
