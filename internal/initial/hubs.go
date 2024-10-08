package initial

import (
	"pickup/internal/game"
	"sync"
)

func InitHubManager() {
	hm := &game.HubManager{
		Hubs: make(map[string]*game.Hub),
		Mu:   sync.RWMutex{},
	}

	h1 := game.NewHub(hm, "A")
	h2 := game.NewHub(hm, "B")

	h1.InitAllItems()
	h2.InitAllItems()

	hm.RegisterHub(h1)
	hm.RegisterHub(h2)

	game.Hm = hm
}
