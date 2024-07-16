package game

import (
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
	for {
		select {
		case position := <-h.PositionChan:
			for client := range h.ClientManager.GetClients() {
				go func(client *Client, position *models.PlayerPosition) {
					// wrap msg
					msg := &models.GameMsg{
						Type:    models.PlayerPositionType,
						Content: position,
					}
					// send
					client.Send <- msg
				}(client, position)
			}
		}
	}
}

func (hm *HubManager) RunHubs() {
	hm.Mu.RLock()
	defer hm.Mu.RUnlock()
	for _, hub := range hm.Hubs {
		go func(h *Hub) {
			h.Run()
		}(hub)
	}
}

func (h *Hub) startRound() {
}

func (h *Hub) endRound() {
	h.mu.Lock()
	defer h.mu.Unlock()
}

func (h *Hub) StartTimer() {
}

func (h *Hub) settleRound() {
	h.mu.Lock()
	defer h.mu.Unlock()

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
	defer hm.Mu.Unlock()
}
