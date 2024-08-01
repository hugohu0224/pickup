package game

import (
	"fmt"
	"github.com/pkg/errors"
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

func (h *Hub) RecoverStartPosition(client *Client) error {
	position, ok := h.UsersInMap.Load(client.ID)
	if !ok {
		return errors.New(fmt.Sprintf("error getting position for client %s", client.ID))
	}

	msg := &models.GameMsg{
		Type: models.PlayerPositionType,
		Content: &models.PlayerPosition{
			Valid:    true,
			ID:       client.ID,
			Position: position.(*models.Position)},
	}

	client.Send <- msg

	return nil
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

func IsValidMove(currentPosition, newPosition *models.Position) (string, bool) {
	if newPosition.X < 0 || newPosition.X >= global.Dv.GetInt("GRIDSIZE") ||
		newPosition.Y < 0 || newPosition.Y >= global.Dv.GetInt("GRIDSIZE") {
		return "The move is out of grid", false
	}

	if abs(newPosition.X-currentPosition.X)+abs(newPosition.Y-currentPosition.Y) > 1 {
		return "The move is over 1 step (you can move only 1 step)", false
	}
	return "", true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
