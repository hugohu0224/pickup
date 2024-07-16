package game

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
)

type Client struct {
	ID     string
	Hub    *Hub
	Conn   *websocket.Conn
	Action chan *models.Action
	Send   chan *models.GameMsg
	Done   chan struct{}
	mu     sync.Mutex
}

func gameMsgContentSwaper[T any](gameMsg *models.GameMsg) (*T, error) {
	var structInstance T
	contentBytes, err := json.Marshal(gameMsg.Content)
	if err != nil {
		zap.S().Errorf("gameMsg.Content json marshal failed: %v", err)
		return nil, err
	}
	if err = json.Unmarshal(contentBytes, &structInstance); err != nil {
		zap.S().Errorf("json unmarshal failed: %v", err)
		return nil, err
	}
	return &structInstance, nil

}
func (c *Client) ReadPump(ctx context.Context) {
	for {
		var gameMsg models.GameMsg
		err := c.Conn.ReadJSON(&gameMsg)
		if err != nil {
			zap.S().Errorf("error reading gameMsg: %v", err)
			return
		}
		switch gameMsg.Type {
		case models.PlayerPositionType:
			position, err := gameMsgContentSwaper[models.PlayerPosition](&gameMsg)
			if err != nil {
				return
			}
			// inject Player ID
			position.ID = c.ID

			zap.S().Debugf("playerPosition: %v", position)

			c.Hub.PositionChan <- position
		case models.PlayerActionType:
		case models.PlayerChatMsgType:
		default:
			zap.S().Errorf("invalid gameMsg type: %v", gameMsg.Type)
		}
	}
}

func (c *Client) WritePump(ctx context.Context) {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			zap.S().Infof("WritePump for client %s stopped due to context cancellation", c.ID)
			return
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteJSON(msg)
		}
	}
}

type ClientManager struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[*Client]bool),
		mu:      sync.RWMutex{},
	}
}

func (cm *ClientManager) RegisterClient(client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[client] = true
}

func (cm *ClientManager) RemoveClient(client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, client)
}

func (cm *ClientManager) GetClients() map[*Client]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.clients
}
