# Design Système — Applications
> Projet : SIM800C USB Supervisor & USSD Manager  
> Version : v1  
> Date : 20 Mai 2026

---

## 1. Architecture Applicative Globale

```
┌─────────────────────────────────────────────────────────────────────┐
│                        COUCHE FRONTEND                              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   Single Page Application                   │   │
│  │                  (HTML + CSS + Vanilla JS)                   │   │
│  │                                                             │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐  │   │
│  │  │ Dashboard │ │  USSD     │ │   SMS     │ │  Settings │  │   │
│  │  │ (Fn 1,2)  │ │ Manager   │ │  Manager  │ │   / Logs  │  │   │
│  │  │           │ │ (Fn 3,4)  │ │  (Fn 5)   │ │           │  │   │
│  │  └───────────┘ └───────────┘ └───────────┘ └───────────┘  │   │
│  │                                                             │   │
│  │  WebSocket Client ←──────── Temps réel ──────────────────  │   │
│  │  REST Client      ←──────── API calls  ──────────────────  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                              │ HTTP / WebSocket
┌─────────────────────────────────────────────────────────────────────┐
│                        COUCHE BACKEND (GoLang)                      │
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │  HTTP Router │  │  WS Hub      │  │   Serial Manager         │  │
│  │  (Gin)       │  │  (Gorilla)   │  │   (goroutines/module)    │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────────┘  │
│         │                 │                       │                  │
│  ┌──────▼─────────────────▼───────────────────────▼──────────────┐  │
│  │                    Service Layer                               │  │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌───────────┐  │  │
│  │  │ AutoDisc.  │ │  USSD Svc  │ │  SMS Svc   │ │ Excel Svc │  │  │
│  │  │ Service    │ │  + FSM     │ │            │ │           │  │  │
│  │  └────────────┘ └────────────┘ └────────────┘ └───────────┘  │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                 │                                    │
│  ┌──────────────────────────────▼─────────────────────────────────┐  │
│  │                    Repository Layer (MySQL)                    │  │
│  │  modules | ussd_codes | ussd_executions | sms | logs | excel  │  │
│  └────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Structure du Projet GoLang

```
GSD/v1-OpenRouter/
│
├── main.go                          ← Point d'entrée, init serveur
├── .env                             ← Configuration (non versionné)
├── go.mod
├── go.sum
│
├── internal/
│   ├── config/
│   │   └── config.go                ← Chargement .env, struct Config
│   │
│   ├── serial/
│   │   ├── manager.go               ← Gestionnaire global des ports COM
│   │   ├── module.go                ← Struct Module + goroutines read/write
│   │   ├── at_commands.go           ← Bibliothèque commandes AT
│   │   └── discovery.go             ← Scan & auto-détection des ports COM
│   │
│   ├── ussd/
│   │   ├── service.go               ← Service USSD (exécution, parsing)
│   │   ├── fsm.go                   ← Machine d'état sessions USSD
│   │   ├── parser.go                ← Parser réponses USSD (texte → struct)
│   │   ├── validator.go             ← Validation des inputs (PIN, numéro, etc.)
│   │   └── auto_discovery.go        ← Exploration récursive des menus
│   │
│   ├── sms/
│   │   ├── service.go               ← CRUD SMS
│   │   ├── monitor.go               ← Surveillance SMS entrants (AT+CNMI)
│   │   └── filter.go                ← Règles de tri (corbeille automatique)
│   │
│   ├── excel/
│   │   ├── reader.go                ← Lecture Codes_USSD_CI.xlsx
│   │   ├── writer.go                ← Écriture + versioning horodaté
│   │   └── sync.go                  ← Sync DB ↔ Excel
│   │
│   ├── db/
│   │   ├── connection.go            ← Pool connexions MySQL
│   │   ├── migrations.go            ← Init schéma DB au démarrage
│   │   ├── modules_repo.go
│   │   ├── ussd_repo.go
│   │   ├── sms_repo.go
│   │   └── logs_repo.go
│   │
│   ├── websocket/
│   │   ├── hub.go                   ← Hub central + broadcast
│   │   ├── client.go                ← Gestion client WS individuel
│   │   └── events.go                ← Définition des types d'événements
│   │
│   └── api/
│       ├── router.go                ← Déclaration des routes
│       ├── middleware.go            ← CORS, logging, recovery
│       ├── handlers/
│       │   ├── modules.go           ← GET /api/modules
│       │   ├── discovery.go         ← POST /api/modules/:id/discover
│       │   ├── ussd.go              ← POST /api/modules/:id/ussd/execute
│       │   ├── ussd_auto.go         ← POST /api/modules/:id/ussd/auto
│       │   ├── sms.go               ← GET/POST/DELETE /api/modules/:id/sms
│       │   └── system.go            ← GET /api/health, /api/logs
│       └── dto/
│           ├── requests.go          ← Structs des requêtes HTTP
│           └── responses.go         ← Structs des réponses HTTP
│
└── public/                          ← Frontend web statique
    ├── index.html
    ├── css/
    │   ├── main.css
    │   └── themes.css               ← Variables CSS thème clair/sombre
    └── js/
        ├── app.js                   ← Initialisation + routing SPA
        ├── ws.js                    ← Client WebSocket
        ├── api.js                   ← Client API REST
        ├── dashboard.js             ← Fonction 1 & 2
        ├── ussd.js                  ← Fonctions 3 & 4
        └── sms.js                   ← Fonction 5
```

---

## 3. Conception du Backend GoLang

### 3.1 Gestionnaire Serial (Serial Manager)

Le `SerialManager` est le composant central de communication avec les modules SIM800C. Il maintient un registre des modules actifs et leurs goroutines dédiées.

```
SerialManager
├── modules: map[string]*Module       ← clé = port COM (ex: "COM5")
├── Discover() error                  ← Scan tous les ports COM disponibles
├── GetModule(port) *Module
└── BroadcastStatus()                 ← Envoi état via WebSocket

Module (par port COM)
├── Port: string                      ← "COM5"
├── IMEI: string
├── PhoneNumber: string
├── Carrier: string
├── Status: enum (online/offline/busy/error)
├── cmdQueue: chan ATCommand           ← File de commandes (buffered 100)
├── readLoop() goroutine              ← Lecture continue du port série
├── writeLoop() goroutine             ← Consommation de la queue
├── SendAT(cmd, timeout) Response     ← Envoi synchrone (via channel)
└── SendUSSD(code, timeout) Response  ← Envoi USSD + attente réponse
```

**Séquence d'initialisation d'un module :**
```
1. Ouvrir le port COM (baud: config, 8N1)
2. AT → OK (test connectivité)
3. ATE0 → OK (désactiver echo)
4. AT+CMEE=2 → OK (erreurs verboses)
5. AT+CPIN? → READY (SIM présente et déverrouillée)
6. AT+CREG? → signal réseau
7. *#06# → récupérer IMEI
8. #99# → récupérer numéro de téléphone
9. Déduire opérateur depuis préfixe du numéro
10. AT+CMGF=1 → mode texte SMS
11. AT+CNMI=2,2,0,0,0 → notifications SMS push
12. Marquer module "online", persister en DB, broadcaster via WS
```

---

### 3.2 Machine d'État USSD (FSM)

Les sessions USSD multi-étapes (menus imbriqués) sont gérées par une FSM par module.

```
États FSM :
  IDLE → EXECUTING → WAITING_INPUT → EXECUTING → ... → COMPLETED
                                                    ↘ ERROR / TIMEOUT

USSDSession
├── ModuleID: string
├── InitialCode: string
├── CurrentMenu: MenuNode
├── History: []MenuNode
├── Status: FSMState
├── CreatedAt: time.Time
└── ExpiresAt: time.Time (now + 30s)

MenuNode
├── Code: string
├── Response: string
├── Options: []MenuOption
└── ParentID: *int

MenuOption
├── Number: int
├── Label: string
└── Code: string    ← code USSD complet pour cette option
```

---

### 3.3 Auto-Discovery USSD (Exploration récursive)

```
Algorithme d'exploration (Fonction 3) :
─────────────────────────────────────
1. Charger tous les codes USSD avec Action=Services_N1, Scope=In, Carrier=<opérateur SIM>
2. Pour chaque code USSD racine :
   a. Exécuter le code USSD → obtenir menu
   b. Parser les options du menu (ex: "1. Solde\n2. Transfert\n3. Retour")
   c. Pour chaque option :
      - Construire le code USSD de l'option (ex: *133*1#)
      - Vérifier si ce code existe dans Codes_USSD_CI.xlsx (DB)
      - Si NOUVEAU → enregistrer en DB + ajouter à la liste des nouveaux
      - Envoyer l'option → récupérer le sous-menu
      - Répéter récursivement (limite depth = 5 niveaux)
      - Envoyer "0" ou "99" pour revenir au niveau précédent
   d. Émettre événement WebSocket pour chaque nœud découvert
3. Si nouveaux codes découverts → générer nouvelle version Excel horodatée
4. Émettre événement "discovery_complete" avec rapport
```

---

### 3.4 API REST — Routes

| Méthode | Route | Description |
|---|---|---|
| GET | `/api/health` | Statut du système |
| GET | `/api/modules` | Liste tous les modules avec statut |
| POST | `/api/modules/discover` | Relancer l'auto-discovery des modules |
| GET | `/api/modules/:id` | Détails d'un module |
| POST | `/api/modules/:id/sim-status` | Exécuter SIM Status (Fn 2-1) |
| POST | `/api/modules/:id/sim-status/auto` | SIM Status Auto (Fn 2-2) |
| POST | `/api/modules/:id/ussd/execute` | Exécuter un code USSD (Fn 4) |
| POST | `/api/modules/:id/ussd/menu-discover` | Menu Discovery manuel (Fn 3-1) |
| POST | `/api/modules/:id/ussd/menu-discover/auto` | Menu Auto-Discovery (Fn 3-2) |
| GET | `/api/modules/:id/ussd/history` | Historique des exécutions USSD |
| GET | `/api/modules/:id/sms` | Lister les SMS |
| POST | `/api/modules/:id/sms` | Envoyer un SMS |
| DELETE | `/api/modules/:id/sms/:smsId` | Supprimer un SMS |
| PATCH | `/api/modules/:id/sms/:smsId/trash` | Déplacer en corbeille |
| GET | `/api/ussd-codes` | Tous les codes USSD (filtrés) |
| GET | `/api/logs` | Logs applicatifs (paginés) |
| GET | `/ws` | WebSocket endpoint |

---

### 3.5 WebSocket — Types d'Événements

```json
// Événement : module_status_changed
{ "type": "module_status_changed", "payload": { "port": "COM5", "status": "online", "carrier": "Orange CI", "phone_number": "0701020304", "signal": 18 } }

// Événement : ussd_result
{ "type": "ussd_result", "payload": { "module_id": 1, "code": "#122#", "response": "Votre solde est de 1500 FCFA", "status": "success", "execution_ms": 3200 } }

// Événement : ussd_menu_node
{ "type": "ussd_menu_node", "payload": { "module_id": 1, "depth": 2, "code": "*133*1#", "is_new": true, "options": ["1. Envoyer", "2. Recevoir"] } }

// Événement : sms_received
{ "type": "sms_received", "payload": { "module_id": 2, "from": "05XXXXXXXX", "message": "Test recharge", "auto_trashed": false } }

// Événement : discovery_progress
{ "type": "discovery_progress", "payload": { "module_id": 1, "progress": 45, "current_code": "*144*81#", "new_codes_found": 3 } }

// Événement : excel_updated
{ "type": "excel_updated", "payload": { "filename": "Codes_USSD_CI-v20052026-143022.xlsx", "new_codes": 5 } }
```

---

## 4. Conception du Frontend

### 4.1 Structure des Pages (SPA)

```
index.html
├── Header
│   ├── Titre / Logo
│   ├── Indicateur WS (connecté/déconnecté)
│   └── Bouton bascule thème clair/sombre
│
├── Navigation (onglets)
│   ├── Dashboard           ← Fonctions 1, 2
│   ├── USSD Manager        ← Fonctions 3, 4
│   └── SMS Manager         ← Fonction 5
│
└── Pages
    ├── [Dashboard]
    │   ├── Grille des modules (1 carte par module SIM800C)
    │   │   ├── Port COM / Statut (online/offline)
    │   │   ├── Opérateur + Numéro SIM
    │   │   ├── Signal réseau (barres)
    │   │   ├── IMEI
    │   │   └── Boutons : SIM Status (manuel) | SIM Status Auto
    │   └── Panneau résultats (flux temps réel)
    │
    ├── [USSD Manager]
    │   ├── Sélecteur de module
    │   ├── Boutons Menu Discovery (par opérateur)
    │   ├── Bouton "USSD Menu Auto-Discovery"
    │   ├── Zone saisie manuelle (Fonction 4)
    │   ├── Arbre de menu découvert (interactive)
    │   └── Historique des exécutions
    │
    └── [SMS Manager]
        ├── Sélecteur de module
        ├── Onglets : Reçus | Envoyés | Corbeille
        ├── Liste des SMS (avec actions)
        ├── Formulaire de rédaction SMS
        └── Compteur de SMS non lus
```

### 4.2 Système de Thèmes CSS

```css
/* Thème clair (défaut) */
:root {
  --bg-primary: #ffffff;
  --bg-secondary: #f5f5f5;
  --bg-card: #ffffff;
  --text-primary: #1a1a1a;
  --text-secondary: #666666;
  --accent: #f97316;        /* Orange CI couleur */
  --accent-2: #fbbf24;      /* MTN CI couleur */
  --accent-3: #3b82f6;      /* Moov CI couleur */
  --success: #22c55e;
  --error: #ef4444;
  --warning: #f59e0b;
  --border: #e2e8f0;
  --shadow: rgba(0,0,0,0.1);
}

/* Thème sombre */
[data-theme="dark"] {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-card: #1e293b;
  --text-primary: #f1f5f9;
  --text-secondary: #94a3b8;
  --border: #334155;
  --shadow: rgba(0,0,0,0.4);
  /* accent colors identiques */
}
```

### 4.3 Composants Clés

**Carte Module** : affiche le statut en temps réel d'un module SIM800C. Couleur de bordure : vert (online), rouge (offline), jaune (busy). Animation pulse pour les opérations en cours.

**Résultats USSD** : panneau de flux avec défilement automatique. Chaque résultat est horodaté, coloré selon le statut (succès/erreur/timeout), avec expansion pour voir la réponse complète.

**Arbre Menu USSD** : visualisation hiérarchique des menus découverts. Les nouveaux nœuds apparaissent avec une animation et un badge "Nouveau".

**Formulaire USSD avec validation** : saisie dynamique selon le type d'input requis (PIN → masqué, numéro → format 10 chiffres, montant → validé ≥ 50, etc.). Messages d'erreur inline.

**Console AT** : composant optionnel affichant le flux brut des commandes AT envoyées et des réponses reçues, par module.

---

## 5. Flux d'Exécution des Fonctions Principales

### Flux Fonction 1 — Module Auto-Discovery

```
Frontend                Backend GoLang              Modules SIM800C
   │                         │                             │
   │── GET /api/modules ─────►│                             │
   │◄─ 200 [] (vide) ─────────│                             │
   │                         │                             │
   │── POST /api/modules/discover ──►│                      │
   │                         │── scanPorts() ──────────────►│
   │                         │◄─ COM5, COM6, COM7 présents  │
   │                         │                             │
   │                         │── initModule(COM5) ─────────►│
   │                         │◄─ OK ────────────────────────│
   │                         │── AT+CGSN → IMEI ───────────►│
   │                         │◄─ "356938035643809" ─────────│
   │                         │── #99# → numéro ────────────►│
   │                         │◄─ "0701020304" ──────────────│
   │                         │── déduire carrier (07→Orange)│
   │                         │── INSERT modules DB          │
   │◄══ WS: module_status_changed (COM5, online, Orange CI) ══│
   │                         │                             │
   │   (répéter pour COM6, COM7)                           │
   │◄── 200 { modules: [...] } ──────────────────────────── │
```

### Flux Fonction 3-1 — Menu Manual Discovery

```
Frontend                Backend                  Module
   │                       │                       │
   │── clic bouton "Orange Money menu" ─────────── │
   │── POST /api/modules/1/ussd/menu-discover       │
   │   body: { ussd_id: 2 } ─────────────────────► │
   │                       │── AT+CUSD=1,"#144#" ──►│
   │◄══ WS: ussd_menu_node (depth:1, options:[...])  │
   │◄══ WS: ussd_menu_node (option 1→ #144*1#)       │
   │                       │── AT+CUSD=1,"#144*1#" ─►│
   │◄══ WS: ussd_menu_node (depth:2, is_new: true)   │
   │                       │── ... (récursif) ──────► │
   │◄══ WS: excel_updated ("Codes_USSD_CI-v...xlsx") │
   │◄═══════ WS: discovery_complete ════════════════  │
```

---

## 6. Validation des Inputs USSD — Implémentation

```go
// validator.go
type InputType string

const (
    InputNone     InputType = "Aucun"
    InputChoix    InputType = "Choix"
    InputPIN      InputType = "PIN"
    InputRecharge InputType = "Code de carte recharge"
    InputNumero   InputType = "Numéro"
    InputMontant  InputType = "Montant"
    InputRef      InputType = "Référence"
)

func ValidateInput(inputType InputType, value string, options []int) error {
    switch inputType {
    case InputNone:
        return nil
    case InputChoix:
        n, err := strconv.Atoi(value)
        if err != nil || !contains(options, n) {
            return fmt.Errorf("choix invalide, options disponibles: %v", options)
        }
    case InputPIN:
        if !regexp.MustCompile(`^\d{4}$`).MatchString(value) {
            return fmt.Errorf("PIN doit être exactement 4 chiffres")
        }
    case InputRecharge:
        if !regexp.MustCompile(`^\d{14}$`).MatchString(value) {
            return fmt.Errorf("code recharge doit être exactement 14 chiffres")
        }
    case InputNumero:
        if !regexp.MustCompile(`^\d{10}$`).MatchString(value) {
            return fmt.Errorf("numéro doit être exactement 10 chiffres (sans indicatif)")
        }
    case InputMontant:
        n, err := strconv.Atoi(value)
        if err != nil || n < 50 {
            return fmt.Errorf("montant doit être un nombre >= 50")
        }
    case InputRef:
        if !regexp.MustCompile(`^\d{14}$`).MatchString(value) {
            return fmt.Errorf("référence doit être exactement 14 chiffres")
        }
    }
    return nil
}
```

---

## 7. Gestion des Versions Excel

```go
// writer.go
func GenerateVersionedFilename() string {
    now := time.Now()
    return fmt.Sprintf("Codes_USSD_CI-v%s.xlsx",
        now.Format("02012006-150405"))  // DDMMYYYYy-HHMMSS
}

func SaveNewVersion(newCodes []USSDCode) (string, error) {
    mu.Lock()   // mutex pour éviter les conflits d'écriture
    defer mu.Unlock()

    // 1. Charger la dernière version
    // 2. Ajouter les nouveaux codes
    // 3. Générer le nom horodaté
    // 4. Sauvegarder dans storage/excel/
    // 5. Enregistrer en DB (table excel_versions)
    // 6. Retourner le nom du nouveau fichier
}
```

---

## 8. Dépendances GoLang (go.mod)

```
github.com/gin-gonic/gin              v1.9.x   ← Router HTTP
github.com/gorilla/websocket          v1.5.x   ← WebSocket
go.bug.st/serial                      v1.6.x   ← Communication port série
github.com/go-sql-driver/mysql        v1.7.x   ← Driver MySQL
github.com/joho/godotenv              v1.5.x   ← Chargement .env
github.com/xuri/excelize/v2           v2.8.x   ← Lecture/écriture Excel
github.com/rs/zerolog                 v1.31.x  ← Logging structuré
```
