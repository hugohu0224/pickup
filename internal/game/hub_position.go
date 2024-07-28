package game

import (
	"fmt"
	"go.uber.org/zap"
	"math/rand"
	"pickup/internal/global"
	"pickup/pkg/models"
)

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
			h.UsersInMap.Store(client.ID, startPosition.Position)
			h.PositionChan <- startPosition
			zap.S().Infof("start position set for client %s at (%d, %d) after %d attempts", client.ID, x, y, attempts+1)
			return
		}
		attempts++
	}
	zap.S().Errorf("failed to find start position for client %s after %d attempts", client.ID, maxAttempts)
}

func (h *Hub) GetPlayerPositionByUserId(userId string) (*models.PlayerPosition, error) {
	position, ok := h.UsersInMap.Load(userId)
	if !ok {
		return nil, fmt.Errorf("no position found for user %s", userId)
	}

	playPosition := &models.PlayerPosition{
		Valid:    false,
		ID:       userId,
		Position: &models.Position{X: position.(*models.Position).X, Y: position.(*models.Position).Y},
	}

	return playPosition, nil
}

func IsValidMove(currentPosition *models.Position, newPosition *models.Position) (reason string, ok bool) {
	// check grid
	if newPosition.X < 0 || newPosition.X >= global.Dv.GetInt("GRIDSIZE") ||
		newPosition.Y < 0 || newPosition.Y >= global.Dv.GetInt("GRIDSIZE") {
		return "The move is out of grid", false
	}

	// check if move only 1 step
	xDiff := abs(newPosition.X - currentPosition.X)
	yDiff := abs(newPosition.Y - currentPosition.Y)
	if (xDiff + yDiff) > 1 {
		return "The move is over 1 temp (you can move only 1 step)", false
	}
	return "", true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
