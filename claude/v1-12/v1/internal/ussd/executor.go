package ussd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"sim800c-supervisor/internal/serial"

	"github.com/sirupsen/logrus"
)

type USSDExecutor struct {
	logger *logrus.Logger
}

type USSDRequest struct {
	Module    *serial.SIM800C
	Code      string
	InputData string
	ModuleID  int
}

type USSDResponse struct {
	Success   bool
	Result    string
	Error     string
	Duration  time.Duration
	SessionID string
}

func NewUSSDExecutor(logger *logrus.Logger) *USSDExecutor {
	return &USSDExecutor{
		logger: logger,
	}
}

func (e *USSDExecutor) Execute(req *USSDRequest) (*USSDResponse, error) {
	startTime := time.Now()

	e.logger.Infof("Exécution USSD sur module %d: %s", req.ModuleID, req.Code)

	// Validation des données d'entrée si nécessaire
	if req.InputData != "" {
		validator := NewInputValidator(e.logger)
		if err := validator.ValidateInput(req.Code, req.InputData); err != nil {
			return &USSDResponse{
				Success:  false,
				Error:    fmt.Sprintf("Validation échouée: %v", err),
				Duration: time.Since(startTime),
			}, err
		}
	}

	// Exécuter la commande USSD
	cmd := serial.Command{
		Type:      "ussd",
		USSDCode:  req.Code,
		InputData: req.InputData,
	}

	result, err := req.Module.SendCommand(cmd)
	if err != nil {
		return &USSDResponse{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, err
	}

	return &USSDResponse{
		Success:  true,
		Result:   FormatUSSDText(result),
		Duration: time.Since(startTime),
	}, nil
}

// ExecuteWithMenu sends a menu choice in an ongoing USSD session.
// In SIM800C mode B, after the initial menu is shown, you send the choice
// as AT+CUSD=1,"<choice>",15 — NOT AT+CUSD=1,"<parentcode>*<choice>#",15
func (e *USSDExecutor) ExecuteWithMenu(req *USSDRequest, choice string) (*USSDResponse, error) {
	startTime := time.Now()
	e.logger.Infof("Navigation menu USSD module %d: choix=%s", req.ModuleID, choice)

	// Send choice directly via ExecuteUSSDRaw (bypasses the commandChan to avoid queuing issues)
	result, err := req.Module.ExecuteUSSDRaw(choice)
	if err != nil {
		return &USSDResponse{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, err
	}

	return &USSDResponse{
		Success:  true,
		Result:   FormatUSSDText(result),
		Duration: time.Since(startTime),
	}, nil
}

// FormatUSSDText nettoie et formate le texte brut d'une réponse USSD.
// Le SIM800C retourne parfois des textes avec espaces superflus, caractères
// de remplacement (▒, □) dus à l'encodage GSM-7, et sauts de ligne incohérents.
//
// Règles :
//  1. Substitution des caractères d'encodage GSM-7 mal décodés
//  2. Normalisation des fins de ligne
//  3. Découpage des options concaténées (séquences de 3+ espaces)
//  4. Préservation des séparateurs "- - -" et "---"
//  5. Suppression des lignes vraiment vides
func FormatUSSDText(raw string) string {
	if raw == "" {
		return raw
	}

	// Substitution des caractères d'encodage (GSM-7 → UTF-8 incomplet)
	replacer := strings.NewReplacer(
		"▒", "é",
		"□", " ",
		"■", " ",
		"\x00", "",
		// Variantes communes d'accents mal encodés
		"Ã©", "é",
		"Ã¨", "è",
		"Ã ", "à",
		"Ã´", "ô",
		"Ã»", "û",
		"Ã®", "î",
		"Ã§", "ç",
	)
	cleaned := replacer.Replace(raw)

	// Normaliser les fins de ligne
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	// Pré-compiler la regex une seule fois (hors boucle)
	multiSpaceRe := regexp.MustCompile(`\s{3,}`)

	// Traiter ligne par ligne
	lines := strings.Split(cleaned, "\n")
	result := make([]string, 0, len(lines)+4)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Préserver les séparateurs visuels "- - -", "---", "──"  tels quels
		stripped := strings.ReplaceAll(trimmed, " ", "")
		if stripped == "---" || stripped == "──────" {
			result = append(result, "- - -")
			continue
		}

		// Découper les options concaténées sur une même ligne (3+ espaces = séparateur)
		parts := multiSpaceRe.Split(trimmed, -1)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
	}

	return strings.Join(result, "\n")
}

// ParseMenuResponse parses USSD menu text into options.
// Handles formats:
//   "1: Option text"
//   "1. Option text"
//   "00:Accueil"  (multi-digit options)
//   "0:Retour"
func (e *USSDExecutor) ParseMenuResponse(response string) []MenuOption {
	var options []MenuOption
	seen := map[string]bool{}

	// Match patterns like "1:", "2:", "00:", "0:" possibly preceded by spaces/dashes
	// and also "1." format
	menuRe := regexp.MustCompile(`(?m)^\s*(\d{1,2})[:.]\s*(.+)$`)

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" {
			continue
		}

		matches := menuRe.FindStringSubmatch(line)
		if len(matches) == 3 {
			num := strings.TrimSpace(matches[1])
			text := strings.TrimSpace(matches[2])
			if text != "" && !seen[num] {
				seen[num] = true
				options = append(options, MenuOption{
					Number:   num,
					Text:     text,
					FullText: line,
				})
			}
		}
	}

	return options
}
