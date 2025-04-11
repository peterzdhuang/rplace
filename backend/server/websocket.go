package server

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
		initialState := json.Marshal(boardState)
		board.mu.RUnlock()

		client <- initialState
	}
}
