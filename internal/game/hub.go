package game

import (
	"fmt"
	"go.uber.org/zap"
	"math/rand"
	"pickup/internal/global"
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
	PositionChan   chan *models.PlayerPosition
	Scores         sync.Map // map[userIdString]int (player score storage)
	ActionChan     chan *models.ItemAction
	MsgChan        chan *models.ChatMsg
	mu             sync.RWMutex
	obstaclesMu    sync.RWMutex
}

func (h *Hub) InitObstacles() {
	numObstacles := global.Dv.GetInt("OBSNUMBER")
	for i := 0; i < numObstacles; i++ {
		x := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		y := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		positionString := fmt.Sprintf("%d-%d", x, y)

		// check if occupied
		if _, occupied := h.OccupiedInMap.LoadOrStore(positionString, "obstacle"); !occupied {
			obstacle := &models.Position{X: x, Y: y}
			h.OccupiedInMap.Store(positionString, obstacle)
			h.ObstaclesInMap = append(h.ObstaclesInMap, obstacle)
		} else {
			i-- // retry if occupied
		}
	}
}

func (h *Hub) InitAllItems() {
	h.InitObstacles()
	h.InitActionItems("COINNUMBER", "coin", 10)
	h.InitActionItems("DIAMOND", "diamond", 100)
}

func (h *Hub) InitStar() {}

func (h *Hub) InitActionItems(itemName string, itemType string, itemValue int) {
	numCoins := global.Dv.GetInt(itemName)
	for i := 0; i < numCoins; i++ {
		x := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		y := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		positionString := fmt.Sprintf("%d-%d", x, y)

		// check if not occupied
		if _, occupied := h.OccupiedInMap.Load(positionString); !occupied {
			itemAction := &models.ItemAction{
				ID:       "",
				Valid:    true,
				Item:     &models.Item{Type: itemType, Value: itemValue},
				Position: &models.Position{X: x, Y: y},
			}
			h.ItemsInMap.Store(positionString, itemAction)
		} else {
			i-- // retry if occupied
		}
	}
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

func (h *Hub) UpdateObstacles(newObstacles []*models.Position) {
	h.obstaclesMu.Lock()
	defer h.obstaclesMu.Unlock()
	h.ObstaclesInMap = newObstacles
}

func (h *Hub) GetObstacles() []*models.Position {
	h.obstaclesMu.RLock()
	defer h.obstaclesMu.RUnlock()
	return h.ObstaclesInMap
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

func (h *Hub) SendAllPositionToClient(client *Client) {
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

func (h *Hub) Run() {
	zap.S().Infof("Hub %s is running", h.ID)
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

func (h *Hub) getItemInMap(positionString string) (*models.ItemAction, error) {
	item, ok := h.ItemsInMap.Load(positionString)
	if !ok {
		return nil, fmt.Errorf("failed to load item at position %s in hub %s", positionString, h.ID)
	}
	itemAction, ok := item.(*models.ItemAction)
	if !ok {
		return nil, fmt.Errorf("invalid item type at position %s in hub %s", positionString, h.ID)
	}
	return itemAction, nil
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

func (h *Hub) updateScore(userID string, value int) int {
	currentScore, _ := h.Scores.LoadOrStore(userID, 0)
	newScore := currentScore.(int) + value
	h.Scores.Store(userID, newScore)
	return newScore
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

func (h *Hub) broadcastCollectedItem(itemAction *models.ItemAction) {
	itemAction.Valid = true
	msg := &models.GameMsg{
		Type:    models.ItemCollectedType,
		Content: itemAction,
	}
	h.ClientManager.BroadcastAll(msg)
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

func (h *Hub) broadcastValidPositionToAllClients(position *models.PlayerPosition) {
	position.Valid = true
	msg := &models.GameMsg{
		Type:    models.PlayerPositionType,
		Content: position,
	}
	h.ClientManager.BroadcastAll(msg)
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

func (h *Hub) GetPlayerPositionByUserId(userId string) (*models.PlayerPosition, error) {
	position, ok := h.UsersInMap.Load(userId)
	if !ok {
		return nil, fmt.Errorf("no position found for user %s", userId)
	}

	playPosition := &models.PlayerPosition{
		Valid:    false,
		ID:       userId,
		Position: &models.Position{X: position.(*models.Position).X, Y: position.(*models.Position).Y},
	}

	return playPosition, nil
}

func (h *Hub) InitStartPosition(client *Client) {
	tryScope := global.Dv.GetInt("GRIDSIZE") - 1
	maxAttempts := tryScope * tryScope
	attempts := 0

	zap.S().Infof("Initializing start position for client %s", client.ID)

	for attempts < maxAttempts {
		x := rand.Intn(tryScope)
		y := rand.Intn(tryScope)
		positionString := fmt.Sprintf("%d-%d", x, y)

		// try to store OccupiedInMap
		if _, occupied := h.OccupiedInMap.Load(positionString); !occupied {
			startPosition := &models.PlayerPosition{
				Valid: true,
				ID:    client.ID,
				Position: &models.Position{
					X: x,
					Y: y,
				},
			}
			h.UsersInMap.Store(client.ID, startPosition.Position)
			h.PositionChan <- startPosition
			zap.S().Infof("start position set for client %s at (%d, %d) after %d attempts", client.ID, x, y, attempts+1)
			return
		}
		attempts++
	}
	zap.S().Errorf("failed to find start position for client %s after %d attempts", client.ID, maxAttempts)
}

func IsValidMove(currentPosition *models.Position, newPosition *models.Position) (reason string, ok bool) {
	// check grid
	if newPosition.X < 0 || newPosition.X >= global.Dv.GetInt("GRIDSIZE") ||
		newPosition.Y < 0 || newPosition.Y >= global.Dv.GetInt("GRIDSIZE") {
		return "The move is out of grid", false
	}

	// check if move only 1 step
	xDiff := abs(newPosition.X - currentPosition.X)
	yDiff := abs(newPosition.Y - currentPosition.Y)
	if (xDiff + yDiff) > 1 {
		return "The move is over 1 temp (you can move only 1 step)", false
	}
	return "", true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// CleanupClient removes all data associated with a client from the hub
func (h *Hub) CleanupClient(client *Client) {
	userId := client.ID

	position, err := h.GetPlayerPositionByUserId(userId)
	if err != nil {
		zap.S().Errorf("failed to get the current position for user %s, due to:%v", userId, err.Error())
	}

	h.OccupiedInMap.Delete(fmt.Sprintf("%d-%d", position.X, position.Y))
	h.UsersInMap.Delete(userId)
	h.Scores.Delete(userId)
	h.ClientManager.RemoveClient(client)
	client.Conn.Close()
	zap.S().Infof("cleaned up data for user %s in Hub %s", userId, h.ID)
}
