# Checkpoint — SIM800C Supervisor v1-5
**Dernière mise à jour :** 23 Mai 2026 — Session 5

---

## Résumé des sessions précédentes

### Sessions 1-3 (voir Checkpoint-v1-3.md)
- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes, SMS Manager
- Navigation interactive USSD step-by-step
- Favoris USSD, Historique, FormatUSSDResponse

### Session 4 (voir Checkpoint-v1-3.md section "Session 4")
- Signal Quality AT+CSQ, réseau AT+CREG
- Struct SIM800C : +DBID, +SignalQuality, +NetworkStatus
- Sync DBID après SaveModule
- Bug carrier "Orange" → "Orange CI" corrigé
- WebSocket signal_update, websocket.js port fix
- Typo USSSCode → USSDCode

---

## Ce qui a été fait — Session 5 (cette session)

### FIX 1 : `GetEffectiveID()` — DBID stable pour toutes les FK ✅
**Problème :** `SaveUSSDHistory` et `SaveSMS` utilisaient `module.ModuleID` (ordre de découverte RAM : 1, 2, 3…). Après redémarrage le même module COM5 peut avoir ModuleID=1 mais DBID=5 en MySQL. Les FK `module_id` dans `ussd_history` et `sms_messages` pointaient vers de mauvaises lignes.

**Solution :**
- Ajout méthode `GetEffectiveID()` sur `SIM800C` dans `manager.go` :
  ```go
  func (s *SIM800C) GetEffectiveID() int {
      if s.DBID > 0 { return s.DBID }
      return s.ModuleID
  }
  ```
- Tous les appels `SaveUSSDHistory`, `SaveSMS`, `SMSExists`, `GetSMSByModule` utilisent maintenant `module.GetEffectiveID()` au lieu de `module.ModuleID`.

**Fichiers :** `internal/serial/manager.go`, `cmd/main.go` (executeUSSDHandler, navigateUSSDHandler), `internal/sms/sms_manager.go` (ReadSMS, AutoFilterTrash)

---

### FIX 2 : `PINFailed` — État distinct pour échec PIN ✅
**Problème :** Si tous les PIN par défaut échouaient, l'indicateur restait "⏳ En attente..." indéfiniment, sans différencier "pas encore tenté" de "tentative échouée".

**Solution :**
- Ajout champ `PINFailed bool` dans `SIM800C` struct (`manager.go`)
- Ajout méthode `markPINFailed()` dans `sim800c.go` : met `PINFailed=true` et diffuse `pin_failed` via WebSocket
- `initialize()` appelle `markPINFailed()` quand `checkAndUnlockPIN()` retourne une erreur
- `module_initialized` broadcast inclut maintenant `pin_failed`
- `/api/modules` retourne `pin_failed` dans chaque module
- Dashboard affiche ❌ Échec PIN (rouge, classe `.pin-error`) au lieu de ⏳

**Fichiers :** `internal/serial/manager.go`, `internal/serial/sim800c.go`, `cmd/main.go`, `web/js/dashboard.js`, `web/js/app.js`, `web/css/main.css`

---

### FIX 3 : Deux délais USSD séparés — explore vs navigation ✅
**Problème :** Un seul `explore_delay_ms = 1500ms` était utilisé pour tout. Or :
- L'exploration auto (Services_N1) doit être plus lente (>3s conseillé pour laisser le réseau répondre)
- La navigation manuelle est interactive (utilisateur attend < 5s, mais côté backend pas de délai artificiel)

**Solution :**
- Ajout champ `NavDelayMs int` dans `USSDConfig` (`internal/config/config.go`)
- `config.yaml` mis à jour :
  ```yaml
  ussd:
    explore_delay_ms: 3000   # délai auto-exploration
    nav_delay_ms: 500         # délai navigation manuelle
  ```
- Valeurs par défaut : `ExploreDelayMs=3000`, `NavDelayMs=500` si absents du YAML

**Fichiers :** `internal/config/config.go`, `config.yaml`

---

### FIX 4 : Broadcasts WebSocket temps réel pour Auto-Status et Auto-Menu ✅
**Problème :** `autoStatusDiscovery()` et `autoMenuDiscovery()` étaient des requêtes HTTP longues (~30-120s). Le frontend affichait "⏳ Exécution..." jusqu'à la fin — zéro feedback intermédiaire.

**Solution :**
- `autoStatusHandler` et `autoMenuHandler` acceptent maintenant un `*websocket.Hub` en paramètre
- À chaque étape (code USSD exécuté), un événement WS est diffusé :
  - `auto_status_progress` : `{ port, operation, ussd_code, result }`
  - `auto_menu_progress` : `{ port, operation, ussd_code, status: "exploring"|"done", result }`
- Frontend `app.js` gère ces 2 nouveaux events → ajoute `.live-progress-item` dans les divs dédiées
- `index.html` : 2 divs séparés `#auto-discovery-result` et `#auto-menu-result`
- À la fin, un `<details>` résumé complet est ajouté (repliable)

**Fichiers :** `cmd/main.go`, `web/js/app.js`, `web/index.html`, `web/css/main.css`

---

### FIX 5 : `SendSMSWithModule()` — SMS envoyé via port série réel ✅
**Problème :** L'ancien `SendSMS(moduleID, number, message)` sauvegardait en DB mais n'envoyait **pas** via le port série ! L'envoi réel manquait.

**Solution :**
- Ajout `SendSMSWithModule(module *serial.SIM800C, number, message string)` dans `sms_manager.go`
- Cette méthode : valide le numéro → appelle `module.SendSMS()` (envoi série réel) → sauvegarde en DB avec `GetEffectiveID()` → notifie via WS
- `sendSMSHandler` dans `main.go` accepte maintenant `*serial.Manager` pour trouver le module et appeler `SendSMSWithModule`

**Fichiers :** `internal/sms/sms_manager.go`, `cmd/main.go`

---

## Fichiers modifiés (session 5)

| Fichier | Modification |
|---------|-------------|
| `internal/serial/manager.go` | +PINFailed bool dans SIM800C struct ; +GetEffectiveID() méthode |
| `internal/serial/sim800c.go` | +markPINFailed() ; initialize() appelle markPINFailed() si PIN échoue ; module_initialized inclut pin_failed |
| `internal/config/config.go` | +NavDelayMs dans USSDConfig ; default ExploreDelayMs=3000, NavDelayMs=500 |
| `internal/sms/sms_manager.go` | +SendSMSWithModule() avec envoi série réel + GetEffectiveID() ; ReadSMS/AutoFilterTrash utilisent GetEffectiveID() |
| `cmd/main.go` | getModulesHandler +pin_failed ; executeUSSDHandler/navigateUSSDHandler → GetEffectiveID() ; autoStatusHandler/autoMenuHandler +hub param +WS broadcasts ; sendSMSHandler → SendSMSWithModule + serialManager param |
| `config.yaml` | ussd.explore_delay_ms=3000, +nav_delay_ms=500 |
| `web/js/app.js` | +case pin_failed ; +case auto_status_progress ; +case auto_menu_progress |
| `web/js/dashboard.js` | Affichage pin_failed : ❌ Échec PIN (classe pin-error) |
| `web/index.html` | +#auto-menu-result div séparé ; autoStatusDiscovery/autoMenuDiscovery live-progress style |
| `web/css/main.css` | +.pin-error ; +.live-progress-item/.live-port/.live-op/.live-result |

---

## État actuel du code — Fonctions par rapport au project_desc.txt

| Fonction | Statut | Notes |
|----------|--------|-------|
| F1 — Module Auto-Discovery (COM scan) | ✅ | COM1-99 + /dev/ttyUSB* |
| F1 — PIN auto-unlock | ✅ | Orange=0000, MTN=12345, Moov=0101 |
| F1 — PIN failed distinct | ✅ *(session 5)* | ❌ Échec PIN, WS event pin_failed |
| F1 — Carrier detection (07/05/01) | ✅ | "Orange CI", "MTN CI", "Moov Africa CI" |
| F1 — Dashboard temps réel | ✅ | Signal quality + réseau + PIN status |
| F2-1 — SIM Status Manual-Discovery | ✅ | Boutons par code/module avec info-bulle |
| F2-2 — SIM Status Auto-Discovery | ✅ | WS temps réel *(session 5)* |
| F3-1 — USSD Menu Manual-Discovery | ✅ | Boutons + navigation step-by-step |
| F3-2 — USSD Menu Auto-Discovery | ✅ | WS temps réel *(session 5)* |
| F4 — USSD Manager | ✅ | Saisie manuelle + nav interactive |
| F5 — SMS Manager (Create/Read/Delete) | ✅ | Envoi série réel *(session 5)* |
| F5 — SMS Corbeille auto (sans "Test") | ✅ | autoTrashKeyword dans config.yaml |
| FK module_id stables (DBID) | ✅ *(session 5)* | GetEffectiveID() partout |
| Thème clair/sombre | ✅ | |
| WebSocket temps réel | ✅ | Tous events + pin_failed + auto_*_progress |
| USSD text formatting | ✅ | FormatUSSDResponse |
| start_app.bat / stop_app.bat | ✅ | |
| Signal Quality dans dashboard | ✅ | CSQ + dBm + réseau |
| Universel codes dans boutons | ✅ | #99#, *#06# inclus pour tous |

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
       ├── GET  /api/modules          → pin_failed inclus
       ├── POST /api/ussd/auto-status → broadcast auto_status_progress par étape
       ├── POST /api/ussd/auto-menu   → broadcast auto_menu_progress par étape
       └── POST /api/modules/{id}/sms/send → SendSMSWithModule (envoi série réel)
```

---

## Décisions prises (session 5)

1. **GetEffectiveID() sur SIM800C** (pas sur Manager) : La méthode est sur le struct lui-même pour être appelée partout où on a un `*serial.SIM800C`, sans passer par le Manager. Pattern simple et sans indirection supplémentaire.

2. **Two delays séparés** : `explore_delay_ms` (3000ms) pour l'auto-exploration qui est asynchrone et peut attendre, `nav_delay_ms` (500ms) réservé pour les éventuels usages futurs où on voudrait throttler la navigation manuelle côté serveur. La navigation manuelle elle-même n'a pas de délai artificiel côté backend.

3. **PINFailed distinct de non-initialisé** : Trois états possibles dans le dashboard : ⏳ En attente (PINUnlocked=false, PINFailed=false), ✅ Déverrouillé (PINUnlocked=true), ❌ Échec PIN (PINFailed=true). Un module peut aussi n'avoir pas besoin de PIN (PINUnlocked=true dès le début).

4. **SendSMSWithModule** : La vieille `SendSMS(moduleID, ...)` est conservée pour compatibilité mais le nouveau handler l'utilise plus. Elle n'envoyait pas via le port série — bug critique silencieux depuis session 1.

5. **Divs séparés pour auto-status vs auto-menu** : `#auto-discovery-result` pour Status, `#auto-menu-result` pour Menu. Évite que les updates temps réel de l'un écrasent ceux de l'autre.

---

## Prochaines étapes — Priorité HAUTE

### 1. Test réel sur Windows/COM5
- Vérifier démarrage, détection module COM5, déverrouillage PIN automatique (0000)
- Vérifier `GetEffectiveID()` : après redémarrage, les module_id en DB restent stables
- Vérifier `FormatUSSDResponse` sur les réponses #111# et #122# réelles
- Vérifier l'envoi SMS réel (SendSMSWithModule appelle `module.SendSMS()` série)

### 2. Synchronisation auto-status par module individuel
Actuellement `/api/ussd/auto-status` exécute pour **tous** les modules. Ajouter un endpoint `POST /api/modules/{id}/ussd/auto-status` pour n'exécuter que sur un module spécifique.

**Fichiers :** `cmd/main.go` (+handler), `web/js/dashboard.js` (bouton par carte module)

### 3. Délai de navigation manuelle configurable côté frontend
Pour la navigation menu (Fonction 3-1), le SIM800C requiert une réponse en < 5 secondes. Ajouter un indicateur countdown dans l'UI quand un menu est affiché et que l'utilisateur doit choisir.

**Fichiers :** `web/js/ussd.js` (+countdown timer), `web/index.html`

---

## Prochaines étapes — Priorité MOYENNE

### 4. Endpoint `/api/modules/{id}/ussd/auto-status` individuel
Permet d'exécuter l'auto-discovery de statut pour un seul module sélectionné.

### 5. Export historique USSD en CSV/Excel
L'API `GET /api/ussd/history` retourne du JSON. Ajouter un bouton "Exporter CSV" dans l'UI history.

**Fichiers :** `web/js/history.js`, `cmd/main.go`

### 6. Whitelist COM ports dans config.yaml
Documenter dans `start_app.bat` et `config.yaml` que `serial.ports` peut lister les ports prioritaires pour accélérer le scan initial.

---

## Prochaines étapes — Priorité BASSE

### 7. Tests unitaires Go
- `GetEffectiveID()` (DBID=0 → retourne ModuleID, DBID>0 → retourne DBID)
- `detectCarrierFromNumber` (cas +225, 00225, sans indicatif)
- `FormatUSSDResponse` (espaces GSM, caractères spéciaux)
- `ParseMenuResponse` (options 0:, 00:, 8:-->)

### 8. Page Settings — Configuration des délais
Permettre de modifier `explore_delay_ms` et `nav_delay_ms` depuis l'UI Settings sans éditer le YAML.

---

## Notes importantes pour la prochaine session

### Structure fichier v1-5.zip
```
v1/
├── cmd/main.go                      ← sendSMSHandler signature changée
├── config.yaml                      ← explore_delay_ms=3000, nav_delay_ms=500
├── internal/
│   ├── config/config.go             ← +NavDelayMs
│   ├── serial/manager.go            ← +PINFailed, +GetEffectiveID()
│   ├── serial/sim800c.go            ← +markPINFailed(), pin_failed dans broadcasts
│   └── sms/sms_manager.go           ← +SendSMSWithModule(), GetEffectiveID() partout
└── web/
    ├── css/main.css                 ← +.pin-error, +.live-progress-item
    ├── index.html                   ← +#auto-menu-result, live progress display
    └── js/
        ├── app.js                   ← +pin_failed, +auto_status_progress, +auto_menu_progress
        └── dashboard.js             ← pin_failed → ❌ Échec PIN
```

### Commandes utiles
```bat
REM Compiler (depuis C:\xampp\htdocs\aa_Toolbox\test_sim800c\claude\v1\)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Initialiser la base de données (première fois)
C:\xampp\mysql\bin\mysql.exe -u root < scripts\init_db.sql
```

### Identifiants par défaut
- Application web : `admin` / `admin123`
- URL : `http://test-sim800c.lan:8082`

### Bug potentiel à surveiller
Le fichier `internal/api/handlers/ussd.go` utilise encore `h.cfg.USSD.ExploreDelayMs` (ligne 149 et 183) mais ce handler séparé n'est probablement pas utilisé activement (les routes réelles passent par `cmd/main.go`). À vérifier si ce handler est référencé et si `NavDelayMs` doit aussi y être utilisé.
