# Guide d'Utilisation de GSD pour le Projet SIM800C USB Manager
> Skill : GSD v2 — https://github.com/gsd-build/gsd-2  
> Mode d'exécution : Auto (`/gsd auto`)  
> Niveau d'autonomie Git : `gsd.approvalMode: auto-approve`  
> Date : 20 Mai 2026

---

## 1. Qu'est-ce que GSD ?

GSD (Get Shit Done) v2 est un système CLI de développement spec-driven qui permet aux agents IA de travailler de façon autonome pendant de longues périodes sans perdre de vue la vision globale du projet. C'est une véritable application TypeScript construite sur le Pi SDK, qui contrôle directement les sessions d'agent : gestion du contexte, branches Git, suivi des coûts, détection de boucles, et reprise après crash.

En résumé pour ce projet : une seule commande, vous partez, vous revenez sur un projet construit avec un historique Git propre.

---

## 2. Prérequis

### 2.1 Environnement Windows

Avant de lancer GSD, s'assurer que les éléments suivants sont en place :

| Prérequis | Version | Vérification |
|---|---|---|
| Node.js | ≥ 22.0.0 (v24 LTS recommandé) | `node --version` |
| npm | ≥ 10.x | `npm --version` |
| Git | Initialisé dans le projet | `git --version` |
| Go | ≥ 1.22 | `go version` |
| MySQL (XAMPP) | ≥ 8.0 | Service MySQL actif dans XAMPP |
| Modules SIM800C | Connectés sur COM5/COM6/COM7 | Gestionnaire de périphériques Windows |

> **Note Windows** : GSD v2.41+ inclut des corrections spécifiques pour Windows : résolution des chemins 8.3, normalisation des backslashes, lancement PowerShell et échappement des parenthèses.

### 2.2 Dépôt Git du Projet

GSD nécessite un dépôt Git initialisé. Si ce n'est pas encore le cas :

```bash
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
git init
git add .
git commit -m "init: projet SIM800C USB Manager"
```

---

## 3. Installation de GSD

### 3.1 Installation globale

```bash
npm install -g gsd-pi@latest
```

Vérifier l'installation :

```bash
gsd --version
```

### 3.2 Lancement et Configuration initiale

```bash
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
gsd
```

Au premier lancement, GSD ouvre un assistant de configuration :

1. **Sélection du provider LLM** → Choisir `OpenRouter` (conformément au nom du dossier `v1-OpenRouter`)
2. **Clé API** → Coller la clé API OpenRouter
3. **Sélection du modèle** → Recommandé : `claude-sonnet-4-6` (via OpenRouter) ou choisir selon budget
4. **Outils optionnels** → Passer (Entrée) sauf si Brave Search / Tavily disponible

---

## 4. Configuration du Projet pour GSD

### 4.1 Fichier de Préférences Projet

Créer le fichier `.gsd/preferences.md` dans le répertoire du projet :

```
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
mkdir .gsd
```

Créer `.gsd/preferences.md` avec le contenu suivant :

```yaml
---
version: 1
models:
  research: openrouter/anthropic/claude-sonnet-4-6
  planning: openrouter/anthropic/claude-sonnet-4-6
  execution: openrouter/anthropic/claude-sonnet-4-6
  completion: openrouter/anthropic/claude-sonnet-4-6
skill_discovery: suggest
auto_supervisor:
  soft_timeout_minutes: 30
  idle_timeout_minutes: 15
  hard_timeout_minutes: 45
budget_ceiling: 100.00
unique_milestone_ids: false
git:
  isolation: worktree
  manage_gitignore: true
verification_commands: []
verification_auto_fix: true
verification_max_retries: 2
auto_report: true
---
```

> **Niveau d'autonomie Git** : La valeur `gsd.approvalMode: auto-approve` correspond à `git.isolation: worktree` + le mode `/gsd auto` qui squash-merge automatiquement chaque milestone sur `main` sans approbation humaine.

### 4.2 Fichier AGENTS.md — Instructions Persistantes

Créer `AGENTS.md` à la racine du projet pour guider l'agent à chaque session :

```markdown
# Agent Instructions — SIM800C USB Manager

## Contexte du Projet
Système de supervision et de contrôle de 3 modules SIM800C USB connectés
sur COM5, COM6, COM7 sous Windows. Backend GoLang + Frontend Web + MySQL.

## Stack Technique Imposée
- Backend : GoLang (Gin framework, gorilla/websocket, go.bug.st/serial)
- Frontend : HTML + CSS (variables thème) + Vanilla JS (pas de framework)
- Base de données : MySQL (XAMPP, port 3306, DB: sim800c_manager)
- Excel : bibliothèque xuri/excelize/v2
- Logging : zerolog

## Contraintes Critiques
- Chaque module SIM800C doit avoir ses propres goroutines read/write dédiées
- Toutes les réponses USSD doivent être diffusées via WebSocket en temps réel
- Valider TOUS les inputs USSD avant envoi (voir règles dans PROJECT.md)
- Les PIN ne doivent JAMAIS apparaître en clair dans les logs
- Filtrer uniquement Scope=In dans Codes_USSD_CI.xlsx
- Timeout USSD : 30 secondes (opérateurs CI peuvent être lents)
- L'encodage UCS2 des réponses USSD doit être décodé en UTF-8

## Chemins Importants
- Projet : C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
- Excel source : C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx
- URL Frontend : test_sim800c.local:80
- Ports COM : COM5 (Orange CI), COM6 (MTN CI), COM7 (Moov Africa CI)

## Plan de Numérotation CI
- Orange CI : préfixe 07 (0701020304)
- MTN CI : préfixe 05 (0501020304)
- Moov Africa CI : préfixe 01 (0101020304)
- Indicatif : +225 / 00225

## Convention de Code GoLang
- Nommer les goroutines avec commentaires (// readLoop goroutine)
- Utiliser des channels bufferisés pour les queues de commandes AT
- Toujours logguer les commandes AT envoyées et les réponses reçues
- Utiliser context.Context pour les timeouts et annulations
```

### 4.3 Fichier PROJECT.md — Vision du Projet

Créer `PROJECT.md` à la racine :

```markdown
# SIM800C USB Manager — PROJECT.md

## Vision
Application web locale de supervision et d'exécution de fonctions sur
3 modules SIM800C USB connectés à un PC Windows. Permet le monitoring
en temps réel, l'exécution de codes USSD des opérateurs CI, et la
gestion des SMS.

## État Actuel
Projet initialisé. Aucun code produit. À construire intégralement.

## Opérateurs Supportés
- Orange CI (SIM sur COM5, préfixe 07)
- MTN CI (SIM sur COM6, préfixe 05)
- Moov Africa CI (SIM sur COM7, préfixe 01)

## Fichier de Référence USSD
C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\Codes_USSD_CI.xlsx
Contient 72 codes USSD (dont ~35 Scope=In actifs).

## Règles de Validation Input USSD
| Type          | Règle                                    |
|---------------|------------------------------------------|
| Aucun         | Pas de saisie                            |
| Choix         | Entier présent dans les options du menu  |
| PIN           | Exactement 4 chiffres                    |
| Code recharge | Exactement 14 chiffres                   |
| Numéro        | Exactement 10 chiffres (sans indicatif)  |
| Montant       | Nombre >= 50                             |
| Référence     | Exactement 14 chiffres                   |
```

---

## 5. Définition du ROADMAP GSD

Créer `.gsd/M001-ROADMAP.md` qui décrit les milestones et slices du projet :

```markdown
# M001 — SIM800C USB Manager v1.0

## Milestone Goal
Application complète fonctionnelle avec les 5 modules principaux.

## Success Criteria
- [ ] Les 3 modules SIM800C sont auto-découverts au démarrage
- [ ] Le dashboard affiche les statuts en temps réel via WebSocket
- [ ] Toutes les fonctions USSD (1-4) sont opérationnelles
- [ ] Le SMS Manager (Fn 5) est opérationnel
- [ ] L'interface supporte thème clair/sombre
- [ ] L'application est accessible sur test_sim800c.local:80

## Slices

- [ ] S01 — Infrastructure & Communication Série
  > Structure GoLang, configuration, driver port série, commandes AT de base, 
  > goroutines par module, init séquence SIM800C, schéma MySQL, migrations DB.
  > Risk: HIGH (hardware dependency)

- [ ] S02 — Module Auto-Discovery & Dashboard Temps Réel (Fn 1)
  > Scan ports COM, identification IMEI + numéro SIM, déduction opérateur,
  > hub WebSocket, dashboard frontend avec cartes modules.
  > Risk: MEDIUM

- [ ] S03 — SIM Status Discovery (Fn 2-1 & 2-2)
  > Chargement codes USSD depuis Excel/DB (Scope=In, Action=Consulter),
  > boutons avec infobulles, exécution individuelle et automatique,
  > affichage résultats temps réel.
  > Risk: MEDIUM

- [ ] S04 — USSD Menu Discovery (Fn 3-1 & 3-2)
  > FSM sessions USSD multi-étapes, exploration récursive menus Services_N1,
  > parser réponses menus, détection nouveaux codes, versioning Excel horodaté.
  > Risk: HIGH (opérateur-dependent behavior)

- [ ] S05 — USSD Manager Manuel (Fn 4)
  > Saisie manuelle code USSD, sélection module, validation inputs,
  > exécution et affichage résultat.
  > Risk: LOW

- [ ] S06 — SMS Manager (Fn 5)
  > Lecture SMS (AT+CMGL), envoi SMS, suppression, notifications push (AT+CNMI),
  > corbeille automatique (filtre "Test"), interface frontend SMS.
  > Risk: LOW

- [ ] S07 — UI Polish & Thèmes
  > Thème clair/sombre (CSS variables), bouton bascule, design impeccable,
  > notifications toast, indicateurs de progression, responsive.
  > Risk: LOW
```

---

## 6. Exécution du Projet avec GSD Auto

### 6.1 Lancement du Mode Auto

```bash
# Se positionner dans le répertoire du projet
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter

# Ouvrir GSD
gsd

# Dans la session GSD, lancer le mode auto
/gsd auto
```

GSD va alors :
1. Lire `.gsd/M001-ROADMAP.md` et `PROJECT.md`
2. Démarrer avec S01 — Infrastructure
3. Pour chaque slice : Research → Plan → Execute (tâche par tâche) → Commit → Slice suivant
4. Squash-merge automatique sur `main` à la fin du milestone

### 6.2 Surveiller la Progression (Terminal 2)

Ouvrir un second terminal pendant que le premier tourne en auto :

```bash
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
gsd

# Vérifier la progression
/gsd status

# Dashboard en overlay
# Ctrl+Alt+G
```

### 6.3 Interagir sans Interrompre Auto

```bash
# Voir ce que GSD fait en ce moment
/gsd status

# Donner une directive architecturale (prise en compte au prochain boundary)
/gsd discuss

# Mettre en file un prochain milestone
/gsd queue

# Arrêter proprement (reprendre avec /gsd auto)
/gsd stop
```

---

## 7. Comportement de GSD selon les Phases

Chaque slice passe automatiquement par les phases : Plan (avec recherche intégrée) → Execute (par tâche) → Complete → Reassess Roadmap → Slice suivant. La phase Plan explore le code, recherche les docs pertinentes, et décompose le slice en tâches vérifiables mécaniquement. Execute exécute chaque tâche dans une fenêtre de contexte fraîche avec uniquement les fichiers pertinents pré-chargés.

Pour ce projet, voici ce que GSD fera concrètement par slice :

| Slice | Ce que GSD va produire |
|---|---|
| S01 | `go.mod`, `main.go`, `internal/config/`, `internal/serial/`, `internal/db/`, migrations MySQL, `.env` |
| S02 | `internal/serial/discovery.go`, `internal/websocket/hub.go`, `public/dashboard.js`, cartes modules frontend |
| S03 | `internal/excel/reader.go`, `internal/ussd/service.go`, boutons SIM Status avec infobulles |
| S04 | `internal/ussd/fsm.go`, `internal/ussd/auto_discovery.go`, `internal/excel/writer.go` |
| S05 | Handler USSD manuel, formulaire frontend avec validation dynamique |
| S06 | `internal/sms/`, `internal/sms/monitor.go` (AT+CNMI), SMS Manager frontend |
| S07 | `public/css/themes.css`, bascule thème, polish design global |

---

## 8. Reprise après Interruption

Si la session est interrompue (crash, fermeture), le fichier de lock trace l'unité en cours. Au prochain `/gsd auto`, GSD lit le fichier de session survivant, synthétise un briefing de reprise depuis chaque outil ayant fonctionné sur le disque, et reprend avec le contexte complet.

```bash
# Reprendre après interruption
gsd --continue
# ou
gsd
/gsd auto
```

---

## 9. Référence Rapide des Commandes GSD Utiles

| Commande | Utilité pour ce projet |
|---|---|
| `/gsd auto` | Lancer l'exécution autonome complète |
| `/gsd status` | Voir la progression (slices, tâches, coûts) |
| `/gsd stop` | Arrêter proprement (reprend avec `/gsd auto`) |
| `/gsd discuss` | Donner des directives architecturales en cours de route |
| `/gsd steer` | Modifier le plan d'un slice en cours d'exécution |
| `/gsd doctor` | Vérifier la santé du projet GSD (worktrees, état) |
| `/gsd forensics` | Déboguer si auto-mode est bloqué |
| `/gsd logs` | Consulter les logs d'activité GSD |
| `/gsd export --html` | Générer un rapport HTML du milestone |
| `Ctrl+Alt+G` | Ouvrir le dashboard overlay en temps réel |

---

## 10. Gestion des Coûts

Chaque unité de travail capture l'usage de tokens et les coûts, décomposés par phase, slice et modèle. Des plafonds de budget peuvent mettre auto mode en pause avant tout dépassement.

Pour ce projet (7 slices, environ 3-5 tâches par slice) :

| Scénario | Estimation coût OpenRouter |
|---|---|
| Modèle économique (claude-haiku-4-5) | ~$3–8 total |
| Modèle standard (claude-sonnet-4-6) | ~$15–30 total |
| Modèle premium (claude-opus-4-6) | ~$50–100 total |

Recommandation : utiliser `claude-sonnet-4-6` pour l'exécution et `claude-haiku` pour la recherche :

```yaml
# Dans .gsd/preferences.md
models:
  research: openrouter/anthropic/claude-haiku-4-5-20251001
  planning: openrouter/anthropic/claude-sonnet-4-6
  execution: openrouter/anthropic/claude-sonnet-4-6
  completion: openrouter/anthropic/claude-haiku-4-5-20251001
budget_ceiling: 50.00
```

---

## 11. Résolution des Problèmes Fréquents

### Problème : GSD ne trouve pas les fichiers du projet

```bash
# Vérifier que Git est initialisé
git status

# Vérifier la structure .gsd/
ls .gsd/
```

### Problème : Auto mode bloqué sur une tâche

```bash
# Dans un second terminal GSD
/gsd forensics
# Lire le diagnostic, puis
/gsd auto   # reprend avec briefing de reprise
```

### Problème : Conflits de merge Git

Chaque milestone tourne dans son propre git worktree avec une branche `milestone/<MID>`. Tout le travail du slice est committé séquentiellement — pas de changement de branche, pas de conflits de merge. Quand le milestone est complet, il est squash-mergé sur main en un seul commit propre.

```bash
# Vérifier l'état du worktree
/gsd doctor
```

### Problème : Erreur de port série sous Windows

Si GSD génère du code qui ne trouve pas les ports COM, ajouter cette instruction dans `AGENTS.md` :

```markdown
## Note ports série Windows
Utiliser `go.bug.st/serial` (pas `github.com/tarm/serial`).
Lister les ports via `serial.GetPortsList()` et filtrer les ports COM actifs.
Tester chaque port avec un timeout de 2s avant de l'inclure dans le registre.
```

---

## 12. Checklist Complète Avant Lancement

```
[ ] Node.js v24 LTS installé (`node --version` → v24.x.x)
[ ] `npm install -g gsd-pi@latest` effectué
[ ] XAMPP démarré (Apache + MySQL)
[ ] Git initialisé dans C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter
[ ] Modules SIM800C connectés et visibles (COM5, COM6, COM7)
[ ] Fichier Codes_USSD_CI.xlsx présent dans storage\excel\
[ ] Fichier .gsd/preferences.md créé (avec clé OpenRouter + budget)
[ ] Fichier AGENTS.md créé à la racine
[ ] Fichier PROJECT.md créé à la racine
[ ] Fichier .gsd/M001-ROADMAP.md créé
[ ] `gsd config` exécuté (provider OpenRouter configuré)
[ ] `gsd` lancé → `/gsd auto` tapé → GSD démarre S01
```

---

## 13. Résultat Attendu

Après exécution complète de `/gsd auto` sur M001 :

```
C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter\
│
├── main.go                 ← Binaire Go compilable
├── .env                    ← Config (COM5/6/7, MySQL, etc.)
├── go.mod / go.sum
├── internal/               ← Toute la logique backend
├── public/                 ← Frontend HTML/CSS/JS complet
├── .gsd/
│   ├── M001-ROADMAP.md     ← Toutes les cases cochées [x]
│   ├── reports/            ← Rapport HTML du milestone
│   └── milestones/M001/    ← Summaries, plans, UAT scripts
└── (historique Git propre avec commits par tâche)
```

**Pour lancer l'application après génération :**

```bash
# Compiler et lancer le backend Go
go build -o sim800c-manager.exe .
./sim800c-manager.exe

# Accéder au frontend
# http://test_sim800c.local:80
```
