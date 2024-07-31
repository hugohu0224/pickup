package game

import (
	"fmt"
	"math/rand"
	"pickup/internal/global"
	"pickup/pkg/models"
)

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

func (h *Hub) GetObstacles() []*models.Position {
	h.obstaclesMu.RLock()
	defer h.obstaclesMu.RUnlock()
	return h.ObstaclesInMap
}

func (h *Hub) UpdateObstacles(newObstacles []*models.Position) {
	h.obstaclesMu.Lock()
	defer h.obstaclesMu.Unlock()
	h.ObstaclesInMap = newObstacles
}

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

func (h *Hub) InitAllItems() {
	h.InitObstacles()
	h.InitActionItems("COINNUMBER", "coin", 10)
	h.InitActionItems("DIAMOND", "diamond", 100)
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

func (h *Hub) updateScore(userID string, value int) int {
	currentScore, _ := h.Scores.LoadOrStore(userID, 0)
	newScore := currentScore.(int) + value
	h.Scores.Store(userID, newScore)
	return newScore
}
