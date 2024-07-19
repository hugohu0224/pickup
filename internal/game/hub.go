package game

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"pickup/internal/global"
	"pickup/pkg/models"
	"sync"
)

var Hm *HubManager

type Hub struct {
	ID            string
	ClientManager *ClientManager
	HubManager    *HubManager
	StartPosition *models.StartPosition
	Occupied      sync.Map // for occupied check => map[positionString]userId
	Positions     sync.Map // for player move validate => map[userId]position
	PositionChan  chan *models.PlayerPosition
	Scores        sync.Map
	ScoresChan    chan *models.PlayerScore
	MsgChan       chan *models.ChatMsg
	roundTimer    int
	roundDuration int
	mu            sync.RWMutex
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
		case position := <-h.PositionChan:
			h.handelPositionUpdate(position)
		}
	}
}

func (h *Hub) handelPositionUpdate(position *models.PlayerPosition) {
	// init
	userId := position.ID
	newPosition := &models.Position{
		X: position.X,
		Y: position.Y,
	}
	currentPosition, ok := h.Positions.Load(userId)
	if !ok {
		zap.S().Errorf("no current position found for user %s", userId)
		return
	}
	// check move
	if !IsValidMove(currentPosition.(*models.Position), newPosition) {
		zap.S().Infof("%v occupied by %v", newPosition, h.ID)
		h.sendInvalidPositionToClient(userId)
		return
	}

	// check occupied
	newPositionString := fmt.Sprintf("%d-%d", newPosition.X, newPosition.Y)
	_, ok = h.Occupied.Load(newPositionString)
	if ok {
		errMsg := fmt.Sprintf("\"%v occupied by %v\", newPositionString, h.ID")
		zap.S().Errorf(errMsg)
		h.sendErrorToClient(userId, errMsg)
		// still need to send server position to sync front-end position
		h.sendInvalidPositionToClient(userId)
		return
	}

	// remove previous position
	currentPositionString := fmt.Sprintf("%d-%d", currentPosition.(*models.Position).X, currentPosition.(*models.Position).Y)
	h.Occupied.Delete(currentPositionString)
	h.Positions.Delete(userId)

	// save new position
	h.Positions.Store(userId, newPosition)
	h.Occupied.Store(newPositionString, userId)

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

	// send
	msg := &models.GameMsg{
		Type:    models.PlayerPositionType,
		Content: currentPosition,
	}

	// invalid position no broadcast
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
	position, ok := h.Positions.Load(userId)
	if !ok {
		zap.S().Errorf("no position found for user %s", userId)
	}

	playPosition := &models.PlayerPosition{
		Valid:    false,
		ID:       userId,
		Position: models.Position{X: position.(*models.Position).X, Y: position.(*models.Position).Y},
	}

	return playPosition
}

func (h *Hub) GetStartPosition() (int, int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// check if full
	if h.StartPosition.UserCount >= 3 {
		errMsg := "fail to get start position, due to full user in hub"
		zap.S().Error(errMsg)
		return -1, -1, errors.New(errMsg)
	}

	// get a position
	position := h.StartPosition.Site[h.StartPosition.UserCount]
	x := position["x"]
	y := position["y"]
	h.StartPosition.UserCount++

	return x, y, nil
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
	h.Occupied.Delete(fmt.Sprintf("%d-%d", position.X, position.Y))
	h.Positions.Delete(userId)
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
		ID:            id,
		ClientManager: NewClientManager(),
		HubManager:    hm,
		StartPosition: &models.StartPosition{
			Site: []map[string]int{
				{"x": 0, "y": 0},
				{"x": 14, "y": 0},
				{"x": 0, "y": 14},
				{"x": 14, "y": 14},
			},
			UserCount: 0,
		},
		Occupied:      sync.Map{},
		Positions:     sync.Map{},
		PositionChan:  make(chan *models.PlayerPosition),
		Scores:        sync.Map{},
		ScoresChan:    make(chan *models.PlayerScore),
		MsgChan:       make(chan *models.ChatMsg),
		roundTimer:    0,
		roundDuration: 0,
		mu:            sync.RWMutex{},
	}
}

func (hm *HubManager) RegisterHub(h *Hub) {
	hm.Mu.Lock()
	hm.Hubs[h.ID] = h
	hm.Mu.Unlock()
	go h.Run()
}
