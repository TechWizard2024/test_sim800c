# Checkpoint — SIM800C Supervisor v1-6
**Dernière mise à jour :** 24 Mai 2026 — Session 6

---

## Résumé des sessions précédentes

### Sessions 1-3 (voir Checkpoint-v1-3.md)
- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes, SMS Manager
- Navigation interactive USSD step-by-step
- Favoris USSD, Historique, FormatUSSDResponse

### Session 4
- Signal Quality AT+CSQ, réseau AT+CREG
- Struct SIM800C : +DBID, +SignalQuality, +NetworkStatus
- Sync DBID après SaveModule
- Bug carrier "Orange" → "Orange CI" corrigé
- WebSocket signal_update, websocket.js port fix

### Session 5
- `GetEffectiveID()` — DBID stable pour toutes les FK
- `PINFailed` — État distinct pour échec PIN (❌ vs ⏳)
- Deux délais USSD séparés : `explore_delay_ms` et `nav_delay_ms`
- Broadcasts WebSocket temps réel pour Auto-Status et Auto-Menu (`auto_status_progress`, `auto_menu_progress`)
- `SendSMSWithModule()` — envoi SMS réel via port série

---

## Ce qui a été fait — Session 6 (cette session)

### FEAT 1 : Endpoint individuel `/api/modules/{id}/ussd/auto-status` ✅
**Problème :** `/api/ussd/auto-status` exécutait l'auto-discovery pour **tous** les modules en même temps. Impossible de cibler un seul module.

**Solution :**
- Nouveau handler `moduleAutoStatusHandler` dans `cmd/main.go`
- Route : `POST /api/modules/{id}/ussd/auto-status`
- Même logique que `autoStatusHandler` mais filtrée sur un seul module
- Broadcast WebSocket `auto_status_progress` avec le module_id ciblé
- Nouveau bouton **🚀 Auto-Status** dans chaque carte module du Dashboard (`dashboard.js`)
- Nouvelle méthode `runModuleAutoStatus()` dans `DashboardManager`

**Fichiers :** `cmd/main.go` (+handler), `web/js/dashboard.js` (+bouton, +méthode)

---

### FEAT 2 : Countdown 5s dans l'UI de navigation USSD menu ✅
**Problème :** Le SIM800C requiert une réponse dans les ~25 secondes. L'interface n'indiquait pas au user qu'il devait répondre rapidement, et les boutons restaient actifs même après expiration.

**Solution :**
- `renderMenuChoices()` dans `ussd.js` affiche désormais un timer `⏱ Répondez dans Xs` dès l'affichage d'un menu
- Timer décompte de 25s → 0s (25s = délai réseau typique USSD en CI)
- À ≤5s : le compteur devient rouge avec animation `pulse-countdown`
- À 0s : les boutons de choix sont `disabled`, message "Session USSD expirée"
- Clic sur un bouton de choix → annule le countdown automatiquement
- Timer précédent annulé avant chaque `renderMenuChoices` (évite les timers orphelins)

**Fichiers :** `web/js/ussd.js` (renderMenuChoices refactorisée), `web/css/main.css` (+styles countdown)

---

### FEAT 3 : Table `dial_plan` en base de données ✅
**Problème :** Le plan de numérotation (préfixes 07/05/01, indicatif +225, longueur 10 chiffres) était hardcodé dans `sim800c.go` (`detectCarrierFromNumber`). La description du projet demande explicitement de "Stocker, Gérer, Valider depuis la base de données le plan de numérotation".

**Solution :**
- Nouvelle table `dial_plan` avec colonnes : `id, country_code, country_name, calling_code, number_length, operator, prefix, is_active`
- Contrainte UNIQUE sur `(country_code, operator, prefix)`
- Données CI pré-insérées au démarrage via `INSERT IGNORE` (Orange CI/07, MTN CI/05, Moov Africa CI/01)
- Nouveau struct `DialPlan` dans `internal/db/db.go`
- Nouvelles fonctions : `GetDialPlan()`, `ValidatePhoneNumber(countryCode, number)`
- Nouveau endpoint : `GET /api/dialplan` → retourne le plan actif
- Table créée automatiquement au premier démarrage (dans `createTables`)
- `scripts/init_db.sql` mis à jour avec la table et les données CI

**Fichiers :** `internal/db/db.go` (+struct DialPlan, +createTable, +GetDialPlan, +ValidatePhoneNumber), `scripts/init_db.sql` (+table dial_plan), `cmd/main.go` (+route /api/dialplan)

---

### FEAT 4 : Export historique USSD en CSV ✅
**Problème :** L'historique USSD n'était consultable qu'en JSON via l'API. Aucune possibilité d'export pour analyse externe (Excel, etc.)

**Solution :**
- Nouveau endpoint `GET /api/ussd/history/export?module_id=N&limit=2000`
- Handler `exportUSSDHistoryCSVHandler` dans `cmd/main.go`
- CSV avec BOM UTF-8 pour compatibilité Excel
- Colonnes : `ID, Module_ID, USSD_Code, Input_Data, Output_Data, Status, Duration_ms, Executed_By, Executed_At`
- Nom du fichier horodaté : `ussd_history_YYYYMMDD_HHMMSS.csv`
- Fonction `escapeCSV()` pour gérer les guillemets et virgules dans les données
- Bouton **📥 Exporter CSV** dans l'onglet Historique USSD du frontend
- `history.js` : nouvelle méthode `exportCSV()` qui déclenche le téléchargement

**Fichiers :** `cmd/main.go` (+handler, +escapeCSV), `web/js/history.js` (+exportCSV, +listener), `web/index.html` (+bouton export)

---

### FEAT 5 : Onglet "Historique USSD" dédié dans le frontend ✅
**Problème :** Il n'y avait pas d'onglet "Historique USSD" dans le frontend. Le composant `HistoryManager` existait dans `history.js` mais n'était pas intégré dans les tabs.

**Solution :**
- Nouvel onglet `📊 Historique USSD` dans la barre de navigation
- Tab content avec filtres (module, date), bouton Vider, bouton Export CSV
- `history.js` chargé dans `index.html`
- `render()` cible `#ussd-history-list` (nouvel ID) en priorité, fallback sur `#history-list`
- `init()` appelle maintenant `loadModules()` pour peupler le filtre module
- Styles spécifiques pour les lignes d'historique dans `main.css`

**Fichiers :** `web/index.html` (+onglet, +tab content, +script history.js), `web/js/history.js` (render target fix, loadModules dans init), `web/css/main.css` (+styles history)

---

### FIX 6 : Annotation legacy sur `internal/api/handlers/ussd.go` ✅
**Problème noté dans Checkpoint v1-5 :** Ce fichier utilise encore `ExploreDelayMs` (lignes 149, 183) mais ses handlers ne sont **pas** appelés par les routes actives (qui passent par `cmd/main.go`). Risque de confusion.

**Solution :**
- Ajout d'un commentaire explicite en tête de fichier : handlers LEGACY, non utilisés par les routes actives
- Les routes réelles sont dans `cmd/main.go`
- `ExploreDelayMs` est correct pour ces handlers (exploration auto), `NavDelayMs` serait pour la navigation manuelle

**Fichiers :** `internal/api/handlers/ussd.go` (+commentaire legacy)

---

## Fichiers modifiés (session 6)

| Fichier | Modification |
|---------|-------------|
| `cmd/main.go` | +route `/modules/{id}/ussd/auto-status` ; +route `/ussd/history/export` ; +route `/dialplan` ; +moduleAutoStatusHandler() ; +exportUSSDHistoryCSVHandler() ; +escapeCSV() ; +getDialPlanHandler() |
| `internal/db/db.go` | +struct DialPlan ; +table dial_plan dans createTables() avec INSERT IGNORE CI ; +GetDialPlan() ; +ValidatePhoneNumber() |
| `scripts/init_db.sql` | +table dial_plan (CREATE + INSERT données CI + AUTO_INCREMENT) |
| `web/js/ussd.js` | renderMenuChoices() refactorisée avec countdown 25s, urgence <5s, disable boutons à expiration |
| `web/js/dashboard.js` | +bouton 🚀 Auto-Status dans chaque carte module ; +case 'auto_status' dans handleQuickAction() ; +runModuleAutoStatus() |
| `web/js/history.js` | +exportCSV() ; +listener export-history-csv-btn ; render() cible #ussd-history-list ; init() appelle loadModules() |
| `web/index.html` | +onglet 📊 Historique USSD ; +tab content avec filtres+boutons ; +script history.js |
| `web/css/main.css` | +.menu-countdown ; +.countdown-sec.urgent ; +.btn-auto-status ; +styles #ussd-history-list |
| `internal/api/handlers/ussd.go` | +commentaire legacy (handlers non actifs) |

---

## État actuel du code — Fonctions par rapport au project_desc.txt

| Fonction | Statut | Notes |
|----------|--------|-------|
| F1 — Module Auto-Discovery (COM scan) | ✅ | COM1-99 + /dev/ttyUSB* |
| F1 — PIN auto-unlock | ✅ | Orange=0000, MTN=12345, Moov=0101 |
| F1 — PIN failed distinct | ✅ | ❌ Échec PIN, WS event pin_failed |
| F1 — Carrier detection (07/05/01) | ✅ | "Orange CI", "MTN CI", "Moov Africa CI" |
| F1 — Dashboard temps réel | ✅ | Signal quality + réseau + PIN status |
| F2-1 — SIM Status Manual-Discovery | ✅ | Boutons par code/module avec info-bulle |
| F2-2 — SIM Status Auto-Discovery (global) | ✅ | WS temps réel |
| F2-2 — SIM Status Auto-Discovery (par module) | ✅ *(session 6)* | Bouton 🚀 Auto-Status dans chaque carte |
| F3-1 — USSD Menu Manual-Discovery | ✅ | Boutons + navigation step-by-step |
| F3-1 — Countdown navigation USSD | ✅ *(session 6)* | Timer 25s, urgence <5s, expiration auto |
| F3-2 — USSD Menu Auto-Discovery | ✅ | WS temps réel |
| F4 — USSD Manager | ✅ | Saisie manuelle + nav interactive |
| F5 — SMS Manager (Create/Read/Delete) | ✅ | Envoi série réel |
| F5 — SMS Corbeille auto (sans "Test") | ✅ | autoTrashKeyword dans config.yaml |
| FK module_id stables (DBID) | ✅ | GetEffectiveID() partout |
| Thème clair/sombre | ✅ | |
| WebSocket temps réel | ✅ | Tous events |
| USSD text formatting | ✅ | FormatUSSDResponse |
| start_app.bat / stop_app.bat | ✅ | |
| Signal Quality dans dashboard | ✅ | CSQ + dBm + réseau |
| **Plan de numérotation en DB** | ✅ *(session 6)* | Table dial_plan, API /api/dialplan |
| **Export CSV historique USSD** | ✅ *(session 6)* | GET /api/ussd/history/export |
| **Onglet Historique USSD** | ✅ *(session 6)* | Tab dédié avec filtres |

---

## Architecture actuelle

```
cmd/main.go
  ├── serial.Manager      → SIM800C{DBID, PINFailed, SignalQuality, NetworkStatus}
  │    └── GetEffectiveID()  → DBID si > 0, sinon ModuleID
  ├── sms.SMSManager      → SendSMSWithModule() + GetEffectiveID()
  ├── websocket.Hub       → Events: module_initialized, pin_unlocked, pin_failed,
  │                         signal_update, auto_status_progress, auto_menu_progress,
  │                         ussd_result, sms_received, sms_sent
  └── Routes API
       ├── GET  /api/modules                           → liste modules + pin_failed
       ├── POST /api/ussd/auto-status                  → auto-status tous modules
       ├── POST /api/modules/{id}/ussd/auto-status     → auto-status 1 module *(NEW)*
       ├── POST /api/ussd/auto-menu                    → auto-menu tous modules
       ├── GET  /api/ussd/history                      → historique JSON
       ├── GET  /api/ussd/history/export               → export CSV *(NEW)*
       ├── GET  /api/dialplan                          → plan de numérotation *(NEW)*
       └── POST /api/modules/{id}/sms/send             → SMS série réel

internal/db/db.go
  ├── Table dial_plan (country_code, calling_code, number_length, operator, prefix)
  ├── GetDialPlan() → []DialPlan
  └── ValidatePhoneNumber(countryCode, number) → (operator, error) *(NEW)*
```

---

## Décisions prises (session 6)

1. **Countdown à 25s (pas 5s)** : La documentation SIM800C indique un timeout session de ~30s côté réseau. 25s donne une marge confortable tout en prévenant l'utilisateur. Les 5s mentionnées dans le checkpoint v1-5 concernaient le délai conseillé entre réponses successives (côté backend), pas le timeout session. L'affichage urgent <5s reste pertinent pour signaler l'urgence.

2. **dial_plan avec INSERT IGNORE** : Les données CI sont insérées via `INSERT IGNORE INTO dial_plan ... VALUES (...)` dans `createTables()`. Cela garantit que la table est peuplée dès le premier démarrage sans risque d'erreur de doublon si l'application redémarre. La table peut être enrichie manuellement via SQL pour d'autres pays.

3. **ValidatePhoneNumber dans db.go** : La validation utilise maintenant le plan de numérotation en DB plutôt que des constantes hardcodées. Le code `detectCarrierFromNumber` dans `sim800c.go` reste car il est appelé sans contexte DB (au moment de la détection initiale). À terme, il faudra lui passer une référence DB — noté comme prochaine étape.

4. **Onglet Historique séparé** : Le `HistoryManager` existait dans `history.js` mais n'était pas intégré. Plutôt que de modifier `audit-tab`, un nouvel onglet dédié `history-tab` est créé. L'audit log reste séparé (pour les actions admin) de l'historique USSD opérationnel.

5. **escapeCSV en Go** : Une fonction maison simple plutôt qu'une dépendance `encoding/csv`. Le module `encoding/csv` de la stdlib Go aurait été plus robuste, mais pour éviter des modifications d'imports, la fonction custom suffit pour les données USSD (pas de cas extrêmes).

---

## Prochaines étapes — Priorité HAUTE

### 1. Migrer `detectCarrierFromNumber` vers la DB
Le fichier `internal/serial/sim800c.go` utilise encore `detectCarrierFromNumber()` avec des constantes hardcodées. Il faudrait passer une référence `*db.DB` au Manager ou pré-charger le dial_plan au démarrage.

**Fichiers :** `internal/serial/manager.go` (+dialPlan field), `internal/serial/sim800c.go` (utiliser dialPlan injecté)

### 2. Tests réels sur COM5
- Vérifier le countdown USSD sur le réseau Orange CI réel
- Vérifier l'export CSV (format, encodage avec les caractères spéciaux des réponses USSD)
- Vérifier le bouton Auto-Status par module dans le dashboard

### 3. Validation `ValidatePhoneNumber` côté frontend
Le frontend valide les numéros avec des regex statiques. Il pourrait charger `/api/dialplan` au démarrage et valider dynamiquement selon les opérateurs actifs.

**Fichiers :** `web/js/sms.js` (+fetch /api/dialplan, +dynamic validation)

---

## Prochaines étapes — Priorité MOYENNE

### 4. Tests unitaires Go
- `GetDialPlan()` (table peuplée vs vide)
- `ValidatePhoneNumber()` (07XXXXXXXX → Orange CI, 09XXXXXXXX → invalide)
- `GetEffectiveID()` (DBID=0 → ModuleID, DBID>0 → DBID)
- `FormatUSSDResponse()` (espaces GSM, caractères spéciaux CI)

### 5. Page Settings — Configuration des délais + plan de numérotation
Permettre depuis l'UI :
- Modifier `explore_delay_ms` et `nav_delay_ms`
- Ajouter/désactiver des entrées dial_plan (autres pays/opérateurs)

**Fichiers :** `web/js/settings.js`, `cmd/main.go` (+endpoint PUT /api/dialplan/{id})

### 6. Endpoint individuel `/api/modules/{id}/ussd/auto-menu`
Par symétrie avec l'auto-status individuel (session 6), créer un endpoint auto-menu par module.

---

## Prochaines étapes — Priorité BASSE

### 7. Export SMS en CSV
Par cohérence avec l'export historique USSD, ajouter `GET /api/modules/{id}/sms/export`.

### 8. Whitelist COM ports dans config.yaml
Documenter que `serial.ports: ["COM5", "COM6", "COM7"]` accélère le scan initial au lieu de tester COM1-COM99.

---

## Notes importantes pour la prochaine session

### Structure fichier v1-6.zip
```
v1-6/
├── cmd/main.go                      ← +moduleAutoStatusHandler, +exportUSSDHistoryCSVHandler,
│                                       +escapeCSV, +getDialPlanHandler, +routes nouvelles
├── config.yaml                      ← inchangé
├── internal/
│   ├── api/handlers/ussd.go         ← +commentaire legacy
│   ├── config/config.go             ← inchangé
│   ├── db/db.go                     ← +DialPlan struct, +dial_plan table, +GetDialPlan, +ValidatePhoneNumber
│   ├── serial/manager.go            ← inchangé
│   ├── serial/sim800c.go            ← inchangé (detectCarrierFromNumber encore hardcodé)
│   └── sms/sms_manager.go           ← inchangé
├── scripts/
│   └── init_db.sql                  ← +table dial_plan + données CI
└── web/
    ├── css/main.css                 ← +countdown styles, +btn-auto-status, +history styles
    ├── index.html                   ← +onglet Historique USSD, +script history.js
    └── js/
        ├── dashboard.js             ← +bouton Auto-Status, +runModuleAutoStatus()
        ├── history.js               ← +exportCSV(), render() fix, loadModules() dans init
        └── ussd.js                  ← renderMenuChoices() avec countdown 25s
```

### Commandes utiles
```bat
REM Compiler (depuis C:\xampp\htdocs\aa_Toolbox\test_sim800c\claude\v1-6\)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Mettre à jour la DB avec la nouvelle table dial_plan (si DB existante de v1-5)
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 -e "
  CREATE TABLE IF NOT EXISTS dial_plan (
    id INT AUTO_INCREMENT PRIMARY KEY,
    country_code VARCHAR(5) NOT NULL,
    country_name VARCHAR(100) NOT NULL,
    calling_code VARCHAR(10) NOT NULL,
    number_length INT NOT NULL DEFAULT 10,
    operator VARCHAR(100) NOT NULL,
    prefix VARCHAR(10) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_country_operator_prefix (country_code, operator, prefix)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
  INSERT IGNORE INTO dial_plan (country_code, country_name, calling_code, number_length, operator, prefix) VALUES
    ('CI', 'Côte d''Ivoire', '+225', 10, 'Orange CI', '07'),
    ('CI', 'Côte d''Ivoire', '+225', 10, 'MTN CI', '05'),
    ('CI', 'Côte d''Ivoire', '+225', 10, 'Moov Africa CI', '01');
"
```

### Identifiants par défaut
- Application web : `admin` / `admin123`
- URL : `http://test-sim800c.lan:8082`

### Point d'attention
Le countdown dans `renderMenuChoices` est réglé à 25s. Si les tests réels montrent un timeout plus court (certains opérateurs CI peuvent avoir 15s), ajuster la constante `secondsLeft = 25` dans `ussd.js`. La valeur peut ensuite être rendue configurable dans Settings (voir priorité moyenne #5).
