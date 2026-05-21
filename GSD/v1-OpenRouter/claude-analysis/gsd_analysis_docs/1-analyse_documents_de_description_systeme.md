# Analyse du Document de Description Système
> Fichier source : `project_desc.txt` + `Codes_USSD_CI.xlsx`  
> Date d'analyse : 20 Mai 2026

---

## 1. Vue d'Ensemble du Projet

### 1.1 Identité du Projet
| Attribut | Valeur |
|---|---|
| Nom du projet | SIM800C USB Supervisor & USSD Manager |
| Environnement cible | Windows (PC local) |
| Chemin du projet | `C:\xampp\htdocs\aa_Toolbox\test_sim800c\` |
| Chemin des livrables | `C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter` |
| URL d'accès frontend | `test_sim800c.local:80` |
| Pays cible | Côte d'Ivoire (CI) |

### 1.2 Stack Technique Définie
| Couche | Technologie |
|---|---|
| Frontend | Web (HTML/CSS/JS) |
| Backend | GoLang |
| Base de données | MySQL |
| Communication hardware | Port série (COM5, COM6, COM7) via USB Hub 3.0 alimenté |
| Fichier de référence | `Codes_USSD_CI.xlsx` |

---

## 2. Matériel Connecté

### 2.1 Modules SIM800C USB
- **Quantité** : 3 modules
- **Ports COM** : COM5, COM6, COM7
- **Connexion** : USB Hub 3.0 alimenté (externe, assurant une alimentation suffisante)
- **Carte SIM** : 1 SIM par module (opérateurs CI : Orange CI, MTN CI, Moov Africa CI)

### 2.2 Indicatif & Plan de Numérotation CI
| Opérateur | Préfixe | Format |
|---|---|---|
| Orange CI | 07 | 07 XX XX XX XX |
| MTN CI | 05 | 05 XX XX XX XX |
| Moov Africa CI | 01 | 01 XX XX XX XX |
| Indicatif pays | +225 / 00225 / 225 | — |

---

## 3. Analyse du Fichier Codes_USSD_CI.xlsx

### 3.1 Structure du Fichier
| Colonne | Rôle |
|---|---|
| ID | Identifiant unique du code USSD |
| Carrier | Opérateur cible (Orange CI, MTN CI, Moov Africa CI, Universel) |
| Action | Catégorie fonctionnelle (Consulter, Services_N1, Services_N2, Achat_N1, Achat_N2) |
| Target | Interne (bénéficiaire = SIM elle-même) / Externe (bénéficiaire = tiers) |
| Operation | Libellé de l'opération |
| USSD_Code | Code USSD à exécuter |
| Information_INPUT | Données requises avant exécution |
| Information_OUTPUT | Données retournées après exécution |
| Scope | In (utile au projet) / Out (hors périmètre) |
| Comment | Commentaire libre |
| Parent_USSD_ID | ID du code parent (hiérarchie de menu) |

### 3.2 Inventaire par Opérateur (Scope = In uniquement)
| Opérateur | Nb codes IN | Actions présentes |
|---|---|---|
| Orange CI | ~10 | Consulter, Services_N1, Services_N2, Achat_N1, Achat_N2 |
| MTN CI | ~9 | Consulter, Services_N1, Achat_N1 |
| Moov Africa CI | ~16 | Consulter, Services_N1, Services_N2, Achat_N1, Achat_N2 |
| Universel | 2 | Consulter (IMEI, numéro de téléphone) |

### 3.3 Règles de Validation des Inputs
| Type d'input | Règle de validation |
|---|---|
| Aucun | Pas de saisie nécessaire |
| Choix | Nombre valable et présent dans les options du menu |
| PIN | Nombre de 4 chiffres exactement |
| Code de carte recharge | Nombre de 14 chiffres |
| Numéro / Numéro de téléphone | Nombre de 10 chiffres (sans indicatif) |
| Montant | Nombre ≥ 2 chiffres, valeur ≥ 50 |
| Référence | Nombre de 14 chiffres |

---

## 4. Analyse des 5 Fonctions Décrites

### Fonction 1 : Module Auto-Discovery
| Sous-fonction | Description |
|---|---|
| 1-1 | Scan et identification de tous les modules SIM800C USB connectés (ports COM) |
| 1-2 | Pour chaque module : exécution des codes USSD Universel pour collecter IMEI et numéro SIM |
| 1-3 | Affichage temps réel dans un dashboard |

**Codes USSD utilisés :**
- `*#06#` → IMEI (ID 71)
- `#99#` → Numéro de téléphone (ID 72)

**Identification de l'opérateur** : basée sur le préfixe du numéro retourné (07 = Orange CI, 05 = MTN CI, 01 = Moov Africa CI).

---

### Fonction 2-1 : SIM Status Manual-Discovery
| Sous-fonction | Description |
|---|---|
| 2-1-1 | Création de boutons (avec infobulle) par code USSD filtré : Carrier = opérateur SIM détecté, Action = Consulter, Target = Interne, Scope = In |
| 2-1-2 | Affichage des boutons par module |
| 2-1-3 | Exécution au clic + affichage résultat temps réel |

---

### Fonction 2-2 : SIM Status Auto-Discovery
| Sous-fonction | Description |
|---|---|
| 2-2-1 | Bouton global déclenchant l'exécution automatique de tous les codes USSD (Consulter / Interne / Scope=In) pour chaque module |
| 2-2-2 | Affichage des résultats en temps réel dans le dashboard |

---

### Fonction 3-1 : USSD Menu Manual-Discovery
| Sous-fonction | Description |
|---|---|
| 3-1-1 | Création de boutons par code USSD filtré : Carrier = opérateur SIM, Action = Services_N1, Target = Interne, Scope = In |
| 3-1-4 | Au clic : exploration récursive du menu USSD jusqu'au dernier niveau, affichage temps réel |
| 3-1-5 | Mise à jour de `Codes_USSD_CI.xlsx` si nouveaux codes découverts → génération version horodatée (ex: `Codes_USSD_CI-v20052026-082405.xlsx`) |

---

### Fonction 3-2 : USSD Menu Auto-Discovery
| Sous-fonction | Description |
|---|---|
| 3-2-1 | Bouton global déclenchant l'exploration automatique de tous les menus Services_N1 pour chaque module |
| 3-2-2 | Exploration récursive de chaque menu et sous-menu jusqu'au dernier niveau, affichage temps réel |
| 3-2-3 | Mise à jour de `Codes_USSD_CI.xlsx` + génération de la nouvelle version horodatée avec toutes les nouvelles options |

---

### Fonction 4 : USSD Manager
| Sous-fonction | Description |
|---|---|
| 4-1 | Saisie manuelle d'un code USSD + sélection du module cible + exécution |
| 4-2 | Affichage du résultat en temps réel |

---

### Fonction 5 : SMS Manager
| Sous-fonction | Description |
|---|---|
| 5-1 | Créer, Lire, Supprimer les SMS sur chaque module en temps réel |
| 5-2 | Corbeille automatique : tout SMS reçu ne contenant pas le mot "Test" est déplacé automatiquement vers la corbeille |

---

## 5. Exigences Transversales

| Exigence | Détail |
|---|---|
| Temps réel | Toutes les fonctions affichent les résultats en temps réel (WebSocket) |
| Thème UI | Clair et sombre, avec bouton de bascule |
| Design Frontend | Utiliser le skill "impeccable" (`/polish`) pour améliorer le design |
| Accessibilité | Frontend via `test_sim800c.local:80` |
| Validation des inputs USSD | Validation stricte selon le type de champ (cf. section 3.3) |
| Gestion du fichier Excel | Filtrer uniquement Scope = In ; générer des versions horodatées en cas de découverte |
| Répertoire de stockage Excel | `C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\` |

---

## 6. Synthèse des Flux de Données

```
[SIM800C USB Module x3]
        |
  COM5 / COM6 / COM7
        |
  [Backend GoLang]
   - Lecture/écriture port série (AT Commands)
   - API REST / WebSocket
   - Gestion des codes USSD depuis MySQL
   - Gestion du fichier Excel
        |
  [Base de données MySQL]
   - Modules, SIMs, Codes USSD, Résultats, SMS, Logs
        |
  [Frontend Web]
   - Dashboard temps réel (WebSocket)
   - Interface des 5 fonctions
   - Thème clair/sombre
```

---

## 7. Commandes AT SIM800C Impliquées (Synthèse)

| Fonction | Commande AT |
|---|---|
| Initialisation | AT, ATE0, AT+CMEE=2, AT+CPIN? |
| Identification module | AT+CGSN (IMEI), AT+CNUM (numéro SIM) |
| Exécution USSD | AT+CUSD=1,"code_ussd",15 |
| Signal réseau | AT+CSQ, AT+CREG? |
| SMS - Lire | AT+CMGF=1, AT+CMGL="ALL" |
| SMS - Envoyer | AT+CMGS="numero" |
| SMS - Supprimer | AT+CMGD=index |
| Identification opérateur | AT+COPS? |
