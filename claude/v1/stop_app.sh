#!/bin/bash
# ============================================================
# SIM800C Supervisor — Script d'arrêt Linux/Ubuntu
# Usage: ./stop_app.sh
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor - Arrêt"
echo -e "========================================${NC}"
echo

echo "[1/2] Arrêt de l'application Go..."

STOPPED=0

# Méthode 1 : via le fichier .pid
if [ -f ".pid" ]; then
    PID=$(cat .pid)
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
        kill -SIGTERM "$PID" 2>/dev/null
        sleep 2
        if kill -0 "$PID" 2>/dev/null; then
            kill -SIGKILL "$PID" 2>/dev/null
        fi
        echo -e "  ${GREEN}[OK]${NC} Application arrêtée (PID: ${PID})"
        STOPPED=1
    fi
fi

# Méthode 2 : par nom de processus
if pkill -f "sim800c-supervisor" 2>/dev/null; then
    echo -e "  ${GREEN}[OK]${NC} Processus sim800c-supervisor arrêté"
    STOPPED=1
fi

if [ "$STOPPED" -eq 0 ]; then
    echo "  [INFO] Application non trouvée (déjà arrêtée ?)"
fi

echo
echo "[2/2] Nettoyage..."
rm -f .pid
echo -e "  ${GREEN}[OK]${NC} Nettoyage terminé"

echo
echo -e "${CYAN}========================================"
echo "  Application arrêtée avec succès"
echo -e "========================================${NC}"
