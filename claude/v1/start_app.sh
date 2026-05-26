#!/bin/bash
# ============================================================
# SIM800C Supervisor — Script de démarrage Linux/Ubuntu
# Usage: ./start_app.sh
# ============================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Couleurs
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor v1"
echo "  Démarrage de l'application (Linux)"
echo -e "========================================${NC}"
echo

# -----------------------------------------------
# Lecture du port depuis .env
# -----------------------------------------------
SERVER_PORT=8082
if [ -f ".env" ]; then
    PORT_FROM_ENV=$(grep -E "^SERVER_PORT=" .env | cut -d'=' -f2 | tr -d '[:space:]' | head -1)
    if [ -n "$PORT_FROM_ENV" ]; then
        SERVER_PORT="$PORT_FROM_ENV"
    fi
fi
echo -e "  [INFO] Port serveur : ${SERVER_PORT}"

# -----------------------------------------------
# PRE-CHECK : Vérifier si déjà en cours d'exécution
# -----------------------------------------------
if pgrep -x "sim800c-supervisor" > /dev/null 2>&1; then
    echo -e "${YELLOW}[AVERT] sim800c-supervisor est déjà en cours d'exécution !${NC}"
    echo
    echo "  [O] Ouvrir navigateur   [N] Nouvelle instance   [A] Arrêter"
    read -r -p "  Votre choix (O/N/A) : " CHOICE
    case "${CHOICE^^}" in
        O)
            echo "  Ouverture du navigateur..."
            xdg-open "http://localhost:${SERVER_PORT}" 2>/dev/null || \
                echo "  Accédez manuellement à : http://localhost:${SERVER_PORT}"
            exit 0
            ;;
        A)
            echo "  Arrêt de l'instance existante..."
            ./stop_app.sh
            sleep 2
            ;;
        *)
            echo "  Démarrage d'une nouvelle instance..."
            ;;
    esac
fi

echo

# -----------------------------------------------
# PRE-CHECK : Vérifier si le port est libre
# -----------------------------------------------
if ss -tlnp 2>/dev/null | grep -q ":${SERVER_PORT} " || \
   netstat -tlnp 2>/dev/null | grep -q ":${SERVER_PORT} "; then
    echo -e "${YELLOW}[AVERT] Le port ${SERVER_PORT} est déjà occupé par un autre processus.${NC}"
    echo "  Vérifiez qu'aucune autre instance ne tourne."
    echo
fi

# -----------------------------------------------
# PRE-CHECK : Créer les dossiers nécessaires
# -----------------------------------------------
mkdir -p storage/logs storage/excel storage/backup
echo -e "  ${GREEN}[OK]${NC} Dossiers storage créés/vérifiés"

echo

# -----------------------------------------------
# ÉTAPE 1/4 : Vérification MySQL
# -----------------------------------------------
echo "[1/4] Vérification de MySQL..."
echo

MYSQL_RUNNING=0
if systemctl is-active --quiet mysql 2>/dev/null || \
   systemctl is-active --quiet mariadb 2>/dev/null || \
   pgrep -x mysqld > /dev/null 2>&1 || \
   pgrep -x mariadbd > /dev/null 2>&1; then
    echo -e "  ${GREEN}[OK]${NC} MySQL/MariaDB est en cours d'exécution"
    MYSQL_RUNNING=1
else
    echo -e "  ${YELLOW}[WARN]${NC} MySQL non détecté - tentative de démarrage..."
    if systemctl start mysql 2>/dev/null || systemctl start mariadb 2>/dev/null; then
        sleep 3
        echo -e "  ${GREEN}[OK]${NC} MySQL démarré avec succès"
        MYSQL_RUNNING=1
    else
        echo -e "  ${YELLOW}[WARN]${NC} Impossible de démarrer MySQL - vérifiez manuellement"
        echo "  Commande : sudo systemctl start mysql"
    fi
fi

echo

# -----------------------------------------------
# ÉTAPE 2/4 : Vérification base de données
# -----------------------------------------------
echo "[2/4] Vérification de la base de données..."
echo

# Lire les paramètres DB depuis .env
DB_HOST="localhost"
DB_PORT="3306"
DB_USER="root"
DB_PASSWORD=""
DB_NAME="sim800c_manager_deepseekv1"

if [ -f ".env" ]; then
    _V=$(grep -E "^DB_HOST=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_HOST="$_V"
    _V=$(grep -E "^DB_PORT=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_PORT="$_V"
    _V=$(grep -E "^DB_USER=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_USER="$_V"
    _V=$(grep -E "^DB_PASSWORD=" .env | cut -d'=' -f2); [ -n "$_V" ] && DB_PASSWORD="${_V// /}"
    _V=$(grep -E "^DB_NAME=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_NAME="$_V"
fi

MYSQL_CMD="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
[ -n "$DB_PASSWORD" ] && MYSQL_CMD="${MYSQL_CMD} -p${DB_PASSWORD}"

if [ "$MYSQL_RUNNING" -eq 1 ]; then
    if $MYSQL_CMD -e "SELECT 1" "$DB_NAME" > /dev/null 2>&1; then
        echo -e "  ${GREEN}[OK]${NC} Base de données ${DB_NAME} accessible"
        # Appliquer migrations
        for SQL_FILE in scripts/migrate_v1-13.sql scripts/migrate_v1-25.sql; do
            if [ -f "$SQL_FILE" ]; then
                $MYSQL_CMD "$DB_NAME" < "$SQL_FILE" 2>/dev/null && \
                    echo -e "  ${GREEN}[OK]${NC} Migration $(basename $SQL_FILE) appliquée" || true
            fi
        done
    else
        echo -e "  ${YELLOW}[WARN]${NC} Base de données inaccessible - initialisation..."
        if [ -f "scripts/init_db.sql" ]; then
            if $MYSQL_CMD < scripts/init_db.sql 2>/dev/null; then
                echo -e "  ${GREEN}[OK]${NC} Base de données initialisée"
            else
                echo -e "  ${YELLOW}[WARN]${NC} Initialisation échouée - vérifiez scripts/init_db.sql"
                echo "  Commande manuelle : mysql -u${DB_USER} < scripts/init_db.sql"
            fi
        fi
    fi
else
    echo "  [SKIP] MySQL inactif - base de données ignorée"
fi

echo

# -----------------------------------------------
# ÉTAPE 3/4 : Détection des modules SIM800C USB
# -----------------------------------------------
echo "[3/4] Détection des modules SIM800C USB..."
echo

USB_SERIAL_PORTS=$(ls /dev/ttyUSB* /dev/ttyACM* 2>/dev/null || true)
if [ -n "$USB_SERIAL_PORTS" ]; then
    echo "  Ports USB détectés :"
    for PORT in $USB_SERIAL_PORTS; do
        echo -e "    ${GREEN}[DÉTECTÉ]${NC} $PORT"
        # Ajouter l'utilisateur courant au groupe dialout si nécessaire
        if ! groups "$USER" | grep -q '\bdialout\b'; then
            echo -e "  ${YELLOW}[WARN]${NC} L'utilisateur '$USER' n'est pas dans le groupe 'dialout'"
            echo "  Pour corriger : sudo usermod -aG dialout $USER && newgrp dialout"
        fi
    done
else
    echo "  [INFO] Aucun port USB-Serial détecté (/dev/ttyUSB*, /dev/ttyACM*)"
    echo "  [INFO] L'application continuera à scanner en arrière-plan"
fi

echo

# -----------------------------------------------
# ÉTAPE 4/4 : Démarrage de l'application Go
# -----------------------------------------------
echo "[4/4] Démarrage de l'application..."
echo

# Compiler si l'exécutable n'existe pas ou si les sources sont plus récentes
if [ ! -f "sim800c-supervisor" ] || \
   find cmd internal -name "*.go" -newer sim800c-supervisor 2>/dev/null | grep -q .; then
    echo "  Compilation en cours..."
    if command -v go > /dev/null 2>&1; then
        if go build -o sim800c-supervisor ./cmd/; then
            echo -e "  ${GREEN}[OK]${NC} Compilation réussie"
        else
            echo -e "  ${RED}[ERREUR]${NC} Compilation échouée - vérifiez les erreurs ci-dessus"
            exit 1
        fi
    else
        echo -e "  ${RED}[ERREUR]${NC} Go n'est pas installé ou pas dans le PATH"
        echo "  Installation : sudo apt-get install golang-go"
        echo "  Ou téléchargez depuis : https://go.dev/dl/"
        exit 1
    fi
fi

echo
echo -e "${CYAN}========================================"
echo "  Application en cours de démarrage..."
echo "  Frontend : http://localhost:${SERVER_PORT}"
echo "  Backend  : http://127.0.0.1:${SERVER_PORT}"
echo "  WebSocket: ws://localhost:${SERVER_PORT}/ws"
echo -e "========================================${NC}"
echo
echo "  Connexion par défaut : admin / admin123"
echo "  Appuyez sur Ctrl+C pour arrêter"
echo

# Lancer l'application en arrière-plan
nohup ./sim800c-supervisor > storage/logs/runtime.log 2>&1 &
APP_PID=$!
echo "$APP_PID" > .pid
echo -e "  ${GREEN}[OK]${NC} Application démarrée (PID: ${APP_PID})"

# Attendre que le serveur soit prêt
echo "  Attente du serveur (5 secondes)..."
sleep 5

# Vérifier que l'application tourne
if kill -0 "$APP_PID" 2>/dev/null; then
    echo -e "  ${GREEN}[OK]${NC} Serveur en écoute sur le port ${SERVER_PORT}"
    echo
    # Ouvrir le navigateur si disponible
    if command -v xdg-open > /dev/null 2>&1; then
        xdg-open "http://localhost:${SERVER_PORT}" 2>/dev/null &
        echo "  Navigateur ouvert automatiquement"
    else
        echo "  Accédez à : http://localhost:${SERVER_PORT}"
    fi
else
    echo -e "  ${RED}[ERREUR]${NC} L'application ne semble pas avoir démarré correctement"
    echo "  Vérifiez les logs dans storage/logs/runtime.log :"
    tail -20 storage/logs/runtime.log 2>/dev/null || true
    exit 1
fi

echo
echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor est actif !"
echo "  Logs : storage/logs/runtime.log"
echo -e "========================================${NC}"
echo

# Afficher les logs en temps réel
echo "  --- Logs en temps réel (Ctrl+C pour quitter) ---"
tail -f storage/logs/runtime.log
