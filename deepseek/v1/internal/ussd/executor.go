package ussd

import (
	"fmt"
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
		Result:   result,
		Duration: time.Since(startTime),
	}, nil
}

func (e *USSDExecutor) ExecuteWithMenu(req *USSDRequest, choice string) (*USSDResponse, error) {
	// Pour les menus USSD, on envoie le choix après le code initial
	fullCode := fmt.Sprintf("%s%s", req.Code, choice)
	req.Code = fullCode
	return e.Execute(req)
}

func (e *USSDExecutor) ParseMenuResponse(response string) []MenuOption {
	var options []MenuOption

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Chercher les options de menu (format: "1. Option texte")
		if len(line) > 3 && line[1] == '.' {
			option := MenuOption{
				Number:   string(line[0]),
				Text:     strings.TrimSpace(line[3:]),
				FullText: line,
			}
			options = append(options, option)
		}

		// Chercher les options avec format "1: Option texte"
		if strings.Contains(line, ":") && len(line) > 2 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && len(parts[0]) == 1 && parts[0][0] >= '0' && parts[0][0] <= '9' {
				option := MenuOption{
					Number:   parts[0],
					Text:     strings.TrimSpace(parts[1]),
					FullText: line,
				}
				options = append(options, option)
			}
		}
	}

	return options
}
