package game

import (
	"fmt"
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
)

var Hm *HubManager

type Hub struct {
	ID             string
	ClientManager  *ClientManager
	HubManager     *HubManager
	OccupiedInMap  sync.Map // map[positionString]*models.Position (for occupied check)
	ObstaclesInMap []*models.Position
	ItemsInMap     sync.Map // map[positionString]*models.ItemAction (for game actions)
	UsersInMap     sync.Map // map[userIdString]*models.Position (for player move validate)
	Scores         sync.Map // map[userIdString]int (player score storage)
	PositionChan   chan *models.PlayerPosition
	ActionChan     chan *models.ItemAction
	MsgChan        chan *models.ChatMsg
	CurrentRound   *GameRound
	mu             sync.RWMutex
	obstaclesMu    sync.RWMutex
}

func (h *Hub) RegisterClient(client *Client) bool {
	if oldClient, exists := h.ClientManager.GetClientByID(client.ID); exists {
		// sync client game state
		client.GameIsActive = oldClient.GameIsActive

		// change to new client conn
		h.ClientManager.clientsById[oldClient.ID] = client
		h.ClientManager.clients[client] = true
		h.ClientManager.UpdateClientConnStateById(client.ID, true)

		// no register
		zap.S().Debug("Client exists", zap.String("client", client.ID))
		return false
	} else {
		// register
		h.ClientManager.RegisterClient(client)
		zap.S().Debug("Client register", zap.String("client", client.ID))
		return true
	}
}

func (h *Hub) SendAllGameRoundStateToClient(client *Client) {
	client.Hub.SendObstaclesToClient(client)
	client.Hub.SendAllItemToClient(client)
	client.Hub.SendAllPlayerPositionToClient(client)
	client.Hub.SendAllScoresToClient(client)
}

func (h *Hub) SendAllItemToClient(client *Client) {
	h.ItemsInMap.Range(func(key, value interface{}) bool {
		msg := &models.GameMsg{
			Type:    "itemPosition",
			Content: value.(*models.ItemAction),
		}
		client.Send <- msg
		return true
	})
}

func (h *Hub) SendObstaclesToClient(client *Client) {
	for _, obstacle := range h.ObstaclesInMap {
		msg := &models.GameMsg{
			Type:    "obstaclePosition",
			Content: obstacle,
		}
		client.Send <- msg
	}
}

func (h *Hub) SendAllPlayerPositionToClient(client *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.UsersInMap.Range(func(key, value interface{}) bool {
		userId, ok := key.(string)
		if !ok {
			zap.S().Errorf("SendAllPositionsToClient error, userId type: %T", userId)
			return false
		}

		// skip self
		if userId == client.ID {
			return true
		}

		position, ok := value.(*models.Position)
		if !ok {
			zap.S().Errorf("SendAllPositionsToClient error, position type: %T", position)
			return false
		}

		playPosition := &models.PlayerPosition{
			Valid:    true,
			ID:       userId,
			Position: position,
		}

		msg := &models.GameMsg{
			Type:    models.PlayerPositionType,
			Content: playPosition,
		}

		client.Send <- msg

		return true
	})
}

func (h *Hub) GetClientManager() *ClientManager {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ClientManager
}

// CleanupClient removes all data associated with a client from the hub
func (h *Hub) CleanupClient(client *Client) {
	userId := client.ID

	position, err := h.GetPlayerPositionByUserId(userId)
	if err != nil {
		zap.S().Errorf("failed to get the current position for user %s, due to:%v", userId, err.Error())
	}
	// clean hub state
	h.OccupiedInMap.Delete(fmt.Sprintf("%d-%d", position.X, position.Y))
	h.UsersInMap.Delete(userId)
	h.Scores.Delete(userId)

	// clean clientManager state
	h.ClientManager.RemoveClient(client)
	client.Conn.Close()
	zap.S().Infof("cleaned up data for user %s in Hub %s", userId, h.ID)
}

func (h *Hub) Run() {
	zap.S().Infof("Hub %s is running", h.ID)

	go h.ManageGameRounds()

	for {
		select {
		case playerPosition := <-h.PositionChan:
			err := h.handlePositionUpdate(playerPosition)
			if err != nil {
				zap.S().Errorf("failed handling PlayerPotition due to: %s", err.Error())
			}
		case itemAction := <-h.ActionChan:
			err := h.handleItemAction(itemAction)
			if err != nil {
				zap.S().Errorf("failed handling ItemAction due to: %s", err.Error())
			}
		}
	}
}

func (h *Hub) broadcastSingleScore(userId string, score int) {
	msg := &models.GameMsg{
		Type: "score",
		Content: &models.ScoreUpdate{
			ID:    userId,
			Score: score,
		},
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) SendAllScoresToClient(client *Client) {
	h.Scores.Range(func(userId, score interface{}) bool {
		msg := &models.GameMsg{
			Type: "score",
			Content: &models.ScoreUpdate{
				ID:    userId.(string),
				Score: score.(int),
			},
		}
		client.Send <- msg
		return true
	})
}

func (h *Hub) sendErrorToClient(userId string, errorMsg string) {
	msg := &models.GameMsg{
		Type: models.ErrorType,
		Content: &models.Error{
			ID:    userId,
			Error: errorMsg,
		},
	}
	h.ClientManager.SendToClient(userId, msg)
}

func (h *Hub) sendAlertToUser(userId string, alertMsg string) {
	msg := &models.GameMsg{
		Type: models.AlertType,
		Content: &models.Alert{
			ID:   userId,
			Text: alertMsg,
		},
	}
	h.ClientManager.SendToClient(userId, msg)
}

func (h *Hub) broadcastCollectedItem(itemAction *models.ItemAction) {
	itemAction.Valid = true
	msg := &models.GameMsg{
		Type:    models.ItemCollectedType,
		Content: itemAction,
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) handleItemAction(itemAction *models.ItemAction) error {
	positionString := fmt.Sprintf("%d-%d", itemAction.Position.X, itemAction.Position.Y)
	itemInMap, err := h.getItemInMap(positionString)
	if err != nil {
		return fmt.Errorf("failed to get item in map: %w", err)
	}

	switch itemInMap.Item.Type {
	case "coin", "diamond":
		h.ItemsInMap.Delete(positionString)
		newScore := h.updateScore(itemAction.ID, itemInMap.Item.Value)
		h.broadcastCollectedItem(itemInMap)
		h.broadcastSingleScore(itemAction.ID, newScore)
	default:
		return fmt.Errorf("unknown item type: %s", itemInMap.Item.Type)
	}
	return nil
}

func (h *Hub) handlePositionUpdate(position *models.PlayerPosition) error {
	userId := position.ID

	newPosition := &models.Position{
		X: position.X,
		Y: position.Y,
	}

	currentPosition, ok := h.UsersInMap.Load(userId)
	if !ok {
		return fmt.Errorf("no current position found for user %s", userId)
	}
	// check move
	if reason, ok := IsValidMove(currentPosition.(*models.Position), newPosition); !ok {
		h.sendInvalidPositionToClient(reason, userId)
		return fmt.Errorf("invalid move from user %s", userId)
	}

	// check occupied
	newPositionString := fmt.Sprintf("%d-%d", newPosition.X, newPosition.Y)
	occupiedPosition, ok := h.OccupiedInMap.Load(newPositionString)
	if ok {
		errMsg := fmt.Sprintf("%v occupied position %v\n", newPositionString, occupiedPosition.(*models.Position))
		zap.S().Debug(errMsg)
		h.sendErrorToClient(userId, errMsg)
		// still need to send server position to sync front-end position
		h.sendInvalidPositionToClient(errMsg, userId)
		return fmt.Errorf(errMsg)
	}

	// remove previous position
	currentPositionString := fmt.Sprintf("%d-%d", currentPosition.(*models.Position).X, currentPosition.(*models.Position).Y)
	h.OccupiedInMap.Delete(currentPositionString)
	h.UsersInMap.Delete(userId)

	// save new position
	h.UsersInMap.Store(userId, newPosition)
	h.OccupiedInMap.Store(newPositionString, newPosition)

	// final
	h.broadcastValidPositionToAllClients(position)
	return nil
}

func (h *Hub) sendInvalidPositionToClient(reason string, userId string) {
	// get position
	currentPosition, err := h.GetPlayerPositionByUserId(userId)
	if err != nil {
		zap.S().Errorf("failed to get the current position for user %s, due to:%v", userId, err)
	}

	// set to invalid for front-end check
	currentPosition.Valid = false
	currentPosition.Reason = reason
	msg := &models.GameMsg{
		Type:    models.PlayerPositionType,
		Content: currentPosition,
	}

	// invalid position only sent to client itself
	h.ClientManager.SendToClient(userId, msg)
}

func (h *Hub) broadcastValidPositionToAllClients(position *models.PlayerPosition) {
	position.Valid = true
	msg := &models.GameMsg{
		Type:    models.PlayerPositionType,
		Content: position,
	}
	h.ClientManager.BroadcastAll(msg)
}
