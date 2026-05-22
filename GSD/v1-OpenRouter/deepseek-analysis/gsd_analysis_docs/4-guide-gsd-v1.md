Copy

Download

```

\---
\### Task 4 - Guide d'utilisation du skill GSD
\*\*Document :\*\* \`4-guide-gsd-v1.md\`
\`\`\`markdown
\# 4. Guide d'utilisation du skill GSD (GSD-2)
\## 1. Installation de GSD
\### 1.1. Prérequis
\- Node.js 18+ et npm installés
\- Git installé
\- Accès à internet pour cloner le dépôt
\### 1.2. Cloner et installer GSD
\`\`\`bash
git clone https://github.com/gsd-build/gsd-2.git
cd gsd-2
npm install
```

### 1.3. Configurer l'environnement

bash

Copy

Download

```
cp .env.example .env
\# Éditer .env avec vos configurations
```

## 2\. Configuration du projet pour GSD

### 2.1. Structure attendue pour le mode auto GSD

GSD s'attend à trouver un fichier de configuration à la racine de votre projet : 
```
gsd.config.json
```

json

Copy

Download

```
{
  "projectName": "sim800c-supervisor",
  "version": "1.0.0",
  "type": "fullstack",
  "techStack": {
    "frontend": "html/css/js",
    "backend": "go",
    "database": "mysql"
  },
  "sourcePath": "C:/xampp/htdocs/aa\_Toolbox/test\_sim800c/blackbox/v1",
  "entryPoints": {
    "backend": "cmd/main.go",
    "frontend": "web/index.html"
  },
  "approvalMode": "auto-approve",
  "steps": \[
    "analyze",
    "design",
    "code-generation",
    "testing",
    "deployment"
  \]
}
```

### 2.2. Fichier de description du projet pour GSD

Créez 
```
project.md
```
 à la racine du projet avec le contenu de 
```
project_desc.txt
```
 reformaté en Markdown.

### 2.3. Stockage des documents d'analyse

GSD utilisera automatiquement le dossier 
```
docs/
```
 pour stocker les analyses. Créez-le :

bash

Copy

Download

```
mkdir \-p docs/analysis
```

## 3\. Exécution en mode auto avec approbation automatique

### 3.1. Commande principale

bash

Copy

Download

```
\# Depuis le dossier du projet
cd C:/xampp/htdocs/aa\_Toolbox/test\_sim800c/blackbox/v1
\# Lancer GSD avec mode auto-approve
node /chemin/vers/gsd-2/index.js @gsd /gsd auto --approval-mode\=auto-approve
```

### 3.2. Alternative avec fichier de réponse

Créez un fichier 
```
gsd-response.txt
```
 :

text

Copy

Download

```
/gsd auto
gsd.approvalMode:auto-approve
/gsd execute project.md
```

Puis :

bash

Copy

Download

```
node /chemin/vers/gsd-2/index.js < gsd-response.txt
```

## 4\. Niveau d'autonomie 
```
auto-approve
```
 - Détails

Avec 
```
gsd.approvalMode:"auto-approve"
```
, GSD va :

 Étape | Comportement |
| --- | --- |
 Analyse des besoins | ✅ Auto, sans confirmation |
 Proposition d'architecture | ✅ Auto |
 Génération du code backend Go | ✅ Auto |
 Génération du code frontend | ✅ Auto |
 Création des scripts SQL | ✅ Auto |
 Écriture des fichiers de configuration | ✅ Auto |
 Exécution des tests unitaires | ✅ Auto |
 Déploiement (si configuré) | ⚠️ Demande confirmation si écriture hors projet |

### 4.1. Ce que GSD génère automatiquement

-   Structure complète du projet Go (cmd, internal, pkg)
    
-   Fichiers handlers API REST
    
-   WebSocket Hub
    
-   Communication série (
    ```
    serial/
    ```
    )
    
-   Modules USSD et SMS
    
-   Lecture/écriture Excel avec 
    ```
    excelize
    ```
    
-   Frontend HTML/CSS/JS avec thème clair/sombre
    
-   Scripts SQL pour MySQL
    
-   Fichiers de configuration (
    ```
    config.yaml
    ```
    , 
    ```
    .env
    ```
    )
    
-   Dockerfile (optionnel)

## 5\. Personnalisation pour ce projet spécifique

### 5.1. Ajouter une règle de validation personnalisée

Créez 
```
gsd-rules.yaml
```
 :

yaml

Copy

Download

```
customValidation:
  \- name: "validate\_phone\_number\_ci"
    pattern: "^0\[157\]\[0-9\]{8}$"
    error: "Numéro CI invalide (doit commencer par 01,05,07 et 10 chiffres)"
  
  \- name: "validate\_pin"
    pattern: "^\[0-9\]{4}$"
    error: "PIN doit être 4 chiffres"
fileTemplates:
  \- source: "templates/sim800c.go.tmpl"
    destination: "internal/serial/sim800c.go"
  \- source: "templates/ussd\_explorer.go.tmpl"
    destination: "internal/ussd/explorer.go"
```

### 5.2. Forcer l'inclusion de bibliothèques spécifiques

Dans 
```
gsd.config.json
```
 :

json

Copy

Download

```
"dependencies": {
  "go": \[
    "github.com/tarm/serial",
    "github.com/xuri/excelize/v2",
    "github.com/gorilla/websocket",
    "github.com/go-sql-driver/mysql"
  \]
}
```

## 6\. Résultats attendus après exécution GSD

### 6.1. Arborescence générée

text

Copy

Download

```
C:\\xampp\\htdocs\\aa\_Toolbox\\test\_sim800c\\blackbox\\v1\\
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   ├── serial/
│   ├── ussd/
│   ├── sms/
│   ├── excel/
│   ├── db/
│   ├── websocket/
│   └── api/
├── web/
│   ├── index.html
│   ├── css/
│   ├── js/
│   └── assets/
├── scripts/
│   ├── init\_db.sql
│   └── install\_service.bat
├── docs/
│   ├── 1-analyse\_documents\_de\_description\_systeme.md
│   ├── 2-information\_additionelle\_analyse.md
│   ├── 3-design\_system\_infrastructure-v1.md
│   └── 3-design\_system\_applications-v1.md
├── gsd.config.json
├── config.yaml
├── go.mod
└── README.md
```

### 6.2. Rapport d'exécution

GSD générera 
```
gsd-report.html
```
 contenant :

-   Chronologie des actions
    
-   Fichiers créés/modifiés
    
-   Erreurs éventuelles
    
-   Tests exécutés et résultats

## 7\. Résolution des problèmes courants

 Problème | Solution |
| --- | --- |
 GSD ne trouve pas les ports COM | Ajouter 
```
"windowsComPorts": ["COM5","COM6","COM7"]
```
 dans 
```
gsd.config.json
```
 |
 Génération Excel incomplète | Vérifier que 
```
excelize
```
 est bien dans les dépendances |
 WebSocket non généré | Spécifier 
```
"realtime": "websocket"
```
 dans la section 
```
features
```
 |
 Thème clair/sombre absent | Ajouter 
```
"themes": ["light","dark"]
```
 dans 
```
gsd.config.json
```
 |
 Erreur sur 
```
@polish
```
 | GSD utilise sa propre génération CSS, pas besoin de commande externe |

## 8\. Après la génération GSD

### 8.1. Compilation et exécution manuelle

bash

Copy

Download

```
cd C:\\xampp\\htdocs\\aa\_Toolbox\\test\_sim800c\\blackbox\\v1
go mod tidy
go build \-o sim800c.exe cmd/main.go
.\\sim800c.exe \--config config.yaml
```

### 8.2. Installation comme service Windows

bash

Copy

Download

```
\# Exécuter en tant qu'Administrateur
scripts\\install\_service.bat
```

### 8.3. Accès à l'application

-   Frontend : http://test\_sim800c.local (après configuration du vhost)
    
-   API : [http://localhost:8080/api/modules](http://localhost:8080/api/modules)
    
-   WebSocket : ws://localhost:8080/ws

## 9\. Commandes GSD utiles

 Commande | Description |
| --- | --- |
 
```
/gsd status
```
 | Vérifier l'état de GSD |
 
```
/gsd analyze project.md
```
 | Analyser sans générer |
 
```
/gsd generate --component=backend
```
 | Générer seulement le backend |
 
```
/gsd test
```
 | Lancer les tests générés |
 
```
/gsd deploy --env=production
```
 | Déployer (si configuré) |
 
```
/gsd clean
```
 | Nettoyer les fichiers générés |

## 10\. Exemple complet d'exécution

bash

Copy

Download

```
\# 1. Préparer l'environnement
mkdir \-p C:\\xampp\\htdocs\\aa\_Toolbox\\test\_sim800c\\blackbox\\v1
cd C:\\xampp\\htdocs\\aa\_Toolbox\\test\_sim800c\\blackbox\\v1
\# 2. Copier les fichiers nécessaires
cp ../project\_desc.txt .
cp ../storage/excel/Codes\_USSD\_CI.xlsx .
\# 3. Créer gsd.config.json (voir section 2.1)
\# 4. Lancer GSD
node D:\\tools\\gsd-2\\index.js @gsd /gsd auto --approval-mode\=auto-approve
\# 5. Vérifier la génération
dir
\# 6. Compiler et lancer
go mod tidy
go build \-o sim800c.exe cmd/main.go
start sim800c.exe \--config config.yaml
```