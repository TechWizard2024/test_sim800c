#!/bin/bash
# ============================================================
# SIM800C Supervisor — Configuration environnement de tests Linux/Ubuntu
# Usage: ./scripts/test_setup.sh
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

echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor"
echo "  Configuration environnement de tests"
echo -e "========================================${NC}"
echo

# Lire les paramètres DB depuis .env
DB_HOST="localhost"
DB_PORT="3306"
DB_USER="root"
DB_PASSWORD=""
DB_NAME_TEST="sim800c_test"

if [ -f ".env" ]; then
    _V=$(grep -E "^DB_HOST=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_HOST="$_V"
    _V=$(grep -E "^DB_PORT=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_PORT="$_V"
    _V=$(grep -E "^DB_USER=" .env | cut -d'=' -f2 | tr -d '[:space:]'); [ -n "$_V" ] && DB_USER="$_V"
    _V=$(grep -E "^DB_PASSWORD=" .env | cut -d'=' -f2); [ -n "$_V" ] && DB_PASSWORD="${_V// /}"
fi

MYSQL_CMD="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
[ -n "$DB_PASSWORD" ] && MYSQL_CMD="${MYSQL_CMD} -p${DB_PASSWORD}"

# -----------------------------------------------
# ÉTAPE 1 : Base de données de test
# -----------------------------------------------
echo "[1/3] Création de la base de données de test..."

if $MYSQL_CMD -e "CREATE DATABASE IF NOT EXISTS ${DB_NAME_TEST} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null; then
    echo -e "  ${GREEN}[OK]${NC} Base de données '${DB_NAME_TEST}' prête"
else
    echo -e "  ${YELLOW}[WARN]${NC} Impossible de créer la base de test"
    echo "  Commande manuelle : mysql -u${DB_USER} -e 'CREATE DATABASE ${DB_NAME_TEST};'"
fi

echo

# -----------------------------------------------
# ÉTAPE 2 : Variable d'environnement TEST_DB_DSN
# -----------------------------------------------
echo "[2/3] Configuration de TEST_DB_DSN..."

DSN_VALUE="${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME_TEST}?parseTime=true"
export TEST_DB_DSN="$DSN_VALUE"

echo "  TEST_DB_DSN=${DSN_VALUE}"
echo
echo -e "  ${YELLOW}[INFO]${NC} Pour exporter cette variable dans votre shell :"
echo "  export TEST_DB_DSN=\"${DSN_VALUE}\""
echo
echo -e "  Ou ajoutez dans ~/.bashrc / ~/.zshrc :"
echo "  export TEST_DB_DSN=\"${DSN_VALUE}\""

echo

# -----------------------------------------------
# ÉTAPE 3 : Lancer les tests
# -----------------------------------------------
echo "[3/3] Lancement des tests Go..."
echo

# Tests du validateur USSD (sans DB)
echo -e "  ${CYAN}--- Tests validateur USSD (sans DB) ---${NC}"
go test ./internal/ussd/ -v -count=1 2>&1 | tail -20 || true

echo
echo -e "  ${CYAN}--- Tests DB (avec ${DB_NAME_TEST}) ---${NC}"
TEST_DB_DSN="$DSN_VALUE" go test ./internal/db/ -v -count=1 2>&1 | tail -30 || true

echo
echo -e "  ${CYAN}--- Couverture globale ---${NC}"
TEST_DB_DSN="$DSN_VALUE" go test ./internal/... -cover 2>&1 | tail -15 || true

echo
echo -e "${GREEN}========================================"
echo "  Configuration de tests terminée !"
echo -e "========================================${NC}"
echo
echo "  Commandes utiles :"
echo "    Tous les tests  : TEST_DB_DSN=\"${DSN_VALUE}\" go test ./internal/... -v"
echo "    Tests DB        : TEST_DB_DSN=\"${DSN_VALUE}\" go test ./internal/db/ -v"
echo "    Tests USSD      : go test ./internal/ussd/ -v"
echo "    Couverture      : TEST_DB_DSN=\"${DSN_VALUE}\" go test ./internal/... -cover"
