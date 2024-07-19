package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
	"time"
)

type Client struct {
	ID              string
	Hub             *Hub
	Conn            *websocket.Conn
	Action          chan *models.Action
	Send            chan *models.GameMsg
	Done            chan struct{}
	DefaultPosition models.Position
	mu              sync.Mutex
}

func NewClient(id string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:     id,
		Hub:    hub,
		Conn:   conn,
		Action: nil,
		Send:   make(chan *models.GameMsg, 128),
		Done:   make(chan struct{}),
		mu:     sync.Mutex{},
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
func (c *Client) ReadPump(ctx context.Context) error {
	zap.S().Infof("ReadPump start Client: %v\n", c.ID)

	for {
		var gameMsg models.GameMsg
		err := c.Conn.ReadJSON(&gameMsg)
		if err != nil {
			c.Hub.ClientManager.RemoveClient(c)
			return err
		}
		switch gameMsg.Type {
		case models.PlayerPositionType:
			position, err := gameMsgContentSwaper[models.PlayerPosition](&gameMsg)
			if err != nil {
				return err
			}
			zap.S().Debugf("ReadPump playerPosition: %v", position)

			// inject Player ID into broadcast msg
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

func (c *Client) WritePump(ctx context.Context) error {
	zap.S().Infof("WritePump start Client: %v\n", c.ID)
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			errors.New(fmt.Sprintf("WritePump for client %s stopped due to context cancellation", c.ID))
		case msg, ok := <-c.Send:
			zap.S().Debugf("WritePump by %v msg: %v", c.ID, msg.Content)
			if !ok {
				errors.New(fmt.Sprintf("failed to send message to client: %v", c.ID))
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			}
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				return err
			}
		}
	}
}

type ClientManager struct {
	clients     map[*Client]bool
	clientsById map[string]*Client
	mu          sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:     make(map[*Client]bool),
		clientsById: make(map[string]*Client),
		mu:          sync.RWMutex{},
	}
}

func (cm *ClientManager) RegisterClient(client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[client] = true
	cm.clientsById[client.ID] = client
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

func (cm *ClientManager) GetClientByID(id string) *Client {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.clientsById[id]
}

func (cm *ClientManager) BroadcastAll(msg *models.GameMsg) {
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

func (cm *ClientManager) SendToClient(userId string, msg *models.GameMsg) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	client := cm.GetClientByID(userId)
	if client == nil {
		zap.S().Errorf("client %s not found", userId)
		return
	}

	select {
	case client.Send <- msg:
	case <-time.After(2 * time.Second):
		zap.S().Warnf("Timeout sending message to client %s", userId)
	}
}
