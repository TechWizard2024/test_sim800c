# SIM800C — Documentation Technique Avancée
## Gestion complète : USSD, SMS, UART, GSM, Encodages

**Version :** 2.0  
**Référence :** SIM800 Series AT Command Manual V1.10 (SIMCom, 2016-10-20)  
**Date :** 2026-05-27

---

## Table des matières

1. [Introduction et architecture](#1-introduction-et-architecture)
2. [Communication UART](#2-communication-uart)
3. [Initialisation du module](#3-initialisation-du-module)
4. [Vérification réseau et SIM](#4-vérification-réseau-et-sim)
5. [Gestion des SMS — Mode Texte](#5-gestion-des-sms--mode-texte)
6. [Gestion des SMS — Détails avancés](#6-gestion-des-sms--détails-avancés)
7. [Gestion USSD](#7-gestion-ussd)
8. [Encodage des caractères](#8-encodage-des-caractères)
9. [Réponses asynchrones (URC)](#9-réponses-asynchrones-urc)
10. [Gestion des erreurs](#10-gestion-des-erreurs)
11. [Gestion alimentation et stabilité](#11-gestion-alimentation-et-stabilité)
12. [Bonnes pratiques et séquences complètes](#12-bonnes-pratiques-et-séquences-complètes)
13. [Tableau de référence rapide](#13-tableau-de-référence-rapide)

---

## 1. Introduction et architecture

### 1.1 Le module SIM800C

Le SIM800C est un modem GSM/GPRS 2G compact développé par SIMCom. Il expose une interface série UART pour recevoir des commandes AT et envoyer des réponses ou notifications asynchrones. Il supporte :

- **SMS** (envoi, réception, stockage, suppression)
- **USSD** (requêtes de service non structurées — solde, menus opérateur)
- **Appels vocaux**
- **GPRS / TCP-IP / HTTP / FTP**
- **Commandes AT conformes 3GPP TS 27.007 et 27.005**

### 1.2 Architecture de communication

```
Application (MCU / Raspberry Pi / PC)
          ↓  (UART TX/RX + RTS/CTS)
        SIM800C
          ↓
      Réseau GSM (opérateur)
```

Chaque échange suit un modèle **requête → réponse** :
1. L'application envoie une commande AT terminée par `\r` (CR, 0x0D)
2. Le modem répond par une ou plusieurs lignes terminées par `\r\n` (CRLF)
3. La réponse finale est toujours `OK` ou `ERROR` / `+CME ERROR` / `+CMS ERROR`
4. Des réponses spontanées (**URC**) peuvent arriver à tout moment, sans qu'une commande ait été envoyée

---

## 2. Communication UART

### 2.1 Paramètres recommandés

| Paramètre | Valeur recommandée |
|---|---|
| Baudrate | 115200 bps |
| Data bits | 8 |
| Stop bits | 1 |
| Parity | None |
| Flow control | **RTS/CTS (matériel)** |

> **Pourquoi RTS/CTS ?** Le SIM800C peut générer des pics de trafic UART lors de la réception simultanée de SMS ou de réponses USSD longues. Sans contrôle de flux matériel, des octets peuvent être perdus. La commande pour activer le flow control matériel est :

```text
AT+IFC=2,2
```

Après configuration, sauvegarder avec `AT&W`.

### 2.2 Gestion des buffers UART

Le module peut envoyer des données :
- en retard (délai réseau GSM)
- en plusieurs fragments (longues réponses)
- de manière asynchrone (URC non sollicités)

**Recommandations d'implémentation :**

```
Buffer circulaire (≥ 512 octets)
  → Parser ligne par ligne sur \r\n
  → Dispatcher : réponse de commande vs URC
  → Timeout par type d'action (voir section 9)
```

Ne jamais envoyer deux commandes AT simultanément. Attendre toujours `OK` ou `ERROR` avant d'envoyer la suivante.

### 2.3 Timeout recommandés

| Action | Timeout conseillé |
|---|---|
| Commande AT simple | 5 secondes |
| Envoi SMS (`AT+CMGS`) | 60 secondes |
| Lecture SMS (`AT+CMGR`) | 5 secondes |
| Requête USSD (`AT+CUSD`) | 30 secondes |
| Enregistrement réseau GSM | 60 secondes |
| Reset logiciel (`AT+CFUN=1,1`) | 15 secondes (attendre `+CPIN: READY`) |

---

## 3. Initialisation du module

### 3.1 Séquence d'initialisation complète

Exécuter cette séquence dans l'ordre exact au démarrage ou après reset :

```text
AT              → Vérifie que le modem répond (attendre OK)
ATE0            → Désactive l'écho des commandes
AT+CMEE=2       → Active les messages d'erreur en clair (verbose)
AT+CPIN?        → Vérifie que la SIM est prête
AT+CREG?        → Vérifie l'enregistrement réseau
AT+CSQ          → Vérifie le niveau de signal
AT+CSCS="GSM"  → Définit l'encodage caractères (GSM 7-bit par défaut)
AT+CMGF=1       → Mode SMS texte
AT+CNMI=2,1,0,0,0 → Active les notifications SMS entrants
AT+IFC=2,2      → Flow control RTS/CTS
AT&W            → Sauvegarde la configuration
```

### 3.2 Désactiver l'écho — `ATE0`

Par défaut, le modem renvoie chaque caractère reçu (écho). Cela pollue le flux UART. `ATE0` désactive cet écho.

```text
ATE0
→ OK
```

Pour réactiver l'écho (debug) : `ATE1`

### 3.3 Activer les erreurs détaillées — `AT+CMEE=2`

Sans cette commande, les erreurs renvoient juste `ERROR`. Avec `AT+CMEE=2`, le modem renvoie un message explicite :

```text
AT+CMEE=2
→ OK

→ Exemple d'erreur sans CMEE=2 :
ERROR

→ Exemple avec CMEE=2 :
+CME ERROR: SIM not inserted
```

Les trois niveaux :
- `AT+CMEE=0` : ERROR simple (par défaut)
- `AT+CMEE=1` : Code numérique (+CME ERROR: 10)
- `AT+CMEE=2` : Message explicite (+CME ERROR: SIM not inserted)

### 3.4 Informations du modem

| Commande | Information retournée |
|---|---|
| `ATI` | Version firmware et modèle |
| `AT+GSN` | IMEI (identifiant unique du modem) |
| `AT+CIMI` | IMSI (identifiant de la SIM) |
| `AT+CCID` | ICCID (numéro de série SIM) |

Exemple :
```text
ATI
→ SIMCom_Ltd
  SIMCom_SIM800C
  Revision: 1308B01SIM800C32
  OK
```

---

## 4. Vérification réseau et SIM

### 4.1 État de la SIM — `AT+CPIN?`

```text
AT+CPIN?
→ +CPIN: READY
   OK
```

| Réponse | Signification |
|---|---|
| `+CPIN: READY` | SIM présente, aucun PIN requis |
| `+CPIN: SIM PIN` | PIN requis — envoyer `AT+CPIN="1234"` |
| `+CPIN: SIM PUK` | PUK requis (trop d'essais PIN) |
| `+CME ERROR: SIM not inserted` | Aucune SIM détectée |

### 4.2 Enregistrement réseau — `AT+CREG?`

```text
AT+CREG?
→ +CREG: 0,1
   OK
```

Format de la réponse : `+CREG: <n>,<stat>`

| `<stat>` | Signification |
|---|---|
| 0 | Non enregistré, pas de recherche |
| 1 | Enregistré sur réseau domestique ✅ |
| 2 | Pas enregistré, en recherche... |
| 3 | Enregistrement refusé |
| 4 | Inconnu |
| 5 | Enregistré en itinérance (roaming) ✅ |

> Les valeurs `1` et `5` indiquent que le module est opérationnel sur le réseau GSM.

### 4.3 Niveau de signal — `AT+CSQ`

```text
AT+CSQ
→ +CSQ: 18,0
   OK
```

Format : `+CSQ: <rssi>,<ber>`

| `<rssi>` | Qualité du signal |
|---|---|
| 0 | −113 dBm ou moins (très mauvais) |
| 1–9 | Mauvais |
| 10–14 | Moyen |
| 15–19 | Correct |
| 20–31 | Bon à excellent |
| 99 | Non détectable |

`<ber>` est le taux d'erreur bit (0 = excellent, 99 = non mesurable). En pratique, surveiller uniquement `<rssi>` : en dessous de 10, les SMS et USSD peuvent échouer.

### 4.4 Vérifier et sélectionner l'opérateur

```text
AT+COPS?           → Opérateur actuel
AT+COPS=0          → Sélection automatique de l'opérateur
AT+CGATT?          → Vérifier l'attachement GPRS
```

---

## 5. Gestion des SMS — Mode Texte

### 5.1 Choisir le mode SMS — `AT+CMGF`

Le SIM800C supporte deux modes SMS :
- **Mode texte** (`AT+CMGF=1`) : format lisible, recommandé pour la plupart des usages
- **Mode PDU** (`AT+CMGF=0`) : format binaire brut, nécessaire pour les SMS multi-octets avancés

```text
AT+CMGF=1     → Mode texte
→ OK

AT+CMGF?      → Lire le mode actuel
→ +CMGF: 1
   OK
```

### 5.2 Envoyer un SMS — `AT+CMGS`

**Processus en 3 étapes :**

**Étape 1 — Indiquer le numéro destinataire**

```text
AT+CMGS="+2250700000000"
→ >
```

Le modem répond par `>` (prompt), signifiant qu'il attend le corps du message.

**Étape 2 — Saisir le texte du message**

Envoyer le texte directement sur le port série, **sans retour à la ligne** :

```text
Bonjour, votre commande a été confirmée.
```

**Étape 3 — Terminer l'envoi avec CTRL+Z (ASCII 26, 0x1A)**

```text
[0x1A]
→ +CMGS: 25
   OK
```

La réponse `+CMGS: <mr>` indique la référence du message envoyé (`<mr>` = Message Reference, entier).

> Pour annuler l'envoi en cours, envoyer ESC (ASCII 27, 0x1B) à la place de CTRL+Z.

**Exemple complet :**

```text
AT+CMGS="+2250700000000"
> Bonjour[0x1A]

+CMGS: 25
OK
```

**Limites importantes :**
- En mode texte 7-bit GSM : maximum **160 caractères** par SMS
- En mode UCS2 (caractères spéciaux/accents) : maximum **70 caractères** par SMS
- Le modem rejette les appels entrants pendant l'envoi d'un SMS

### 5.3 Recevoir un SMS — `AT+CNMI`

`AT+CNMI` configure la manière dont le modem notifie l'application d'un nouveau SMS entrant.

```text
AT+CNMI=<mode>,<mt>,<bm>,<ds>,<bfr>
```

**Configuration recommandée :**

```text
AT+CNMI=2,1,0,0,0
→ OK
```

| Paramètre | Valeur | Signification |
|---|---|---|
| `<mode>` | 2 | Bufferise les URCs si UART occupé, puis les vide |
| `<mt>` | 1 | Notifie par URC `+CMTI` avec index de stockage |
| `<bm>` | 0 | Pas de notification Cell Broadcast |
| `<ds>` | 0 | Pas de notification de statut de livraison |
| `<bfr>` | 0 | Vide le buffer lors du passage en mode 2 |

**Détail du paramètre `<mt>` :**

| `<mt>` | Comportement |
|---|---|
| 0 | Aucune notification |
| 1 | URC `+CMTI: "SM",<index>` — SMS stocké, notifie l'index |
| 2 | URC `+CMT: ...` — SMS livré directement au TE (non stocké) |
| 3 | Classe 3 directs, autres → stockés |

### 5.4 Notification SMS entrant — URC `+CMTI`

Quand un SMS arrive avec `<mt>=1`, le modem envoie spontanément :

```text
+CMTI: "SM",3
```

- `"SM"` : stocké en mémoire SIM
- `3` : index de l'emplacement (à utiliser avec `AT+CMGR`)

### 5.5 Lire un SMS — `AT+CMGR`

```text
AT+CMGR=<index>[,<mode>]
```

```text
AT+CMGR=3
→ +CMGR: "REC UNREAD","+2250700000000",,"26/05/27,10:15:00+00"
   Votre solde est insuffisant.
   OK
```

Format de la réponse en mode texte :

```text
+CMGR: <stat>,<oa>[,<alpha>],<scts>[,<tooa>,<fo>,<pid>,<dcs>,<sca>,<tosca>,<length>]
<data>
```

| Champ | Description |
|---|---|
| `<stat>` | État : "REC UNREAD", "REC READ", "STO UNSENT", "STO SENT" |
| `<oa>` | Numéro de l'expéditeur (Originating Address) |
| `<scts>` | Horodatage du message |
| `<data>` | Corps du message (ligne suivante) |

> Le paramètre optionnel `<mode>=1` lit le SMS **sans changer son statut** (reste "REC UNREAD"). Utile pour les systèmes qui veulent traiter le SMS plus tard.

### 5.6 Lister les SMS — `AT+CMGL`

```text
AT+CMGL="<stat>"
```

| Valeur `<stat>` | Signification |
|---|---|
| `"REC UNREAD"` | SMS reçus non lus |
| `"REC READ"` | SMS reçus lus |
| `"STO UNSENT"` | SMS rédigés non envoyés |
| `"STO SENT"` | SMS envoyés |
| `"ALL"` | Tous les SMS |

Exemple :
```text
AT+CMGL="ALL"
→ +CMGL: 1,"REC READ","+2250701234567",,"26/05/20,09:00:00+00"
   Votre recharge de 500 FCFA a été effectuée.

   +CMGL: 2,"REC UNREAD","+2250712345678",,"26/05/27,10:15:00+00"
   Nouveau message de votre banque.

   OK
```

> Lister avec `"REC UNREAD"` marque automatiquement les messages comme lus (`"REC READ"`). Utiliser `AT+CMGL="REC UNREAD",1` (avec `<mode>=1`) pour lire sans changer le statut.

### 5.7 Supprimer un SMS — `AT+CMGD`

```text
AT+CMGD=<index>[,<delflag>]
```

| `<delflag>` | Action |
|---|---|
| 0 (défaut) | Supprime uniquement le SMS à `<index>` |
| 1 | Supprime tous les SMS lus |
| 2 | Supprime tous les SMS lus + envoyés |
| 3 | Supprime tous les SMS lus + envoyés + non envoyés |
| 4 | Supprime **tous** les SMS (y compris non lus) |

Exemples :
```text
AT+CMGD=3         → Supprime SMS à l'index 3
AT+CMGD=1,4       → Supprime TOUS les SMS (peu importe l'index 1 ici)
```

**Alternative pour supprimer tout :**
```text
AT+CMGDA="DEL ALL"
→ OK
```

> Attention : `AT+CMGDA` peut prendre jusqu'à 25 secondes pour 150 messages.

### 5.8 Écrire un SMS en mémoire sans envoyer — `AT+CMGW`

Utile pour stocker un SMS à envoyer plus tard :

```text
AT+CMGW="+2250700000000"
> Message à envoyer plus tard[0x1A]
→ +CMGW: 5
   OK
```

L'index retourné (`5`) peut être utilisé avec `AT+CMSS=5` pour envoyer depuis le stockage.

### 5.9 Envoyer un SMS depuis la mémoire — `AT+CMSS`

```text
AT+CMSS=5
→ +CMSS: 26
   OK
```

---

## 6. Gestion des SMS — Détails avancés

### 6.1 Centre de service SMS — `AT+CSCA`

Si les SMS ne partent pas, vérifier que le SMSC (numéro du centre de service) est configuré :

```text
AT+CSCA?
→ +CSCA: "+22500",145
   OK
```

Si le SMSC est vide ou incorrect :
```text
AT+CSCA="+22500",145
→ OK
```

Le numéro du SMSC est fourni par l'opérateur. En Côte d'Ivoire, les SMSC courants : MTN (+22500), Orange (+2250), Moov (+2250).

### 6.2 Paramètres du mode texte — `AT+CSMP`

Pour configurer la validité et le type d'encodage des SMS en mode texte :

```text
AT+CSMP=17,167,0,0
```

| Paramètre | Valeur | Signification |
|---|---|---|
| `<fo>` | 17 | SMS-SUBMIT standard |
| `<vp>` | 167 | Validité : 24h |
| `<pid>` | 0 | Protocole standard |
| `<dcs>` | 0 | GSM 7-bit (alphabet par défaut) |

Pour envoyer en UCS2 (caractères spéciaux, accents) :
```text
AT+CSMP=17,167,0,8
```

### 6.3 Afficher les paramètres texte — `AT+CSDH`

Pour voir tous les détails dans les réponses `AT+CMGR` et `AT+CMGL` :

```text
AT+CSDH=1     → Affiche les paramètres détaillés
AT+CSDH=0     → Masque les paramètres (réponse compacte)
```

---

## 7. Gestion USSD

### 7.1 Introduction USSD

L'USSD (Unstructured Supplementary Service Data) permet d'envoyer des codes courts à l'opérateur (ex: `*111#`) pour consulter un solde, acheter un forfait, ou naviguer dans un menu interactif. Le dialogue USSD est **synchrone** : le modem envoie la requête au réseau et attend une réponse.

**Différences USSD vs SMS :**
- USSD est **temps réel** (réponse en quelques secondes)
- USSD ne laisse pas de trace dans la mémoire SIM
- USSD peut ouvrir une **session interactive** (menu multi-niveaux)
- L'encodage peut varier : GSM 7-bit ou UCS2 selon l'opérateur

### 7.2 Commande principale — `AT+CUSD`

```text
AT+CUSD=<n>[,"<str>",<dcs>]
```

| Paramètre | Valeur | Signification |
|---|---|---|
| `<n>` | 0 | Désactive la présentation du résultat en TE |
| `<n>` | 1 | **Active la présentation + envoie la requête USSD** |
| `<n>` | 2 | Annule la session USSD en cours |
| `<str>` | — | Chaîne USSD à envoyer (ex: `"*111#"`) |
| `<dcs>` | 15 | DCS (Data Coding Scheme) : 15 = GSM par défaut |

> **Note officielle SIMCom :** Si l'USSD n'est pas supporté par le réseau ou retourne une erreur, le modem renvoie `+CUSD: 4`.

### 7.3 Format de la réponse USSD

```text
+CUSD: <m>,"<message>",<dcs>
```

| `<m>` | Signification |
|---|---|
| 0 | Réponse finale — session terminée par le réseau |
| 1 | Session ouverte — le réseau attend une réponse |
| 2 | Session terminée par le mobile |
| 4 | Opération non supportée ou erreur réseau |

### 7.4 Cas 1 — Requête USSD simple (réponse directe)

Exemple : consulter le solde

```text
AT+CUSD=1,"*111#",15
→ OK

[Après quelques secondes, le modem reçoit la réponse réseau :]
+CUSD: 0,"Votre solde est 1250 FCFA. Validite: 30/06/2026",15
```

- `<m>=0` : la session est close, c'est une réponse finale
- Le message est la chaîne retournée par l'opérateur

### 7.5 Cas 2 — Menu USSD interactif

Certains codes USSD ouvrent un menu à plusieurs niveaux.

**Envoi de la requête initiale :**
```text
AT+CUSD=1,"*200#",15
→ OK

+CUSD: 1,"1: Consulter solde\n2: Acheter forfait\n3: Transfert credit",15
```

- `<m>=1` : session ouverte, le réseau attend un choix

**Répondre au menu (choisir l'option 1) :**
```text
AT+CUSD=1,"1",15
→ OK

+CUSD: 0,"Votre solde credit est 1250 FCFA",15
```

- La réponse finale avec `<m>=0` clôt la session

### 7.6 Cas 3 — Session multi-niveaux

```text
AT+CUSD=1,"*555#",15
→ OK
+CUSD: 1,"1: Services\n2: Compte\n3: Assistance",15

AT+CUSD=1,"2",15
→ OK
+CUSD: 1,"1: Solde\n2: Numero\n3: Offre active",15

AT+CUSD=1,"1",15
→ OK
+CUSD: 0,"Solde: 1250 FCFA. Expire: 30/06/2026",15
```

### 7.7 Annuler une session USSD

Pour fermer une session interactive ouverte :

```text
AT+CUSD=2
→ OK
```

Après `+CUSD: 2`, la session est considérée terminée côté mobile.

### 7.8 Vérifier la configuration USSD

```text
AT+CUSD?
→ +CUSD: 1
   OK
```

Indique si la présentation des résultats est activée (1) ou non (0).

### 7.9 Problèmes courants USSD

| Problème | Cause probable | Solution |
|---|---|---|
| `+CUSD: 4` | USSD non supporté ou erreur réseau | Vérifier le code USSD, vérifier le signal |
| Timeout sans réponse | Signal faible ou congestion réseau | Vérifier `AT+CSQ`, réessayer |
| Message encodé en HEX | Réponse UCS2 de l'opérateur | Décoder HEX → UTF-16 BE (voir section 8) |
| `+CME ERROR: 4` | Commande non autorisée dans cet état | Annuler session avec `AT+CUSD=2` |
| Session bloquée | Session précédente non fermée | Envoyer `AT+CUSD=2` puis réessayer |

---

## 8. Encodage des caractères

### 8.1 Charsets supportés par le SIM800C

```text
AT+CSCS=?
→ +CSCS: ("IRA","GSM","UCS2","HEX","PCCP","PCDN","8859-1")
   OK
```

| Charset | Description |
|---|---|
| `"GSM"` | Alphabet GSM 7-bit (Latin de base, pas d'accents africains) |
| `"IRA"` | International Reference Alphabet (ASCII étendu) |
| `"UCS2"` | UTF-16 Big Endian, encodé en hexadécimal |
| `"HEX"` | Chaînes purement hexadécimales |
| `"8859-1"` | ISO Latin-1 |

### 8.2 Charset GSM (défaut)

```text
AT+CSCS="GSM"
→ OK
```

L'alphabet GSM 7-bit couvre le latin de base : a–z, A–Z, 0–9, et quelques caractères spéciaux. Il **ne supporte pas** les accents (é, è, à, ê…) ni les caractères africains spéciaux.

### 8.3 Charset UCS2

```text
AT+CSCS="UCS2"
→ OK
```

En UCS2, chaque caractère est encodé sur 2 octets (4 hex). Exemple : `"Bonjour"` devient :

```
0042006F006E006A006F007500720020
B    o    n    j    o    u    r   (espace)
```

**Quand utiliser UCS2 :**
- SMS contenant des accents (é, è, à, ê, ô…)
- SMS en langues africaines
- Quand l'opérateur renvoie une réponse USSD encodée en UCS2

### 8.4 Décoder une réponse UCS2

Certains opérateurs renvoient les réponses USSD en UCS2 :

```text
+CUSD: 0,"0056006F00740072006500200073006F006C00640065",15
```

**Procédure de décodage :**

1. Prendre la chaîne hex : `0056006F00740072...`
2. Découper en groupes de 4 : `0056`, `006F`, `0074`, `0072`...
3. Convertir chaque groupe en entier 16 bits (big-endian)
4. Interpréter comme codepoint Unicode → caractère

En Python :
```python
def decode_ucs2(hex_string):
    raw = bytes.fromhex(hex_string)
    return raw.decode('utf-16-be')

msg = "00560 06F00740072006500200073006F006C006400650"
print(decode_ucs2(msg.replace(" ", "")))
# → "Votre solde"
```

### 8.5 Envoyer un SMS avec accents (UCS2)

```text
AT+CSCS="UCS2"
AT+CSMP=17,167,0,8     ← DCS=8 pour UCS2
AT+CMGS="0042006F006E006A006F007500720020"   ← numéro encodé UCS2 aussi
> 00420069006500760065006E007500650020 [0x1A]
```

> **Note :** En UCS2, le numéro de téléphone dans `AT+CMGS` doit aussi être encodé en UCS2. Sinon, utiliser le mode PDU qui offre plus de contrôle.

### 8.6 Stratégie d'encodage recommandée

```
Si message uniquement latin de base → AT+CSCS="GSM" (max 160 car.)
Si message avec accents/spéciaux   → AT+CSCS="UCS2" + AT+CSMP=17,167,0,8 (max 70 car.)
Si réponse USSD reçue en HEX       → détecter et décoder côté application
```

---

## 9. Réponses asynchrones (URC)

### 9.1 Principe des URC

Les URC (Unsolicited Result Codes) sont des messages envoyés spontanément par le modem, **sans commande préalable**. Ils doivent être gérés dans un thread ou un handler séparé du flux de commandes.

### 9.2 URC importants

| URC | Exemple | Déclencheur |
|---|---|---|
| `+CMTI` | `+CMTI: "SM",3` | Nouveau SMS reçu (stocké à l'index 3) |
| `+CMT` | `+CMT: "+225...",,"26/05/27,10:00:00+00"` | SMS livré directement (si `<mt>=2`) |
| `+CUSD` | `+CUSD: 0,"Solde: 1250 FCFA",15` | Réponse USSD reçue |
| `+CREG` | `+CREG: 1` | Changement d'état d'enregistrement réseau |
| `RING` | `RING` | Appel vocal entrant |
| `+CLIP` | `+CLIP: "+2250700000000",145` | Identification de l'appelant |
| `NO CARRIER` | `NO CARRIER` | Appel terminé |

### 9.3 Architecture de gestion des URC

```
Thread UART reader
  → Ligne reçue
      ├── Commence par AT response attendue ? → dispatcher commande
      └── URC non attendu ?
            ├── +CMTI → déclencher lecture SMS
            ├── +CUSD → traiter réponse USSD
            ├── +CREG → mettre à jour état réseau
            └── RING  → gérer appel entrant
```

**Règle critique :** Ne jamais bloquer le thread UART en attendant une réponse de commande. Utiliser un mécanisme de callback ou de queue.

---

## 10. Gestion des erreurs

### 10.1 Types d'erreurs

| Code retour | Type | Description |
|---|---|---|
| `OK` | Succès | Commande exécutée avec succès |
| `ERROR` | Erreur générique | Commande invalide ou refusée |
| `+CME ERROR: <err>` | Erreur modem | Erreur liée au modem ou au réseau |
| `+CMS ERROR: <err>` | Erreur SMS | Erreur liée au service SMS |

### 10.2 Codes CME ERROR principaux

(Source : SIM800 AT Command Manual V1.10, §19.1)

| Code | Message |
|---|---|
| 0 | phone failure |
| 3 | operation not allowed |
| 4 | operation not supported |
| 10 | SIM not inserted |
| 11 | SIM PIN required |
| 12 | SIM PUK required |
| 13 | SIM failure |
| 14 | SIM busy |
| 16 | SIM wrong |
| 21 | SIM PUK2 required |
| 22 | memory full |
| 23 | invalid index |
| 24 | not found |
| 30 | no network service |
| 31 | network timeout |
| 32 | network not allowed — emergency calls only |

### 10.3 Codes CMS ERROR principaux

(Source : SIM800 AT Command Manual V1.10, §19.2)

| Code | Message |
|---|---|
| 1 | Unassigned (unallocated) number |
| 10 | Call barred |
| 21 | Short message transfer rejected |
| 28 | Invalid number format |
| 30 | No route to destination |
| 38 | No circuit/channel available |
| 41 | Temporary failure |
| 42 | Switching equipment congestion |
| 50 | Requested facility not subscribed |

### 10.4 Réponses à surveiller

```text
+CME ERROR: SIM not inserted    → vérifier la SIM physiquement
+CME ERROR: no network service  → vérifier AT+CSQ et AT+CREG
+CME ERROR: memory full         → supprimer des SMS avec AT+CMGDA
+CMS ERROR: 21                  → SMS refusé par le réseau (solde insuffisant ?)
+CUSD: 4                        → USSD non supporté ou code invalide
```

---

## 11. Gestion alimentation et stabilité

### 11.1 Exigences d'alimentation

Le SIM800C est **très sensible** à la qualité de l'alimentation.

| Paramètre | Valeur |
|---|---|
| Tension d'alimentation | 3.4V – 4.4V (typique : 4.0V) |
| Courant en veille | ~1 mA |
| Courant en communication GSM | 500 mA – 2 A (pics) |
| Condensateur recommandé | ≥ 1000 µF sur VBAT |

### 11.2 Symptômes d'alimentation insuffisante

| Symptôme | Cause probable |
|---|---|
| Reset intempestif du modem | Chute de tension lors des pics GSM |
| Freeze UART | Chute de tension soudaine |
| Perte réseau répétée | Courant insuffisant lors de l'émission GSM |
| SMS non envoyés | Tension trop basse pendant l'émission |
| `+CME ERROR: 0` (phone failure) | Alimentation instable |

**Recommandation matérielle :** Placer un condensateur électrolytique de 1000 µF (minimum) et un condensateur céramique de 100 nF directement sur les broches VBAT du module. L'alimentation doit être capable de fournir des pics de 2 A sans chuter sous 3.4V.

### 11.3 Reset logiciel

```text
AT+CFUN=1,1
→ OK
[le modem redémarre, attendre ~5 secondes puis +CPIN: READY]
```

### 11.4 Mode veille — `AT+CSCLK`

```text
AT+CSCLK=1    → Active le mode veille (consommation réduite)
AT+CSCLK=0    → Désactive le mode veille
```

Pour réveiller le modem en mode veille, envoyer simplement `AT`. Le modem doit répondre `OK` avant d'accepter d'autres commandes.

---

## 12. Bonnes pratiques et séquences complètes

### 12.1 Règles fondamentales

1. **1 commande AT à la fois** — ne jamais envoyer une seconde commande avant d'avoir reçu `OK` ou `ERROR`
2. **Toujours gérer les timeouts** — une commande sans réponse doit être considérée en erreur
3. **Logger tous les échanges UART** — indispensable pour le debug
4. **Vérifier `AT+CSQ` régulièrement** — un signal en dessous de 10 peut causer des échecs SMS/USSD
5. **Gérer les URC dans un thread séparé** — ne pas bloquer le flux principal
6. **Tester l'alimentation** — la majorité des bugs inexplicables viennent d'une alimentation instable

### 12.2 Séquence complète — Envoi SMS

```text
[1. Vérification préalable]
AT+CSQ          → signal OK ?
AT+CREG?        → enregistré réseau ?

[2. Configuration]
AT+CMGF=1       → mode texte
AT+CSCS="GSM"   → charset GSM (ou UCS2 si accents)

[3. Envoi]
AT+CMGS="+2250700000000"
> Votre commande #1234 a ete confirmee.[0x1A]

[4. Confirmation]
+CMGS: 42
OK
```

### 12.3 Séquence complète — Consultation solde USSD

```text
[1. Vérification préalable]
AT+CSQ
AT+CREG?

[2. Envoi USSD]
AT+CUSD=1,"*111#",15
→ OK

[3. Attente réponse (jusqu'à 30 sec)]
+CUSD: 0,"Votre solde est 1250 FCFA. Expire: 30/06/2026",15

[4. Traitement du message]
→ Extraire la valeur du solde par parsing
```

### 12.4 Séquence complète — Menu USSD interactif

```text
AT+CUSD=1,"*200#",15
→ OK
+CUSD: 1,"1: Solde\n2: Forfaits\n3: Aide",15

[Choisir option 2]
AT+CUSD=1,"2",15
→ OK
+CUSD: 1,"1: Internet\n2: Appels\n3: SMS",15

[Choisir option 1]
AT+CUSD=1,"1",15
→ OK
+CUSD: 0,"Forfait 1Go a 500 FCFA. Tapez 1 pour confirmer.",15

[Confirmer]
AT+CUSD=1,"1",15
→ OK
+CUSD: 0,"Activation reussie. Forfait 1Go valable 7 jours.",15
```

### 12.5 Séquence complète — Réception et lecture SMS

```text
[URC reçu spontanément :]
+CMTI: "SM",5

[Lire le SMS à l'index 5]
AT+CMGR=5
→ +CMGR: "REC UNREAD","+2250701234567",,"26/05/27,10:15:00+00"
   Confirmation paiement 5000 FCFA. Ref: TXN202605271015
   OK

[Supprimer après traitement]
AT+CMGD=5
→ OK
```

---

## 13. Tableau de référence rapide

### 13.1 Commandes essentielles

| Commande | Description |
|---|---|
| `AT` | Test présence modem |
| `ATE0` | Désactiver écho |
| `AT+CMEE=2` | Erreurs verbose |
| `AT+CPIN?` | État SIM |
| `AT+CREG?` | État réseau GSM |
| `AT+CSQ` | Niveau signal |
| `AT+COPS?` | Opérateur actuel |
| `AT+CSCS="GSM"` | Charset GSM |
| `AT+CSCS="UCS2"` | Charset UCS2 |
| `AT+CMGF=1` | Mode SMS texte |
| `AT+CMGF=0` | Mode SMS PDU |
| `AT+CNMI=2,1,0,0,0` | Notifications SMS |
| `AT+CMGS="<num>"` | Envoyer SMS |
| `AT+CMGR=<index>` | Lire SMS |
| `AT+CMGL="ALL"` | Lister SMS |
| `AT+CMGD=<index>` | Supprimer SMS |
| `AT+CMGDA="DEL ALL"` | Supprimer tous SMS |
| `AT+CUSD=1,"<code>",15` | Envoyer USSD |
| `AT+CUSD=2` | Annuler session USSD |
| `AT+IFC=2,2` | Flow control RTS/CTS |
| `AT+CFUN=1,1` | Reset logiciel |
| `AT+CSCLK=1` | Mode veille |
| `AT&W` | Sauvegarder config |
| `AT+GSN` | IMEI |
| `AT+CIMI` | IMSI |
| `AT+CCID` | ICCID |

### 13.2 Sections du manuel SIM800 V1.10

| Section | Commande | Sujet |
|---|---|---|
| 3.2.12 | AT+CSCS | Charset |
| 3.2.32 | AT+CREG | Enregistrement réseau |
| 3.2.35 | AT+CSQ | Niveau signal |
| 3.2.53 | AT+CUSD | USSD |
| 4.2.1 | AT+CMGD | Supprimer SMS |
| 4.2.2 | AT+CMGF | Format SMS |
| 4.2.3 | AT+CMGL | Lister SMS |
| 4.2.4 | AT+CMGR | Lire SMS |
| 4.2.5 | AT+CMGS | Envoyer SMS |
| 4.2.8 | AT+CNMI | Notifications SMS |
| 6.2.20 | AT+CSCLK | Veille |
| 6.2.25 | AT+CMGDA | Supprimer tous SMS |
| 19.1 | — | Codes CME ERROR |
| 19.2 | — | Codes CMS ERROR |

---

*Documentation générée à partir du SIM800 Series AT Command Manual V1.10 (SIMCom, 2016-10-20) et des bonnes pratiques terrain.*
