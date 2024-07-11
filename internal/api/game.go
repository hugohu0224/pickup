package api

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"pickup/internal/game"
	"pickup/pkg/models"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64,
	WriteBufferSize: 64,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GetWebSocketURL(c *gin.Context) {
	url := "ws://localhost:8080/v1/game/ws"
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func WebsocketEndpoint(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade connection"})
		return
	}

	fmt.Printf("websocket connected to server %v\n", conn.RemoteAddr())

	client := &game.Client{
		ID:     "test",
		Hub:    nil,
		Conn:   conn,
		Action: make(chan models.Action),
		Done:   make(chan struct{}),
	}

	go serveWs(client)

}

func serveWs(client *game.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		client.Conn.Close()
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.ReadPump(ctx)
	}()

	wg.Wait()

}

func GetGamePage(c *gin.Context) {
	c.HTML(http.StatusOK, "game.html", gin.H{})
}
