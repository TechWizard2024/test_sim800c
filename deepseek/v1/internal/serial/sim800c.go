package serial

import (
	"bufio"
	"fmt"
	"strings"
	"time"
)

// SendAT - Envoie une commande AT de test
func (s *SIM800C) SendAT() error {
	return s.sendCommand("AT", "OK")
}

// initialize - Initialise le module
func (s *SIM800C) initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Test AT
	if err := s.sendCommand("AT", "OK"); err != nil {
		return fmt.Errorf("AT test échoué: %w", err)
	}

	// Mode SMS texte
	if err := s.sendCommand("AT+CMGF=1", "OK"); err != nil {
		return fmt.Errorf("mode SMS texte échoué: %w", err)
	}

	// Notification SMS
	s.sendCommand("AT+CNMI=2,1,0,0,0", "OK")

	// Lire IMEI
	imei, err := s.getIMEI()
	if err == nil && imei != "" {
		s.IMEI = imei
		s.Logger.Infof("IMEI: %s", imei)
	}

	// Lire numéro de téléphone via commande AT+CNUM
	phoneNumber, err := s.getPhoneNumber()
	if err == nil && phoneNumber != "" && phoneNumber != "ERROR" {
		s.PhoneNumber = phoneNumber
		s.Logger.Infof("Numéro (AT+CNUM): %s", phoneNumber)
	} else {
		// Essayer d'obtenir le numéro via USSD #99#
		s.Logger.Info("Tentative d'obtention du numéro via USSD #99#")
		number, err := s.getPhoneNumberViaUSSD()
		if err == nil && number != "" {
			s.PhoneNumber = number
			s.Logger.Infof("Numéro (USSD): %s", number)
		}
	}

	return nil
}

// getPhoneNumberViaUSSD - Obtient le numéro via USSD #99#
func (s *SIM800C) getPhoneNumberViaUSSD() (string, error) {
	response, err := s.sendCommandWithResponse("AT+CUSD=1,\"#99#\",15")
	if err != nil {
		return "", err
	}

	// Parser la réponse CUSD
	// Format attendu: +CUSD: 0,"+225XXXXXXXXXX",15
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.Contains(line, "+CUSD:") {
			// Extraire le numéro entre guillemets
			start := strings.Index(line, "\"")
			if start != -1 {
				end := strings.Index(line[start+1:], "\"")
				if end != -1 {
					number := line[start+1 : start+1+end]
					// Nettoyer le numéro
					number = strings.TrimSpace(number)
					if strings.Contains(number, "+225") {
						return number, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("numéro non trouvé")
}

// sendCommand - Envoie une commande et attend une réponse
func (s *SIM800C) sendCommand(cmd, expected string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Vider le buffer
	s.SerialPort.Read(make([]byte, 1024))

	_, err := s.SerialPort.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return err
	}

	timeout := time.After(10 * time.Second)
	scanner := bufio.NewScanner(s.SerialPort)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout en attente de %s", expected)
		default:
			if scanner.Scan() {
				line := scanner.Text()
				s.Logger.Debugf("Réponse: %s", line)
				if strings.Contains(line, expected) {
					return nil
				}
				if strings.Contains(line, "ERROR") {
					return fmt.Errorf("erreur commande: %s", line)
				}
			}
		}
	}
}

// sendCommandWithResponse - Envoie une commande et retourne la réponse complète
func (s *SIM800C) sendCommandWithResponse(cmd string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Vider le buffer
	s.SerialPort.Read(make([]byte, 1024))

	_, err := s.SerialPort.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return "", err
	}

	var response strings.Builder
	timeout := time.After(30 * time.Second)
	scanner := bufio.NewScanner(s.SerialPort)

	for {
		select {
		case <-timeout:
			return response.String(), fmt.Errorf("timeout")
		default:
			if scanner.Scan() {
				line := scanner.Text()
				response.WriteString(line + "\n")

				if strings.Contains(line, "OK") || strings.Contains(line, "ERROR") {
					return response.String(), nil
				}
			}
		}
	}
}

// getIMEI - Récupère l'IMEI du module
func (s *SIM800C) getIMEI() (string, error) {
	response, err := s.sendCommandWithResponse("AT+CGSN")
	if err != nil {
		return "", err
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// L'IMEI fait 15 chiffres
		if len(line) == 15 && isDigits(line) {
			return line, nil
		}
		// Parfois sur deux lignes
		if strings.Contains(line, "AT+CGSN") {
			continue
		}
		if isDigits(line) && len(line) >= 14 {
			return line, nil
		}
	}

	return "", fmt.Errorf("IMEI non trouvé")
}

// getPhoneNumber - Récupère le numéro via AT+CNUM
func (s *SIM800C) getPhoneNumber() (string, error) {
	response, err := s.sendCommandWithResponse("AT+CNUM")
	if err != nil {
		return "", err
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.Contains(line, "+CNUM") {
			// Format: +CNUM: "line1","+225XXXXXXXXXX",145
			start := strings.Index(line, "\"")
			if start != -1 {
				secondQuote := strings.Index(line[start+1:], "\"")
				if secondQuote != -1 {
					thirdQuote := strings.Index(line[start+secondQuote+2:], "\"")
					if thirdQuote != -1 {
						number := line[start+secondQuote+2 : start+secondQuote+2+thirdQuote]
						number = strings.TrimSpace(number)
						if number != "" {
							return number, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("numéro non trouvé")
}

// ExecuteUSSD - Exécute un code USSD
func (s *SIM800C) ExecuteUSSD(code string, inputData string) (string, error) {
	s.Logger.Infof("Exécution USSD: %s", code)

	// Commande CUSD
	cmd := fmt.Sprintf("AT+CUSD=1,\"%s\",15", code)
	response, err := s.sendCommandWithResponse(cmd)
	if err != nil {
		return "", err
	}

	// Parser la réponse CUSD
	if strings.Contains(response, "+CUSD:") {
		// Extraire le message entre guillemets
		start := strings.Index(response, "\"")
		if start != -1 {
			end := strings.LastIndex(response, "\"")
			if end > start {
				result := response[start+1 : end]
				// Décoder le texte (peut être en UCS2)
				return result, nil
			}
		}
		return response, nil
	}

	return response, fmt.Errorf("pas de réponse CUSD")
}

// SendSMS - Envoie un SMS
func (s *SIM800C) SendSMS(number, message string) error {
	s.Logger.Infof("Envoi SMS à %s", number)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Commande CMGS
	cmd := fmt.Sprintf("AT+CMGS=\"%s\"", number)
	_, err := s.SerialPort.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return err
	}

	// Attendre le prompt >
	time.Sleep(1 * time.Second)

	// Envoyer le message
	_, err = s.SerialPort.Write([]byte(message + "\x1A"))
	if err != nil {
		return err
	}

	// Attendre la confirmation
	timeout := time.After(30 * time.Second)
	scanner := bufio.NewScanner(s.SerialPort)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout envoi SMS")
		default:
			if scanner.Scan() {
				line := scanner.Text()
				s.Logger.Debugf("Réponse SMS: %s", line)
				if strings.Contains(line, "+CMGS:") {
					s.Logger.Info("SMS envoyé avec succès")
					return nil
				}
				if strings.Contains(line, "ERROR") {
					return fmt.Errorf("erreur envoi SMS")
				}
			}
		}
	}
}

// ReadSMS - Lit un SMS par son index
func (s *SIM800C) ReadSMS(index int) (string, string, error) {
	cmd := fmt.Sprintf("AT+CMGR=%d", index)
	response, err := s.sendCommandWithResponse(cmd)
	if err != nil {
		return "", "", err
	}

	// Parser la réponse
	lines := strings.Split(response, "\n")
	var sender, message string

	for i, line := range lines {
		if strings.Contains(line, "+CMGR:") {
			// Extraire l'expéditeur
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

	return sender, message, nil
}

// DeleteSMS - Supprime un SMS par son index
func (s *SIM800C) DeleteSMS(index int) error {
	cmd := fmt.Sprintf("AT+CMGD=%d", index)
	return s.sendCommand(cmd, "OK")
}

// ListSMS - Liste tous les SMS
func (s *SIM800C) ListSMS() ([]map[string]interface{}, error) {
	response, err := s.sendCommandWithResponse("AT+CMGL=\"ALL\"")
	if err != nil {
		return nil, err
	}

	var smsList []map[string]interface{}
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		if strings.Contains(line, "+CMGL:") {
			// Format: +CMGL: index,status,sender,,date
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				indexStr := strings.TrimSpace(parts[0])
				indexStr = strings.TrimPrefix(indexStr, "+CMGL: ")
				sms := map[string]interface{}{
					"index":  indexStr,
					"status": strings.TrimSpace(parts[1]),
					"sender": strings.Trim(parts[2], "\""),
				}
				smsList = append(smsList, sms)
			}
		}
	}

	return smsList, nil
}

// handleCommands - Gère la file d'attente de commandes
func (s *SIM800C) handleCommands() {
	for {
		select {
		case cmd := <-s.commandChan:
			switch cmd.Type {
			case "ussd":
				result, err := s.ExecuteUSSD(cmd.USSDCode, cmd.InputData)
				if err != nil {
					cmd.Response <- fmt.Sprintf("Erreur: %v", err)
				} else {
					cmd.Response <- result
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

// readResponses - Lit les réponses asynchrones du module
func (s *SIM800C) readResponses() {
	scanner := bufio.NewScanner(s.SerialPort)
	for scanner.Scan() {
		line := scanner.Text()
		s.Logger.Debugf("Réception: %s", line)

		// Gérer les SMS entrants
		if strings.Contains(line, "+CMTI:") {
			go s.handleIncomingSMS(line)
		}

		// Gérer les réponses USSD asynchrones
		if strings.Contains(line, "+CUSD:") {
			s.Logger.Infof("USSD Response: %s", line)
		}
	}
}

// handleIncomingSMS - Gère la réception d'un SMS entrant
func (s *SIM800C) handleIncomingSMS(notification string) {
	// Extraire l'index du SMS
	var index int
	fmt.Sscanf(notification, "+CMTI: \"SM\",%d", &index)

	sender, message, err := s.ReadSMS(index)
	if err != nil {
		s.Logger.Errorf("Erreur lecture SMS: %v", err)
		return
	}

	s.Logger.Infof("SMS reçu de %s: %s", sender, message)
}

// SendCommand - Envoie une commande et attend la réponse
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

// isDigits - Vérifie si une chaîne ne contient que des chiffres
func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
