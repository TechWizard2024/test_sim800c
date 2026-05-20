# Guide GSD v1 — Exécution automatique du projet

# 1. Objectif
Utiliser GSD pour :
- générer automatiquement le projet,
- construire l’architecture,
- générer le frontend,
- générer le backend,
- améliorer le design,
- automatiser la structure du système.

# 2. Références
- GSD : https://github.com/gsd-build/gsd-2
- Impeccable : https://github.com/pbakaus/impeccable

# 3. Structure cible

C:\xampp\htdocs\aa_Toolbox\test_sim800c\GSD\v1-OpenRouter

# 4. Préparation environnement

## Installer NodeJS
Vérifier :
```bash
node -v
npm -v
```

## Installer GSD
```bash
npx get-shit-done-cc@latest
```

# 5. Initialisation projet

```bash
cd C:\xampp\htdocs\aa_Toolbox\test_sim800c
```

```bash
mkdir GSD
cd GSD
mkdir v1-OpenRouter
cd v1-OpenRouter
```

# 6. Fichier de configuration recommandé

## gsd.config.json
```json
{
  "gsd.approvalMode": "auto-approve"
}
```

# 7. Prompt principal GSD

## Prompt recommandé
```text
Construire une plateforme GSM USSD/SMS professionnelle utilisant :
- Backend GoLang
- Frontend React
- WebSocket temps réel
- MySQL
- Modules SIM800C USB

Fonctionnalités :
- auto discovery modem
- exploration USSD
- SMS manager
- dashboard temps réel
- dark/light mode
- logs
- monitoring
- sécurité
- architecture scalable

Optimiser :
- performance
- sécurité
- modularité
- maintenabilité
```

# 8. Lancement GSD

```bash
@gsd /gsd auto
```

# 9. Paramètres autonomie

```text
gsd.approvalMode:"auto-approve"
```

# 10. Utilisation impeccable

## Amélioration design frontend
```bash
/polish
```

# 11. Pipeline recommandé

## Étape 1
- générer architecture

## Étape 2
- générer backend Go

## Étape 3
- générer frontend React

## Étape 4
- générer websocket

## Étape 5
- générer UI/UX

## Étape 6
- générer tests

## Étape 7
- optimisation

# 12. Structure finale recommandée

backend/
frontend/
storage/
logs/
exports/
docs/

# 13. Recommandations GSD

## Toujours demander :
- clean architecture
- code scalable
- websocket temps réel
- gestion erreurs
- retry modem
- logs structurés
- sécurité JWT

# 14. Prompt avancé recommandé

```text
Construire une plateforme GSM industrielle temps réel :
- gestion multi SIM800C
- supervision temps réel
- moteur USSD intelligent
- moteur SMS
- exploration automatique USSD
- websocket
- React + Tailwind
- Go Fiber
- GORM
- MySQL

Le système doit être :
- scalable
- sécurisé
- modulaire
- maintenable
- professionnel
```

# 15. Conclusion
Le workflow GSD permettra :
- génération rapide,
- architecture cohérente,
- automatisation maximale,
- amélioration continue,
- qualité professionnelle.
