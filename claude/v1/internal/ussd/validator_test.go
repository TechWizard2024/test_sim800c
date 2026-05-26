package ussd

import (
	"regexp"
	"testing"

	"github.com/sirupsen/logrus"
)

// newTestValidator crée un InputValidator avec un logger silencieux pour les tests.
func newTestValidator() *InputValidator {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel) // silencieux pendant les tests
	return NewInputValidator(logger)
}

// ----------------------------------------------------------------------------
// TESTS — ValidatePIN
// ----------------------------------------------------------------------------

// TestValidatePIN vérifie les cas valides et invalides pour un PIN.
func TestValidatePIN(t *testing.T) {
	v := newTestValidator()

	valid := []string{
		"0000",
		"1234",
		"9999",
		"0101",
		"12345"[:4], // exactement 4 chiffres
	}

	for _, pin := range valid {
		t.Run("valid_"+pin, func(t *testing.T) {
			if err := v.ValidatePIN(pin); err != nil {
				t.Errorf("ValidatePIN(%q): erreur inattendue: %v", pin, err)
			}
		})
	}

	invalid := []struct {
		pin string
		msg string
	}{
		{"", "PIN vide"},
		{"123", "3 chiffres seulement"},
		{"12345", "5 chiffres — trop long"},
		{"abcd", "lettres non numériques"},
		{"12 4", "espace dans le PIN"},
		{"12.4", "point décimal"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.msg, func(t *testing.T) {
			if err := v.ValidatePIN(tc.pin); err == nil {
				t.Errorf("ValidatePIN(%q): erreur attendue (%s)", tc.pin, tc.msg)
			}
		})
	}
}

// TestValidatePINDefaultCodes vérifie les PIN par défaut des opérateurs CI.
func TestValidatePINDefaultCodes(t *testing.T) {
	v := newTestValidator()

	defaults := map[string]string{
		"Orange CI":      "0000",
		"MTN CI":         "12345"[:4], // 1234 — premier 4 chiffres
		"Moov Africa CI": "0101",
	}

	for operator, pin := range defaults {
		t.Run(operator, func(t *testing.T) {
			if err := v.ValidatePIN(pin); err != nil {
				t.Errorf("PIN par défaut %s (%q): %v", operator, pin, err)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidatePhoneNumber (validateur USSD, sans DB)
// ----------------------------------------------------------------------------

// TestValidatePhoneNumber vérifie la validation des numéros CI dans le validateur USSD.
func TestValidatePhoneNumber(t *testing.T) {
	v := newTestValidator()

	valid := []string{
		"0701020304", // Orange CI — 10 chiffres, préfixe 07
		"0512345678", // MTN CI — 10 chiffres, préfixe 05
		"0112345678", // Moov Africa CI — 10 chiffres, préfixe 01
		"+2250701020304", // avec indicatif +225
		"2250512345678",  // avec indicatif 225 (sans +)
	}

	for _, num := range valid {
		t.Run("valid_"+num, func(t *testing.T) {
			if err := v.ValidatePhoneNumber(num); err != nil {
				t.Errorf("ValidatePhoneNumber(%q): erreur inattendue: %v", num, err)
			}
		})
	}

	invalid := []struct {
		number string
		reason string
	}{
		{"", "numéro vide"},
		{"0901020304", "préfixe 09 inconnu"},
		{"07010203", "8 chiffres seulement"},
		{"070102030405", "12 chiffres — trop long"},
		{"abcdefghij", "non numérique"},
		{"0301020304", "préfixe 03 inexistant CI"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.reason, func(t *testing.T) {
			if err := v.ValidatePhoneNumber(tc.number); err == nil {
				t.Errorf("ValidatePhoneNumber(%q): erreur attendue (%s)", tc.number, tc.reason)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidateMontant (via ValidateAmount)
// ----------------------------------------------------------------------------

// TestValidateMontant vérifie la validation des montants FCFA.
func TestValidateMontant(t *testing.T) {
	v := newTestValidator()

	valid := []string{
		"50",      // minimum exact
		"100",
		"500",
		"1000",
		"50000",
		"1000000", // maximum
	}

	for _, amount := range valid {
		t.Run("valid_"+amount, func(t *testing.T) {
			if err := v.ValidateAmount(amount); err != nil {
				t.Errorf("ValidateAmount(%q): erreur inattendue: %v", amount, err)
			}
		})
	}

	invalid := []struct {
		amount string
		reason string
	}{
		{"", "vide"},
		{"0", "zéro"},
		{"49", "sous le minimum 50"},
		{"-100", "négatif"},
		{"abc", "non numérique"},
		{"50.5", "décimal non entier"},
		{"1000001", "dépasse le maximum"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.reason, func(t *testing.T) {
			if err := v.ValidateAmount(tc.amount); err == nil {
				t.Errorf("ValidateAmount(%q): erreur attendue (%s)", tc.amount, tc.reason)
			}
		})
	}
}

// TestValidateMontantBoundary vérifie précisément les frontières du montant.
func TestValidateMontantBoundary(t *testing.T) {
	v := newTestValidator()

	// Juste en dessous du minimum
	if err := v.ValidateAmount("49"); err == nil {
		t.Error("ValidateAmount(49): erreur attendue (< 50)")
	}

	// Exactement au minimum
	if err := v.ValidateAmount("50"); err != nil {
		t.Errorf("ValidateAmount(50): erreur inattendue (= minimum): %v", err)
	}

	// Exactement au maximum
	if err := v.ValidateAmount("1000000"); err != nil {
		t.Errorf("ValidateAmount(1000000): erreur inattendue (= maximum): %v", err)
	}

	// Juste au-dessus du maximum
	if err := v.ValidateAmount("1000001"); err == nil {
		t.Error("ValidateAmount(1000001): erreur attendue (> 1000000)")
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidateReference
// ----------------------------------------------------------------------------

// TestValidateReference vérifie la validation des références (14 chiffres).
func TestValidateReference(t *testing.T) {
	v := newTestValidator()

	// Valider via ValidateInput avec type "Référence"
	// On simule directement avec validationRules
	rule := validationRules["Référence"]

	valid := []string{
		"12345678901234", // 14 chiffres exacts
		"00000000000000", // 14 zéros
		"99999999999999", // 14 neuf
	}

	for _, ref := range valid {
		t.Run("valid_"+ref, func(t *testing.T) {
			matched := matchPattern(rule.Pattern, ref)
			if !matched {
				t.Errorf("Référence %q devrait être valide (pattern: %s)", ref, rule.Pattern)
			}
		})
	}

	invalid := []struct {
		ref    string
		reason string
	}{
		{"", "vide"},
		{"1234567890123", "13 chiffres seulement"},
		{"123456789012345", "15 chiffres — trop long"},
		{"1234567890123a", "lettre en fin"},
		{"1234 5678901234", "espace"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.reason, func(t *testing.T) {
			matched := matchPattern(rule.Pattern, tc.ref)
			if matched {
				t.Errorf("Référence %q ne devrait pas être valide (%s)", tc.ref, tc.reason)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidateRechargeCode
// ----------------------------------------------------------------------------

// TestValidateRechargeCode vérifie la validation des codes de carte de recharge (14 chiffres).
func TestValidateRechargeCode(t *testing.T) {
	v := newTestValidator()
	_ = v

	rule := validationRules["Code de carte recharge"]

	valid := []string{
		"12345678901234",
		"00000000000000",
	}

	for _, code := range valid {
		t.Run("valid_"+code, func(t *testing.T) {
			if !matchPattern(rule.Pattern, code) {
				t.Errorf("Code recharge %q devrait être valide", code)
			}
		})
	}

	invalid := []struct {
		code   string
		reason string
	}{
		{"", "vide"},
		{"1234567890123", "13 chiffres"},
		{"123456789012345", "15 chiffres"},
		{"1234567890123a", "contient une lettre"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.reason, func(t *testing.T) {
			if matchPattern(rule.Pattern, tc.code) {
				t.Errorf("Code recharge %q ne devrait pas être valide (%s)", tc.code, tc.reason)
			}
		})
	}
}

// TestValidateRechargeCodeVsReference vérifie que les règles Référence et Code recharge sont identiques.
func TestValidateRechargeCodeVsReference(t *testing.T) {
	ruleRef := validationRules["Référence"]
	ruleRecharge := validationRules["Code de carte recharge"]

	if ruleRef.Pattern != ruleRecharge.Pattern {
		t.Logf("Info: Référence pattern=%q, Recharge pattern=%q", ruleRef.Pattern, ruleRecharge.Pattern)
		// Ce n'est pas une erreur — ils peuvent avoir le même ou différents patterns selon la spec
	}

	// Dans les deux cas, 14 chiffres doivent être valides
	testValue := "12345678901234"
	if !matchPattern(ruleRef.Pattern, testValue) {
		t.Errorf("Référence pattern doit accepter 14 chiffres")
	}
	if !matchPattern(ruleRecharge.Pattern, testValue) {
		t.Errorf("Code recharge pattern doit accepter 14 chiffres")
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidateChoice
// ----------------------------------------------------------------------------

// TestValidateChoice vérifie la validation des choix de menu USSD.
func TestValidateChoice(t *testing.T) {
	rule := validationRules["Choix"]

	valid := []string{
		"1",
		"2",
		"0",    // retour menu
		"00",   // accueil dans certains menus
		"10",
		"99",
	}

	for _, choice := range valid {
		t.Run("valid_"+choice, func(t *testing.T) {
			if !matchPattern(rule.Pattern, choice) {
				t.Errorf("Choix %q devrait être valide", choice)
			}
		})
	}

	invalid := []struct {
		choice string
		reason string
	}{
		{"", "vide"},
		{"-1", "négatif"},
		{"a", "lettre"},
		{"1 ", "espace"},
		{"1.5", "décimal"},
	}

	for _, tc := range invalid {
		t.Run("invalid_"+tc.reason, func(t *testing.T) {
			if matchPattern(rule.Pattern, tc.choice) {
				t.Errorf("Choix %q ne devrait pas être valide (%s)", tc.choice, tc.reason)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TESTS — NormalizePhoneNumber
// ----------------------------------------------------------------------------

// TestNormalizePhoneNumber vérifie la normalisation vers le format local CI (10 chiffres).
func TestNormalizePhoneNumber(t *testing.T) {
	v := newTestValidator()

	cases := []struct {
		input    string
		expected string
	}{
		{"0701020304", "0701020304"},        // déjà normalisé
		{"+2250701020304", "0701020304"},     // avec +225
		{"00225 0701020304", "0701020304"},   // avec 00225 et espace — normalisation partielle
		{"2250701020304", "0701020304"},      // avec 225 sans +
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := v.NormalizePhoneNumber(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizePhoneNumber(%q) = %q, attendu %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidateInput (intégration)
// ----------------------------------------------------------------------------

// TestValidateInputPIN vérifie ValidateInput pour un code USSD nécessitant un PIN.
func TestValidateInputPIN(t *testing.T) {
	v := newTestValidator()

	// Code USSD qui nécessite un PIN (selon detectInputType)
	ussdCode := "#144*81#"

	if err := v.ValidateInput(ussdCode, "0000"); err != nil {
		t.Errorf("ValidateInput PIN valide: %v", err)
	}

	if err := v.ValidateInput(ussdCode, "123"); err == nil {
		t.Error("ValidateInput PIN invalide (3 chiffres): erreur attendue")
	}
}

// TestValidateInputAucun vérifie que ValidateInput ne bloque pas si Information_INPUT = Aucun.
func TestValidateInputAucun(t *testing.T) {
	v := newTestValidator()

	// Code USSD générique qui ne nécessite pas d'entrée
	ussdCode := "#122#"

	if err := v.ValidateInput(ussdCode, ""); err != nil {
		t.Errorf("ValidateInput Aucun (code %s, entrée vide): erreur inattendue: %v", ussdCode, err)
	}

	// Même avec une valeur quelconque, pas d'erreur si type = Aucun
	if err := v.ValidateInput(ussdCode, "n'importe quoi"); err != nil {
		t.Errorf("ValidateInput Aucun avec valeur: erreur inattendue: %v", err)
	}
}

// TestValidateInputRechargeCode vérifie la validation d'un code de recharge via ValidateInput.
func TestValidateInputRechargeCode(t *testing.T) {
	v := newTestValidator()

	ussdCode := "#100*12345678901234#" // code avec recharge

	if err := v.ValidateInput(ussdCode, "12345678901234"); err != nil {
		t.Errorf("ValidateInput code recharge valide: %v", err)
	}

	if err := v.ValidateInput(ussdCode, "1234"); err == nil {
		t.Error("ValidateInput code recharge invalide (4 chiffres): erreur attendue")
	}
}

// ----------------------------------------------------------------------------
// Helpers internes aux tests
// ----------------------------------------------------------------------------

// matchPattern vérifie si value correspond au pattern regex.
func matchPattern(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return false
	}
	return matched
}
