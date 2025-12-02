package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kubestack-ai/kubestack-ai/internal/common/config"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	hub    *Handler
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.RWMutex
}

type Handler struct {
	upgrader   websocket.Upgrader
	clients    map[*Client]bool
	clientsMu  sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	log        logger.Logger
}

type Message struct {
	Topic   string      `json:"topic"`
	Payload interface{} `json:"payload"`
}

func NewHandler(cfg config.WebSocketConfig) *Handler {
	h := &Handler{
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
	// Start the run loop immediately in a goroutine
	go h.Run()
	return h
}

func (h *Handler) Run() {
	h.log.Info("WebSocket Hub started")
	for {
		select {
		case client := <-h.register:
			h.clientsMu.Lock()
			h.clients[client] = true
			h.clientsMu.Unlock()
			h.log.Debug("New client registered")

		case client := <-h.unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.log.Debug("Client unregistered")
			}
			h.clientsMu.Unlock()

		case message := <-h.broadcast:
            // Safely iterate over clients
			h.clientsMu.RLock()
            var disconnected []*Client
			for client := range h.clients {
				client.mu.RLock()
				subscribed := client.topics[message.Topic] || message.Topic == "all"
				client.mu.RUnlock()

				if subscribed {
					select {
					case client.send <- h.encodeMessage(message):
					default:
                        // Collect disconnected clients to remove later
						h.log.Warn("Client send buffer full, marking for disconnect")
                        disconnected = append(disconnected, client)
					}
				}
			}
			h.clientsMu.RUnlock()

            // Remove disconnected clients safely
            if len(disconnected) > 0 {
                h.clientsMu.Lock()
                for _, client := range disconnected {
                    if _, ok := h.clients[client]; ok {
                        delete(h.clients, client)
                        close(client.send)
                    }
                }
                h.clientsMu.Unlock()
            }
		}
	}
}

func (h *Handler) encodeMessage(msg Message) []byte {
	b, _ := json.Marshal(msg)
	return b
}

func (h *Handler) ServeHTTP(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		h.log.Warn("WebSocket connection attempt without ID")
		c.Status(http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Errorf("Failed to upgrade websocket: %v", err)
		return
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		topics: map[string]bool{id: true},
	}
	h.register <- client

	// Allow collection of memory stats by the runtime.
	// p.s. We should not block the handler.
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// log error
			}
			break
		}
		// We can add logic here to handle incoming messages (e.g. subscribe/unsubscribe)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Handler) Broadcast(topic string, payload interface{}) {
	h.broadcast <- Message{Topic: topic, Payload: payload}
}
