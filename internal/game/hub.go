package game

import (
	"errors"
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
	StartPosition  *models.StartPosition
	OccupiedInMap  sync.Map // for occupied check => map[positionString]*models.Position
	ObstaclesInMap []*models.Position
	ItemsInMap     sync.Map // map[positionString]*models.ItemAction
	PositionsInMap sync.Map // for player move validate => map[userId]*models.Position
	PositionChan   chan *models.PlayerPosition
	Scores         sync.Map
	ActionChan     chan *models.ItemAction
	MsgChan        chan *models.ChatMsg
	roundTimer     int
	roundDuration  int
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
	h.InitCoins()
}

func (h *Hub) InitCoins() {
	numCoins := global.Dv.GetInt("COINNUMBER")
	for i := 0; i < numCoins; i++ {
		x := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		y := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		positionString := fmt.Sprintf("%d-%d", x, y)

		// check if not occupied
		if _, occupied := h.OccupiedInMap.Load(positionString); !occupied {
			itemAction := &models.ItemAction{
				ID:       "",
				Valid:    true,
				Item:     &models.Item{Type: "coin", Value: 10},
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
	h.PositionsInMap.Range(func(key, value interface{}) bool {
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
			h.handelPositionUpdate(playerPosition)
		case itemAction := <-h.ActionChan:
			h.handelItemAction(itemAction)
		}
	}
}

func (h *Hub) getItemInMap(positionString string) (*models.ItemAction, error) {
	item, ok := h.ItemsInMap.Load(positionString)
	if !ok {
		return nil, errors.New(fmt.Sprintf("faild to load item: %v, %v", positionString, h.ID))
	}
	return item.(*models.ItemAction), nil
}

func (h *Hub) handelItemAction(itemAction *models.ItemAction) {
	positionString := fmt.Sprintf("%d-%d", itemAction.Position.X, itemAction.Position.Y)
	itemInMap, err := h.getItemInMap(positionString)
	if err != nil {
		zap.S().Errorf("failed to get itemImMap: %v", positionString)
	}
	switch itemInMap.Item.Type {
	case "coin":
		// remove item after activate
		h.ItemsInMap.Delete(positionString)
		h.broadcastCollectedCoin(itemAction)
	}
}

func (h *Hub) broadcastCollectedCoin(itemAction *models.ItemAction) {
	itemAction.Valid = true
	msg := &models.GameMsg{
		Type:    models.ItemCollectedType,
		Content: itemAction,
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) handelPositionUpdate(position *models.PlayerPosition) {
	userId := position.ID

	newPosition := &models.Position{
		X: position.X,
		Y: position.Y,
	}

	currentPosition, ok := h.PositionsInMap.Load(userId)
	if !ok {
		zap.S().Errorf("no current position found for user %s", userId)
		return
	}
	// check move
	if !IsValidMove(currentPosition.(*models.Position), newPosition) {
		zap.S().Infof("Invalid move from user %s", userId)
		h.sendInvalidPositionToClient(userId)
		return
	}

	// check occupied
	newPositionString := fmt.Sprintf("%d-%d", newPosition.X, newPosition.Y)
	occupiedPosition, ok := h.OccupiedInMap.Load(newPositionString)
	if ok {
		errMsg := fmt.Sprintf("%v occupied position %v\n", newPositionString, occupiedPosition.(*models.Position))
		zap.S().Debug(errMsg)
		h.sendErrorToClient(userId, errMsg)
		// still need to send server position to sync front-end position
		h.sendInvalidPositionToClient(userId)
		return
	}

	// remove previous position
	currentPositionString := fmt.Sprintf("%d-%d", currentPosition.(*models.Position).X, currentPosition.(*models.Position).Y)
	h.OccupiedInMap.Delete(currentPositionString)
	h.PositionsInMap.Delete(userId)

	// save new position
	h.PositionsInMap.Store(userId, newPosition)
	h.OccupiedInMap.Store(newPositionString, newPosition)

	// final
	h.broadcastValidPositionToAllClients(position)
}

func (h *Hub) broadcastValidPositionToAllClients(position *models.PlayerPosition) {
	position.Valid = true
	msg := &models.GameMsg{
		Type:    models.PlayerPositionType,
		Content: position,
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) sendInvalidPositionToClient(userId string) {
	// get position
	currentPosition := h.GetPositionByUserId(userId)

	// set to invalid for front-end check
	currentPosition.Valid = false
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

func (h *Hub) GetPositionByUserId(userId string) *models.PlayerPosition {
	position, ok := h.PositionsInMap.Load(userId)
	if !ok {
		zap.S().Errorf("no position found for user %s", userId)
	}

	playPosition := &models.PlayerPosition{
		Valid:    false,
		ID:       userId,
		Position: &models.Position{X: position.(*models.Position).X, Y: position.(*models.Position).Y},
	}

	return playPosition
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
			h.PositionsInMap.Store(client.ID, startPosition.Position)
			h.PositionChan <- startPosition
			zap.S().Infof("start position set for client %s at (%d, %d) after %d attempts", client.ID, x, y, attempts+1)
			return
		}
		attempts++
	}
	zap.S().Errorf("failed to find start position for client %s after %d attempts", client.ID, maxAttempts)
}

func IsValidMove(currentPosition *models.Position, newPosition *models.Position) bool {
	// check grid
	if newPosition.X < 0 || newPosition.X >= global.Dv.GetInt("GRIDSIZE") ||
		newPosition.Y < 0 || newPosition.Y >= global.Dv.GetInt("GRIDSIZE") {
		return false
	}

	// check if move only 1 step
	xDiff := abs(newPosition.X - currentPosition.X)
	yDiff := abs(newPosition.Y - currentPosition.Y)
	if (xDiff + yDiff) > 1 {
		return false
	}
	return true
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

	position := h.GetPositionByUserId(userId)
	h.OccupiedInMap.Delete(fmt.Sprintf("%d-%d", position.X, position.Y))
	h.PositionsInMap.Delete(userId)
	h.Scores.Delete(userId)
	h.ClientManager.RemoveClient(client)
	client.Conn.Close()

	// update start position count
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.StartPosition.UserCount > 0 {
		h.StartPosition.UserCount--
	}
	zap.S().Infof("cleaned up data for user %s in Hub %s", userId, h.ID)
}

type HubManager struct {
	Hubs map[string]*Hub
	Mu   sync.RWMutex
}

func (hm *HubManager) GetHubById(id string) *Hub {
	hm.Mu.RLock()
	defer hm.Mu.RUnlock()
	if hub, ok := hm.Hubs[id]; ok {
		return hub
	}
	return nil
}

func NewHub(hm *HubManager, id string) *Hub {
	return &Hub{
		ID:             id,
		ClientManager:  NewClientManager(),
		HubManager:     hm,
		OccupiedInMap:  sync.Map{},
		PositionsInMap: sync.Map{},
		PositionChan:   make(chan *models.PlayerPosition),
		Scores:         sync.Map{},
		ActionChan:     make(chan *models.ItemAction),
		MsgChan:        make(chan *models.ChatMsg),
		roundTimer:     0,
		roundDuration:  0,
		mu:             sync.RWMutex{},
	}
}

func (hm *HubManager) RegisterHub(h *Hub) {
	hm.Mu.Lock()
	hm.Hubs[h.ID] = h
	hm.Mu.Unlock()
	go h.Run()
}
