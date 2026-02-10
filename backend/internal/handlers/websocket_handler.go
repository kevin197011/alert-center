package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"alert-center/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	clients map[string]*Client
	mu      sync.RWMutex
	broadcast chan []byte
}

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients:   make(map[string]*Client),
		broadcast: make(chan []byte, 256),
	}
}

func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	userIDVal, _ := c.Get("user_id")
	usernameVal, _ := c.Get("username")
	userID := fmt.Sprintf("anon-%s", uuid.New().String()[:8])
	if id, ok := userIDVal.(string); ok && id != "" {
		userID = id
	}
	username := ""
	if name, ok := usernameVal.(string); ok {
		username = name
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	h.mu.Lock()
	h.clients[userID] = client
	h.mu.Unlock()

	log.Printf("WebSocket client connected: %s (%s)", username, userID)

	go client.writePump()
	go client.readPump(h)
}

func (h *WebSocketHandler) RemoveClient(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		close(client.send)
		delete(h.clients, userID)
		log.Printf("WebSocket client disconnected: %s", userID)
	}
}

func (h *WebSocketHandler) Broadcast(message WebSocketMessage) {
	data, _ := json.Marshal(message)
	h.broadcast <- data
}

func (h *WebSocketHandler) SendToUser(userID string, message WebSocketMessage) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		data, _ := json.Marshal(message)
		select {
		case client.send <- data:
		default:
			h.RemoveClient(userID)
		}
	}
}

func (h *WebSocketHandler) HandleBroadcast() {
	for {
		message := <-h.broadcast
		h.mu.RLock()
		for _, client := range h.clients {
			select {
			case client.send <- message:
			default:
				h.RemoveClient(client.userID)
			}
		}
		h.mu.RUnlock()
	}
}

func (c *Client) readPump(h *WebSocketHandler) {
	defer func() {
		h.RemoveClient(c.userID)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type AlertNotification struct {
	AlertID    string            `json:"alert_id"`
	RuleID     string            `json:"rule_id"`
	RuleName   string            `json:"rule_name"`
	Severity   string            `json:"severity"`
	Status     string            `json:"status"`
	Labels     map[string]string `json:"labels"`
	Timestamp  time.Time         `json:"timestamp"`
}

func (h *WebSocketHandler) SendAlertNotification(notification *services.AlertNotification) {
	message := WebSocketMessage{
		Type:    "alert",
		Payload: notification,
	}
	h.Broadcast(message)
}

func (h *WebSocketHandler) SendSLABreachNotification(notification *services.SLABreachNotification) {
	message := WebSocketMessage{
		Type:    "sla_breach",
		Payload: notification,
	}
	h.Broadcast(message)
}

func (h *WebSocketHandler) SendTicketNotification(notification *services.TicketNotification) {
	message := WebSocketMessage{
		Type:    "ticket",
		Payload: notification,
	}
	h.Broadcast(message)
}
