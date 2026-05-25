# Checkpoint — SIM800C Supervisor v1-12
**Dernière mise à jour :** 25 Mai 2026 — Session 12

---

## Résumé des sessions précédentes (1-11)

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
- Export Dial Plan CSV, Broadcast `discovery_scan_complete`, Badge whitelist active
- Auto-refresh panel statut via WebSocket, Audit log config, Fix `db_id` frontend
- Dashboard: barres signal ASCII + badge PIN OK/KO

---

## Ce qui a été fait — Session 12

### FEAT 1 : SMS — Suppression définitive depuis la corbeille ✅
**Problème (Priorité MOYENNE #3 de v1-11) :** `deletePermanent()` dans `sms.js` était un stub vide qui se contentait de recharger la liste sans rien supprimer.

**Solution :**
- `internal/db/db.go` : nouvelle méthode `DeleteSMSPermanent(smsID int)` — suppression physique `DELETE FROM sms_messages WHERE id = ?`
- `internal/db/db.go` : nouvelle méthode `RestoreSMSFromTrash(smsID int)` — `UPDATE sms_messages SET is_trash = FALSE WHERE id = ?`
- `internal/sms/sms_manager.go` : méthode `DeletePermanent(smsID int)` — appelle DB + broadcast WS `sms_deleted_permanent`
- `internal/sms/sms_manager.go` : méthode `RestoreFromTrash(smsID int)` — appelle DB + broadcast WS `sms_restored`
- `cmd/main.go` : routes `POST /api/sms/restore/{id}` et `DELETE /api/sms/delete-permanent/{id}` + handlers `restoreFromTrashHandler` et `deletePermanentHandler`
- `web/js/sms.js` : implémentation réelle de `restoreFromTrash()` et `deletePermanent()` avec `fetch()` vers les nouvelles routes + toast notification + `showToast()` helper

**Fichiers :** `internal/db/db.go`, `internal/sms/sms_manager.go`, `cmd/main.go`, `web/js/sms.js`

---

### FEAT 2 : Notification WS `sms_auto_trash` ✅
**Problème (Priorité MOYENNE #4 de v1-11) :** Quand un SMS entrant était automatiquement placé en corbeille (car ne contenant pas le mot-clé), seul `sms_received` était émis. Aucun retour visuel spécifique ne distinguait un SMS normal d'un SMS auto-trashé.

**Solution :**
- `internal/sms/sms_manager.go` dans `ReadIncomingSMS()` : après le broadcast `sms_received`, si `isTrash && m.autoTrashKeyword != ""`, émettre un broadcast `sms_auto_trash` avec `{ module_id, sender, preview }` (preview tronqué à 60 chars)
- `web/js/app.js` : nouveau `case 'sms_auto_trash'` — affiche un toast warning "📂 SMS auto-corbeille (module X) de [sender] — [preview]" + reload `smsManager.loadSMS()`
- `web/js/app.js` : nouveaux cases `sms_moved_to_trash`, `sms_restored`, `sms_deleted_permanent` → `smsManager.loadSMS()` (mise à jour temps réel de la liste SMS)

**Fichiers :** `internal/sms/sms_manager.go`, `web/js/app.js`

---

### FEAT 3 : Indicateur visuel "Exploration en cours" sur les cartes modules ✅
**Problème (Priorité MOYENNE #5 de v1-11) :** Lors d'une exploration USSD Menu Auto-Discovery, rien n'indiquait visuellement dans la carte du module concerné que l'exploration était en cours. Les boutons restaient actifs.

**Solution :**
- `web/js/app.js` dans `case 'auto_menu_progress'` : après mise à jour du `liveDiv`, chercher `#module-{event.module_id}` et :
  - Si `d.status === 'exploring'` : injecter un badge `<span class="exploring-badge">⏳ Exploration...</span>` dans le header de la carte (via `.module-header, h3, .card-title`) + animation `pulse` CSS + désactiver les boutons `.btn-auto-menu, .btn-menu-explore` avec `data-was-disabled-by-exploration`
  - Si `d.status === 'done'` : supprimer le badge + réactiver les boutons désactivés
- `web/css/main.css` : règle `@keyframes pulse` (0%/100% opacity:1, 50% opacity:0.5) + style `.exploring-badge` via inline CSS dans le JS

**Fichiers :** `web/js/app.js`, `web/css/main.css`

---

### FEAT 4 : Pagination de l'historique USSD (50 entrées/page) ✅
**Problème (Priorité BASSE #6 de v1-11) :** L'historique USSD affichait toutes les entrées en une seule table sans pagination. Avec des centaines d'entrées, l'interface devenait très lente.

**Solution :**
- `cmd/main.go` : augmentation du `limit` par défaut de 100 → 2000 dans `getUSSDHistoryHandler` (retourne tout pour la pagination côté frontend)
- `web/js/history.js` : refactoring complet — classe `HistoryManager` avec :
  - `this.currentPage = 1`, `this.pageSize = 50`
  - Méthode `applyFilterAndRender()` : applique filtres puis `render()`
  - Méthode `render()` : découpe `this.filtered` en pages avec `slice(startIdx, startIdx + pageSize)`, construit `buildPaginationHTML(totalPages)` avec numéros de page, boutons Préc./Suiv., ellipses pour grandes séries
  - Pagination rebindée à chaque render via querySelectorAll + addEventListener
- `web/css/main.css` : styles `.pagination-controls`, `.pagination-btn`, `.pagination-btn.active`, `.pagination-ellipsis`, `.pagination-info`

**Fichiers :** `cmd/main.go`, `web/js/history.js`, `web/css/main.css`

---

### FEAT 5 : Filtre/recherche dans l'historique USSD ✅
**Problème (Priorité BASSE #7 de v1-11) :** Impossible de filtrer l'historique USSD par code ou résultat.

**Solution :**
- `web/index.html` : ajout de deux nouveaux contrôles dans la toolbar de l'historique :
  - `<select id="history-status-filter">` : filtrer par statut `all / success / error`
  - `<input type="search" id="history-search">` : recherche texte libre dans code USSD + résultat + opération + module_id
  - `<span id="history-filter-stats">` : compteur "X / Y entrée(s)" mis à jour par `updateFilterStats()`
- `web/js/history.js` : `this.searchTerm` + `applyFilterAndRender()` qui filtre `this.history` selon statut et terme de recherche — reset `currentPage = 1` à chaque changement de filtre
- La div `ussd-history-list` a été renommée `history-list` (cohérence avec le JS refactorisé)

**Fichiers :** `web/index.html`, `web/js/history.js`

---

### FEAT 6 : Bouton "📋 Copier" le résultat USSD ✅
**Problème (Priorité BASSE #8 de v1-11) :** Impossible de copier facilement un résultat USSD (solde, numéro, etc.) depuis l'interface.

**Solution :**
- `web/js/ussd.js` : nouvelle méthode `addCopyButton(container, text)` :
  - Ajoute un `<button class="btn-copy-ussd">📋 Copier</button>` sous le résultat dans `#ussd-output`
  - Utilise `navigator.clipboard.writeText()` avec fallback `execCommand('copy')` pour les navigateurs anciens
  - Feedback visuel : "✅ Copié!" pendant 1.8s puis retour à "📋 Copier"
  - Supprime le bouton précédent avant d'en ajouter un nouveau (évite les doublons)
- Appelé dans `executeUSSD()` après affichage du résultat ET dans `navigateChoice()` après chaque réponse de navigation
- `web/js/history.js` : bouton `📋` dans chaque ligne de l'historique (colonne "Actions") — même logique clipboard avec feedback visuel

**Fichiers :** `web/js/ussd.js`, `web/js/history.js`

---

## Fichiers modifiés (session 12)

| Fichier | Modification |
|---------|-------------|
| `internal/db/db.go` | +`RestoreSMSFromTrash()` ; +`DeleteSMSPermanent()` |
| `internal/sms/sms_manager.go` | +`RestoreFromTrash()` ; +`DeletePermanent()` ; +broadcast `sms_auto_trash` dans `ReadIncomingSMS()` |
| `cmd/main.go` | +route `POST /api/sms/restore/{id}` ; +route `DELETE /api/sms/delete-permanent/{id}` ; +`restoreFromTrashHandler` ; +`deletePermanentHandler` ; history limit 100→2000 |
| `web/index.html` | history section: +`history-status-filter` ; +`history-search` ; +`history-filter-stats` ; div renommée `history-list` |
| `web/js/app.js` | +`case 'sms_auto_trash'` ; +`case 'sms_moved_to_trash/restored/deleted_permanent'` ; auto_menu_progress: +badge exploring + disable/enable btns |
| `web/js/sms.js` | `restoreFromTrash()` implémenté (fetch POST /restore) ; `deletePermanent()` implémenté (fetch DELETE) ; +`showToast()` helper |
| `web/js/ussd.js` | +`addCopyButton()` ; appel dans `executeUSSD()` et `navigateChoice()` |
| `web/js/history.js` | Refactoring complet: pagination 50/page, filtre statut, recherche texte, bouton 📋, stats |
| `web/css/main.css` | +`.pagination-controls/btn/active/disabled` ; +`@keyframes pulse` ; +`.pagination-ellipsis` |

---

## État actuel — Fonctionnalités implémentées

| Fonctionnalité | État | Notes |
|----------------|------|-------|
| Auto-Discovery modules | ✅ | COM1..COM99 + Linux /dev/ttyUSB* |
| Identification SIM/Carrier | ✅ | CNUM + USSD universel |
| PIN auto-unlock | ✅ | Codes par défaut Orange/MTN/Moov |
| Dashboard temps réel | ✅ | WebSocket |
| Signal visuel (barres + RSSI) | ✅ | |
| Badge PIN status (OK/KO) | ✅ | |
| Panel statut système | ✅ | Auto-refresh via WS |
| Fonction 2-1: Status Manual (boutons Consulter) | ✅ | |
| Fonction 2-2: Status Auto-Discovery | ✅ | Global + par module |
| Fonction 3-1: USSD Menu Manual-Discovery | ✅ | |
| Fonction 3-2: USSD Menu Auto-Discovery | ✅ | Global + par module |
| Indicateur "Exploration en cours" carte module | ✅ | NEW session 12 |
| Fonction 4: USSD Manager (saisie libre) | ✅ | |
| Bouton 📋 Copier résultat USSD | ✅ | NEW session 12 |
| Fonction 5: SMS Manager | ✅ | Créer, Lire, Supprimer, Export CSV |
| SMS Restaurer depuis corbeille | ✅ | NEW session 12 (était un stub) |
| SMS Supprimer définitivement | ✅ | NEW session 12 (était un stub) |
| Notification WS `sms_auto_trash` | ✅ | NEW session 12 |
| Corbeille SMS automatique | ✅ | Mot-clé configurable + persistant |
| Navigation USSD interactive (step-by-step) | ✅ | Countdown 25s |
| Formatage texte USSD | ✅ | ▒→é, - - - préservé |
| Signal quality + réseau | ✅ | AT+CSQ, AT+CREG, WebSocket |
| Historique USSD + pagination (50/page) | ✅ | NEW session 12 |
| Historique USSD filtre statut + recherche | ✅ | NEW session 12 |
| Historique USSD bouton 📋 Copier | ✅ | NEW session 12 |
| Export CSV historique USSD | ✅ | |
| Favoris USSD | ✅ | |
| Export Dial Plan CSV | ✅ | |
| Plan de numérotation (CRUD) | ✅ | |
| Thème clair/sombre | ✅ | |
| start_app.bat / stop_app.bat | ✅ | |
| Audit logs | ✅ | Config, USSD, etc. |
| Config avancée (delays, depth, keyword) | ✅ | Persistant DB |
| Whitelist ports COM | ✅ | Badge + restauration démarrage |

---

## Routes API complètes

```
GET  /api/modules
GET  /api/modules/{id}
POST /api/modules/{id}/ussd/execute
POST /api/modules/{id}/ussd/navigate
POST /api/modules/{id}/ussd/auto-status
POST /api/modules/{id}/ussd/auto-menu
POST /api/ussd/auto-status
POST /api/ussd/auto-menu
POST /api/ussd/explore/{id}/{code}
GET  /api/modules/{id}/signal
GET  /api/ussd/history            ← limit 2000 par défaut (v1-12)
GET  /api/ussd/history/export
GET  /api/ussd/favorites
POST /api/ussd/favorites
DELETE /api/ussd/favorites/{id}
GET  /api/dialplan
POST /api/dialplan
POST /api/dialplan/reload
PUT  /api/dialplan/{id}
DELETE /api/dialplan/{id}
GET  /api/dialplan/export
GET  /api/config
PUT  /api/config/delays
GET  /api/config/advanced
PUT  /api/config/advanced
GET  /api/config/ports
PUT  /api/config/ports
GET  /api/system/status
GET  /api/modules/{id}/sms
POST /api/modules/{id}/sms/send
GET  /api/modules/{id}/sms/export
DELETE /api/modules/{id}/sms/{index}
POST /api/sms/trash/{id}
POST /api/sms/restore/{id}         ← NEW v1-12
DELETE /api/sms/delete-permanent/{id} ← NEW v1-12
POST /api/sms/read-all
GET  /api/user/profile
POST /api/user/password
GET  /api/audit/logs
POST /api/excel/reload
GET  /api/excel/versions
GET  /api/ws  (WebSocket)
```

---

## WebSocket Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `module_update` | Server→Client | Mise à jour état module |
| `module_connected` | Server→Client | Nouveau module détecté |
| `module_initialized` | Server→Client | Module initialisé (SIM OK) |
| `module_disconnected` | Server→Client | Module débranché |
| `discovery_scan_complete` | Server→Client | Scan terminé `{modules_total, new_found, ports_scanned}` |
| `pin_unlock_failed` | Server→Client | Échec déverrouillage PIN |
| `auto_status_progress` | Server→Client | Progression status auto `{port, operation, ussd_code, result}` |
| `auto_menu_progress` | Server→Client | Progression menu auto `{port, ussd_code, operation, status, result}` |
| `signal_update` | Server→Client | Mise à jour signal CSQ/CREG |
| `ussd_result` | Server→Client | Résultat USSD exécuté |
| `sms_received` | Server→Client | Nouveau SMS reçu |
| `sms_auto_trash` | Server→Client | SMS auto-placé en corbeille `{module_id, sender, preview}` — NEW v1-12 |
| `sms_moved_to_trash` | Server→Client | SMS déplacé vers corbeille |
| `sms_restored` | Server→Client | SMS restauré — NEW v1-12 |
| `sms_deleted_permanent` | Server→Client | SMS supprimé définitivement — NEW v1-12 |
| `sms_deleted` | Server→Client | SMS supprimé (soft) |
| `config_updated` | Server→Client | Configuration modifiée |
| `dialplan_reloaded` | Server→Client | Plan de numérotation rechargé |

---

## Décisions prises (session 12)

1. **Pagination côté frontend** : Plutôt qu'ajouter `LIMIT/OFFSET` SQL dans l'API (et gérer le total count), on charge jusqu'à 2000 entrées et on pagine en JS. Simple à maintenir, suffisant pour l'usage courant. Si l'historique dépasse 10 000 entrées, il faudra passer à une vraie pagination API.

2. **`DeleteSMSPermanent` = DELETE physique** : Contrairement à `MarkSMSDeleted` (soft delete avec `is_deleted=TRUE`), la suppression depuis la corbeille fait un `DELETE` réel pour libérer l'espace et éviter la confusion entre "soft deleted" et "in trash".

3. **`sms_auto_trash` uniquement si `autoTrashKeyword != ""`** : Si aucun mot-clé n'est configuré, tous les SMS iraient en corbeille et le broadcast serait spammant. La condition `m.autoTrashKeyword != ""` évite ce cas.

4. **`addCopyButton` remplace l'ancien** : À chaque appel USSD (execute ou navigate), on supprime le bouton Copier précédent avant d'en créer un nouveau. Évite l'accumulation de boutons dans `#ussd-output` qui est un `pre`/`div` réutilisé.

5. **`history-list` vs `ussd-history-list`** : Renommage de la div pour cohérence avec le nouveau `HistoryManager` refactorisé (l'ancien code ciblait `#ussd-history-list` qui n'existait pas dans le JS).

---

## Prochaines étapes — Priorité HAUTE

### 1. Tests réels sur COM5 (Orange CI) — validation complète
- Vérifier que `sms_auto_trash` s'affiche bien (configurer keyword ≠ "Test" pour tester)
- Vérifier restauration SMS depuis la corbeille (bouton "↩️ Restaurer")
- Vérifier pagination historique après quelques exécutions USSD
- Vérifier le badge "⏳ Exploration..." sur la carte module pendant Auto-Menu

### 2. Tests unitaires Go (critiques)
- `RestoreSMSFromTrash` / `DeleteSMSPermanent` : vérifier que les requêtes SQL sont correctes
- `sms_auto_trash` broadcast : mock du hub WS, vérifier que l'event est émis seulement si `isTrash && keyword != ""`

---

## Prochaines étapes — Priorité MOYENNE

### 3. SMS : Notification sonore/visuelle pour nouveau SMS
Actuellement un toast informatif apparaît. Ajouter un badge de compteur non-lu sur l'onglet SMS dans la navigation + option son (optionnel, désactivable).

### 4. SMS : Marquer comme lu/non-lu
Ajouter un champ `is_read BOOLEAN DEFAULT FALSE` en DB + indicateur visuel (point bleu) pour les SMS non lus + action "Marquer comme lu".

### 5. USSD Manager : Historique rapide (derniers codes exécutés)
Dans le USSD Manager, afficher les 5 derniers codes USSD exécutés sur ce module comme raccourcis cliquables (type "historique du navigateur").

---

## Prochaines étapes — Priorité BASSE

### 6. Audit log : Pagination + recherche
La table Audit Logs est actuellement limitée à 100 entrées et sans filtre. Ajouter pagination + filtre par action/userID comme pour l'historique USSD.

### 7. Dashboard : Graphique de signal dans le temps
Pour chaque module, enregistrer les valeurs CSQ au fil du temps et afficher un mini graphique sparkline dans la carte module (chart.js ou SVG inline).

### 8. Export global SMS (tous modules)
Actuellement l'export SMS est par module. Ajouter un endpoint `GET /api/sms/export?module_id=all` pour exporter tous les SMS en un seul CSV.

---

## Structure fichier v1-12.zip

```
v1-12/
├── cmd/main.go              ← +routes sms/restore + sms/delete-permanent ;
│                               +restoreFromTrashHandler + deletePermanentHandler ;
│                               history limit 100→2000
├── config.yaml              ← inchangé
├── internal/
│   ├── db/db.go             ← +RestoreSMSFromTrash() ; +DeleteSMSPermanent()
│   ├── serial/manager.go    ← inchangé
│   ├── serial/sim800c.go    ← inchangé
│   ├── sms/sms_manager.go   ← +RestoreFromTrash() ; +DeletePermanent() ;
│   │                           +broadcast sms_auto_trash dans ReadIncomingSMS
│   └── ussd/
│       ├── executor.go      ← inchangé
│       ├── explorer.go      ← inchangé
│       └── validator.go     ← inchangé
├── scripts/
│   └── init_db.sql          ← inchangé
└── web/
    ├── index.html           ← history section: +status-filter +search +stats ; div→history-list
    ├── css/main.css         ← +pagination CSS ; +@keyframes pulse
    └── js/
        ├── app.js           ← +case sms_auto_trash/restored/deleted_permanent ;
        │                       auto_menu_progress: +badge exploring +disable btns
        ├── history.js       ← Refactoring complet: pagination+filtre+recherche+copy
        ├── sms.js           ← restoreFromTrash() implémenté ; deletePermanent() implémenté ;
        │                       +showToast() helper
        └── ussd.js          ← +addCopyButton() ; appel dans executeUSSD + navigateChoice
```

## Commandes utiles

```bat
REM Compiler (depuis le dossier v1-12)
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Tester restauration SMS
curl -X POST http://test-sim800c.lan:8082/api/sms/restore/42

REM Tester suppression définitive SMS
curl -X DELETE http://test-sim800c.lan:8082/api/sms/delete-permanent/42

REM Vérifier historique avec 2000 entrées
curl "http://test-sim800c.lan:8082/api/ussd/history?limit=2000" | python3 -m json.tool | grep -c ussd_code
```

## Points d'attention

- **Renommage `ussd-history-list` → `history-list`** : Si vous avez d'autres scripts ou extensions qui ciblent `#ussd-history-list`, les mettre à jour.

- **`DeleteSMSPermanent` irréversible** : Contrairement à `MarkSMSDeleted` (soft delete, récupérable), `DeleteSMSPermanent` fait un `DELETE` physique. La confirmation JavaScript `confirm()` est la seule protection — considérer une seconde confirmation si besoin.

- **Pagination frontend vs API** : La pagination actuelle est entièrement côté frontend (JS). Si `history.length > 2000`, les entrées les plus anciennes ne seront pas visibles. Pour les installations très actives, passer à une pagination côté API avec `LIMIT/OFFSET` SQL et un endpoint `/api/ussd/history/count`.

- **`exploring-badge` sélecteur de carte** : Le badge cherche `.module-header, h3, .card-title` dans `#module-{id}`. Si le rendu de `renderModules()` change le HTML des cartes, vérifier que l'un de ces sélecteurs est toujours présent.
