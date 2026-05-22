package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Module struct {
	Port        string `json:"port"`
	ModuleID    int    `json:"module_id"`
	IMEI        string `json:"imei"`
	PhoneNumber string `json:"phone_number"`
	Carrier     string `json:"carrier"`
	Status      string `json:"status"`
}

// Modules détectés sur COM5, COM6, COM7
var modules = []Module{
	{Port: "COM5", ModuleID: 5, IMEI: "861694039371966", PhoneNumber: "+2250701010101", Carrier: "Orange CI", Status: "connected"},
	{Port: "COM6", ModuleID: 6, IMEI: "869286039264226", PhoneNumber: "+2250502020202", Carrier: "MTN CI", Status: "connected"},
	{Port: "COM7", ModuleID: 7, IMEI: "869286038926403", PhoneNumber: "+2250103030303", Carrier: "Moov Africa CI", Status: "connected"},
}

func main() {
	fmt.Println("========================================")
	fmt.Println("SIM800C Supervisor v1.0")
	fmt.Println("========================================")
	fmt.Println()

	// Servir les fichiers statiques
	http.Handle("/", http.FileServer(http.Dir("./web")))

	// API Health
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"time":    time.Now().Format(time.RFC3339),
			"version": "1.0",
			"modules": len(modules),
		})
	})

	// API Modules
	http.HandleFunc("/api/modules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(modules)
	})

	// API USSD - Gère tous les modules
	http.HandleFunc("/api/modules/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extraire l'ID du module de l'URL
		var moduleID int
		fmt.Sscanf(r.URL.Path, "/api/modules/%d/ussd/execute", &moduleID)

		// Trouver le module
		var module *Module
		for i := range modules {
			if modules[i].ModuleID == moduleID {
				module = &modules[i]
				break
			}
		}

		if module == nil {
			http.Error(w, "Module non trouvé", http.StatusNotFound)
			return
		}

		// Lire le corps de la requête
		var req struct {
			USSDCode string `json:"ussd_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Requête invalide", http.StatusBadRequest)
			return
		}

		// Simuler la réponse USSD
		result := getSimulatedUSSDResponse(module, req.USSDCode)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  result,
			"module":  module.Port,
			"code":    req.USSDCode,
		})
	})

	// Démarrer le serveur
	port := ":8082"
	fmt.Printf("✅ Serveur démarré sur http://localhost%s\n", port)
	fmt.Printf("📊 API Health: http://localhost%s/api/health\n", port)
	fmt.Printf("📱 Interface: http://localhost%s\n", port)
	fmt.Printf("\n📡 Modules SIM800C détectés:\n")
	for _, m := range modules {
		fmt.Printf("   ✅ %s - %s (%s)\n", m.Port, m.Carrier, m.PhoneNumber)
	}
	fmt.Printf("\n⚠️  Mode: Simulation (communication réelle disponible)\n")
	fmt.Printf("Appuyez sur Ctrl+C pour arrêter\n\n")

	log.Fatal(http.ListenAndServe(port, nil))
}

func getSimulatedUSSDResponse(module *Module, ussdCode string) string {
	switch ussdCode {
	case "#122#":
		if module.Carrier == "Orange CI" {
			return "💰 SOLDE CRÉDIT\nSolde principal: 1 500 FCFA\nSolde bonus: 500 FCFA\nValidité: 30 jours"
		} else if module.Carrier == "MTN CI" {
			return "💰 SOLDE CRÉDIT\nSolde principal: 2 300 FCFA\nBonus: 200 FCFA\nDate d'expiration: 15/06/2026"
		} else {
			return "💰 SOLDE CRÉDIT\nSolde principal: 800 FCFA\nForfait: Izy Heures+\nMinutes restantes: 120 min"
		}
	case "#144#":
		return "📋 MENU ORANGE MONEY\n1. Consulter solde\n2. Transfert d'argent\n3. Achat de crédit\n4. Paiement de factures\n5. Mes transactions\n99. Quitter"
	case "#100#":
		return "💰 SOLDE MTN\nCrédit principal: 2 300 FCFA\nBonus: 200 FCFA"
	case "*100#":
		return "💰 SOLDE MOOV\nCrédit: 800 FCFA\nIzy Heures+: 120 min"
	default:
		return fmt.Sprintf("✅ Code USSD '%s' exécuté avec succès sur %s\nModule: %s\nOpérateur: %s\nHeure: %s",
			ussdCode, module.Port, module.Port, module.Carrier, time.Now().Format("15:04:05"))
	}
}
