package handlers

import (
	"net/http"

	"sim800c-supervisor/internal/websocket"

	gorillaWebsocket "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebSocketHandler struct {
	hub      *websocket.Hub
	logger   *logrus.Logger
	upgrader gorillaWebsocket.Upgrader
}

func NewWebSocketHandler(hub *websocket.Hub, logger *logrus.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
		upgrader: gorillaWebsocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // En production, restreindre les origines
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Errorf("Erreur upgrade WebSocket: %v", err)
		return
	}

	client := &websocket.Client{
		Hub:  h.hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.hub.RegisterClient(client)
	go client.WritePump()
	go client.ReadPump()

	h.logger.Info("Nouveau client WebSocket connecté")
}
