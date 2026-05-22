### Task 3 - Proposition d'architecture

**Document 1 :** 
```
3-design_system_infrastructure-v1.md
```

markdown

Copy

Download

```
# 3. Design de l'infrastructure système - v1
## 1. Architecture physique et réseau
### 1.1. Nœuds matériels
\`\`\`mermaid
flowchart TB
    subgraph Windows\_Host \[PC Windows\]
        direction LR
        USB\_Hub\[USB Hub 3.0 alimenté\]
        SIM800C\_1\[SIM800C #1<br/>COM5\]
        SIM800C\_2\[SIM800C #2<br/>COM6\]
        SIM800C\_3\[SIM800C #3<br/>COM7\]
        USB\_Hub --> SIM800C\_1 & SIM800C\_2 & SIM800C\_3
    end
    subgraph Application\_Host \[Même PC Windows\]
        Backend\[Backend Go<br/>:8080\]
        MySQL\[MySQL<br/>:3306\]
        Frontend\[Frontend Static<br/>:80 via Nginx/Apache\]
        Excel\_Files\[Excel Files<br/>C:\\storage\\excel\]
    end
    SIM800C\_1 & SIM800C\_2 & SIM800C\_3 -->|Serial over USB| Backend
    Backend -->|Lecture/Écriture| MySQL
    Backend -->|Lecture/Écriture| Excel\_Files
    Frontend -->|WebSocket / API| Backend
```

### 1.2. Topologie réseau

-   **Frontend** : Accessible sur 
    ```
    test_sim800c.local:80
    ```
     (via hosts ou DNS local)
    
-   **Backend API** : Écoute sur 
    ```
    localhost:8080
    ```
     (non exposé directement)
    
-   **Base de données** : Uniquement accessible en localhost
    
-   **WebSocket** : Endpoint 
    ```
    /ws
    ```
     sur le backend pour le temps réel

## 2\. Architecture applicative (composants)

### 2.1. Backend Go - Structure modulaire

text

Copy

Download

```
backend/
├── cmd/
│   └── main.go                 # Point d'entrée
├── internal/
│   ├── config/                 # Configuration (env, yaml)
│   ├── serial/                 # Communication avec SIM800C
│   │   ├── manager.go          # Découverte et gestion ports
│   │   ├── sim800c.go          # Commandes AT spécifiques
│   │   └── ussd\_session.go     # Gestion sessions USSD
│   ├── ussd/                   # Logique USSD
│   │   ├── executor.go         # Exécution codes USSD
│   │   ├── explorer.go         # Exploration menus
│   │   └── validator.go        # Validation entrées (PIN, montant...)
│   ├── sms/                    # Gestion SMS
│   │   ├── manager.go          # CRUD SMS + corbeille
│   │   └── filter.go           # Filtre "Test"
│   ├── excel/                  # Manipulation Excel
│   │   ├── reader.go           # Lecture Codes\_USSD\_CI.xlsx
│   │   ├── writer.go           # Création nouvelles versions
│   │   └── cache.go            # Cache des codes USSD
│   ├── db/                     # MySQL
│   │   ├── models.go           # Structs (SIM, History, SMS...)
│   │   └── repository.go       # Accès données
│   ├── websocket/              # Temps réel
│   │   ├── hub.go              # Gestion connexions clients
│   │   └── client.go           # Client WebSocket
│   └── api/                    # API REST
│       ├── routes.go
│       ├── handlers/           # Handlers par fonctionnalité
│       └── middleware/         # Auth, logs, CORS
├── pkg/                        # Bibliothèques partageables
│   ├── logger/                 # Logging structuré
│   └── errors/                 # Gestion erreurs
└── web/                        # Frontend statique (embedded)
```

### 2.2. Base de données MySQL - Schéma

sql

Copy

Download

```
\-- Base de données: sim800c\_manager
CREATE DATABASE sim800c\_manager;
USE sim800c\_manager;
\-- Modules SIM800C
CREATE TABLE modules (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    com\_port VARCHAR(10) NOT NULL UNIQUE,
    imei VARCHAR(15),
    phone\_number VARCHAR(15),
    carrier VARCHAR(50),
    status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
    last\_seen TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP
);
\-- Historique des commandes USSD
CREATE TABLE ussd\_history (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    module\_id INT NOT NULL,
    ussd\_code VARCHAR(50) NOT NULL,
    input\_data TEXT,
    output\_data TEXT,
    status ENUM('success', 'error', 'timeout') NOT NULL,
    duration\_ms INT,
    executed\_by VARCHAR(50) DEFAULT 'system',
    executed\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    FOREIGN KEY (module\_id) REFERENCES modules(id) ON DELETE CASCADE
);
\-- Messages SMS
CREATE TABLE sms\_messages (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    module\_id INT NOT NULL,
    sender\_number VARCHAR(20),
    receiver\_number VARCHAR(20),
    message TEXT NOT NULL,
    direction ENUM('in', 'out') NOT NULL,
    is\_deleted BOOLEAN DEFAULT FALSE,
    is\_trash BOOLEAN DEFAULT FALSE,  \-- Corbeille automatique
    sms\_index INT,  \-- Index sur la SIM
    received\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    FOREIGN KEY (module\_id) REFERENCES modules(id) ON DELETE CASCADE
);
\-- Corbeille automatique (filtre "Test")
CREATE TABLE sms\_trash\_rules (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    module\_id INT NOT NULL,
    keyword VARCHAR(50) NOT NULL DEFAULT 'Test',
    is\_active BOOLEAN DEFAULT TRUE,
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    FOREIGN KEY (module\_id) REFERENCES modules(id) ON DELETE CASCADE
);
\-- Journal d'audit
CREATE TABLE audit\_log (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    user\_id VARCHAR(50),
    action VARCHAR(100) NOT NULL,
    target\_type VARCHAR(50),
    target\_id INT,
    details JSON,
    ip\_address VARCHAR(45),
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP
);
\-- Versions du fichier Excel (pour traçabilité)
CREATE TABLE excel\_versions (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    version\_date TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,
    created\_by VARCHAR(50) DEFAULT 'system',
    new\_codes\_count INT DEFAULT 0
);
\-- Favoris USSD (utilisateurs)
CREATE TABLE ussd\_favorites (
    id INT AUTO\_INCREMENT PRIMARY KEY,
    user\_id VARCHAR(50) NOT NULL,
    ussd\_code\_id INT,  \-- Référence à l'ID du code dans l'Excel (pas de FK pour flexibilité)
    ussd\_code VARCHAR(50) NOT NULL,
    carrier VARCHAR(50),
    operation VARCHAR(100),
    created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP
);
\-- Index pour performance
CREATE INDEX idx\_modules\_status ON modules(status);
CREATE INDEX idx\_ussd\_history\_module ON ussd\_history(module\_id);
CREATE INDEX idx\_sms\_module\_date ON sms\_messages(module\_id, received\_at);
CREATE INDEX idx\_audit\_user ON audit\_log(user\_id, created\_at);
```

### 2.3. Frontend - Structure recommandée

text

Copy

Download

```
frontend/
├── index.html                 # Page principale
├── css/
│   ├── main.css               # Styles communs
│   ├── theme-light.css        # Thème clair
│   └── theme-dark.css         # Thème sombre
├── js/
│   ├── app.js                 # Initialisation
│   ├── websocket.js           # Connexion WS temps réel
│   ├── modules.js             # Gestion modules SIM800C
│   ├── ussd.js                # USSD Manager, Menu Discovery
│   ├── sms.js                 # SMS Manager
│   ├── dashboard.js           # Dashboard et auto-refresh
│   └── theme.js               # Gestion thème clair/sombre
├── lib/
│   └── chart.js               # Graphiques KPI (optionnel)
└── assets/
    ├── icons/                 # Icônes modules, opérateurs
    └── fonts/
```

## 3\. Flux de communication

### 3.1. Séquence auto-discovery

Diagram

Code

Copy

Download

Fullscreen

```
sequenceDiagram
    participant Frontend
    participant Backend
    participant SerialManager
    participant SIM800C\_COM5
    Frontend\->>Backend: GET /api/discover
    Backend\->>SerialManager: Scan COM ports (5-7)
    SerialManager\->>SIM800C\_COM5: AT (check alive)
    SIM800C\_COM5\-->>SerialManager: OK
    SerialManager\->>SIM800C\_COM5: AT+CUSD=1,"#99#",15
    SIM800C\_COM5\-->>SerialManager: +CUSD: 0,"+225XXXXXXXXXX",15
    SerialManager\->>Backend: Module info (IMEI, numéro)
    Backend\->>MySQL: INSERT/UPDATE modules
    Backend\-->>Frontend: WebSocket: module\_update
```

### 3.2. Flux temps réel

-   **WebSocket** : Bi-directionnel pour événements (nouveau module, résultat USSD, SMS reçu)
    
-   **API REST** : Commandes déclenchées par utilisateur (exécuter USSD, envoyer SMS)
    
-   **Polling optionnel** : Fallback si WebSocket non supporté

## 4\. Sécurité

### 4.1. Authentification (recommandée)

-   JWT (JSON Web Token) pour API REST
    
-   Stockage token en 
    ```
    httpOnly
    ```
     cookie ou localStorage
    
-   Session WebSocket authentifiée via token dans l'URL ou header

### 4.2. Chiffrement

-   AES-256 pour PIN et codes de recharge en base
    
-   TLS/SSL pour communication frontend-backend (si exposé en réseau)

### 4.3. Contrôle d'accès (RBAC)

-   **Admin** : Toutes fonctionnalités + configuration
    
-   **Opérateur** : Exécution USSD, SMS, dashboard
    
-   **Lecteur** : Consultation uniquement

## 5\. Scalabilité

-   **Horizontal scaling limité** : Un PC Windows hôte unique.
    
-   **Vertical scaling** : Optimisation Go (goroutines, pool connexions).
    
-   **Future évolution** : Possibilité de déporter le backend sur un serveur Linux et de communiquer avec les modules via TCP/série distante (ser2net).

## 6\. Performance

-   **Pool worker** : 1 goroutine par module SIM800C pour éviter les blocages.
    
-   **File d'attente** : Buffer 100 commandes par module.
    
-   **Cache** : Codes USSD chargés depuis Excel au démarrage, rechargement périodique (toutes les 5min).
    
-   **Index MySQL** : Optimisés pour les requêtes fréquentes.

## 7\. Haute disponibilité

-   **Service Windows** : Backend et frontend tournant comme services Windows (re-démarrage auto en cas de crash).
    
-   **Watchdog** : Surveillance de l'état des modules avec reconnexion automatique.
    
-   **Sauvegarde** : Backup quotidien de la base MySQL et des fichiers Excel.