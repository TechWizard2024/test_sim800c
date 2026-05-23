package ussd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type InputValidator struct {
	logger *logrus.Logger
}

type ValidationRule struct {
	Type    string
	Pattern string
	Min     int
	Max     int
	Message string
}

var validationRules = map[string]ValidationRule{
	"Choix": {
		Type:    "Choix",
		Pattern: `^\d+$`,
		Min:     1,
		Message: "Le choix doit être un nombre valide",
	},
	"PIN": {
		Type:    "PIN",
		Pattern: `^\d{4}$`,
		Message: "Le PIN doit être composé de 4 chiffres",
	},
	"Code de carte recharge": {
		Type:    "Code de carte recharge",
		Pattern: `^\d{14}$`,
		Message: "Le code de recharge doit contenir 14 chiffres",
	},
	"Référence": {
		Type:    "Référence",
		Pattern: `^\d{14}$`,
		Message: "La référence doit contenir 14 chiffres",
	},
	"Numéro": {
		Type:    "Numéro",
		Pattern: `^0[157]\d{8}$`,
		Message: "Le numéro doit être un numéro CI valide (10 chiffres, commençant par 01, 05 ou 07)",
	},
	"numero de téléphone": {
		Type:    "Numéro",
		Pattern: `^0[157]\d{8}$`,
		Message: "Le numéro doit être un numéro CI valide (10 chiffres, commençant par 01, 05 ou 07)",
	},
	"Montant": {
		Type:    "Montant",
		Pattern: `^\d+$`,
		Min:     50,
		Message: "Le montant doit être un nombre supérieur ou égal à 50",
	},
}

func NewInputValidator(logger *logrus.Logger) *InputValidator {
	return &InputValidator{
		logger: logger,
	}
}

func (v *InputValidator) ValidateInput(ussdCode, input string) error {
	// Déterminer le type d'entrée attendu en fonction du code USSD
	inputType := v.detectInputType(ussdCode)

	if inputType == "" || inputType == "Aucun" {
		// Pas de validation nécessaire
		return nil
	}

	rule, exists := validationRules[inputType]
	if !exists {
		v.logger.Warnf("Type d'entrée non reconnu: %s", inputType)
		return nil
	}

	// Nettoyer l'entrée
	input = strings.TrimSpace(input)

	// Valider avec regex
	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, input)
		if err != nil {
			return fmt.Errorf("erreur validation: %w", err)
		}
		if !matched {
			return fmt.Errorf("%s", rule.Message)
		}
	}

	// Valider les valeurs numériques
	if rule.Min > 0 {
		value, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("le montant doit être un nombre")
		}
		if value < rule.Min {
			return fmt.Errorf("%s", rule.Message)
		}
	}

	if rule.Max > 0 {
		value, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("valeur invalide")
		}
		if value > rule.Max {
			return fmt.Errorf("la valeur ne doit pas dépasser %d", rule.Max)
		}
	}

	v.logger.Debugf("Validation réussie pour %s: %s", ussdCode, input)
	return nil
}

func (v *InputValidator) detectInputType(ussdCode string) string {
	// Logique de détection basée sur le code USSD
	// À enrichir avec les données de l'Excel

	switch {
	case strings.Contains(ussdCode, "*144*") && strings.Contains(ussdCode, "#"):
		return "PIN"
	case strings.Contains(ussdCode, "#100*"):
		return "Code de carte recharge"
	case strings.Contains(ussdCode, "*102*") || strings.Contains(ussdCode, "*108*"):
		return "Numéro"
	case strings.Contains(ussdCode, "*124*"):
		return "Numéro"
	case strings.Contains(ussdCode, "*111*"):
		return "Montant"
	case strings.Contains(ussdCode, "*155*"):
		if strings.Contains(ussdCode, "1#") {
			return "Numéro"
		}
		return "PIN"
	case ussdCode == "#144*81#" || ussdCode == "#144*84#":
		return "PIN"
	default:
		return "Aucun"
	}
}

func (v *InputValidator) ValidatePhoneNumber(number string) error {
	// Vérifier le format CI
	pattern := `^(0[157]|2250[157]|\+2250[157])\d{8}$`
	matched, err := regexp.MatchString(pattern, number)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("numéro de téléphone invalide")
	}
	return nil
}

func (v *InputValidator) NormalizePhoneNumber(number string) string {
	// Normaliser en format local (10 chiffres commençant par 0)
	number = strings.TrimSpace(number)

	// Enlever le préfixe international
	number = strings.TrimPrefix(number, "+225")
	number = strings.TrimPrefix(number, "00225")
	number = strings.TrimPrefix(number, "225")

	// Vérifier que ça commence par 0
	if !strings.HasPrefix(number, "0") {
		number = "0" + number
	}

	return number
}

func (v *InputValidator) ValidatePIN(pin string) error {
	if len(pin) != 4 {
		return fmt.Errorf("le PIN doit contenir 4 chiffres")
	}

	for _, c := range pin {
		if c < '0' || c > '9' {
			return fmt.Errorf("le PIN ne doit contenir que des chiffres")
		}
	}

	return nil
}

func (v *InputValidator) ValidateAmount(amount string) error {
	value, err := strconv.Atoi(amount)
	if err != nil {
		return fmt.Errorf("le montant doit être un nombre")
	}

	if value < 50 {
		return fmt.Errorf("le montant minimum est de 50 FCFA")
	}

	if value > 1000000 {
		return fmt.Errorf("le montant maximum est de 1 000 000 FCFA")
	}

	return nil
}
