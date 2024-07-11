package models

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PlayerPosition struct {
	Position Position `json:"position"`
}

type ChatMsg struct {
	Content string `json:"content"`
}

type Action struct {
	MyAction string `json:"myAction"`
}

type GameMsgType string

const (
	PlayerPositionType GameMsgType = "playerPosition"
	PlayerActionType   GameMsgType = "playerAction"
	PlayerChatMsgType  GameMsgType = "playerMsg"
)

type GameMsg struct {
	Type    GameMsgType `json:"type"`
	Content interface{} `json:"content"`
}

type Score struct {
	score int
}
