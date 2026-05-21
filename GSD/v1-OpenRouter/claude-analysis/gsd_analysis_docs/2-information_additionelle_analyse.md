# Informations Additionnelles & Analyse Complémentaire
> Document de référence : `project_desc.txt`  
> Date : 20 Mai 2026

---

## PARTIE A — Informations Manquantes ou Imprécises

### A.1 — Couche Hardware / Communication Série

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| H1 | Débit baud (baud rate) des modules SIM800C USB | Critique — sans cela, la connexion série ne peut pas s'ouvrir | Par défaut SIM800C = **9600 bps**, mais certains modules USB-série le configurent à 115200. À confirmer ou rendre configurable. |
| H2 | Délai de timeout pour les commandes AT | Critique — un timeout trop court rate des réponses USSD lentes | Recommandé : 10–30s pour USSD, 5s pour commandes AT simples |
| H3 | Gestion du reconnect automatique si un module se débranche | Important — sans reconnect, le système plante silencieusement | Stratégie de reconnect avec retry exponentiel à définir |
| H4 | Comportement si une commande AT est envoyée pendant une session USSD en cours | Important — les sessions USSD sont stateful | File d'attente (queue) par module à implémenter |
| H5 | Encodage des réponses USSD (GSM7, UCS2) | Important — les réponses en UCS2 apparaissent en hexa sans décodage | Le backend doit détecter et décoder UCS2 → UTF-8 |
| H6 | Comportement à la mise sous tension (init séquence) | Moyen — le module doit être initialisé avant usage | Séquence AT d'init à standardiser |

---

### A.2 — Backend GoLang

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| B1 | Framework web GoLang souhaité | Moyen | Recommandé : **Gin** (performances, middleware, WebSocket facile avec `gorilla/websocket`) |
| B2 | Gestion des sessions USSD multi-étapes (menus imbriqués) | Critique — sans cela, la Fonction 3 est impossible | Implémenter une machine d'état (FSM) par session USSD par module |
| B3 | Gestion de la concurrence : les 3 modules sont actifs simultanément | Critique — goroutines Go dédiées par module | 1 goroutine de lecture + 1 goroutine d'écriture par module COM |
| B4 | Authentification de l'API | Moyen | API interne (localhost) → token statique ou pas d'auth, à préciser |
| B5 | Gestion des erreurs USSD (timeout opérateur, code invalide) | Important | Parser les codes d'erreur AT+CUSD et les afficher clairement |
| B6 | Logging applicatif | Moyen | Format de log, niveau (DEBUG/INFO/ERROR), rotation des fichiers |

---

### A.3 — Base de Données MySQL

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| D1 | Schéma de base de données | Critique — sans schéma, la DB n'est pas exploitable | Voir section B pour le schéma proposé |
| D2 | Credentials MySQL | Déploiement | XAMPP par défaut : root / (vide), port 3306 — à configurer via `.env` |
| D3 | Nom de la base de données | Déploiement | Recommandé : `sim800c_manager` |
| D4 | Politique de rétention des données (SMS, résultats USSD) | Moyen | Durée de conservation ? Archivage ? |
| D5 | Gestion de l'historique des versions Excel | Moyen | Stocker les références de chaque version générée en DB |

---

### A.4 — Frontend

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| F1 | Framework frontend (React, Vue, Vanilla JS, etc.) | Moyen | Description dit "Frontend web" sans préciser → Recommandé : **Vanilla JS + Tailwind CSS** ou **Vue.js** pour le temps réel |
| F2 | Mode d'accès multi-utilisateur (1 seul utilisateur local ou plusieurs) | Moyen | Vu l'URL locale, probablement mono-utilisateur |
| F3 | Comportement si le WebSocket se déconnecte | Important | Reconnect automatique avec indicateur visuel |
| F4 | Langue de l'interface | Moyen | Français (implicite vu le contexte) |
| F5 | Responsive design (mobile/tablette) | Faible | Priorité desktop vu l'usage local |

---

### A.5 — Gestion du Fichier Excel

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| E1 | Qui est propriétaire du fichier Excel (backend lit/écrit, ou frontend déclenche) | Important | Backend GoLang doit lire au démarrage et écrire lors des découvertes |
| E2 | Gestion des conflits d'écriture (2 modules découvrent en même temps) | Important | Mutex sur l'écriture du fichier Excel |
| E3 | Format de l'horodatage dans le nom du fichier | Précision | Exemple fourni : `v20052026-082405` = `vDDMMYYYY-HHMMSS` |
| E4 | Synchronisation de la DB MySQL avec le fichier Excel | Important | À la découverte, écrire en DB ET mettre à jour Excel |

---

### A.6 — Fonction SMS Manager (Fonction 5)

| # | Information manquante | Impact | Recommandation |
|---|---|---|---|
| S1 | Critère de filtrage de la corbeille : "ne contient pas le mot Test" — sensible à la casse ? | Important | Clarifier : "Test", "test", "TEST" ? Recommandé : insensible à la casse |
| S2 | Fréquence de polling des SMS entrants | Important | SIM800C supporte les notifications push SMS (AT+CNMI) — à préférer au polling |
| S3 | Capacité SMS de stockage sur la SIM (généralement 20-30 SMS) | Important | Surveiller l'espace disponible, alerter si plein |
| S4 | SMS sortants : numéros autorisés ? | Sécurité | Valider le format des numéros destinataires |

---

## PARTIE B — Aspects à Prendre en Compte pour un Système Complet et Professionnel

### B.1 — Architecture & Infrastructure

1. **Gestion multi-thread/goroutine** : chaque module SIM800C doit avoir ses propres goroutines de lecture/écriture pour éviter les blocages.
2. **File de commandes (Command Queue)** : file FIFO par module pour séquencer les commandes AT sans collision.
3. **Machine d'état USSD (FSM)** : gestion des sessions USSD multi-étapes (menu → sous-menu → réponse).
4. **WebSocket centralisé** : un hub WebSocket diffusant les événements en temps réel à tous les clients connectés.
5. **Gestion des ports COM dynamique** : détection automatique des ports COM disponibles (pas seulement COM5/6/7 codés en dur).
6. **Configuration via fichier `.env`** : baud rate, ports COM, credentials MySQL, timeouts, etc.
7. **Healthcheck et watchdog** : surveillance de l'état de chaque module et reconnexion automatique.

### B.2 — Sécurité

1. **Validation stricte de tous les inputs USSD** (cf. règles section A.3 du doc d'analyse).
2. **Pas d'injection de commandes AT** : les codes USSD doivent être validés avant envoi.
3. **Logs d'audit** : toutes les commandes AT envoyées et réponses reçues doivent être loggées.
4. **Protection des PIN** : les codes PIN saisis ne doivent pas apparaître en clair dans les logs.
5. **Rate limiting** : limiter la fréquence d'exécution des USSD pour éviter le blocage par l'opérateur.

### B.3 — Qualité et Fiabilité

1. **Retry logic** : en cas d'échec d'une commande AT, réessayer N fois avant de déclarer l'erreur.
2. **Timeout adaptatif** : les USSD peuvent mettre jusqu'à 30s sur certains opérateurs CI.
3. **Gestion du signal réseau** : vérifier la qualité du signal avant d'exécuter un USSD.
4. **Tests de connectivité** : vérifier que la SIM est bien enregistrée sur le réseau avant toute opération.
5. **État des modules persisté en DB** : pour retrouver l'état après redémarrage.

### B.4 — Expérience Utilisateur

1. **Dashboard en temps réel** : indicateurs visuels clairs (signal réseau, état de la SIM, dernière activité).
2. **Notifications** : alertes visuelles en cas d'erreur, de SMS reçu, ou de nouvelle découverte USSD.
3. **Historique** : historique paginé des commandes USSD exécutées et de leurs résultats.
4. **Export des résultats** : possibilité d'exporter les résultats en CSV ou Excel.
5. **Indicateurs de progression** : lors des explorations automatiques (Fonctions 2-2, 3-2), afficher la progression.

---

## PARTIE C — Fonctionnalités Additionnelles Recommandées

### C.1 — Monitoring & Observabilité

**Nom** : Module de Monitoring Système  
**Description** : Tableau de bord affichant en temps réel pour chaque module : niveau de signal réseau (CSQ en dBm), état d'enregistrement réseau (CREG), température approximative, statut de la SIM (PUK, PIN, prête), opérateur actif (COPS), type de réseau (2G/GPRS), version firmware du module, uptime depuis connexion.  
**Valeur ajoutée** : Permet de diagnostiquer immédiatement les problèmes de connectivité sans outil externe.

---

### C.2 — Journal d'Audit AT Commands

**Nom** : Console AT / Log en temps réel  
**Description** : Affichage en temps réel de toutes les commandes AT envoyées et de toutes les réponses reçues, par module, avec horodatage. Possibilité de filtrer par module et par type de commande. Export en fichier texte.  
**Valeur ajoutée** : Outil de débogage indispensable pour le développement et la maintenance.

---

### C.3 — Scheduler / Planificateur de Tâches

**Nom** : Task Scheduler  
**Description** : Permettre de programmer l'exécution automatique d'un code USSD ou d'un scan de statut à une heure donnée ou selon une fréquence (ex: vérifier le solde toutes les heures). Interface de gestion des tâches planifiées (créer, activer/désactiver, supprimer).  
**Valeur ajoutée** : Automatise la surveillance continue sans intervention manuelle.

---

### C.4 — Gestion des Profils de SIM

**Nom** : SIM Profile Manager  
**Description** : Associer un profil nommé à chaque SIM (nom, opérateur, numéro, rôle). Permettre de réassigner manuellement le profil si la SIM est remplacée. Historique des SIMs ayant occupé chaque slot de module.  
**Valeur ajoutée** : Facilite la gestion dans un contexte multi-SIM.

---

### C.5 — Notifications & Alertes

**Nom** : Système d'Alertes  
**Description** : Alertes configurables pour : SMS reçu contenant un mot-clé spécifique, solde en dessous d'un seuil, module déconnecté, signal réseau faible, erreur USSD répétée. Notifications affichées dans l'interface + optionnellement envoyées par SMS via un autre module.  
**Valeur ajoutée** : Transforme le système en outil de supervision proactive.

---

### C.6 — Templates USSD

**Nom** : USSD Template Manager  
**Description** : Créer et sauvegarder des séquences USSD pré-remplies avec des variables (ex: transfert d'argent vers un numéro favori). Exécution en un clic avec substitution des variables. Partage des templates entre modules.  
**Valeur ajoutée** : Accélère les opérations répétitives.

---

### C.7 — Comparaison Multi-Opérateurs

**Nom** : Vue Comparative Multi-SIM  
**Description** : Tableau comparatif affichant côte à côte les soldes, volumes internet et statuts de chaque SIM/module. Idéal pour comparer les offres opérateurs en temps réel.  
**Valeur ajoutée** : Vue globale immédiate sans naviguer entre les modules.

---

### C.8 — Import/Export & Synchronisation Excel

**Nom** : Excel Sync Manager  
**Description** : Interface pour importer manuellement une nouvelle version de `Codes_USSD_CI.xlsx`, visualiser les différences avec la version précédente, valider ou rejeter les changements. Historique complet des versions avec diff.  
**Valeur ajoutée** : Contrôle qualité sur les mises à jour du référentiel USSD.

---

### C.9 — API REST Documentée

**Nom** : API REST publique  
**Description** : Exposer toutes les fonctions via une API REST documentée (Swagger/OpenAPI). Permettre à d'autres applications locales d'interagir avec les modules SIM800C via l'API.  
**Valeur ajoutée** : Extensibilité et intégration avec d'autres outils.

---

### C.10 — Backup & Restore

**Nom** : Sauvegarde automatique  
**Description** : Export périodique (quotidien) de la DB MySQL + des fichiers Excel versionnés. Interface de restauration depuis un backup.  
**Valeur ajoutée** : Résilience des données.
