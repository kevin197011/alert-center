package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type AlertMessage struct {
	Type      string                 `json:"type"`
	AlertID   string                 `json:"alert_id"`
	RuleID    string                 `json:"rule_id"`
	RuleName  string                 `json:"rule_name"`
	Severity  string                 `json:"severity"`
	Status    string                 `json:"status"` // firing, resolved
	Labels    map[string]interface{} `json:"labels"`
	Message   string                 `json:"message"`
	Timestamp time.Time             `json:"timestamp"`
}

type Client struct {
	conn   *websocket.Conn
	send   chan AlertMessage
	filter map[string]string
}

type WebSocketServer struct {
	clients  map[*Client]bool
	broadcast chan AlertMessage
	register chan *Client
	unregister chan *Client
	mu       sync.RWMutex
}

var wsServer *WebSocketServer

func initWebSocketServer() {
	wsServer = &WebSocketServer{
		clients:   make(map[*Client]bool),
		broadcast:  make(chan AlertMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go wsServer.run()
}

func (s *WebSocketServer) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.mu.Unlock()

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

func (s *WebSocketServer) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan AlertMessage, 256),
	}

	wsServer.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		wsServer.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		json.Unmarshal(message, &msg)

		if msg["type"] == "subscribe" {
			c.filter = msg["filter"].(map[string]string)
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

			data, _ := json.Marshal(message)
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
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

func BroadcastAlert(alert AlertMessage) {
	if wsServer != nil {
		wsServer.broadcast <- alert
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.ReadInConfig()
}

func main() {
	initConfig()
	initWebSocketServer()

	router := gin.Default()
	router.GET("/ws", wsServer.HandleWebSocket)

	addr := fmt.Sprintf(":%d", viper.GetInt("app.websocket_port"))
	log.Printf("WebSocket server starting on %s", addr)
	log.Fatal(router.Run(addr))
}

