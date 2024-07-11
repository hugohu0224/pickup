package game

import (
	"pickup/pkg/models"
	"sync"
	"time"
)

var SingleHubManager *HubManager

type Hub struct {
	ID            string
	ClientManager *ClientManager
	Positions     map[string][]models.Position
	PositionChan  chan models.Position
	MsgChan       chan models.ChatMsg
	Scores        map[string][]models.Score
	ScoresChan    chan map[string][]models.Score
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

}

func (h *Hub) startRound() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// reset
	h.roundTimer = h.roundDuration
}

func (h *Hub) endRound() {
	h.mu.Lock()
	defer h.mu.Unlock()
}

func (h *Hub) StartTimer() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.roundTimer--

			if h.roundTimer < 0 {
				h.endRound()
				time.Sleep(5 * time.Second)
				h.startRound()
			}

			//h.broadcast <- message
		}
	}
}

func (h *Hub) settleRound() {
	h.mu.Lock()
	defer h.mu.Unlock()

}

type HubManager struct {
	Hubs map[string]*Hub
	Mu   sync.RWMutex
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

func NewHub(id string) *Hub {
	return &Hub{
		ID:            "",
		ClientManager: NewClientManager(),
		Positions:     make(map[string][]models.Position),
		Scores:        make(map[string][]models.Score),
		roundTimer:    0,
		roundDuration: 0,
		mu:            sync.RWMutex{},
	}
}

func (hm *HubManager) RegisterHub(h *Hub) {
	hm.Mu.Lock()
	defer hm.Mu.Unlock()
}
