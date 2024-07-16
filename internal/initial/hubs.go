package initial

import (
	"pickup/internal/game"
	"pickup/internal/global"
	"sync"
)

func InitHubs() map[string]*game.Hub {
	var hubs = make(map[string]*game.Hub)
	h1 := game.NewHub("room1")
	h2 := game.NewHub("room2")
	hubs["room1"] = h1
	hubs["room2"] = h2
	return hubs
}

func InitHubManager(hubs map[string]*game.Hub) {
	hm := &game.HubManager{
		Hubs: hubs,
		Mu:   sync.RWMutex{},
	}
	global.HubManager = hm
	hm.RunHubs()
}
