package initial

import (
	"pickup/internal/game"
	"pickup/internal/global"
	"sync"
)

func InitHubManager() {
	hm := &game.HubManager{
		Hubs: make(map[string]*game.Hub),
		Mu:   sync.RWMutex{},
	}
	h1 := game.NewHub("A")
	h2 := game.NewHub("B")
	hm.RegisterHub(h1)
	hm.RegisterHub(h2)

	global.HubManager = hm
}
