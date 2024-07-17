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

func NewClient(id string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:   id,
		Hub:  hub,
		Conn: conn,
		Send: make(chan *models.GameMsg, 256),
		Done: make(chan struct{}),
	}
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
	zap.S().Infof("ReadPump start Client: %v\n", c.ID)
	for {
		var gameMsg models.GameMsg
		err := c.Conn.ReadJSON(&gameMsg)
		if err != nil {
			zap.S().Errorf("error reading gameMsg: %v", err)
			c.Hub.ClientManager.RemoveClient(c)
			return
		}
		switch gameMsg.Type {
		case models.PlayerPositionType:
			position, err := gameMsgContentSwaper[models.PlayerPosition](&gameMsg)
			if err != nil {
				return
			}
			zap.S().Debugf("ReadPump playerPosition: %v", position)

			// inject Player ID
			position.ID = c.ID

			c.Hub.PositionChan <- position
		case models.PlayerActionType:
		case models.PlayerChatMsgType:
		default:
			c.Hub.ClientManager.RemoveClient(c)
			zap.S().Errorf("invalid gameMsg type: %v", gameMsg.Type)
		}
	}
}

func (c *Client) WritePump(ctx context.Context) {
	zap.S().Infof("WritePump start Client: %v\n", c.ID)
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			zap.S().Infof("WritePump for client %s stopped due to context cancellation", c.ID)
			return
		case msg, ok := <-c.Send:
			zap.S().Debugf("WritePump by %v msg: %v", c.ID, msg.Content)
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

func (cm *ClientManager) Broadcast(msg *models.GameMsg) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for client := range cm.clients {
		select {
		case client.Send <- msg:
		default:
			close(client.Send)
			delete(cm.clients, client)
		}
	}
}
