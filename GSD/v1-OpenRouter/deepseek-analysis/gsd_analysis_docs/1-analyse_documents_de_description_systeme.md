# 1. Analyse des documents de description système

**Date :** 20/05/2026
**Projet :** Supervision et exécution de fonctions SIM800C USB
**Fichier source :** `project_desc.txt`, `Codes_USSD_CI.xlsx`

## 1. Résumé du besoin

Développement d'une application web (Frontend + Backend Go + MySQL) pour superviser et contrôler en temps réel 3 modules SIM800C USB (COM5,6,7). L'application exploite un fichier Excel (`Codes_USSD_CI.xlsx`) contenant les codes USSD des opérateurs mobiles de Côte d'Ivoire (Orange, MTN, Moov Africa).

## 2. Analyse détaillée des fonctionnalités requises

### 2.1. Module Auto-Discovery (Fonction 1)
- **1-1** : Scan et identification automatique des modules SIM800C sur les ports COM (Windows).
- **1-2** : Collecte des informations SIM via code USSD universel (`#99#` ou `*#06#`).
- **1-3** : Affichage tableau de bord temps réel.

### 2.2. SIM Status (Fonctions 2-1 et 2-2)
- **2-1 (Manuel)** : Boutons par module pour exécuter les codes USSD `Action=Consulter`, `Target=Interne`, `Scope=In`, selon l'opérateur détecté.
- **2-2 (Auto)** : Bouton global exécutant tous ces codes sur tous les modules. Résultats en temps réel.

### 2.3. USSD Menu (Fonctions 3-1 et 3-2)
- **3-1 (Manuel)** : Boutons par module pour explorer les menus USSD (`Action=Services_N1`, `Target=Interne`, `Scope=In`). Exploration récursive jusqu'à la fin.
- **3-2 (Auto)** : Exploration automatique de tous ces menus sur tous les modules. Mise à jour du fichier Excel avec les nouveaux codes USSD découverts (création nouvelle version).

### 2.4. USSD Manager (Fonction 4)
- Saisie manuelle et exécution d'un code USSD sur un module sélectionné. Résultat temps réel.

### 2.5. SMS Manager (Fonction 5)
- CRUD des SMS sur chaque module.
- Corbeille automatique pour les SMS ne contenant pas le mot "Test".

## 3. Contraintes techniques et règles de gestion

### 3.1. Fichier Excel `Codes_USSD_CI.xlsx`
- Structure : `ID, Carrier, Action, Target, Operation, USSD_Code, Information_INPUT, Information_OUTPUT, Scope, Comment, Parent_USSD_ID`
- Filtre : Seules les lignes avec `Scope == "In"` sont utilisées.
- Validation des entrées selon `Information_INPUT` :
  - *Choix* : nombre valide présent dans les options du menu
  - *PIN* : nombre de 4 chiffres
  - *Code de carte recharge* : nombre de 14 chiffres
  - *Numéro* : nombre de 10 chiffres (sans indicatif)
  - *Montant* : nombre >= 50 (2 chiffres ou plus)
  - *Référence* : nombre de 14 chiffres

### 3.2. Plan de numérotation Côte d'Ivoire
| Opérateur | Indicatif | Préfixe | Format numéro |
|-----------|-----------|---------|----------------|
| Orange CI | +225 / 00225 / 225 | 07 | 07 XX XX XX XX |
| MTN CI    | +225 / 00225 / 225 | 05 | 05 XX XX XX XX |
| Moov Africa CI | +225 / 00225 / 225 | 01 | 01 XX XX XX XX |

### 3.3. Technologies imposées
- **Backend** : GoLang
- **Base de données** : MySQL
- **Frontend** : Web (HTML/CSS/JS) avec thème clair/sombre
- **Communication temps réel** : WebSocket ou SSE
- **Accès** : `test_sim800c.local:80`

### 3.4. Stockage des fichiers
- Code : `C:\xampp\htdocs\aa_Toolbox\test_sim800c\blackbox\v1`
- Excel source : `C:\xampp\htdocs\aa_Toolbox\test_sim800c\storage\excel\`
- Nouvelles versions Excel : `Codes_USSD_CI-v<DDMMYYYY-HHMMSS>.xlsx`

## 4. Points de vigilance

- Les modules SIM800C sont en USB, donc accessibles comme ports série COM sous Windows.
- La communication avec les modules nécessite l'envoi de commandes AT (ex: `AT+CMGF=1`, `AT+CMGS`, `AT+CUSD`).
- Les codes USSD universels (`#99#`, `*#06#`) ne dépendent pas de l'opérateur.
- L'exploration récursive des menus USSD nécessite de gérer les sessions USSD (commande `AT+CUSD=1,"code",15`).
- La mise à jour de l'Excel nécessite une lecture/écriture fiable du fichier .xlsx (utilisation bibliothèque Go comme `excelize`).

## 5. Schéma fonctionnel global

```mermaid
graph TD
    A[Modules SIM800C USB] -->|Ports COM5-7| B[Backend Go]
    B -->|WebSocket| C[Frontend Web]
    B -->|Lecture/Ecriture| D[MySQL]
    B -->|Lecture/Mise à jour| E[Codes_USSD_CI.xlsx]
    C -->|Commandes utilisateur| B
    D -->|Stockage historique| B
    E -->|Fournit codes USSD| B