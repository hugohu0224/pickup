package models

type Position struct {
	X int8 `json:"x"`
	Y int8 `json:"y"`
}

type PlayerPosition struct {
	ID       string `json:"id"`
	Position `json:"position"`
}

type PlayerScore struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

type ChatMsg struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type Action struct {
	ID     string `json:"id"`
	Effect string `json:"effect"`
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
