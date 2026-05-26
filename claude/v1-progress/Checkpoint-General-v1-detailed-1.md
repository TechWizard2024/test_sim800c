# Checkpoint Général — SIM800C Supervisor
**Date :** 25 Mai 2026 — Révision post-session 12 (blocs réduits)
**Version actuelle :** v1-12
**Auteur :** Analyse automatique complète

---

## 1. BILAN GÉNÉRAL — ARCHITECTURE DU PROJET

### Structure des fichiers (v1-12)
```
v1-12/
├── cmd/main.go                        ← Serveur HTTP, routes API, handlers
├── config.yaml                        ← Configuration globale
├── go.mod / go.sum                    ← Dépendances Go
├── start_app.bat                      ← Script démarrage
├── stop_app.bat                       ← Script arrêt
├── .env                               ← Variables d'environnement
├── DEPLOYMENT_GUIDE.md                ← Guide déploiement
├── internal/
│   ├── api/handlers/
│   │   ├── module.go                  ← Handlers modules
│   │   ├── sms.go                     ← Handlers SMS
│   │   ├── ussd.go                    ← Handlers USSD
│   │   └── websocket.go               ← Handler WebSocket
│   ├── auth/auth.go                   ← JWT + authentification
│   ├── config/config.go               ← Chargement config YAML
│   ├── db/db.go                       ← Couche base de données MySQL
│   ├── excel/
│   │   ├── cache.go                   ← Cache fichier Excel
│   │   ├── reader.go                  ← Lecture Codes_USSD_CI.xlsx
│   │   └── writer.go                  ← Création nouvelles versions Excel
│   ├── serial/
│   │   ├── manager.go                 ← Gestionnaire ports série (auto-discovery)
│   │   └── sim800c.go                 ← Communication AT commands SIM800C
│   ├── sms/sms_manager.go             ← Gestion SMS (envoyer, lire, corbeille)
│   ├── ussd/
│   │   ├── executor.go                ← Exécution USSD + formatage
│   │   ├── explorer.go                ← Exploration menus USSD
│   │   └── validator.go               ← Validation entrées USSD
│   └── websocket/hub.go               ← Hub WebSocket temps réel
├── scripts/
│   ├── init_db.sql                    ← Script initialisation MySQL
│   ├── deploy.ps1                     ← Script déploiement PowerShell
│   ├── install_service.bat            ← Installation service Windows
│   └── test_setup.ps1                 ← Tests setup
├── storage/
│   ├── excel/Codes_USSD_CI.xlsx       ← Fichier codes USSD CI
│   └── logs/                          ← Logs application
└── web/
    ├── index.html                     ← Application frontend SPA
    ├── css/
    │   ├── main.css                   ← Styles principaux
    │   └── theme-dark.css             ← Thème sombre
    └── js/
        ├── app.js                     ← App principale + WS handler
        ├── dashboard.js               ← Rendu cartes modules
        ├── history.js                 ← Historique USSD + pagination
        ├── settings.js                ← Paramètres + dial plan
        ├── sms.js                     ← Gestionnaire SMS
        ├── theme.js                   ← Basculement thème
        ├── ussd.js                    ← USSD Manager + navigation
        └── websocket.js               ← Client WebSocket
```

---

## 2. BILAN COMPLET DES FONCTIONNALITÉS

### LÉGENDE
- ✅ Implémenté et fonctionnel
- ⚠️ Implémenté partiellement / avec limitations
- ❌ Non implémenté
- 🔧 Bug connu / à corriger

---

### FONCTION 1 — Module Auto-Discovery

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 1.1 | Scan COM1-COM99 + /dev/ttyUSB* | ✅ | `serial/manager.go` |
| 1.1a | Identification USB-SERIAL CH340 via AT/ATI | ✅ | Réponse `SIM800 R14.18` |
| 1.1b | Support n'importe quel nombre de modules | ✅ | Dynamique |
| 1.1c | Whitelist ports COM (UI + DB + restauration) | ✅ | `app_settings` |
| 1.2 | Collecte infos SIM (IMEI, numéro, opérateur) | ✅ | CNUM + USSD universel |
| 1.2a | Identification opérateur via plan de numérotation | ✅ | DB `dial_plan` |
| 1.2b | PIN auto-unlock (Orange=0000, MTN=12345, Moov=0101) | ✅ | `checkAndUnlockPIN()` |
| 1.2c | Gestion SIM PIN requis avant USSD | ✅ | Détecté + déverrouillé |
| 1.3 | Dashboard temps réel WebSocket | ✅ | `module_update` WS |
| 1.3a | Cartes par module (IMEI, numéro, opérateur, signal) | ✅ | `dashboard.js` |
| 1.3b | Barres signal ASCII + RSSI | ✅ | `getSignalIcon()` |
| 1.3c | Badge PIN OK/KO/En attente | ✅ | `dashboard.js` |
| 1.3d | Badge "⏳ Exploration en cours" | ✅ | Session 12 |
| 1.3e | Panel statut système (modules, RAM, uptime) | ✅ | `/api/system/status` |
| 1.3f | Badge whitelist active | ✅ | |
| 1.3g | Broadcast `discovery_scan_complete` | ✅ | |
| **1.X** | **Graphique signal dans le temps (sparkline)** | **❌** | Table `signal_log` absente |

---

### FONCTION 2-1 — SIM Status Manual-Discovery

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 2.1.1 | Boutons USSD par opérateur (Action=Consulter, Target=Interne, Scope=In) | ✅ | `buildManualStatusSection()` |
| 2.1.2 | Info-bulles sur chaque bouton | ✅ | `title` HTML |
| 2.1.3 | Exécution USSD au clic + résultat temps réel | ✅ | WebSocket `ussd_result` |
| 2.1.4 | Formatage texte résultat USSD (GSM-7, options) | ✅ | `FormatUSSDText()` |
| **2.1.X** | **Bouton copier résultat dans section manual** | **⚠️** | Disponible dans USSD Manager, pas dans la section 2-1 |

---

### FONCTION 2-2 — SIM Status Auto-Discovery

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 2.2.1 | Bouton "SIM Status Auto-Discovery" global | ✅ | `POST /api/ussd/auto-status` |
| 2.2.1a | Bouton "Auto-Status" par module | ✅ | `POST /api/modules/{id}/ussd/auto-status` |
| 2.2.2 | Exécution automatique séquentielle de tous les codes Consulter | ✅ | |
| 2.2.3 | Résultats temps réel via WS (`auto_status_progress`) | ✅ | |

---

### FONCTION 3-1 — USSD Menu Manual-Discovery

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 3.1.1 | Boutons USSD par opérateur (Action=Services_N1, Scope=In) | ✅ | `buildMenuExplorerSection()` |
| 3.1.2 | Info-bulles sur chaque bouton | ✅ | |
| 3.1.3 | Exploration récursive jusqu'à la fin | ✅ | `ExploreMenu()` |
| 3.1.4 | Affichage résultat / sous-menus temps réel | ✅ | `auto_menu_progress` WS |
| 3.1.5 | Mise à jour Excel si nouvelle option trouvée | ✅ | `excel/writer.go` |
| 3.1.5a | Création nouvelle version `Codes_USSD_CI-vDATE.xlsx` | ✅ | |
| **3.1.X** | **Navigation pas-à-pas dans menu (step-by-step interactif)** | **⚠️** | Disponible dans USSD Manager (section 4), pas dans section 3-1 dédiée |

---

### FONCTION 3-2 — USSD Menu Auto-Discovery

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 3.2.1 | Bouton "USSD Menu Auto-Discovery" global | ✅ | `POST /api/ussd/auto-menu` |
| 3.2.1a | Bouton "Auto-Menu" par module | ✅ | `POST /api/modules/{id}/ussd/auto-menu` |
| 3.2.2 | Exploration automatique tous codes Services_N1 | ✅ | |
| 3.2.3 | Exploration récursive sous-menus | ✅ | `maxDepth` configurable |
| 3.2.4 | Résultats temps réel via WS | ✅ | `auto_menu_progress` |
| 3.2.5 | Mise à jour Excel nouvelles options | ✅ | |

---

### FONCTION 4 — USSD Manager

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 4.1 | Saisie manuelle code USSD libre | ✅ | `ussd.js` |
| 4.2 | Sélection module cible | ✅ | |
| 4.3 | Exécution + résultat temps réel | ✅ | |
| 4.4 | Navigation interactive step-by-step (countdown 25s) | ✅ | `navigateChoice()` |
| 4.5 | Bouton 📋 Copier résultat | ✅ | Session 12 |
| 4.6 | Validation entrées (PIN 4 chiffres, numéro 10 chiffres, etc.) | ✅ | `validator.go` |
| 4.7 | Favoris USSD (ajouter, lister, supprimer) | ✅ | `ussd_favorites` |
| **4.X** | **Historique rapide (5 derniers codes = raccourcis cliquables)** | **❌** | Non implémenté |

---

### FONCTION 5 — SMS Manager

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| 5.1 | Lire SMS par module | ✅ | `GET /api/modules/{id}/sms` |
| 5.2 | Envoyer SMS | ✅ | `POST /api/modules/{id}/sms/send` |
| 5.3 | Supprimer SMS (soft delete) | ✅ | `DELETE /api/modules/{id}/sms/{index}` |
| 5.4 | Corbeille automatique (mot-clé configurable) | ✅ | `auto_trash_keyword` |
| 5.5 | Restaurer SMS depuis corbeille | ✅ | Session 12 |
| 5.6 | Supprimer définitivement depuis corbeille | ✅ | Session 12 |
| 5.7 | Notification WS `sms_received` | ✅ | Toast informatif |
| 5.8 | Notification WS `sms_auto_trash` | ✅ | Session 12 |
| 5.9 | Export SMS CSV par module | ✅ | `GET /api/modules/{id}/sms/export` |
| 5.10 | Surveillance SMS en temps réel (polling AT) | ✅ | `StartMonitoring()` |
| **5.X** | **SMS marquer comme lu/non-lu** | **❌** | Champ `is_read` absent en DB |
| **5.Y** | **Badge compteur SMS non-lus sur onglet** | **❌** | Dépend de `is_read` |
| **5.Z** | **Notification sonore nouveau SMS** | **❌** | Non implémenté |
| **5.W** | **Export SMS tous modules (module_id=all)** | **❌** | Route uniquement par module |

---

### FONCTIONNALITÉS TRANSVERSALES

| # | Fonctionnalité | Statut | Notes |
|---|----------------|--------|-------|
| T.1 | Thème clair/sombre + bouton bascule | ✅ | `theme.js` + `theme-dark.css` |
| T.2 | start_app.bat (MySQL → DB init → scan → Go) | ✅ | 4 étapes |
| T.3 | stop_app.bat | ✅ | taskkill + nettoyage PID |
| T.4 | Plan de numérotation CRUD en DB | ✅ | Table `dial_plan` |
| T.5 | Validation numéros via dial_plan DB | ✅ | `ValidatePhoneNumber()` |
| T.6 | Multi-pays plan de numérotation | ✅ | `country_code`, `calling_code` |
| T.7 | Export dial plan CSV | ✅ | `GET /api/dialplan/export` |
| T.8 | Authentification JWT | ✅ | `auth.go` |
| T.9 | Audit logs (config, USSD, SMS) | ✅ | Table `audit_log` |
| T.10 | Config avancée (delays, depth, keyword, retry) | ✅ | Persistant DB `app_settings` |
| T.11 | Historique USSD + pagination 50/page | ✅ | Session 12 |
| T.12 | Historique USSD filtre statut + recherche texte | ✅ | Session 12 |
| T.13 | Historique USSD export CSV | ✅ | |
| T.14 | Historique USSD bouton 📋 Copier | ✅ | Session 12 |
| T.15 | Versions Excel (liste + rechargement) | ✅ | `GET /api/excel/versions` |
| T.16 | Signal quality + réseau (AT+CSQ, AT+CREG) | ✅ | Rafraîchissement WS |
| T.17 | Délais USSD configurables + persistants DB | ✅ | `explore_delay_ms`, `nav_delay_ms` |
| **T.X** | **Historique USSD tous modules (module_id=0)** | **⚠️** | API filtre uniquement par module_id, pas "tous" |
| **T.Y** | **Audit logs pagination + filtre** | **❌** | Limite 100 sans filtre |
| **T.Z** | **Graphique signal dans le temps** | **❌** | Table `signal_log` absente |
| **T.W** | **Historique rapide USSD Manager** | **❌** | 5 derniers codes raccourcis |

---

### ROUTES API — ÉTAT COMPLET

```
GET  /api/modules                       ✅
GET  /api/modules/{id}                  ✅
POST /api/modules/{id}/ussd/execute     ✅
POST /api/modules/{id}/ussd/navigate    ✅
POST /api/modules/{id}/ussd/auto-status ✅
POST /api/modules/{id}/ussd/auto-menu   ✅
POST /api/ussd/auto-status              ✅
POST /api/ussd/auto-menu                ✅
POST /api/ussd/explore/{id}/{code}      ✅
GET  /api/modules/{id}/signal           ✅
GET  /api/ussd/history                  ⚠️ (filtre module_id obligatoire)
GET  /api/ussd/history/export           ✅
GET  /api/ussd/favorites                ✅
POST /api/ussd/favorites                ✅
DELETE /api/ussd/favorites/{id}         ✅
GET  /api/dialplan                      ✅
POST /api/dialplan                      ✅
POST /api/dialplan/reload               ✅
PUT  /api/dialplan/{id}                 ✅
DELETE /api/dialplan/{id}               ✅
GET  /api/dialplan/export               ✅
GET  /api/config                        ✅
PUT  /api/config/delays                 ✅
GET  /api/config/advanced               ✅
PUT  /api/config/advanced               ✅
GET  /api/config/ports                  ✅
PUT  /api/config/ports                  ✅
GET  /api/system/status                 ✅
GET  /api/modules/{id}/sms              ✅
POST /api/modules/{id}/sms/send         ✅
GET  /api/modules/{id}/sms/export       ✅
DELETE /api/modules/{id}/sms/{index}    ✅
POST /api/sms/trash/{id}                ✅
POST /api/sms/restore/{id}              ✅ (v1-12)
DELETE /api/sms/delete-permanent/{id}   ✅ (v1-12)
POST /api/sms/read-all                  ✅
GET  /api/user/profile                  ✅
POST /api/user/password                 ✅
GET  /api/audit/logs                    ✅
POST /api/excel/reload                  ✅
GET  /api/excel/versions                ✅
GET  /api/ws (WebSocket)               ✅
GET  /api/ussd/history?module_id=0      ❌ (tous modules — non supporté)
GET  /api/sms/export?module_id=all      ❌ (export SMS tous modules)
GET  /api/signal/history/{id}           ❌ (historique signal — non implémenté)
```

---

### WEBSOCKET EVENTS — ÉTAT COMPLET

| Event | Direction | Statut |
|-------|-----------|--------|
| `module_update` | S→C | ✅ |
| `module_connected` | S→C | ✅ |
| `module_initialized` | S→C | ✅ |
| `module_disconnected` | S→C | ✅ |
| `discovery_scan_complete` | S→C | ✅ |
| `pin_unlocked` | S→C | ✅ |
| `pin_unlock_failed` | S→C | ✅ |
| `auto_status_progress` | S→C | ✅ |
| `auto_menu_progress` | S→C | ✅ |
| `signal_update` | S→C | ✅ |
| `ussd_result` | S→C | ✅ |
| `sms_received` | S→C | ✅ |
| `sms_auto_trash` | S→C | ✅ (v1-12) |
| `sms_moved_to_trash` | S→C | ✅ |
| `sms_restored` | S→C | ✅ (v1-12) |
| `sms_deleted_permanent` | S→C | ✅ (v1-12) |
| `sms_deleted` | S→C | ✅ |
| `config_updated` | S→C | ✅ |
| `dialplan_reloaded` | S→C | ✅ |
| `signal_history` | S→C | ❌ |
| `sms_unread_count` | S→C | ❌ |

---

## 3. LACUNES IDENTIFIÉES PAR RAPPORT À project_desc.txt

| Écart | Priorité | Description |
|-------|----------|-------------|
| **SMS non-lu (is_read)** | HAUTE | Champ `is_read` absent de la table `sms_messages` |
| **Badge non-lus SMS** | HAUTE | Aucun indicateur visuel SMS non lus sur l'onglet |
| **Historique USSD tous modules** | MOYENNE | `GET /api/ussd/history` sans `module_id` retourne rien |
| **Export SMS tous modules** | BASSE | Uniquement par module, pas d'export global |
| **Historique rapide USSD Manager** | BASSE | 5 derniers codes comme raccourcis cliquables |
| **Graphique signal dans le temps** | BASSE | Table `signal_log` manquante, pas de sparkline |
| **Audit logs : pagination + filtre** | BASSE | Limit 100, sans pagination ni filtre |
| **Bouton copier dans section 2-1** | BASSE | Disponible en USSD Manager mais pas dans Status Manual |

---

## 4. PROCHAINES ÉTAPES — ORGANISÉES PAR MICRO-BLOCS DE SESSION

> **Note importante :** Chaque micro-bloc est dimensionné pour être exécuté en **une seule session Claude (version gratuite)**.  
> Chaque micro-bloc ne touche qu'**un seul fichier ou deux fichiers maximum** et produit un Checkpoint + zip.  
> Les micro-blocs sont ordonnés par priorité et dépendances (les blocs avec 🔒 dépendent d'un bloc précédent).

---

### TABLEAU DE BORD DES MICRO-BLOCS

| Bloc | Priorité | Fichiers modifiés | Dépend de | Version cible |
|------|----------|-------------------|-----------|---------------|
| **A1** | 🔴 HAUTE | `internal/db/db.go` | — | v1-13 |
| **A2** | 🔴 HAUTE | `internal/sms/sms_manager.go` + `cmd/main.go` | 🔒 A1 | v1-14 |
| **A3** | 🔴 HAUTE | `web/index.html` + `web/js/sms.js` | 🔒 A2 | v1-15 |
| **A4** | 🔴 HAUTE | `web/js/app.js` | 🔒 A2 | v1-16 |
| **B1** | 🟡 MOYENNE | `internal/db/db.go` + `cmd/main.go` | — | v1-17 |
| **B2** | 🟡 MOYENNE | `web/js/history.js` | 🔒 B1 | v1-18 |
| **B3** | 🟡 MOYENNE | `web/js/ussd.js` | 🔒 B1 | v1-19 |
| **B4** | 🟡 MOYENNE | `web/js/sms.js` | — | v1-20 |
| **C1** | 🟢 BASSE | `internal/db/db.go` + `cmd/main.go` | — | v1-21 |
| **C2** | 🟢 BASSE | `web/js/settings.js` | 🔒 C1 | v1-22 |
| **C3** | 🟢 BASSE | `internal/db/db.go` + `cmd/main.go` | — | v1-23 |
| **C4** | 🟢 BASSE | `web/js/sms.js` | 🔒 C3 | v1-24 |
| **C5** | 🟢 BASSE | `internal/db/db.go` + `internal/serial/sim800c.go` | — | v1-25 |
| **C6** | 🟢 BASSE | `cmd/main.go` + `web/js/dashboard.js` | 🔒 C5 | v1-26 |
| **D1** | 🔵 FIX | `internal/db/db.go` + `cmd/main.go` | — | v1-27 |
| **D2** | 🔵 FIX | `start_app.bat` | — | v1-28 |
| **D3** | 🔵 FIX | `scripts/init_db.sql` + `scripts/migrate_v1-13.sql` | 🔒 A1 | v1-29 |
| **D4** | 🔵 FIX | `config.yaml` + `.env` + `start_app.bat` | — | v1-30 |
| **E1** | 🟣 TESTS | `internal/db/db_test.go` | 🔒 A1, D1 | v1-31 |
| **E2** | 🟣 TESTS | `internal/ussd/validator_test.go` | — | v1-32 |
| **E3** | 🟣 DOC | `README.md` + `DEPLOYMENT_GUIDE.md` | — | v1-33 |

---

### 🔴 MICRO-BLOC A1 — DB : is_read SMS (Session 13)
**Fichiers :** `internal/db/db.go` uniquement  
**Durée estimée :** 1 session

**Tâches :**
- Ajouter dans `createTables()` : `ALTER TABLE sms_messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE`
- Nouvelle méthode `MarkSMSRead(smsID int) error`
- Nouvelle méthode `MarkAllSMSRead(moduleID int) error`
- Modifier `GetSMSByModule()` : inclure le champ `is_read` dans le SELECT et le struct de retour
- Nouvelle méthode `GetUnreadSMSCount(moduleID int) (int, error)`

**Livrables :** `Checkpoint-v1-13.md` + `v1-13.zip`

---

### 🔴 MICRO-BLOC A2 — Backend : routes SMS is_read (Session 14)
**Fichiers :** `internal/sms/sms_manager.go` + `cmd/main.go`  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 A1

**Tâches dans `sms_manager.go` :**
- Ajouter méthode `MarkRead(smsID int) error` (appelle `db.MarkSMSRead`)
- Ajouter méthode `MarkAllRead(moduleID int) error`
- Dans `ReadIncomingSMS()` : après insertion, appeler `db.GetUnreadSMSCount()` et broadcaster `sms_unread_count`

**Tâches dans `cmd/main.go` :**
- Route `POST /api/sms/mark-read/{id}`
- Route `POST /api/modules/{id}/sms/mark-all-read`
- Route `GET /api/modules/{id}/sms/unread-count`

**Livrables :** `Checkpoint-v1-14.md` + `v1-14.zip`

---

### 🔴 MICRO-BLOC A3 — Frontend SMS : style non-lu + boutons (Session 15)
**Fichiers :** `web/index.html` + `web/js/sms.js`  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 A2

**Tâches dans `web/index.html` :**
- Ajouter badge `<span id="sms-unread-badge" class="badge">0</span>` sur le bouton de navigation SMS

**Tâches dans `web/js/sms.js` :**
- Style visuel SMS non-lu : point bleu + texte en gras sur les lignes où `is_read == false`
- Bouton "✓ Lu" par SMS (appelle `POST /api/sms/mark-read/{id}`)
- Bouton "Tout marquer comme lu" dans le header SMS (appelle `POST /api/modules/{id}/sms/mark-all-read`)
- Après chaque action mark-read : recharger la liste SMS + mettre à jour badge

**Livrables :** `Checkpoint-v1-15.md` + `v1-15.zip`

---

### 🔴 MICRO-BLOC A4 — Frontend app.js : badge WS sms_unread_count (Session 16)
**Fichiers :** `web/js/app.js` uniquement  
**Durée estimée :** 1 session (court)  
**Dépend de :** 🔒 A2

**Tâches :**
- Dans le handler WebSocket, ajouter le case `sms_unread_count` :
  - Mettre à jour le texte du badge `#sms-unread-badge`
  - Si count > 0 : afficher le badge, sinon le masquer
- Dans le case `sms_received` : incrémenter le badge de 1
- Au chargement initial de l'app : appeler `GET /api/modules/{id}/sms/unread-count` pour chaque module et cumuler

**Livrables :** `Checkpoint-v1-16.md` + `v1-16.zip`

---

### 🟡 MICRO-BLOC B1 — Backend : historique USSD tous modules + codes récents (Session 17)
**Fichiers :** `internal/db/db.go` + `cmd/main.go`  
**Durée estimée :** 1 session

**Tâches dans `db.go` :**
- Modifier `GetUSSDHistory(moduleID, limit, offset int, ...)` : si `moduleID == 0` → SELECT sans filtre WHERE module_id
- Nouvelle méthode `GetRecentUSSDCodes(moduleID, limit int) ([]string, error)` — retourne les N derniers codes USSD distincts exécutés sur ce module

**Tâches dans `cmd/main.go` :**
- Dans `getUSSDHistoryHandler` : si `module_id` absent ou `0` → appeler sans filtre
- Nouvelle route `GET /api/modules/{id}/ussd/recent` — retourne les 5 derniers codes

**Livrables :** `Checkpoint-v1-17.md` + `v1-17.zip`

---

### 🟡 MICRO-BLOC B2 — Frontend history.js : filtre tous modules (Session 18)
**Fichiers :** `web/js/history.js` uniquement  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 B1

**Tâches :**
- Ajouter dropdown "Filtrer par module" dans la toolbar de l'historique :
  - Option "Tous les modules" (envoie `module_id=0` ou pas de paramètre)
  - Une option par module détecté
- Au chargement initial de la page historique : charger sans filtre module (tous modules)
- Mettre à jour la requête `GET /api/ussd/history` en fonction de la sélection du dropdown

**Livrables :** `Checkpoint-v1-18.md` + `v1-18.zip`

---

### 🟡 MICRO-BLOC B3 — Frontend ussd.js : raccourcis codes récents (Session 19)
**Fichiers :** `web/js/ussd.js` uniquement  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 B1

**Tâches :**
- Après sélection d'un module dans USSD Manager : appeler `GET /api/modules/{id}/ussd/recent`
- Afficher max 5 boutons de raccourcis sous le champ de saisie USSD
- Format bouton : `[#122#]` — au clic, pré-remplir le champ de saisie avec ce code
- Si aucun historique : masquer la section raccourcis

**Livrables :** `Checkpoint-v1-19.md` + `v1-19.zip`

---

### 🟡 MICRO-BLOC B4 — Frontend sms.js : notification sonore (Session 20)
**Fichiers :** `web/js/sms.js` uniquement  
**Durée estimée :** 1 session (court)

**Tâches :**
- Créer une fonction `playSMSBeep()` utilisant `AudioContext` Web API (bip court 440Hz, 150ms)
- Appeler `playSMSBeep()` quand un événement WS `sms_received` arrive
- Ajouter préférence `localStorage.getItem('sms_sound_enabled')` (défaut : `true`)
- Ajouter bouton 🔔/🔕 dans le header du SMS Manager pour activer/désactiver le son

**Livrables :** `Checkpoint-v1-20.md` + `v1-20.zip`

---

### 🟢 MICRO-BLOC C1 — Backend : audit logs pagination (Session 21)
**Fichiers :** `internal/db/db.go` + `cmd/main.go`  
**Durée estimée :** 1 session

**Tâches dans `db.go` :**
- Modifier `GetAuditLogs(limit int)` → `GetAuditLogs(limit, offset int, action, userID string) ([]AuditLog, error)`
- Ajouter `GetAuditLogsCount(action, userID string) (int, error)`

**Tâches dans `cmd/main.go` :**
- `getAuditLogsHandler` : lire `?page=`, `?action=`, `?user=` depuis query string
- Calculer offset = (page-1) * pageSize (défaut 50)
- Réponse JSON : `{logs: [...], total: N, page: P, page_size: 50}`

**Livrables :** `Checkpoint-v1-21.md` + `v1-21.zip`

---

### 🟢 MICRO-BLOC C2 — Frontend settings.js : pagination audit logs (Session 22)
**Fichiers :** `web/js/settings.js` uniquement  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 C1

**Tâches :**
- Dans la section Audit Logs du settings panel :
  - Ajouter pagination (boutons Précédent/Suivant + numéro de page)
  - Ajouter dropdown filtre "Action" (valeurs : `database_initialized`, `ussd_execute`, `sms_send`, etc.)
  - Ajouter filtre "Utilisateur" (champ texte)
  - Logique : rechargement de la liste audit à chaque changement filtre ou page

**Livrables :** `Checkpoint-v1-22.md` + `v1-22.zip`

---

### 🟢 MICRO-BLOC C3 — Backend : export SMS tous modules (Session 23)
**Fichiers :** `internal/db/db.go` + `cmd/main.go`  
**Durée estimée :** 1 session (court)

**Tâches dans `db.go` :**
- Ajouter méthode `GetAllSMS(limit int) ([]SMSMessage, error)` — SELECT sans filtre module_id

**Tâches dans `cmd/main.go` :**
- Nouvelle route `GET /api/sms/export` (sans module_id) — génère CSV de tous les SMS de tous les modules
- Format CSV : mêmes colonnes que l'export par module + colonne `module_id`

**Livrables :** `Checkpoint-v1-23.md` + `v1-23.zip`

---

### 🟢 MICRO-BLOC C4 — Frontend sms.js : bouton export global (Session 24)
**Fichiers :** `web/js/sms.js` uniquement  
**Durée estimée :** 1 session (court)  
**Dépend de :** 🔒 C3

**Tâches :**
- Ajouter bouton "📥 Exporter tous les SMS (CSV)" dans le header global du SMS Manager
- Au clic : appeler `GET /api/sms/export` et déclencher le téléchargement du CSV

**Livrables :** `Checkpoint-v1-24.md` + `v1-24.zip`

---

### 🟢 MICRO-BLOC C5 — Backend : table signal_log + logging (Session 25)
**Fichiers :** `internal/db/db.go` + `internal/serial/sim800c.go`  
**Durée estimée :** 1 session

**Tâches dans `db.go` :**
- Créer table `signal_log(id INT AUTO_INCREMENT, module_id INT, csq INT, rssi FLOAT, network_status VARCHAR(20), logged_at DATETIME)`
- Méthode `LogSignal(moduleID, csq int, rssi float64, networkStatus string) error`
- Méthode `GetSignalHistory(moduleID, limit int) ([]SignalLog, error)`

**Tâches dans `sim800c.go` :**
- Dans `getSignalQuality()` : après chaque mesure réussie, appeler `db.LogSignal()`

**Livrables :** `Checkpoint-v1-25.md` + `v1-25.zip`

---

### 🟢 MICRO-BLOC C6 — Route signal history + sparkline frontend (Session 26)
**Fichiers :** `cmd/main.go` + `web/js/dashboard.js`  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 C5

**Tâches dans `cmd/main.go` :**
- Nouvelle route `GET /api/modules/{id}/signal/history?limit=20`
- Réponse JSON : liste de `{csq, rssi, network_status, logged_at}`

**Tâches dans `dashboard.js` :**
- Dans `renderModuleCard()` : ajouter mini graphique SVG sparkline (20 dernières valeurs CSQ)
- Appel `GET /api/modules/{id}/signal/history?limit=20` au chargement et toutes les 30s
- SVG inline : polyline sur viewBox 80x20, points = valeurs CSQ normalisées 0–31

**Livrables :** `Checkpoint-v1-26.md` + `v1-26.zip`

---

### 🔵 MICRO-BLOC D1 — Fix : historique USSD query module_id (Session 27)
**Fichiers :** `internal/db/db.go` + `cmd/main.go`  
**Durée estimée :** 1 session (court)

**Tâches :**
- Dans `db.go`, corriger `GetUSSDHistory` : si `moduleID == 0` → query sans `WHERE module_id`
  *(Note : ce fix est identique à B1 — si B1 a été fait, ce bloc est déjà résolu. Vérifier avant d'exécuter)*
- Dans `cmd/main.go` handler `getUSSDHistoryHandler` : si param `module_id` absent → passer 0

**Livrables :** `Checkpoint-v1-27.md` + `v1-27.zip`

---

### 🔵 MICRO-BLOC D2 — Fix : start_app.bat robustesse (Session 28)
**Fichiers :** `start_app.bat` uniquement  
**Durée estimée :** 1 session (court)

**Tâches :**
- Vérifier si `sim800c-supervisor.exe` tourne déjà (`tasklist | find "sim800c"`) → message si oui, arrêt ou skip
- Créer le dossier `storage/logs/` s'il n'existe pas (`if not exist ... mkdir`)
- Vérifier si port 8082 est déjà occupé (`netstat -an | find "8082"`) → avertissement
- Après démarrage réussi : ouvrir automatiquement `http://test-sim800c.lan:8082` dans le navigateur par défaut (`start http://test-sim800c.lan:8082`)

**Livrables :** `Checkpoint-v1-28.md` + `v1-28.zip`

---

### 🔵 MICRO-BLOC D3 — Fix : init_db.sql + script migration (Session 29)
**Fichiers :** `scripts/init_db.sql` + nouveau `scripts/migrate_v1-13.sql`  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 A1 (pour connaître la colonne `is_read`)

**Tâches :**
- Auditer `init_db.sql` et `db.go` : synchroniser toutes les tables/colonnes (checklist complète)
- Ajouter dans `init_db.sql` les colonnes manquantes : `is_read` dans `sms_messages`
- Créer `scripts/migrate_v1-13.sql` :
  ```sql
  ALTER TABLE sms_messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE;
  ```
- Créer `scripts/migrate_v1-15.sql` (préparation pour C5) :
  ```sql
  CREATE TABLE IF NOT EXISTS signal_log (...);
  ```

**Livrables :** `Checkpoint-v1-29.md` + `v1-29.zip`

---

### 🔵 MICRO-BLOC D4 — Fix : config.yaml + JWT secret via .env (Session 30)
**Fichiers :** `config.yaml` + `.env` + `internal/config/config.go`  
**Durée estimée :** 1 session (court)

**Tâches :**
- `config.yaml` : remplacer le chemin Excel hardcodé par un chemin relatif (`./storage/excel/`)
- `.env` : ajouter `SIM800C_JWT_SECRET=<valeur>` (retirer la valeur de `config.yaml`)
- `config.go` : lire `SIM800C_JWT_SECRET` depuis `os.Getenv()` en priorité sur `config.yaml`
- `start_app.bat` : copier `storage/excel/Codes_USSD_CI.xlsx` vers le bon chemin si `excel.base_path` est différent

**Livrables :** `Checkpoint-v1-30.md` + `v1-30.zip`

---

### 🟣 MICRO-BLOC E1 — Tests Go : DB SMS + historique USSD (Session 31)
**Fichiers :** `internal/db/db_test.go`  
**Durée estimée :** 1 session  
**Dépend de :** 🔒 A1, D1

**Tâches :**
- Test `TestMarkSMSRead` : insérer SMS, appeler `MarkSMSRead`, vérifier `is_read == true`
- Test `TestGetUnreadSMSCount` : insérer 3 SMS, en marquer 1 comme lu, vérifier count == 2
- Test `TestRestoreSMSFromTrash` : mettre en corbeille, restaurer, vérifier statut
- Test `TestDeleteSMSPermanent` : supprimer définitivement, vérifier absence
- Test `TestGetUSSDHistoryAllModules` : insérer historique pour 2 modules, appeler avec moduleID=0, vérifier les 2 sont retournés
- Test `TestValidatePhoneNumber` : tester 07XXXXXXXX (Orange), 05XXXXXXXX (MTN), 01XXXXXXXX (Moov)

**Livrables :** `Checkpoint-v1-31.md` + `v1-31.zip`

---

### 🟣 MICRO-BLOC E2 — Tests Go : validateur USSD (Session 32)
**Fichiers :** `internal/ussd/validator_test.go`  
**Durée estimée :** 1 session (court)

**Tâches :**
- Test `TestValidatePIN` : "0000" ✅, "123" ❌ (3 chiffres), "abcd" ❌
- Test `TestValidatePhoneNumber` : "0712345678" ✅, "071234567" ❌ (9 chiffres), "0512345678" ✅
- Test `TestValidateMontant` : "50" ✅, "100" ✅, "49" ❌, "5" ❌
- Test `TestValidateReference` : "12345678901234" ✅ (14 chiffres), "1234567890123" ❌
- Test `TestValidateRechargeCode` : "12345678901234" ✅, "1234567890123" ❌
- Test `TestValidateChoice` : valid si dans la liste d'options, invalide sinon

**Livrables :** `Checkpoint-v1-32.md` + `v1-32.zip`

---

### 🟣 MICRO-BLOC E3 — Documentation : README + DEPLOYMENT_GUIDE (Session 33)
**Fichiers :** `README.md` (créer) + `DEPLOYMENT_GUIDE.md` (mettre à jour)  
**Durée estimée :** 1 session

**Tâches `README.md` (nouveau) :**
- Prérequis (Go 1.21+, XAMPP/MySQL, pilote CH340)
- Installation rapide (5 étapes : clone → `go build` → `init_db.sql` → `config.yaml` → `start_app.bat`)
- Accès : `http://test-sim800c.lan:8082`
- Liste des modules SIM800C supportés

**Tâches `DEPLOYMENT_GUIDE.md` (mise à jour) :**
- Ajouter toutes les nouvelles routes API (v1-13 à v1-32)
- Ajouter section scripts de migration DB
- Mettre à jour la liste des variables `.env`

**Livrables :** `Checkpoint-v1-33.md` + `v1-33.zip` (version stable finale)

---

## 5. RÉSUMÉ DES PRIORITÉS

### Fonctionnalités implémentées (✅) : 52 sur 60
### Fonctionnalités partielles (⚠️) : 3
### Fonctionnalités manquantes (❌) : 8 (réparties sur 21 micro-blocs)

### Ordre recommandé d'exécution :
1. **A1 → A2 → A3 → A4** (SMS is_read — impact UX fort — 4 sessions)
2. **B1 → B2 → B3** (Historique global + raccourcis — 3 sessions)
3. **B4** (Son SMS — 1 session, court)
4. **D1** (Fix historique, potentiellement couvert par B1)
5. **D2** (Fix start_app.bat — 1 session, court)
6. **D3** (Fix init_db.sql — 1 session)
7. **D4** (Fix config + JWT — 1 session, court)
8. **C1 → C2** (Audit logs pagination — 2 sessions)
9. **C3 → C4** (Export SMS global — 2 sessions courtes)
10. **C5 → C6** (Sparkline signal — 2 sessions)
11. **E1 → E2 → E3** (Tests + documentation — 3 sessions)

**Total estimé : 21 micro-sessions**

---

## 6. ÉTAT ACTUEL — RÉSUMÉ EXÉCUTIF

Le projet SIM800C Supervisor est **fonctionnel à ~87%** des spécifications.  
Les 5 fonctions principales (Auto-Discovery, Status Manual/Auto, Menu Discovery Manual/Auto, USSD Manager, SMS Manager) sont **toutes implémentées et opérationnelles**.

Les manques restants sont des **améliorations UX** (SMS non-lu, historique global, raccourcis), des **corrections de robustesse** (start_app.bat, init_db.sql, JWT secret), et des **fonctionnalités bonus** (graphique signal, export global SMS).

Le code est proprement structuré (Go + WebSocket + MySQL + frontend SPA) et prêt pour une utilisation en production avec les modules SIM800C réels sur COM5 (Orange CI) confirmé fonctionnel via tests Putty.

---

## 7. GLOSSAIRE DES ABRÉVIATIONS ET SYMBOLES

### Symboles de statut

| Symbole | Signification |
|---------|---------------|
| ✅ | Implémenté et fonctionnel |
| ⚠️ | Implémenté partiellement ou avec limitations |
| ❌ | Non implémenté |
| 🔧 | Bug connu / à corriger |
| 📋 | Bouton "Copier dans le presse-papiers" |
| 📥 | Bouton "Télécharger / Exporter" |
| ⏳ | Indicateur "En cours / En attente" |
| 🔔 | Notifications sonores activées |
| 🔕 | Notifications sonores désactivées |
| **S→C** | Server → Client (événement WebSocket) |
| **C→S** | Client → Server |
| 🔒 | Le micro-bloc dépend d'un bloc précédent |

### Abréviations des micro-blocs

| Bloc | Signification |
|------|---------------|
| **A1–A4** | SMS is_read : DB → backend → frontend (sms.js) → frontend (app.js) |
| **B1–B4** | Historique global + raccourcis USSD + son SMS |
| **C1–C6** | Audit logs pagination, export SMS global, sparkline signal |
| **D1–D4** | Corrections robustesse (query, start_app.bat, init_db.sql, config) |
| **E1–E3** | Tests unitaires Go + documentation |

### Abréviations techniques

| Abréviation | Définition |
|-------------|------------|
| **AT** | ATtention command — commandes de contrôle SIM800C |
| **USSD** | Unstructured Supplementary Service Data — codes `*xxx#` ou `#xxx#` |
| **COM** | Port série virtuel Windows (COM1, COM5, etc.) |
| **WS** | WebSocket — communication temps réel bidirectionnelle |
| **JWT** | JSON Web Token — authentification sécurisée |
| **IMEI** | Identifiant unique matériel du module SIM800C |
| **CSQ** | Channel Signal Quality — valeur brute 0–31 |
| **RSSI** | Received Signal Strength en dBm = -113 + 2×CSQ |
| **PIN** | Code de verrouillage SIM (4 chiffres) |
| **CI** | Côte d'Ivoire — indicatif +225 |
| **dial_plan** | Table MySQL : plan de numérotation par pays/opérateur |
| **signal_log** | Table MySQL prévue (Bloc C5) : historique signal |
| **ussd_favorites** | Table MySQL : codes USSD mis en favori |
| **ussd_history** | Table MySQL : historique des exécutions USSD |

---

## 8. COMMANDES UTILES

```bat
REM Compiler
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Tests Go
go test ./internal/...

REM Migration DB (après Bloc A1)
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-13.sql

REM Vérifier SMS non-lus (après Bloc A2)
curl "http://test-sim800c.lan:8082/api/modules/1/sms/unread-count"

REM Vérifier historique global (après Bloc B1)
curl "http://test-sim800c.lan:8082/api/ussd/history?limit=100"

REM Vérifier export SMS global (après Bloc C3)
curl "http://test-sim800c.lan:8082/api/sms/export" -o tous_les_sms.csv

REM Vérifier historique signal (après Bloc C6)
curl "http://test-sim800c.lan:8082/api/modules/1/signal/history?limit=20"
```
