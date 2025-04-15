package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 15) / 10
	maxMessageSize = 512
	boardWidth     = 1000
	boardHeight    = 1000
)

type Colour struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

type Pixel struct {
	Colour Colour `json:"colour"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

type Board struct {
	Width  int
	Height int
	Pixels [boardHeight][boardWidth]Pixel
	mu     sync.RWMutex
}

type Client struct {
	uuid     uuid.UUID
	Socket   *websocket.Conn
	Send     chan Update
	Username string
}

type InitBoardState struct {
	Pixels [boardHeight][boardWidth]Pixel `json:"pixels"`
}
type Update struct {
	Pixel      Pixel     `json:"pixel"`
	SenderUUID uuid.UUID `json:"-"`
}

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Update
	mu         sync.RWMutex
}

var (
	HubInstance = &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Update),
	}
	board = &Board{
		Width:  boardWidth,
		Height: boardHeight,
	}

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)
