#!/bin/bash
# ============================================================
# SIM800C Supervisor — Script de déploiement Linux/Ubuntu
# Usage: ./scripts/deploy.sh [--build-only] [--no-service]
# ============================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$APP_DIR"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

BUILD_ONLY=0
NO_SERVICE=0

for ARG in "$@"; do
    case "$ARG" in
        --build-only) BUILD_ONLY=1 ;;
        --no-service) NO_SERVICE=1 ;;
    esac
done

echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor"
echo "  Déploiement Linux/Ubuntu"
echo -e "========================================${NC}"
echo

# -----------------------------------------------
# Lecture de la configuration
# -----------------------------------------------
SERVER_PORT=8082
DB_NAME="sim800c_manager_deepseekv1"
DB_USER="root"
DB_PASSWORD=""
DB_HOST="localhost"
DB_PORT="3306"

if [ -f ".env" ]; then
    _V=$(grep -E "^SERVER_PORT=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && SERVER_PORT="$_V"
    _V=$(grep -E "^DB_NAME=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_NAME="$_V"
    _V=$(grep -E "^DB_USER=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_USER="$_V"
    _V=$(grep -E "^DB_PASSWORD=" .env | cut -d'=' -f2); [ -n "$_V" ] && DB_PASSWORD="${_V// /}"
    _V=$(grep -E "^DB_HOST=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_HOST="$_V"
    _V=$(grep -E "^DB_PORT=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_PORT="$_V"
fi

MYSQL_CMD="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
[ -n "$DB_PASSWORD" ] && MYSQL_CMD="${MYSQL_CMD} -p${DB_PASSWORD}"

echo "  Port        : ${SERVER_PORT}"
echo "  Base de données : ${DB_NAME}"
echo

# -----------------------------------------------
# ÉTAPE 1 : Vérification des prérequis
# -----------------------------------------------
echo "[1/5] Vérification des prérequis..."

MISSING=0
for CMD in go mysql; do
    if ! command -v "$CMD" > /dev/null 2>&1; then
        echo -e "  ${RED}[MANQUANT]${NC} $CMD"
        MISSING=1
    else
        echo -e "  ${GREEN}[OK]${NC} $CMD : $(${CMD} version 2>/dev/null | head -1 || true)"
    fi
done

if [ "$MISSING" -eq 1 ]; then
    echo
    echo "  Installez les prérequis manquants :"
    echo "    Go      : sudo apt-get install golang-go"
    echo "    MySQL   : sudo apt-get install mysql-server"
    exit 1
fi

echo

# -----------------------------------------------
# ÉTAPE 2 : Compilation
# -----------------------------------------------
echo "[2/5] Compilation du projet Go..."

go build -ldflags="-s -w" -o sim800c-supervisor ./cmd/
echo -e "  ${GREEN}[OK]${NC} Binaire compilé : sim800c-supervisor"
echo "  Taille : $(du -h sim800c-supervisor | cut -f1)"

if [ "$BUILD_ONLY" -eq 1 ]; then
    echo
    echo -e "${GREEN}  Build uniquement terminé.${NC}"
    exit 0
fi

echo

# -----------------------------------------------
# ÉTAPE 3 : Base de données
# -----------------------------------------------
echo "[3/5] Configuration de la base de données..."

if $MYSQL_CMD -e "USE ${DB_NAME}" 2>/dev/null; then
    echo -e "  ${GREEN}[OK]${NC} Base de données ${DB_NAME} existante"
    for SQL_FILE in scripts/migrate_v1-13.sql scripts/migrate_v1-25.sql; do
        if [ -f "$SQL_FILE" ]; then
            $MYSQL_CMD "$DB_NAME" < "$SQL_FILE" 2>/dev/null && \
                echo -e "  ${GREEN}[OK]${NC} Migration $(basename $SQL_FILE) appliquée" || true
        fi
    done
else
    echo "  Initialisation de la base de données..."
    $MYSQL_CMD < scripts/init_db.sql 2>/dev/null && \
        echo -e "  ${GREEN}[OK]${NC} Base de données initialisée" || \
        echo -e "  ${YELLOW}[WARN]${NC} Vérifiez scripts/init_db.sql manuellement"
fi

echo

# -----------------------------------------------
# ÉTAPE 4 : Structure des dossiers
# -----------------------------------------------
echo "[4/5] Structure des dossiers..."

mkdir -p storage/logs storage/excel storage/backup
chmod 755 storage storage/logs storage/excel storage/backup
echo -e "  ${GREEN}[OK]${NC} Dossiers storage créés"

# Vérifier le fichier Excel
if [ ! -f "storage/excel/Codes_USSD_CI.xlsx" ]; then
    echo -e "  ${YELLOW}[WARN]${NC} Codes_USSD_CI.xlsx absent de storage/excel/"
    echo "  Copiez le fichier Excel dans storage/excel/"
fi

echo

# -----------------------------------------------
# ÉTAPE 5 : Démarrage / Service
# -----------------------------------------------
echo "[5/5] Démarrage de l'application..."

if [ "$NO_SERVICE" -eq 0 ] && [ "$EUID" -eq 0 ]; then
    echo "  Installation comme service systemd..."
    bash scripts/install_service.sh
else
    # Redémarrage direct
    ./stop_app.sh 2>/dev/null || true
    sleep 1
    nohup ./sim800c-supervisor > storage/logs/runtime.log 2>&1 &
    echo $! > .pid
    sleep 3
    if kill -0 "$(cat .pid)" 2>/dev/null; then
        echo -e "  ${GREEN}[OK]${NC} Application démarrée (PID: $(cat .pid))"
    else
        echo -e "  ${RED}[ERREUR]${NC} Démarrage échoué - voir storage/logs/runtime.log"
        exit 1
    fi
fi

echo
echo -e "${CYAN}========================================"
echo "  Déploiement terminé !"
echo "  URL : http://localhost:${SERVER_PORT}"
echo "  Connexion : admin / admin123"
echo -e "========================================${NC}"
