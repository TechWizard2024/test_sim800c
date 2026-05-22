package handlers

import (
	"net/http"

	"sim800c-supervisor/internal/auth"
	"sim800c-supervisor/internal/websocket"

	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebSocketHandler struct {
	hub    *websocket.Hub
	logger *logrus.Logger
	auth   interface {
		ValidateToken(string) (*auth.Claims, error)
	}
	upgrader gorillaWebsocket.Upgrader
}

func NewWebSocketHandler(hub *websocket.Hub, logger *logrus.Logger, authManager interface {
	ValidateToken(string) (*auth.Claims, error)
}) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
		auth:   authManager,
		upgrader: gorillaWebsocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En production, restreindre les origines
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Auth obligatoire via header Authorization: Bearer <JWT>
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		http.Error(w, "Token manquant", http.StatusUnauthorized)
		return
	}
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	claims, err := h.auth.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "Token invalide", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("Erreur upgrade WebSocket: %v", err)
		return
	}

	client := &websocket.Client{
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: claims.UserID,
	}

	h.hub.RegisterClient(client)
	go client.WritePump()
	go client.ReadPump()

	h.logger.Infof("Nouveau client WebSocket connecté: user_id=%s", claims.UserID)
}
