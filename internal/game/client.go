package game

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
)

type Client struct {
	ID   string
	Hub  *Hub
	Conn *websocket.Conn
	Send chan *models.GameMsg
	Done chan struct{}
	mu   sync.Mutex
}

func NewClient(id string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:   id,
		Hub:  hub,
		Conn: conn,
		Send: make(chan *models.GameMsg, 128),
		Done: make(chan struct{}),
	}
}

func gameMsgContentSwapper[T any](gameMsg *models.GameMsg) (*T, error) {
	var structInstance T
	contentBytes, err := json.Marshal(gameMsg.Content)
	if err != nil {
		return nil, fmt.Errorf("gameMsg.Content json marshal failed: %w", err)
	}
	if err := json.Unmarshal(contentBytes, &structInstance); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}
	return &structInstance, nil
}

func (c *Client) ReadPump(ctx context.Context) error {
	zap.S().Infof("ReadPump start Client: %v", c.ID)
	defer c.Hub.ClientManager.RemoveClient(c)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ReadPump for client %s stopped due to context cancellation", c.ID)
		default:
			var gameMsg models.GameMsg
			if err := c.Conn.ReadJSON(&gameMsg); err != nil {
				return fmt.Errorf("failed to read JSON: %w", err)
			}

			if err := c.handleGameMsg(&gameMsg); err != nil {
				return fmt.Errorf("failed to handle game message: %w", err)
			}
		}
	}
}

func (c *Client) handleGameMsg(gameMsg *models.GameMsg) error {
	switch gameMsg.Type {
	case models.PlayerPositionType:
		return c.handlePlayerPosition(gameMsg)
	case models.ItemActionType:
		return c.handleItemAction(gameMsg)
	case models.PlayerChatMsgType:

		return nil
	default:
		return fmt.Errorf("invalid gameMsg type: %v", gameMsg.Type)
	}
}

func (c *Client) handlePlayerPosition(gameMsg *models.GameMsg) error {
	position, err := gameMsgContentSwapper[models.PlayerPosition](gameMsg)
	if err != nil {
		return err
	}
	position.ID = c.ID
	c.Hub.PositionChan <- position
	return nil
}

func (c *Client) handleItemAction(gameMsg *models.GameMsg) error {
	itemAction, err := gameMsgContentSwapper[models.ItemAction](gameMsg)
	if err != nil {
		return err
	}
	itemAction.ID = c.ID
	c.Hub.ActionChan <- itemAction
	return nil
}

func (c *Client) WritePump(ctx context.Context) error {
	zap.S().Infof("WritePump start Client: %v", c.ID)
	defer c.Conn.Close()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("WritePump for client %s stopped due to context cancellation", c.ID)
		case msg, ok := <-c.Send:
			if !ok {
				return c.writeCloseMessage()
			}
			if err := c.Conn.WriteJSON(msg); err != nil {
				return fmt.Errorf("failed to write JSON: %w", err)
			}
		}
	}
}

func (c *Client) writeCloseMessage() error {
	return c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
