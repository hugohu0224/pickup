package models

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type PlayerPosition struct {
	Valid     bool   `json:"valid"`
	ID        string `json:"id"`
	Reason    string `json:"reason,omitempty"`
	*Position `json:"position"`
}

type Item struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type ItemAction struct {
	Valid     bool   `json:"valid"`
	ID        string `json:"id"`
	*Item     `json:"item"`
	*Position `json:"position"`
}

type ScoreUpdate struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

type ItemType string

type ChatMsg struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type Action struct {
	ID     string `json:"id"`
	Effect string `json:"effect"`
}

type Alert struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Error struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type StartPosition struct {
	Site      []map[string]int `json:"site"`
	UserCount int
}

type GameMsgType string

const (
	PlayerPositionType GameMsgType = "playerPosition"
	ItemActionType     GameMsgType = "itemAction"
	ItemCollectedType  GameMsgType = "itemCollected"
	PlayerChatMsgType  GameMsgType = "playerChatMsg"
	ErrorType          GameMsgType = "errorMsg"
	AlertType          GameMsgType = "alertMsg"
)

type GameMsg struct {
	Type    GameMsgType `json:"type"`
	Content interface{} `json:"content"`
}
