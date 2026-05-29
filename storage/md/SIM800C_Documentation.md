# SIM800C Documentation
## Architecture Frontend Web + Backend GoLang + MySQL
### Gestion des fonctions USSD et SMS avec les modules SIM800C

Version: 1.0  
Date: 2026-05-27

---

# 1. Objectif du projet

Cette documentation décrit l’architecture, les bonnes pratiques, les commandes AT, les workflows, les modèles de données et les mécanismes logiciels nécessaires pour construire une plateforme complète de gestion de modules SIM800C.

Le système cible comprend :

- Frontend Web
- Backend API GoLang
- Base de données MySQL
- Gestion série UART/USB
- Gestion USSD
- Gestion SMS
- Gestion multi-modems
- Gestion des sessions et files d’attente
- Monitoring des modules SIM800C

---

# 2. Architecture globale

```text
┌────────────────────┐
│ Frontend Web       │
│ React/Vue/HTML     │
└─────────┬──────────┘
          │ HTTP/REST/WebSocket
┌─────────▼──────────┐
│ Backend GoLang     │
│ API + Workers      │
└─────────┬──────────┘
          │
 ┌────────▼─────────┐
 │ MySQL Database   │
 └────────┬─────────┘
          │
 ┌────────▼─────────┐
 │ Serial Manager   │
 │ SIM800C Drivers  │
 └────────┬─────────┘
          │ UART/USB
 ┌────────▼─────────┐
 │ SIM800C Modules  │
 └──────────────────┘
```

---

# 3. Cas d’utilisation principaux

## USSD

- Consultation de solde
- Achat de forfait
- Mobile Money
- Menus interactifs
- Vérification statut SIM

## SMS

- Envoi SMS
- Réception SMS
- Broadcast SMS
- OTP
- Notifications
- Archivage messages

---

# 4. Architecture logicielle recommandée

## Backend GoLang

### Modules recommandés

| Module | Rôle |
|---|---|
| api | API REST |
| serial | Communication UART |
| modem | Gestion modem |
| ussd | Gestion sessions USSD |
| sms | Gestion SMS |
| worker | Files d’attente |
| database | Accès MySQL |
| websocket | Temps réel |
| scheduler | Tâches automatiques |

---

# 5. Architecture Backend détaillée

```text
cmd/
internal/
    api/
    serial/
    modem/
    ussd/
    sms/
    queue/
    database/
    websocket/
pkg/
configs/
```

---

# 6. Communication série

## Interfaces supportées

- USB UART CH340
- CH341SER
- FTDI
- TTL UART

## Paramètres recommandés

| Paramètre | Valeur |
|---|---|
| Baudrate | 115200 |
| Data bits | 8 |
| Stop bits | 1 |
| Parity | None |
| Flow Control | RTS/CTS |

---

# 7. Initialisation SIM800C

## Séquence recommandée

```text
AT
ATE0
AT+CMEE=2
AT+CPIN?
AT+CREG?
AT+CSQ
AT+CSCS="GSM"
AT+CMGF=1
AT+CNMI=2,1,0,0,0
```

---

# 8. Gestion des erreurs

## Activer erreurs détaillées

```text
AT+CMEE=2
```

## Types d’erreurs

| Type | Description |
|---|---|
| ERROR | Erreur générique |
| +CME ERROR | Erreur modem |
| +CMS ERROR | Erreur SMS |

---

# 9. Gestion des modules multiples

## Problème principal

Un modem SIM800C ne doit pas recevoir plusieurs commandes simultanément.

## Solution

Utiliser :

- mutex
- file FIFO
- worker dédié par modem

---

# 10. Architecture Worker recommandée

```text
1 modem = 1 goroutine dédiée
```

## Exemple

```text
SIM800C_1 -> Worker_1
SIM800C_2 -> Worker_2
SIM800C_3 -> Worker_3
```

---

# 11. Gestion USSD

## Commande principale

```text
AT+CUSD
```

---

# 12. Envoyer une requête USSD

## Exemple

```text
AT+CUSD=1,"*111#",15
```

---

# 13. Réponses USSD

## Format

```text
+CUSD: <m>,"message",<dcs>
```

---

# 14. États USSD

| Valeur | Description |
|---|---|
| 0 | Réponse finale |
| 1 | Session ouverte |
| 2 | Session terminée |
| 4 | Opération non supportée |

---

# 15. Workflow USSD

```text
Client Web
    ↓
API REST
    ↓
Queue USSD
    ↓
Worker Modem
    ↓
SIM800C
    ↓
Réponse réseau
    ↓
Parsing
    ↓
Base MySQL
    ↓
WebSocket Frontend
```

---

# 16. Gestion des sessions USSD

## Important

Les sessions USSD sont :

- asynchrones
- temporisées
- interactives

## Recommandation

Créer une table :

```sql
ussd_sessions
```

---

# 17. Structure table ussd_sessions

```sql
CREATE TABLE ussd_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    modem_id BIGINT,
    session_id VARCHAR(100),
    phone_number VARCHAR(30),
    ussd_code VARCHAR(50),
    current_menu TEXT,
    session_state VARCHAR(30),
    network_state INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

---

# 18. Parsing des réponses USSD

## Exemple

```text
+CUSD: 1,"1: Solde\n2: Internet",15
```

## Parser :

- état session
- contenu
- encodage

---

# 19. Gestion UCS2

## Certains opérateurs utilisent UCS2

Exemple :

```text
0042006F006E006A006F0075
```

## Solution

- HEX → bytes
- UTF16-BE → UTF8

---

# 20. Timeout USSD

## Recommandation

| Action | Timeout |
|---|---|
| Réponse menu | 10 sec |
| Session complète | 60 sec |
| Commande AT | 5 sec |

---

# 21. Gestion SMS

## Mode texte recommandé

```text
AT+CMGF=1
```

---

# 22. Envoyer un SMS

## Étape 1

```text
AT+CMGS="+2250700000000"
```

## Étape 2

```text
Bonjour
```

## Étape 3

Envoyer CTRL+Z

---

# 23. Réception SMS

## Configuration

```text
AT+CNMI=2,1,0,0,0
```

## URC reçu

```text
+CMTI: "SM",1
```

---

# 24. Lire SMS

```text
AT+CMGR=1
```

---

# 25. Supprimer SMS

```text
AT+CMGD=1
```

---

# 26. Architecture SMS

```text
SMS Queue
    ↓
SMS Worker
    ↓
SIM800C
    ↓
Delivery Status
```

---

# 27. Table sms_messages

```sql
CREATE TABLE sms_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    modem_id BIGINT,
    direction VARCHAR(10),
    phone_number VARCHAR(30),
    message TEXT,
    status VARCHAR(30),
    provider_message_id VARCHAR(100),
    created_at TIMESTAMP
);
```

---

# 28. États SMS recommandés

| État | Description |
|---|---|
| pending | En attente |
| sending | En cours |
| sent | Envoyé |
| delivered | Livré |
| failed | Échec |

---

# 29. Gestion ports COM

## Windows

```text
COM3
COM4
COM5
```

## Linux

```text
/dev/ttyUSB0
/dev/ttyUSB1
```

---

# 30. Détection automatique modems

## Vérification

```text
ATI
AT+GSN
AT+CIMI
AT+CCID
```

---

# 31. Informations importantes modem

| Commande | Description |
|---|---|
| ATI | Modèle |
| AT+GSN | IMEI |
| AT+CIMI | IMSI |
| AT+CCID | ICCID |

---

# 32. Gestion état réseau

## Vérification réseau

```text
AT+CREG?
```

## Signal

```text
AT+CSQ
```

---

# 33. Interprétation CSQ

| Valeur | Qualité |
|---|---|
| 0-9 | Mauvais |
| 10-14 | Moyen |
| 15-31 | Bon |

---

# 34. Architecture Frontend

## Fonctionnalités

- Dashboard modems
- Monitoring temps réel
- Console AT
- Gestion SMS
- Gestion USSD
- Logs
- Alertes

---

# 35. Dashboard recommandé

## Informations temps réel

- État modem
- Signal GSM
- Réseau
- SIM présente
- SMS envoyés
- Sessions USSD

---

# 36. WebSocket

## Utilisation recommandée

Temps réel pour :

- réponses USSD
- SMS entrants
- changement état modem
- alertes

---

# 37. API REST recommandée

## Endpoints

### Modems

```text
GET /api/modems
POST /api/modems
```

### SMS

```text
POST /api/sms/send
GET /api/sms
```

### USSD

```text
POST /api/ussd/send
POST /api/ussd/respond
```

---

# 38. Exemple payload USSD

```json
{
  "modem_id": 1,
  "code": "*111#"
}
```

---

# 39. Exemple réponse USSD

```json
{
  "session_id": "abc123",
  "state": 1,
  "message": "1: Solde\n2: Internet"
}
```

---

# 40. Architecture base de données

## Tables principales

| Table | Description |
|---|---|
| modems | Modules SIM800C |
| sms_messages | SMS |
| ussd_sessions | Sessions USSD |
| ussd_logs | Logs USSD |
| modem_logs | Logs techniques |
| users | Utilisateurs |

---

# 41. Logs recommandés

## Toujours logger :

- commandes AT
- réponses modem
- erreurs
- timeouts
- reconnexions

---

# 42. Gestion reconnexion

## Cas fréquents

- USB déconnecté
- modem freeze
- perte réseau
- SIM absente

## Solution

- watchdog
- reconnexion automatique
- retry queue

---

# 43. Machine d’état modem

## États recommandés

```text
DISCONNECTED
CONNECTING
READY
BUSY
ERROR
RECONNECTING
```

---

# 44. Sécurité

## Recommandations

- authentification API
- JWT
- HTTPS
- rate limiting
- audit logs

---

# 45. Haute disponibilité

## Recommandations

- workers isolés
- retry automatique
- persistence queue
- monitoring

---

# 46. Performance

## Recommandations

- max 1 commande AT active par modem
- pooling workers
- batching SMS
- timeout strict

---

# 47. Monitoring

## Métriques importantes

| Métrique | Description |
|---|---|
| SMS/min | Débit |
| USSD success rate | Taux succès |
| CSQ moyen | Qualité réseau |
| modem uptime | Disponibilité |

---

# 48. Monitoring Prometheus

## Recommandé

- Prometheus
- Grafana

---

# 49. Bibliothèques GoLang recommandées

| Librairie | Usage |
|---|---|
| tarm/serial | UART |
| gorilla/websocket | WebSocket |
| gin | API REST |
| gorm | ORM MySQL |
| zap | Logging |

---

# 50. Architecture concurrente GoLang

## Utiliser :

- goroutines
- channels
- mutex
- context timeout

---

# 51. Exemple architecture Go

```text
API Request
    ↓
Channel Queue
    ↓
Worker Goroutine
    ↓
Serial Port
    ↓
SIM800C
```

---

# 52. Gestion asynchrone

## Important

Les réponses SIM800C sont :

- asynchrones
- non bloquantes
- parfois retardées

---

# 53. URC importants

## Exemples

```text
+CMTI
+CUSD
+CREG
+CMT
```

---

# 54. Parsing UART

## Important

Utiliser :

- buffer circulaire
- parser ligne
- gestion CRLF

---

# 55. Flow Control

## Recommandé

RTS/CTS

Commande :

```text
AT+IFC=2,2
```

---

# 56. Sauvegarde configuration

```text
AT&W
```

---

# 57. Gestion alimentation

## Très important

Le SIM800C nécessite :

- 4V stable
- pics jusqu’à 2A

---

# 58. Causes fréquentes de problèmes

| Cause | Symptôme |
|---|---|
| alimentation faible | reset modem |
| mauvais signal | timeout |
| USB instable | déconnexion |
| commandes parallèles | corruption réponses |

---

# 59. Recommandations production

## Utiliser :

- hub USB alimenté
- watchdog matériel
- logs persistants
- supervision

---

# 60. Références importantes

## Documentation officielle SIM800

- SIM800 Series AT Command Manual V1.10

## Sections importantes

| Section | Sujet |
|---|---|
| 3.2.53 | AT+CUSD |
| 4.2.5 | AT+CMGS |
| 4.2.4 | AT+CMGR |
| 4.2.8 | AT+CNMI |
| 3.2.32 | AT+CREG |
| 3.2.35 | AT+CSQ |

---

# 61. Conclusion

Pour construire une plateforme robuste de gestion SIM800C :

- isoler chaque modem dans un worker dédié
- gérer proprement les timeouts
- parser les URC asynchrones
- centraliser les logs
- utiliser une architecture concurrente GoLang
- protéger les accès API
- surveiller alimentation et qualité réseau

Le point critique du système reste :

```text
1 modem = 1 file de commandes = 1 worker dédié
```

afin d’éviter toute corruption des échanges UART.

