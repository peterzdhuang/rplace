package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/peterzdhuang/rplace/backend/server"
)

func main() {

	go server.

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
	fmt.Println("Server starting on :8080")
	r.GET("/ws", server.InitWebSocket())
	r.Run(":8000")
}
