# Checkpoint — SIM800C Supervisor v1-11
**Dernière mise à jour :** 25 Mai 2026 — Session 11

---

## Résumé des sessions précédentes (1-10)

- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes clair/sombre, SMS Manager
- Navigation interactive USSD step-by-step, Favoris USSD, Historique USSD + export CSV
- Signal Quality AT+CSQ / réseau AT+CREG, CRUD dial_plan depuis l'API
- `GetEffectiveID()`, `PINFailed`, délais USSD configurables + persistants en DB
- Broadcasts WebSocket Auto-Status / Auto-Menu / `dialplan_reloaded`
- Section Config avancée (auto_trash_keyword, retry_on_error, max_retries, max_menu_depth)
- `FormatUSSDText()` (substitutions GSM-7, découpage options concaténées, préservation `- - -`)
- Table `app_settings(setting_key, setting_value)` — persistance générique
- Panel statut système `/api/system/status`, broadcast `config_updated`, `SetMaxDepth` dynamique
- Whitelist ports COM (UI + DB persistance + restauration au démarrage)

---

## Ce qui a été fait — Session 11

### FEAT 1 : Export Dial Plan CSV — `GET /api/dialplan/export` ✅
**Problème (Priorité HAUTE #4 de v1-10) :** Pas de moyen d'exporter le plan de numérotation, contrairement à l'export SMS et USSD history.

**Solution :**
- Nouveau handler `exportDialPlanCSVHandler` dans `cmd/main.go`
- Route `GET /api/dialplan/export` ajoutée
- CSV avec BOM UTF-8 (compatible Excel) : colonnes `ID, Country_Code, Country_Name, Calling_Code, Operator, Prefix, Number_Length, Is_Active`
- Bouton "📥 Exporter CSV" ajouté dans la section Plan de numérotation des Settings, à côté de "🔄 Recharger" et "➕ Ajouter"
- Méthode `exportDialPlanCSV()` dans `SettingsManager` (settings.js) — déclenche un téléchargement direct

**Fichiers :** `cmd/main.go` (+route +handler), `web/index.html` (+bouton), `web/js/settings.js` (+méthode)

---

### FEAT 2 : Broadcast WebSocket `discovery_scan_complete` ✅
**Problème (Priorité MOYENNE #3 de v1-10) :** Quand le cycle de monitoring détectait un nouveau module, aucune notification WebSocket n'était émise sauf `module_connected` (envoyé plus tard, après initialisation du module). Impossible de savoir en temps réel combien de ports ont été scannés.

**Solution :**
- Modification de `discoverNewModules()` dans `internal/serial/manager.go` :
  - Compteur `newFound` incrémenté pour chaque nouveau port détecté
  - Après la boucle de scan : broadcast `discovery_scan_complete` avec `{ modules_total, new_found, ports_scanned }`
- Dans `web/js/app.js`, nouveau `case 'discovery_scan_complete'` :
  - Si `new_found > 0` : toast notification "🔍 Scan terminé: X nouveau(x) module(s) détecté(s)" + refresh modules + refresh system status
  - Si `new_found == 0` : événement silencieux (pas de notification superflue)

**Fichiers :** `internal/serial/manager.go` (+newFound counter +broadcast), `web/js/app.js` (+case discovery_scan_complete)

---

### FEAT 3 : Indicateur visuel "Whitelist active" (badge) ✅
**Problème (Priorité BASSE #5 de v1-10) :** Dans la section whitelist de Settings, aucun indicateur visuel ne montrait si des ports étaient déjà configurés.

**Solution :**
- Badge `<span id="com-whitelist-badge">` ajouté dans le header de la card whitelist (en haut à droite)
- `loadPortsWhitelist()` dans `settings.js` met à jour ce badge après chaque chargement :
  - `count > 0` → "✅ X port(s) en priorité" (vert)
  - `count == 0` → "⚠️ Aucun port en whitelist (scan complet COM1..COM99)" (gris)
- `savePortsWhitelist()` appelle `loadPortsWhitelist()` après sauvegarde pour rafraîchir le badge

**Fichiers :** `web/index.html` (+span#com-whitelist-badge), `web/js/settings.js` (badge update dans loadPortsWhitelist + appel après save)

---

### FEAT 4 : Auto-refresh panel statut système via WebSocket ✅
**Problème (Priorité BASSE #6 de v1-10) :** Le panel statut système se rafraîchissait uniquement par polling toutes les 30s. Lors d'une déconnexion de module, le panel ne se mettait pas à jour immédiatement.

**Solution :**
- `case 'module_update'`, `'module_connected'`, `'module_initialized'` → appel `this.loadSystemStatus()` en plus de `this.loadModules()`
- Nouveau `case 'module_disconnected'` explicitement ajouté :
  - `this.loadModules()` + `this.loadSystemStatus()`
  - Toast notification "⚠️ Module déconnecté: port" (error)
- `case 'discovery_scan_complete'` si `new_found > 0` → aussi `this.loadSystemStatus()`

**Fichiers :** `web/js/app.js` (module_connected/initialized/disconnected → loadSystemStatus)

---

### FEAT 5 : Audit log des changements de configuration ✅
**Problème (Priorité BASSE #7 de v1-10) :** Les modifications de configuration (délais, mot-clé, profondeur, whitelist) n'étaient pas tracées dans la table `audit_log`.

**Solution :**
- `updateDelaysHandler` : appel `dbConn.SaveAuditLog("system", "config_update_delays", ...)` après chaque mise à jour des délais
- `updateAdvancedSettingsHandler` : appel `dbConn.SaveAuditLog("system", "config_update_advanced", ...)` si `len(changed) > 0`
- `updatePortsWhitelistHandler` : appel `dbConn.SaveAuditLog("system", "config_update_ports_whitelist", ...)` après mise à jour

Les logs sont visibles dans l'onglet Audit Logs de l'UI.

**Fichiers :** `cmd/main.go` (+3 appels SaveAuditLog dans les handlers de config)

---

### BUG FIX : ID cohérence modules dans le frontend ✅
**Problème :** Le JSON `/api/modules` retourne `"id": m.ModuleID` (compteur en mémoire séquentiel) ET `"db_id": m.DBID` (ID en base de données). Les appels API `/api/modules/{id}/...` utilisent `GetModuleByDBID(id)` qui cherche d'abord par `DBID`. Or le frontend utilisait `m.id || m.module_id` pour construire les URLs, ce qui pouvait pointer vers le mauvais module si ModuleID ≠ DBID.

**Solution :**
- Remplacement systématique de `m.id||m.module_id` par `m.db_id||m.id||m.module_id` dans `web/index.html`
- Priorité : `db_id` (fiable, persistant en DB) → `id` (ModuleID in-memory) → `module_id` (alias)
- Correction aussi dans `loadStatusButtons()`, `buildManualStatusSection()`, `buildMenuExplorerSection()`, `loadSMS()`, `loadModulesSelectors()`

**Fichiers :** `web/index.html` (14 occurrences remplacées)

---

### AMÉLIORATION : Dashboard — signal visuel + badge PIN ✅
**Problème :** Les cartes modules dans le Dashboard n'affichaient pas visuellement la qualité du signal ni le statut PIN.

**Solution :**
- Refactoring de la fonction `renderModules()` dans `web/index.html` :
  - Barres de signal ASCII selon CSQ : `▁___` (faible/rouge), `▁▃__` (moyen/orange), `▁▃▅_` (bon/vert clair), `▁▃▅▇` (excellent/vert) + valeur RSSI
  - Badge réseau coloré : vert si `registered`/`roaming`, rouge sinon
  - Badge PIN : "🔒 PIN KO" (rouge) si `pin_failed`, "🔓 PIN OK" (vert) si `pin_unlocked`
  - Header card restructuré en flex avec infos signal à droite

**Fichiers :** `web/index.html` (renderModules refactorisé)

---

## Fichiers modifiés (session 11)

| Fichier | Modification |
|---------|-------------|
| `cmd/main.go` | +route `GET /api/dialplan/export` ; +`exportDialPlanCSVHandler` ; +`SaveAuditLog` dans updateDelays, updateAdvanced, updatePorts |
| `internal/serial/manager.go` | `discoverNewModules` : +`newFound` counter ; +broadcast `discovery_scan_complete` |
| `web/index.html` | +bouton Export CSV dialplan ; +badge `com-whitelist-badge` ; Fix 14x `m.id` → `m.db_id||m.id` ; Refactor `renderModules` (signal bars + PIN badge) |
| `web/js/app.js` | +`case 'discovery_scan_complete'` ; +`case 'module_disconnected'` ; +`loadSystemStatus()` sur module events |
| `web/js/settings.js` | `loadPortsWhitelist` +badge update ; `savePortsWhitelist` +badge refresh ; +`exportDialPlanCSV()` |

---

## État actuel — Fonctionnalités implémentées

| Fonctionnalité | État | Notes |
|----------------|------|-------|
| Auto-Discovery modules | ✅ | COM1..COM99 + Linux /dev/ttyUSB* |
| Identification SIM/Carrier | ✅ | CNUM + USSD universel |
| PIN auto-unlock | ✅ | Codes par défaut Orange/MTN/Moov |
| Dashboard temps réel | ✅ | WebSocket |
| Signal visuel (barres + RSSI) | ✅ | NEW session 11 |
| Badge PIN status (OK/KO) | ✅ | NEW session 11 |
| Panel statut système | ✅ | Auto-refresh via WS (session 11) |
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
| Plan de numérotation (DB + CRUD API) | ✅ | Broadcast WS dialplan_reloaded |
| Export Dial Plan CSV | ✅ | NEW session 11 |
| Paramètres avancés (UI + DB persistance) | ✅ | |
| Broadcast WS config_updated | ✅ | |
| Broadcast WS discovery_scan_complete | ✅ | NEW session 11 |
| Whitelist ports COM (UI + DB) | ✅ | +badge indicateur (session 11) |
| SetMaxDepth dynamique (sans redémarrage) | ✅ | |
| Délais USSD persistants | ✅ | |
| Audit log changements config | ✅ | NEW session 11 |
| Fix ID cohérence modules (db_id prioritaire) | ✅ | NEW session 11 |

---

## Architecture API (complète v1-11)

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
GET  /api/dialplan/export       ← NEW v1-11 (CSV téléchargeable)
GET  /api/config
PUT  /api/config/delays         ← persistant DB + audit log (session 11)
GET  /api/config/advanced
PUT  /api/config/advanced       ← +audit log (session 11)
GET  /api/config/ports
PUT  /api/config/ports          ← +audit log (session 11)
GET  /api/system/status
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

## Décisions prises (session 11)

1. **`discovery_scan_complete` seulement si `new_found > 0`** dans le frontend : évite des toasts répétitifs à chaque cycle de monitoring (toutes les N secondes). L'événement WS est toujours émis côté backend pour permettre d'autres abonnés futurs.

2. **`db_id` prioritaire sur `id` (ModuleID)** : `DBID` est l'identifiant stable (persistant en DB). `ModuleID` est un compteur en mémoire qui peut diverger si des modules sont connectés/déconnectés. Utiliser `db_id||id||module_id` garantit la cohérence même si `DBID` est 0 au démarrage (avant sync DB).

3. **Audit log avec `"system"` comme userID** : Pour les changements de configuration système (pas d'utilisateur identifié dans les handlers sans JWT), on utilise `"system"` + l'IP de la requête. Les vrais utilisateurs auront leur JWT dans les futurs handlers protégés.

4. **Badge whitelist CSS inline** : Cohérent avec le style du reste de l'UI qui utilise des spans inline plutôt que des classes CSS dédiées pour les badges contextuels.

5. **Signal bars en ASCII** (`▁▃▅▇`) : Pas de dépendance externe, rendu cohérent sur tous les OS (Windows/Linux/Mac), visible en thème clair et sombre.

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI) — validation complète
- Vérifier que le fix `db_id` résout les erreurs "Module non trouvé" sur les boutons Consulter
- Vérifier que `discovery_scan_complete` s'affiche au démarrage
- Vérifier les barres de signal dans les cartes modules

### 2. Tests unitaires Go (unitaires critiques)
- `FormatUSSDText` : cas `- - -`, `▒`, multi-espaces, encodage double UTF-8
- `discoverNewModules` : vérifier que `newFound` est correct
- `GetModuleByDBID` : test avec DBID=0 (fallback ModuleID)
- `exportDialPlanCSVHandler` : CSV bien formé avec virgules et BOM

---

## Prochaines étapes — Priorité MOYENNE

### 3. SMS : Suppression depuis la corbeille (Delete permanent)
Actuellement on peut déplacer un SMS en corbeille mais pas le supprimer définitivement depuis l'UI. Ajouter un bouton "🗑 Supprimer définitivement" pour les SMS en corbeille.

### 4. Notification WS `sms_auto_trash` 
Quand un SMS est automatiquement placé en corbeille par le mot-clé, broadcaster un event WS `sms_auto_trash` avec `{ module_id, sender, preview }`. Actuellement seul `sms_received` est émis.

### 5. Indicateur "Exploration en cours" dans les cartes modules
Lors d'une exploration USSD Menu Auto-Discovery sur un module, désactiver les boutons de ce module et afficher un spinner. Reprendre l'état normal quand le broadcast `auto_menu_progress status=done` est reçu.

---

## Prochaines étapes — Priorité BASSE

### 6. Pagination de l'historique USSD
L'historique peut devenir très long. Ajouter `?page=N&limit=50` côté API et un navigateur de pages dans l'onglet Historique.

### 7. Recherche/filtre dans l'historique USSD
Filtrer par code USSD ou par résultat (ex: afficher seulement les exécutions qui contiennent "Compte principal").

### 8. Copier résultat USSD dans le presse-papiers
Ajouter un bouton "📋 Copier" à côté du résultat dans le USSD Manager. Utile pour copier rapidement un solde ou numéro affiché.

---

## Structure fichier v1-11.zip

```
v1-11/
├── cmd/main.go              ← +route GET /dialplan/export ; +exportDialPlanCSVHandler ;
│                               +SaveAuditLog dans updateDelays, updateAdvanced, updatePorts
├── config.yaml              ← inchangé
├── internal/
│   ├── db/db.go             ← inchangé
│   ├── serial/manager.go    ← +newFound counter dans discoverNewModules ;
│                               +broadcast discovery_scan_complete
│   ├── serial/sim800c.go    ← inchangé
│   ├── sms/sms_manager.go   ← inchangé
│   └── ussd/
│       ├── executor.go      ← inchangé
│       ├── explorer.go      ← inchangé
│       └── validator.go     ← inchangé
├── scripts/
│   └── init_db.sql          ← inchangé
└── web/
    ├── index.html           ← +bouton Export CSV dialplan ; +badge com-whitelist-badge ;
    │                           Fix 14x m.id → m.db_id||m.id ; Refactor renderModules
    │                           (signal bars + PIN badge)
    ├── css/main.css         ← inchangé
    └── js/
        ├── app.js           ← +case discovery_scan_complete ; +case module_disconnected ;
        │                       +loadSystemStatus() sur module events
        └── settings.js      ← loadPortsWhitelist +badge ; savePortsWhitelist +badge refresh ;
                                +exportDialPlanCSV()
```

## Commandes utiles

```bat
REM Compiler (depuis le dossier v1-11)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Vérifier le statut système
curl http://test-sim800c.lan:8082/api/system/status

REM Exporter le plan de numérotation en CSV
curl -O http://test-sim800c.lan:8082/api/dialplan/export

REM Voir les audit logs de configuration
curl http://test-sim800c.lan:8082/api/audit/logs | grep config_update
```

## Points d'attention

- **Fix db_id** : Si un module est connecté mais pas encore persisté en DB (`DBID == 0`), `m.db_id` sera `0` dans le JSON et le fallback `||m.id||m.module_id` sera utilisé. Ce cas peut se produire pendant les premières secondes après la connexion d'un module, avant que `OnModuleInitialized` ait pu sync le DBID. C'est acceptable — après le premier rafraîchissement (30s ou event WS), le DBID sera disponible.

- **discovery_scan_complete fréquence** : Cet événement est émis à chaque appel de `discoverNewModules()`, qui est appelé dans le cycle de monitoring (toutes les `cfg.Monitoring.CheckIntervalSeconds` secondes). Le frontend ignore silencieusement les broadcasts avec `new_found == 0`, évitant le bruit.

- **Audit log userID "system"** : Les handlers de config n'extraient pas le JWT pour identifier l'utilisateur. Si la traçabilité fine par utilisateur est requise, il faudra extraire le claim `sub` du JWT dans le middleware et l'injecter dans le contexte de la requête.
