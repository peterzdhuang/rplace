package server

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Colour struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type Pixel struct {
	Username string `json:"username"`
	Colour   Colour `json:"colour"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

type Board struct {
	Width  int
	Height int
	Pixels map[string]*Pixel
	mu     sync.RWMutex
}

type Client struct {
	uuid     uuid.UUID
	Socket   *websocket.Conn
	Send     chan []byte
	Username string
}

type BoardState struct {
	Type   string            `json:"type"`
	Pixels map[string]*Pixel `json:"pixels"`
	uuid   uuid.UUID         `json:"uuid"`
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.Mutex
}

var (
	HubInstance = &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
	board = &Board{
		Width:  1000,
		Height: 1000,
		Pixels: make(map[string]*Pixel),
	}

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)
