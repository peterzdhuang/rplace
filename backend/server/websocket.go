package server

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (h *Hub) Run() {
	for {
		select {
		case client := <-Hub.register:
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

		case message := <-h.broadcast:
			h.mu.RLock()
			for uuid, client := range h.clients {
				if uuid != message.SenderUUID {
					select {
					case client.Send <- message:
					default:
						go func(c *Client) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()
		}

	}
}

func (c *Client) Read() {
	defer func() {
		Hub.unregister <- c
		c.Socket.Close()
	}()

	c.Socket.SetReadLimit(maxMessageSize)
	c.Socket.SetReadDeadline(time.Now().Add(pongWait))
	c.Socket.SetPongHandler(func(string) error { c.Socket.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {

		var msg Update
		err := c.Socket.ReadJSON(&msg)

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client ReadPump Error (%s): %v", c.uuid, err)
			} else {
				log.Printf("Client ReadPump: Normal closure or read error for %s: %v", c.uuid, err)
			}
			break // Exit loop on error
		}
		log.Printf("Client ReadPump: Received message from %s: Pos=%v, Rgb=%v", c.uuid, msg.Pos, msg.Rgb)

		msg.SenderUUID = c.uuid
		Hub.broadcast <- msg

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
		boardState := InitBoardState{
			Type:   "init",
			Pixels: board.Pixels,
		}
		initialState, _ := json.Marshal(boardState)
		board.mu.RUnlock()

		client.Send <- initialState

		go client.Read()
	}
}
