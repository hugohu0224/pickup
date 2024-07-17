package game

import (
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
)

type Hub struct {
	ID            string
	ClientManager *ClientManager
	Positions     sync.Map
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
			msg := &models.GameMsg{
				Type:    models.PlayerPositionType,
				Content: position,
			}
			h.ClientManager.Broadcast(msg)
		}
	}
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

func NewHub(id string) *Hub {
	return &Hub{
		ID:            id,
		ClientManager: NewClientManager(),
		PositionChan:  make(chan *models.PlayerPosition),
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
