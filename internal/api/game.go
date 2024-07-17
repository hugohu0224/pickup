package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"pickup/internal/game"
	"pickup/internal/global"
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
	// get userId
	userId, err := c.Cookie("userId")
	if err != nil {
		zap.S().Error("failed to get user id", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user id"})
		return
	}

	// get roomId
	roomId, err := c.Cookie("roomId")
	if err != nil || len(roomId) == 0 {
		zap.S().Error("failed to get roomId", zap.String("roomId", c.Query("room_id")))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get roomId"})
		return
	}

	// get hub
	hub := global.HubManager.GetHubById(roomId)
	if hub == nil {
		zap.S().Error("failed to get hub")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get hub"})
		return
	}

	// http upgrade
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.S().Error("websocket upgrade failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade connection"})
		return
	}
	zap.S().Debugf("websocket connected to server %v\n", conn.RemoteAddr())

	// init Client
	client := game.NewClient(userId, hub, conn)

	hub.ClientManager.RegisterClient(client)
	defer hub.ClientManager.RemoveClient(client)

	serveWs(client)
}

func serveWs(client *game.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		client.Conn.Close()
		cancel()
	}()

	zap.S().Infof("stast to serve client %v", client.ID)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		client.ReadPump(ctx)
	}()

	go func() {
		defer wg.Done()
		client.WritePump(ctx)
	}()

	wg.Wait()
}

func GetGamePage(c *gin.Context) {
	roomId := c.Query("roomId")
	c.SetCookie("roomId", roomId, 3600, "/", global.Dv.GetString("DOMAIN"), true, true)
	c.HTML(http.StatusOK, "game.html", gin.H{})
}

func GetGameRoom(c *gin.Context) {
	c.HTML(http.StatusOK, "room.html", gin.H{})
}
