package game

import (
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
	"time"
)

type GameRound struct {
	Hub           *Hub
	IsWaiting     bool
	ActivePlayers map[string]*Client
	mu            sync.RWMutex
}

func (h *Hub) NewGameRound() *GameRound {
	return &GameRound{
		Hub:           h,
		IsWaiting:     true,
		ActivePlayers: make(map[string]*Client),
	}
}

func (h *Hub) ManageGameRounds() {
	h.initializeGameState()

	ticker := time.NewTicker(time.Second)
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

	if second < 30 {
		h.StartWaitingPeriod()
	} else {
		h.StartGameRound()
	}

	h.BroadcastCountdown()
}

func (h *Hub) updateGameState() {
	now := time.Now()
	second := now.Second()

	if second == 0 {
		h.StartWaitingPeriod()
	} else if second == 30 {
		h.StartGameRound()
	} else if second == 59 {
		h.EndGameRound()
	}

	h.BroadcastCountdown()
}

func (h *Hub) StartWaitingPeriod() {
	h.CurrentRound.mu.Lock()
	defer h.CurrentRound.mu.Unlock()

	if !h.CurrentRound.IsWaiting {
		h.CurrentRound.IsWaiting = true
		zap.S().Infof("hub: %v round is waiting", h.ID)
		h.BroadcastRoundState("waiting")
	}
}

func (h *Hub) StartGameRound() {
	h.CurrentRound.mu.Lock()
	defer h.CurrentRound.mu.Unlock()

	if h.CurrentRound.IsWaiting {
		h.CurrentRound.IsWaiting = false
		zap.S().Infof("hub: %v round is starting", h.ID)

		go func() {
			h.InitializeRound()
			h.BroadcastRoundState("playing")
			zap.S().Infof("hub: %v round initialization completed", h.ID)
		}()
	}
}

func (h *Hub) EndGameRound() {
	h.CurrentRound.mu.Lock()
	defer h.CurrentRound.mu.Unlock()

	h.ClearPreviousRoundData()

	for _, client := range h.CurrentRound.ActivePlayers {
		client.IsActive = false
	}
	h.CurrentRound.ActivePlayers = make(map[string]*Client)

	h.BroadcastRoundState("ended")
}

// InitializeRound 初始化回合
func (h *Hub) InitializeRound() {
	h.ClearPreviousRoundData()
	h.InitAllItems()

	for _, client := range h.CurrentRound.ActivePlayers {
		h.SendAllGameRoundStateToClient(client)
	}
}

func (h *Hub) ClearPreviousRoundData() {
	h.OccupiedInMap = sync.Map{}
	h.ObstaclesInMap = make([]*models.Position, 0)
	h.ItemsInMap = sync.Map{}
	h.UsersInMap = sync.Map{}
	h.Scores = sync.Map{}
}

func (h *Hub) BroadcastRoundState(state string) {
	now := time.Now()
	var endTime time.Time
	if state == "waiting" {
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 30, 0, now.Location())
	} else {
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

func (h *Hub) BroadcastCountdown() {
	now := time.Now()
	var target time.Time
	if h.CurrentRound.IsWaiting {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 30, 0, now.Location())
	} else {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	}

	remaining := target.Sub(now)
	if remaining <= 10*time.Second && remaining > 0 {

		currentState := "waiting"
		if !h.CurrentRound.IsWaiting {
			currentState = "playing"
		}

		msg := &models.GameMsg{
			Type: "countdown",
			Content: map[string]interface{}{
				"remainingTime": int(remaining.Seconds()),
				"currentState":  currentState,
			},
		}
		h.ClientManager.BroadcastAll(msg)
	}
}

func (h *Hub) RegisterClient(client *Client) {
	h.ClientManager.RegisterClient(client)

	if !h.CurrentRound.IsWaiting {
		client.IsActive = false
		h.sendWaitingNotificationToClient(client)
	} else {
		h.CurrentRound.mu.Lock()
		h.CurrentRound.ActivePlayers[client.ID] = client
		h.CurrentRound.mu.Unlock()
		client.IsActive = true
	}
}

func (h *Hub) sendWaitingNotificationToClient(client *Client) {
	msg := &models.GameMsg{
		Type: "waitingNotification",
		Content: map[string]interface{}{
			"message":        "The game has already started. You will join in the next round.",
			"nextRoundStart": getNextRoundStartTime(),
		},
	}
	client.Send <- msg
}

func getNextRoundStartTime() time.Time {
	now := time.Now()
	if now.Second() < 30 {
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 30, 0, now.Location())
	}
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 30, 0, now.Location())
}

func getTimeToNextState() time.Duration {
	now := time.Now()
	var target time.Time
	if now.Second() < 30 {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 30, 0, now.Location())
	} else {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	}
	return target.Sub(now)
}
