# SIM800C Supervisor

Application web de supervision et de contrôle de modules **SIM800C USB** connectés à un PC Windows.

**Stack :** Frontend HTML/CSS/JS · Backend Go · Base de données MySQL  
**Version :** v1-27 · **Accès :** `http://test-sim800c.lan:8082`

---

## Table des matières

1. [Prérequis](#1-prérequis)
2. [Installation rapide](#2-installation-rapide-5-étapes)
3. [Démarrage et arrêt](#3-démarrage-et-arrêt)
4. [Variables d'environnement](#4-variables-denvironnement)
5. [Configuration (config.yaml)](#5-configuration-configyaml)
6. [Fonctionnalités](#6-fonctionnalités)
7. [Structure du projet](#7-structure-du-projet)
8. [Tests](#8-tests)
9. [Dépannage rapide](#9-dépannage-rapide)

---

## 1. Prérequis

| Composant | Version minimale | Remarque |
|-----------|-----------------|----------|
| **Go** | 1.21+ | [golang.org/dl](https://golang.org/dl) |
| **XAMPP** (MySQL) | 8.0+ | MySQL seul suffit — Apache facultatif |
| **Pilote CH340** | — | [wch-ic.com](http://www.wch-ic.com/downloads/CH341SER_EXE.html) — nécessaire pour USB-SERIAL CH340 |
| **Windows** | 10/11 | Port COM reconnu dans le Gestionnaire de périphériques |
| **USB Hub alimenté** | — | Recommandé si ≥ 2 modules |

> **Important :** Chaque module SIM800C doit apparaître dans le Gestionnaire de périphériques avec le nom `USB-SERIAL CH340`.  
> Si ce n'est pas le cas, installez ou réinstallez le pilote CH340.

---

## 2. Installation rapide (5 étapes)

### Étape 1 — Cloner / extraire le projet

```bat
cd C:\xampp\htdocs\aa_Toolbox
:: Extraire v1-27.zip ici
```

### Étape 2 — Créer la base de données MySQL

```bat
C:\xampp\mysql\bin\mysql.exe -u root -e "CREATE DATABASE IF NOT EXISTS sim800c_manager_deepseekv1 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\init_db.sql
```

### Étape 3 — Configurer l'environnement

Copiez `.env.example` vers `.env` et ajustez :

```env
SIM800C_JWT_SECRET=ChangezCeSecretEnProduction2026!
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=sim800c_manager_deepseekv1
```

### Étape 4 — Compiler l'application

```bat
go build -o sim800c-supervisor.exe ./cmd/
```

### Étape 5 — Démarrer

```bat
start_app.bat
```

L'application s'ouvre automatiquement sur `http://test-sim800c.lan:8082`.

> **Première connexion :** identifiants par défaut `admin` / `admin123` — changez le mot de passe immédiatement dans *Paramètres → Profil*.

---

## 3. Démarrage et arrêt

| Script | Action |
|--------|--------|
| `start_app.bat` | Lance MySQL (si XAMPP), compile si nécessaire, applique les migrations, démarre le serveur et ouvre le navigateur |
| `stop_app.bat` | Arrête proprement le serveur Go |

`start_app.bat` effectue automatiquement :
- Vérification si le port 8082 est déjà occupé
- Détection d'une instance déjà en cours (menu O/N/Arrêter)
- Création des dossiers `storage/`, `storage/logs/`, `storage/excel/` si absents
- Copie de `Codes_USSD_CI.xlsx` vers `storage/excel/` si absent
- Application des migrations `migrate_v1-13.sql` et `migrate_v1-25.sql`
- Écriture du PID dans `.pid` pour arrêt propre

---

## 4. Variables d'environnement

Toutes les variables sont lues depuis `.env` à la racine du projet.  
Les variables d'environnement système **remplacent** les valeurs de `config.yaml`.

| Variable | Défaut (config.yaml) | Description |
|----------|----------------------|-------------|
| `SIM800C_JWT_SECRET` | *(valeur config.yaml)* | **Secret JWT — à changer en production** |
| `DB_HOST` | `localhost` | Hôte MySQL |
| `DB_PORT` | `3306` | Port MySQL |
| `DB_USER` | `root` | Utilisateur MySQL |
| `DB_PASSWORD` | *(vide)* | Mot de passe MySQL |
| `DB_NAME` | `sim800c_manager_deepseekv1` | Nom de la base de données |
| `EXCEL_PATH` | `./storage/excel` | Chemin vers le répertoire des fichiers USSD Excel |
| `COM_PORTS` | *(auto-discovery)* | Ports COM forcés, séparés par virgule (ex: `COM3,COM5`) — laisser vide pour l'auto-discovery |

---

## 5. Configuration (config.yaml)

Le fichier `config.yaml` centralise toutes les options de l'application :

```yaml
server:
  port: 8082                    # Port HTTP

serial:
  baud_rate: 9600               # Vitesse série SIM800C

ussd:
  max_menu_depth: 10            # Profondeur max d'exploration de menu
  explore_delay_ms: 3000        # Délai entre étapes d'exploration auto (ms)
  nav_delay_ms: 500             # Délai minimum navigation manuelle (ms)

sms:
  auto_trash_keyword: "Test"    # SMS sans ce mot → corbeille automatique

security:
  jwt_expiration_hours: 24
  enable_auth: true
```

> **Note :** Le `jwt_secret` dans `config.yaml` est un fallback de développement.  
> En production, utilisez toujours `SIM800C_JWT_SECRET` dans `.env`.

---

## 6. Fonctionnalités

### Fonction 1 — Module Auto-Discovery
Scan automatique des ports COM (COM1–COM99) et `/dev/ttyUSB*`, identification des modules SIM800C (commandes `AT`/`ATI`), collecte IMEI / numéro / opérateur, déverrouillage PIN automatique (Orange: `0000`, MTN: `1234`, Moov: `0101`), dashboard temps réel avec graphique de signal (sparkline 20 points).

### Fonction 2 — SIM Status Discovery
- **Manuel :** Boutons USSD générés dynamiquement par opérateur (`Action=Consulter, Target=Interne, Scope=In`) avec info-bulles
- **Auto :** Exécution séquentielle de tous les codes *Consulter* sur chaque module

### Fonction 3 — USSD Menu Discovery
- **Manuel :** Exploration récursive de menus `Services_N1`, enregistrement des nouvelles options dans une nouvelle version de `Codes_USSD_CI-vAAAAMMJJ-HHmmSS.xlsx`
- **Auto :** Exploration automatique multi-modules en parallèle

### Fonction 4 — USSD Manager
Saisie libre d'un code USSD, exécution sur un module sélectionné, navigation interactive dans les sous-menus, favoris, raccourcis des 5 derniers codes.

### Fonction 5 — SMS Manager
Lecture / envoi / suppression de SMS, corbeille automatique (SMS sans le mot "Test"), restauration, suppression définitive, export CSV (par module ou tous modules), badge SMS non-lus, notification sonore.

### Fonctionnalités transversales
- Thème clair / sombre (bouton de bascule)
- Audit logs paginés avec filtres (action, utilisateur)
- Plan de numérotation CI géré en base (Orange `07`, MTN `05`, Moov `01`)
- Gestion des profils utilisateurs et changement de mot de passe
- Export historique USSD CSV
- WebSocket pour toutes les mises à jour temps réel

---

## 7. Structure du projet

```
v1-27/
├── cmd/main.go                  ← Serveur HTTP, routes API, handlers
├── config.yaml                  ← Configuration globale
├── .env                         ← Variables d'environnement (ne pas versionner)
├── start_app.bat / stop_app.bat ← Scripts démarrage / arrêt
├── internal/
│   ├── api/handlers/            ← Handlers HTTP (module, sms, ussd, websocket)
│   ├── auth/auth.go             ← JWT, authentification
│   ├── config/config.go         ← Chargement config + variables env
│   ├── db/db.go                 ← Accès MySQL, toutes les fonctions DB
│   ├── excel/                   ← Lecture/écriture Codes_USSD_CI.xlsx
│   ├── serial/                  ← Communication série SIM800C
│   ├── sms/sms_manager.go       ← Gestion SMS (lecture, envoi, corbeille)
│   ├── ussd/                    ← Exécution, exploration, validation USSD
│   └── websocket/hub.go         ← Hub WebSocket temps réel
├── scripts/
│   ├── init_db.sql              ← Initialisation complète de la base
│   ├── migrate_v1-13.sql        ← Migration is_read (idempotente)
│   └── migrate_v1-25.sql        ← Migration signal_log
├── storage/
│   ├── excel/                   ← Fichiers Codes_USSD_CI*.xlsx
│   └── logs/                    ← Logs applicatifs
└── web/
    ├── index.html               ← SPA principale
    ├── css/main.css + theme-dark.css
    └── js/                      ← Modules JS (app, dashboard, sms, ussd, etc.)
```

---

## 8. Tests

```bat
REM Lancer tous les tests Go
go test ./internal/...

REM Tests DB uniquement (nécessite MySQL de test)
set TEST_DB_DSN=root:@tcp(127.0.0.1:3306)/sim800c_test?parseTime=true
go test ./internal/db/ -v

REM Tests validateur USSD (sans DB)
go test ./internal/ussd/ -v

REM Tests avec couverture
go test ./internal/... -cover
```

> **Base de test DB :** Créez une base `sim800c_test` avant de lancer les tests DB :
> ```bat
> C:\xampp\mysql\bin\mysql.exe -u root -e "CREATE DATABASE IF NOT EXISTS sim800c_test CHARACTER SET utf8mb4;"
> ```
> Les tests créent et nettoient leurs propres données. Si la base est inaccessible, les tests DB sont **skippés** automatiquement (pas d'échec bloquant).

---

## 9. Dépannage rapide

| Symptôme | Cause probable | Solution |
|----------|---------------|----------|
| Module non détecté | Pilote CH340 absent | Installer [CH341SER.exe](http://www.wch-ic.com/downloads/CH341SER_EXE.html) |
| `ERROR` sur codes USSD | SIM verrouillée PIN | L'app tente le PIN par défaut automatiquement ; vérifier l'opérateur détecté |
| Port 8082 occupé | Autre processus | `netstat -ano \| find ":8082"` puis terminer le PID |
| Base de données inaccessible | MySQL non démarré | Lancer XAMPP → démarrer MySQL, ou `start_app.bat` (démarre MySQL automatiquement) |
| `+CUSD: 2` sans réponse | Session USSD expirée | Réduire `explore_delay_ms` dans `config.yaml` ou répondre en < 5 secondes |
| SMS non reçus | Module non enregistré réseau | Vérifier `AT+CREG?` → doit retourner `+CREG: 0,1` |

---

## Plan de numérotation CI

| Opérateur | Préfixe | Exemple |
|-----------|---------|---------|
| Orange CI | `07` | `07 XX XX XX XX` |
| MTN CI | `05` | `05 XX XX XX XX` |
| Moov Africa CI | `01` | `01 XX XX XX XX` |

Indicatif international : `+225` · Format : 10 chiffres

---

*Projet SIM800C Supervisor — © 2026*
