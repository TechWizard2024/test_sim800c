package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *logrus.Logger
}

type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	UserID   string
	ModuleID int
}

type Event struct {
	Type      string      `json:"type"`
	ModuleID  int         `json:"module_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logrus.New(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Infof("Client connecté. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			h.logger.Infof("Client déconnecté. Total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// RegisterClient - Enregistre un client dans le hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient - Désenregistre un client du hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastEvent - Diffuse un événement à tous les clients
func (h *Hub) BroadcastEvent(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Errorf("Erreur marshalling event: %v", err)
		return
	}
	h.broadcast <- data
}

// SendToModule - Envoie un événement à un module spécifique
func (h *Hub) SendToModule(moduleID int, event Event) {
	event.ModuleID = moduleID
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Errorf("Erreur marshalling event: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ModuleID == moduleID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// ReadPump - Lit les messages du client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.UnregisterClient(c)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Errorf("Erreur lecture WebSocket: %v", err)
			}
			break
		}

		// Traiter les messages du client
		var event Event
		if err := json.Unmarshal(message, &event); err == nil {
			c.Hub.logger.Debugf("Message reçu: %+v", event)
		}
	}
}

// WritePump - Écrit les messages vers le client
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
