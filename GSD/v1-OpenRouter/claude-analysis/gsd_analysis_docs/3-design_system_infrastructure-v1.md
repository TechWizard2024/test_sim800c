# Design Système — Infrastructure
> Projet : SIM800C USB Supervisor & USSD Manager  
> Version : v1  
> Date : 20 Mai 2026

---

## 1. Vue Globale de l'Infrastructure

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         MACHINE WINDOWS LOCALE                          │
│                                                                         │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │
│  │  SIM800C USB #1  │  │  SIM800C USB #2  │  │  SIM800C USB #3  │      │
│  │  SIM: Orange CI  │  │  SIM: MTN CI     │  │  SIM: Moov CI    │      │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘      │
│           │                     │                      │                │
│           └─────────────────────┴──────────────────────┘                │
│                                 │                                       │
│                    ┌────────────▼──────────────┐                        │
│                    │   USB Hub 3.0 Alimenté    │                        │
│                    └────────────┬──────────────┘                        │
│                                 │                                       │
│              COM5              COM6            COM7                     │
│           (Orange CI)        (MTN CI)       (Moov CI)                  │
│                    ┌────────────▼──────────────┐                        │
│                    │   Backend GoLang          │                        │
│                    │   Port HTTP : 8080        │                        │
│                    │   Port WS   : 8080/ws     │                        │
│                    └────────────┬──────────────┘                        │
│                                 │                                       │
│              ┌──────────────────┼──────────────────┐                   │
│              │                  │                  │                    │
│   ┌──────────▼──────┐  ┌───────▼────────┐  ┌─────▼──────────────┐     │
│   │  MySQL (XAMPP)  │  │  Fichiers Excel│  │  Logs Fichiers     │     │
│   │  Port : 3306    │  │  storage/excel │  │  logs/app.log      │     │
│   │  DB: sim800c_mg │  │                │  │  logs/at_cmds.log  │     │
│   └─────────────────┘  └────────────────┘  └────────────────────┘     │
│                                 │                                       │
│                    ┌────────────▼──────────────┐                        │
│                    │   Frontend Web (Static)   │                        │
│                    │   Servi par GoLang        │                        │
│                    │   test_sim800c.local:80   │                        │
│                    └───────────────────────────┘                        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Composants d'Infrastructure

### 2.1 Serveur Web — GoLang (Backend)

| Attribut | Valeur |
|---|---|
| Port HTTP | 8080 (proxy depuis port 80 via hosts) |
| Port WebSocket | ws://localhost:8080/ws |
| URL locale | http://test_sim800c.local:80 |
| Fichiers statiques | Servis directement par le backend Go (embed ou dossier `public/`) |
| Mode | Single binary exécutable Windows |

**Résolution DNS locale :**
```
# C:\Windows\System32\drivers\etc\hosts
127.0.0.1    test_sim800c.local
```

**Proxy port 80 → 8080** : via configuration XAMPP Apache (VirtualHost) ou directement Go sur port 80 avec droits administrateur.

---

### 2.2 Base de Données — MySQL (XAMPP)

| Attribut | Valeur |
|---|---|
| Port | 3306 |
| Host | localhost |
| Base de données | `sim800c_manager` |
| Utilisateur | `sim800c_user` (ou `root` en dev) |
| Mot de passe | Configurable via `.env` |
| Moteur | InnoDB |
| Encodage | UTF-8 (`utf8mb4`) |
| Collation | `utf8mb4_unicode_ci` |

---

### 2.3 Ports COM — Communication Série

| Port | Module | SIM | Baud Rate | Data Bits | Stop Bits | Parité |
|---|---|---|---|---|---|---|
| COM5 | SIM800C #1 | Orange CI (07...) | 9600 | 8 | 1 | Aucune |
| COM6 | SIM800C #2 | MTN CI (05...) | 9600 | 8 | 1 | Aucune |
| COM7 | SIM800C #3 | Moov Africa CI (01...) | 9600 | 8 | 1 | Aucune |

> **Note** : Le baud rate est configurable via `.env` — certains modules SIM800C USB utilisent 115200.

---

### 2.4 Fichiers Système

```
C:\xampp\htdocs\aa_Toolbox\test_sim800c\
│
├── GSD\v1-OpenRouter\              ← Code source généré
│   ├── main.go
│   ├── .env
│   ├── go.mod / go.sum
│   ├── internal\
│   │   ├── serial\                 ← Communication ports COM
│   │   ├── ussd\                   ← Logique USSD (FSM, parser)
│   │   ├── sms\                    ← Logique SMS
│   │   ├── db\                     ← Couche MySQL
│   │   ├── excel\                  ← Lecture/écriture Excel
│   │   ├── websocket\              ← Hub WebSocket
│   │   └── api\                    ← Handlers HTTP/WS
│   └── public\                     ← Frontend web statique
│       ├── index.html
│       ├── css\
│       └── js\
│
└── storage\
    └── excel\
        ├── Codes_USSD_CI.xlsx      ← Fichier de référence actuel
        └── Codes_USSD_CI-v*.xlsx   ← Versions horodatées
```

---

## 3. Réseau Local

### 3.1 Configuration DNS Local

```
# Fichier hosts Windows
127.0.0.1    test_sim800c.local
```

### 3.2 Flux Réseau

```
Navigateur (localhost)
        │
        │ HTTP GET /         → Frontend statique
        │ HTTP * /api/*      → API REST GoLang
        │ WS /ws             → WebSocket temps réel
        ▼
Backend GoLang :80 (ou :8080)
        │
        ├── MySQL :3306
        ├── COM5 / COM6 / COM7 (serial)
        └── storage/excel/ (filesystem)
```

---

## 4. Configuration Environnement (.env)

```env
# Serveur
SERVER_PORT=80
SERVER_HOST=0.0.0.0

# Base de données
DB_HOST=localhost
DB_PORT=3306
DB_NAME=sim800c_manager
DB_USER=root
DB_PASSWORD=

# Ports COM
COM_PORTS=COM5,COM6,COM7
COM_BAUD_RATE=9600
COM_TIMEOUT_MS=30000

# Excel
EXCEL_DIR=C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel
EXCEL_FILE=Codes_USSD_CI.xlsx

# Logs
LOG_LEVEL=INFO
LOG_DIR=logs

# USSD
USSD_SESSION_TIMEOUT_MS=30000
USSD_MAX_RETRY=3
```

---

## 5. Stratégie de Performance

| Axe | Stratégie |
|---|---|
| Concurrence | 1 goroutine de lecture + 1 goroutine d'écriture par port COM (6 goroutines au total) |
| WebSocket | Hub centralisé avec broadcast par canal Go (channels) |
| MySQL | Pool de connexions (max 10) via `database/sql` |
| Excel | Chargé en mémoire au démarrage, mis à jour sur disque uniquement lors des découvertes |
| USSD Queue | Channel Go (buffered, capacité 100) par module |
| Logs | Logger non-bloquant via goroutine dédiée |

---

## 6. Stratégie de Sécurité

| Axe | Mesure |
|---|---|
| Commandes AT | Whitelist des commandes autorisées, pas d'injection possible |
| Inputs USSD | Validation stricte côté backend (regex + règles métier) |
| PIN | Masqué dans les logs (remplacé par `****`) |
| API | Accès limité à localhost uniquement (bind sur 127.0.0.1) |
| MySQL | Utilisateur dédié avec droits minimaux |
| Fichiers | Chemin d'accès Excel validé (pas de path traversal) |

---

## 7. Stratégie de Disponibilité & Résilience

| Scénario | Comportement |
|---|---|
| Module SIM800C déconnecté | Détection automatique, statut "offline" dans le dashboard, retry toutes les 30s |
| Timeout USSD opérateur | Abandon après 30s, message d'erreur affiché, statut mis à jour en DB |
| Perte connexion MySQL | Retry avec backoff exponentiel (1s, 2s, 4s, 8s...) |
| WebSocket déconnecté | Reconnect automatique côté frontend toutes les 3s |
| Erreur écriture Excel | Log de l'erreur, conservation en DB, retry à la prochaine occasion |

---

## 8. Schéma de Base de Données

### Table : `modules`
```sql
CREATE TABLE modules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    com_port VARCHAR(10) NOT NULL UNIQUE,
    imei VARCHAR(20),
    phone_number VARCHAR(15),
    carrier VARCHAR(50),
    carrier_prefix VARCHAR(5),
    status ENUM('online', 'offline', 'error') DEFAULT 'offline',
    signal_quality INT,
    network_registration VARCHAR(50),
    last_seen_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### Table : `ussd_codes`
```sql
CREATE TABLE ussd_codes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    source_id INT,
    carrier VARCHAR(50),
    action VARCHAR(50),
    target VARCHAR(20),
    operation VARCHAR(200),
    ussd_code VARCHAR(100),
    information_input VARCHAR(100),
    information_output VARCHAR(200),
    scope ENUM('In', 'Out') DEFAULT 'In',
    comment TEXT,
    parent_ussd_id INT,
    discovered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_ussd_id) REFERENCES ussd_codes(id)
);
```

### Table : `ussd_executions`
```sql
CREATE TABLE ussd_executions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    ussd_code_id INT,
    ussd_code_raw VARCHAR(100),
    input_data TEXT,
    response TEXT,
    status ENUM('success', 'error', 'timeout', 'pending') DEFAULT 'pending',
    execution_time_ms INT,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id)
);
```

### Table : `sms_messages`
```sql
CREATE TABLE sms_messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT NOT NULL,
    sim_index INT,
    direction ENUM('received', 'sent') DEFAULT 'received',
    phone_number VARCHAR(20),
    message TEXT,
    status ENUM('inbox', 'trash', 'sent', 'deleted') DEFAULT 'inbox',
    received_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (module_id) REFERENCES modules(id)
);
```

### Table : `excel_versions`
```sql
CREATE TABLE excel_versions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    filename VARCHAR(200) NOT NULL,
    version_tag VARCHAR(30),
    new_codes_count INT DEFAULT 0,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Table : `app_logs`
```sql
CREATE TABLE app_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    module_id INT,
    level ENUM('DEBUG', 'INFO', 'WARN', 'ERROR') DEFAULT 'INFO',
    category VARCHAR(50),
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_level (level),
    INDEX idx_created_at (created_at)
);
```
