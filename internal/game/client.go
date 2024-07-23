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
			zap.S().Debugf("ReadPump msg: %v", gameMsg)

			position.ID = c.ID
			c.Hub.PositionChan <- position
		case models.ItemActionType:
			itemAction, err := gameMsgContentSwaper[models.ItemAction](&gameMsg)
			if err != nil {
				return err
			}
			itemAction.ID = c.ID
			c.Hub.ActionChan <- itemAction

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
			zap.S().Debugf("WritePump msg: %v", msg)
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
