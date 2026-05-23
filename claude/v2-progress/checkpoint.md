# Checkpoint — SIM800C Supervisor v1
**Dernière mise à jour :** 23 Mai 2026

---

## Ce qui a été fait (session actuelle)

### 1. Auto-Discovery des ports COM (`internal/serial/manager.go`) ✅
- **Avant :** Ports hardcodés dans `config.yaml` (COM5, COM6, COM7)
- **Après :** Scan dynamique COM1–COM99 + `/dev/ttyUSB*` + `/dev/ttyACM*` via `scanCOMPorts()`
- Le scan essaie d'ouvrir chaque port avec un timeout court (1s) pour détecter les modems connectés
- Les ports configurés dans `config.yaml` sont toujours essayés en premier (`ports: []` par défaut)
- `monitorModules()` re-scan régulièrement pour détecter les nouveaux modules branchés à chaud

### 2. Déverrouillage automatique du PIN SIM (`internal/serial/sim800c.go`) ✅
- **Avant :** Aucune gestion de `+CPIN: SIM PIN` → les codes USSD échouaient silencieusement
- **Après :** `checkAndUnlockPIN()` détecte `SIM PIN` via `AT+CPIN?` et tente les codes PIN par défaut
  - Orange CI → `0000`
  - MTN CI → `12345`
  - Moov Africa CI → `0101`
- Appelé pendant `initialize()` ET lors d'une erreur d'exécution USSD (retry automatique)

### 3. Détection automatique de l'opérateur (`internal/serial/sim800c.go`) ✅
- **Avant :** Champ `Carrier` jamais rempli
- **Après :** `detectCarrierFromNumber()` identifie l'opérateur depuis le préfixe du numéro CI :
  - `07XXXXXXXX` → Orange
  - `05XXXXXXXX` → MTN
  - `01XXXXXXXX` → Moov
- Appelé après récupération du numéro (AT+CNUM ou USSD)

### 4. Formatage du texte USSD (`internal/serial/sim800c.go`) ✅
- **Avant :** Texte brut retourné avec espaces excessifs (alignement pour vieux téléphones)
- **Après :** `FormatUSSDResponse()` normalise les espaces multiples → espace simple, supprime lignes vides
- Résultat propre et lisible dans l'interface

### 5. Bouton de changement de thème (`web/index.html`) ✅
- **Avant :** `theme.js` existait mais n'était pas chargé, aucun bouton dans l'UI
- **Après :**
  - `<script src="/js/theme.js">` ajouté dans l'HTML
  - Bouton "🌙 Thème sombre / ☀️ Thème clair" dans le header
  - Feuilles CSS light/dark swappées via JS
  - `theme-dark.css` amélioré avec toutes les variables CSS nécessaires

### 6. WebSocket temps réel connecté (`web/index.html`) ✅
- **Avant :** WebSocket hub Go existait mais frontend ne se connectait pas
- **Après :**
  - `connectWebSocket()` appelé au démarrage de l'app
  - Reconnexion automatique (5s) en cas de déconnexion
  - Indicateur de statut WS dans le header (point vert/gris)
  - `handleWSEvent()` gère : `module_connected`, `module_disconnected`, `ussd_result`, `sms_received`, `auto_discovery_progress`
  - Panneau "Événements temps réel" en haut de page (50 derniers événements)

### 7. Boutons Manual Status Discovery (Fonction 2-1) ✅
- **Avant :** Aucun bouton par code USSD/module
- **Après :**
  - Dans chaque module card du Dashboard : boutons générés dynamiquement depuis `GET /api/modules/{id}/ussd/status-codes`
  - Dans l'onglet USSD Manager : section dédiée "SIM Status Manual-Discovery" par module
  - Chaque bouton = un code USSD (Carrier = opérateur module, Action=Consulter, Target=Interne, Scope=In)
  - Info-bulle (title) avec : opération, code USSD, entrée/sortie attendue

### 8. Boutons Menu Explorer (Fonction 3-1) ✅
- **Avant :** Aucun bouton individuel par code Services_N1
- **Après :**
  - Dans l'onglet Explorer : section dédiée par module avec boutons générés depuis `GET /api/modules/{id}/ussd/menu-codes`
  - Chaque bouton = un code USSD (Action=Services_N1, Target=Interne, Scope=In)
  - Au clic → exploration complète du menu et sous-menus (`/api/ussd/explore/{id}/{code}`)

### 9. Nouveaux endpoints API (`cmd/main.go` + `internal/api/handlers/ussd.go`)
- `GET /api/modules/{id}/ussd/status-codes` → codes Consulter/Interne/In pour l'opérateur du module
- `GET /api/modules/{id}/ussd/menu-codes` → codes Services_N1/Interne/In pour l'opérateur du module

### 10. config.yaml mis à jour
- `serial.ports: []` (vide = auto-discovery uniquement)
- `excel.base_path` corrigé vers `C:/xampp/htdocs/aa_Toolbox/test_sim800c/storage/excel`

---

## Fichiers modifiés

| Fichier | Modification |
|---------|-------------|
| `internal/serial/manager.go` | Auto-discovery COM ports, scan dynamique, hot-plug |
| `internal/serial/sim800c.go` | PIN unlock, carrier detection, USSD text formatting |
| `internal/api/handlers/ussd.go` | Ajout GetStatusCodes, GetMenuCodes, imports strconv+mux |
| `cmd/main.go` | Routes status-codes, menu-codes, handlers, import strconv |
| `web/index.html` | Thème toggle, WebSocket, boutons F2-1, boutons F3-1, RT events |
| `web/css/theme-dark.css` | Variables CSS complètes pour thème sombre |
| `config.yaml` | Ports vides (auto-discovery), chemin excel corrigé |

---

## État actuel du code

### Fonctions implémentées

| Fonction | Statut |
|----------|--------|
| F1 — Module Auto-Discovery | ✅ Complet (scan COM + PIN + carrier) |
| F2-1 — SIM Status Manual-Discovery | ✅ Complet (boutons par code/module) |
| F2-2 — SIM Status Auto-Discovery | ✅ Complet (bouton global) |
| F3-1 — USSD Menu Manual-Discovery | ✅ Complet (boutons par code/module) |
| F3-2 — USSD Menu Auto-Discovery | ✅ Complet (bouton global) |
| F4 — USSD Manager | ✅ Complet |
| F5 — SMS Manager | ✅ Complet (create/read/delete/corbeille) |
| Thème clair/sombre | ✅ Complet |
| WebSocket temps réel | ✅ Complet |
| USSD text formatting | ✅ Complet |
| PIN auto-unlock | ✅ Complet |
| Carrier detection | ✅ Complet |
| Input validation | ✅ Complet (validator.go existant) |

---

## Décisions prises

- **Scan COM1–99 :** Approche simple et fiable sous Windows. Le timeout de 1s évite les blocages longs. Les erreurs sont silencieuses (normal pour les ports inexistants).
- **PIN retry via tous opérateurs :** Si l'opérateur n'est pas encore connu au moment du déverrouillage, on essaie les 3 codes par défaut dans l'ordre.
- **FormatUSSDResponse côté Go :** Le texte est nettoyé avant d'être renvoyé en JSON — le frontend reçoit toujours du texte propre.
- **Boutons F2-1 / F3-1 :** Chargés via appels API distincts (`/status-codes`, `/menu-codes`) plutôt qu'intégrés dans le JSON module, pour rester modulaires et éviter de surcharger `/api/modules`.
- **WebSocket event panel :** Affiché en permanence en haut de page pour visibilité maximale des événements temps réel.

---

## Prochaines étapes

1. **Test réel sur Windows avec COM5** — vérifier que le scan détecte bien le module, que le PIN est déverrouillé automatiquement
2. **Tester FormatUSSDResponse** sur les réponses réelles de `#111#` et `#122#`
3. **Validation des codes USSD avec Information_INPUT** — tester le validator.go avec les codes nécessitant une entrée
4. **Optimisation du scan COM** — possibilité d'ajouter une whitelist de préfixes dans config.yaml pour accélérer le scan
5. **Persistance des modules en base** — actuellement les modules sont en mémoire ; si le serveur redémarre, les IDs changent
6. **Tests unitaires** — ajouter des tests pour `checkAndUnlockPIN`, `detectCarrierFromNumber`, `FormatUSSDResponse`
7. **USSD menu navigation interactive** — permettre à l'utilisateur de naviguer dans les menus USSD de façon interactive depuis le frontend