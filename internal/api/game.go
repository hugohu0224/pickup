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
	hub := game.Hm.GetHubById(roomId)
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

	// new Client
	client := game.NewClient(userId, hub, conn)

	// init client info to hub
	hub.ClientManager.RegisterClient(client)
	hub.InitStartPosition(client)
	hub.SendObstaclesToClient(client)
	hub.SendAllItemToClient(client)
	hub.SendAllPositionToClient(client)
	serveWs(client)
}

func serveWs(client *game.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	errChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		if err := client.ReadPump(ctx); err != nil {
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := client.WritePump(ctx); err != nil {
			errChan <- err
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			zap.S().Errorf("Error in client pump: %v", err)
			break
		}
	}

	client.Hub.CleanupClient(client)
	zap.S().Infof("finished serving client %v", client.ID)
}

func GetGamePage(c *gin.Context) {
	roomId := c.Query("roomId")
	c.SetCookie("roomId", roomId, 3600, "/", global.Dv.GetString("DOMAIN"), false, true)
	c.HTML(http.StatusOK, "game.html", gin.H{})
}

func GetGameRoom(c *gin.Context) {
	c.HTML(http.StatusOK, "room.html", gin.H{})
}
