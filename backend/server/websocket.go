package server

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (h *Hub) Run() {

	board.InitBoard()
	for {
		select {
		case client := <-h.register:
			log.Printf("DEBUG: Registering client %s (%s)", client.Username, client.uuid)
			h.mu.Lock()
			h.clients[client.uuid] = client
			h.mu.Unlock()
			log.Printf("Client connected: %s (%s)", client.Username, client.uuid)
		case client := <-h.unregister:
			log.Printf("DEBUG: Unregistering client %s (%s)", client.Username, client.uuid)
			h.mu.Lock()
			if _, ok := h.clients[client.uuid]; ok {
				delete(h.clients, client.uuid)
				close(client.Send)
				log.Printf("Client disconnected: %s (%s)", client.Username, client.uuid)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			log.Printf("DEBUG: Broadcasting message from %s: %+v", message.SenderUUID, message)

			h.mu.RLock()
			for uuid, client := range h.clients {
				if uuid != message.SenderUUID {
					select {
					case client.Send <- message:
						log.Printf("DEBUG: Sent message to client %s", client.uuid)
					default:
						log.Printf("DEBUG: Client %s send channel blocked, unregistering", client.uuid)
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
		log.Printf("DEBUG: Exiting Read loop for client %s", c.uuid)
		HubInstance.unregister <- c
		c.Socket.Close()
	}()

	log.Printf("DEBUG: Starting Read loop for client %s", c.uuid)

	c.Socket.SetReadLimit(maxMessageSize)
	c.Socket.SetReadDeadline(time.Now().Add(pongWait))
	c.Socket.SetPongHandler(func(string) error {
		log.Printf("DEBUG: Received pong from client %s", c.uuid)
		c.Socket.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		log.Printf("DEBUG: Waiting for next message from client %s", c.uuid)
		var msg Update
		err := c.Socket.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client ReadPump Error (%s): %v", c.uuid, err)
			} else {
				log.Printf("Client ReadPump: Normal closure or read error for %s: %v", c.uuid, err)
			}
			break
		}
		log.Printf("DEBUG: Received message from client %s", c.uuid)
		msg.SenderUUID = c.uuid
		msg.Type = "update"

		HubInstance.broadcast <- msg
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Printf("DEBUG: Exiting Write loop for client %s", c.uuid)
		ticker.Stop()
		c.Socket.Close()
	}()

	log.Printf("DEBUG: Starting Write loop for client %s", c.uuid)

	for {
		select {
		case message, ok := <-c.Send:
			log.Printf("DEBUG: Write loop - message received for client %s", c.uuid)
			c.Socket.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				log.Printf("Client WritePump: Hub closed send channel for %s", c.uuid)
				c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			log.Printf("DEBUG: Write message from %s: %+v", message.SenderUUID, message)
			err := c.Socket.WriteJSON(message)
			if err != nil {
				log.Printf("Client WritePump Error (%s): %v", c.uuid, err)
				return
			}
		case <-ticker.C:
			log.Printf("DEBUG: Sending ping to client %s", c.uuid)
			c.Socket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Client WritePump: Ping Error (%s): %v", c.uuid, err)
				return
			}
		}
	}
}

func InitWebSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("DEBUG: Upgrading connection to WebSocket")
		username := c.Query("username")
		if username == "" {
			username = "anonymous"
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("Websocket upgrade error:", err)
			return
		}
		client := &Client{
			uuid:     uuid.New(),
			Socket:   conn,
			Send:     make(chan Update, 256),
			Username: username,
		}
		HubInstance.clients[client.uuid] = client
		log.Printf("DEBUG: New client created: %s (%s)", client.Username, client.uuid)

		board.mu.RLock()
		boardState := InitBoardState{
			Type:   "init",
			Pixels: board.Pixels,
		}
		board.mu.RUnlock()

		log.Printf("DEBUG: Sending initial board state to client %s", client.uuid)
		client.Socket.WriteJSON(boardState)

		go client.Read()
		go client.Write()
	}
}
