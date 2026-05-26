#!/bin/bash
# ============================================================
# SIM800C Supervisor — Installation service systemd (Linux/Ubuntu)
# Usage: sudo ./scripts/install_service.sh
# ============================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

SERVICE_NAME="sim800c-supervisor"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
BINARY="${APP_DIR}/sim800c-supervisor"

echo -e "${CYAN}========================================"
echo "  SIM800C Supervisor"
echo "  Installation service systemd"
echo -e "========================================${NC}"
echo

# Vérifier les droits root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[ERREUR]${NC} Ce script doit être exécuté en tant que root"
    echo "  Commande : sudo $0"
    exit 1
fi

# Lire le port depuis .env
SERVER_PORT=8082
if [ -f "${APP_DIR}/.env" ]; then
    PORT_FROM_ENV=$(grep -E "^SERVER_PORT=" "${APP_DIR}/.env" | cut -d'=' -f2 | tr -d '[:space:]' | head -1)
    [ -n "$PORT_FROM_ENV" ] && SERVER_PORT="$PORT_FROM_ENV"
fi

# Déterminer l'utilisateur qui exécutera le service
SERVICE_USER="${SUDO_USER:-$(logname 2>/dev/null || echo 'root')}"
echo "  Utilisateur du service : ${SERVICE_USER}"
echo "  Répertoire de l'app   : ${APP_DIR}"
echo "  Port serveur          : ${SERVER_PORT}"
echo

# -----------------------------------------------
# ÉTAPE 1 : Compiler le binaire si nécessaire
# -----------------------------------------------
echo "[1/4] Vérification du binaire..."
if [ ! -f "$BINARY" ]; then
    echo "  Compilation en cours..."
    if ! command -v go > /dev/null 2>&1; then
        echo -e "${RED}[ERREUR]${NC} Go n'est pas installé."
        echo "  Installation : sudo apt-get install golang-go"
        exit 1
    fi
    cd "$APP_DIR"
    sudo -u "$SERVICE_USER" go build -o sim800c-supervisor ./cmd/
    echo -e "  ${GREEN}[OK]${NC} Binaire compilé"
else
    echo -e "  ${GREEN}[OK]${NC} Binaire existant : ${BINARY}"
fi

# -----------------------------------------------
# ÉTAPE 2 : Permissions USB serial
# -----------------------------------------------
echo
echo "[2/4] Configuration des permissions USB..."
if ! groups "$SERVICE_USER" | grep -q '\bdialout\b'; then
    usermod -aG dialout "$SERVICE_USER"
    echo -e "  ${GREEN}[OK]${NC} Utilisateur '${SERVICE_USER}' ajouté au groupe 'dialout'"
    echo -e "  ${YELLOW}[INFO]${NC} Reconnectez-vous ou exécutez : newgrp dialout"
else
    echo -e "  ${GREEN}[OK]${NC} Utilisateur déjà dans le groupe 'dialout'"
fi

# -----------------------------------------------
# ÉTAPE 3 : Créer le fichier de service systemd
# -----------------------------------------------
echo
echo "[3/4] Création du service systemd..."

# Arrêter le service existant si nécessaire
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
systemctl disable "$SERVICE_NAME" 2>/dev/null || true

# Créer le fichier de service
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=SIM800C Supervisor — Service de supervision modules GSM
Documentation=file://${APP_DIR}/README.md
After=network.target mysql.service mariadb.service
Wants=mysql.service

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${APP_DIR}
ExecStart=${BINARY}
Restart=on-failure
RestartSec=5s
StandardOutput=append:${APP_DIR}/storage/logs/runtime.log
StandardError=append:${APP_DIR}/storage/logs/runtime.log

# Variables d'environnement
EnvironmentFile=${APP_DIR}/.env

# Limites
LimitNOFILE=65536
TimeoutStartSec=30
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF

echo -e "  ${GREEN}[OK]${NC} Fichier de service créé : ${SERVICE_FILE}"

# -----------------------------------------------
# ÉTAPE 4 : Activer et démarrer le service
# -----------------------------------------------
echo
echo "[4/4] Activation et démarrage du service..."

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"
sleep 3

if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo -e "  ${GREEN}[OK]${NC} Service démarré et activé au boot"
    echo
    echo -e "${CYAN}========================================"
    echo "  Service installé avec succès !"
    echo "  URL : http://localhost:${SERVER_PORT}"
    echo
    echo "  Commandes utiles :"
    echo "    sudo systemctl status ${SERVICE_NAME}"
    echo "    sudo systemctl restart ${SERVICE_NAME}"
    echo "    sudo systemctl stop ${SERVICE_NAME}"
    echo "    sudo journalctl -u ${SERVICE_NAME} -f"
    echo -e "========================================${NC}"
else
    echo -e "${RED}[ERREUR]${NC} Le service n'a pas démarré correctement"
    echo "  Vérifiez les logs : sudo journalctl -u ${SERVICE_NAME} -n 50"
    systemctl status "$SERVICE_NAME" --no-pager || true
    exit 1
fi
