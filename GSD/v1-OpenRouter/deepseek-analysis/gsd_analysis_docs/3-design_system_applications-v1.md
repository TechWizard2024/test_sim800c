markdown

Copy

Download

```
# 3. Design des applications - v1
## Version : 1.0
## Date : 22/05/2026
## Projet : SIM800C Supervisor
\---
## 1. Architecture applicative globale
### 1.1 Vue d'ensemble

```

┌─────────────────────────────────────────────────────────────────────────────┐  
│ SIM800C Supervisor │  
├─────────────────────────────────────────────────────────────────────────────┤  
│ │  
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ │  
│ │ Frontend Web │ │ Backend Go │ │ Base de données│ │  
│ │ │ │ │ │ MySQL │ │  
│ │ - HTML5/CSS3 │◄──►│ - API REST │◄──►│ - Modules │ │  
│ │ - JavaScript │ │ - WebSocket │ │ - Historique │ │  
│ │ - WebSocket │ │ - Serial COM │ │ - SMS │ │  
│ │ - Thèmes clair/ │ │ - USSD Engine │ │ - Audit logs │ │  
│ │ sombre │ │ - SMS Engine │ │ - Users │ │  
│ └──────────────────┘ │ - Auth JWT │ └──────────────────┘ │  
│ │ └──────────────────┘ │ │  
│ │ │ │ │  
│ ▼ ▼ ▼ │  
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ │  
│ │ Apache/XAMPP │ │ Modules │ │ Fichier Excel │ │  
│ │ Port 80 │ │ SIM800C USB │ │ Codes USSD │ │  
│ │ │ │ COM5/6/7 │ │ │ │  
│ └──────────────────┘ └──────────────────┘ └──────────────────┘ │  
│ │  
└─────────────────────────────────────────────────────────────────────────────┘

text

Copy

Download

```

\### 1.2 Stack technologique
| Composant | Technologie | Version | Justification |
|-----------|-------------|---------|----------------|
| \*\*Backend\*\* | Go | 1.21+ | Performance, concurrence native, compilation statique |
| \*\*Base de données\*\* | MySQL | 8.0+ | Fiabilité, support transactionnel, XAMPP intégré |
| \*\*Frontend\*\* | HTML5/CSS3/JS | ES6+ | Léger, responsive, pas de framework lourd |
| \*\*Communication temps réel\*\* | WebSocket | RFC 6455 | Bidirectionnel, faible latence |
| \*\*Authentification\*\* | JWT | RFC 7519 | Stateless, scalable |
| \*\*Communication série\*\* | tarm/serial | v0.0+ | Bibliothèque Go standard pour sérial |
| \*\*Manipulation Excel\*\* | excelize | v2.8+ | Support complet des fichiers .xlsx |
| \*\*Logging\*\* | logrus | v1.9+ | Logging structuré, niveaux multiples |
\---
\## 2. Backend Go - Architecture détaillée
\### 2.1 Structure des packages

```

internal/  
├── config/ # Configuration de l'application  
│ └── config.go # Lecture YAML, variables d'environnement  
│  
├── db/ # Accès base de données  
│ └── db.go # Connexion MySQL, modèles, queries  
│  
├── serial/ # Communication avec SIM800C  
│ ├── manager.go # Gestionnaire des ports COM  
│ └── sim800c.go # Commandes AT spécifiques  
│  
├── ussd/ # Logique USSD  
│ ├── executor.go # Exécution de codes USSD  
│ ├── explorer.go # Exploration récursive de menus  
│ └── validator.go # Validation des entrées (PIN, montant...)  
│  
├── sms/ # Gestion des SMS  
│ └── sms\_manager.go # Envoi, lecture, corbeille  
│  
├── excel/ # Manipulation Excel  
│ ├── reader.go # Lecture Codes\_USSD\_CI.xlsx  
│ ├── writer.go # Création nouvelles versions  
│ └── cache.go # Cache des codes USSD  
│  
├── websocket/ # Communication temps réel  
│ └── hub.go # Gestion des connexions WebSocket  
│  
├── auth/ # Authentification  
│ └── auth.go # JWT, login, sessions  
│  
└── api/ # API REST  
└── handlers/ # Handlers par fonctionnalité  
├── module.go # Modules SIM800C  
├── ussd.go # Commandes USSD  
├── sms.go # SMS  
└── websocket.go # WebSocket

text

Copy

Download

```

\### 2.2 Diagramme de classes simplifié
\`\`\`mermaid
classDiagram
    class Config {
        +ServerConfig Server
        +SerialConfig Serial
        +MySQLConfig MySQL
        +ExcelConfig Excel
        +USSDConfig USSD
        +SMSConfig SMS
        +SecurityConfig Security
        +Load(path) Config
    }
    class SIM800C {
        -port string
        -serialPort SerialPort
        -mu Mutex
        +ModuleID int
        +IMEI string
        +PhoneNumber string
        +Carrier string
        +Connect() error
        +ExecuteUSSD(code, input) (string, error)
        +SendSMS(number, message) error
        +ReadSMS(index) (sender, message, error)
        +DeleteSMS(index) error
        +ListSMS() \[\]SMS
    }
    class SerialManager {
        -modules map\[string\]\*SIM800C
        +Start() error
        +Stop()
        +GetModule(port) \*SIM800C
        +GetAllModules() \[\]\*SIM800C
    }
    class USSDExecutor {
        +Execute(req \*USSDRequest) (\*USSDResponse, error)
        +ParseMenuResponse(response string) \[\]MenuOption
    }
    class USSDExplorer {
        -maxDepth int
        +ExploreMenu(module, startCode, parentID) (\*ExplorationResult, error)
        +FormatMenuTree(node, indent) string
    }
    class SMSManager {
        -autoTrashKeyword string
        +SendSMS(moduleID, number, message) error
        +ReadSMS(module) error
        +DeleteSMS(moduleID, index) error
        +MoveToTrash(smsID) error
        +StartMonitoring(interval)
    }
    class ExcelReader {
        -cache map\[int\]USSDCode
        +Load() error
        +GetConsultCodes(carrier) \[\]USSDCode
        +GetServiceNCodes(carrier) \[\]USSDCode
        +GetByUSSDCode(code) (bool, USSDCode)
    }
    class WebSocketHub {
        -clients map\[\*Client\]bool
        +BroadcastEvent(event)
        +RegisterClient(client)
        +UnregisterClient(client)
    }
    class AuthManager {
        +Login(username, password) (token, error)
        +ValidateToken(token) (\*Claims, error)
        +AuthMiddleware(next) http.HandlerFunc
    }
    SerialManager --> SIM800C
    USSDExecutor --> SIM800C
    USSDExplorer --> USSDExecutor
    USSDExplorer --> ExcelReader
    SMSManager --> SIM800C
    SMSManager --> WebSocketHub
```

### 2.3 Interfaces principales

go

Copy

Download

```
// Interface pour les modules SIM800C
type SIM800CInterface interface {
    Connect() error
    Disconnect() error
    ExecuteUSSD(code string, inputData string) (string, error)
    SendSMS(number string, message string) error
    ReadSMS(index int) (string, string, error)
    DeleteSMS(index int) error
    ListSMS() (\[\]map\[string\]interface{}, error)
    GetIMEI() (string, error)
    GetPhoneNumber() (string, error)
}
// Interface pour la gestion USSD
type USSDInterface interface {
    Execute(code string, inputData string) (\*USSDResponse, error)
    ExploreMenu(startCode string, parentID int) (\*ExplorationResult, error)
    ValidateInput(inputType string, value string) error
}
// Interface pour la gestion SMS
type SMSInterface interface {
    Send(moduleID int, number string, message string) error
    Receive(moduleID int) (\[\]SMS, error)
    Delete(moduleID int, index int) error
    MoveToTrash(smsID int) error
}
// Interface pour la base de données
type RepositoryInterface interface {
    SaveModule(module \*Module) error
    GetModuleByCOMPort(comPort string) (\*Module, error)
    GetAllModules() (\[\]Module, error)
    SaveUSSDHistory(history \*USSDHistory) error
    SaveSMS(sms \*SMSMessage) error
    GetSMSByModule(moduleID int, includeTrash bool) (\[\]SMSMessage, error)
    SaveAuditLog(userID, action, targetType string, targetID int, details interface{}, ipAddress string) error
    GetUserByUsername(username string) (\*User, error)
    CreateUser(user \*User) error
}
```

---

## 3\. Base de données - Schéma complet

### 3.1 Modèle relationnel

sql

Copy

Download

```
\-- Table des modules SIM800C
CREATE TABLE modules (
    id INT PRIMARY KEY AUTO\_INCREMENT,
    com\_port VARCHAR(10) UNIQUE NOT NULL,
    imei VARCHAR(15),
    phone\_number VARCHAR(20),
    carrier VARCHAR(50),
    status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
    last\_seen TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    INDEX idx\_status (status),
    INDEX idx\_com\_port (com\_port)
);
\-- Table d'historique USSD
CREATE TABLE ussd\_history (
    id INT PRIMARY KEY AUTO\_INCREMENT,
    module\_id INT NOT NULL,
    ussd\_code VARCHAR(50) NOT NULL,
    input\_data TEXT,
    output\_data TEXT,
    status ENUM('success', 'error', 'timeout') NOT NULL,
    duration\_ms INT,
    executed\_by VARCHAR(50),
    executed\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    FOREIGN KEY (module\_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx\_module (module\_id),
    INDEX idx\_executed\_at (executed\_at)
);
\-- Table des SMS
CREATE TABLE sms\_messages (
    id INT PRIMARY KEY AUTO\_INCREMENT,
    module\_id INT NOT NULL,
    sender\_number VARCHAR(20),
    receiver\_number VARCHAR(20),
    message TEXT NOT NULL,
    direction ENUM('in', 'out') NOT NULL,
    is\_deleted BOOLEAN DEFAULT FALSE,
    is\_trash BOOLEAN DEFAULT FALSE,
    sms\_index INT,
    received\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    FOREIGN KEY (module\_id) REFERENCES modules(id) ON DELETE CASCADE,
    INDEX idx\_module\_direction (module\_id, direction),
    INDEX idx\_is\_trash (is\_trash)
);
\-- Table des utilisateurs
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password\_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'operator', 'viewer') DEFAULT 'viewer',
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    INDEX idx\_username (username)
);
\-- Table d'audit
CREATE TABLE audit\_log (
    id INT PRIMARY KEY AUTO\_INCREMENT,
    user\_id VARCHAR(36),
    action VARCHAR(100) NOT NULL,
    target\_type VARCHAR(50),
    target\_id INT,
    details JSON,
    ip\_address VARCHAR(45),
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    INDEX idx\_user (user\_id),
    INDEX idx\_created\_at (created\_at),
    FOREIGN KEY (user\_id) REFERENCES users(id) ON DELETE SET NULL
);
\-- Table des versions Excel
CREATE TABLE excel\_versions (
    id INT PRIMARY KEY AUTO\_INCREMENT,
    filename VARCHAR(255) NOT NULL,
    version\_date TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    created\_by VARCHAR(50),
    new\_codes\_count INT DEFAULT 0,
    INDEX idx\_version\_date (version\_date)
);
```

### 3.2 Diagramme ER

Diagram

Code

Copy

Download

Fullscreen

```
erDiagram
    MODULES {
        int id PK
        string com\_port UK
        string imei
        string phone\_number
        string carrier
        enum status
        timestamp last\_seen
    }
    
    USSD\_HISTORY {
        int id PK
        int module\_id FK
        string ussd\_code
        text input\_data
        text output\_data
        enum status
        int duration\_ms
        timestamp executed\_at
    }
    
    SMS\_MESSAGES {
        int id PK
        int module\_id FK
        string sender\_number
        string receiver\_number
        text message
        enum direction
        boolean is\_trash
        timestamp received\_at
    }
    
    USERS {
        string id PK
        string username UK
        string password\_hash
        enum role
        timestamp created\_at
    }
    
    AUDIT\_LOG {
        int id PK
        string user\_id FK
        string action
        string target\_type
        int target\_id
        json details
        timestamp created\_at
    }
    
    MODULES ||--o{ USSD\_HISTORY : has
    MODULES ||--o{ SMS\_MESSAGES : has
    USERS ||--o{ AUDIT\_LOG : generates
```

---

## 4\. Frontend - Architecture

### 4.1 Structure des fichiers

text

Copy

Download

```
web/
├── index.html              # Page principale
├── css/
│   ├── main.css           # Styles communs
│   ├── theme-light.css    # Thème clair
│   └── theme-dark.css     # Thème sombre
├── js/
│   ├── app.js             # Application principale
│   ├── websocket.js       # Gestion WebSocket
│   ├── dashboard.js       # Tableau de bord
│   ├── ussd.js            # USSD Manager
│   ├── sms.js             # SMS Manager
│   ├── explorer.js        # Menu Explorer
│   ├── auth.js            # Authentification
│   └── theme.js           # Gestion thèmes
└── assets/
    ├── icons/             # Icônes opérateurs
    └── fonts/             # Polices personnalisées
```

### 4.2 Composants UI

Diagram

Code

Copy

Download

Fullscreen

```
graph TD
    App\[Application Principale\]
    
    App \--> Header\[En-tête\]
    App \--> Nav\[Navigation\]
    App \--> Tabs\[Onglets\]
    
    Tabs \--> Dashboard\[Dashboard\]
    Tabs \--> USSD\[USSD Manager\]
    Tabs \--> SMS\[SMS Manager\]
    Tabs \--> Explorer\[Menu Explorer\]
    Tabs \--> Audit\[Audit Logs\]
    Tabs \--> Settings\[Paramètres\]
    
    Dashboard \--> ModuleCard\[Module Card ×3\]
    ModuleCard \--> SIMInfo\[Informations SIM\]
    ModuleCard \--> ActionButtons\[Boutons Actions\]
    ModuleCard \--> ResultArea\[Zone Résultat\]
    
    USSD \--> USSDSelect\[Module Select\]
    USSD \--> USSDInput\[Code USSD\]
    USSD \--> USSDResult\[Résultat\]
    
    SMS \--> SMSSelect\[Module Select\]
    SMS \--> SMSForm\[Formulaire Envoi\]
    SMS \--> SMSList\[Liste Messages\]
    
    Explorer \--> ExplorerSelect\[Module Select\]
    Explorer \--> MenuTree\[Arbre Menu\]
    
    Audit \--> AuditTable\[Tableau Logs\]
    
    Settings \--> ConfigForm\[Configuration\]
    Settings \--> ThemeToggle\[Thème Clair/Sombre\]
```

### 4.3 États de l'interface

 État | Description | Affichage |
| --- | --- | --- |
 **Non authentifié** | Utilisateur non connecté | Formulaire de login |
 **Chargement** | Attente des données | Spinner / skeleton |
 **Connecté** | API et WebSocket OK | Dashboard complet |
 **Module connecté** | SIM800C détecté | Carte verte |
 **Module déconnecté** | SIM800C non répondant | Carte grise/rouge |
 **Erreur** | Problème réseau ou API | Message d'erreur |
 **Simulation** | Mode démo actif | Badge "Simulation" |

### 4.4 Thèmes

#### Thème Clair

css

Copy

Download

```
:root {
    \--bg-primary: #f5f5f5;
    \--text-primary: #333333;
    \--card-bg: #ffffff;
    \--border-color: #e0e0e0;
    \--primary-color: #007bff;
    \--success-color: #4caf50;
    \--error-color: #f44336;
    \--warning-color: #ff9800;
}
```

#### Thème Sombre

css

Copy

Download

```
:root {
    \--bg-primary: #1a1a2e;
    \--text-primary: #f0f0f0;
    \--card-bg: #16213e;
    \--border-color: #0f3460;
    \--primary-color: #e94560;
    \--success-color: #4caf50;
    \--error-color: #f44336;
    \--warning-color: #ff9800;
}
```

---

## 5\. API REST - Spécification

### 5.1 Endpoints

 Méthode | Endpoint | Description | Auth |
| --- | --- | --- | --- |
 **POST** | 
```
/api/login
```
 | Authentification | Non |
 **POST** | 
```
/api/logout
```
 | Déconnexion | Oui |
 **GET** | 
```
/api/health
```
 | Vérification santé | Non |
 **GET** | 
```
/api/modules
```
 | Liste des modules | Oui |
 **GET** | 
```
/api/modules/{id}
```
 | Détail module | Oui |
 **POST** | 
```
/api/discover
```
 | Découverte modules | Oui |
 **POST** | 
```
/api/modules/{id}/ussd/execute
```
 | Exécuter USSD | Oui |
 **POST** | 
```
/api/ussd/auto-status
```
 | Auto-discovery status | Oui |
 **POST** | 
```
/api/ussd/auto-menu
```
 | Auto-discovery menu | Oui |
 **POST** | 
```
/api/ussd/explore/{id}/{code}
```
 | Explorer menu | Oui |
 **GET** | 
```
/api/modules/{id}/sms
```
 | Liste SMS | Oui |
 **POST** | 
```
/api/modules/{id}/sms/send
```
 | Envoyer SMS | Oui |
 **DELETE** | 
```
/api/modules/{id}/sms/{index}
```
 | Supprimer SMS | Oui |
 **POST** | 
```
/api/sms/trash/{id}
```
 | Déplacer vers corbeille | Oui |
 **GET** | 
```
/api/audit/logs
```
 | Logs d'audit | Oui |
 **GET** | 
```
/api/user/profile
```
 | Profil utilisateur | Oui |
 **POST** | 
```
/api/user/password
```
 | Changer mot de passe | Oui |
 **POST** | 
```
/api/excel/reload
```
 | Recharger Excel | Oui |
 **GET** | 
```
/api/excel/versions
```
 | Versions Excel | Oui |

### 5.2 Exemples de requêtes/réponses

#### Login

json

Copy

Download

```
// Requête
POST /api/login
{
    "username": "admin",
    "password": "admin123"
}
// Réponse
{
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user\_id": "admin-001",
    "username": "admin",
    "role": "admin"
}
```

#### Exécuter USSD

json

Copy

Download

```
// Requête
POST /api/modules/5/ussd/execute
{
    "ussd\_code": "#122#",
    "input\_data": ""
}
// Réponse
{
    "success": true,
    "result": "💰 SOLDE CRÉDIT\\nSolde principal: 1 500 FCFA",
    "duration": 1234
}
```

#### Envoyer SMS

json

Copy

Download

```
// Requête
POST /api/modules/5/sms/send
{
    "number": "+2250701010101",
    "message": "Test message"
}
// Réponse
{
    "status": "SMS envoyé"
}
```

---

## 6\. WebSocket - Protocole

### 6.1 Événements

 Type | Direction | Description | Payload |
| --- | --- | --- | --- |
 
```
module_connected
```
 | Server → Client | Module connecté | 
```
{module_id, port, imei}
```
 |
 
```
module_disconnected
```
 | Server → Client | Module déconnecté | 
```
{module_id, port}
```
 |
 
```
ussd_result
```
 | Server → Client | Résultat USSD | 
```
{module_id, result, duration}
```
 |
 
```
sms_received
```
 | Server → Client | Nouveau SMS reçu | 
```
{module_id, sender, message}
```
 |
 
```
sms_sent
```
 | Server → Client | SMS envoyé | 
```
{module_id, number, status}
```
 |
 
```
sms_deleted
```
 | Server → Client | SMS supprimé | 
```
{module_id, index}
```
 |
 
```
exploration_progress
```
 | Server → Client | Progression exploration | 
```
{module_id, progress, total}
```
 |
 
```
excel_updated
```
 | Server → Client | Excel mis à jour | 
```
{filename, new_codes}
```
 |

### 6.2 Exemple de message

json

Copy

Download

```
{
    "type": "ussd\_result",
    "module\_id": 5,
    "data": {
        "code": "#122#",
        "result": "💰 SOLDE CRÉDIT\\nSolde principal: 1 500 FCFA",
        "duration": 1234
    },
    "timestamp": "2026-05-22T10:30:00Z"
}
```

---

## 7\. Sécurité

### 7.1 Authentification JWT

text

Copy

Download

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │     │   Serveur   │     │   Base de   │
│             │     │             │     │   données   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │  POST /login      │                   │
       │  (username/pwd)   │                   │
       │──────────────────>│                   │
       │                   │  SELECT user      │
       │                   │──────────────────>│
       │                   │                   │
       │                   │  Vérification     │
       │                   │  bcrypt compare   │
       │                   │                   │
       │  {token}          │  Génération JWT   │
       │<──────────────────│                   │
       │                   │                   │
       │  GET /api/modules │                   │
       │  Authorization:   │                   │
       │  Bearer {token}   │                   │
       │──────────────────>│                   │
       │                   │  Validation JWT   │
       │                   │  Vérification     │
       │                   │  signature        │
       │                   │                   │
       │  \[{modules}\]      │                   │
       │<──────────────────│                   │
       │                   │                   │
```

### 7.2 Roles et permissions

 Rôle | Modules | USSD | SMS | Excel | Audit | Configuration |
| --- | --- | --- | --- | --- | --- | --- |
 **admin** | CRUD | CRUD | CRUD | CRUD | Lecture | Lecture |
 **operator** | Lecture | Exécution | Envoi | Lecture | Non | Non |
 **viewer** | Lecture | Non | Non | Non | Non | Non |

---

## 8\. Communication série - Commandes AT

### 8.1 Commandes implémentées

 Commande | Description | Réponse attendue |
| --- | --- | --- |
 
```
AT
```
 | Test de communication | 
```
OK
```
 |
 
```
AT+CMGF=1
```
 | Mode SMS texte | 
```
OK
```
 |
 
```
AT+CNMI=2,1,0,0,0
```
 | Notification SMS | 
```
OK
```
 |
 
```
AT+CGSN
```
 | Lire IMEI | 
```
861694039371966
```
 |
 
```
AT+CNUM
```
 | Lire numéro | 
```
+2250701010101
```
 |
 
```
AT+CUSD=1,"#122#",15
```
 | Exécuter USSD | 
```
+CUSD: 0,"...",15
```
 |
 
```
AT+CMGS="+225..."
```
 | Envoyer SMS | 
```
+CMGS: <mr>
```
 |
 
```
AT+CMGR=<index>
```
 | Lire SMS | 
```
+CMGR: "REC READ",...
```
 |
 
```
AT+CMGD=<index>
```
 | Supprimer SMS | 
```
OK
```
 |
 
```
AT+CMGL="ALL"
```
 | Lister SMS | 
```
+CMGL: <index>,...
```
 |

### 8.2 Séquence d'initialisation

text

Copy

Download

```
\[Module\] → AT
\[PC\]     → OK
\[Module\] → AT+CMGF=1
\[PC\]     → OK
\[Module\] → AT+CNMI=2,1,0,0,0
\[PC\]     → OK
\[Module\] → AT+CGSN
\[PC\]     → 861694039371966
\[Module\] → AT+CNUM
\[PC\]     → ERROR (ou +CNUM...)
\[Module\] → AT+CUSD=1,"#99#",15
\[PC\]     → +CUSD: 0,"+2250701010101",15
```

---

## 9\. Flux de données

### 9.1 Auto-discovery des modules

Diagram

Code

Copy

Download

Fullscreen

```
sequenceDiagram
    participant UI as Frontend
    participant API as Backend API
    participant Serial as Serial Manager
    participant COM5 as SIM800C COM5
    participant DB as MySQL
    participant WS as WebSocket
    UI\->>API: POST /api/discover
    API\->>Serial: Scan ports (COM5/6/7)
    
    Serial\->>COM5: AT
    COM5\-->>Serial: OK
    Serial\->>COM5: AT+CGSN
    COM5\-->>Serial: 861694039371966
    Serial\->>COM5: AT+CUSD=1,"#99#",15
    COM5\-->>Serial: +2250701010101
    
    Serial\->>DB: Save module
    DB\-->>Serial: OK
    
    Serial\->>WS: Broadcast module\_connected
    WS\-->>UI: WebSocket event
    
    API\-->>UI: 200 OK {modules: 3}
```

### 9.2 Envoi de SMS

Diagram

Code

Copy

Download

Fullscreen

```
sequenceDiagram
    participant UI as Frontend
    participant API as Backend API
    participant SMS as SMS Manager
    participant COM5 as SIM800C
    participant DB as MySQL
    participant WS as WebSocket
    UI\->>API: POST /api/modules/5/sms/send
    API\->>SMS: SendSMS(5, number, message)
    
    SMS\->>DB: Save SMS (status: pending)
    SMS\->>COM5: AT+CMGS="+225..."
    COM5\-->>SMS: >
    SMS\->>COM5: message + ^Z
    COM5\-->>SMS: +CMGS: 15
    
    SMS\->>DB: Update SMS (status: sent)
    SMS\->>WS: Broadcast sms\_sent
    WS\-->>UI: WebSocket event
    
    API\-->>UI: 200 OK
```

---

## 10\. Gestion des erreurs

### 10.1 Codes d'erreur HTTP

 Code | Description | Cas d'utilisation |
| --- | --- | --- |
 200 | Succès | Requête traitée avec succès |
 400 | Requête invalide | Corps JSON mal formé |
 401 | Non authentifié | Token manquant ou invalide |
 403 | Non autorisé | Rôle insuffisant |
 404 | Non trouvé | Module ou ressource inexistante |
 408 | Timeout | Commande USSD sans réponse |
 500 | Erreur interne | Panic ou erreur inattendue |

### 10.2 Gestion des erreurs série

go

Copy

Download

```
type SerialError struct {
    Port     string
    Command  string
    Message  string
    Timeout  time.Duration
    Retry    int
}
func (e \*SerialError) Error() string {
    return fmt.Sprintf("\[%s\] %s: %s (retry %d)", e.Port, e.Command, e.Message, e.Retry)
}
// Stratégie de récupération
func (s \*SIM800C) executeWithRetry(cmd string, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := s.sendCommand(cmd)
        if err \== nil {
            return nil
        }
        if i < maxRetries\-1 {
            time.Sleep(time.Duration(i+1) \* time.Second)
        }
    }
    return &SerialError{Port: s.Port, Command: cmd, Message: "max retries exceeded", Retry: maxRetries}
}
```

---

## 11\. Performance et optimisation

### 11.1 Points de mesure

 Métrique | Cible | Instrumentation |
| --- | --- | --- |
 Latence API | < 100ms | Middleware timing |
 Connexion WebSocket | < 500ms | Handshake time |
 Commande USSD | < 5s | Duration\_ms en DB |
 Envoi SMS | < 3s | Duration\_ms en DB |
 Lecture Excel | < 200ms | Cache + reload |

### 11.2 Optimisations implémentées

go

Copy

Download

```
// Pool de connexions MySQL
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(60 \* time.Minute)
// Cache des codes USSD
type ExcelCache struct {
    data       map\[string\]\[\]USSDCode
    ttl        time.Duration
    lastUpdate time.Time
    mu         sync.RWMutex
}
// File d'attente pour les commandes série
type CommandQueue struct {
    queue    chan Command
    maxSize  int
    workers  int
}
```

---

## 12\. Déploiement

### 12.1 Structure des dossiers de production

text

Copy

Download

```
C:\\xampp\\htdocs\\aa\_Toolbox\\test\_sim800c\\deepseek\\v1\\
├── sim800c-supervisor.exe    # Binaire compilé
├── config.yaml               # Configuration
├── web/                      # Frontend statique
├── storage/
│   ├── excel/               # Fichiers Excel
│   ├── logs/                # Logs application
│   └── backup/              # Sauvegardes
└── scripts/                 # Utilitaires
```

### 12.2 Configuration service Windows

batch

Copy

Download

```
sc create SIM800C\_Supervisor binPath= "C:\\...\\sim800c-supervisor.exe" start= auto
sc failure SIM800C\_Supervisor reset= 86400 actions= restart/5000/restart/10000/restart/30000
```

---

## 13\. Tests

### 13.1 Tests unitaires

go

Copy

Download

```
func TestUSSDValidator(t \*testing.T) {
    v := NewInputValidator()
    
    assert.NoError(t, v.ValidateInput("#144\*81#", "1234"))
    assert.Error(t, v.ValidateInput("#144\*81#", "123"))
    assert.Error(t, v.ValidateInput("#144\*81#", "abcd"))
}
func TestSIM800C\_ExecuteUSSD(t \*testing.T) {
    // Mock du port série
    mockPort := &MockSerialPort{}
    sim := &SIM800C{SerialPort: mockPort, Logger: logrus.New()}
    
    mockPort.On("Write", mock.Anything).Return(0, nil)
    mockPort.On("Read", mock.Anything).Return(\[\]byte("OK"), nil)
    
    result, err := sim.ExecuteUSSD("#122#", "")
    assert.NoError(t, err)
    assert.Contains(t, result, "SOLDE")
}
```

### 13.2 Tests d'intégration

bash

Copy

Download

```
\# Tester l'API
curl \-X POST http://localhost:8082/api/login \-d '{"username":"admin","password":"admin123"}'
\# Tester WebSocket
wscat \-c ws://localhost:8082/api/ws
\# Tester module COM5
curl \-X POST http://localhost:8082/api/modules/5/ussd/execute \-d '{"ussd\_code":"#122#"}'
```

---

## 14\. Maintenance

### 14.1 Logs applicatifs

text

Copy

Download

```
{"level":"info","msg":"Serveur démarré sur le port 8082","time":"2026-05-22T10:00:00Z"}
{"level":"debug","msg":"Requête: GET /api/modules","time":"2026-05-22T10:00:01Z"}
{"level":"info","msg":"Module COM5 connecté","time":"2026-05-22T10:00:02Z"}
{"level":"error","msg":"Erreur commande USSD: timeout","time":"2026-05-22T10:00:05Z"}
```

### 14.2 Sauvegarde

powershell

Copy

Download

```
\# Sauvegarde base de données
mysqldump \-u root sim800c\_manager > backup\_$(Get-Date \-Format yyyyMMdd).sql
\# Sauvegarde fichiers
Compress-Archive \-Path storage\\excel\\\*.xlsx \-DestinationPath backup\_excel\_$(Get-Date \-Format yyyyMMdd).zip
```

---

## 15\. Évolutions futures

 Version | Fonctionnalité | Description |
| --- | --- | --- |
 v2.1 | API REST externe | Exposition des fonctionnalités via API sécurisée |
 v2.2 | Dashboard graphique | Graphiques d'utilisation, statistiques |
 v2.3 | Notifications push | Alertes par email/webhook |
 v2.4 | Multi-utilisateurs | Gestion complète des rôles et permissions |
 v3.0 | Mode cluster | Support de multiples stations SIM800C |

---

\*Document mis à jour le 22/05/2026\*