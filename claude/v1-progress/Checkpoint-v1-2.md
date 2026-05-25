# Checkpoint — SIM800C Supervisor v1
**Dernière mise à jour :** 23 Mai 2026 — Session 2

---

## Ce qui a été fait (session 1 — résumé)
1. **Auto-Discovery COM ports** — scan dynamique COM1–COM99 + `/dev/ttyUSB*`
2. **PIN auto-unlock** — détection `+CPIN: SIM PIN` et essai des codes par défaut (Orange=0000, MTN=12345, Moov=0101)
3. **Carrier detection** — depuis le préfixe du numéro CI (07→Orange, 05→MTN, 01→Moov)
4. **USSD text formatting** — `FormatUSSDResponse()` normalise les espaces excessifs
5. **Theme toggle** — bouton clair/sombre dans le header, `theme.js` chargé
6. **WebSocket temps réel** — connexion WS avec reconnexion automatique, panneau d'événements
7. **Boutons F2-1** — SIM Status Manual-Discovery (codes Consulter/Interne/In)
8. **Boutons F3-1** — USSD Menu Manual-Discovery (codes Services_N1/Interne/In)
9. **Endpoints API** — `/status-codes`, `/menu-codes`

---

## Ce qui a été fait (session 2 — corrections critiques)

### CORRECTION 1 : Deadlock Go — CRITIQUE ✅
**Problème :** `initialize()` appelait `s.mu.Lock()` puis `sendCommandWithResponse()` qui appelait aussi `s.mu.Lock()` → deadlock garanti, le serveur se bloquait au démarrage.

**Solution :** Architecture dual-mutex :
- `s.mu` — protège les champs de la struct (PhoneNumber, Carrier, IMEI, readerStarted, rb)
- `s.cmdMu` — sérialise les commandes AT sur le port série (empêche l'entrelacement)
- Renommage `sendCommandWithResponse` → `sendCommandRaw` (n'utilise que `cmdMu`)
- `initialize()` n'acquiert plus `s.mu` — appelle directement `sendCommandRaw`

### CORRECTION 2 : cond.Wait goroutine — CRITIQUE ✅
**Problème :** Le pattern `go func() { rb.cond.Wait(); rb.mu.Unlock(); waitCh <- struct{}{} }()` crée un data race : la goroutine déverrouillerait un mutex qu'elle n'a pas locké.

**Solution :** Remplacement par un simple polling avec `time.Sleep(50ms)` — plus simple, sans race condition, performances identiques pour ce cas d'usage.

### CORRECTION 3 : ParseMenuResponse trop restrictif ✅
**Problème :** Ne reconnaissait que les options à 1 chiffre (0-9), manquait `00:Accueil`, `00:Retour`, et ignorait les lignes avec format `"N: texte"` après FormatUSSDResponse.

**Solution :** Regex `(?m)^\s*(\d{1,2})[:.]\s*(.+)$` — supporte 1 ou 2 chiffres, les deux formats (`:` et `.`), avec déduplication par numéro d'option.

### CORRECTION 4 : ExecuteWithMenu incorrect ✅
**Problème :** `ExecuteWithMenu` utilisait `req.Code = choice` puis appelait `Execute()` → passait par le `commandChan` et le validator USSD alors qu'un choix de menu (`"1"`, `"2"`) doit juste être envoyé brut.

**Solution :** `ExecuteWithMenu` appelle directement `req.Module.ExecuteUSSDRaw(choice)` — contourne le canal de commandes et le validator, envoie le choix immédiatement.

### CORRECTION 5 : start_app.bat et stop_app.bat ✅
**Problèmes :**
- Port hardcodé `8080` (devrait être `8082`)
- Ports COM hardcodés COM5/6/7 (pas d'auto-discovery)
- Pas de vérification/démarrage MySQL
- Pas de `stop_app.bat`

**Solution :**
- `start_app.bat` reécrit : détecte MySQL, tente de le démarrer via XAMPP, initialise la DB si besoin, informe l'utilisateur de l'auto-scan COM1-COM20
- `stop_app.bat` créé : `taskkill` sur `sim800c-supervisor.exe`

---

## Fichiers modifiés (session 2)

| Fichier | Modification |
|---------|-------------|
| `internal/serial/sim800c.go` | **REFACTORING MAJEUR** : dual-mutex (mu + cmdMu), sendCommandWithResponse→sendCommandRaw, initialize() sans lock, suppression cond.Wait goroutine, waitReadUntil avec sleep polling |
| `internal/serial/manager.go` | Struct SIM800C avec champ `cmdMu sync.Mutex`, connectModule appelle startSingleReader avant initialize |
| `internal/ussd/executor.go` | ParseMenuResponse avec regex multi-digits, ExecuteWithMenu via ExecuteUSSDRaw direct |
| `start_app.bat` | Réécriture complète : MySQL auto-start, port 8082, info auto-scan, compilation Go |
| `stop_app.bat` | **NOUVEAU** — arrêt propre via taskkill |

---

## État actuel du code

### Architecture de concurrence (après corrections)
```
Manager
  └─ connectModule(port)
       ├─ tserial.OpenPort(port)          // ouvre le port série
       ├─ SIM800C{mu, cmdMu, ...}         // struct avec deux mutex
       ├─ startSingleReader()             // goroutine lecteur (protégée par mu)
       │    └─ goroutine: bufio.ReadBytes → rb.push(line)
       ├─ go initialize()                 // initialise sans tenir de lock
       │    └─ sendCommandRaw(cmd)        // utilise cmdMu pour sérialiser
       └─ go handleCommands()             // lit commandChan, appelle sendCommandRaw
```

### Fonctions implémentées

| Fonction | Statut |
|----------|--------|
| F1 — Module Auto-Discovery | ✅ COM1-99 scan, PIN unlock, carrier detection |
| F2-1 — SIM Status Manual-Discovery | ✅ Boutons par code/module |
| F2-2 — SIM Status Auto-Discovery | ✅ Bouton global |
| F3-1 — USSD Menu Manual-Discovery | ✅ Boutons par code/module |
| F3-2 — USSD Menu Auto-Discovery | ✅ Bouton global |
| F4 — USSD Manager | ✅ Saisie manuelle et exécution |
| F5 — SMS Manager | ✅ Create/Read/Delete/Corbeille |
| Thème clair/sombre | ✅ Toggle button + theme.js |
| WebSocket temps réel | ✅ Reconnexion auto, événements |
| USSD text formatting | ✅ FormatUSSDResponse |
| PIN auto-unlock | ✅ checkAndUnlockPIN |
| Carrier detection | ✅ detectCarrierFromNumber |
| Input validation | ✅ validator.go |
| start_app.bat / stop_app.bat | ✅ Scripts complets |

---

## Décisions prises (session 2)

- **Dual-mutex plutôt que RWMutex seul :** `mu` protège l'état de la struct, `cmdMu` sérialise les échanges série. Séparation claire des responsabilités et évite tout deadlock.
- **Polling sleep 50ms au lieu de cond.Wait :** Pour un port série avec timeout de 30s, attendre 50ms entre les lectures est négligeable. Élimine un pattern goroutine dangereux.
- **ExecuteUSSDRaw exposé publiquement :** Permet au module USSD Explorer d'envoyer des choix de menu directement sans passer par le commandChan (qui bloquerait si le canal est plein ou si handleCommands n'est pas lancé).
- **start_app.bat complet :** XAMPP MySQL est l'infrastructure standard pour ce projet. Le bat tente de démarrer MySQL automatiquement avant l'app Go.

---

## Prochaines étapes

### Priorité HAUTE
1. **Test réel sur Windows/COM5** — vérifier que l'app démarre, que le module est détecté, que le PIN est déverrouillé automatiquement
2. **Vérifier le délai de navigation menu** — entre les choix de menu (AT+CUSD doit être envoyé en < 5 secondes selon les notes du projet). Vérifier `explore_delay_ms` dans config.yaml (actuellement 1500ms — peut-être trop court pour la navigation orange).
3. **Tester FormatUSSDResponse** sur les vraies réponses de `#111#` et `#122#` — vérifier que le texte est propre.

### Priorité MOYENNE
4. **Persistance des modules en base de données** — actuellement les modules sont en mémoire RAM. Si le serveur redémarre, les IDs changent, l'historique USSD perd sa référence. Ajouter une table `modules` avec `port` comme clé unique.
5. **Indicateur de statut PIN** dans le dashboard — afficher si la SIM est déverrouillée ou non.
6. **Notification WebSocket** après PIN unlock — broadcaster un événement `pin_unlocked` pour mettre à jour l'UI en temps réel.

### Priorité BASSE
7. **Tests unitaires** — `checkAndUnlockPIN`, `detectCarrierFromNumber`, `FormatUSSDResponse`, `ParseMenuResponse`
8. **Navigation interactive USSD** — permettre à l'utilisateur de naviguer dans les menus USSD step-by-step depuis le frontend (input texte pour envoyer le prochain choix)
9. **Whitelist préfixes COM** dans config.yaml pour accélérer le scan si l'utilisateur connaît les ports probables

---

## Notes importantes pour la prochaine session

### Fichier Excel Codes_USSD_CI.xlsx
- Chemin : `C:/xampp/htdocs/aa_Toolbox/test_sim800c/storage/excel/Codes_USSD_CI.xlsx`
- Utilisé par : `internal/excel/reader.go` → `GetConsultCodes(carrier)` et `GetServiceNCodes(carrier)`
- Le fichier est séparé du répertoire de déploiement (`v1/`) mais référencé par `config.yaml`

### Identifiants par défaut
- Application web : `admin` / `admin123`
- URL : `http://test-sim800c.lan:8082`

### Commandes utiles
```bat
REM Compiler
go build -o sim800c-supervisor.exe ./cmd/

REM Démarrer
start_app.bat

REM Arrêter
stop_app.bat

REM Initialiser la base de données (première fois)
C:\xampp\mysql\bin\mysql.exe -u root < scripts\init_db.sql
```
