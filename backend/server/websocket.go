package server

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)


func (h *Hub) Run() {
	for {
		select {
		case client := <- hub.register:
			h.mu.Lock()
			h.clients[client.uuid] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s (%s)", client.Username, client.uuid)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.uuid]; ok {
				delete(h.clients, client.uuid)
				close(client.Send)
				log.Printf("Client disconnected: %s (%s)", client.Username, client.uuid)
			}
			h.mu.Unlock()
		case message := <- h.broadcast:
			h.mu.Lock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.uuid)
				}
			}
			h.mu.Unlock()
		}
		
	}
}
func InitWebSocket() gin.HandlerFunc {
	return func(c *gin.Context) {

		username := c.Query("username")

		if username == "" {
			username = "anonymous"
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println("Websocket upgrade error: ", err)
			return
		}

		client := &Client{
			uuid:     uuid.New(),
			Socket:   conn,
			Send:     make(chan []byte, 256),
			Username: username,
		}

		board.mu.RLock()
		boardState := BoardState{
			Type:   "init",
			Pixels: board.Pixels,
		}
		initialState, _ := json.Marshal(boardState)
		board.mu.RUnlock()

		client.Send <- initialState

		go client.
	}
}
