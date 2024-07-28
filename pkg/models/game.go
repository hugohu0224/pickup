package models

/*
GameMsgType category of msg pipeline control
*/
type GameMsgType string

const (
	PlayerPositionType GameMsgType = "playerPosition"
	ItemActionType     GameMsgType = "itemAction"
	ItemCollectedType  GameMsgType = "itemCollected"
	PlayerChatMsgType  GameMsgType = "playerChatMsg"
	ErrorType          GameMsgType = "errorMsg"
	AlertType          GameMsgType = "alertMsg"
)

/*
Position category of move action control
*/
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

type StartPosition struct {
	Site      []map[string]int `json:"site"`
	UserCount int
}

/*
Item category of item collected control
*/
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

type Action struct {
	ID     string `json:"id"`
	Effect string `json:"effect"`
}

type ScoreUpdate struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
}

/*
ChatMsg not implement yet
*/
type ChatMsg struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

/*
Others
*/
type Alert struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Error struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type GameMsg struct {
	Type    GameMsgType `json:"type"`
	Content interface{} `json:"content"`
}
