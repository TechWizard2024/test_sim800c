package serial

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"sim800c-supervisor/internal/websocket"

	"github.com/tarm/serial"
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

// SIM800C methods live on the SIM800C struct defined in manager.go

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
			// notify asynchronous events (CUSD / CMTI)
			if strings.Contains(line, "+CMTI:") {
				// monitoring is handled by SMSManager polling, so just log
				// (we keep it lightweight to avoid blocking reader)
			}
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
		// Format: +CNUM: "line1","+225...",145
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

func (s *SIM800C) getPhoneNumberViaUSSD() (string, error) {
	resp, err := s.sendCommandWithResponse("AT+CUSD=1,\"#99#\",15", "+CUSD:", 30*time.Second)
	if err != nil {
		return "", err
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
		if strings.Contains(num, "+225") {
			return num, nil
		}
	}
	return "", fmt.Errorf("numéro non trouvé")
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
		s.Logger.Infof("Numéro (AT+CNUM): %s", phoneNumber)
	} else {
		if number, err := s.getPhoneNumberViaUSSD(); err == nil && number != "" {
			s.PhoneNumber = number
			s.Logger.Infof("Numéro (USSD): %s", number)
		}
	}

	// Détecter l'opérateur à partir du préfixe du numéro (+22507 -> Orange, +22505 -> MTN, +22501 -> Moov)
	if s.PhoneNumber != "" {
		if strings.HasPrefix(s.PhoneNumber, "+22507") {
			s.Carrier = "Orange"
		} else if strings.HasPrefix(s.PhoneNumber, "+22505") {
			s.Carrier = "MTN"
		} else if strings.HasPrefix(s.PhoneNumber, "+22501") {
			s.Carrier = "Moov"
		}
		if s.Carrier != "" {
			s.Logger.Infof("Opérateur détecté: %s", s.Carrier)
		}
	}

	// If SIM is PIN-locked, attempt common default PINs (Orange/MTN/Moov)
	_ = s.attemptPINUnlock()

	return nil
}

// attemptPINUnlock checks CPIN status and tries common default PINs if SIM is locked
func (s *SIM800C) attemptPINUnlock() error {
	resp, err := s.sendCommandWithResponse("AT+CPIN?", "OK", 5*time.Second)
	if err != nil {
		return err
	}
	if strings.Contains(resp, "SIM PIN") || strings.Contains(resp, "SIM PUK") {
		pins := []string{"0000", "12345", "0101"}
		for _, p := range pins {
			s.Logger.Infof("Tentative déverrouillage PIN %s sur %s", p, s.Port)
			if _, err := s.sendCommandWithResponse(fmt.Sprintf("AT+CPIN=\"%s\"", p), "OK", 8*time.Second); err == nil {
				// give module time to update state
				time.Sleep(2 * time.Second)
				resp2, _ := s.sendCommandWithResponse("AT+CPIN?", "OK", 5*time.Second)
				if strings.Contains(resp2, "READY") {
					s.Logger.Infof("SIM déverrouillée avec PIN %s", p)
					return nil
				}
			}
		}
		s.Logger.Warn("Aucun PIN par défaut n'a permis de déverrouiller la SIM")
	}
	return nil
}

func formatUSSDText(t string) string {
	if t == "" {
		return t
	}
	// Normalize line endings and remove weird replacement characters
	t = strings.ReplaceAll(t, "\r", "")
	t = strings.ReplaceAll(t, "�", " ")
	// Trim and collapse multiple blank lines and spaces
	lines := strings.Split(t, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// collapse multiple spaces
		l = strings.Join(strings.Fields(l), " ")
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func (s *SIM800C) ExecuteUSSD(code string, inputData string) (string, error) {
	_ = inputData
	cmd := fmt.Sprintf("AT+CUSD=1,\"%s\",15", code)
	resp, err := s.sendCommandWithResponse(cmd, "+CUSD:", 30*time.Second)
	if err != nil {
		return "", err
	}

	if strings.Contains(resp, "+CUSD:") {
		start := strings.Index(resp, "\"")
		if start != -1 {
			endRel := strings.LastIndex(resp[start+1:], "\"")
			if endRel != -1 {
				end := start + 1 + endRel
				raw := resp[start+1 : end]
				return formatUSSDText(raw), nil
			}
		}
	}
	return resp, fmt.Errorf("pas de réponse CUSD")
}

func (s *SIM800C) SendSMS(number, message string) error {
	// NOTE: on garde l’envoi synchrone sous la même exclusion que les autres commandes.
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

	// Wait for +CMGS:
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

// handleCommands and SendCommand keep compatibility with current manager.go.
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

// ensure websocket import not removed by gofmt in case of future use
var _ = websocket.Event{}
var _ = serial.Port{}
