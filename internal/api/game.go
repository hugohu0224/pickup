package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"pickup/internal/game"
	"pickup/internal/global"
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
	zap.S().Infof("websocket connected to server %v\n", conn.RemoteAddr())

	client := &game.Client{
		ID: "test",
		// TODO: replace hardcode Hub id
		Hub:    global.HubManager.GetHubById("room1"),
		Conn:   conn,
		Action: make(chan *models.Action),
		Done:   make(chan struct{}),
	}
	client.Hub.ClientManager.RegisterClient(client)
	defer client.Hub.ClientManager.RemoveClient(client)
	go serveWs(client)
}

func serveWs(client *game.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		client.Conn.Close()
		cancel()
	}()

	zap.S().Infof("stast to serve client %v", client.ID)

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

func GetGameRoom(c *gin.Context) {
	c.HTML(http.StatusOK, "room.html", gin.H{})
}
