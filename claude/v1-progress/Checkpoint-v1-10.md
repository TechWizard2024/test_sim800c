# Checkpoint — SIM800C Supervisor v1-10
**Dernière mise à jour :** 25 Mai 2026 — Session 10

---

## Résumé des sessions précédentes (1-9)

- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes clair/sombre, SMS Manager
- Navigation interactive USSD step-by-step, Favoris USSD, Historique USSD + export CSV
- Signal Quality AT+CSQ / réseau AT+CREG, CRUD dial_plan depuis l'API
- `GetEffectiveID()`, `PINFailed`, délais USSD configurables + persistants en DB
- Broadcasts WebSocket Auto-Status / Auto-Menu / `dialplan_reloaded`
- Section Config avancée (auto_trash_keyword, retry_on_error, max_retries, max_menu_depth)
- `FormatUSSDText()` (substitutions GSM-7, découpage options concaténées)
- Table `app_settings(setting_key, setting_value)` — persistance générique

---

## Ce qui a été fait — Session 10

### FEAT 1 : Amélioration `FormatUSSDText` — préservation `- - -` ✅
**Problème (Priorité HAUTE #2 de v1-9) :** Le séparateur visuel `- - -` qui précède `0:Retour` et `00:Accueil` était supprimé (traité comme ligne vide après trim). Cela dégradait la lisibilité des menus USSD dans le dashboard.

**Solution :**
- Ajout d'une détection des séparateurs `- - -` et `---` avant le split multi-espaces
- Ligne dont le contenu (espaces supprimés) == `"---"` ou `"──────"` → remplacée par `"- - -"` et conservée dans le résultat
- Ajout de substitutions d'encodage supplémentaires (`Ã©`→`é`, `Ã¨`→`è`, `Ã `→`à`, `Ã´`→`ô`, etc.) pour couvrir les cas de double-encodage UTF-8
- Déplacement de la compilation de la regex `multiSpaceRe` hors de la boucle (optimisation)

**Fichiers :** `internal/ussd/executor.go` (refactoring de `FormatUSSDText`)

---

### FEAT 2 : `SetMaxDepth()` sur `USSDExplorer` ✅
**Problème (Priorité MOYENNE #5 de v1-9) :** `PUT /api/config/advanced` avec `max_menu_depth` modifiait `cfg.USSD.MaxMenuDepth` et le persistait en DB, mais l'instance `ussdExplorer` déjà créée conservait sa valeur initiale. Le changement ne prenait effect qu'après redémarrage.

**Solution :**
- Nouvelle méthode `func (e *USSDExplorer) SetMaxDepth(depth int)` dans `internal/ussd/explorer.go`
- `updateAdvancedSettingsHandler` reçoit maintenant `ussdExp *ussd.USSDExplorer` en paramètre
- Quand `MaxMenuDepth` change, `ussdExp.SetMaxDepth(payload.MaxMenuDepth)` est appelé immédiatement
- Route `PUT /api/config/advanced` mise à jour avec la nouvelle signature

**Fichiers :** `internal/ussd/explorer.go` (+SetMaxDepth), `cmd/main.go` (signature handler + appel SetMaxDepth)

---

### FEAT 3 : Broadcast WebSocket `config_updated` ✅
**Problème (Priorité BASSE #6 de v1-9) :** Après `PUT /api/config/advanced`, les autres onglets ouverts ne recevaient pas de notification.

**Solution :**
- `updateAdvancedSettingsHandler` reçoit `hub *websocket.Hub` en paramètre
- Après chaque modification, broadcast d'un événement `config_updated` avec `{ changed[], auto_trash_keyword, retry_on_error, max_retries, max_menu_depth, message }`
- Dans `web/js/app.js`, nouveau `case 'config_updated'` :
  - Affiche une notification toast `⚙️ Configuration mise à jour: <champs changés>`
  - Appelle `window.settingsManager.loadAdvancedSettings()` pour rafraîchir l'UI

**Fichiers :** `cmd/main.go` (signature + broadcast), `web/js/app.js` (+case config_updated)

---

### FEAT 4 : Page de statut système — `/api/system/status` + Dashboard panel ✅
**Problème (Priorité BASSE #7 de v1-9) :** Impossible de voir l'état général de l'application depuis l'UI (uptime, connexion DB, modules, config courante).

**Solution :**
- Nouvelle variable globale `var startupTime = time.Now()` dans `cmd/main.go`
- Nouveau handler `systemStatusHandler` → `GET /api/system/status` retournant :
  - `version`, `startup_time`, `uptime` (format "Xd HHh MMm SSs"), `uptime_seconds`
  - `modules_total`, `modules_pin_ok`, `modules_pin_fail`, `modules[]` (résumé de chaque module)
  - `database.ok`, `database.error`, `database.host`, `database.database`
  - `config.explore_delay_ms`, `config.nav_delay_ms`, `config.max_menu_depth`, `config.auto_trash_kw`
  - `server_time`
- Utilise `dbConn.Ping()` natif (DB embeds `*sql.DB`)
- Panel "🖥️ Statut système" ajouté **en tête** du Dashboard tab dans `web/index.html` avec 6 cartes : Uptime, Modules, DB, Heure serveur, Délai exploration, Profondeur max menu
- Bouton "🔄 Actualiser" pour rafraîchissement manuel
- `loadSystemStatus()` appelé automatiquement au démarrage et toutes les 30s (même cycle que `loadModules`)
- Fonction globale `loadSystemStatus()` exposée pour l'attribut `onclick`
- CSS `.sys-stat-card`, `.sys-stat-label`, `.sys-stat-val` dans `web/css/main.css`

**Fichiers :** `cmd/main.go` (+startupTime, +systemStatusHandler, +route GET /system/status), `web/index.html` (+panel status), `web/js/app.js` (+loadSystemStatus, +periodic call), `web/css/main.css` (+3 règles)

---

### FEAT 5 : Whitelist ports COM dans Settings ✅
**Problème (Priorité MOYENNE #4 de v1-9) :** La liste `cfg.Serial.Ports` (ports prioritaires) n'était modifiable que via `config.yaml` + redémarrage. Pas d'UI, pas de persistance en DB.

**Solution :**
- Nouveaux endpoints :
  - `GET /api/config/ports` → `{ ports: [] }`
  - `PUT /api/config/ports` → body `{ ports_csv: "COM5, COM6" }` ou `{ ports: ["COM5","COM6"] }`
- Persistance via `app_settings` avec la clé `serial.ports_whitelist` (CSV)
- Restauration au démarrage depuis la DB avant `serialManager.Start()`
- Section "🔌 Ports COM prioritaires (Whitelist)" dans l'onglet Paramètres de `web/index.html` :
  - Input texte avec placeholder `"COM5, COM6, /dev/ttyUSB0"`
  - Bouton "💾 Sauvegarder"
  - Message de confirmation auto-masqué après 4s
- `web/js/settings.js` : méthodes `loadPortsWhitelist()` et `savePortsWhitelist()`, listener `save-com-whitelist-btn`, chargée au démarrage de `SettingsManager`

**Note importante :** La whitelist modifie `cfg.Serial.Ports` en mémoire, ce qui prend effet au prochain scan automatique (monitoring cycle) ou redémarrage. L'auto-discovery en cours n'est pas interrompue.

**Fichiers :** `cmd/main.go` (+restauration au startup, +2 handlers, +2 routes), `web/index.html` (+section whitelist), `web/js/settings.js` (+loadPortsWhitelist, +savePortsWhitelist, +listener)

---

## Fichiers modifiés (session 10)

| Fichier | Modification |
|---------|-------------|
| `internal/ussd/executor.go` | Refactoring `FormatUSSDText` : préserve `- - -`, substitutions encodage étendues, regex hors boucle |
| `internal/ussd/explorer.go` | +`SetMaxDepth(depth int)` |
| `cmd/main.go` | +`strings` import ; +restauration whitelist ports au startup ; route /config/advanced PUT mise à jour (ussdExp + hub) ; +`startupTime` var ; +`systemStatusHandler` ; +route /system/status ; +`getPortsWhitelistHandler` ; +`updatePortsWhitelistHandler` ; +routes /config/ports |
| `web/index.html` | +panel statut système dans Dashboard ; +section whitelist ports COM dans Settings |
| `web/js/app.js` | +`loadSystemStatus()` ; +appel au démarrage + periodic ; +global `loadSystemStatus()` ; +`case 'config_updated'` dans WS handler |
| `web/js/settings.js` | +`loadPortsWhitelist()` ; +`savePortsWhitelist()` ; +listener `save-com-whitelist-btn` ; +init call `loadPortsWhitelist()` |
| `web/css/main.css` | +`.sys-stat-card`, `.sys-stat-label`, `.sys-stat-val` |

---

## État actuel — Fonctionnalités implémentées

| Fonctionnalité | État | Notes |
|----------------|------|-------|
| Auto-Discovery modules | ✅ | COM1..COM99 + Linux /dev/ttyUSB* |
| Identification SIM/Carrier | ✅ | CNUM + USSD universel |
| PIN auto-unlock | ✅ | Codes par défaut Orange/MTN/Moov |
| Dashboard temps réel | ✅ | WebSocket |
| Panel statut système | ✅ | NEW session 10 |
| Fonction 2-1: Status Manual (boutons Consulter) | ✅ | |
| Fonction 2-2: Status Auto-Discovery | ✅ | Global + par module |
| Fonction 3-1: USSD Menu Manual-Discovery | ✅ | |
| Fonction 3-2: USSD Menu Auto-Discovery | ✅ | Global + par module |
| Fonction 4: USSD Manager (saisie libre) | ✅ | |
| Fonction 5: SMS Manager | ✅ | Créer, Lire, Supprimer, Export CSV |
| Corbeille SMS automatique | ✅ | Mot-clé configurable + persistant |
| Navigation USSD interactive (step-by-step) | ✅ | Countdown 25s |
| Formatage texte USSD | ✅ | ▒→é, - - - préservé, multi-espaces découpés |
| Signal quality + réseau | ✅ | AT+CSQ, AT+CREG, WebSocket |
| Historique USSD + export CSV | ✅ | |
| Favoris USSD | ✅ | |
| Thème clair/sombre | ✅ | |
| start_app.bat / stop_app.bat | ✅ | |
| Plan de numérotation (DB + CRUD API) | ✅ | Broadcat WS dialplan_reloaded |
| Paramètres avancés (UI + DB persistance) | ✅ | +propagation max_menu_depth en temps réel |
| Broadcast WS config_updated | ✅ | NEW session 10 |
| Whitelist ports COM (UI + DB) | ✅ | NEW session 10 |
| SetMaxDepth dynamique (sans redémarrage) | ✅ | NEW session 10 |
| Délais USSD persistants | ✅ | |

---

## Architecture API (complète v1-10)

```
GET  /api/health
POST /api/login
POST /api/logout
GET  /api/modules
GET  /api/modules/{id}
POST /api/discover
GET  /api/modules/{id}/ussd/status-codes
GET  /api/modules/{id}/ussd/menu-codes
POST /api/modules/{id}/ussd/execute
POST /api/modules/{id}/ussd/navigate
POST /api/modules/{id}/ussd/auto-status
POST /api/modules/{id}/ussd/auto-menu
POST /api/ussd/auto-status
POST /api/ussd/auto-menu
POST /api/ussd/explore/{id}/{code}
GET  /api/modules/{id}/signal
GET  /api/ussd/history
GET  /api/ussd/history/export
GET  /api/ussd/favorites
POST /api/ussd/favorites
DELETE /api/ussd/favorites/{id}
GET  /api/dialplan
POST /api/dialplan
POST /api/dialplan/reload       ← broadcast WS dialplan_reloaded
PUT  /api/dialplan/{id}
DELETE /api/dialplan/{id}
GET  /api/config
PUT  /api/config/delays         ← persistant DB
GET  /api/config/advanced
PUT  /api/config/advanced       ← +SetMaxDepth dynamique +broadcast config_updated (NEW v1-10)
GET  /api/config/ports          ← NEW v1-10
PUT  /api/config/ports          ← NEW v1-10
GET  /api/system/status         ← NEW v1-10
GET  /api/modules/{id}/sms
POST /api/modules/{id}/sms/send
GET  /api/modules/{id}/sms/export
DELETE /api/modules/{id}/sms/{index}
POST /api/sms/trash/{id}
POST /api/sms/read-all
GET  /api/user/profile
POST /api/user/password
GET  /api/audit/logs
POST /api/excel/reload
GET  /api/excel/versions
GET  /api/ws  (WebSocket)
```

---

## Décisions prises (session 10)

1. **Préservation `- - -`** : Détection avant le split multi-espaces via normalisation `stripped = replace(" ", "")`. Fiable car aucune option de menu ne contient uniquement des tirets et espaces.

2. **`SetMaxDepth` sur l'instance existante** : Plus propre qu'un accès direct au champ `maxDepth` (non exporté). La méthode valide aussi `depth > 0` pour éviter les valeurs incohérentes. Pas de mutex nécessaire car `maxDepth` est lu/écrit sur le goroutine principal HTTP handler.

3. **`systemStatusHandler` utilise `dbConn.Ping()`** : `db.DB` embeds `*sql.DB` donc `Ping()` est directement disponible. Pas besoin d'ajouter de méthode wrapper.

4. **Whitelist ports stockée en CSV dans `app_settings`** : Cohérent avec les autres paramètres. Pas besoin d'une table dédiée pour une simple liste de ports.

5. **Panel statut système en haut du Dashboard** : Positionnement intentionnel — l'utilisateur voit l'état global avant les cartes modules, ce qui aide au diagnostic immédiat (DB down, aucun module, etc.).

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI)
- Vérifier que `FormatUSSDText` préserve bien `- - -` dans le menu `#111#`
- Vérifier `0:Retour` et `00:Accueil` sur des lignes séparées après le `- - -`
- Vérifier que `SetMaxDepth` prend effet immédiatement (changer via l'UI → lancer une exploration → confirmer la profondeur)

---

## Prochaines étapes — Priorité MOYENNE

### 2. Tests unitaires Go
- `FormatUSSDText` : cas `- - -`, `▒`, multi-espaces, encodage double UTF-8
- `SetMaxDepth(0)` → doit être ignoré (valeur non modifiée)
- `GetSetting` / `SetSetting` : clé inexistante → erreur non fatale
- `updatePortsWhitelistHandler` : CSV vide → `ports: []`

### 3. Notification WS au redémarrage automatique
Quand le monitoring cycle détecte un nouveau module (via `discoverNewModules`), broadcaster un event `discovery_scan_complete` avec le nombre de modules découverts.

### 4. Export dial plan en CSV
Ajouter `GET /api/dialplan/export` → CSV téléchargeable (symétrique de l'export SMS/USSD).

---

## Prochaines étapes — Priorité BASSE

### 5. Indicateur visuel "Whitelist active"
Dans la section whitelist de Settings, afficher un badge si la liste est non vide : "✅ X port(s) en priorité". Aider l'utilisateur à savoir si le scan sera accéléré ou non.

### 6. Auto-refresh du panel statut système via WebSocket
Plutôt que du polling toutes les 30s, mettre à jour le panel statut quand un event `module_connected`, `module_disconnected`, `signal_update` est reçu (plus réactif, moins de requêtes HTTP).

### 7. Historique des changements de configuration
Logger dans la table `audit_logs` tous les changements de configuration (délais, mot-clé, profondeur, whitelist) pour auditabilité.

---

## Structure fichier v1-10.zip

```
v1-10/
├── cmd/main.go              ← +strings import ; +startupTime ; +systemStatusHandler ;
│                               +getPortsWhitelistHandler ; +updatePortsWhitelistHandler ;
│                               +restauration whitelist startup ; route /config/advanced mise à jour ;
│                               +routes /config/ports, /system/status
├── config.yaml              ← inchangé
├── internal/
│   ├── db/db.go             ← inchangé
│   ├── serial/manager.go    ← inchangé
│   ├── serial/sim800c.go    ← inchangé
│   ├── sms/sms_manager.go   ← inchangé
│   └── ussd/
│       ├── executor.go      ← FormatUSSDText amélioré (- - -, encodages, regex hors boucle)
│       ├── explorer.go      ← +SetMaxDepth()
│       └── validator.go     ← inchangé
├── scripts/
│   └── init_db.sql          ← inchangé
└── web/
    ├── index.html           ← +panel statut système Dashboard ; +section whitelist ports COM
    ├── css/main.css         ← +.sys-stat-card, .sys-stat-label, .sys-stat-val
    └── js/
        ├── app.js           ← +loadSystemStatus() ; +case config_updated ; +global fn
        └── settings.js      ← +loadPortsWhitelist() ; +savePortsWhitelist() ; +listener
```

## Commandes utiles

```bat
REM Compiler (depuis le dossier v1-10)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Vérifier le statut système
curl http://test-sim800c.lan:8082/api/system/status

REM Lire la whitelist ports
curl http://test-sim800c.lan:8082/api/config/ports

REM Mettre à jour la whitelist ports
curl -X PUT http://test-sim800c.lan:8082/api/config/ports \
     -H "Content-Type: application/json" \
     -d "{\"ports_csv\": \"COM5, COM6, COM7\"}"
```

## Points d'attention

- **Whitelist ports COM** : Le changement est appliqué à `cfg.Serial.Ports` en mémoire immédiatement. Le prochain cycle `discoverNewModules` (toutes les N secondes selon `monitoring.check_interval_seconds`) utilisera cette liste. Les modules déjà connectés ne sont pas déconnectés.
- **SetMaxDepth** : Pas de mutex sur `maxDepth` dans `USSDExplorer`. Acceptable car les explorations sont déclenchées séquentiellement par des requêtes HTTP. Si des explorations concurrentes sont ajoutées dans le futur, il faudra ajouter un `sync.RWMutex`.
- **`dbConn.Ping()`** : Utilise la connexion pool existante. Si la DB est momentanément surchargée, le ping peut échouer alors que la DB fonctionne. Ce cas est affiché comme `❌` dans le panel statut — à considérer comme un avertissement, pas nécessairement une panne.
