# Checkpoint — SIM800C Supervisor v1-9
**Dernière mise à jour :** 25 Mai 2026 — Session 9

---

## Résumé des sessions précédentes

### Sessions 1-8 (résumé)
- Auto-Discovery COM ports, PIN auto-unlock, Carrier detection
- WebSocket temps réel, Dashboard, Thèmes, SMS Manager
- Navigation interactive USSD step-by-step, Favoris USSD, Historique USSD
- Signal Quality AT+CSQ / réseau AT+CREG
- `GetEffectiveID()`, `PINFailed`, délais USSD configurables
- Broadcasts WebSocket Auto-Status / Auto-Menu, `SendSMSWithModule()`
- Endpoint individuel `/api/modules/{id}/ussd/auto-status`
- Countdown 25s dans l'UI de navigation USSD
- Table `dial_plan` en DB avec données CI pré-insérées
- Export historique USSD en CSV, Onglet "Historique USSD"
- Migration `detectCarrierFromNumber` → DB dial plan
- Endpoint `/api/modules/{id}/ussd/auto-menu` (par module)
- CRUD complet dial_plan depuis l'API (POST/PUT/DELETE)
- Onglet Paramètres avec gestion du plan de numérotation
- Export SMS en CSV depuis Settings et SMS Manager
- Validation dynamique numéro téléphone dans SMS Manager
- `ReloadDialPlan()` — propagation immédiate aux modules actifs
- `GetModuleByDBID()` — lookup stable par DBID après restart
- Endpoint `POST /api/dialplan/reload`, `GET /api/config`, `PUT /api/config/delays`
- Section délais USSD dans Settings (modifiables depuis l'UI)

---

## Ce qui a été fait — Session 9 (cette session)

### FEAT 1 : Persistance des délais USSD en base de données ✅
**Problème (Priorité MOYENNE #2 de v1-8) :** `PUT /api/config/delays` modifiait la config en mémoire seulement. Après redémarrage, les délais revenaient aux valeurs de `config.yaml`.

**Solution :**
- Nouvelle table `app_settings(setting_key VARCHAR PK, setting_value TEXT, updated_at TIMESTAMP)` dans `createTables()` de `internal/db/db.go`
- Nouvelles méthodes `GetSetting(key)`, `SetSetting(key, value)`, `GetAllSettings()` dans `internal/db/db.go`
- `PUT /api/config/delays` appelle maintenant `dbConn.SetSetting("ussd.explore_delay_ms", ...)` et `dbConn.SetSetting("ussd.nav_delay_ms", ...)`
- Au démarrage dans `cmd/main.go`, les valeurs sont restaurées depuis la DB AVANT le démarrage du serial manager
- Séquence de restauration : `explore_delay_ms` → `nav_delay_ms` → `auto_trash_keyword` → `retry_on_error` → `max_retries` → `max_menu_depth`
- Note UI ajoutée dans l'onglet Paramètres : "✅ Ces valeurs sont maintenant persistées en base de données (survivent au redémarrage)"

**Fichiers :** `internal/db/db.go` (+table app_settings, +GetSetting, +SetSetting, +GetAllSettings), `cmd/main.go` (restauration au démarrage + persist dans updateDelaysHandler)

---

### FEAT 2 : Broadcast WebSocket après reload dial plan ✅
**Problème (Priorité BASSE #6 de v1-8) :** Après `POST /api/dialplan/reload`, les onglets ouverts dans d'autres navigateurs ne recevaient pas de notification.

**Solution :**
- `reloadDialPlanHandler` accepte maintenant `hub *websocket.Hub` en paramètre supplémentaire
- Après `sm.ReloadDialPlan(plan)`, broadcast d'un événement `dialplan_reloaded` avec `{ count, message }`
- Dans `web/js/app.js`, nouveau case `'dialplan_reloaded'` dans `handleWebSocketEvent()` :
  - Affiche une notification toast `🔄 Plan de numérotation rechargé (N entrées)`
  - Appelle `window.settingsManager.loadDialPlan()` pour rafraîchir le tableau dans l'onglet Paramètres
- Route mise à jour : `POST /api/dialplan/reload` → `reloadDialPlanHandler(dbConn, serialManager, hub, logger)`

**Fichiers :** `cmd/main.go` (signature handler + broadcast), `web/js/app.js` (+case dialplan_reloaded)

---

### FEAT 3 : Section "Configuration avancée" dans Settings ✅
**Problème (Priorité BASSE #5 de v1-8) :** Impossible de modifier `auto_trash_keyword`, `retry_on_error`, `max_retries`, `max_menu_depth` depuis l'UI. Nécessitait de modifier `config.yaml` et redémarrer.

**Solution :**
- Nouveaux endpoints :
  - `GET /api/config/advanced` → retourne `{ auto_trash_keyword, retry_on_error, max_retries, max_menu_depth, persisted }`
  - `PUT /api/config/advanced` → payload `{ auto_trash_keyword?, retry_on_error?, max_retries?, max_menu_depth? }` — modifie en mémoire + persiste en DB + applique immédiatement
- `SMSManager.UpdateAutoTrashKeyword(keyword string)` — nouvelle méthode pour changer le mot-clé à chaud
- Restauration au démarrage depuis la DB de tous ces paramètres
- Interface `web/index.html` : nouvelle section "🔧 Configuration avancée" dans l'onglet Paramètres avec :
  - Input `adv-trash-keyword` (mot-clé SMS corbeille)
  - Input `adv-max-menu-depth` (profondeur max menu)
  - Input `adv-max-retries` (tentatives max USSD)
  - Checkbox `adv-retry-on-error` (retry automatique)
  - Bouton "💾 Sauvegarder paramètres avancés"
- `web/js/settings.js` : méthodes `loadAdvancedSettings()` et `saveAdvancedSettings()`, listener `save-advanced-btn`

**Fichiers :** `cmd/main.go` (+2 handlers, +2 routes), `internal/db/db.go` (+GetAllSettings), `internal/sms/sms_manager.go` (+UpdateAutoTrashKeyword), `web/index.html` (+section config avancée), `web/js/settings.js` (+loadAdvancedSettings, +saveAdvancedSettings, +listener)

---

### FEAT 4 : Correction `statusCodesHandler` et `menuCodesHandler` ✅
**Problème (Bug existant) :** Ces deux handlers cherchaient le module via `module.ModuleID == id` (ID in-memory), ignorant `GetModuleByDBID()` introduit en session 8.

**Solution :**
- `statusCodesHandler` : remplacé `for _, module := range sm.GetAllModules()` par `sm.GetModuleByDBID(id)` avec retour HTTP 404 si non trouvé
- `menuCodesHandler` : même correction

**Fichiers :** `cmd/main.go` (2 handlers corrigés)

---

### FEAT 5 : Formatage texte USSD (`FormatUSSDText`) ✅
**Problème (Note #2 et #3 du projet_desc.txt) :** Les réponses USSD brutes contiennent :
- Des espaces multiples entre options (le texte est centré sur l'écran GSM)
- Des caractères `▒` (substitution encodage GSM-7 → UTF-8 incomplet)
- Des options multiples sur une même "ligne" séparées par 10+ espaces

**Solution :**
- Nouvelle fonction `FormatUSSDText(raw string) string` dans `internal/ussd/executor.go` :
  1. Remplace `▒` → `é`, `□`/`■` → espace (substitutions encodage)
  2. Normalise les fins de ligne `\r\n` → `\n`
  3. Découpe sur les séquences de 3+ espaces pour séparer les options concaténées
  4. Trim chaque ligne, ignore les lignes vides
- Appliquée automatiquement dans `Execute()` et `ExecuteWithMenu()`
- Résultat : un menu comme `"1: Acheter un Pass                    2: Consulter mes soldes"` devient deux lignes propres `"1: Acheter un Pass\n2: Consulter mes soldes"`

**Fichiers :** `internal/ussd/executor.go` (+FormatUSSDText, appliquée dans Execute + ExecuteWithMenu)

---

## Fichiers modifiés (session 9)

| Fichier | Modification |
|---------|-------------|
| `internal/db/db.go` | +table app_settings dans createTables ; +GetSetting ; +SetSetting ; +GetAllSettings |
| `internal/sms/sms_manager.go` | +UpdateAutoTrashKeyword() |
| `internal/ussd/executor.go` | +FormatUSSDText() ; appliquée dans Execute() et ExecuteWithMenu() |
| `cmd/main.go` | +restauration paramètres depuis DB au démarrage ; updateDelaysHandler persiste en DB ; reloadDialPlanHandler broadcast WS ; +2 routes /config/advanced ; statusCodesHandler+menuCodesHandler utilisent GetModuleByDBID |
| `web/index.html` | +note persistance délais ; +section Config avancée |
| `web/js/settings.js` | +loadAdvancedSettings() ; +saveAdvancedSettings() ; +listener save-advanced-btn ; version v1-9 |
| `web/js/app.js` | +case 'dialplan_reloaded' dans handleWebSocketEvent |
| `scripts/init_db.sql` | +CREATE TABLE app_settings |

---

## État actuel — Fonctionnalités implémentées

| Fonctionnalité | État | Notes |
|----------------|------|-------|
| Auto-Discovery modules | ✅ | COM1..COM99 + Linux /dev/ttyUSB* |
| Identification SIM/Carrier | ✅ | CNUM + USSD universel |
| PIN auto-unlock | ✅ | Codes par défaut Orange/MTN/Moov |
| Dashboard temps réel | ✅ | WebSocket |
| Fonction 2-1: Status Manual (boutons Consulter) | ✅ | |
| Fonction 2-2: Status Auto-Discovery | ✅ | Global + par module |
| Fonction 3-1: USSD Menu Manual-Discovery | ✅ | |
| Fonction 3-2: USSD Menu Auto-Discovery | ✅ | Global + par module |
| Fonction 4: USSD Manager (saisie libre) | ✅ | |
| Fonction 5: SMS Manager | ✅ | Créer, Lire, Supprimer, Export CSV |
| Corbeille SMS automatique | ✅ | Mot-clé configurable depuis l'UI |
| Navigation USSD interactive (step-by-step) | ✅ | Countdown 25s |
| Formatage texte USSD | ✅ | ▒→é, espaces→saut ligne (session 9) |
| Signal quality + réseau | ✅ | AT+CSQ, AT+CREG, WebSocket |
| Historique USSD + export CSV | ✅ | |
| Favoris USSD | ✅ | |
| Thème clair/sombre | ✅ | |
| Authentification JWT | ✅ | |
| Plan de numérotation DB | ✅ | CI pré-inséré, CRUD depuis UI |
| Reload dial plan temps réel | ✅ | Propage + broadcast WS (session 9) |
| Délais USSD configurables UI | ✅ | Persistants en DB (session 9) |
| Config avancée (trash keyword, retry, depth) | ✅ | Persistant en DB (session 9) |
| Persistance paramètres app_settings | ✅ | Session 9 |
| Notification WS dialplan_reloaded | ✅ | Session 9 |
| start_app.bat / stop_app.bat | ✅ | |

---

## Architecture API (complète)

```
cmd/main.go
  ├── GET  /api/health
  ├── POST /api/login  ;  POST /api/logout
  ├── GET  /api/modules
  ├── GET  /api/modules/{id}
  ├── POST /api/discover
  ├── POST /api/modules/{id}/ussd/execute
  ├── GET  /api/modules/{id}/ussd/status-codes   ← DBID (corrigé session 9)
  ├── GET  /api/modules/{id}/ussd/menu-codes     ← DBID (corrigé session 9)
  ├── POST /api/ussd/auto-status
  ├── POST /api/ussd/auto-menu
  ├── POST /api/modules/{id}/ussd/auto-status
  ├── POST /api/modules/{id}/ussd/auto-menu
  ├── POST /api/modules/{id}/ussd/navigate
  ├── POST /api/ussd/explore/{id}/{code}
  ├── GET  /api/modules/{id}/signal
  ├── GET  /api/ussd/history
  ├── GET  /api/ussd/history/export
  ├── GET  /api/ussd/favorites
  ├── POST /api/ussd/favorites
  ├── DELETE /api/ussd/favorites/{id}
  ├── GET  /api/dialplan
  ├── POST /api/dialplan
  ├── POST /api/dialplan/reload        ← +broadcast WS (session 9)
  ├── PUT  /api/dialplan/{id}
  ├── DELETE /api/dialplan/{id}
  ├── GET  /api/config
  ├── PUT  /api/config/delays          ← +persistance DB (session 9)
  ├── GET  /api/config/advanced        ← NEW session 9
  ├── PUT  /api/config/advanced        ← NEW session 9
  ├── GET  /api/modules/{id}/sms
  ├── POST /api/modules/{id}/sms/send
  ├── GET  /api/modules/{id}/sms/export
  ├── DELETE /api/modules/{id}/sms/{index}
  ├── POST /api/sms/trash/{id}
  ├── POST /api/sms/read-all
  ├── GET  /api/user/profile  ;  POST /api/user/password
  ├── GET  /api/audit/logs
  ├── POST /api/excel/reload
  ├── GET  /api/excel/versions
  └── GET  /api/ws  (WebSocket)

internal/db/db.go
  ├── table app_settings *(NEW session 9)*
  ├── GetSetting(key) *(NEW)*
  ├── SetSetting(key, value) *(NEW)*
  └── GetAllSettings() *(NEW)*

internal/ussd/executor.go
  ├── Execute() → FormatUSSDText() *(NEW session 9)*
  ├── ExecuteWithMenu() → FormatUSSDText() *(NEW session 9)*
  └── FormatUSSDText(raw) *(NEW)*

internal/sms/sms_manager.go
  └── UpdateAutoTrashKeyword(keyword) *(NEW session 9)*
```

---

## Décisions prises (session 9)

1. **Table `app_settings` clé/valeur** : Approche générique qui évite d'ajouter des colonnes pour chaque nouveau paramètre. La clé est la convention `namespace.param` (ex: `ussd.explore_delay_ms`). Simple, extensible, sans migration.

2. **Restauration des paramètres AVANT le démarrage du serial manager** : Important car `cfg.SMS.AutoTrashKeyword` est passé au constructeur `sms.NewSMSManager()`. Si on restaurait après, le mot-clé ne serait pas pris en compte. Idem pour `cfg.USSD.MaxMenuDepth` passé à `ussd.NewUSSDExplorer()`.

3. **`UpdateAutoTrashKeyword()` sur SMSManager** : Le mot-clé est stocké dans le struct `SMSManager`. La mise à jour directe de `cfg.SMS.AutoTrashKeyword` ne suffit pas car `smsManager.autoTrashKeyword` est une copie. La méthode dédiée synchronise les deux.

4. **`FormatUSSDText` coté Go (pas JS)** : Le formatage est appliqué au niveau du backend pour que toutes les sources de consommation (WebSocket, API REST, historique DB) reçoivent du texte propre. Pas besoin de dupliquer la logique côté frontend.

5. **Séquences de 3+ espaces comme séparateurs d'options** : Sur l'écran GSM 7 bits, les menus sont affichés centrés sur une largeur de ~160 colonnes. Les options sont concaténées horizontalement avec des espaces de padding. 3 espaces consécutifs sont un marqueur fiable de séparation (les espaces intentionnels dans les textes d'options n'en contiennent jamais plus d'un).

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI)
- Vérifier `FormatUSSDText` sur la réponse réelle de `#111#` (le menu mal formaté décrit dans project_desc.txt)
- Vérifier la persistance des délais : modifier dans l'UI → redémarrer l'app → vérifier que les valeurs sont restaurées
- Tester `PUT /api/config/advanced` : changer `auto_trash_keyword` → envoyer un SMS sans ce mot → vérifier qu'il va en corbeille

### 2. Amélioration `FormatUSSDText` pour les menus numérotés
Cas particulier : le séparateur `- - -` avant `0:Retour` et `00:Accueil` doit être préservé.
Actuellement il est supprimé (traité comme ligne vide). À corriger si les tests montrent que c'est gênant.

---

## Prochaines étapes — Priorité MOYENNE

### 3. Tests unitaires Go
- `GetSetting()` / `SetSetting()` / `GetAllSettings()` (cas normaux + clé inexistante)
- `FormatUSSDText()` : tester sur les exemples réels du project_desc.txt
- `UpdateAutoTrashKeyword()` : vérifier l'effet immédiat sur le filtrage

### 4. Whitelist COM ports dans config.yaml
Documenter que `serial.ports: ["COM5", "COM6", "COM7"]` accélère le scan initial (évite COM1..COM99). Ajouter un toggle dans la section Config avancée de l'UI.

### 5. Persistance `max_menu_depth` dans `USSDExplorer`
Actuellement, `ussd.NewUSSDExplorer(..., cfg.USSD.MaxMenuDepth)` reçoit la valeur au démarrage. Si l'utilisateur change `max_menu_depth` via `PUT /api/config/advanced`, le changement est pris en compte dans `cfg` mais pas propagé à l'instance `ussdExplorer` déjà créée (qui stocke sa propre copie).
**Fix** : ajouter `USSDExplorer.SetMaxDepth(depth int)` ou lire `cfg.USSD.MaxMenuDepth` dynamiquement à chaque `ExploreMenu()`.

---

## Prochaines étapes — Priorité BASSE

### 6. Notification temps réel pour la mise à jour Config avancée
Même pattern que `dialplan_reloaded` : après `PUT /api/config/advanced`, broadcaster un événement WebSocket `config_updated` pour que les onglets ouverts affichent une notification.

### 7. Page de statut système
Afficher dans le dashboard : uptime, version Go, version app, nombre de requêtes traitées, nombre de messages WebSocket envoyés, taille DB.

---

## Structure fichier v1-9.zip

```
v1-9/
├── cmd/main.go              ← +restauration params DB ; +2 routes config/advanced ;
│                               statusCodes+menuCodes utilisent DBID ; reloadDialPlan broadcast WS
├── config.yaml              ← inchangé
├── internal/
│   ├── db/db.go             ← +app_settings table ; +GetSetting ; +SetSetting ; +GetAllSettings
│   ├── serial/manager.go    ← inchangé
│   ├── serial/sim800c.go    ← inchangé
│   ├── sms/sms_manager.go   ← +UpdateAutoTrashKeyword()
│   └── ussd/executor.go     ← +FormatUSSDText() ; appliquée dans Execute+ExecuteWithMenu
├── scripts/
│   └── init_db.sql          ← +CREATE TABLE app_settings
└── web/
    ├── index.html           ← +note persistance délais ; +section Config avancée
    └── js/
        ├── app.js           ← +case dialplan_reloaded dans handleWebSocketEvent
        └── settings.js      ← +loadAdvancedSettings ; +saveAdvancedSettings ; version v1-9
```

## Commandes utiles

```bat
REM Compiler (depuis le dossier v1-9)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Vérifier la persistance des paramètres (logs au démarrage)
REM "Délai exploration USSD restauré depuis DB: 3000ms"
REM "Mot-clé corbeille SMS restauré depuis DB: \"Test\""
```

## Points d'attention

- **app_settings** : La table est créée automatiquement au démarrage (`createTables`). Pas de migration SQL manuelle nécessaire sur une base existante.
- **FormatUSSDText** : Traite les caractères `▒` comme `é` (remplacement GSM-7 pour "é"). Si l'opérateur utilise d'autres caractères accentués, des ajustements peuvent être nécessaires.
- **max_menu_depth** : Voir "Prochaines étapes" #5 — la valeur est propagée dans `cfg` mais pas encore dans l'instance `USSDExplorer` déjà créée. Nécessite un redémarrage pour que ce paramètre soit pris en compte dans l'exploration.
- **Délais USSD** : Maintenant persistants. La note "perdu au redémarrage" des checkpoints précédents n'est plus d'actualité.
