Voici les principaux codes d’erreur détaillés 
```
+CME ERROR
```
 utilisés par les modules SIM800C / SIM900, avec leurs descriptions et les causes fréquentes observées en pratique.

Pour activer ces erreurs détaillées :

```


```
AT+CMEE=2
```








```

Sinon le module retourne seulement :

```


```
ERROR
```








```

au lieu de :

```


```
+CME ERROR: 10
```








```

[manualzz.com+2Scribd+2](https://manualzz.com/doc/1524264/simcom-sim900-gsm-gprs-module-at-commands-manual?utm_source=chatgpt.com)

---

# Codes +CME ERROR principaux

 Code | Description | Causes fréquentes |
| --- | --- | --- |
 0 | phone failure | Crash interne module |
 1 | no connection to phone | UART/COM déconnecté |
 2 | phone-adaptor link reserved | Port déjà utilisé |
 3 | operation not allowed | Commande interdite dans l’état actuel |
 4 | operation not supported | Commande non supportée |
 5 | PH-SIM PIN required | PIN téléphone requis |
 6 | PH-FSIM PIN required | PIN FSIM requis |
 7 | PH-FSIM PUK required | PUK FSIM requis |
 10 | SIM not inserted | Carte SIM absente |
 11 | SIM PIN required | PIN SIM requis |
 12 | SIM PUK required | PUK SIM requis |
 13 | SIM failure | SIM défectueuse |
 14 | SIM busy | SIM occupée |
 15 | SIM wrong | SIM incompatible |
 16 | incorrect password | Mauvais PIN |
 17 | SIM PIN2 required | PIN2 requis |
 18 | SIM PUK2 required | PUK2 requis |
 20 | memory full | Mémoire pleine |
 21 | invalid index | Index SMS invalide |
 22 | not found | Élément introuvable |
 23 | memory failure | Erreur mémoire |
 24 | text string too long | Texte trop long |
 25 | invalid characters in text string | Caractères invalides |
 26 | dial string too long | Numéro trop long |
 27 | invalid characters in dial string | Numéro invalide |
 30 | no network service | Aucun réseau |
 31 | network timeout | Timeout réseau |
 32 | network not allowed - emergency calls only | Réseau limité urgences |
 40 | network personalization PIN required | PIN opérateur requis |
 41 | network personalization PUK required | PUK opérateur requis |
 42 | network subset personalization PIN required | PIN subset requis |
 43 | network subset personalization PUK required | PUK subset requis |
 44 | service provider personalization PIN required | PIN fournisseur requis |
 45 | service provider personalization PUK required | PUK fournisseur requis |
 46 | corporate personalization PIN required | PIN entreprise requis |
 47 | corporate personalization PUK required | PUK entreprise requis |
 99 | resource limitation | Ressources insuffisantes |
 100 | unknown | Erreur inconnue |
 103 | illegal MS | Mobile station interdite |
 106 | illegal ME | Equipement interdit |
 107 | GPRS services not allowed | GPRS interdit |
 111 | PLMN not allowed | Réseau interdit |
 112 | location area not allowed | Zone interdite |
 113 | roaming not allowed in this location area | Roaming interdit |
 132 | service option not supported | Service non supporté |
 133 | requested service option not subscribed | Service non souscrit |
 134 | service option temporarily out of order | Service indisponible |
 148 | unspecified GPRS error | Erreur GPRS |
 149 | PDP authentication failure | Authentification PDP échouée |
 150 | invalid mobile class | Classe mobile invalide |

[dfimg.dfrobot.com+3Scribd+3Scribd+3](https://www.scribd.com/document/370492683/SIM900-at-Command-Manual-V1-03?utm_source=chatgpt.com)

---

# Codes SIM800/SIM900 étendus fréquemment rencontrés

 Code | Description |
| --- | --- |
 310 | SIM not inserted |
 311 | SIM PIN necessary |
 312 | PH SIM PIN necessary |
 313 | SIM failure |
 314 | SIM busy |
 315 | SIM wrong |
 316 | SIM PUK required |
 317 | SIM PIN2 required |
 318 | SIM PUK2 required |
 320 | memory failure |
 321 | invalid memory index |
 322 | memory full |
 323 | invalid parameter |
 324 | invalid input format |
 330 | SMSC address unknown |
 331 | no network |
 332 | network timeout |
 340 | no CNMA acknowledgment |
 500 | unknown |
 512 | SIM not ready |
 513 | unread records on SIM |
 514 | CB error unknown |
 515 | PS busy |

[Manuelshelf](https://www.manualshelf.com/manual/simcom/sim900/at-command-english.html?utm_source=chatgpt.com)

---

# Erreurs les plus importantes en pratique

## 
```
+CME ERROR: 10
```

```


```
SIM not inserted
```








```

Causes :

-   SIM mal insérée,
    
-   faux contact,
    
-   SIM oxydée,
    
-   tension instable.
---

## 
```
+CME ERROR: 14
```

```


```
SIM busy
```








```

Très fréquent.

Causes :

-   USSD encore actif,
    
-   SMS en cours,
    
-   lecture SIM simultanée,
    
-   commandes trop rapides.

Solution :

```


```
AT+CUSD=2
```








```

Puis attendre 1–2 sec.

---

## 
```
+CME ERROR: 30
```

```


```
no network service
```








```

Causes :

-   antenne absente,
    
-   mauvais signal,
    
-   opérateur indisponible,
    
-   SIM inactive.

Vérifier :

```


```
AT+CSQ  
AT+CREG?
```








```
---

## 
```
+CME ERROR: 31
```

```


```
network timeout
```








```

Très fréquent avec USSD.

Causes :

-   opérateur lent,
    
-   mauvaise qualité radio,
    
-   session USSD bloquée.
---

## 
```
+CME ERROR: 515
```

```


```
PS busy
```








```

Très fréquent sur SIM800C modernes.

Signifie :

-   pile protocolaire occupée,
    
-   session data/USSD/SMS encore active.

Solution :

-   attendre,
    
-   fermer session,
    
-   reset radio si nécessaire.

[Manuelshelf](https://www.manualshelf.com/manual/simcom/sim900/at-command-english.html?utm_source=chatgpt.com)

---

# Commandes utiles de diagnostic

## Vérifier SIM

```


```
AT+CPIN?
```








```

Réponse correcte :

```


```
+CPIN: READY
```








```
---

## Vérifier réseau

```


```
AT+CREG?
```








```

Bon état :

```


```
+CREG: 0,1
```








```

ou :

```


```
+CREG: 0,5
```








```
---

## Vérifier signal

```


```
AT+CSQ
```








```

Bon signal :

```


```
15 à 31
```








```
---

## Vérifier opérateur

```


```
AT+COPS?
```








```
---

# Bonnes pratiques de gestion des erreurs

## Toujours :

```


```
AT+CMEE=2
```








```
---

## Ajouter timeout logiciel

Exemple :

-   USSD → 45 sec max
    
-   SMS → 60 sec max
    
-   Commande AT → 5 sec max
---

## Ajouter retry intelligent

Exemple :

 Erreur | Retry |
| --- | --- |
 14 | Oui |
 31 | Oui |
 515 | Oui |
 10 | Non |
 11 | Non |
 30 | Oui |

---

# Exemple de logique robuste

```


```
envoyer commande  
↓  
attendre réponse  
↓  
OK ?  
  → continuer  
↓  
+CME ERROR ?  
  → parser code  
↓  
retry / reset / reconnect
```








```
---

# Différence importante

 Type | Domaine |
| --- | --- |
 
```
+CME ERROR
```
 | module / SIM / réseau |
 
```
+CMS ERROR
```
 | SMS |
 
```
ERROR
```
 | erreur générique |

Le SIM800C utilise énormément :

-   ```
    +CME ERROR
    ```
    
-   ```
    +CMS ERROR
    ```

dans les systèmes professionnels GSM.