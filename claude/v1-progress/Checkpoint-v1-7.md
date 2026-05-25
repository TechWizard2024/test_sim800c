# Checkpoint — SIM800C Supervisor v1-7
**Dernière mise à jour :** 24 Mai 2026 — Session 7

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
- Annotation legacy sur `internal/api/handlers/ussd.go`

---

## Ce qui a été fait — Session 7 (cette session)

### FEAT 1 : Migration `detectCarrierFromNumber` vers la DB ✅
**Problème :** `detectCarrierFromNumber()` dans `sim800c.go` utilisait des préfixes hardcodés (07/05/01 CI). La description demande de stocker et gérer le plan de numérotation depuis la DB.

**Solution :**
- Nouveau type `DialPlanEntry` dans `internal/serial/manager.go` (évite import cycle avec `db`)
- Champ `DialPlan []DialPlanEntry` ajouté au `Manager`
- Champ `dialPlan []DialPlanEntry` injecté dans chaque `SIM800C` au moment de `connectModule()`
- `detectCarrierFromNumber(phoneNumber, dialPlan)` prend maintenant le dial plan en paramètre
- Si dial plan non nil → lookup dynamique ; sinon fallback hardcodé CI
- Dans `cmd/main.go` : chargement `dbConn.GetDialPlan()` après init DB, injection dans `serialManager.DialPlan` avant `Start()`
- Log : "Plan de numérotation chargé depuis DB: N entrées"

**Fichiers :** `internal/serial/manager.go` (+DialPlanEntry, +Manager.DialPlan), `internal/serial/sim800c.go` (signature detectCarrierFromNumber mise à jour), `cmd/main.go` (+injection dial plan)

---

### FEAT 2 : Endpoint individuel `/api/modules/{id}/ussd/auto-menu` ✅
**Problème :** Par symétrie avec `/api/modules/{id}/ussd/auto-status` (session 6), il manquait un endpoint auto-menu par module. L'endpoint global `/api/ussd/auto-menu` exécutait sur tous les modules.

**Solution :**
- Nouveau handler `moduleAutoMenuHandler` dans `cmd/main.go`
- Route : `POST /api/modules/{id}/ussd/auto-menu`
- Utilise `reader.GetServiceNCodes(carrier)` (même méthode que l'handler global)
- Appelle `explorer.ExploreMenu(module, code.USSDCode, code.ID)` avec la bonne signature 3 args
- Broadcast WebSocket `auto_menu_progress` avec `new_codes_count` pour chaque code
- Nouveau bouton **🌲 Auto-Menu** dans chaque carte module du Dashboard
- Nouvelle méthode `runModuleAutoMenu()` dans `DashboardManager` qui formate le résultat

**Fichiers :** `cmd/main.go` (+handler, +route), `web/js/dashboard.js` (+bouton, +case, +méthode)

---

### FEAT 3 : CRUD complet dial_plan depuis l'API ✅
**Problème :** Seul `GET /api/dialplan` existait (session 6). Impossible d'ajouter/modifier/supprimer des entrées depuis l'UI.

**Solution :**
- `POST /api/dialplan` → `createDialPlanHandler` : crée une nouvelle entrée
- `PUT /api/dialplan/{id}` → `updateDialPlanHandler` : modifie une entrée existante
- `DELETE /api/dialplan/{id}` → `deleteDialPlanHandler` : soft-delete (`is_active = FALSE`)
- Nouvelles méthodes DB : `CreateDialPlanEntry()`, `UpdateDialPlanEntry()`, `DeleteDialPlanEntry()`
- Validation basique : `country_code`, `operator`, `prefix` requis ; `number_length` défaut = 10

**Fichiers :** `cmd/main.go` (+3 handlers, +3 routes), `internal/db/db.go` (+3 méthodes CRUD)

---

### FEAT 4 : Onglet Paramètres (Settings) avec gestion du plan de numérotation ✅
**Problème :** Aucun onglet Settings dans le frontend. Le `settings.js` existait mais n'était pas chargé ni intégré dans les tabs. La gestion du plan de numérotation n'était accessible que via SQL.

**Solution :**
- Nouvel onglet **⚙️ Paramètres** dans la barre de navigation
- Section "Modules connectés" : liste les modules avec boutons Réinitialiser / Tester / **📥 Export SMS**
- Section "Plan de numérotation" : tableau avec toutes les entrées actives + boutons ✏️ Modifier / 🗑️ Désactiver
- Bouton **➕ Ajouter** : ouvre une modale avec 6 champs (code pays, nom, indicatif, opérateur, préfixe, nb chiffres)
- Modale également accessible en mode édition (✏️) avec pré-remplissage
- Section "Configuration serveur" : affiche port, baud rate, DB host
- Section "Actions système" : Export logs, Vider logs, Sauvegarde DB
- `settings.js` entièrement réécrit avec classe `SettingsManager` intégrant `loadDialPlan()`, `renderDialPlan()`, `showAddDialPlanModal()`, `showEditDialPlanModal()`, `saveDialPlanEntry()`, `deleteDialPlanEntry()`
- Chargement automatique quand on clique sur l'onglet (`showTab('settings')`)
- `<script src="/js/settings.js">` ajouté dans `index.html`

**Fichiers :** `web/index.html` (+tab btn, +tab content, +modal dialplan, +script), `web/js/settings.js` (réécriture complète)

---

### FEAT 5 : Export SMS en CSV ✅
**Problème :** L'export CSV existait pour l'historique USSD mais pas pour les SMS.

**Solution :**
- Nouveau endpoint `GET /api/modules/{id}/sms/export`
- Handler `exportSMSCSVHandler` dans `cmd/main.go`
- CSV avec BOM UTF-8 pour compatibilité Excel
- Colonnes : `ID, Module_ID, Direction, Sender, Receiver, Message, Is_Trash, Received_At`
- Nom horodaté : `sms_module{id}_{YYYYMMDD_HHMMSS}.csv`
- Nouvelle méthode DB : `GetSMSMessages(moduleID, limit)` — avec filtre `is_deleted = FALSE`
- Bouton **📥 Export SMS** sous chaque module dans l'onglet Paramètres (lien `<a>` download direct)

**Fichiers :** `cmd/main.go` (+handler, +route), `internal/db/db.go` (+GetSMSMessages), `web/js/settings.js` (+lien export dans loadModulesConfig)

---

### FEAT 6 : Validation dynamique du numéro de téléphone dans SMS Manager ✅
**Problème :** La validation du numéro dans `sms.js` utilisait une regex statique `^0[157]\d{8}$` hardcodée. La description demande de valider depuis la DB.

**Solution :**
- `SMSManager` charge `/api/dialplan` au démarrage via `loadDialPlan()`
- Nouvelle méthode `validatePhoneNumber(number)` : lookup dynamique dans le dial plan chargé
  - Strip indicatif (+225, 00225, 225) puis compare longueur et préfixe
  - Si dial plan vide : fallback hardcodé CI (regex `^0[157]\d{8}$`)
  - Retourne `{ valid, operator?, message? }`
- Validation appelée avant `sendSMS()` dans le formulaire de composition
- Message d'erreur explicite si numéro invalide

**Fichiers :** `web/js/sms.js` (+dialPlan, +loadDialPlan, +validatePhoneNumber, +validation avant envoi)

---

### FEAT 7 : CSS bouton Auto-Menu + styles settings ✅
- `.btn-auto-menu` : gradient vert (#11998e → #38ef7d)
- `.dialplan-table` : styles th/td + hover
- `.btn-export-sms` : lien stylé comme bouton

**Fichiers :** `web/css/main.css` (+3 blocs de styles)

---

## Fichiers modifiés (session 7)

| Fichier | Modification |
|---------|-------------|
| `cmd/main.go` | +injection dialPlan dans serialManager ; +routes POST/PUT/DELETE /dialplan ; +route /modules/{id}/ussd/auto-menu ; +route /modules/{id}/sms/export ; +moduleAutoMenuHandler() ; +createDialPlanHandler() ; +updateDialPlanHandler() ; +deleteDialPlanHandler() ; +exportSMSCSVHandler() |
| `internal/db/db.go` | +CreateDialPlanEntry() ; +UpdateDialPlanEntry() ; +DeleteDialPlanEntry() ; +GetSMSMessages() |
| `internal/serial/manager.go` | +DialPlanEntry struct ; +Manager.DialPlan field ; +dialPlan field dans SIM800C ; +injection dialPlan dans connectModule() |
| `internal/serial/sim800c.go` | detectCarrierFromNumber() prend dialPlan []DialPlanEntry en 2e arg |
| `web/index.html` | +tab btn ⚙️ Paramètres ; +tab content settings-tab ; +modal dialplan ; +script settings.js ; showTab() +case settings |
| `web/js/settings.js` | Réécriture complète avec SettingsManager, gestion dialplan CRUD, export SMS |
| `web/js/dashboard.js` | +bouton 🌲 Auto-Menu dans cartes modules ; +case 'auto_menu' ; +runModuleAutoMenu() |
| `web/js/sms.js` | +dialPlan field ; +loadDialPlan() ; +validatePhoneNumber() ; +validation avant sendSMS |
| `web/css/main.css` | +.btn-auto-menu ; +.dialplan-table ; +.btn-export-sms |

---

## État actuel du code — Fonctions par rapport au project_desc.txt

| Fonction | Statut | Notes |
|----------|--------|-------|
| F1 — Module Auto-Discovery (COM scan) | ✅ | COM1-99 + /dev/ttyUSB* |
| F1 — PIN auto-unlock | ✅ | Orange=0000, MTN=12345, Moov=0101 |
| F1 — PIN failed distinct | ✅ | ❌ Échec PIN, WS event pin_failed |
| F1 — Carrier detection via DB dial_plan | ✅ *(session 7)* | Dynamique DB + fallback CI |
| F1 — Dashboard temps réel | ✅ | Signal quality + réseau + PIN status |
| F2-1 — SIM Status Manual-Discovery | ✅ | Boutons par code/module avec info-bulle |
| F2-2 — SIM Status Auto-Discovery (global) | ✅ | WS temps réel |
| F2-2 — SIM Status Auto-Discovery (par module) | ✅ | Bouton 🚀 Auto-Status dans chaque carte |
| F3-1 — USSD Menu Manual-Discovery | ✅ | Navigation interactive + countdown 25s |
| F3-2 — USSD Menu Auto-Discovery (global) | ✅ | WS temps réel |
| F3-2 — USSD Menu Auto-Discovery (par module) | ✅ *(session 7)* | Bouton 🌲 Auto-Menu dans chaque carte |
| F4 — USSD Manager (saisie manuelle) | ✅ | Exécution + historique |
| F5-1 — SMS Manager (créer/lire/supprimer) | ✅ | Temps réel WebSocket |
| F5-2 — Corbeille SMS auto (sans "Test") | ✅ | Auto-tri à réception |
| Plan numérotation en DB | ✅ *(sessions 6+7)* | Table dial_plan + CRUD complet API + UI Settings |
| Validation numéro via DB | ✅ *(session 7)* | sms.js charge /api/dialplan au démarrage |
| Export historique USSD CSV | ✅ | BOM UTF-8 pour Excel |
| Export SMS CSV | ✅ *(session 7)* | Par module depuis onglet Paramètres |
| Thème clair/sombre | ✅ | Bouton toggle dans header |
| Onglet Paramètres (Settings) | ✅ *(session 7)* | Modules + Dial Plan + Config + Actions |
| start_app.bat / stop_app.bat | ✅ | Scripts Windows |

---

## Architecture actuelle

```
cmd/main.go
  └── Routes API
       ├── GET  /api/modules                            → liste modules
       ├── POST /api/ussd/auto-status                   → auto-status tous modules
       ├── POST /api/modules/{id}/ussd/auto-status      → auto-status 1 module
       ├── POST /api/ussd/auto-menu                     → auto-menu tous modules
       ├── POST /api/modules/{id}/ussd/auto-menu        → auto-menu 1 module *(NEW)*
       ├── GET  /api/ussd/history                       → historique JSON
       ├── GET  /api/ussd/history/export                → export CSV
       ├── GET  /api/dialplan                           → plan de numérotation
       ├── POST /api/dialplan                           → créer entrée *(NEW)*
       ├── PUT  /api/dialplan/{id}                      → modifier entrée *(NEW)*
       ├── DELETE /api/dialplan/{id}                    → désactiver entrée *(NEW)*
       ├── POST /api/modules/{id}/sms/send              → SMS série réel
       └── GET  /api/modules/{id}/sms/export            → export SMS CSV *(NEW)*

internal/serial/manager.go
  ├── DialPlanEntry struct (évite import cycle avec db) *(NEW)*
  ├── Manager.DialPlan []DialPlanEntry *(NEW)*
  └── SIM800C.dialPlan []DialPlanEntry (injecté à connectModule) *(NEW)*

internal/serial/sim800c.go
  └── detectCarrierFromNumber(number, dialPlan) *(signature mise à jour)*

internal/db/db.go
  ├── CreateDialPlanEntry() *(NEW)*
  ├── UpdateDialPlanEntry() *(NEW)*
  ├── DeleteDialPlanEntry() *(soft-delete, NEW)*
  └── GetSMSMessages(moduleID, limit) *(NEW)*

web/
  ├── index.html — +onglet Settings, +modal dialplan, +script settings.js
  ├── js/settings.js — réécriture complète (SettingsManager + dialplan CRUD)
  ├── js/dashboard.js — +Auto-Menu par module
  ├── js/sms.js — +validation dynamique via /api/dialplan
  └── css/main.css — +btn-auto-menu, +dialplan-table, +btn-export-sms
```

---

## Décisions prises (session 7)

1. **`DialPlanEntry` dans `serial/manager.go` (pas `db`)** : Injecter directement `[]db.DialPlan` dans le package `serial` créerait un import cycle (`serial` → `db` → déjà importé via `cmd`). Un struct miroir minimal dans `serial` évite ce problème et reste cohérent avec le principe de "lean domain struct" Go.

2. **Soft-delete pour `DeleteDialPlanEntry`** : `DELETE /api/dialplan/{id}` met `is_active = FALSE` plutôt que de supprimer la ligne. Cela préserve l'historique (pour audit), permet de réactiver facilement, et évite des FK violations potentielles. L'UI affiche "Désactiver" plutôt que "Supprimer".

3. **`GetSMSMessages` séparé de `GetSMSByModule`** : `GetSMSByModule` existe mais prend `includeTrash bool`. `GetSMSMessages` est spécifique à l'export (avec `LIMIT` et sans trash). Évite de modifier l'existant.

4. **Validation numéro avec fallback** : Si `/api/dialplan` échoue ou renvoie une liste vide, `sms.js` revient à la regex CI hardcodée. Robustesse prioritaire sur la cohérence parfaite.

5. **Lien `<a download>` pour export SMS** : Plus simple qu'un fetch + Blob pour un export statique. Le navigateur gère le téléchargement directement, sans JS supplémentaire.

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI)
- Vérifier que le dial plan chargé depuis DB est bien injecté dans les modules
- Vérifier que `detectCarrierFromNumber` identifie bien "Orange CI" depuis le DB dial plan
- Vérifier le bouton 🌲 Auto-Menu dans le dashboard
- Vérifier l'onglet Paramètres : ajout/modification/suppression d'une entrée dial plan

### 2. Rafraîchissement du dial plan en temps réel dans le serial Manager
Actuellement, si un utilisateur modifie le dial plan via l'UI, les modules déjà connectés gardent l'ancien dial plan (injecté au `connectModule`). Il faudrait soit :
- Un endpoint `POST /api/dialplan/reload` qui recharge le dial plan dans le Manager et tous les modules actifs
- Ou une lecture DB à chaque détection de carrier (plus coûteux mais plus simple)

**Fichiers :** `cmd/main.go` (+endpoint reload), `internal/serial/manager.go` (+ReloadDialPlan() méthode)

---

## Prochaines étapes — Priorité MOYENNE

### 3. Tests unitaires Go
- `GetDialPlan()`, `CreateDialPlanEntry()`, `UpdateDialPlanEntry()`, `DeleteDialPlanEntry()`
- `ValidatePhoneNumber()` (07XXXXXXXX → Orange CI, 09XXXXXXXX → invalide)
- `GetEffectiveID()`, `FormatUSSDResponse()`

### 4. Page Settings — Configuration des délais USSD
Permettre depuis l'UI de modifier `explore_delay_ms` et `nav_delay_ms` (actuellement dans `config.yaml`).

**Fichiers :** `web/js/settings.js` (+section délais), `cmd/main.go` (+endpoint PUT /api/config/delays)

### 5. Whitelist COM ports dans config.yaml
Documenter que `serial.ports: ["COM5", "COM6", "COM7"]` accélère le scan initial.

---

## Prochaines étapes — Priorité BASSE

### 6. Endpoint individuel auto-status/menu via DBID
Les endpoints `/api/modules/{id}/ussd/auto-status` et `/api/modules/{id}/ussd/auto-menu` cherchent par `ModuleID` (in-memory). Si l'app redémarre, les IDs peuvent changer. Chercher d'abord par `DBID`.

### 7. Export SMS depuis SMS Manager (pas seulement Settings)
Ajouter un bouton "📥 Export CSV" dans l'onglet SMS Manager lui-même, filtré sur le module sélectionné.

---

## Notes importantes pour la prochaine session

### Structure fichier v1-7.zip
```
v1-7/
├── cmd/main.go                      ← +injection dialPlan ; +5 nouveaux handlers ; +4 nouvelles routes
├── config.yaml                      ← inchangé
├── internal/
│   ├── api/handlers/ussd.go         ← inchangé (commentaire legacy)
│   ├── config/config.go             ← inchangé
│   ├── db/db.go                     ← +CreateDialPlanEntry, +UpdateDialPlanEntry, +DeleteDialPlanEntry, +GetSMSMessages
│   ├── serial/manager.go            ← +DialPlanEntry, +Manager.DialPlan, +SIM800C.dialPlan, +injection connectModule
│   ├── serial/sim800c.go            ← detectCarrierFromNumber() signature mise à jour
│   └── sms/sms_manager.go           ← inchangé
├── scripts/
│   └── init_db.sql                  ← inchangé (dial_plan était déjà là en v1-6)
└── web/
    ├── css/main.css                 ← +.btn-auto-menu ; +.dialplan-table ; +.btn-export-sms
    ├── index.html                   ← +onglet Settings ; +modal dialplan ; +script settings.js ; showTab(settings)
    └── js/
        ├── dashboard.js             ← +bouton 🌲 Auto-Menu ; +runModuleAutoMenu()
        ├── settings.js              ← réécriture complète (SettingsManager + CRUD dialplan)
        └── sms.js                   ← +loadDialPlan() ; +validatePhoneNumber() ; +validation avant envoi
```

### Commandes utiles
```bat
REM Compiler (depuis C:\xampp\htdocs\aa_Toolbox\test_sim800c\claude\v1-7\)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Vérifier que le dial plan est chargé au démarrage (chercher dans les logs)
REM "Plan de numérotation chargé depuis DB: 3 entrées"
```

### Identifiants par défaut
- Application web : `admin` / `admin123`
- URL : `http://test-sim800c.lan:8082`

### Point d'attention — Reload dial plan
Si vous ajoutez un opérateur dans l'onglet Paramètres → Plan de numérotation, les modules déjà connectés ne verront pas le changement immédiatement. Il faut relancer l'application pour que le nouveau dial plan soit injecté dans les modules au `connectModule`. La priorité HAUTE #2 ci-dessus adresse ce problème.
