# Guide de Déploiement — SIM800C Supervisor

**Version :** v1-27 · **Date :** 26 Mai 2026

---

## 1. Prérequis système

- Windows 10/11 (64 bits)
- Go 1.21 ou supérieur (`go version`)
- XAMPP avec MySQL 8.0+ (ou MySQL standalone)
- Pilote USB-SERIAL CH340 installé
- Hub USB alimenté recommandé (si ≥ 2 modules SIM800C)

---

## 2. Préparation de la base de données

### 2.1 Créer la base

```bat
C:\xampp\mysql\bin\mysql.exe -u root -e "CREATE DATABASE IF NOT EXISTS sim800c_manager_deepseekv1 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 2.2 Initialiser le schéma complet

```bat
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\init_db.sql
```

`init_db.sql` crée toutes les tables avec `CREATE TABLE IF NOT EXISTS` (idempotent) :

| Table | Description |
|-------|-------------|
| `modules` | Modules SIM800C détectés |
| `sms_messages` | SMS (avec `is_read`, `is_trash`, `is_deleted`) |
| `ussd_history` | Historique des exécutions USSD |
| `ussd_favorites` | Favoris USSD |
| `dial_plan` | Plan de numérotation par pays/opérateur |
| `users` | Utilisateurs de l'application |
| `audit_log` | Journal d'audit des actions |
| `excel_versions` | Historique des versions de Codes_USSD_CI.xlsx |
| `app_settings` | Paramètres applicatifs persistants |
| `signal_log` | Historique du signal radio par module |

### 2.3 Migrations (si mise à jour depuis une version antérieure)

```bat
REM Migration v1-13 : ajout colonne is_read dans sms_messages
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-13.sql

REM Migration v1-25 : création table signal_log
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-25.sql
```

> Ces migrations sont **idempotentes** — elles peuvent être exécutées plusieurs fois sans risque.  
> `start_app.bat` les applique automatiquement à chaque démarrage.

---

## 3. Configuration de l'environnement

### 3.1 Fichier `.env`

Créez `.env` à la racine du projet (ou modifiez le fichier existant) :

```env
# Secret JWT — OBLIGATOIRE en production
SIM800C_JWT_SECRET=ChangezCeSecretEnProduction2026!

# Base de données
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=sim800c_manager_deepseekv1

# Chemin Excel (optionnel — défaut : ./storage/excel)
# EXCEL_PATH=./storage/excel

# Ports COM forcés (optionnel — laisser vide pour l'auto-discovery)
# COM_PORTS=COM3,COM5,COM7
```

### 3.2 Fichier `config.yaml`

Les paramètres `config.yaml` sont surchargés par les variables d'environnement `.env`.  
Paramètres clés :

```yaml
server:
  port: 8082

ussd:
  explore_delay_ms: 3000   # Délai entre étapes exploration auto (ms) — min 3000 recommandé
  nav_delay_ms: 500         # Délai navigation manuelle

sms:
  auto_trash_keyword: "Test"  # SMS sans ce mot → corbeille automatique
```

---

## 4. Compilation et démarrage

### 4.1 Compilation manuelle

```bat
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c
go build -o sim800c-supervisor.exe ./cmd/
```

### 4.2 Démarrage via script (recommandé)

```bat
start_app.bat
```

### 4.3 Arrêt

```bat
stop_app.bat
```

---

## 5. Accès à l'application

URL : `http://test-sim800c.lan:8082`

> Si `test-sim800c.lan` ne résout pas, ajoutez cette ligne dans `C:\Windows\System32\drivers\etc\hosts` :
> ```
> 127.0.0.1   test-sim800c.lan
> ```

**Identifiants par défaut :** `admin` / `admin123`  
**Changer le mot de passe :** *Paramètres → Profil → Changer le mot de passe*

---

## 6. Routes API — Référence complète (v1-27)

### Modules

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/modules` | Liste tous les modules détectés |
| `GET` | `/api/modules/{id}` | Détails d'un module |
| `GET` | `/api/modules/{id}/signal` | Signal actuel du module |
| `GET` | `/api/modules/{id}/signal/history?limit=20` | Historique signal (sparkline) |

### USSD

| Méthode | Route | Description |
|---------|-------|-------------|
| `POST` | `/api/modules/{id}/ussd/execute` | Exécuter un code USSD |
| `POST` | `/api/modules/{id}/ussd/navigate` | Navigation dans un menu USSD |
| `POST` | `/api/modules/{id}/ussd/auto-status` | SIM Status Auto-Discovery (module) |
| `POST` | `/api/modules/{id}/ussd/auto-menu` | USSD Menu Auto-Discovery (module) |
| `POST` | `/api/ussd/auto-status` | SIM Status Auto-Discovery (tous modules) |
| `POST` | `/api/ussd/auto-menu` | USSD Menu Auto-Discovery (tous modules) |
| `POST` | `/api/ussd/explore/{id}/{code}` | Explorer un menu USSD spécifique |
| `GET` | `/api/ussd/history?module_id=0&limit=50` | Historique USSD (`module_id=0` = tous modules) |
| `GET` | `/api/ussd/history/export` | Export CSV historique USSD |
| `GET` | `/api/modules/{id}/ussd/recent?limit=5` | 5 derniers codes exécutés sur un module |
| `GET` | `/api/ussd/favorites` | Liste des favoris USSD |
| `POST` | `/api/ussd/favorites` | Ajouter un favori |
| `DELETE` | `/api/ussd/favorites/{id}` | Supprimer un favori |

### SMS

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/modules/{id}/sms` | SMS d'un module |
| `POST` | `/api/modules/{id}/sms/send` | Envoyer un SMS |
| `GET` | `/api/modules/{id}/sms/export` | Export CSV SMS d'un module |
| `DELETE` | `/api/modules/{id}/sms/{index}` | Supprimer un SMS (soft delete) |
| `POST` | `/api/sms/trash/{id}` | Déplacer en corbeille |
| `POST` | `/api/sms/restore/{id}` | Restaurer depuis la corbeille |
| `DELETE` | `/api/sms/delete-permanent/{id}` | Suppression définitive |
| `POST` | `/api/sms/mark-read/{id}` | Marquer un SMS comme lu |
| `POST` | `/api/modules/{id}/sms/mark-all-read` | Marquer tous les SMS d'un module comme lus |
| `POST` | `/api/sms/read-all` | Marquer tous les SMS (tous modules) comme lus |
| `GET` | `/api/modules/{id}/sms/unread-count` | Nombre de SMS non lus d'un module |
| `GET` | `/api/sms/export` | Export CSV SMS tous modules |

### Plan de numérotation

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/dialplan` | Liste des entrées |
| `POST` | `/api/dialplan` | Ajouter une entrée |
| `PUT` | `/api/dialplan/{id}` | Modifier une entrée |
| `DELETE` | `/api/dialplan/{id}` | Supprimer une entrée |
| `POST` | `/api/dialplan/reload` | Recharger depuis la DB |
| `GET` | `/api/dialplan/export` | Export CSV |

### Configuration

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/config` | Configuration courante |
| `PUT` | `/api/config/delays` | Modifier les délais USSD |
| `GET` | `/api/config/advanced` | Configuration avancée |
| `PUT` | `/api/config/advanced` | Modifier configuration avancée |
| `GET` | `/api/config/ports` | Whitelist ports COM |
| `PUT` | `/api/config/ports` | Modifier whitelist ports COM |

### Système / Sécurité

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/system/status` | Statut système |
| `GET` | `/api/user/profile` | Profil utilisateur connecté |
| `POST` | `/api/user/password` | Changer le mot de passe |
| `GET` | `/api/audit/logs?page=1&action=&user=` | Audit logs paginés + filtrés |

### Excel

| Méthode | Route | Description |
|---------|-------|-------------|
| `POST` | `/api/excel/reload` | Recharger le fichier Excel USSD |
| `GET` | `/api/excel/versions` | Historique des versions Excel |

### WebSocket

| Route | Description |
|-------|-------------|
| `GET /api/ws` | Connexion WebSocket temps réel |

---

## 7. Événements WebSocket

| Événement | Direction | Description |
|-----------|-----------|-------------|
| `module_update` | Serveur → Client | Mise à jour d'un module |
| `module_connected` | S→C | Nouveau module connecté |
| `module_initialized` | S→C | Module initialisé (SIM info collectées) |
| `module_disconnected` | S→C | Module déconnecté |
| `discovery_scan_complete` | S→C | Scan auto-discovery terminé |
| `pin_unlocked` | S→C | PIN déverrouillé avec succès |
| `pin_unlock_failed` | S→C | Échec déverrouillage PIN |
| `ussd_result` | S→C | Résultat exécution USSD |
| `auto_status_progress` | S→C | Progression SIM Status Auto |
| `auto_menu_progress` | S→C | Progression USSD Menu Auto |
| `signal_update` | S→C | Mise à jour signal temps réel |
| `signal_history` | S→C | Historique signal (sparkline) |
| `sms_received` | S→C | Nouveau SMS reçu |
| `sms_auto_trash` | S→C | SMS déplacé automatiquement en corbeille |
| `sms_moved_to_trash` | S→C | SMS mis en corbeille manuellement |
| `sms_restored` | S→C | SMS restauré depuis la corbeille |
| `sms_deleted_permanent` | S→C | SMS supprimé définitivement |
| `sms_deleted` | S→C | SMS supprimé (soft) |
| `sms_unread_count` | S→C | Mise à jour compteur SMS non-lus |
| `config_updated` | S→C | Configuration modifiée |
| `dialplan_reloaded` | S→C | Plan de numérotation rechargé |

---

## 8. Historique des versions

| Version | Date | Changements principaux |
|---------|------|----------------------|
| v1-1 à v1-12 | Avril 2026 | Fondations : auto-discovery, USSD, SMS, thème, JWT |
| **v1-13** | Mai 2026 | A1 : Ajout `is_read` dans `sms_messages` + migration |
| **v1-14** | Mai 2026 | A2 : Backend mark-read / unread-count |
| **v1-15** | Mai 2026 | A3 : Frontend badge SMS non-lus |
| **v1-16** | Mai 2026 | A4 : Synchronisation badge temps réel (WebSocket) |
| **v1-17** | Mai 2026 | B1 : `GetUSSDHistory(0)` → tous modules |
| **v1-18** | Mai 2026 | B2 : Page historique USSD global (frontend) |
| **v1-19** | Mai 2026 | B3 : Raccourcis 5 derniers codes USSD Manager |
| **v1-20** | Mai 2026 | B4 : Notification sonore nouveau SMS |
| **v1-21** | Mai 2026 | C1 : Audit logs paginés (backend) |
| **v1-22** | Mai 2026 | C2 : Audit logs avec filtres (frontend) |
| **v1-23** | Mai 2026 | C3 : Export SMS tous modules (backend) |
| **v1-24** | Mai 2026 | C4 : Bouton export SMS global (frontend) |
| **v1-25** | Mai 2026 | C5 : Table `signal_log` + `LogSignal` après mesure CSQ |
| **v1-26** | Mai 2026 | C6 : Route `signal/history` + sparkline SVG (frontend) |
| **v1-27** | Mai 2026 | D1-D4 : Corrections robustesse (query all-modules, start_app.bat, init_db.sql, config env) |
| **v1-28** | Mai 2026 | E1 : Tests Go — `internal/db/db_test.go` |
| **v1-29** | Mai 2026 | E2 : Tests Go — `internal/ussd/validator_test.go` |
| **v1-30** | Mai 2026 | E3 : Documentation README + DEPLOYMENT_GUIDE |

---

## 9. Lancer les tests

```bat
REM Créer la base de test
C:\xampp\mysql\bin\mysql.exe -u root -e "CREATE DATABASE IF NOT EXISTS sim800c_test CHARACTER SET utf8mb4;"

REM Tous les tests
go test ./internal/... -v

REM Tests DB (avec base de test)
set TEST_DB_DSN=root:@tcp(127.0.0.1:3306)/sim800c_test?parseTime=true
go test ./internal/db/ -v

REM Tests validateur USSD (sans DB)
go test ./internal/ussd/ -v

REM Couverture de code
go test ./internal/... -cover
```

> Si `TEST_DB_DSN` n'est pas défini ou si la base est inaccessible, les tests DB sont **skippés** automatiquement.

---

## 10. Commandes utiles

```bat
REM Compiler
go build -o sim800c-supervisor.exe ./cmd/

REM Vérifier signal temps réel
curl "http://test-sim800c.lan:8082/api/modules/1/signal/history?limit=20"

REM Export SMS tous modules
curl "http://test-sim800c.lan:8082/api/sms/export" -o tous_les_sms.csv

REM Historique USSD tous modules
curl "http://test-sim800c.lan:8082/api/ussd/history?module_id=0&limit=50"

REM Audit logs page 1, filtrer action ussd_execute
curl "http://test-sim800c.lan:8082/api/audit/logs?page=1&action=ussd_execute"

REM Appliquer migration is_read
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-13.sql

REM Appliquer migration signal_log
C:\xampp\mysql\bin\mysql.exe -u root sim800c_manager_deepseekv1 < scripts\migrate_v1-25.sql
```

---

*SIM800C Supervisor — Guide de Déploiement v1-27 — 26 Mai 2026*
