# Analyse du document de description système

## 1. Objectif global du système
Le projet consiste à construire une plateforme de supervision et d'automatisation USSD/SMS autour de plusieurs modules GSM SIM800C USB connectés à un ordinateur Windows.

Le système devra :
- détecter automatiquement les modules GSM,
- communiquer en temps réel avec les ports COM,
- exécuter des commandes AT,
- envoyer/exécuter des requêtes USSD,
- gérer les SMS,
- explorer automatiquement les menus USSD,
- mettre à jour dynamiquement une base de connaissances USSD,
- fournir une interface web moderne en temps réel.

## 2. Technologies imposées
### Backend
- GoLang
- Communication série (COM5, COM6, COM7)
- WebSocket temps réel
- API REST
- Gestion multi-thread / concurrente

### Frontend
- Application Web
- Mode clair/sombre
- Dashboard temps réel
- Responsive UI
- Amélioration UI via impeccable `/polish`

### Base de données
- MySQL

## 3. Infrastructure matérielle détectée
- PC Windows
- USB Hub 3.0 alimenté
- 3 modules SIM800C USB
- 3 cartes SIM ivoiriennes

## 4. Fonctionnalités principales identifiées

### Fonction 1 — Module Auto-Discovery
Objectif :
- détecter automatiquement tous les modules SIM800C,
- identifier les ports COM,
- lire les informations SIM,
- afficher les données dans un dashboard temps réel.

Informations récupérées :
- IMEI
- IMSI
- ICCID
- numéro téléphone
- opérateur
- qualité réseau
- état SIM
- niveau signal

---

### Fonction 2 — SIM Status Discovery
#### Mode manuel
- génération dynamique des boutons USSD,
- filtrage par opérateur,
- exécution à la demande,
- affichage temps réel.

#### Mode automatique
- exécution automatique de tous les codes "Consulter",
- collecte des informations,
- mise à jour du dashboard.

---

### Fonction 3 — USSD Menu Discovery
#### Mode manuel
- exploration des menus USSD,
- navigation automatique,
- sauvegarde des résultats,
- détection des nouveaux sous-menus.

#### Mode automatique
- exploration récursive complète,
- enrichissement automatique du fichier Excel,
- versionning automatique des fichiers.

---

### Fonction 4 — USSD Manager
- saisie manuelle des codes USSD,
- sélection du module,
- affichage temps réel.

---

### Fonction 5 — SMS Manager
- lecture SMS,
- suppression SMS,
- envoi SMS,
- corbeille automatique,
- filtrage automatique des messages.

## 5. Contraintes métier identifiées

### Validation des entrées
Le système devra :
- nettoyer les données,
- valider les numéros,
- vérifier les montants,
- contrôler les PIN,
- vérifier les choix menu.

### Contraintes télécom CI
Préfixes :
- Orange : 07
- MTN : 05
- Moov : 01

Longueur :
- 10 chiffres.

## 6. Contraintes temps réel
Le système doit :
- recevoir les événements modem instantanément,
- pousser les données au frontend via WebSocket,
- supporter plusieurs exécutions simultanées.

## 7. Risques techniques identifiés
- blocage USSD opérateur,
- timeout réseau,
- verrouillage port série,
- perte connexion USB,
- limitations firmware SIM800C,
- différences comportement opérateurs,
- menus USSD dynamiques.

## 8. Besoins implicites détectés
Le document implique également :
- système de logs,
- gestion d'erreurs,
- sécurité API,
- gestion sessions,
- monitoring,
- reprise automatique,
- historique des opérations.

## 9. Architecture logique implicite
Le projet nécessite :
- couche Hardware,
- couche Driver GSM,
- moteur USSD,
- moteur SMS,
- moteur découverte,
- API backend,
- frontend temps réel,
- stockage historique.

## 10. Conclusion
Le système demandé est une plateforme GSM/USSD temps réel avancée orientée :
- supervision,
- automatisation,
- exploration USSD,
- collecte d'informations,
- enrichissement intelligent de données.

Le projet nécessite :
- forte robustesse,
- architecture concurrente,
- gestion événementielle,
- sécurité,
- scalabilité.
