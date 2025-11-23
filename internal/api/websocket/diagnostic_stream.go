package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

type Client struct {
	hub   *Handler
	conn  *websocket.Conn
	send  chan []byte
	topics map[string]bool
}

type Handler struct {
	upgrader    websocket.Upgrader
	clients     map[*Client]bool
	clientsMu   sync.RWMutex
	broadcast   chan Message
	register    chan *Client
	unregister  chan *Client
	log         logger.Logger
}

type Message struct {
    Topic   string      `json:"topic"`
    Payload interface{} `json:"payload"`
}

func NewHandler(cfg config.WebSocketConfig) *Handler {
	return &Handler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all for now, should check origin in prod
			},
		},
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		log:        logger.NewLogger("websocket"),
	}
}

func (h *Handler) Run() {
	for {
		select {
		case client := <-h.register:
			h.clientsMu.Lock()
			h.clients[client] = true
			h.clientsMu.Unlock()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.clientsMu.Lock()
				delete(h.clients, client)
				close(client.send)
				h.clientsMu.Unlock()
			}
		case message := <-h.broadcast:
			h.clientsMu.RLock()
			for client := range h.clients {
                // Check if client is subscribed to topic
                if client.topics[message.Topic] || message.Topic == "all" {
                    select {
                    case client.send <- h.encodeMessage(message):
                    default:
                        close(client.send)
                        delete(h.clients, client)
                    }
                }
			}
			h.clientsMu.RUnlock()
		}
	}
}

func (h *Handler) encodeMessage(msg Message) []byte {
    b, _ := json.Marshal(msg)
    return b
}

func (h *Handler) ServeHTTP(c *gin.Context) {
    id := c.Param("id")
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Errorf("Failed to upgrade websocket: %v", err)
		return
	}

	client := &Client{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, 256),
        topics: map[string]bool{id: true},
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
        // Handle incoming messages if needed (e.g. subscribe to more topics)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (h *Handler) Broadcast(topic string, payload interface{}) {
    // Ensure Run() is running. In a real app, Run() should be started by Server.Start()
    // For now, I'll assume it's running.
    h.broadcast <- Message{Topic: topic, Payload: payload}
}
