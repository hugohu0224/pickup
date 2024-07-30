package game

import (
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
	"time"
)

type GameRound struct {
	Hub   *Hub
	State string // "waiting", "cleanup", "preparing", "playing", "ended"
	Mu    sync.RWMutex
}

func (h *Hub) NewGameRound() *GameRound {
	return &GameRound{
		Hub:   h,
		State: "waiting",
	}
}

func (h *Hub) ManageGameRounds() {
	h.initializeGameState()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.updateGameState()
		}
	}
}

func (h *Hub) initializeGameState() {
	now := time.Now()
	second := now.Second()

	switch {
	case second < 5:
		h.StartWaitingPeriod()
	case second < 10:
		h.CleanUpPeriod()
	case second < 15:
		h.StartPreparePeriod()
	case second < 59:
		h.StartGameRound()
	default:
		h.EndGameRound()
	}

	h.BroadcastCountdown()
}

func (h *Hub) updateGameState() {
	now := time.Now()
	second := now.Second()

	switch {
	case second == 0 && h.CurrentRound.State != "waiting":
		h.StartWaitingPeriod()
	case second == 5 && h.CurrentRound.State != "cleanup":
		h.CleanUpPeriod()
	case second == 10 && h.CurrentRound.State != "preparing":
		h.StartPreparePeriod()
	case second == 15 && h.CurrentRound.State != "playing":
		h.StartGameRound()
	case second == 59 && h.CurrentRound.State != "ended":
		h.EndGameRound()
	}

	h.BroadcastCountdown()
}

func (h *Hub) StartWaitingPeriod() {
	h.CurrentRound.Mu.Lock()
	defer h.CurrentRound.Mu.Unlock()

	if h.CurrentRound.State != "waiting" {
		h.CurrentRound.State = "waiting"
		zap.S().Infof("hub: %v round is waiting", h.ID)
		h.BroadcastRoundState("waiting")
	}
}

func (h *Hub) CleanUpPeriod() {
	h.CurrentRound.Mu.Lock()
	defer h.CurrentRound.Mu.Unlock()
	if h.CurrentRound.State != "cleanup" {
		h.CurrentRound.State = "cleanup"
		h.ClearPreviousRoundData()
		h.BroadcastRoundState("cleanup")
		zap.S().Infof("hub: %v round cleanup finished", h.ID)
	}
}

func (h *Hub) StartPreparePeriod() {
	h.CurrentRound.Mu.Lock()
	defer h.CurrentRound.Mu.Unlock()

	if h.CurrentRound.State != "preparing" {
		h.CurrentRound.State = "preparing"
		zap.S().Infof("hub: %v round preparing", h.ID)
		h.InitializeRoundState()
		h.BroadcastRoundState("preparing")
	}
}

func (h *Hub) InitializeRoundState() {
	zap.S().Infof("hub: %v initializing round started", h.ID)

	h.ClearPreviousRoundData()
	h.InitAllItems()

	// clear disconnect client
	for _, client := range h.ClientManager.GetDisconnectedClients() {
		zap.S().Debugf("get disconnected client %v, start to remove", client.ID)
		h.CleanupClient(client)
	}

	// reset position
	for client, _ := range h.ClientManager.GetClients() {
		client.Hub.InitStartPosition(client)
		client.GameIsActive = true
	}

	// send new game state to clients
	for client, _ := range h.ClientManager.GetClients() {
		h.SendAllGameRoundStateToClient(client)
	}

	zap.S().Infof("hub: %v initializing round completed", h.ID)
}

func (h *Hub) StartGameRound() {
	h.CurrentRound.Mu.Lock()
	defer h.CurrentRound.Mu.Unlock()

	if h.CurrentRound.State != "playing" {
		h.CurrentRound.State = "playing"
		zap.S().Infof("hub: %v round is starting", h.ID)
		h.BroadcastRoundState("playing")
	}
}

func (h *Hub) EndGameRound() {
	h.CurrentRound.Mu.Lock()
	defer h.CurrentRound.Mu.Unlock()

	if h.CurrentRound.State != "ended" {
		h.CurrentRound.State = "ended"
		for client, _ := range h.ClientManager.GetClients() {
			client.GameIsActive = false
		}
		h.BroadcastRoundState("ended")
	}
}

func (h *Hub) BroadcastRoundState(state string) {
	now := time.Now()
	var endTime time.Time

	switch state {
	case "waiting":
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 5, 0, now.Location())
	case "cleanup":
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 10, 0, now.Location())
	case "preparing":
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 15, 0, now.Location())
	case "playing":
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 59, 0, now.Location())
	case "ended":
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	}

	msg := &models.GameMsg{
		Type: "roundState",
		Content: map[string]interface{}{
			"state":       state,
			"currentTime": now,
			"endTime":     endTime,
		},
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) ClearPreviousRoundData() {
	h.OccupiedInMap = sync.Map{}
	h.ObstaclesInMap = make([]*models.Position, 0)
	h.ItemsInMap = sync.Map{}
	h.UsersInMap = sync.Map{}
	h.Scores = sync.Map{}
}

func (h *Hub) BroadcastCountdown() {
	now := time.Now()
	second := now.Second()
	var target time.Time
	var currentState string

	switch {
	case second < 5:
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 5, 0, now.Location())
		currentState = "waiting"
	case second < 10:
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 10, 0, now.Location())
		currentState = "cleanup"
	case second < 15:
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 15, 0, now.Location())
		currentState = "preparing"
	case second < 59:
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 59, 0, now.Location())
		currentState = "playing"
	default:
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
		currentState = "ended"
	}

	remaining := target.Sub(now)
	stateRemainingTime := int(remaining.Seconds()) % 60

	msg := &models.GameMsg{
		Type: "countdown",
		Content: map[string]interface{}{
			"remainingTime": stateRemainingTime,
			"currentState":  currentState,
		},
	}
	h.ClientManager.BroadcastAll(msg)
}

func (h *Hub) GetNextRoundStartTime() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
}
