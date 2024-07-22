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
	ID            string
	ClientManager *ClientManager
	HubManager    *HubManager
	StartPosition *models.StartPosition
	Occupied      sync.Map // for occupied check => map[positionString]objectId
	Obstacles     []*models.Position
	Positions     sync.Map // for player move validate => map[userId]*models.Position
	PositionChan  chan *models.PlayerPosition
	Scores        sync.Map
	ScoresChan    chan *models.PlayerScore
	MsgChan       chan *models.ChatMsg
	roundTimer    int
	roundDuration int
	mu            sync.RWMutex
	obstaclesMu   sync.RWMutex
}

func (h *Hub) InitObstacles() {
	numObstacles := 15
	for i := 0; i < numObstacles; i++ {
		x := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		y := rand.Intn(global.Dv.GetInt("GRIDSIZE") - 1)
		positionString := fmt.Sprintf("%d-%d", x, y)

		// check if occupied
		if _, occupied := h.Occupied.LoadOrStore(positionString, "obstacle"); !occupied {
			obstacle := &models.Position{X: x, Y: y}
			h.Occupied.Store(positionString, "obstacle")
			h.Obstacles = append(h.Obstacles, obstacle)
		} else {
			i-- // retry if occupied
		}
	}
}

func (h *Hub) UpdateObstacles(newObstacles []*models.Position) {
	h.obstaclesMu.Lock()
	defer h.obstaclesMu.Unlock()
	h.Obstacles = newObstacles
}

func (h *Hub) GetObstacles() []*models.Position {
	h.obstaclesMu.RLock()
	defer h.obstaclesMu.RUnlock()
	return h.Obstacles
}

func (h *Hub) SendObstaclesToClient(client *Client) {
	for _, obstacle := range h.Obstacles {
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
	h.Positions.Range(func(key, value interface{}) bool {
		userId, ok := key.(string)
		if !ok {
			zap.S().Errorf("SendAllPositionsToClient error, userId type: %T", userId)
			return false
		}

		// skip self
		if userId == client.ID {
			return true
		}

		position, ok := value.(models.Position)
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
		case position := <-h.PositionChan:
			h.handelPositionUpdate(position)
		}
	}
}

func (h *Hub) handelPositionUpdate(position *models.PlayerPosition) {
	userId := position.ID

	newPosition := models.Position{
		X: position.X,
		Y: position.Y,
	}

	currentPosition, ok := h.Positions.Load(userId)
	if !ok {
		zap.S().Errorf("no current position found for user %s", userId)
		return
	}
	// check move
	if !IsValidMove(currentPosition.(models.Position), newPosition) {
		zap.S().Infof("Invalid move from user %s", userId)
		h.sendInvalidPositionToClient(userId)
		return
	}

	// check occupied
	newPositionString := fmt.Sprintf("%d-%d", newPosition.X, newPosition.Y)
	occupiedUserId, ok := h.Occupied.Load(newPositionString)
	if ok {
		errMsg := fmt.Sprintf("%v occupied by %v\n", newPositionString, occupiedUserId.(string))
		zap.S().Debug(errMsg)
		h.sendErrorToClient(userId, errMsg)
		// still need to send server position to sync front-end position
		h.sendInvalidPositionToClient(userId)
		return
	}

	// remove previous position
	currentPositionString := fmt.Sprintf("%d-%d", currentPosition.(models.Position).X, currentPosition.(models.Position).Y)
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
	position, ok := h.Positions.Load(userId)
	if !ok {
		zap.S().Errorf("no position found for user %s", userId)
	}

	playPosition := &models.PlayerPosition{
		Valid:    false,
		ID:       userId,
		Position: models.Position{X: position.(models.Position).X, Y: position.(models.Position).Y},
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

		// try to store Occupied
		if _, occupied := h.Occupied.Load(positionString); !occupied {
			startPosition := &models.PlayerPosition{
				Valid: true,
				ID:    client.ID,
				Position: models.Position{
					X: x,
					Y: y,
				},
			}
			h.Positions.Store(client.ID, startPosition.Position)
			h.PositionChan <- startPosition

			zap.S().Infof("Start position set for client %s at (%d, %d) after %d attempts", client.ID, x, y, attempts+1)
			return
		}

		attempts++
	}

	zap.S().Errorf("failed to find start position for client %s after %d attempts", client.ID, maxAttempts)
}

func IsValidMove(currentPosition models.Position, newPosition models.Position) bool {
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
