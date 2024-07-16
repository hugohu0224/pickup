package global

import (
	"github.com/spf13/viper"
	"pickup/internal/game"
	"sync"
)

var (
	HubManager     *game.HubManager
	Dv             *viper.Viper // default viper
	Gv             *viper.Viper // google client viper
	UserTokenMap   sync.Map
	UserLoginState sync.Map // map[hashedEmail]bool if player into the room and playing
)
