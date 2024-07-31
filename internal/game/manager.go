package game

import (
	"fmt"
	"go.uber.org/zap"
	"pickup/pkg/models"
	"sync"
	"time"
)

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
	hub := &Hub{
		ID:             id,
		ClientManager:  NewClientManager(),
		HubManager:     hm,
		OccupiedInMap:  sync.Map{},
		ObstaclesInMap: nil,
		ItemsInMap:     sync.Map{},
		UsersInMap:     sync.Map{},
		PositionChan:   make(chan *models.PlayerPosition),
		Scores:         sync.Map{},
		ActionChan:     make(chan *models.ItemAction),
		MsgChan:        make(chan *models.ChatMsg),
		CurrentRound:   nil,
		mu:             sync.RWMutex{},
		obstaclesMu:    sync.RWMutex{},
	}

	hub.CurrentRound = hub.NewRound()

	return hub
}

func (hm *HubManager) RegisterHub(h *Hub) {
	hm.Mu.Lock()
	hm.Hubs[h.ID] = h
	hm.Mu.Unlock()
	go h.Run()
}

type ClientManager struct {
	clients          map[*Client]bool
	clientsById      map[string]*Client
	clientsConnState map[string]bool
	mu               sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:          make(map[*Client]bool),
		clientsById:      make(map[string]*Client),
		clientsConnState: make(map[string]bool),
		mu:               sync.RWMutex{},
	}
}

func (cm *ClientManager) RegisterClient(client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[client] = true
	cm.clientsById[client.ID] = client
	cm.clientsConnState[client.ID] = true
}

func (cm *ClientManager) UpdateClientConnStateById(userId string, bool bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clientsConnState[userId] = bool
	zap.S().Debug(fmt.Sprintf("UpdateClientConnState client:%v", userId), zap.Bool("bool", bool))
}

func (cm *ClientManager) RemoveClient(client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.clients, client)
	delete(cm.clientsConnState, client.ID)
	delete(cm.clientsById, client.ID)
	client.Conn.Close()

	zap.S().Debugf("Client %s removed", client.ID)
}

func (cm *ClientManager) GetClients() map[*Client]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.clients
}

func (cm *ClientManager) GetDisconnectedClients() []*Client {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	disconnected := make([]*Client, 0)
	for userId, isOnline := range cm.clientsConnState {
		if !isOnline {
			disconnected = append(disconnected, cm.clientsById[userId])
		}
	}
	return disconnected
}

func (cm *ClientManager) GetClientByID(userId string) (*Client, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	client, exists := cm.clientsById[userId]
	return client, exists
}

func (cm *ClientManager) BroadcastAll(msg *models.GameMsg) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for client := range cm.clients {
		select {
		case client.Send <- msg:
		default:
			close(client.Send)
			delete(cm.clients, client)
		}
	}
}

func (cm *ClientManager) SendToClient(userId string, msg *models.GameMsg) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	client, exists := cm.GetClientByID(userId)
	if !exists {
		zap.S().Errorf("client %s not found", userId)
		return
	}

	select {
	case client.Send <- msg:
	case <-time.After(2 * time.Second):
		zap.S().Warnf("Timeout sending message to client %s", userId)
	}
}
