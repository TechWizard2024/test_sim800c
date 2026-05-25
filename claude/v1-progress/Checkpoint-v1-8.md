# Checkpoint — SIM800C Supervisor v1-8
**Dernière mise à jour :** 24 Mai 2026 — Session 8

---

## Résumé des sessions précédentes

### Sessions 1-3
- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes, SMS Manager
- Navigation interactive USSD step-by-step, Favoris USSD, Historique

### Session 4
- Signal Quality AT+CSQ, réseau AT+CREG
- Bug carrier "Orange" → "Orange CI" corrigé
- WebSocket signal_update

### Session 5
- `GetEffectiveID()` — DBID stable pour FK
- `PINFailed` — État distinct (❌ vs ⏳)
- Deux délais USSD : `explore_delay_ms` et `nav_delay_ms`
- Broadcasts WebSocket Auto-Status / Auto-Menu
- `SendSMSWithModule()` — envoi SMS réel

### Session 6
- Endpoint individuel `/api/modules/{id}/ussd/auto-status`
- Countdown 25s dans l'UI de navigation USSD
- Table `dial_plan` en DB avec données CI pré-insérées
- Export historique USSD en CSV
- Onglet "Historique USSD" dédié dans le frontend

### Session 7
- Migration `detectCarrierFromNumber` → DB dial plan
- Endpoint `/api/modules/{id}/ussd/auto-menu` (par module)
- CRUD complet dial_plan depuis l'API (POST/PUT/DELETE)
- Onglet Paramètres avec gestion du plan de numérotation
- Export SMS en CSV depuis Settings
- Validation dynamique numéro téléphone dans SMS Manager (via DB dial plan)

---

## Ce qui a été fait — Session 8 (cette session)

### FEAT 1 : Endpoint `/api/dialplan/reload` + `ReloadDialPlan()` ✅
**Problème (Priorité HAUTE #2 de v1-7) :** Si l'utilisateur modifiait le plan de numérotation via l'UI, les modules déjà connectés gardaient l'ancien dial plan (injecté au `connectModule`). Il fallait redémarrer l'app.

**Solution :**
- Nouvelle méthode `Manager.ReloadDialPlan(newPlan []DialPlanEntry)` dans `internal/serial/manager.go`
  - Met à jour `m.DialPlan` (pour les futures connexions)
  - Propage immédiatement `module.dialPlan = newPlan` à **tous les modules connectés** (avec `module.mu.Lock()`)
  - Log: "Plan de numérotation rechargé: N entrées propagées à M modules"
- Nouvelle méthode `Manager.GetModuleByDBID(dbID int)` dans `internal/serial/manager.go`
  - Cherche d'abord par `DBID`, puis fallback sur `ModuleID`
  - Remplace tous les `for _, m := range GetAllModules()` dans les handlers
- Nouveau handler `reloadDialPlanHandler()` dans `cmd/main.go`
  - Route: `POST /api/dialplan/reload`
  - Charge depuis DB → convertit en `[]serial.DialPlanEntry` → appelle `sm.ReloadDialPlan()`
  - Retourne `{ success, count, message }`
- Bouton **🔄 Recharger** ajouté dans l'onglet Paramètres → Plan de numérotation
- Méthode `reloadDialPlan()` dans `settings.js` : POST `/api/dialplan/reload` puis rechargement du tableau

**Fichiers :** `internal/serial/manager.go` (+ReloadDialPlan, +GetModuleByDBID), `cmd/main.go` (+handler, +route), `web/js/settings.js` (+reloadDialPlan()), `web/index.html` (+bouton 🔄 Recharger)

---

### FEAT 2 : Endpoint GET/PUT `/api/config` + section délais USSD dans Settings ✅
**Problème (Priorité MOYENNE #4 de v1-7) :** Les délais USSD (`explore_delay_ms`, `nav_delay_ms`) n'étaient modifiables que dans `config.yaml`, nécessitant un redémarrage. L'onglet Settings avait un appel à `/api/config` non implémenté côté backend.

**Solution :**
- Nouveau handler `getConfigHandler(cfg, logger)` dans `cmd/main.go`
  - Route: `GET /api/config`
  - Retourne la config courante sans secrets (pas de JWT secret, pas de mot de passe DB)
  - Sections exposées: server, serial, mysql (host/port/database), ussd (avec délais), sms, monitoring
- Nouveau handler `updateDelaysHandler(cfg, logger)` dans `cmd/main.go`
  - Route: `PUT /api/config/delays`
  - Payload: `{ explore_delay_ms, nav_delay_ms }`
  - Validation: explore ≥ 500ms, nav ≥ 100ms
  - Modifie `cfg.USSD.ExploreDelayMs` et `cfg.USSD.NavDelayMs` **en mémoire** (sans toucher au YAML)
  - Prise en compte immédiate car l'explorateur USSD lit `cfg.USSD.ExploreDelayMs` à chaque exécution
- `settings.js` : méthode `displayDelays(config)` qui pré-remplit les inputs depuis la config
- `settings.js` : méthode `saveDelays()` qui appelle PUT `/api/config/delays`
- `web/index.html` : nouvelle section **⏱️ Délais USSD** dans l'onglet Paramètres
  - Input `explore-delay-input` (min:500, max:30000, step:500)
  - Input `nav-delay-input` (min:100, max:10000, step:100)
  - Bouton **💾 Appliquer les délais** (id: `save-delays-btn`)

**Fichiers :** `cmd/main.go` (+2 handlers, +2 routes), `web/js/settings.js` (+displayDelays, +saveDelays, +listener save-delays-btn), `web/index.html` (+section délais USSD)

---

### FEAT 3 : Export SMS CSV depuis SMS Manager (pas seulement Settings) ✅
**Problème (Priorité BASSE #7 de v1-7) :** Le bouton export SMS n'était accessible que dans l'onglet Paramètres. L'utilisateur devait quitter l'onglet SMS pour exporter.

**Solution :**
- Dans l'onglet **SMS Manager** (`web/index.html`), ajout d'une ligne d'export dans le header "Messages reçus"
  - Select `sms-export-module-select` : choisir le module à exporter
  - Bouton **📥 Exporter CSV** (id: `sms-export-btn`)
- `web/js/sms.js` : méthode `exportSMSCSV()` qui lit le module sélectionné et déclenche le téléchargement via `<a download>`
- `web/js/sms.js` : `loadModules()` peuple maintenant aussi `sms-export-module-select` en plus de `sms-module-select`
- Event listener `sms-export-btn` → `exportSMSCSV()` dans `setupEventListeners()`
- L'export utilise l'endpoint existant `GET /api/modules/{id}/sms/export`

**Fichiers :** `web/index.html` (+header export dans SMS tab), `web/js/sms.js` (+exportSMSCSV, +populate export select, +listener)

---

### FEAT 4 : Lookup modules par DBID dans tous les handlers ✅
**Problème (Priorité BASSE #6 de v1-7) :** Tous les handlers `/api/modules/{id}/...` cherchaient le module par `m.ModuleID == moduleID` (ID in-memory). Si l'app redémarre, les IDs peuvent changer (ex: 1,2,3 → après restart: 2,3,1 selon l'ordre de connexion série).

**Solution :**
- Nouvelle méthode `GetModuleByDBID(dbID int) (*SIM800C, bool)` dans `internal/serial/manager.go`
  - Cherche d'abord dans `module.DBID` (ID persisté en DB)
  - Fallback sur `module.ModuleID` (ID in-memory, pour compatibilité)
- **Tous les handlers** suivants ont été mis à jour pour utiliser `GetModuleByDBID` :
  - `getModuleHandler` (GET /api/modules/{id})
  - `executeUSSDHandler` (POST /api/modules/{id}/ussd/execute)
  - `exploreMenuHandler` (POST /api/ussd/explore/{id}/{code})
  - `sendSMSHandler` (POST /api/modules/{id}/sms/send)
  - `navigateUSSDHandler` (POST /api/modules/{id}/ussd/navigate)
  - `getModuleSignalHandler` (GET /api/modules/{id}/signal)
  - `moduleAutoStatusHandler` (POST /api/modules/{id}/ussd/auto-status)
  - `moduleAutoMenuHandler` (POST /api/modules/{id}/ussd/auto-menu)

**Fichiers :** `internal/serial/manager.go` (+GetModuleByDBID), `cmd/main.go` (8 handlers mis à jour)

---

## Fichiers modifiés (session 8)

| Fichier | Modification |
|---------|-------------|
| `cmd/main.go` | +reloadDialPlanHandler ; +getConfigHandler ; +updateDelaysHandler ; +routes POST/dialplan/reload, GET/config, PUT/config/delays ; 8 handlers refactorisés avec GetModuleByDBID |
| `internal/serial/manager.go` | +ReloadDialPlan() ; +GetModuleByDBID() |
| `web/index.html` | +bouton 🔄 Recharger dans dialplan ; +section ⏱️ Délais USSD ; +export CSV dans SMS tab |
| `web/js/settings.js` | +displayDelays() ; +saveDelays() ; +reloadDialPlan() ; +listeners save-delays-btn, reload-dialplan-btn ; currentConfig field |
| `web/js/sms.js` | +exportSMSCSV() ; +populate sms-export-module-select dans loadModules() ; +listener sms-export-btn |

---

## État actuel — Fonctionnalités implémentées

| Fonctionnalité | État | Notes |
|----------------|------|-------|
| Auto-Discovery modules | ✅ | COM1..COM99 + Linux /dev/ttyUSB* |
| Identification SIM/Carrier | ✅ | CNUM + USSD universel |
| PIN auto-unlock | ✅ | Codes par défaut Orange/MTN/Moov |
| Dashboard temps réel | ✅ | WebSocket |
| Fonction 2-1: Status Manual (boutons Consulter) | ✅ | |
| Fonction 2-2: Status Auto (auto-status tous / par module) | ✅ | |
| Fonction 3-1: USSD Menu Manual | ✅ | |
| Fonction 3-2: USSD Menu Auto (tous / par module) | ✅ | |
| Fonction 4: USSD Manager (saisie manuelle) | ✅ | |
| Fonction 5: SMS Manager (CRUD + corbeille auto) | ✅ | |
| Navigation interactive USSD (step-by-step, countdown 25s) | ✅ | |
| Favoris USSD | ✅ | |
| Historique USSD + Export CSV | ✅ | |
| Export SMS CSV | ✅ | Settings + SMS Manager *(session 8)* |
| Thème clair/sombre | ✅ | |
| Onglet Paramètres (Settings) | ✅ | |
| Plan de numérotation CRUD | ✅ | |
| Reload dial plan temps réel | ✅ | *(session 8)* |
| Configuration délais USSD depuis UI | ✅ | *(session 8)* |
| GET /api/config | ✅ | *(session 8)* |
| Lookup module par DBID (stable après restart) | ✅ | *(session 8)* |
| start_app.bat / stop_app.bat | ✅ | |

---

## Architecture actuelle

```
cmd/main.go
  └── Routes API
       ├── GET  /api/modules                            → liste modules
       ├── GET  /api/modules/{id}                       → module par DBID (fallback ModuleID)
       ├── POST /api/discover                           → déclencher discovery
       ├── POST /api/modules/{id}/ussd/execute          → exécuter code USSD
       ├── POST /api/ussd/auto-status                   → auto-status tous modules
       ├── POST /api/modules/{id}/ussd/auto-status      → auto-status 1 module (DBID)
       ├── POST /api/ussd/auto-menu                     → auto-menu tous modules
       ├── POST /api/modules/{id}/ussd/auto-menu        → auto-menu 1 module (DBID)
       ├── POST /api/modules/{id}/ussd/navigate         → navigation step-by-step
       ├── POST /api/ussd/explore/{id}/{code}           → explorer menu
       ├── GET  /api/modules/{id}/signal                → qualité signal
       ├── GET  /api/ussd/history                       → historique JSON
       ├── GET  /api/ussd/history/export                → export CSV
       ├── GET  /api/ussd/favorites                     → favoris
       ├── POST /api/ussd/favorites                     → ajouter favori
       ├── DELETE /api/ussd/favorites/{id}             → supprimer favori
       ├── GET  /api/dialplan                           → plan de numérotation
       ├── POST /api/dialplan                           → créer entrée
       ├── POST /api/dialplan/reload                    → recharger + propager *(NEW)*
       ├── PUT  /api/dialplan/{id}                      → modifier entrée
       ├── DELETE /api/dialplan/{id}                    → désactiver entrée
       ├── GET  /api/config                             → config sans secrets *(NEW)*
       ├── PUT  /api/config/delays                      → modifier délais USSD *(NEW)*
       ├── GET  /api/modules/{id}/sms                   → SMS d'un module
       ├── POST /api/modules/{id}/sms/send              → envoyer SMS
       ├── GET  /api/modules/{id}/sms/export            → export SMS CSV
       ├── DELETE /api/modules/{id}/sms/{index}        → supprimer SMS
       ├── POST /api/sms/trash/{id}                     → corbeille
       ├── POST /api/sms/read-all                       → lire tous les SMS
       └── GET  /api/ws                                 → WebSocket

internal/serial/manager.go
  ├── DialPlanEntry struct
  ├── Manager.DialPlan []DialPlanEntry
  ├── ReloadDialPlan(newPlan) *(NEW)* — propage aux modules actifs
  ├── GetModuleByDBID(id) *(NEW)* — cherche par DBID puis fallback ModuleID
  └── SIM800C.dialPlan []DialPlanEntry

internal/serial/sim800c.go
  └── detectCarrierFromNumber(number, dialPlan) — lookup DB ou fallback CI

internal/db/db.go
  ├── GetDialPlan()
  ├── CreateDialPlanEntry(), UpdateDialPlanEntry(), DeleteDialPlanEntry()
  └── GetSMSMessages(moduleID, limit)

web/
  ├── index.html — +bouton 🔄 Recharger dialplan ; +section délais USSD ; +export CSV SMS tab
  ├── js/settings.js — +displayDelays ; +saveDelays ; +reloadDialPlan
  └── js/sms.js — +exportSMSCSV ; +populate export select
```

---

## Décisions prises (session 8)

1. **`ReloadDialPlan` avec double verrou** : La méthode verrouille `m.mu` (Manager) pour itérer sur les modules, puis verrouille chaque `module.mu` individuellement pour mettre à jour `dialPlan`. Ce pattern est sûr car `module.mu` n'est jamais tenu pendant qu'on tient `m.mu` dans les autres fonctions (sauf `connectModule` qui est appelé en goroutine séparée). Pas de deadlock possible.

2. **`GetModuleByDBID` avec double recherche** : Recherche d'abord par `DBID` (persistant après restart), puis fallback sur `ModuleID` (in-memory, pour les modules qui n'ont pas encore été sauvegardés en DB). Cette approche garantit la compatibilité ascendante sans migration nécessaire.

3. **`PUT /api/config/delays` modifie `cfg` en mémoire uniquement** : Ne modifie pas `config.yaml` sur disque. Les délais sont réinitialisés au redémarrage. Choix délibéré : la persistance des délais est moins critique que la simplicité d'implémentation (pas besoin de réécrire le YAML). Si besoin, l'utilisateur peut modifier `config.yaml` manuellement.

4. **`GET /api/config` sans secrets** : Le handler `getConfigHandler` ne retourne jamais `jwt_secret`, `encryption_key`, ni `mysql.password`. Ces champs sont volontairement omis même si l'endpoint est derrière l'auth JWT.

5. **Export SMS CSV via `<a download>` dans SMS Manager** : Même pattern que Settings — plus simple qu'un fetch+Blob, le navigateur gère le téléchargement. Le select `sms-export-module-select` est peuplé dans `loadModules()` pour éviter un appel API supplémentaire.

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI)
- Vérifier que `ReloadDialPlan` propage bien le dial plan après ajout depuis l'UI
- Vérifier que `GetModuleByDBID` retourne bien le bon module via DBID après restart
- Vérifier les nouveaux boutons dans Settings (🔄 Recharger, 💾 Appliquer délais)
- Vérifier le bouton 📥 Exporter CSV dans SMS Manager

---

## Prochaines étapes — Priorité MOYENNE

### 2. Persistance des délais USSD en DB
Actuellement `PUT /api/config/delays` modifie la config en mémoire. Si l'app redémarre, les délais reviennent aux valeurs de `config.yaml`. Deux options :
- Stocker les délais dans une table `app_settings(key VARCHAR, value VARCHAR)` en DB
- Ou réécrire `config.yaml` avec `yaml.Marshal`

**Fichiers :** `internal/db/db.go` (+table app_settings, +GetSetting, +SetSetting), `cmd/main.go` (charger depuis DB au démarrage, sauvegarder au PUT)

### 3. Tests unitaires Go
- `GetDialPlan()`, `CreateDialPlanEntry()`, `UpdateDialPlanEntry()`, `DeleteDialPlanEntry()`
- `ReloadDialPlan()` : vérifier que tous les modules reçoivent le nouveau plan
- `GetModuleByDBID()` : tester lookup par DBID et fallback ModuleID
- `ValidatePhoneNumber()` (07XXXXXXXX → Orange CI, 09XXXXXXXX → invalide)

### 4. Whitelist COM ports dans config.yaml
Documenter que `serial.ports: ["COM5", "COM6", "COM7"]` accélère le scan initial (évite COM1..COM99).

---

## Prochaines étapes — Priorité BASSE

### 5. Section "Configuration avancée" dans Settings
- Toggle pour activer/désactiver la whitelist des ports COM
- Champ pour modifier `auto_trash_keyword` (actuellement "Test")
- Toggle pour `retry_on_error`

### 6. Notification WebSocket après reload dial plan
Actuellement, après `POST /api/dialplan/reload`, les autres onglets ouverts (dans d'autres navigateurs) ne savent pas que le dial plan a changé. Ajouter un broadcast `hub.BroadcastEvent(Type: "dialplan_reloaded", ...)` dans `reloadDialPlanHandler`.

---

## Notes importantes pour la prochaine session

### Structure fichier v1-8.zip
```
v1-8/
├── cmd/main.go                     ← +3 nouveaux handlers ; +3 nouvelles routes ; 8 handlers refactorisés GetModuleByDBID
├── config.yaml                     ← inchangé
├── internal/
│   ├── api/handlers/...            ← inchangés
│   ├── config/config.go            ← inchangé
│   ├── db/db.go                    ← inchangé
│   ├── serial/manager.go           ← +ReloadDialPlan() ; +GetModuleByDBID()
│   ├── serial/sim800c.go           ← inchangé
│   └── sms/sms_manager.go          ← inchangé
├── scripts/
│   └── init_db.sql                 ← inchangé
└── web/
    ├── css/main.css                ← inchangé
    ├── index.html                  ← +bouton 🔄 Recharger ; +section délais USSD ; +export CSV SMS tab
    └── js/
        ├── settings.js             ← +displayDelays ; +saveDelays ; +reloadDialPlan ; +listeners
        └── sms.js                  ← +exportSMSCSV ; +loadModules peuple export select ; +listener
```

### Commandes utiles
```bat
REM Compiler (depuis le dossier v1-8)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Vérifier que le dial plan est rechargé en temps réel (logs)
REM "Plan de numérotation rechargé: 3 entrées propagées à 3 modules"
```

### Points d'attention
- **Reload dial plan** : Après ajout/modification dans Paramètres → Plan de numérotation, cliquer sur 🔄 Recharger pour propager aux modules actifs. Sans ce clic, les modules gardent l'ancien plan jusqu'au prochain restart.
- **Délais USSD** : Les valeurs modifiées via l'UI sont perdues au redémarrage (voir Priorité MOYENNE #2 ci-dessus pour la persistance).
- **DBID vs ModuleID** : Les endpoints `/api/modules/{id}/...` acceptent désormais le DBID (depuis la table `modules` en DB). Le fallback sur ModuleID reste actif.
