package models

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type PlayerPosition struct {
	Valid     bool   `json:"valid"`
	ID        string `json:"id"`
	*Position `json:"position"`
}

type CoinItem struct {
	ItemType string `json:"item_type"`
	Value    string `json:"value"`
}

type ItemType string

const (
	BoxCoin ItemType = "boxCoin"
	OneCoin ItemType = "oneCoin"
)

type PlayerScore struct {
	Valid     bool   `json:"valid"`
	ID        string `json:"id"`
	*Position `json:"position"`
	*CoinItem `json:"coin_item"`
}

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
	PlayerActionType   GameMsgType = "playerAction"
	CoinActionType     GameMsgType = "coinAction"
	CoinCollectedType  GameMsgType = "coinCollected"
	PlayerChatMsgType  GameMsgType = "playerChatMsg"
	ErrorType          GameMsgType = "errorChatMsg"
	AlertType          GameMsgType = "alertMsg"
)

type GameMsg struct {
	Type    GameMsgType `json:"type"`
	Content interface{} `json:"content"`
}
