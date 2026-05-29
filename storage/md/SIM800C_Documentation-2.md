# SIM800C Documentation
## Documentation spécialisée sur la gestion des modules SIM800C
### Focus : Commandes AT, SMS, USSD, UART, Réseau GSM

Version: 1.0  
Date: 2026-05-27

---

# 1. Introduction

Cette documentation est spécialisée sur la gestion technique des modules SIM800C.

Elle couvre principalement :

- communication série UART
- commandes AT
- gestion USSD
- gestion SMS
- gestion réseau GSM
- gestion des réponses asynchrones
- gestion des erreurs
- encodage GSM/UCS2
- bonnes pratiques matérielles

---

# 2. Présentation du SIM800C

Le module SIM800C est un modem GSM/GPRS 2G développé par SIMCom.

Fonctionnalités principales :

- SMS
- USSD
- appels vocaux
- GPRS
- TCP/IP
- HTTP
- FTP
- UART AT Commands

---

# 3. Architecture de communication

```text
Application
    ↓
Port Série UART / USB
    ↓
SIM800C
    ↓
Réseau GSM
```

---

# 4. Communication UART

## Paramètres recommandés

| Paramètre | Valeur |
|---|---|
| Baudrate | 115200 |
| Data bits | 8 |
| Stop bits | 1 |
| Parity | None |
| Flow control | RTS/CTS |

---

# 5. Syntaxe des commandes AT

Chaque commande doit commencer par :

```text
AT
```

et se terminer par :

```text
<CR>
```

Exemple :

```text
AT+CSQ
```

---

# 6. Réponses modem

## Réponses classiques

| Réponse | Signification |
|---|---|
| OK | Succès |
| ERROR | Erreur |
| +CME ERROR | Erreur modem |
| +CMS ERROR | Erreur SMS |

---

# 7. Initialisation recommandée

## Séquence complète

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

# 8. Désactiver l’écho

```text
ATE0
```

---

# 9. Activer les erreurs détaillées

```text
AT+CMEE=2
```

---

# 10. Vérification SIM

## Vérifier état SIM

```text
AT+CPIN?
```

Réponse :

```text
+CPIN: READY
```

---

# 11. Vérification réseau GSM

```text
AT+CREG?
```

Réponses valides :

```text
+CREG: 0,1
```

ou

```text
+CREG: 0,5
```

---

# 12. Vérification signal GSM

```text
AT+CSQ
```

Exemple :

```text
+CSQ: 18,0
```

---

# 13. Interprétation CSQ

| Valeur | Qualité |
|---|---|
| 0-9 | Mauvaise |
| 10-14 | Moyenne |
| 15-31 | Bonne |

---

# 14. Informations modem

## IMEI

```text
AT+GSN
```

## IMSI

```text
AT+CIMI
```

## ICCID

```text
AT+CCID
```

## Version firmware

```text
ATI
```

---

# 15. Gestion des SMS

---

# 16. Modes SMS

## Mode texte

```text
AT+CMGF=1
```

## Mode PDU

```text
AT+CMGF=0
```

Le mode texte est recommandé pour la plupart des usages.

---

# 17. Envoi SMS

## Étape 1

```text
AT+CMGS="+2250700000000"
```

Le modem répond :

```text
>
```

---

## Étape 2

Envoyer le texte :

```text
Bonjour
```

---

## Étape 3

Envoyer CTRL+Z

Code ASCII :

```text
26
```

---

# 18. Réception SMS

## Configuration notifications

```text
AT+CNMI=2,1,0,0,0
```

---

# 19. Notification SMS entrant

Exemple :

```text
+CMTI: "SM",1
```

---

# 20. Lire SMS

```text
AT+CMGR=1
```

---

# 21. Lister SMS

```text
AT+CMGL="ALL"
```

---

# 22. Supprimer SMS

## Supprimer un SMS

```text
AT+CMGD=1
```

## Supprimer tous les SMS

```text
AT+CMGDA="DEL ALL"
```

---

# 23. États SMS

| État | Description |
|---|---|
| REC UNREAD | Non lu |
| REC READ | Lu |
| STO UNSENT | Non envoyé |
| STO SENT | Envoyé |

---

# 24. Gestion USSD

---

# 25. Commande USSD principale

```text
AT+CUSD
```

---

# 26. Envoyer une requête USSD

## Exemple

```text
AT+CUSD=1,"*111#",15
```

---

# 27. Réponse USSD

## Format

```text
+CUSD: <m>,"message",<dcs>
```

---

# 28. États USSD

| Valeur | Signification |
|---|---|
| 0 | Réponse finale |
| 1 | Session ouverte |
| 2 | Session terminée |
| 4 | Opération non supportée |

---

# 29. Exemple USSD simple

## Envoi

```text
AT+CUSD=1,"*111#",15
```

## Réponse

```text
+CUSD: 0,"Votre solde est 1000 FCFA",15
```

---

# 30. Exemple menu interactif

## Réponse réseau

```text
+CUSD: 1,"1: Solde\n2: Internet",15
```

Le réseau attend une réponse.

---

# 31. Répondre à un menu USSD

```text
AT+CUSD=1,"1",15
```

---

# 32. Fin de session USSD

```text
+CUSD: 2
```

Signifie :

- session fermée
- menu terminé

---

# 33. Annuler une session USSD

```text
AT+CUSD=2
```

---

# 34. Lire configuration USSD

```text
AT+CUSD?
```

---

# 35. Encodage des caractères

Le SIM800C supporte :

- GSM
- IRA
- UCS2
- HEX

---

# 36. Définir charset GSM

```text
AT+CSCS="GSM"
```

---

# 37. Définir charset UCS2

```text
AT+CSCS="UCS2"
```

---

# 38. Gestion UCS2

Certains opérateurs renvoient :

```text
0042006F006E006A006F0075
```

Il faut :

1. convertir HEX → bytes
2. décoder UTF16-BE

---

# 39. Réponses asynchrones (URC)

Le SIM800C envoie des réponses spontanées appelées :

```text
URC
```

---

# 40. Exemples URC

| URC | Description |
|---|---|
| +CMTI | Nouveau SMS |
| +CUSD | Réponse USSD |
| +CREG | État réseau |
| RING | Appel entrant |

---

# 41. Gestion UART

## Important

Les réponses peuvent arriver :

- en retard
- en plusieurs lignes
- de manière asynchrone

---

# 42. Gestion buffers UART

## Recommandations

Utiliser :

- buffer circulaire
- parser CRLF
- timeout lecture

---

# 43. Timeout recommandés

| Action | Timeout |
|---|---|
| Commande AT | 5 sec |
| USSD | 30 sec |
| SMS | 15 sec |
| Réseau | 60 sec |

---

# 44. Gestion Flow Control

## Recommandé

RTS/CTS

Commande :

```text
AT+IFC=2,2
```

---

# 45. Sauvegarder configuration

```text
AT&W
```

---

# 46. Redémarrage modem

## Software reset

```text
AT+CFUN=1,1
```

---

# 47. Gestion alimentation

## Très important

Le SIM800C nécessite :

- tension stable autour de 4V
- pics de courant jusqu’à 2A

---

# 48. Symptômes alimentation insuffisante

| Symptôme | Cause |
|---|---|
| reset modem | alimentation faible |
| freeze | chute tension |
| perte réseau | courant insuffisant |

---

# 49. Gestion réseau GSM

## Vérifier opérateur

```text
AT+COPS?
```

---

# 50. Sélection automatique opérateur

```text
AT+COPS=0
```

---

# 51. Vérifier attachement GPRS

```text
AT+CGATT?
```

---

# 52. Gestion veille modem

## Sleep mode

```text
AT+CSCLK=1
```

---

# 53. Réveil modem

Envoyer :

```text
AT
```

---

# 54. Gestion ports série

## Windows

```text
COM3
COM4
```

## Linux

```text
/dev/ttyUSB0
/dev/ttyUSB1
```

---

# 55. Détection automatique modem

## Vérification présence

```text
AT
ATI
AT+GSN
```

---

# 56. Causes fréquentes d’erreurs

| Cause | Symptôme |
|---|---|
| mauvais câble USB | déconnexion |
| alimentation faible | reset |
| mauvais signal GSM | timeout |
| commandes simultanées | réponses corrompues |

---

# 57. Bonnes pratiques

## Recommandations

- 1 commande AT à la fois
- attendre OK avant prochaine commande
- gérer les timeouts
- logger les échanges UART
- vérifier CSQ régulièrement

---

# 58. Exemple séquence complète SMS

```text
AT
OK

AT+CMGF=1
OK

AT+CMGS="+2250700000000"
>

Bonjour
CTRL+Z

+CMGS: 25
OK
```

---

# 59. Exemple séquence complète USSD

```text
AT+CUSD=1,"#111#",15
OK

+CUSD: 1,"1: Solde\n2: Internet",15

AT+CUSD=1,"1",15
OK

+CUSD: 0,"Votre solde est 1000 FCFA",15
```

---

# 60. Références importantes du manuel SIM800

## Sections importantes

| Section | Sujet |
|---|---|
| 3.2.53 | AT+CUSD |
| 4.2.5 | AT+CMGS |
| 4.2.4 | AT+CMGR |
| 4.2.8 | AT+CNMI |
| 3.2.32 | AT+CREG |
| 3.2.35 | AT+CSQ |
| 3.2.12 | AT+CSCS |
| 6.2.20 | AT+CSCLK |

---

# 61. Références officielles

## Documentation officielle

SIM800 Series AT Command Manual V1.10

## Lien GitHub

https://github.com/anothermist/LIBRARIES/blob/master/TinyGSM/extras/AT%20Command%20Manuals/SIM800%20Series%20AT%20Command%20Manual%20V1.10.pdf

---

# 62. Conclusion

Le SIM800C est un module très puissant mais sensible :

- à l’alimentation
- à la qualité du signal GSM
- à la gestion UART
- aux timeouts

Les points critiques pour un système stable sont :

- gestion correcte des réponses asynchrones
- contrôle strict des commandes AT
- gestion des buffers UART
- surveillance réseau GSM
- gestion des encodages GSM/UCS2

